package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
)

const WORKERS = 8 //? number of parallel download chunks

func main() {
	//? Extract video URL from command line
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <YouTube_URL>")
		return
	}
	var videoURL string = os.Args[1]

	//? Step 1: Extract direct audio URL + title
	audioURL, err := getAudioURL(videoURL)
	if err != nil {
		fmt.Println("Error getting audio URL:", err)
		return
	}
	title, err := getVideoTitle(videoURL)
	if err != nil {
		fmt.Println("Error getting video title:", err)
		return
	}
	//? Sanitize title for filename
	title = strings.ReplaceAll(title, "/", "-")
	title = strings.ReplaceAll(title, "\\", "-")

	fmt.Printf("🎵 Downloading: %s\n", title)

	//? Output filenames
	var outputWebm string = title + ".webm"
	var outputMp3 string = title + ".mp3"

	//? Step 2: Download audio in parallel chunks
	size, err := getFileSize(audioURL)
	if err != nil {
		fmt.Println("Error getting file size:", err)
		return
	}
	fmt.Printf("📦 File size: %d bytes\n", size)

	chunk := size / WORKERS //? Calculate chunk size
	var wg sync.WaitGroup   //? Create a wait group to synchronize goroutines
	partFiles := []string{} //? Store each part filenames

	for i := 0; i < WORKERS; i++ {
		//? Calculate byte range for each part
		start := int64(i) * chunk
		end := start + chunk - 1
		//? Ensure the last part goes to the end of the file
		if i == WORKERS-1 {
			end = size - 1
		}
		partFileName := fmt.Sprintf("part-%d.tmp", i)
		partFiles = append(partFiles, partFileName)

		wg.Add(1) //? Increment wait group counter
		//! Start downloading part in a goroutine
		go downloadPart(audioURL, start, end, partFiles[i], &wg)
	}
	wg.Wait() //? Wait for all downloads to finish

	//? Step 3 : Merge parts using ffmpeg
	err = mergeParts(outputWebm, partFiles)
	if err != nil {
		fmt.Println("Error merging parts:", err)
		return
	}

	//? Step 4: Convert to mp3 using ffmpeg
	cmd := exec.Command("ffmpeg", "-y", "-i", outputWebm, "-vn", "-c:a", "libmp3lame", "-q:a", "2", outputMp3)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		fmt.Println("Error converting to mp3:", err)
		return
	}

	//? Cleanup
	os.Remove(outputWebm)

	fmt.Printf("✅ Saved as %s\n", outputMp3)
}

// * Run yt-dlp to get the best audio URL
func getAudioURL(videoURL string) (string, error) {
	cmd := exec.Command("yt-dlp", "-f", "bestaudio", "--get-url", videoURL)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		fmt.Println("Error executing yt-dlp:", err)
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

// * Run yt-dlp to get the video title
func getVideoTitle(videoURL string) (string, error) {
	cmd := exec.Command("yt-dlp", "--get-title", videoURL)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		fmt.Println("Error executing yt-dlp for title:", err)
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

// * Get file size from URL using HEAD request
func getFileSize(url string) (int64, error) {
	res, err := http.Head(url)
	if err != nil {
		fmt.Println("Error making HEAD request:", err)
		return 0, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("server returned non-200 status: %v", res.Status)
	}

	//? Check Content-Length header
	size := res.ContentLength
	if size <= 0 {
		return 0, fmt.Errorf("invalid content length: %d", size)
	}
	return size, nil
}

// * Download one part using HTTP Range
func downloadPart(url string, start, end int64, filename string, wg *sync.WaitGroup) {
	defer wg.Done()
	//? Create HTTP request with Range header
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	//? Perform the request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error downloading part:", err)
		return
	}
	defer res.Body.Close()

	partFile, err := os.Create(filename)
	if err != nil {
		fmt.Println("Error creating part file:", err)
		return
	}
	defer partFile.Close()

	//? Write response body to part file
	_, err = io.Copy(partFile, res.Body)
	if err != nil {
		fmt.Println("Error writing to part file:", err)
		return
	}
	fmt.Printf("✅ Downloaded part: %s\n", filename)
}

// * Merge temporary parts into one file using ffmpeg
func mergeParts(output string, parts []string) error {
	//? Create output file
	out, err := os.Create(output)
	if err != nil {
		return err
	}
	defer out.Close()

	//? Append each part to the output file
	for _, pf := range parts {
		part, err := os.Open(pf)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, part)
		part.Close() //? Close part file after copying
		if err != nil {
			return err
		}
		os.Remove(pf) //? Remove part file after merging
	}
	return nil
}

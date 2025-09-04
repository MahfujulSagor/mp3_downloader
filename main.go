package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

const workers = 8 //? number of parallel download chunks

// * Get file size from HEAD request
func getFileSize(url string) (int64, error) {
	resp, err := http.Head(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	sizeStr := resp.Header.Get("Content-Length")
	return strconv.ParseInt(sizeStr, 10, 64)
}

// * Download one part using HTTP Range
func downloadPart(url string, start, end int64, filename string, wg *sync.WaitGroup) {
	defer wg.Done()
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("❌ Error:", err)
		return
	}
	defer resp.Body.Close()

	partFile, _ := os.Create(filename)
	defer partFile.Close()
	io.Copy(partFile, resp.Body)
}

// * Merge temporary parts into one file
func mergeParts(output string, parts []string) error {
	out, err := os.Create(output)
	if err != nil {
		return err
	}
	defer out.Close()

	for _, pf := range parts {
		part, err := os.Open(pf)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, part)
		part.Close()
		if err != nil {
			return err
		}
		os.Remove(pf) //? cleanup
	}
	return nil
}

// * Run yt-dlp to get the best audio URL
func getAudioURL(videoURL string) (string, error) {
	cmd := exec.Command("yt-dlp", "-f", "bestaudio", "--get-url", videoURL)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
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
		return "", err
	}
	//? sanitize title for filenames
	title := strings.TrimSpace(out.String())
	title = strings.ReplaceAll(title, "/", "-")
	title = strings.ReplaceAll(title, "\\", "-")
	return title, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <YouTube_URL>")
		return
	}
	videoURL := os.Args[1]

	//? Step 1: Extract direct audio URL + title
	audioURL, err := getAudioURL(videoURL)
	if err != nil {
		panic(err)
	}
	title, err := getVideoTitle(videoURL)
	if err != nil {
		panic(err)
	}
	fmt.Printf("🎵 Downloading: %s\n", title)

	//? Output file names
	outputWebm := title + ".webm"
	outputMp3 := title + ".mp3"

	//? Step 2: Parallel download
	size, err := getFileSize(audioURL)
	if err != nil {
		panic(err)
	}
	fmt.Printf("📦 File size: %d bytes\n", size)

	chunk := size / workers
	var wg sync.WaitGroup
	partFiles := []string{}

	for i := 0; i < workers; i++ {
		start := int64(i) * chunk
		end := start + chunk - 1
		if i == workers-1 {
			end = size - 1
		}
		partFile := fmt.Sprintf("part-%d.tmp", i)
		partFiles = append(partFiles, partFile)
		wg.Add(1)
		go downloadPart(audioURL, start, end, partFile, &wg)
	}
	wg.Wait()

	err = mergeParts(outputWebm, partFiles)
	if err != nil {
		panic(err)
	}

	//? Step 3: Convert to MP3
	cmd := exec.Command("ffmpeg", "-y", "-i", outputWebm, "-vn", "-c:a", "libmp3lame", "-q:a", "2", outputMp3)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		panic(err)
	}

	fmt.Printf("✅ Saved as %s\n", outputMp3)
}

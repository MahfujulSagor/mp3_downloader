# YouTube MP3 Downloader (Go)

A fast and simple **YouTube audio downloader** written in Go.
Downloads audio in parallel chunks and converts it to MP3 using **ffmpeg**.

---

## 📂 Project Structure

```markdown
mp3_downloader/
├── cmd/
│ └── main.go # Main application
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

---

## ⚡ Features
- Extracts the best audio stream from YouTube videos
- Downloads audio in **parallel chunks** for faster downloads
- Converts audio to MP3 using `ffmpeg`
- Shows colored logs in terminal using [`fatih/color`](https://github.com/fatih/color)
- Automatic cleanup of temporary files
- Works cross-platform (requires `yt-dlp` and `ffmpeg` installed)

---

## 🔧 Prerequisites
Make sure the following are installed and available in your PATH:

- [Go](https://go.dev/)
- [yt-dlp](https://github.com/yt-dlp/yt-dlp)
- [ffmpeg](https://ffmpeg.org/)

---

## 🚀 Running the Downloader

Clone the repository:

```bash
git clone https://github.com/yourusername/mp3_downloader.git
cd mp3_downloader
```

Run the downloader with a YouTube URL:

```bash
go run cmd/main.go <YouTube_URL>
```

Example:

```bash
go run cmd/main.go https://www.youtube.com/watch?v=dQw4w9WgXcQ
```

## 🎵 Output

- Downloads audio in parallel chunks (`part-*.tmp`)

- Merges chunks into `.webm`

- Converts to `.mp3`

- Deletes temporary files after conversion

Example output file:

```bash
Rick Astley - Never Gonna Give You Up.mp3
```

## ⚙️ How It Works

1. Extract audio URL and video title using yt-dlp.

2. Download in parallel using HTTP Range requests.

3. Merge all parts into a single .webm file.

4. Convert to MP3 with ffmpeg.

5. Cleanup temporary files.

## 🛠 Dependencies

- `yt-dlp` - Extract video/audio URLs

- `ffmpeg` - Audio conversion

- `fatih/color` - Colored terminal output

## 📜 License

MIT License. Free to use and modify.

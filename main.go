package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

const AppName = "m3u8-downloader"
const AppVersion = "1.0.0"
const defaultFfmpegPath = "ffmpeg"

const usage = `
NAME
  {{ .AppName }} - is a tool to download videos from m3u8 manifest files.

SYNOPSIS
  {{ .AppName }} [OPTIONS] <m3u8_manifest_url>

OPTIONS
`
const usageExample = `
EXAMPLES
  To extact the manifest file from a URL:
    {{ .AppName }} https://example.com/video.m3u8

  To download a video from a m3u8 manifest file:
    {{ .AppName }} -download https://example.com/video.m3u8

  To download a video from a m3u8 manifest file with a custom filename:
    {{ .AppName }} -download -filename my_video https://example.com/video.m3u8
  
  To download a video from a m3u8 manifest file with a custom ffmpeg path:
    {{ .AppName }} -download -ffmpeg /usr/local/bin/ffmpeg https://example.com/video.m3u8
`

func main() {
	flag.Usage = usageFunc

	help := flag.Bool("help", false, "Show usage")
	version := flag.Bool("version", false, "Show version")
	videoTitle := flag.String("filename", "", "Filename of the downloaded video")
	shouldDownload := flag.Bool("download", false, "Set to true to download the video")
	ffmpegPath := flag.String("ffmpeg", "", "Path to ffmpeg executable")
	flag.Parse()

	if *help {
		flag.Usage()
		return
	}

	if *version {
		fmt.Printf("%s %s\n", AppName, AppVersion)
		return
	}

	if *ffmpegPath == "" {
		*ffmpegPath = defaultFfmpegPath
	}

	// check if ffmpeg is installed
	if *shouldDownload {
		_, err := exec.LookPath(*ffmpegPath)
		if err != nil {
			log.Printf("Error: %s is not installed.\n", *ffmpegPath)
			return
		}
	}

	manifestURL := flag.Arg(0)

	if manifestURL == "" {
		flag.Usage()
		return
	}

	// convert video title to a valid file name by replacing invalid characters with underscores using a regular expression
	var videoTitleFilename string

	if *videoTitle == "" {
		// extract the filename from the URL
		fn := path.Base(manifestURL)
		// remove file extension
		fn = strings.TrimSuffix(fn, path.Ext(fn))
		videoTitle = &fn
	}
	re := regexp.MustCompile(`[^\w\d]+`)
	videoTitleFilename = re.ReplaceAllString(*videoTitle, "_")

	// Open the manifest file
	var file io.ReadCloser
	{
		resp, err := http.Get(manifestURL)
		if err != nil {
			log.Println("Error opening manifest URL:", err)
			return
		}
		defer resp.Body.Close()
		file = resp.Body
	}

	// Create a temporary playlist to write the modified content
	playlistFile, err := os.CreateTemp("", videoTitleFilename+".playlist.*.m3u8")
	log.Printf("Created temporary playlist %s\n", playlistFile.Name())
	if err != nil {
		log.Println("Error creating temporary playlist:", err)
		return
	}
	defer playlistFile.Close()

	// Regular expression to match the resolution
	reResolution := regexp.MustCompile(`RESOLUTION=(\d+)x(\d+)`)
	// Variables to store the highest resolution and corresponding URL
	var maxResolution int
	var maxResolutionURL string

	log.Println("Reading manifest file...")
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#EXT-X-STREAM-INF") {
			// Extract resolution
			matches := reResolution.FindStringSubmatch(line)
			if len(matches) == 3 {
				width, _ := strconv.Atoi(matches[1])
				height, _ := strconv.Atoi(matches[2])
				resolution := width * height
				// Check if this is the highest resolution
				if resolution > maxResolution {
					maxResolution = resolution
					// Read the next line for the URL
					if scanner.Scan() {
						maxResolutionURL = scanner.Text()
					}
				}
			}
		}
	}

	if maxResolutionURL == "" {
		log.Println("No valid resolution found in the manifest.")
		return
	}

	// Extract base path from the URL
	basePath := path.Dir(maxResolutionURL)
	if !strings.HasPrefix(basePath, "https://") {
		if strings.HasPrefix(basePath, "https:/") {
			basePath = "https://" + basePath[7:]
		} else if strings.HasPrefix(basePath, "https:") {
			basePath = "https://" + basePath[6:]
		} else if strings.HasPrefix(basePath, "http:/") {
			basePath = "http://" + basePath[6:]
		} else if strings.HasPrefix(basePath, "http:") {
			basePath = "http://" + basePath[5:]
		} else {
			basePath = "https://" + basePath
		}
	}

	// Download the video manifest
	log.Println("Downloading video manifest...")
	resp, err := http.Get(maxResolutionURL)
	if err != nil {
		log.Println("Error downloading video manifest:", err)
		return
	}
	defer resp.Body.Close()

	manifestBody := resp.Body

	log.Println("Writing modified playlist...")
	scanner = bufio.NewScanner(manifestBody)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#EXTINF") {
			// Write the #EXTINF line
			playlistFile.WriteString(line + "\n")
			// Read the next line for the URL and prepend the base path
			if scanner.Scan() {
				urlLine := scanner.Text()
				if !strings.HasPrefix(urlLine, "http") {
					urlLine = basePath + "/" + urlLine
				}
				playlistFile.WriteString(urlLine + "\n")
			}
		} else {
			playlistFile.WriteString(line + "\n")
		}
	}

	if err := scanner.Err(); err != nil {
		log.Println("Error reading manifest file:", err)
	}

	// Execute ffmpeg command
	if *shouldDownload {
		log.Println("Starting video download...")
		downloadVideo(*ffmpegPath, playlistFile.Name(), *videoTitle)
	} else {
		log.Printf("Video download skipped. To download see usage: %s -help\n", AppName)
	}
}

func usageFunc() {
	usageTmpl, err := template.New("usage").Parse(usage)
	if err != nil {
		log.Println("Error parsing usage template:", err)
		return
	}

	usageExampleTmpl, err := template.New("usageExample").Parse(usageExample)
	if err != nil {
		log.Println("Error parsing usage example template:", err)
		return
	}

	err = usageTmpl.Execute(os.Stdout, map[string]string{"AppName": AppName})
	if err != nil {
		log.Println("Error executing usage template:", err)
		return
	}
	flag.PrintDefaults()
	err = usageExampleTmpl.Execute(os.Stdout, map[string]string{"AppName": AppName})
	if err != nil {
		log.Println("Error executing usage example template:", err)
		return
	}
}

func downloadVideo(ffmpegPath string, playlistFilename string, videoTitle string) {
	log.Println("Converting video...")
	cmd := exec.Command(ffmpegPath, "-protocol_whitelist", "https,file,tls,tcp", "-i", playlistFilename, "-c", "copy", videoTitle+".mp4")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	start := time.Now() // Start the timer

	if err := cmd.Run(); err != nil {
		log.Println("Error executing ffmpeg command:", err)
		return
	}

	elapsed := time.Since(start) // Calculate the elapsed time
	log.Printf("Video conversion completed in %s.\n", elapsed)
}

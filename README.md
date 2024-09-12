# m3u8-downloader - download m3u8 files and convert to mp4

This project downloads a m3u8 file and converts it to mp4. It uses ffmpeg to convert the file.

## Installation

```bash
go install github.com/jrschumacher/m3u8-downloader@latest
```

## Usage

```bash
NAME
  m3u8-downloader - is a tool to download videos from m3u8 manifest files.

SYNOPSIS
  m3u8-downloader [OPTIONS] <m3u8_manifest_url>

OPTIONS
  -download
        Set to true to download the video
  -ffmpeg string
        Path to ffmpeg executable
  -filename string
        Filename of the downloaded video
  -help
        Show usage
  -version
        Show version

EXAMPLES
  To extact the manifest file from a URL:
    m3u8-downloader https://example.com/video.m3u8

  To download a video from a m3u8 manifest file:
    m3u8-downloader -download https://example.com/video.m3u8

  To download a video from a m3u8 manifest file with a custom filename:
    m3u8-downloader -download -filename my_video https://example.com/video.m3u8
  
  To download a video from a m3u8 manifest file with a custom ffmpeg path:
    m3u8-downloader -download -ffmpeg /usr/local/bin/ffmpeg https://example.com/video.m3u8
```

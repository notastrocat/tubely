package main

import (
	"bytes"
	"errors"
	"log"
	"os/exec"
	"strings"
)

func getVideoAspectRatio(filePath string) (string, error) {
	// Command to get the Display Aspect Ratio (DAR)
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=display_aspect_ratio",
		"-of", "default=noprint_wrappers=1:nokey=1",
		filePath,
	)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		log.Println(err)
		return "", errors.New("Cannot get video aspect ratio.")
	}

	// Clean up the output (remove any trailing newlines)
	ratio := strings.TrimSpace(out.String())

	// Handle cases where the ratio isn't explicitly set in metadata
	if ratio == "0:1" || ratio == "" {
		log.Println("video has an aspect ratio of", ratio)
		return "", errors.New("Cannot get video aspect ratio.")
	}

	switch ratio {
	case "16:9":
		return "landscape", nil
	case "9:16":
		return "portrait", nil
	default:
		return "other", nil
	}
}

package storage

import (
	"bytes"
	"image"
	"os"
	"strconv"
)

// decodeDimensions extracts image width/height from raw bytes.
// Falls back to optional SCREENSHOT_DEFAULT_WIDTH/HEIGHT env vars when decoding fails.
func decodeDimensions(payload []byte) (int, int) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(payload))
	if err == nil && cfg.Width > 0 && cfg.Height > 0 {
		return cfg.Width, cfg.Height
	}

	width := 0
	height := 0
	if widthStr := os.Getenv("SCREENSHOT_DEFAULT_WIDTH"); widthStr != "" {
		if w, convErr := strconv.Atoi(widthStr); convErr == nil {
			width = w
		}
	}
	if heightStr := os.Getenv("SCREENSHOT_DEFAULT_HEIGHT"); heightStr != "" {
		if h, convErr := strconv.Atoi(heightStr); convErr == nil {
			height = h
		}
	}

	return width, height
}

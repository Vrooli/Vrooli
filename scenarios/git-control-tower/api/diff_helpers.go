package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	maxDiffFileBytes  int64 = 4 * 1024 * 1024
	binarySampleBytes       = 8 * 1024
)

type FileTooLargeError struct {
	Path  string
	Size  int64
	Limit int64
}

func (e *FileTooLargeError) Error() string {
	path := e.Path
	if path == "" {
		path = "file"
	}
	return fmt.Sprintf("file too large to display: %s (%s > %s)", path, formatBytes(e.Size), formatBytes(e.Limit))
}

type UnsupportedBinaryError struct {
	Path string
}

func (e *UnsupportedBinaryError) Error() string {
	path := e.Path
	if path == "" {
		path = "file"
	}
	return fmt.Sprintf("binary file not supported for display: %s", path)
}

type binaryKind int

const (
	binaryNone binaryKind = iota
	binaryImage
	binaryUnsupported
)

// binaryImageExtensions lists raster image formats whose bytes cannot be shown
// as text and are therefore base64-encoded for the client.
//
// SVG is deliberately absent: it is XML text, so git diffs it line by line and
// so do we. Treating it as binary hid its source, its hunks, and its line counts
// behind an opaque base64 blob. It is still previewable as an image — that is a
// rendering decision the client makes from the extension, not a transport one.
var binaryImageExtensions = map[string]struct{}{
	".png":  {},
	".jpg":  {},
	".jpeg": {},
	".gif":  {},
	".webp": {},
	".ico":  {},
	".bmp":  {},
	".tiff": {},
}

var binaryUnsupportedExtensions = map[string]struct{}{
	".pdf": {},
}

func binaryKindForPath(path string) binaryKind {
	ext := strings.ToLower(filepath.Ext(path))
	if _, ok := binaryImageExtensions[ext]; ok {
		return binaryImage
	}
	if _, ok := binaryUnsupportedExtensions[ext]; ok {
		return binaryUnsupported
	}
	return binaryNone
}

func detectBinaryKind(path string, data []byte) binaryKind {
	kind := binaryKindForPath(path)
	if kind != binaryNone {
		return kind
	}
	if isBinaryData(binarySample(data)) {
		return binaryUnsupported
	}
	return binaryNone
}

// binarySample returns the leading bytes used to classify content, truncated on
// a UTF-8 rune boundary. Cutting mid-rune would leave a trailing partial
// sequence that isBinaryData reads as invalid UTF-8, misreporting a text file
// with any multi-byte character near the cutoff as binary.
func binarySample(data []byte) []byte {
	if len(data) <= binarySampleBytes {
		return data
	}
	sample := data[:binarySampleBytes]
	// A UTF-8 sequence is at most 4 bytes, so at most 3 trailing bytes can be
	// the start of a rune that continues past the cutoff.
	for i := 0; i < utf8.UTFMax-1 && len(sample) > 0; i++ {
		if r, size := utf8.DecodeLastRune(sample); r != utf8.RuneError || size > 1 {
			break
		}
		sample = sample[:len(sample)-1]
	}
	return sample
}

func isBinaryData(sample []byte) bool {
	if len(sample) == 0 {
		return false
	}
	if bytes.IndexByte(sample, 0) >= 0 {
		return true
	}
	if !utf8.Valid(sample) {
		return true
	}
	return false
}

func readFileForDisplay(absPath, displayPath string) ([]byte, int64, error) {
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, 0, err
	}
	if info.IsDir() {
		return nil, 0, fmt.Errorf("path is a directory")
	}
	size := info.Size()
	if size > maxDiffFileBytes {
		return nil, size, &FileTooLargeError{Path: displayPath, Size: size, Limit: maxDiffFileBytes}
	}

	file, err := os.Open(absPath)
	if err != nil {
		return nil, size, err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxDiffFileBytes+1))
	if err != nil {
		return nil, size, err
	}
	if int64(len(data)) > maxDiffFileBytes {
		return nil, size, &FileTooLargeError{Path: displayPath, Size: int64(len(data)), Limit: maxDiffFileBytes}
	}
	return data, size, nil
}

func ensureFileWithinLimit(repoDir, cleanPath string) error {
	absPath := filepath.Join(repoDir, cleanPath)
	info, err := os.Stat(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory")
	}
	if info.Size() > maxDiffFileBytes {
		return &FileTooLargeError{Path: cleanPath, Size: info.Size(), Limit: maxDiffFileBytes}
	}
	return nil
}

func formatBytes(size int64) string {
	const kb = 1024
	const mb = 1024 * 1024
	if size >= mb {
		return fmt.Sprintf("%.1f MB", float64(size)/float64(mb))
	}
	if size >= kb {
		return fmt.Sprintf("%.1f KB", float64(size)/float64(kb))
	}
	return fmt.Sprintf("%d B", size)
}

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

var binaryImageExtensions = map[string]struct{}{
	".png":  {},
	".jpg":  {},
	".jpeg": {},
	".gif":  {},
	".svg":  {},
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
	sample := data
	if len(sample) > binarySampleBytes {
		sample = sample[:binarySampleBytes]
	}
	if isBinaryData(sample) {
		return binaryUnsupported
	}
	return binaryNone
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

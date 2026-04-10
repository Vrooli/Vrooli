package state

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
	"strings"
	"time"
)

// CompressLog compresses log content using gzip and returns a CompressedLog.
// The compressed data is base64-encoded for safe JSON storage.
func CompressLog(serviceID, content string) (CompressedLog, error) {
	lines := countLines(content)

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	if _, err := gz.Write([]byte(content)); err != nil {
		gz.Close()
		return CompressedLog{}, err
	}

	if err := gz.Close(); err != nil {
		return CompressedLog{}, err
	}

	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())

	return CompressedLog{
		ServiceID:      serviceID,
		CompressedData: encoded,
		OriginalLines:  lines,
		CompressedSize: buf.Len(),
		CapturedAt:     time.Now().Format(time.RFC3339),
	}, nil
}

// DecompressLog decompresses a CompressedLog and returns the original content.
func DecompressLog(cl CompressedLog) (string, error) {
	data, err := base64.StdEncoding.DecodeString(cl.CompressedData)
	if err != nil {
		return "", err
	}

	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer gz.Close()

	result, err := io.ReadAll(gz)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

// CompressLogs compresses multiple log inputs into CompressedLog entries.
func CompressLogs(inputs []LogTailInput) ([]CompressedLog, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	result := make([]CompressedLog, 0, len(inputs))
	for _, input := range inputs {
		if input.Content == "" {
			continue
		}

		cl, err := CompressLog(input.ServiceID, input.Content)
		if err != nil {
			return nil, err
		}
		result = append(result, cl)
	}

	return result, nil
}

// FindCompressedLog finds a compressed log by service ID.
func FindCompressedLog(logs []CompressedLog, serviceID string) *CompressedLog {
	for i := range logs {
		if logs[i].ServiceID == serviceID {
			return &logs[i]
		}
	}
	return nil
}

// MergeLogs merges new logs into existing logs, replacing by service ID.
func MergeLogs(existing, newLogs []CompressedLog) []CompressedLog {
	if len(newLogs) == 0 {
		return existing
	}

	// Create map for quick lookup
	logMap := make(map[string]CompressedLog, len(existing)+len(newLogs))
	for _, cl := range existing {
		logMap[cl.ServiceID] = cl
	}
	for _, cl := range newLogs {
		logMap[cl.ServiceID] = cl // Overwrite existing
	}

	// Convert back to slice
	result := make([]CompressedLog, 0, len(logMap))
	for _, cl := range logMap {
		result = append(result, cl)
	}

	return result
}

// countLines counts the number of lines in a string.
func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

// CompressionRatio returns the compression ratio for a CompressedLog.
// Returns 0 if original size is unknown.
func CompressionRatio(cl CompressedLog, originalSize int) float64 {
	if originalSize == 0 || cl.CompressedSize == 0 {
		return 0
	}
	return float64(originalSize-cl.CompressedSize) / float64(originalSize)
}

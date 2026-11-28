package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/storage"
)

// LoadJSON loads and unmarshals a JSON file into the target type
func LoadJSON[T any](path string) (*T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	return &result, nil
}

// SaveJSON publishes a complete JSON snapshot without truncating concurrent reads.
func SaveJSON[T any](path string, value *T) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling: %w", err)
	}

	if err := storage.WriteFileAtomic(path, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ListDirectories returns all directories in a path
func ListDirectories(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() && !isHiddenFile(entry.Name()) {
			dirs = append(dirs, entry.Name())
		}
	}
	return dirs, nil
}

// ListFiles returns all files in a path matching a pattern
func ListFiles(path, pattern string) ([]string, error) {
	fullPattern := filepath.Join(path, pattern)
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil, err
	}
	return matches, nil
}

// isHiddenFile checks if a filename starts with a dot
func isHiddenFile(name string) bool {
	return len(name) > 0 && name[0] == '.'
}

// ReadContent reads text content from a file
func ReadContent(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	return string(data), nil
}

// WriteContent writes text content to a file
func WriteContent(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// AppendJSONL appends a JSON line to a JSONL file
func AppendJSONL[T any](path string, entry T) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling entry: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("writing to %s: %w", path, err)
	}

	return nil
}

// appendBoundedJSONL appends an entry and applies the store's existing
// retention and line-count bounds in one read. Store-specific types, paths,
// and limits stay with their owners; malformed rows are retained unless an
// expired valid row causes the existing typed rewrite path.
func appendBoundedJSONL[T any](path string, entry T, now time.Time, maxEntries int, retention time.Duration, timestamp func(T) string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := AppendJSONL(path, entry); err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	cutoff := now.Add(-retention)
	kept := make([]T, 0, len(lines))
	droppedExpired := false
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var decoded T
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			continue
		}
		if at, err := time.Parse(time.RFC3339, timestamp(decoded)); err == nil && at.Before(cutoff) {
			droppedExpired = true
			continue
		}
		kept = append(kept, decoded)
	}
	if droppedExpired {
		if maxEntries > 0 && len(kept) > maxEntries {
			kept = kept[len(kept)-maxEntries:]
		}
		return rewriteJSONL(path, kept)
	}
	if maxEntries <= 0 || len(lines) <= maxEntries {
		return nil
	}
	lines = lines[len(lines)-maxEntries:]
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

// readJSONLSince reads a typed JSONL log in file order. Malformed rows are
// ignored and timestamps that cannot be parsed remain visible, matching the
// telemetry stores' existing read behavior.
func readJSONLSince[T any](path string, window time.Duration, now time.Time, timestamp func(T) string) ([]T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if window > 0 {
				return []T{}, nil
			}
			return nil, nil
		}
		return nil, err
	}
	cutoff := now.Add(-window)
	var out []T
	if window > 0 {
		out = make([]T, 0)
	}
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var entry T
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if window > 0 {
			if at, err := time.Parse(time.RFC3339, timestamp(entry)); err == nil && at.Before(cutoff) {
				continue
			}
		}
		out = append(out, entry)
	}
	return out, nil
}

func rewriteJSONL[T any](path string, entries []T) error {
	var builder strings.Builder
	for _, entry := range entries {
		raw, err := json.Marshal(entry)
		if err != nil {
			continue
		}
		builder.Write(raw)
		builder.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(builder.String()), 0o644)
}

// DeleteFile removes a file
func DeleteFile(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting %s: %w", path, err)
	}
	return nil
}

// DeleteDirectory removes a directory and its contents
func DeleteDirectory(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("deleting directory %s: %w", path, err)
	}
	return nil
}

// FileContentIO implements ContentIO using the real filesystem.
// This is the production implementation.
type FileContentIO struct{}

// NewFileContentIO creates a new FileContentIO instance
func NewFileContentIO() *FileContentIO {
	return &FileContentIO{}
}

// ReadContent reads content from a file path
func (f *FileContentIO) ReadContent(path string) (string, error) {
	return ReadContent(path)
}

// WriteContent writes content to a file path, creating directories as needed
func (f *FileContentIO) WriteContent(path, content string) error {
	return WriteContent(path, content)
}

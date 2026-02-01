package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// SaveJSON marshals and writes a value to a JSON file
func SaveJSON[T any](path string, value *T) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling: %w", err)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
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

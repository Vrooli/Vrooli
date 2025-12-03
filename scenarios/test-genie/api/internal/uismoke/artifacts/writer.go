package artifacts

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"test-genie/internal/uismoke/orchestrator"
)

// Writer persists test artifacts to the filesystem.
type Writer struct {
	fs FileSystem
}

// FileSystem abstracts filesystem operations for testing.
type FileSystem interface {
	WriteFile(path string, data []byte, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
}

// OSFileSystem is the default filesystem implementation using os package.
type OSFileSystem struct{}

// WriteFile writes data to a file.
func (OSFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// MkdirAll creates a directory and all parents.
func (OSFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// NewWriter creates a new artifact Writer.
func NewWriter(opts ...WriterOption) *Writer {
	w := &Writer{
		fs: OSFileSystem{},
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// WriterOption configures a Writer.
type WriterOption func(*Writer)

// WithFileSystem sets a custom filesystem implementation.
func WithFileSystem(fs FileSystem) WriterOption {
	return func(w *Writer) {
		w.fs = fs
	}
}

// coverageDir returns the coverage directory path for UI smoke artifacts.
func coverageDir(scenarioDir, scenarioName string) string {
	return filepath.Join(scenarioDir, "coverage", scenarioName, "ui-smoke")
}

// Ensure Writer implements orchestrator.ArtifactWriter.
var _ orchestrator.ArtifactWriter = (*Writer)(nil)

// WriteAll writes all artifacts and returns their paths.
func (w *Writer) WriteAll(ctx context.Context, scenarioDir, scenarioName string, response *orchestrator.BrowserResponse) (*orchestrator.ArtifactPaths, error) {
	dir := coverageDir(scenarioDir, scenarioName)
	if err := w.fs.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create coverage directory: %w", err)
	}

	paths := &orchestrator.ArtifactPaths{}

	// Write screenshot
	if response.Screenshot != "" {
		screenshotPath := filepath.Join(dir, "screenshot.png")
		decoded, err := base64.StdEncoding.DecodeString(response.Screenshot)
		if err != nil {
			return nil, fmt.Errorf("failed to decode screenshot: %w", err)
		}
		if err := w.fs.WriteFile(screenshotPath, decoded, 0o644); err != nil {
			return nil, fmt.Errorf("failed to write screenshot: %w", err)
		}
		paths.Screenshot = relPath(scenarioDir, screenshotPath)
	}

	// Write console logs
	if len(response.Console) > 0 {
		consolePath := filepath.Join(dir, "console.json")
		data, err := json.MarshalIndent(response.Console, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal console: %w", err)
		}
		if err := w.fs.WriteFile(consolePath, data, 0o644); err != nil {
			return nil, fmt.Errorf("failed to write console: %w", err)
		}
		paths.Console = relPath(scenarioDir, consolePath)
	}

	// Write network failures
	if len(response.Network) > 0 {
		networkPath := filepath.Join(dir, "network.json")
		data, err := json.MarshalIndent(response.Network, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal network: %w", err)
		}
		if err := w.fs.WriteFile(networkPath, data, 0o644); err != nil {
			return nil, fmt.Errorf("failed to write network: %w", err)
		}
		paths.Network = relPath(scenarioDir, networkPath)
	}

	// Write DOM snapshot
	if response.HTML != "" {
		htmlPath := filepath.Join(dir, "dom.html")
		if err := w.fs.WriteFile(htmlPath, []byte(response.HTML), 0o644); err != nil {
			return nil, fmt.Errorf("failed to write html: %w", err)
		}
		paths.HTML = relPath(scenarioDir, htmlPath)
	}

	// Write raw response (without screenshot)
	if len(response.Raw) > 0 {
		rawPath := filepath.Join(dir, "raw.json")
		// Pretty-print the raw JSON
		var obj interface{}
		if err := json.Unmarshal(response.Raw, &obj); err == nil {
			prettyData, _ := json.MarshalIndent(obj, "", "  ")
			if err := w.fs.WriteFile(rawPath, prettyData, 0o644); err != nil {
				return nil, fmt.Errorf("failed to write raw: %w", err)
			}
		} else {
			if err := w.fs.WriteFile(rawPath, response.Raw, 0o644); err != nil {
				return nil, fmt.Errorf("failed to write raw: %w", err)
			}
		}
		paths.Raw = relPath(scenarioDir, rawPath)
	}

	return paths, nil
}

// WriteResultJSON writes a result object as JSON.
func (w *Writer) WriteResultJSON(ctx context.Context, scenarioDir, scenarioName string, result interface{}) error {
	dir := coverageDir(scenarioDir, scenarioName)
	if err := w.fs.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create coverage directory: %w", err)
	}

	resultPath := filepath.Join(dir, "latest.json")
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := w.fs.WriteFile(resultPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write result: %w", err)
	}

	return nil
}

// relPath returns a relative path from baseDir to path.
// If relativization fails, returns the absolute path.
func relPath(baseDir, path string) string {
	rel, err := filepath.Rel(baseDir, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

package storage

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
)

// FileStorage stores screenshots on the local filesystem for offline/desktop use.
type FileStorage struct {
	root string
	log  *logrus.Logger
}

// NewFileStorage initializes a filesystem-backed storage rooted at the provided directory.
func NewFileStorage(root string, log *logrus.Logger) (*FileStorage, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("screenshot root is required")
	}
	absRoot := root
	if resolved, err := filepath.Abs(root); err == nil {
		absRoot = resolved
	}
	if err := os.MkdirAll(absRoot, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create screenshot root: %w", err)
	}
	return &FileStorage{root: absRoot, log: log}, nil
}

func (f *FileStorage) objectPath(objectName string) (string, error) {
	cleaned := filepath.Clean(strings.TrimSpace(objectName))
	cleaned = strings.TrimPrefix(cleaned, string(filepath.Separator))
	if cleaned == "." || strings.HasPrefix(cleaned, "..") || strings.Contains(cleaned, string(filepath.Separator)+"..") {
		return "", fmt.Errorf("invalid object name")
	}
	return filepath.Join(f.root, cleaned), nil
}

func (f *FileStorage) ensureParent(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

// GetScreenshot reads a screenshot from disk.
func (f *FileStorage) GetScreenshot(_ context.Context, objectName string) (io.ReadCloser, *minio.ObjectInfo, error) {
	path, err := f.objectPath(objectName)
	if err != nil {
		return nil, nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open screenshot: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, fmt.Errorf("failed to stat screenshot: %w", err)
	}
	header := make([]byte, 512)
	n, _ := file.Read(header)
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		file.Close()
		return nil, nil, fmt.Errorf("failed to rewind screenshot: %w", err)
	}
	contentType := http.DetectContentType(header[:n])

	return file, &minio.ObjectInfo{
		Key:         objectName,
		Size:        info.Size(),
		ContentType: contentType,
	}, nil
}

// StoreScreenshot writes raw bytes to disk and returns API URLs.
func (f *FileStorage) StoreScreenshot(_ context.Context, executionID uuid.UUID, stepName string, data []byte, contentType string) (*ScreenshotInfo, error) {
	objectName := fmt.Sprintf("screenshots/%s/%s-%s.png", executionID, sanitizeStepName(stepName), uuid.New())
	path, err := f.objectPath(objectName)
	if err != nil {
		return nil, err
	}
	if err := f.ensureParent(path); err != nil {
		return nil, fmt.Errorf("failed to ensure screenshot directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return nil, fmt.Errorf("failed to write screenshot: %w", err)
	}
	width, height := decodeDimensions(data)
	if contentType == "" {
		if ext := filepath.Ext(path); ext != "" {
			contentType = mime.TypeByExtension(ext)
		}
		if contentType == "" {
			contentType = http.DetectContentType(data)
		}
	}

	url := fmt.Sprintf("/api/v1/screenshots/%s", objectName)
	return &ScreenshotInfo{
		URL:          url,
		ThumbnailURL: url,
		SizeBytes:    int64(len(data)),
		Width:        width,
		Height:       height,
	}, nil
}

// StoreScreenshotFromFile copies a screenshot file into storage.
func (f *FileStorage) StoreScreenshotFromFile(ctx context.Context, executionID uuid.UUID, stepName string, filePath string) (*ScreenshotInfo, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read screenshot file: %w", err)
	}
	return f.StoreScreenshot(ctx, executionID, stepName, data, mime.TypeByExtension(filepath.Ext(filePath)))
}

// DeleteScreenshot removes the screenshot file.
func (f *FileStorage) DeleteScreenshot(_ context.Context, objectName string) error {
	path, err := f.objectPath(objectName)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete screenshot: %w", err)
	}
	return nil
}

// ListExecutionScreenshots lists stored screenshot object names for an execution.
func (f *FileStorage) ListExecutionScreenshots(_ context.Context, executionID uuid.UUID) ([]string, error) {
	base := filepath.Join(f.root, "screenshots", executionID.String())
	var objects []string
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(f.root, path)
		if err != nil {
			return err
		}
		objects = append(objects, filepath.ToSlash(rel))
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to list screenshots: %w", err)
	}
	return objects, nil
}

// HealthCheck verifies the root is writable.
func (f *FileStorage) HealthCheck(_ context.Context) error {
	testFile := filepath.Join(f.root, ".healthcheck")
	if err := os.MkdirAll(filepath.Dir(testFile), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(testFile, []byte("ok"), 0o644); err != nil {
		return err
	}
	return os.Remove(testFile)
}

// GetBucketName returns a stable identifier for the file backend.
func (f *FileStorage) GetBucketName() string {
	return "screenshots-local"
}

func sanitizeStepName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	if name == "" {
		return "step"
	}
	return name
}

var _ StorageInterface = (*FileStorage)(nil)

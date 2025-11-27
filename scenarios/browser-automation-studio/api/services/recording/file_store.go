package recording

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/storage"
)

// recordingFileStore implements StorageInterface by writing assets to the local
// recordings root so legacy asset streaming continues to work.
type recordingFileStore struct {
	root string
	log  *logrus.Logger
}

func newRecordingFileStore(root string, log *logrus.Logger) *recordingFileStore {
	return &recordingFileStore{root: root, log: log}
}

func (s *recordingFileStore) objectPath(executionID uuid.UUID, objectName string) string {
	baseDir := filepath.Join(strings.TrimSpace(s.root), executionID.String())
	return filepath.Join(baseDir, objectName)
}

func (s *recordingFileStore) ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func (s *recordingFileStore) GetScreenshot(_ context.Context, objectName string) (io.ReadCloser, *minio.ObjectInfo, error) {
	return nil, nil, fmt.Errorf("recording file store does not support GetScreenshot (%s)", objectName)
}

func (s *recordingFileStore) StoreScreenshot(ctx context.Context, executionID uuid.UUID, stepName string, data []byte, contentType string) (*storage.ScreenshotInfo, error) {
	dir := s.objectPath(executionID, "frames")
	if err := s.ensureDir(dir); err != nil {
		return nil, err
	}
	filename := fmt.Sprintf("%s-%s.png", recordingSanitizeFilename(stepName), uuid.New().String())
	if err := os.WriteFile(filepath.Join(dir, filename), data, 0o644); err != nil {
		return nil, err
	}

	width, height := decodeDimensions(data)
	publicURL := fmt.Sprintf("/api/v1/recordings/assets/%s/frames/%s", executionID.String(), filename)

	return &storage.ScreenshotInfo{
		URL:          publicURL,
		ThumbnailURL: publicURL,
		SizeBytes:    int64(len(data)),
		Width:        width,
		Height:       height,
	}, nil
}

func (s *recordingFileStore) StoreScreenshotFromFile(ctx context.Context, executionID uuid.UUID, stepName string, filePath string) (*storage.ScreenshotInfo, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return s.StoreScreenshot(ctx, executionID, stepName, data, "image/png")
}

func (s *recordingFileStore) DeleteScreenshot(_ context.Context, objectName string) error {
	_ = objectName
	return nil
}

func (s *recordingFileStore) ListExecutionScreenshots(_ context.Context, executionID uuid.UUID) ([]string, error) {
	return nil, nil
}

func (s *recordingFileStore) HealthCheck(_ context.Context) error {
	return nil
}

func (s *recordingFileStore) GetBucketName() string {
	return "recordings-local"
}

func recordingSanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	if name == "" {
		return "frame"
	}
	return name
}

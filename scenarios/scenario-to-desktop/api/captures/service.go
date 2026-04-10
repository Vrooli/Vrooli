package captures

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/storage"
)

// Service encapsulates file + metadata operations for captures.
type Service struct {
	resolver *storage.Resolver
	opts     storage.Options
	store    Store
}

// NewService creates a new captures service.
func NewService(resolver *storage.Resolver, opts storage.Options, store Store) *Service {
	return &Service{
		resolver: resolver,
		opts:     opts,
		store:    store,
	}
}

// Store returns the underlying metadata store.
func (s *Service) Store() Store {
	return s.store
}

func (s *Service) capturesDir() (string, error) {
	return storage.EnsureClassDir(s.resolver, s.opts, storage.ClassData, 0)
}

// SaveCapture moves a file from srcPath into the persistent captures directory and records metadata.
func (s *Service) SaveCapture(scenarioName string, captureType CaptureType, sourceSession, srcPath string, width, height int, durationMs int64) (*Capture, error) {
	dir, err := s.capturesDir()
	if err != nil {
		return nil, fmt.Errorf("resolving captures directory: %w", err)
	}

	id := uuid.New().String()
	ext := filepath.Ext(srcPath)
	filename := fmt.Sprintf("%s-%d%s", captureType, time.Now().UnixMilli(), ext)
	destPath := filepath.Join(dir, filename)

	// Read source and write atomically to destination
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, fmt.Errorf("reading source file: %w", err)
	}
	if err := storage.WriteFileAtomic(destPath, data, 0); err != nil {
		return nil, fmt.Errorf("writing capture file: %w", err)
	}

	// Remove original
	_ = os.Remove(srcPath)

	info, err := os.Stat(destPath)
	if err != nil {
		return nil, fmt.Errorf("stat capture file: %w", err)
	}

	capture := Capture{
		ID:            id,
		ScenarioName:  scenarioName,
		Type:          captureType,
		Filename:      filename,
		FileSizeBytes: info.Size(),
		Width:         width,
		Height:        height,
		DurationMs:    durationMs,
		SourceSession: sourceSession,
		CreatedAt:     time.Now(),
	}

	if err := s.store.Add(capture); err != nil {
		return nil, fmt.Errorf("persisting capture metadata: %w", err)
	}

	return &capture, nil
}

// DeleteCapture removes a single capture file and its metadata.
func (s *Service) DeleteCapture(scenarioName, captureID string) error {
	caps, err := s.store.List(scenarioName)
	if err != nil {
		return err
	}
	for _, c := range caps {
		if c.ID == captureID {
			dir, err := s.capturesDir()
			if err != nil {
				return err
			}
			_ = os.Remove(filepath.Join(dir, c.Filename))
			return s.store.Delete(scenarioName, captureID)
		}
	}
	return fmt.Errorf("capture %q not found", captureID)
}

// CleanAll removes all captures for a scenario.
func (s *Service) CleanAll(scenarioName string) error {
	deleted, err := s.store.DeleteAll(scenarioName)
	if err != nil {
		return err
	}
	dir, err := s.capturesDir()
	if err != nil {
		return nil // metadata already cleaned, file cleanup is best-effort
	}
	for _, c := range deleted {
		_ = os.Remove(filepath.Join(dir, c.Filename))
	}
	return nil
}

// CaptureFilePath resolves the absolute path for serving a capture file.
func (s *Service) CaptureFilePath(scenarioName, captureID string) (string, error) {
	caps, err := s.store.List(scenarioName)
	if err != nil {
		return "", err
	}
	for _, c := range caps {
		if c.ID == captureID {
			dir, err := s.capturesDir()
			if err != nil {
				return "", err
			}
			return filepath.Join(dir, c.Filename), nil
		}
	}
	return "", fmt.Errorf("capture %q not found", captureID)
}

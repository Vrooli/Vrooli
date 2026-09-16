package captures

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/storage"
)

// Service encapsulates file + metadata operations for captures.
type Service struct {
	resolver *storage.Resolver
	opts     storage.Options
	store    Store
	filesDir string
}

// NewService creates a new captures service.
func NewService(resolver *storage.Resolver, opts storage.Options, filesDir string, store Store) *Service {
	return &Service{
		resolver: resolver,
		opts:     opts,
		store:    store,
		filesDir: filesDir,
	}
}

// Store returns the underlying metadata store.
func (s *Service) Store() Store {
	return s.store
}

func (s *Service) capturesDir() (string, error) {
	if s.filesDir != "" {
		if err := os.MkdirAll(s.filesDir, storage.DefaultDirPerm); err != nil {
			return "", err
		}
		return s.filesDir, nil
	}
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
	digest := checksum(data)
	if existing, ok := s.store.(interface {
		List(string) ([]Capture, error)
	}); ok {
		if captures, listErr := existing.List(scenarioName); listErr == nil {
			for _, prior := range captures {
				if prior.Checksum == digest {
					filename = prior.Filename
					destPath = filepath.Join(dir, filename)
					break
				}
			}
		}
	}
	if _, statErr := os.Stat(destPath); os.IsNotExist(statErr) {
		if err := storage.WriteFileAtomic(destPath, data, 0); err != nil {
			return nil, fmt.Errorf("writing capture file: %w", err)
		}
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
		Checksum:      digest,
		SourceSession: sourceSession,
		CreatedAt:     time.Now(),
	}

	if err := s.store.Add(capture); err != nil {
		return nil, fmt.Errorf("persisting capture metadata: %w", err)
	}

	return &capture, nil
}

func checksum(data []byte) string {
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
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
			path, err := capturePath(dir, c.Filename)
			if err != nil {
				return err
			}
			if refs, ok := s.store.(FilenameReferences); !ok || func() bool { n, e := refs.FilenameReferences(c.Filename); return e != nil || n <= 1 }() {
				_ = os.Remove(path)
			}
			return s.store.Delete(scenarioName, captureID)
		}
	}
	return fmt.Errorf("capture %q not found", captureID)
}

func (s *Service) VoidCapture(scenarioName, captureID, reason, supersededBy string) error {
	voider, ok := s.store.(Voider)
	if !ok {
		return fmt.Errorf("capture store does not support voiding")
	}
	return voider.Void(scenarioName, captureID, reason, supersededBy)
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
		refs := 0
		if referenceStore, ok := s.store.(FilenameReferences); ok {
			refs, _ = referenceStore.FilenameReferences(c.Filename)
		}
		if refs == 0 {
			if path, err := capturePath(dir, c.Filename); err == nil {
				_ = os.Remove(path)
			}
		}
	}
	return nil
}

// OrphanFiles reports regular capture files with no durable metadata owner.
func (s *Service) OrphanFiles() ([]string, error) {
	dir, err := s.capturesDir()
	if err != nil {
		return nil, err
	}
	if store, ok := s.store.(interface {
		OrphanFiles(string) ([]string, error)
	}); ok {
		return store.OrphanFiles(dir)
	}
	return nil, fmt.Errorf("capture store does not support orphan detection")
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
			return capturePath(dir, c.Filename)
		}
	}
	return "", fmt.Errorf("capture %q not found", captureID)
}

// capturePath resolves a metadata filename beneath the configured captures
// root. It validates both lexical traversal and symlink resolution so a
// corrupted metadata file cannot turn the file-serving or cleanup paths into
// arbitrary filesystem access.
func capturePath(dir, filename string) (string, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	candidate, err := filepath.Abs(filepath.Join(dir, filename))
	if err != nil {
		return "", err
	}
	if candidate != root && !strings.HasPrefix(candidate, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("capture path escapes captures root")
	}
	resolved := candidate
	if real, err := filepath.EvalSymlinks(candidate); err == nil {
		resolved, err = filepath.Abs(real)
		if err != nil {
			return "", err
		}
	}
	if resolved != root && !strings.HasPrefix(resolved, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("capture path resolves outside captures root")
	}
	return candidate, nil
}

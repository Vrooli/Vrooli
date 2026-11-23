package blobstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type filesystemBlobStore struct {
	root       string
	mu         sync.Mutex
	renameFile func(string, string) error
}

type fileMeta struct {
	MIME string `json:"mime"`
}

// NewFilesystemBlobStore stores blobs below root. Keys are relative slash paths.
func NewFilesystemBlobStore(root string) BlobStore {
	return &filesystemBlobStore{root: root}
}

func (s *filesystemBlobStore) Put(ctx context.Context, key string, r io.Reader, mime string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := s.pathFor(key)
	if err != nil {
		return err
	}
	if r == nil {
		return errors.New("blobstore: reader is nil")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("blobstore: create parent directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".blob-*")
	if err != nil {
		return fmt.Errorf("blobstore: create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}
	defer cleanup()
	if _, err := io.Copy(tmp, r); err != nil {
		cleanup()
		return fmt.Errorf("blobstore: write blob: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("blobstore: close blob: %w", err)
	}
	meta, err := json.Marshal(fileMeta{MIME: strings.TrimSpace(mime)})
	if err != nil {
		return fmt.Errorf("blobstore: encode metadata: %w", err)
	}
	metaFile, err := os.CreateTemp(filepath.Dir(path), ".blob-meta-*")
	if err != nil {
		cleanup()
		return fmt.Errorf("blobstore: create metadata temp file: %w", err)
	}
	metaPath := metaFile.Name()
	defer os.Remove(metaPath)
	if _, err := metaFile.Write(meta); err != nil {
		_ = metaFile.Close()
		cleanup()
		return fmt.Errorf("blobstore: write metadata temp file: %w", err)
	}
	if err := metaFile.Close(); err != nil {
		cleanup()
		return fmt.Errorf("blobstore: close metadata temp file: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	metadataPath := path + ".meta"
	previousMetadata, previousMode, metadataExisted, err := readMetadataForReplace(metadataPath)
	if err != nil {
		cleanup()
		return err
	}
	// Commit the metadata first. If either rename fails, restore the previous
	// sidecar before returning so a reported failed Put never leaves a new blob
	// paired with stale or missing MIME metadata.
	if err := s.rename(metaPath, metadataPath); err != nil {
		cleanup()
		return fmt.Errorf("blobstore: write metadata: %w", err)
	}
	if err := s.rename(tmpPath, path); err != nil {
		rollbackErr := restoreMetadata(metadataPath, previousMetadata, previousMode, metadataExisted)
		cleanup()
		if rollbackErr != nil {
			return fmt.Errorf("blobstore: commit blob: %w (metadata rollback failed: %v)", err, rollbackErr)
		}
		return fmt.Errorf("blobstore: commit blob: %w", err)
	}
	return nil
}

func (s *filesystemBlobStore) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path, err := s.pathFor(key)
	if err != nil {
		return nil, "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, "", fmt.Errorf("blobstore: open blob: %w", err)
	}
	mime := ""
	if data, err := os.ReadFile(path + ".meta"); err == nil {
		var meta fileMeta
		if json.Unmarshal(data, &meta) == nil {
			mime = meta.MIME
		}
	}
	return f, mime, nil
}

func (s *filesystemBlobStore) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path, err := s.pathFor(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("blobstore: delete blob: %w", err)
	}
	if err := os.Remove(path + ".meta"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("blobstore: delete metadata: %w", err)
	}
	return nil
}

func (s *filesystemBlobStore) rename(source, destination string) error {
	if s.renameFile != nil {
		return s.renameFile(source, destination)
	}
	return os.Rename(source, destination)
}

func readMetadataForReplace(path string) ([]byte, os.FileMode, bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, 0, false, nil
	}
	if err != nil {
		return nil, 0, false, fmt.Errorf("blobstore: inspect metadata: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, 0, false, fmt.Errorf("blobstore: metadata destination is not a regular file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, false, fmt.Errorf("blobstore: read existing metadata: %w", err)
	}
	return data, info.Mode().Perm(), true, nil
}

func restoreMetadata(path string, previous []byte, mode os.FileMode, existed bool) error {
	if !existed {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".blob-meta-rollback-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(previous); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func (s *filesystemBlobStore) pathFor(key string) (string, error) {
	key = strings.TrimSpace(filepath.ToSlash(key))
	if key == "" {
		return "", errors.New("blobstore: key is required")
	}
	if strings.HasPrefix(key, "/") || strings.Contains(key, "../") || key == ".." {
		return "", fmt.Errorf("blobstore: invalid key %q", key)
	}
	rootValue := strings.TrimSpace(s.root)
	if rootValue == "" {
		return "", errors.New("blobstore: root is required")
	}
	root, err := filepath.Abs(rootValue)
	if err != nil {
		return "", fmt.Errorf("blobstore: resolve root: %w", err)
	}
	path := filepath.Join(root, filepath.FromSlash(key))
	if !strings.HasPrefix(path, root+string(os.PathSeparator)) && path != root {
		return "", fmt.Errorf("blobstore: key escapes root %q", key)
	}
	return path, nil
}

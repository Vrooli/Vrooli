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
)

type filesystemBlobStore struct {
	root string
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
	if _, err := io.Copy(tmp, r); err != nil {
		cleanup()
		return fmt.Errorf("blobstore: write blob: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("blobstore: close blob: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return fmt.Errorf("blobstore: commit blob: %w", err)
	}
	meta, err := json.Marshal(fileMeta{MIME: strings.TrimSpace(mime)})
	if err != nil {
		return fmt.Errorf("blobstore: encode metadata: %w", err)
	}
	if err := os.WriteFile(path+".meta", meta, 0o644); err != nil {
		return fmt.Errorf("blobstore: write metadata: %w", err)
	}
	return nil
}

func (s *filesystemBlobStore) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
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

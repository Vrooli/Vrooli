package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/vrooli/api-core/storage"
)

// FileContentDeps contains dependencies for text file content operations.
type FileContentDeps struct {
	FS      FileIO
	RepoDir string
}

// FileContentConflictError indicates optimistic-concurrency conflict.
type FileContentConflictError struct {
	Path        string
	CurrentHash string
}

func (e *FileContentConflictError) Error() string {
	path := e.Path
	if path == "" {
		path = "file"
	}
	return fmt.Sprintf("file content changed on disk: %s", path)
}

func hashContentBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// SaveFileContent updates a text file in the repo with optimistic concurrency.
func SaveFileContent(ctx context.Context, deps FileContentDeps, req SaveFileContentRequest) (*SaveFileContentResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return nil, fmt.Errorf("repo dir is required")
	}
	if deps.FS == nil {
		deps.FS = OSFileIO{}
	}

	cleanPath := cleanFilePath(req.Path)
	if cleanPath == "" || strings.HasPrefix(cleanPath, "..") {
		return nil, fmt.Errorf("invalid path")
	}

	absPath := filepath.Join(repoDir, cleanPath)
	info, err := deps.FS.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", cleanPath)
		}
		return nil, fmt.Errorf("stat file: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory")
	}
	if info.Size() > maxDiffFileBytes {
		return nil, &FileTooLargeError{Path: cleanPath, Size: info.Size(), Limit: maxDiffFileBytes}
	}

	currentBytes, err := deps.FS.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	if int64(len(currentBytes)) > maxDiffFileBytes {
		return nil, &FileTooLargeError{Path: cleanPath, Size: int64(len(currentBytes)), Limit: maxDiffFileBytes}
	}

	if kind := detectBinaryKind(cleanPath, currentBytes); kind != binaryNone {
		return nil, &UnsupportedBinaryError{Path: cleanPath}
	}
	if !utf8.Valid(currentBytes) {
		return nil, &UnsupportedBinaryError{Path: cleanPath}
	}

	currentHash := hashContentBytes(currentBytes)
	if strings.TrimSpace(req.ExpectedHash) != "" && req.ExpectedHash != currentHash {
		return nil, &FileContentConflictError{Path: cleanPath, CurrentHash: currentHash}
	}

	nextBytes := []byte(req.Content)
	if int64(len(nextBytes)) > maxDiffFileBytes {
		return nil, &FileTooLargeError{Path: cleanPath, Size: int64(len(nextBytes)), Limit: maxDiffFileBytes}
	}
	if !utf8.Valid(nextBytes) {
		return nil, &UnsupportedBinaryError{Path: cleanPath}
	}

	if err := storage.WriteFileAtomic(absPath, nextBytes, info.Mode().Perm()); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}

	return &SaveFileContentResponse{
		Success:      true,
		Path:         cleanPath,
		ContentHash:  hashContentBytes(nextBytes),
		BytesWritten: len(nextBytes),
		Timestamp:    time.Now().UTC(),
	}, nil
}

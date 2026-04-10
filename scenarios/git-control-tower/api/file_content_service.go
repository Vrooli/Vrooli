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

// validateFileForRead checks that a file exists, is not a directory, and is within size limits.
func validateFileForRead(fs FileIO, absPath, cleanPath string) (os.FileInfo, error) {
	info, err := fs.Stat(absPath)
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
	return info, nil
}

// validateTextContent ensures bytes are valid UTF-8 text and within size limits.
func validateTextContent(data []byte, cleanPath string) error {
	if int64(len(data)) > maxDiffFileBytes {
		return &FileTooLargeError{Path: cleanPath, Size: int64(len(data)), Limit: maxDiffFileBytes}
	}
	if !utf8.Valid(data) {
		return &UnsupportedBinaryError{Path: cleanPath}
	}
	return nil
}

// readAndValidateCurrentContent reads a file and validates it is editable text.
func readAndValidateCurrentContent(fs FileIO, absPath, cleanPath string) ([]byte, error) {
	currentBytes, err := fs.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	if err := validateTextContent(currentBytes, cleanPath); err != nil {
		return nil, err
	}
	if kind := detectBinaryKind(cleanPath, currentBytes); kind != binaryNone {
		return nil, &UnsupportedBinaryError{Path: cleanPath}
	}
	return currentBytes, nil
}

// validateSaveDeps validates and normalizes SaveFileContent dependencies.
func validateSaveDeps(deps *FileContentDeps) (string, error) {
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return "", fmt.Errorf("repo dir is required")
	}
	if deps.FS == nil {
		deps.FS = OSFileIO{}
	}
	return repoDir, nil
}

// checkOptimisticConcurrency verifies the expected hash matches the current file content.
func checkOptimisticConcurrency(currentBytes []byte, expectedHash, cleanPath string) error {
	if strings.TrimSpace(expectedHash) == "" {
		return nil
	}
	currentHash := hashContentBytes(currentBytes)
	if expectedHash != currentHash {
		return &FileContentConflictError{Path: cleanPath, CurrentHash: currentHash}
	}
	return nil
}

// SaveFileContent updates a text file in the repo with optimistic concurrency.
func SaveFileContent(ctx context.Context, deps FileContentDeps, req SaveFileContentRequest) (*SaveFileContentResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repoDir, err := validateSaveDeps(&deps)
	if err != nil {
		return nil, err
	}

	cleanPath := cleanFilePath(req.Path)
	if cleanPath == "" || strings.HasPrefix(cleanPath, "..") {
		return nil, fmt.Errorf("invalid path")
	}

	absPath := filepath.Join(repoDir, cleanPath)
	info, err := validateFileForRead(deps.FS, absPath, cleanPath)
	if err != nil {
		return nil, err
	}

	currentBytes, err := readAndValidateCurrentContent(deps.FS, absPath, cleanPath)
	if err != nil {
		return nil, err
	}

	if err := checkOptimisticConcurrency(currentBytes, req.ExpectedHash, cleanPath); err != nil {
		return nil, err
	}

	nextBytes := []byte(req.Content)
	if err := validateTextContent(nextBytes, cleanPath); err != nil {
		return nil, err
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

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	maxPrecommitDeclaredInputs     = 32
	maxPrecommitDeclaredInputBytes = 16 << 20
	maxPrecommitInputBytesTotal    = 64 << 20
)

func normalizePrecommitInputs(inputs []string) ([]string, error) {
	if len(inputs) > maxPrecommitDeclaredInputs {
		return nil, fmt.Errorf("at most %d declared check inputs are supported", maxPrecommitDeclaredInputs)
	}
	seen := make(map[string]struct{}, len(inputs))
	clean := make([]string, 0, len(inputs))
	for _, input := range inputs {
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		if strings.Contains(input, "\\") {
			return nil, fmt.Errorf("declared check inputs must use repository-relative slash paths")
		}
		path := filepath.ToSlash(filepath.Clean(input))
		first := strings.SplitN(path, "/", 2)[0]
		if path == "." || path == ".." || strings.HasPrefix(path, "../") || filepath.IsAbs(input) || strings.HasPrefix(path, "//") || strings.Contains(first, ":") {
			return nil, fmt.Errorf("declared check input must stay inside the repository")
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		clean = append(clean, path)
	}
	sort.Strings(clean)
	return clean, nil
}

func hashPrecommitInputs(ctx context.Context, repoDir string, inputs []string) (string, error) {
	normalized, err := normalizePrecommitInputs(inputs)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	rootPath, err := filepath.Abs(repoDir)
	if err != nil {
		return "", fmt.Errorf("resolve repository for check inputs")
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return "", fmt.Errorf("open repository for check inputs")
	}
	defer root.Close()
	total := 0
	for _, relative := range normalized {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		file, err := root.Open(filepath.FromSlash(relative))
		if err != nil {
			return "", fmt.Errorf("declared check input is unavailable or outside the repository: %s", relative)
		}
		before, statErr := file.Stat()
		if statErr != nil || !before.Mode().IsRegular() {
			file.Close()
			return "", fmt.Errorf("declared check input must be a regular file: %s", relative)
		}
		remaining := maxPrecommitInputBytesTotal - total
		limit := maxPrecommitDeclaredInputBytes
		if remaining < limit {
			limit = remaining
		}
		if before.Size() > int64(limit) {
			file.Close()
			return "", fmt.Errorf("declared check inputs exceed the supported size limit")
		}
		fileHash := sha256.New()
		written, copyErr := io.Copy(fileHash, io.LimitReader(&contextReader{ctx: ctx, r: file}, int64(limit)+1))
		after, afterErr := file.Stat()
		closeErr := file.Close()
		if copyErr != nil || afterErr != nil || closeErr != nil || written > int64(limit) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || written != before.Size() {
			return "", fmt.Errorf("declared check input changed while reading: %s", relative)
		}
		total += int(written)
		fmt.Fprintf(hash, "%s\x00%d\x00%s\x00", relative, written, hex.EncodeToString(fileHash.Sum(nil)))
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r *contextReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(buffer)
}

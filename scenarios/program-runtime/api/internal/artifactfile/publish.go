// Package artifactfile publishes one bounded artifact stream into an admitted
// Program Runtime workspace. The caller owns source identity and admission;
// this package owns path containment, size/hash accounting, and atomic publish.
package artifactfile

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const DefaultMaxBytes int64 = 64 << 20

type Receipt struct {
	Path   string `json:"output_path"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// Publish consumes stream once and atomically publishes it under workspace.
// It never buffers the artifact in memory. SizeBytes is producer-declared;
// SHA256 in the receipt is computed from the received bytes.
func Publish(workspace, relativePath string, stream io.Reader, sizeBytes, maxBytes int64, overwrite bool) (Receipt, error) {
	if stream == nil {
		return Receipt{}, errors.New("artifact stream is required")
	}
	if sizeBytes < 0 {
		return Receipt{}, errors.New("declared artifact size must be nonnegative")
	}
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}
	if sizeBytes > maxBytes {
		return Receipt{}, fmt.Errorf("declared artifact size %d exceeds %d-byte limit", sizeBytes, maxBytes)
	}
	rel, err := cleanRelative(relativePath)
	if err != nil {
		return Receipt{}, err
	}
	workspaceInfo, err := os.Lstat(workspace)
	if err != nil {
		return Receipt{}, fmt.Errorf("inspect admitted workspace: %w", err)
	}
	if !workspaceInfo.IsDir() || workspaceInfo.Mode()&os.ModeSymlink != 0 {
		return Receipt{}, errors.New("admitted workspace root must be a real directory")
	}
	root, err := os.OpenRoot(workspace)
	if err != nil {
		return Receipt{}, fmt.Errorf("open admitted workspace: %w", err)
	}
	defer root.Close()
	openedInfo, err := root.Stat(".")
	currentInfo, pathErr := os.Lstat(workspace)
	if err != nil || pathErr != nil || !currentInfo.IsDir() || currentInfo.Mode()&os.ModeSymlink != 0 || !os.SameFile(workspaceInfo, currentInfo) || !os.SameFile(currentInfo, openedInfo) {
		return Receipt{}, errors.New("admitted workspace changed while opening its file root")
	}
	parent := filepath.ToSlash(filepath.Dir(filepath.FromSlash(rel)))
	base := filepath.Base(filepath.FromSlash(rel))
	if parent == "." {
		parent = "."
	}
	if err := root.MkdirAll(parent, 0o700); err != nil {
		return Receipt{}, fmt.Errorf("create destination directory: %w", err)
	}
	dir, err := root.OpenRoot(parent)
	if err != nil {
		return Receipt{}, fmt.Errorf("open destination directory: %w", err)
	}
	defer dir.Close()
	temp, err := createTemp(dir)
	if err != nil {
		return Receipt{}, err
	}
	tempName := filepath.Base(temp.Name())
	published := false
	defer func() {
		_ = temp.Close()
		if !published {
			_ = dir.Remove(tempName)
		}
	}()
	hash := sha256.New()
	count, err := io.Copy(io.MultiWriter(temp, hash), io.LimitReader(stream, maxBytes+1))
	if err != nil {
		return Receipt{}, fmt.Errorf("stream artifact: %w", err)
	}
	if count > maxBytes {
		return Receipt{}, fmt.Errorf("artifact stream exceeds %d-byte limit", maxBytes)
	}
	if count != sizeBytes {
		return Receipt{}, fmt.Errorf("artifact byte count %d does not match declared size %d", count, sizeBytes)
	}
	if err := temp.Sync(); err != nil {
		return Receipt{}, fmt.Errorf("sync artifact temporary file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return Receipt{}, fmt.Errorf("close artifact temporary file: %w", err)
	}
	if overwrite {
		if err := dir.Rename(tempName, base); err != nil {
			return Receipt{}, fmt.Errorf("publish artifact: %w", err)
		}
	} else {
		// A same-directory hard link is an atomic no-replace publication. It
		// cannot overwrite a path created after the caller's earlier checks.
		if err := dir.Link(tempName, base); err != nil {
			return Receipt{}, fmt.Errorf("publish artifact without overwrite: %w", err)
		}
		if err := dir.Remove(tempName); err != nil {
			_ = dir.Remove(base)
			return Receipt{}, fmt.Errorf("remove published temporary link: %w", err)
		}
	}
	published = true
	if directory, err := dir.Open("."); err == nil {
		_ = directory.Sync()
		_ = directory.Close()
	}
	return Receipt{Path: rel, Bytes: count, SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

func cleanRelative(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsRune(value, '\x00') || filepath.IsAbs(value) || strings.HasPrefix(value, "/") {
		return "", errors.New("destination must be a nonempty relative path")
	}
	clean := filepath.Clean(filepath.FromSlash(value))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("destination escapes the admitted workspace")
	}
	return filepath.ToSlash(clean), nil
}

func createTemp(dir *os.Root) (*os.File, error) {
	for attempt := 0; attempt < 8; attempt++ {
		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			return nil, fmt.Errorf("create artifact temp name: %w", err)
		}
		name := ".program-artifact-" + hex.EncodeToString(random[:]) + ".tmp"
		file, err := dir.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			return file, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("create artifact temp file: %w", err)
		}
	}
	return nil, errors.New("could not allocate unique artifact temp file")
}

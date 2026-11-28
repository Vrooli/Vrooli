package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	maxArchiveEntries      = 10_000
	maxArchiveEntryBytes   = int64(256 << 20)
	maxArchiveTotalBytes   = int64(1 << 30)
	defaultArchiveFileMode = 0o600
)

type archiveLimits struct {
	maxEntries    int
	maxEntryBytes int64
	maxTotalBytes int64
}

var defaultArchiveLimits = archiveLimits{
	maxEntries:    maxArchiveEntries,
	maxEntryBytes: maxArchiveEntryBytes,
	maxTotalBytes: maxArchiveTotalBytes,
}

var (
	errArchiveTooLarge    = errors.New("archive extraction limit exceeded")
	errUnsafeArchivePath  = errors.New("unsafe archive entry")
	errUnsupportedArchive = errors.New("unsupported archive format")
)

func extractArchive(archivePath, destination string) ([]string, int64, error) {
	return extractArchiveWithLimits(archivePath, destination, defaultArchiveLimits)
}

func extractArchiveWithLimits(archivePath, destination string, limits archiveLimits) ([]string, int64, error) {
	if archivePath == "" || destination == "" {
		return nil, 0, fmt.Errorf("archive and destination paths are required")
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return nil, 0, fmt.Errorf("create destination: %w", err)
	}
	root, err := os.OpenRoot(destination)
	if err != nil {
		return nil, 0, fmt.Errorf("open destination: %w", err)
	}
	defer root.Close()

	ext := strings.ToLower(filepath.Ext(archivePath))
	var files []string
	var total int64
	entryCount := 0
	addEntry := func() error {
		entryCount++
		if entryCount > limits.maxEntries {
			return errArchiveTooLarge
		}
		return nil
	}
	remaining := func() int64 { return limits.maxTotalBytes - total }
	outputPath := func(name string, isDir bool, mode os.FileMode) (string, error) {
		rel, err := archiveEntryRelative(name, isDir)
		if err != nil {
			return "", err
		}
		if rel == "." {
			return rel, nil
		}
		if isDir {
			if mode == 0 {
				mode = 0o755
			}
			if err := root.MkdirAll(rel, mode.Perm()); err != nil {
				return "", fmt.Errorf("create archive directory: %w", err)
			}
			return rel, nil
		}
		if err := root.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
			return "", fmt.Errorf("create archive parent: %w", err)
		}
		return rel, nil
	}
	createFile := func(rel string, mode os.FileMode) (*os.File, error) {
		if mode == 0 {
			mode = defaultArchiveFileMode
		}
		// O_EXCL prevents an archive entry from replacing an existing file or
		// following a final-component symlink. os.Root confines parents too.
		return root.OpenFile(rel, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode.Perm())
	}
	writeFile := func(rel string, mode os.FileMode, source io.Reader, declaredSize uint64) (int64, error) {
		allowed := remaining()
		if allowed > limits.maxEntryBytes {
			allowed = limits.maxEntryBytes
		}
		if declaredSize > uint64(allowed) {
			return 0, errArchiveTooLarge
		}
		target, err := createFile(rel, mode)
		if err != nil {
			return 0, fmt.Errorf("create extracted file: %w", err)
		}
		written, copyErr := copyArchiveEntry(target, source, int64(declaredSize), allowed)
		closeErr := target.Close()
		if copyErr != nil || closeErr != nil {
			_ = root.Remove(rel)
			if copyErr != nil {
				return written, copyErr
			}
			return written, fmt.Errorf("close extracted file: %w", closeErr)
		}
		total += written
		files = append(files, filepath.Join(destination, rel))
		return written, nil
	}

	switch ext {
	case ".zip":
		reader, err := zip.OpenReader(archivePath)
		if err != nil {
			return nil, 0, fmt.Errorf("open zip: %w", err)
		}
		defer reader.Close()
		for _, entry := range reader.File {
			if err := addEntry(); err != nil {
				return nil, total, err
			}
			isDir := entry.FileInfo().IsDir()
			mode := entry.Mode()
			if mode&os.ModeSymlink != 0 || (mode.Type() != 0 && !isDir && !mode.IsRegular()) {
				return nil, total, errUnsafeArchivePath
			}
			rel, err := outputPath(entry.Name, isDir, mode)
			if err != nil {
				return nil, total, fmt.Errorf("%w: %s", errUnsafeArchivePath, entry.Name)
			}
			if isDir {
				continue
			}
			if entry.UncompressedSize64 > uint64(limits.maxEntryBytes) || entry.UncompressedSize64 > uint64(remaining()) {
				return nil, total, errArchiveTooLarge
			}
			source, err := entry.Open()
			if err != nil {
				return nil, total, fmt.Errorf("open zip entry: %w", err)
			}
			_, writeErr := writeFile(rel, mode, source, entry.UncompressedSize64)
			closeErr := source.Close()
			if writeErr != nil {
				return nil, total, writeErr
			}
			if closeErr != nil {
				return nil, total, fmt.Errorf("close zip entry: %w", closeErr)
			}
		}

	case ".tar", ".gz", ".tgz":
		archive, err := os.Open(archivePath)
		if err != nil {
			return nil, 0, fmt.Errorf("open tar: %w", err)
		}
		defer archive.Close()
		var source io.Reader = archive
		var gzipReader *gzip.Reader
		if ext == ".gz" || ext == ".tgz" {
			gzipReader, err = gzip.NewReader(archive)
			if err != nil {
				return nil, total, fmt.Errorf("open gzip: %w", err)
			}
			defer gzipReader.Close()
			source = gzipReader
		}
		reader := tar.NewReader(source)
		for {
			header, err := reader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, total, fmt.Errorf("read tar entry: %w", err)
			}
			if err := addEntry(); err != nil {
				return nil, total, err
			}
			isDir := header.Typeflag == tar.TypeDir
			isFile := header.Typeflag == tar.TypeReg || header.Typeflag == tar.TypeRegA
			if !isDir && !isFile {
				return nil, total, fmt.Errorf("%w: tar entry type %d", errUnsafeArchivePath, header.Typeflag)
			}
			rel, err := outputPath(header.Name, isDir, os.FileMode(header.Mode))
			if err != nil {
				return nil, total, fmt.Errorf("%w: %s", errUnsafeArchivePath, header.Name)
			}
			if isDir {
				continue
			}
			if header.Size < 0 || header.Size > limits.maxEntryBytes || header.Size > remaining() {
				return nil, total, errArchiveTooLarge
			}
			if _, err := writeFile(rel, os.FileMode(header.Mode), reader, uint64(header.Size)); err != nil {
				return nil, total, err
			}
		}
	default:
		return nil, 0, errUnsupportedArchive
	}
	return files, total, nil
}

func archiveEntryRelative(name string, isDir bool) (string, error) {
	if name == "" || strings.ContainsRune(name, '\x00') {
		return "", errUnsafeArchivePath
	}
	name = strings.ReplaceAll(name, `\`, "/")
	if strings.HasPrefix(name, "/") || hasWindowsVolume(name) {
		return "", errUnsafeArchivePath
	}
	clean := path.Clean(name)
	if clean == "." && isDir {
		return clean, nil
	}
	if !filepath.IsLocal(filepath.FromSlash(clean)) {
		return "", errUnsafeArchivePath
	}
	return filepath.FromSlash(clean), nil
}

func hasWindowsVolume(name string) bool {
	if len(name) < 2 || name[1] != ':' {
		return false
	}
	first := name[0]
	return (first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z')
}

func copyArchiveEntry(dst io.Writer, src io.Reader, declaredSize, maxAllowed int64) (int64, error) {
	if declaredSize < 0 || maxAllowed < 0 || declaredSize > maxAllowed {
		return 0, errArchiveTooLarge
	}
	written, err := io.CopyN(dst, src, declaredSize)
	if err != nil {
		return written, fmt.Errorf("read archive entry: %w", err)
	}
	var extra [1]byte
	n, probeErr := io.ReadFull(src, extra[:])
	if n > 0 {
		return written, errArchiveTooLarge
	}
	if probeErr != nil && probeErr != io.EOF {
		return written, fmt.Errorf("check archive entry size: %w", probeErr)
	}
	return written, nil
}

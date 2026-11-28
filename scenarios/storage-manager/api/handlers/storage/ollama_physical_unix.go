//go:build aix || android || darwin || dragonfly || freebsd || hurd || illumos || linux || netbsd || openbsd || solaris

package storage

import (
	"encoding/hex"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type ollamaBlobPhysicalInventory struct {
	Count             int64
	LogicalBytes      int64
	PhysicalBytes     int64
	UnrecognizedFiles int64
}

type physicalFileIdentity struct {
	device uint64
	inode  uint64
}

// directoryAllocatedBytes measures allocated blocks beneath one owner root.
// WalkDir plus Lstat never follows symlinks; hard-linked files are counted
// once, and mount points on other filesystems are excluded.
func directoryAllocatedBytes(root string) (int64, error) {
	info, err := os.Lstat(root)
	if err != nil {
		return 0, fmt.Errorf("inspect Ollama model root %q: %w", root, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return 0, fmt.Errorf("Ollama model root %q must be a real directory", root)
	}
	rootStat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("Ollama model root %q has no filesystem identity", root)
	}
	rootDevice := uint64(rootStat.Dev)
	seen := map[physicalFileIdentity]struct{}{}
	var total int64
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		fileInfo, err := os.Lstat(path)
		if err != nil {
			return err
		}
		stat, ok := fileInfo.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("%s has no allocated-block metadata", path)
		}
		if uint64(stat.Dev) != rootDevice {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if stat.Blocks < 0 || stat.Blocks > math.MaxInt64/512 {
			return fmt.Errorf("%s has an invalid allocated-block count", path)
		}
		if stat.Nlink > 1 {
			id := physicalFileIdentity{device: uint64(stat.Dev), inode: uint64(stat.Ino)}
			if _, duplicate := seen[id]; duplicate {
				return nil
			}
			seen[id] = struct{}{}
		}
		bytes := int64(stat.Blocks) * 512
		if total > math.MaxInt64-bytes {
			return fmt.Errorf("allocated bytes overflow under %s", root)
		}
		total += bytes
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("walk Ollama model root allocated bytes: %w", err)
	}
	return total, nil
}

// inspectOllamaBlobs measures each content-addressed blob once. Reported
// logical tag sizes are deliberately not used to derive these physical bytes.
func inspectOllamaBlobs(root string) (ollamaBlobPhysicalInventory, error) {
	var out ollamaBlobPhysicalInventory
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return out, fmt.Errorf("inspect Ollama blob directory %q: %w", root, err)
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return out, fmt.Errorf("Ollama blob path %q must be a real directory", root)
	}
	rootStat, ok := rootInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return out, fmt.Errorf("Ollama blob directory %q has no filesystem identity", root)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return out, fmt.Errorf("read Ollama blob directory %q: %w", root, err)
	}
	seenDigests := map[string]struct{}{}
	seenInodes := map[physicalFileIdentity]struct{}{}
	for _, entry := range entries {
		name := entry.Name()
		digest := strings.TrimPrefix(name, "sha256-")
		if len(digest) != 64 || hex.DecodedLen(len(digest)) != 32 {
			out.UnrecognizedFiles++
			continue
		}
		if _, err := hex.DecodeString(digest); err != nil {
			out.UnrecognizedFiles++
			continue
		}
		path := filepath.Join(root, name)
		info, err := os.Lstat(path)
		if err != nil {
			return out, fmt.Errorf("inspect Ollama blob %q: %w", path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return out, fmt.Errorf("Ollama content-addressed blob %q is not a regular file", path)
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || uint64(stat.Dev) != uint64(rootStat.Dev) {
			return out, fmt.Errorf("Ollama content-addressed blob %q has unknown filesystem identity", path)
		}
		if _, duplicate := seenDigests[digest]; duplicate {
			return out, fmt.Errorf("duplicate Ollama blob digest %s", digest)
		}
		seenDigests[digest] = struct{}{}
		if out.LogicalBytes > math.MaxInt64-info.Size() || info.Size() < 0 {
			return out, fmt.Errorf("Ollama blob logical byte count overflow at %q", path)
		}
		out.LogicalBytes += info.Size()
		out.Count++
		if stat.Blocks < 0 || stat.Blocks > math.MaxInt64/512 {
			return out, fmt.Errorf("Ollama blob %q has an invalid allocated-block count", path)
		}
		if stat.Nlink > 1 {
			id := physicalFileIdentity{device: uint64(stat.Dev), inode: uint64(stat.Ino)}
			if _, duplicate := seenInodes[id]; duplicate {
				continue
			}
			seenInodes[id] = struct{}{}
		}
		allocated := int64(stat.Blocks) * 512
		if out.PhysicalBytes > math.MaxInt64-allocated {
			return out, fmt.Errorf("Ollama blob allocated-byte count overflow")
		}
		out.PhysicalBytes += allocated
	}
	return out, nil
}

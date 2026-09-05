//go:build !windows

package hostinventory

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/vrooli/vrooli/internal/tuning"
)

func platformStorageMounts() ([]storageMount, error) {
	if runtimeMountFile, ok := mountTablePath(); ok {
		file, err := os.Open(runtimeMountFile)
		if err == nil {
			defer file.Close()
			mounts := make([]storageMount, 0)
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				fields := strings.Fields(scanner.Text())
				if len(fields) < 3 || !isUsableFilesystem(fields[2]) {
					continue
				}
				location := strings.ReplaceAll(fields[1], "\\040", " ")
				if info, statErr := os.Stat(location); statErr != nil || !info.IsDir() {
					continue
				}
				kind := "local"
				if strings.HasPrefix(fields[0], "/dev/") {
					kind = "removable-or-local"
				}
				mounts = append(mounts, storageMount{Location: location, Kind: kind, Filesystem: fields[2]})
			}
			if err := scanner.Err(); err != nil {
				return nil, err
			}
			return dedupeStorageMounts(mounts), nil
		}
	}
	// macOS and other Unix platforms do not expose /proc/mounts. /Volumes is
	// the conventional removable-volume root; inability to enumerate it is an
	// explicit degraded result rather than a guessed temporary directory.
	volumes := "/Volumes"
	entries, err := os.ReadDir(volumes)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("enumerate mounted volumes: %w", err)
	}
	mounts := make([]storageMount, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			mounts = append(mounts, storageMount{Location: filepath.Join(volumes, entry.Name()), Kind: "removable-or-local"})
		}
	}
	return dedupeStorageMounts(mounts), nil
}

func mountTablePath() (string, bool) {
	if _, err := os.Stat("/proc/mounts"); err == nil {
		return "/proc/mounts", true
	}
	return "", false
}

func isUsableFilesystem(value string) bool {
	switch strings.ToLower(value) {
	case "proc", "sysfs", "devtmpfs", "devpts", "cgroup", "cgroup2", "pstore", "securityfs", "debugfs", "tracefs", "configfs", "fusectl", "mqueue", "hugetlbfs", "overlay", "squashfs", "ramfs", "tmpfs", "binfmt_misc", "efivarfs", "bpf", "autofs":
		return false
	default:
		return value != "" && !strings.HasPrefix(strings.ToLower(value), "fuse.")
	}
}

func dedupeStorageMounts(input []storageMount) []storageMount {
	seen := map[string]struct{}{}
	output := make([]storageMount, 0, len(input))
	for _, mount := range input {
		key := filepath.Clean(mount.Location)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		output = append(output, mount)
	}
	return output
}

func physicalDeviceIdentity(path string) (string, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return "", false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("dev:%d", stat.Dev), true
}

func resolvePathForStorage(path string) (string, error) {
	path, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return filepath.Clean(resolved), nil
	}
	parent := filepath.Dir(path)
	if parent == path {
		return path, nil
	}
	resolvedParent, err := resolvePathForStorage(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(resolvedParent, filepath.Base(path)), nil
}

func probeWritableDirectory(path string) error {
	if err := os.MkdirAll(path, tuning.PermPrivateDir); err != nil {
		return err
	}
	probe, err := os.CreateTemp(path, ".vrooli-escrow-probe-*")
	if err != nil {
		return err
	}
	name := probe.Name()
	if err := probe.Chmod(tuning.PermSecret); err != nil {
		_ = probe.Close()
		_ = os.Remove(name)
		return err
	}
	if err := probe.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	return os.Remove(name)
}

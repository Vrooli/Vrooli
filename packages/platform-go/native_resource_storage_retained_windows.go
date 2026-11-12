//go:build windows

package platform

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"golang.org/x/sys/windows"
)

// Read existing protected publisher objects only; no new pointer or registry.
// Unknown, foreign, inaccessible or excessive authority is an error, not absence.
func windowsRetainedNativeResourceStorageOptions(ctx context.Context) (NativeSystemServiceBrokerOptions, bool, error) {
	var zero NativeSystemServiceBrokerOptions
	roots, err := nativeWindowsSystemBrokerRoots()
	if err != nil {
		return zero, false, err
	}
	base := filepath.Join(roots.Install, "versions")
	held, err := windowsEnsureExistingBrokerParents(base)
	if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) || errors.Is(err, windows.ERROR_PATH_NOT_FOUND) {
		return zero, false, nil
	}
	if err != nil {
		return zero, false, err
	}
	defer func() {
		for _, h := range held {
			windows.CloseHandle(h)
		}
	}()
	tokenUser, err := windows.GetCurrentThreadEffectiveToken().GetTokenUser()
	if err != nil {
		return zero, false, err
	}
	paths, err := windowsRetainedNativeDescriptorPaths(base)
	if err != nil {
		return zero, false, err
	}
	var descriptorBytes int64
	options, configured, err := retainedNativeResourceStorageOptionsWith(ctx, "windows:"+tokenUser.User.Sid.String(), paths, func(path string) (NativeSystemServiceBrokerOptions, error) {
		// Namespace observation needs only bounded immutable metadata. Executable
		// opening/boot paths retain their separate full image proof.
		h, e := windowsOpenBrokerPath(path, false, windows.FILE_READ_ATTRIBUTES)
		if e != nil {
			return zero, e
		}
		file := os.NewFile(uintptr(h), path)
		info, e := file.Stat()
		file.Close()
		if e != nil {
			return zero, e
		}
		if !info.Mode().IsRegular() || info.Size() <= 0 {
			return zero, ErrOperationIdentity
		}
		descriptorBytes += info.Size()
		if descriptorBytes > 16<<20 {
			return zero, ErrOperationIdentity
		}

		d, _, e := windowsReadBrokerDescriptorWithImageProof(path, false)
		return d.Options, e
	})
	if err != nil {
		return zero, false, err
	}
	after, err := windowsRetainedNativeDescriptorPaths(base)
	if err != nil {
		return zero, false, err
	}
	if !slices.Equal(paths, after) {
		return zero, false, ErrOperationIdentity
	}
	return options, configured, nil
}

func windowsRetainedNativeDescriptorPaths(base string) ([]string, error) {
	versions, err := boundedNativeBrokerDirectory(base, 128)
	if err != nil {
		return nil, err
	}
	var paths []string
	total := 0
	for _, version := range versions {
		if !validNativeBrokerDigest(version.Name()) || !version.IsDir() {
			return nil, ErrOperationIdentity
		}
		dir := filepath.Join(base, version.Name())
		held, err := windowsEnsureExistingBrokerParents(dir)
		if err != nil {
			return nil, err
		}
		files, err := boundedNativeBrokerDirectory(dir, 256)
		for _, h := range held {
			windows.CloseHandle(h)
		}
		if err != nil {
			return nil, err
		}
		total += len(files)
		if total > 512 {
			return nil, ErrOperationIdentity
		}
		for _, file := range files {
			name := file.Name()
			if knownNativeBrokerUnpublishedStage(name) {
				if !file.Type().IsRegular() {
					return nil, ErrOperationIdentity
				}
				h, e := windowsOpenBrokerPath(filepath.Join(dir, name), false, windows.FILE_READ_ATTRIBUTES)
				if e != nil {
					return nil, e
				}
				windows.CloseHandle(h)
				continue
			}

			if name == "vrooli.exe" && file.Type().IsRegular() {
				continue
			}
			if !strings.HasPrefix(name, "broker-") || !strings.HasSuffix(name, ".json") || !validNativeBrokerDigest(strings.TrimSuffix(strings.TrimPrefix(name, "broker-"), ".json")) || !file.Type().IsRegular() {
				return nil, ErrOperationIdentity
			}
			paths = append(paths, filepath.Join(dir, name))
			if len(paths) > 256 {
				return nil, ErrOperationIdentity
			}
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func boundedNativeBrokerDirectory(path string, limit int) ([]os.DirEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	entries, err := file.ReadDir(limit + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if len(entries) > limit {
		return nil, ErrOperationIdentity
	}
	return entries, nil
}

package bountycases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

const maxBundleRegistrationBytes = 1 << 20

// LoadDirectory imports owner-managed immutable JSON registrations from one
// fixed directory. Replaying an already registered file is idempotent; each
// new version must append to that case's prior version. The directory must
// live under the configured trusted corpus root and is never selected by an
// RPC caller.
func LoadDirectory(ctx context.Context, service Service, registrationDir string) ([]CaseBundle, error) {
	if service == nil {
		return nil, errors.New("bounty case registration service is required")
	}
	info, err := os.Lstat(registrationDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []CaseBundle{}, nil
		}
		return nil, fmt.Errorf("inspect bounty case registration directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, errors.New("bounty case registration path must be a regular directory")
	}
	entries, err := os.ReadDir(registrationDir)
	if err != nil {
		return nil, fmt.Errorf("read bounty case registrations: %w", err)
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 || entry.IsDir() {
			return nil, fmt.Errorf("registration %q must be a regular JSON file", entry.Name())
		}
		paths = append(paths, filepath.Join(registrationDir, entry.Name()))
	}
	sort.Strings(paths)
	registered := make([]CaseBundle, 0, len(paths))
	for _, filePath := range paths {
		fileInfo, err := os.Lstat(filePath)
		if err != nil || !fileInfo.Mode().IsRegular() || fileInfo.Size() > maxBundleRegistrationBytes {
			return nil, fmt.Errorf("registration %q must be a regular file no larger than %d bytes", filepath.Base(filePath), maxBundleRegistrationBytes)
		}
		file, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("open bounty case registration %q: %w", filepath.Base(filePath), err)
		}
		decoder := json.NewDecoder(io.LimitReader(file, maxBundleRegistrationBytes+1))
		decoder.DisallowUnknownFields()
		var bundle CaseBundle
		err = decoder.Decode(&bundle)
		if err == nil {
			var trailing any
			if decoder.Decode(&trailing) != io.EOF {
				err = errors.New("registration contains trailing JSON data")
			}
		}
		closeErr := file.Close()
		if err != nil {
			return nil, fmt.Errorf("decode bounty case registration %q: %w", filepath.Base(filePath), err)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close bounty case registration %q: %w", filepath.Base(filePath), closeErr)
		}
		item, err := service.Register(ctx, bundle)
		if err != nil {
			return nil, fmt.Errorf("register bounty case from %q: %w", filepath.Base(filePath), err)
		}
		registered = append(registered, item)
	}
	return registered, nil
}

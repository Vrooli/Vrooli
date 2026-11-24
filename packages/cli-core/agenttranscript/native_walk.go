package agenttranscript

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
)

// NativeWalkOptions controls the differences between source catalogs. The
// default walk skips non-regular entries and reports filesystem errors.
type NativeWalkOptions struct {
	VisitNonRegularEntries bool
	Errors                 NativeWalkErrorPolicy
}

// NativeWalkErrorPolicy preserves the two supported source failure modes.
type NativeWalkErrorPolicy uint8

const (
	NativeWalkFailOnError NativeWalkErrorPolicy = iota
	NativeWalkSkipMissingRoot
	NativeWalkSkipUnreadableSubdirectories
)

// WalkNativeFiles walks root using filepath.WalkDir and calls visit for regular
// files. A source may opt into non-regular entries when its established
// discovery behavior inspects them later. Symlink directories are never
// traversed; each consumer retains file-name, harness, and identity policy.
func WalkNativeFiles(ctx context.Context, root string, options NativeWalkOptions, visit func(path string, entry fs.DirEntry, info os.FileInfo) error) error {
	if root == "" || visit == nil {
		return nil
	}
	root = filepath.Clean(root)
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nativeWalkError(options.Errors, root, path, entry, walkErr)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() && !options.VisitNonRegularEntries {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return visit(path, entry, info)
	})
}

func nativeWalkError(policy NativeWalkErrorPolicy, root, path string, entry fs.DirEntry, err error) error {
	switch policy {
	case NativeWalkSkipMissingRoot:
		if path == root && os.IsNotExist(err) {
			return nil
		}
	case NativeWalkSkipUnreadableSubdirectories:
		if entry != nil && entry.IsDir() && path != root {
			return filepath.SkipDir
		}
		if entry == nil && path == root && os.IsNotExist(err) {
			return nil
		}
		if entry == nil && path == root {
			return err
		}
		return nil
	}
	return err
}

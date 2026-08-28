package storage

import (
	"os"
	"path/filepath"
)

const (
	// DefaultDirPerm is used when creating storage directories.
	DefaultDirPerm os.FileMode = 0o755
	// DefaultFilePerm is used for regular storage files.
	DefaultFilePerm os.FileMode = 0o644
	// SecretFilePerm is used for secret-bearing files.
	SecretFilePerm os.FileMode = 0o600
)

// Internal filesystem seams for deterministic failure-path tests.
// Keep package-private to avoid API surface expansion.
var (
	mkdirAllFn   = os.MkdirAll
	createTempFn = os.CreateTemp
	chmodFn      = os.Chmod
	renameFn     = os.Rename
	removeFn     = os.Remove
	writeFn      = func(f *os.File, b []byte) (int, error) { return f.Write(b) }
	syncFn       = func(f *os.File) error { return f.Sync() }
	closeFn      = func(f *os.File) error { return f.Close() }
)

// EnsureClassDir creates class directory if missing and returns its absolute path.
//
// When perm is 0, DefaultDirPerm is used.
func EnsureClassDir(r *Resolver, opts Options, class Class, perm os.FileMode) (string, error) {
	paths, err := r.Resolve(opts)
	if err != nil {
		return "", err
	}
	p, err := paths.ForClass(class)
	if err != nil {
		return "", err
	}
	if perm == 0 {
		perm = DefaultDirPerm
	}
	if err := mkdirAllFn(p, perm); err != nil {
		return "", &Error{Kind: ErrIO, Message: "create class directory", Details: p, Err: err}
	}
	return p, nil
}

// EnsureAllDirs creates all class directories for scenario and returns resolved Paths.
//
// When perm is 0, DefaultDirPerm is used.
func EnsureAllDirs(r *Resolver, opts Options, perm os.FileMode) (Paths, error) {
	paths, err := r.Resolve(opts)
	if err != nil {
		return Paths{}, err
	}
	if perm == 0 {
		perm = DefaultDirPerm
	}
	for _, p := range []string{paths.ConfigDir, paths.DataDir, paths.CacheDir, paths.LogsDir, paths.StateDir} {
		if err := mkdirAllFn(p, perm); err != nil {
			return Paths{}, &Error{Kind: ErrIO, Message: "create storage directory", Details: p, Err: err}
		}
	}
	return paths, nil
}

// WriteFileAtomic writes a file atomically by writing to a temp file in the same
// directory and renaming it into place.
//
// Semantics:
//   - parent directory is created if needed
//   - temp file is fsync'd before rename
//   - target file permissions are set before rename
//
// When perm is 0, DefaultFilePerm is used.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := mkdirAllFn(dir, DefaultDirPerm); err != nil {
		return &Error{Kind: ErrIO, Message: "create parent directory", Details: dir, Err: err}
	}
	if perm == 0 {
		perm = DefaultFilePerm
	}

	tmp, err := createTempFn(dir, ".tmp-*")
	if err != nil {
		return &Error{Kind: ErrIO, Message: "create temp file", Details: path, Err: err}
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = closeFn(tmp)
		_ = removeFn(tmpPath)
	}

	if _, err := writeFn(tmp, data); err != nil {
		cleanup()
		return &Error{Kind: ErrIO, Message: "write temp file", Details: path, Err: err}
	}
	if err := syncFn(tmp); err != nil {
		cleanup()
		return &Error{Kind: ErrIO, Message: "sync temp file", Details: path, Err: err}
	}
	if err := closeFn(tmp); err != nil {
		cleanup()
		return &Error{Kind: ErrIO, Message: "close temp file", Details: path, Err: err}
	}
	if err := chmodFn(tmpPath, perm); err != nil {
		cleanup()
		return &Error{Kind: ErrIO, Message: "chmod temp file", Details: path, Err: err}
	}
	if err := renameFn(tmpPath, path); err != nil {
		cleanup()
		return &Error{Kind: ErrIO, Message: "rename temp file", Details: path, Err: err}
	}
	return nil
}

// EnsureDirectory creates a directory through the reviewed storage seam.
func EnsureDirectory(path string, perm os.FileMode) error {
	if perm == 0 {
		perm = DefaultDirPerm
	}
	if err := mkdirAllFn(path, perm); err != nil {
		return &Error{Kind: ErrIO, Message: "create directory", Details: path, Err: err}
	}
	return nil
}

// OpenAppendFile opens a managed append-only file after creating its parent.
// Callers remain responsible for closing the returned handle.
func OpenAppendFile(path string, perm os.FileMode) (*os.File, error) {
	if err := EnsureDirectory(filepath.Dir(path), DefaultDirPerm); err != nil {
		return nil, err
	}
	if perm == 0 {
		perm = DefaultFilePerm
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, perm)
	if err != nil {
		return nil, &Error{Kind: ErrIO, Message: "open append file", Details: path, Err: err}
	}
	return file, nil
}

// RenameFile and RemoveFile keep managed filesystem mutations behind the
// storage package's reviewed seam.
func RenameFile(source, destination string) error {
	if err := renameFn(source, destination); err != nil {
		return &Error{Kind: ErrIO, Message: "rename file", Details: source + " -> " + destination, Err: err}
	}
	return nil
}

func RemoveFile(path string) error {
	if err := removeFn(path); err != nil {
		return &Error{Kind: ErrIO, Message: "remove file", Details: path, Err: err}
	}
	return nil
}

package storage

import (
	"os"
	"path/filepath"
)

const (
	DefaultDirPerm  = 0o755
	DefaultFilePerm = 0o644
)

func EnsureAllDirs(r *Resolver, opts Options, perm os.FileMode) (Paths, error) {
	if perm == 0 {
		perm = DefaultDirPerm
	}
	paths, err := r.Resolve(opts)
	if err != nil {
		return Paths{}, err
	}
	for _, dir := range []string{paths.ConfigDir, paths.DataDir, paths.CacheDir, paths.LogsDir, paths.StateDir} {
		if err := os.MkdirAll(dir, perm); err != nil {
			return Paths{}, &Error{Kind: ErrResolve, Message: "ensure storage dir", Details: dir, Err: err}
		}
	}
	return paths, nil
}

func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	if perm == 0 {
		perm = DefaultFilePerm
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, DefaultDirPerm); err != nil {
		return &Error{Kind: ErrResolve, Message: "ensure parent dir", Details: dir, Err: err}
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return &Error{Kind: ErrResolve, Message: "create temp file", Details: dir, Err: err}
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return &Error{Kind: ErrResolve, Message: "write temp file", Details: tmpName, Err: err}
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return &Error{Kind: ErrResolve, Message: "chmod temp file", Details: tmpName, Err: err}
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return &Error{Kind: ErrResolve, Message: "sync temp file", Details: tmpName, Err: err}
	}
	if err := tmp.Close(); err != nil {
		return &Error{Kind: ErrResolve, Message: "close temp file", Details: tmpName, Err: err}
	}
	if err := os.Rename(tmpName, path); err != nil {
		return &Error{Kind: ErrResolve, Message: "rename temp file", Details: path, Err: err}
	}
	return nil
}


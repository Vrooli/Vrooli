package graphreconcile

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/storage"
)

// rootedPath converts an absolute path into a slash-separated path relative
// to rootPath. Callers must still perform IO through os.Root after this check.
func rootedPath(rootPath, absolute string) (string, error) {
	rootPath, err := filepath.Abs(rootPath)
	if err != nil {
		return "", err
	}
	absolute, err = filepath.Abs(absolute)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(rootPath, absolute)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q is outside root %q", absolute, rootPath)
	}
	return filepath.ToSlash(relative), nil
}

// rootedGlob limits discovery to a lexical subtree and leaves opening each
// result to the corresponding rooted read helper. Glob itself only enumerates
// names; contents are always opened through os.Root.
func rootedGlob(rootPath, pattern string) ([]string, error) {
	rootPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}
	pattern, err = filepath.Abs(pattern)
	if err != nil {
		return nil, err
	}
	prefix := pattern
	if index := strings.IndexAny(prefix, "*?["); index >= 0 {
		prefix = filepath.Dir(prefix[:index])
	}
	if _, err := rootedPath(rootPath, prefix); err != nil {
		return nil, err
	}
	return filepath.Glob(pattern)
}

func rootedReadFile(rootPath, absolute string) ([]byte, error) {
	relative, err := rootedPath(rootPath, absolute)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	file, err := root.Open(relative)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

func rootedReadDir(rootPath, absolute string) ([]os.DirEntry, error) {
	relative, err := rootedPath(rootPath, absolute)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	file, err := root.Open(relative)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return file.ReadDir(-1)
}

func rootedMkdirTemp(rootPath, parent, prefix string) (string, error) {
	relativeParent, err := rootedPath(rootPath, parent)
	if err != nil {
		return "", err
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return "", err
	}
	defer root.Close()
	for attempt := 0; attempt < 10; attempt++ {
		var token [12]byte
		if _, err := rand.Read(token[:]); err != nil {
			return "", err
		}
		name := prefix + hex.EncodeToString(token[:])
		relative := filepath.ToSlash(filepath.Join(relativeParent, name))
		if err := root.Mkdir(relative, 0o700); err != nil {
			if !os.IsExist(err) {
				return "", err
			}
			continue
		}
		return filepath.Join(parent, name), nil
	}
	return "", fmt.Errorf("could not reserve a unique temporary directory")
}

func rootedWriteFile(rootPath, absolute string, body []byte, mode os.FileMode) error {
	relative, err := rootedPath(rootPath, absolute)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return err
	}
	defer root.Close()
	return storage.WriteFileAtomicInRoot(root, relative, body, mode)
}

func rootedRemoveAll(rootPath, absolute string) error {
	relative, err := rootedPath(rootPath, absolute)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return err
	}
	defer root.Close()
	return root.RemoveAll(relative)
}

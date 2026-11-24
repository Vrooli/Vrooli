package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/storage"
)

func overlayRelativePath(root *os.Root, absolute string) (string, error) {
	if root == nil {
		return "", fmt.Errorf("calibration overlay root is not open")
	}
	relative, err := filepath.Rel(root.Name(), absolute)
	if err != nil {
		return "", fmt.Errorf("resolve overlay path: %w", err)
	}
	if relative == ".." || filepath.IsAbs(relative) || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("overlay path escapes temporary root")
	}
	return filepath.Clean(relative), nil
}

func overlayMkdirAll(root *os.Root, absolute string, mode os.FileMode) error {
	path, err := overlayRelativePath(root, absolute)
	if err != nil {
		return err
	}
	return root.MkdirAll(path, mode)
}

func overlayWriteFile(root *os.Root, absolute string, data []byte, mode os.FileMode) error {
	path, err := overlayRelativePath(root, absolute)
	if err != nil {
		return err
	}
	return storage.WriteFileAtomicInRoot(root, path, data, mode)
}

func overlaySymlink(root *os.Root, target, absolute string) error {
	path, err := overlayRelativePath(root, absolute)
	if err != nil {
		return err
	}
	return root.Symlink(target, path)
}

func overlayChtimes(root *os.Root, absolute string, atime, mtime time.Time) error {
	path, err := overlayRelativePath(root, absolute)
	if err != nil {
		return err
	}
	return root.Chtimes(path, atime, mtime)
}

func overlayRemove(root *os.Root, absolute string) error {
	path, err := overlayRelativePath(root, absolute)
	if err != nil {
		return err
	}
	return root.Remove(path)
}

func overlayRemoveAll(root *os.Root, absolute string) error {
	path, err := overlayRelativePath(root, absolute)
	if err != nil {
		return err
	}
	return root.RemoveAll(path)
}

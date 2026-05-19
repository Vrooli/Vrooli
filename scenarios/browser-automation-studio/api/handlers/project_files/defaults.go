package project_files //nolint:revive

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// defaultFS is the production filesystem seam, backed by the os package.
type defaultFS struct{}

func (defaultFS) Stat(path string) (FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	return fsInfo{info}, nil
}

func (defaultFS) MkdirAll(path string, perm uint32) error {
	return os.MkdirAll(path, os.FileMode(perm))
}

func (defaultFS) Rename(oldPath, newPath string) error { return os.Rename(oldPath, newPath) }

func (defaultFS) RemoveAll(path string) error { return os.RemoveAll(path) }

func (defaultFS) ReadDir(path string) ([]DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	out := make([]DirEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, fsDirEntry{e})
	}
	return out, nil
}

type fsInfo struct{ os.FileInfo }

func (i fsInfo) IsDir() bool { return i.FileInfo.IsDir() }

type fsDirEntry struct{ os.DirEntry }

func (e fsDirEntry) Name() string { return e.DirEntry.Name() }
func (e fsDirEntry) IsDir() bool  { return e.DirEntry.IsDir() }

// defaultOS implements OSIntegration via exec.Command in a platform-aware
// way. Mirrors the implementations historically housed in handlers/exports.go.
type defaultOS struct{}

func (defaultOS) OpenFolder(folderPath string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", folderPath)
	case "windows":
		cmd = exec.Command("explorer", folderPath)
	default:
		cmd = exec.Command("xdg-open", folderPath)
	}
	return cmd.Run()
}

func (defaultOS) RevealInFileManager(filePath string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-R", filePath)
	case "windows":
		cmd = exec.Command("explorer", "/select,", filePath)
	default:
		// Linux file managers generally cannot select a specific file —
		// open the containing directory.
		cmd = exec.Command("xdg-open", filepath.Dir(filePath))
	}
	return cmd.Run()
}

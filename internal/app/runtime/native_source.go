package runtimeapp

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// ResolveNativeOperatorSource reads only the bounded maintained source pointer.
// The installed operator manager need not depend on its machine working directory
// or a source path compiled into a temporary build.
func ResolveNativeOperatorSource(home string) (string, error) {
	path := repocontract.SourceRootPointerPath(home)
	if path == "" {
		return "", errors.New("native operator source pointer requires a home")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() > 4096 {
		return "", errors.New("native source pointer must be a bounded regular file")
	}
	f, err := openNativeSourcePointer(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	actual, err := f.Stat()
	if err != nil {
		return "", err
	}
	if !actual.Mode().IsRegular() || !os.SameFile(info, actual) {
		return "", errors.New("native source pointer changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(f, 4097))
	if err != nil {
		return "", err
	}
	if len(data) > 4096 {
		return "", errors.New("native operator source pointer exceeds bound")
	}
	root := strings.TrimSpace(string(data))
	if root == "" || !filepath.IsAbs(root) || filepath.Clean(root) != root || strings.ContainsAny(root, "\x00\r\n") {
		return "", errors.New("native operator source pointer is not canonical")
	}
	return root, nil
}

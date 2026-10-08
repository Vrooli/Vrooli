// Package pydeps embeds the program-runtime kernel's deliberately empty Python
// dependency lock. The lock is materialized into scenario data at startup so
// the shared pyenv-go provisioner can enforce a reproducible environment.
package pydeps

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed requirements.lock
var lockBytes []byte

const LockName = "requirements.lock"

func LockBytes() []byte { return append([]byte(nil), lockBytes...) }

func Materialize(dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("pydeps: create dir %q: %w", dir, err)
	}
	dst := filepath.Join(dir, LockName)
	if existing, err := os.ReadFile(dst); err == nil && bytes.Equal(existing, lockBytes) {
		return dst, nil
	}
	if err := os.WriteFile(dst, lockBytes, 0o644); err != nil {
		return "", fmt.Errorf("pydeps: write lock %q: %w", dst, err)
	}
	return dst, nil
}

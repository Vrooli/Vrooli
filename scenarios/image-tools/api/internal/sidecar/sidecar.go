// Package sidecar ships the in-repo Python backend (image_tools_sidecar) that
// the AI engine shells out to for CPU-tractable ONNX ops. The Python sources are
// embedded in the Go binary and materialized to a cache directory at boot; that
// directory is placed on PYTHONPATH so `python3 -m image_tools_sidecar.<op>`
// resolves regardless of the process working directory.
//
// Only the Python *code* is shipped this way. The runtime packages it imports
// (onnxruntime, Pillow, numpy) are host provisioning — see Provisioning and
// docs/reference/backends.md. This is deliberate: weights + heavy native wheels
// are not vendored into the Go binary.
package sidecar

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// PackageName is the importable module name (and the embedded directory name).
const PackageName = "image_tools_sidecar"

//go:embed py/image_tools_sidecar/*.py
var pyFS embed.FS

// embedRoot is the path prefix inside pyFS the package lives under.
const embedRoot = "py"

// Materialize writes the embedded sidecar package under root/image_tools_sidecar
// and returns the directory that must be on PYTHONPATH (i.e. root). It is
// idempotent: existing files are overwritten so a binary upgrade refreshes the
// sidecar. Callers typically pass a per-data-dir cache path.
func Materialize(root string) (pythonPath string, err error) {
	if root == "" {
		return "", fmt.Errorf("sidecar: materialize root is empty")
	}
	err = fs.WalkDir(pyFS, embedRoot, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		// Strip the embed prefix so files land at <root>/image_tools_sidecar/...
		rel, relErr := filepath.Rel(embedRoot, p)
		if relErr != nil {
			return relErr
		}
		dest := filepath.Join(root, rel)
		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		data, readErr := pyFS.ReadFile(p)
		if readErr != nil {
			return readErr
		}
		if mkErr := os.MkdirAll(filepath.Dir(dest), 0o755); mkErr != nil {
			return mkErr
		}
		return os.WriteFile(dest, data, 0o644)
	})
	if err != nil {
		return "", fmt.Errorf("sidecar: materialize: %w", err)
	}
	return root, nil
}

// EnsureOnPath materializes the sidecar under root and prepends it to the
// process PYTHONPATH so child python3 invocations can import the package. Safe
// to call once at boot.
func EnsureOnPath(root string) (string, error) {
	path, err := Materialize(root)
	if err != nil {
		return "", err
	}
	existing := os.Getenv("PYTHONPATH")
	if existing == "" {
		if setErr := os.Setenv("PYTHONPATH", path); setErr != nil {
			return "", fmt.Errorf("sidecar: set PYTHONPATH: %w", setErr)
		}
	} else if !pathContains(existing, path) {
		if setErr := os.Setenv("PYTHONPATH", path+string(os.PathListSeparator)+existing); setErr != nil {
			return "", fmt.Errorf("sidecar: set PYTHONPATH: %w", setErr)
		}
	}
	return path, nil
}

func pathContains(list, want string) bool {
	for _, p := range strings.Split(list, string(os.PathListSeparator)) {
		if p == want {
			return true
		}
	}
	return false
}

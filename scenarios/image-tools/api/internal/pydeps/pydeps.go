// Package pydeps embeds the image-tools Python dependency manifests — the
// ranged, governed requirements.in and the fully pinned + hashed
// requirements.lock generated from it — so they travel with the compiled binary.
// At boot the lock is materialized into the scenario data dir, where the uv venv
// provisioner (internal/pyenv) syncs the private venv from it. requirements.lock
// is the single version source of truth; requirements.in carries the
// app-compatibility ceilings (e.g. transformers<5). Regenerate the lock with the
// `uv pip compile` command in internal/pydeps/README.md — never edit
// requirements.lock by hand.
package pydeps

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed requirements.lock
var lockBytes []byte

//go:embed requirements.in
var inBytes []byte

// LockName is the filename the lock is materialized under.
const LockName = "requirements.lock"

// LockBytes returns the embedded requirements.lock contents.
func LockBytes() []byte { return append([]byte(nil), lockBytes...) }

// InBytes returns the embedded requirements.in contents.
func InBytes() []byte { return append([]byte(nil), inBytes...) }

// LockHash is the hex sha-256 of the embedded lock — the same value pyenv records
// after a sync. Callers use it synchronously (without building the venv) as the
// smoke-cache invalidator and as half of the per-model verdict cache key.
func LockHash() string {
	sum := sha256.Sum256(lockBytes)
	return hex.EncodeToString(sum[:])
}

// Materialize writes the embedded lock into dir (created if absent) and returns
// the absolute lockfile path. It is idempotent and content-stable: the file is
// rewritten only when its bytes differ, so the pyenv lock-hash sentinel stays
// meaningful (an unchanged lock does not trigger a venv resync) across restarts.
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

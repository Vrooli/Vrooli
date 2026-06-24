// Package pyenv provisions and validates a private, lock-pinned Python
// virtualenv built by `uv`, invoked by absolute interpreter path.
//
// It is deliberately scenario-agnostic: it knows a venv directory, a lockfile,
// a base interpreter, and an injected exec Runner — nothing about image-tools,
// models, or backends. That keeps it a near-mechanical lift to a platform
// package (packages/pyenv-go) when a second consumer (video-tools) appears; see
// scenarios/image-tools/docs/internal/SEAMS.md for the extraction boundary.
//
// The contract Ensure enforces:
//   - build-if-absent: create the venv with `uv venv` when its interpreter is
//     missing;
//   - repair-if-lock-changed: `uv pip sync` the venv to the lockfile whenever the
//     lockfile's content hash differs from the hash recorded in the venv's
//     sentinel file (so a committed requirements.lock is the single version SSOT);
//   - validate: confirm the absolute interpreter exists after provisioning.
//
// Ensure never deletes or mutates the host user-site; the venv is additive and
// isolated. Import-level readiness (can the interpreter import torch/diffusers?)
// is intentionally NOT pyenv's job — that belongs to the caller's backend
// availability probe, which runs against the interpreter pyenv returns.
package pyenv

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// sentinelName records, inside the venv, the lockfile hash the venv was last
// synced from. A mismatch (or absence) triggers a resync.
const sentinelName = ".vrooli-pyenv-lock-hash"

// Spec declares where the venv lives and what it is pinned to. All fields are
// plain paths/config — no scenario types — so the provisioner is reusable.
type Spec struct {
	// VenvDir is the absolute directory the virtualenv is created in (e.g.
	// <scenario-data>/pyenv). Created if absent.
	VenvDir string
	// LockFile is the absolute path to the fully-pinned (+hashed) requirements
	// lock that is the version source of truth (`uv pip sync` target).
	LockFile string
	// BasePython is the interpreter uv seeds the venv from. Empty lets uv pick a
	// discovered interpreter (its default behaviour).
	BasePython string
	// UV is the uv command (name or absolute path). Empty defaults to "uv",
	// resolved on PATH (uv is provisioned as a platform host tool).
	UV string
}

// Interpreter is the provisioned result: the absolute interpreter path the
// caller invokes, and the lockfile hash it was synced from (for cache keys).
type Interpreter struct {
	// Python is the absolute path to the venv interpreter (<venv>/bin/python on
	// unix, <venv>\Scripts\python.exe on windows).
	Python string
	// LockHash is the hex sha-256 of the lockfile the venv is synced to.
	LockHash string
}

// Runner executes a command and returns its combined output. It is injected so
// tests assert argv assembly without a real uv/python on the host. A nil Runner
// uses the real os/exec implementation.
type Runner func(ctx context.Context, name string, args []string) ([]byte, error)

// InterpreterPath returns the absolute interpreter path for a venv directory,
// without provisioning. Exposed so callers can reference the interpreter (e.g.
// for a doctor message) before/without an Ensure.
func InterpreterPath(venvDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvDir, "Scripts", "python.exe")
	}
	return filepath.Join(venvDir, "bin", "python")
}

// Ensure provisions the venv to match the lockfile and returns the absolute
// interpreter. It is idempotent: an already-synced venv (interpreter present and
// sentinel hash matching the lockfile) is returned without invoking uv.
func Ensure(ctx context.Context, spec Spec, run Runner) (Interpreter, error) {
	if strings.TrimSpace(spec.VenvDir) == "" {
		return Interpreter{}, fmt.Errorf("pyenv: VenvDir is required")
	}
	if strings.TrimSpace(spec.LockFile) == "" {
		return Interpreter{}, fmt.Errorf("pyenv: LockFile is required")
	}
	if !filepath.IsAbs(spec.VenvDir) {
		return Interpreter{}, fmt.Errorf("pyenv: VenvDir must be absolute, got %q", spec.VenvDir)
	}
	lockBytes, err := os.ReadFile(spec.LockFile)
	if err != nil {
		return Interpreter{}, fmt.Errorf("pyenv: read lockfile %q: %w", spec.LockFile, err)
	}
	if len(strings.TrimSpace(string(lockBytes))) == 0 {
		return Interpreter{}, fmt.Errorf("pyenv: lockfile %q is empty", spec.LockFile)
	}
	lockHash := hashBytes(lockBytes)
	interp := InterpreterPath(spec.VenvDir)

	if upToDate(interp, sentinelPath(spec.VenvDir), lockHash) {
		return Interpreter{Python: interp, LockHash: lockHash}, nil
	}

	if run == nil {
		run = defaultRunner
	}
	uv := strings.TrimSpace(spec.UV)
	if uv == "" {
		uv = "uv"
	}

	// build-if-absent: create the venv only when its interpreter is missing, so a
	// resync (lock changed) reuses the existing environment.
	if !fileExists(interp) {
		args := []string{"venv", spec.VenvDir}
		if strings.TrimSpace(spec.BasePython) != "" {
			args = append(args, "--python", spec.BasePython)
		}
		if out, err := run(ctx, uv, args); err != nil {
			return Interpreter{}, fmt.Errorf("pyenv: uv venv failed: %w: %s", err, strings.TrimSpace(string(out)))
		}
	}

	// repair/sync: install exactly the locked set into the venv interpreter.
	if out, err := run(ctx, uv, []string{"pip", "sync", "--python", interp, spec.LockFile}); err != nil {
		return Interpreter{}, fmt.Errorf("pyenv: uv pip sync failed: %w: %s", err, strings.TrimSpace(string(out)))
	}

	// validate: the interpreter must exist now (a successful sync implies the venv
	// is intact); fail loud rather than hand back a path that is not there.
	if !fileExists(interp) {
		return Interpreter{}, fmt.Errorf("pyenv: interpreter %q absent after provisioning", interp)
	}

	if err := os.WriteFile(sentinelPath(spec.VenvDir), []byte(lockHash), 0o644); err != nil {
		return Interpreter{}, fmt.Errorf("pyenv: write lock sentinel: %w", err)
	}
	return Interpreter{Python: interp, LockHash: lockHash}, nil
}

// upToDate reports whether the venv interpreter exists and its recorded sentinel
// hash matches the current lock hash (no uv work needed).
func upToDate(interp, sentinel, lockHash string) bool {
	if !fileExists(interp) {
		return false
	}
	recorded, err := os.ReadFile(sentinel)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(recorded)) == lockHash
}

func sentinelPath(venvDir string) string { return filepath.Join(venvDir, sentinelName) }

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func defaultRunner(ctx context.Context, name string, args []string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// Package pyenv provisions and validates a private, lock-pinned Python
// virtualenv built by uv and invoked by absolute interpreter path.
//
// The package is intentionally scenario-agnostic. Consumers own their Python
// dependency manifests; this package owns only the reproducible environment
// lifecycle and accepts a lock file path as input.
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

const sentinelName = ".vrooli-pyenv-lock-hash"

// Spec declares where the venv lives and which lock file is authoritative.
type Spec struct {
	VenvDir    string
	LockFile   string
	BasePython string
	UV         string
}

// Interpreter is the provisioned absolute interpreter and its lock hash.
type Interpreter struct {
	Python   string
	LockHash string
}

// Runner executes a command and returns combined output. It is injectable for
// deterministic tests; nil uses os/exec.
type Runner func(ctx context.Context, name string, args []string) ([]byte, error)

// InterpreterPath returns the platform-specific interpreter path in venvDir.
func InterpreterPath(venvDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvDir, "Scripts", "python.exe")
	}
	return filepath.Join(venvDir, "bin", "python")
}

// Ensure creates or repairs the venv so it matches the lock file.
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
	if strings.TrimSpace(spec.BasePython) != "" {
		if out, err := run(ctx, uv, []string{"python", "install", spec.BasePython}); err != nil {
			return Interpreter{}, fmt.Errorf("pyenv: uv python install %s failed: %w: %s", spec.BasePython, err, strings.TrimSpace(string(out)))
		}
	}
	if !fileExists(interp) {
		args := []string{"venv", spec.VenvDir}
		if strings.TrimSpace(spec.BasePython) != "" {
			args = append(args, "--python", spec.BasePython)
		}
		if out, err := run(ctx, uv, args); err != nil {
			return Interpreter{}, fmt.Errorf("pyenv: uv venv failed: %w: %s", err, strings.TrimSpace(string(out)))
		}
	}
	if out, err := run(ctx, uv, []string{"pip", "sync", "--python", interp, spec.LockFile}); err != nil {
		return Interpreter{}, fmt.Errorf("pyenv: uv pip sync failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	if !fileExists(interp) {
		return Interpreter{}, fmt.Errorf("pyenv: interpreter %q absent after provisioning", interp)
	}
	if err := os.WriteFile(sentinelPath(spec.VenvDir), []byte(lockHash), 0o644); err != nil {
		return Interpreter{}, fmt.Errorf("pyenv: write lock sentinel: %w", err)
	}
	return Interpreter{Python: interp, LockHash: lockHash}, nil
}

func upToDate(interp, sentinel, lockHash string) bool {
	if !fileExists(interp) {
		return false
	}
	recorded, err := os.ReadFile(sentinel)
	return err == nil && strings.TrimSpace(string(recorded)) == lockHash
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

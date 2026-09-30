package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// installExecutable replaces target with this executable only after the full
// image has been copied and flushed. The platform-specific replacement keeps
// the lifecycle build step free of shell-specific sync/move semantics.
func installExecutable(target string) error {
	source, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve current executable: %w", err)
	}
	source, err = filepath.EvalSymlinks(source)
	if err != nil {
		return fmt.Errorf("resolve executable symlink: %w", err)
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolve install target: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(target), ".vrooli-autoheal-loop-*")
	if err != nil {
		return fmt.Errorf("create install temporary: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}
	defer cleanup()

	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open current executable: %w", err)
	}
	_, copyErr := io.Copy(tmp, input)
	closeInputErr := input.Close()
	if copyErr != nil {
		return fmt.Errorf("copy current executable: %w", copyErr)
	}
	if closeInputErr != nil {
		return fmt.Errorf("close current executable: %w", closeInputErr)
	}
	if err := tmp.Chmod(0o755); err != nil {
		return fmt.Errorf("set executable mode: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("flush installed executable: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close installed executable: %w", err)
	}
	if err := atomicReplace(tmpPath, target); err != nil {
		return fmt.Errorf("replace executable: %w", err)
	}
	return nil
}

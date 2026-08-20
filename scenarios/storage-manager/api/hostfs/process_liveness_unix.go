//go:build !windows

package hostfs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type processLiveness struct{}

func NewProcessLiveness() *processLiveness { return &processLiveness{} }

func (p *processLiveness) IsRunning(ctx context.Context, path string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	target, err := filepath.EvalSymlinks(filepath.Clean(path))
	if err != nil {
		// A missing executable cannot be running under this exact path.
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false, fmt.Errorf("read process table: %w", err)
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if _, err := strconv.ParseUint(entry.Name(), 10, 64); err != nil {
			continue
		}
		exe, err := os.Readlink(filepath.Join("/proc", entry.Name(), "exe"))
		if err != nil {
			continue
		}
		exe = strings.TrimSuffix(exe, " (deleted)")
		resolved, err := filepath.EvalSymlinks(exe)
		if err != nil {
			resolved = filepath.Clean(exe)
		}
		if resolved == target {
			return true, nil
		}
	}
	return false, nil
}

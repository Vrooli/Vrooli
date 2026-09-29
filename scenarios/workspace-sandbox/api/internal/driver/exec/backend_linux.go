//go:build linux

package exec

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"

	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/types"
)

// platformContainmentBackend returns the containment backend for this OS.
// On Linux that is bubblewrap.
func platformContainmentBackend() containmentBackend {
	return bwrapBackend{}
}

// bwrapBackend contains processes with bubblewrap: user/ipc/uts/cgroup
// (and optionally pid/net) namespace unsharing plus bind-mounted paths.
// Resource limits are applied by prepending prlimit (see BuildExecCommand).
type bwrapBackend struct{}

func (bwrapBackend) id() string { return "bwrap" }

func (bwrapBackend) available(starter process.Starter) error {
	if _, err := starter.LookPath("bwrap"); err != nil {
		return fmt.Errorf("bubblewrap (bwrap) not found: %w. Install with: apt-get install bubblewrap", err)
	}
	return nil
}

func (bwrapBackend) buildStartOpts(starter process.Starter, s *types.Sandbox, cfg BwrapConfig, cmd string, args ...string) (process.StartOpts, error) {
	for _, file := range cfg.PolicyFiles {
		info, err := os.Lstat(file.Source)
		if err != nil {
			return process.StartOpts{}, err
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Nlink != 1 {
			return process.StartOpts{}, fmt.Errorf("policy file has a hardlink or unverifiable inode: %q", file.Source)
		}
	}
	if policy := s.Behavior.WritePolicy; policy != nil {
		// A pre-existing hardlink in a writable subtree can mutate an inode
		// also named by a read-only control path. Reject such grants. WalkDir
		// does not follow symlinks; normal symlink resolution remains subject
		// to the read-only mount boundaries. Only granted trees are scanned.
		for _, path := range policy.Paths {
			if err := filepath.WalkDir(filepath.Join(s.MergedDir, path), func(path string, entry fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !entry.Type().IsRegular() {
					return nil
				}
				info, err := entry.Info()
				if err != nil {
					return err
				}
				stat, ok := info.Sys().(*syscall.Stat_t)
				if !ok || stat.Nlink != 1 {
					return fmt.Errorf("write grant contains a hardlink or unverifiable inode: %q", path)
				}
				return nil
			}); err != nil {
				return process.StartOpts{}, err
			}
		}
	}
	executable, execArgs := BuildExecCommand(s, cfg, cmd, args...)
	execPath, err := starter.LookPath(executable)
	if err != nil {
		if executable == "prlimit" {
			return process.StartOpts{}, fmt.Errorf("prlimit not found: %w. Resource limits require prlimit (part of util-linux)", err)
		}
		return process.StartOpts{}, fmt.Errorf("bubblewrap (bwrap) not found: %w", err)
	}
	return process.StartOpts{
		Path: execPath,
		Args: append([]string(nil), execArgs...),
	}, nil
}

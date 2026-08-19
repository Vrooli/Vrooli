package main

// copyCodexHome lives in package main because it depends on the per-session
// CODEX_HOME layout (sessionCodexHome in pty.go). It is injected into the
// sessions handler's Adapter via the CopyCodexHome field.

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"web-console/internal/sessionstore"
)

// archivedAgentHistoryPresent predicts whether the agent-specific resume
// command still has durable state to consume. It never mutates that state.
func archivedAgentHistoryPresent(meta sessionstore.Metadata) bool {
	path, recursive := archivedAgentHistoryPath(meta)
	info, err := os.Stat(path)
	return err == nil && (!recursive || info.IsDir())
}

// archivedAgentHistoryPath resolves retention paths without calling the
// session-home constructors: those constructors create missing directories,
// which would make a successfully pruned archive appear reopenable again.
func archivedAgentHistoryPath(meta sessionstore.Metadata) (string, bool) {
	if meta.ID == "" {
		return "", false
	}
	root := resolveSessionStateRoot()
	switch meta.AgentType {
	case sessionstore.AgentCodex:
		return filepath.Join(root, "codex", meta.ID), true
	case sessionstore.AgentGrok:
		return filepath.Join(root, "grok", meta.ID), true
	default:
		// Claude/OpenCode history is not stored in a per-session home. The
		// recorded rollout is the only path known to belong to this session;
		// never remove its shared project directory.
		return filepath.Clean(meta.LastRolloutPath), false
	}
}

func archivedAgentHistorySize(meta sessionstore.Metadata) (int64, error) {
	path, _ := archivedAgentHistoryPath(meta)
	if path == "" || path == "." {
		return 0, nil
	}
	var total int64
	err := filepath.WalkDir(path, func(_ string, entry fs.DirEntry, err error) error {
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			total += info.Size()
		}
		return nil
	})
	if os.IsNotExist(err) {
		return 0, nil
	}
	return total, err
}

func pruneArchivedAgentHistory(meta sessionstore.Metadata) (int64, error) {
	path, recursive := archivedAgentHistoryPath(meta)
	if path == "" || path == "." {
		return 0, nil
	}
	size, err := archivedAgentHistorySize(meta)
	if err != nil {
		return 0, err
	}
	if recursive {
		kind := string(meta.AgentType)
		expectedParent := filepath.Join(resolveSessionStateRoot(), kind)
		if filepath.Dir(path) != expectedParent || filepath.Base(path) != meta.ID || strings.ContainsAny(meta.ID, `/\\`) {
			return 0, fmt.Errorf("refusing unsafe agent-home path %q", path)
		}
		if err := os.RemoveAll(path); err != nil {
			return 0, err
		}
		return size, nil
	}
	if !filepath.IsAbs(path) {
		return 0, fmt.Errorf("refusing non-absolute agent-history path %q", path)
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if info.IsDir() {
		return 0, fmt.Errorf("refusing to remove shared agent-history directory %q", path)
	}
	if err := os.Remove(path); err != nil {
		return 0, err
	}
	return size, nil
}

// copyCodexHome rsyncs (or copies via tar fallback) the per-session
// CODEX_HOME from the orphan to the fresh pane. Bounded: codex homes
// contain symlinks to global config + per-session rollouts, typically
// < 50MB.
func copyCodexHome(oldID, newID string) error {
	src := sessionCodexHome(oldID)
	dst := sessionCodexHome(newID)
	if _, err := os.Stat(src); err != nil {
		return nil
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return fmt.Errorf("mkdirall %s: %w", dst, err)
	}
	if path, err := exec.LookPath("rsync"); err == nil {
		out, err := exec.Command(path, "-a", "--", src+"/", dst+"/").CombinedOutput()
		if err != nil {
			return fmt.Errorf("rsync %s -> %s: %v: %s", src, dst, err, string(out))
		}
		return nil
	}
	if path, err := exec.LookPath("cp"); err == nil {
		out, err := exec.Command(path, "-a", filepath.Join(src, "."), dst).CombinedOutput()
		if err != nil {
			return fmt.Errorf("cp -a %s -> %s: %v: %s", src, dst, err, string(out))
		}
		return nil
	}
	return fmt.Errorf("neither rsync nor cp available")
}

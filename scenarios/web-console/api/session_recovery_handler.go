package main

// copyCodexHome lives in package main because it depends on the per-session
// CODEX_HOME layout (sessionCodexHome in pty.go). It is injected into the
// sessions handler's Adapter via the CopyCodexHome field; the recovery copy
// itself is limited to the rollout subtree.

import (
	"fmt"
	"io"
	"io/fs"
	"os"
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

// copyCodexHome copies only the session-owned rollout tree from the orphan to
// the fresh pane. Shared configuration and regenerable runtime state are
// linked by the launcher, so recovery never traverses or duplicates them.
func copyCodexHome(oldID, newID string) error {
	src := sessionCodexSessionsDir(oldID)
	dst := sessionCodexSessionsDir(newID)
	if _, err := os.Stat(src); err != nil {
		return nil
	}
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing symlink in rollout tree %q", path)
		}
		return copyRolloutFile(path, target, entry)
	})
}

func copyRolloutFile(src, dst string, entry fs.DirEntry) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	mode := entry.Type().Perm()
	if mode == 0 {
		mode = 0o600
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

package main

// copyCodexHome lives in package main because it copies one per-session Codex
// home, whose layout codex.SessionHomePath and sessionHomeRegenerableDirs own;
// the function is injected into the sessions handler's Adapter via
// CopyCodexHome.

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	coreRetention "github.com/vrooli/api-core/retention"
	"web-console/backends/codex"
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
		if err := deleteArchivedPath(path, expectedParent, size); err != nil {
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
	if err := deleteArchivedPath(path, filepath.Dir(path), size); err != nil {
		return 0, err
	}
	return size, nil
}

func deleteArchivedPath(path, root string, size int64) error {
	pruner, err := coreRetention.NewDirectoryPruner(coreRetention.DirectoryConfig{Path: root})
	if err != nil {
		return err
	}
	_, err = pruner.Delete(context.Background(), coreRetention.Candidates{{Path: path, Bytes: size}}, coreRetention.Batch{MaxItems: 1})
	return err
}

// copyCodexHome copies one session's durable Codex record into the reopened
// session's home: the rollout tree plus the goal, thread, state, memory and
// queue stores, history and attachments. Without the goal store a reopened
// lead came back as a bare transcript and its goal was gone. What is left
// behind is runtime the launcher re-links or Codex re-creates: symlinks to the
// shared home, sessionHomeRegenerableDirs (daemon installs, sockets, caches),
// trace-log SQLite, SQLite shared-memory indexes and lock files. The source
// session has no running Codex (it is archived or orphaned), so copying each
// SQLite database with its write-ahead log is a consistent copy.
func copyCodexHome(oldID, newID string) error {
	root := resolveSessionStateRoot()
	src := codex.SessionHomePath(root, oldID)
	dst := codex.SessionHomePath(root, newID)
	if _, err := os.Stat(src); err != nil {
		return nil
	}
	regenerable := sessionHomeRegenerableDirs["codex"]
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if rel == "." {
			return os.MkdirAll(target, 0o755)
		}
		top, _, nested := strings.Cut(rel, string(filepath.Separator))
		if entry.Type()&os.ModeSymlink != 0 {
			if top == "sessions" {
				return fmt.Errorf("refusing symlink in rollout tree %q", path)
			}
			return nil
		}
		if regenerable[top] {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		name := entry.Name()
		if !nested && (codexTraceLogSQLite.MatchString(name) || strings.HasSuffix(name, "-shm") || strings.HasSuffix(name, ".lock")) {
			return nil
		}
		return copyRolloutFile(path, target, entry)
	})
}

// codexRootThread resolves a session's recorded Codex thread to the root
// thread of its conversation, read from the rollouts in the session's home.
func codexRootThread(sessionID, threadID string) string {
	return codex.RootThreadID(codex.SessionHomePath(resolveSessionStateRoot(), sessionID), threadID)
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

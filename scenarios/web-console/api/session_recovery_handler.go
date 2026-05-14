package main

// session_recovery_handler.go holds the recovery-flow primitives used by
// sessions_adapter.Recover. The HTTP handler surface itself lives in the
// handlers/sessions Connect-RPC package; this file now exposes only the
// helpers that decide whether a session is recoverable and how to resume it.
//
// See docs/internal/plans/persistent-session-recovery-hardening-plan.md §4
// for the design behind the recover/dismiss flow.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"web-console/internal/sessionstore"
)

func recoverabilityOf(m sessionstore.Metadata) (bool, string) {
	switch m.AgentType {
	case sessionstore.AgentNone:
		return false, "no agent identity recorded"
	case sessionstore.AgentClaude:
		if m.AgentSessionID == "" {
			return false, "claude session id is required (resuming the wrong project is unsafe)"
		}
		return true, ""
	case sessionstore.AgentCodex:
		// Codex can resume by id OR fall back to --last given a copied home.
		return true, ""
	default:
		return false, "unknown agent type: " + string(m.AgentType)
	}
}

// buildResumeCommand returns the literal string to paste into the new pane's
// stdin. Includes a trailing newline so it executes immediately. Never
// returns the empty string.
func buildResumeCommand(m sessionstore.Metadata) string {
	switch m.AgentType {
	case sessionstore.AgentCodex:
		if m.AgentSessionID != "" {
			return "codex --yolo resume " + m.AgentSessionID + "\n"
		}
		return "codex --yolo resume --last\n"
	case sessionstore.AgentClaude:
		// Caller has already checked AgentSessionID != "".
		return "claude --resume " + m.AgentSessionID + " --dangerously-skip-permissions\n"
	}
	return "echo 'no agent identity recorded; nothing to resume'\n"
}

// copyCodexHome rsyncs (or copies via tar fallback) the per-session
// CODEX_HOME from the orphan to the fresh pane. Bounded: codex homes
// contain symlinks to global config + per-session rollouts, typically
// < 50MB.
func copyCodexHome(oldID, newID string) error {
	src := sessionCodexHome(oldID)
	dst := sessionCodexHome(newID)
	if _, err := os.Stat(src); err != nil {
		// Nothing to copy — the codex tailer never saw a rollout. Recovery
		// can still proceed (`codex --yolo resume <id>` will go fetch from
		// the upstream rollout path if codex knows the global location).
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

package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"web-console/backends/codex"
	"web-console/internal/sessionstore"
	"web-console/session"
)

// DOC: docs/concepts/GLOSSARY.md#core-terms

// sessionIdleJudge decides the idle expiry policy for one live session. The
// session is archived only when every condition holds, each read from the
// session's own evidence:
//
//   - the activity detector shows the agent at its prompt or asking something
//     (an in-chat prompt counts as idle and is recorded in the note);
//   - the agent's work record (every rollout of the Codex home, sub-agent
//     threads included; Claude's transcript and its subagents folder) has not
//     been written for the whole window, nor has the session been opened
//     within it;
//   - for Codex, no goal in the home is active, and a managed native owner
//     reports no active thread or running turn.
//
// The activity detector's own "since" cannot measure the window: a Codex TUI
// redraws about once a minute, which restarts it. Anything missing or
// unreadable keeps the session.
//
// Memory is recorded in the note, not used in the rule: a finished lead holds
// anywhere from 0.5 GB to 2.6 GB depending on how long it ran (measured
// 2026-11-21), and resident size alone hides the swapped-out bulk.
type sessionIdleJudge struct {
	now         func() time.Time
	metadata    func(ctx context.Context, id string) (sessionstore.Metadata, error)
	stateRoot   func() string
	nativeState func(ctx context.Context, meta sessionstore.Metadata) ([]codex.NativeThreadState, error)
	memory      func(sessionID string) (int64, bool)
}

func (s *Server) newSessionIdleJudge() *sessionIdleJudge {
	return &sessionIdleJudge{
		now: time.Now,
		metadata: func(ctx context.Context, id string) (sessionstore.Metadata, error) {
			if s.sessionStore == nil {
				return sessionstore.Metadata{}, errors.New("session store not configured")
			}
			return s.sessionStore.Get(ctx, id)
		},
		stateRoot: resolveSessionStateRoot,
		nativeState: func(ctx context.Context, meta sessionstore.Metadata) ([]codex.NativeThreadState, error) {
			if s.managedCodex == nil {
				return nil, errors.New("native owner registry not configured")
			}
			return s.managedCodex.ReadThreadStates(ctx, meta, []string{meta.NativeThreadID})
		},
		memory: sessionAgentMemory,
	}
}

func idleKeep(format string, args ...any) session.IdleVerdict {
	return session.IdleVerdict{Note: fmt.Sprintf(format, args...)}
}

func (j *sessionIdleJudge) judge(ctx context.Context, sess *session.Session, idleAfter time.Duration) session.IdleVerdict {
	detector := sess.ActivityDetector()
	if detector == nil {
		return idleKeep("no activity detector")
	}
	activity := detector.Current()
	if activity.State != session.ActivityIdle && activity.State != session.ActivityWaiting {
		return idleKeep("agent is %s", activity.State)
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	meta, err := j.metadata(ctx, sess.ID)
	if err != nil {
		return idleKeep("session metadata unreadable: %v", err)
	}
	if !meta.ArchivedAt.IsZero() {
		return idleKeep("already archived")
	}
	lastWork, err := agentLastWork(meta, j.stateRoot())
	if err != nil {
		return idleKeep("%v", err)
	}
	if sess.CreatedAt.After(lastWork) {
		lastWork = sess.CreatedAt
	}
	now := j.now()
	idleFor := now.Sub(lastWork)
	if idleFor < idleAfter {
		return idleKeep("agent worked %s ago", idleFor.Round(time.Minute))
	}

	goalLine := ""
	if meta.AgentType == sessionstore.AgentCodex {
		home := codex.SessionHomePath(j.stateRoot(), meta.ID)
		goals, err := codex.ReadHomeGoals(ctx, home)
		if err != nil {
			return idleKeep("goal store unreadable: %v", err)
		}
		for _, goal := range goals {
			if goal.Status == "active" {
				return idleKeep("goal active on thread %s", goal.ThreadID)
			}
		}
		if meta.LaunchMode == sessionstore.LaunchModeCodexAppServer && meta.NativeThreadID != "" {
			states, err := j.nativeState(ctx, meta)
			if err != nil {
				return idleKeep("native owner unreadable: %v", err)
			}
			for _, state := range states {
				switch {
				case state.Status == "unavailable":
					return idleKeep("native thread %s state unavailable", state.ThreadID)
				case state.Status == "active" || state.ActiveTurnID != "":
					return idleKeep("native thread %s is running a turn", state.ThreadID)
				case state.Goal != nil && state.Goal.Status == "active":
					return idleKeep("native goal active on thread %s", state.ThreadID)
				}
			}
		}
		goalLine = describeGoals(goals)
	}

	var note strings.Builder
	fmt.Fprintf(&note, "Archived by the idle policy (%s): no agent work since %s (%s idle)", idleAfter, lastWork.UTC().Format("2006-01-02 15:04 UTC"), idleFor.Round(time.Minute))
	if goalLine != "" {
		fmt.Fprintf(&note, "; %s", goalLine)
	}
	note.WriteString("; no turn or sub-agent running.")
	if bytes, ok := j.memory(sess.ID); ok {
		fmt.Fprintf(&note, " Memory freed: %.1f GB (agent processes, resident plus swap).", float64(bytes)/(1<<30))
	}
	if activity.State == session.ActivityWaiting {
		note.WriteString(" " + describePrompt(activity, sess.PlainText(false)))
	}
	fmt.Fprintf(&note, " Reopen brings back the thread and its goal: Reopen in the archive, or `web-console session reopen %s`.", sess.ID)
	return session.IdleVerdict{Archive: true, Note: note.String()}
}

// agentLastWork is the newest write to the files the session's agent writes
// as it works. An agent without a readable work record is an error.
func agentLastWork(meta sessionstore.Metadata, stateRoot string) (time.Time, error) {
	switch meta.AgentType {
	case sessionstore.AgentCodex:
		newest, ok, err := codex.NewestRolloutWrite(codex.SessionHomePath(stateRoot, meta.ID))
		if err != nil {
			return time.Time{}, fmt.Errorf("codex rollouts unreadable: %w", err)
		}
		if !ok {
			return time.Time{}, errors.New("no codex rollout tree for this session")
		}
		return newest, nil
	case sessionstore.AgentClaude:
		if meta.LastRolloutPath == "" {
			return time.Time{}, errors.New("no claude transcript recorded for this session")
		}
		info, err := os.Stat(meta.LastRolloutPath)
		if err != nil {
			return time.Time{}, fmt.Errorf("claude transcript unreadable: %w", err)
		}
		newest := info.ModTime()
		// Claude writes each sub-agent's transcript beside the session's.
		sub := strings.TrimSuffix(meta.LastRolloutPath, filepath.Ext(meta.LastRolloutPath))
		err = filepath.WalkDir(sub, func(_ string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type().IsRegular() {
				if fileInfo, infoErr := entry.Info(); infoErr == nil && fileInfo.ModTime().After(newest) {
					newest = fileInfo.ModTime()
				}
			}
			return nil
		})
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return time.Time{}, fmt.Errorf("claude sub-agent transcripts unreadable: %w", err)
		}
		return newest, nil
	default:
		return time.Time{}, fmt.Errorf("no work record is read for agent %q", meta.AgentType)
	}
}

func describeGoals(goals []codex.HomeGoal) string {
	if len(goals) == 0 {
		return "no goal set"
	}
	parts := make([]string, 0, len(goals))
	for _, goal := range goals {
		parts = append(parts, fmt.Sprintf("goal %s on thread %s", goal.Status, goal.ThreadID))
	}
	return strings.Join(parts, ", ")
}

// describePrompt records what the agent was asking so nothing is lost when
// the session closes. A prompt the detector could not parse (kind "unknown",
// no text) is recorded as the bottom of the screen, where the question sits.
func describePrompt(activity session.Activity, screen string) string {
	kind, text := "prompt", ""
	var options []string
	if p := activity.Prompt; p != nil {
		if p.Kind != "" && p.Kind != "unknown" {
			kind = strings.ReplaceAll(p.Kind, "_", " ")
		}
		text = strings.Join(strings.Fields(p.Text), " ")
		for _, option := range p.Options {
			options = append(options, option.Key+" "+option.Label)
		}
	}
	if text == "" && len(options) == 0 {
		if tail := screenTail(screen, 8); tail != "" {
			return fmt.Sprintf("It was waiting on a %s; the screen read: %q.", kind, clip(tail, 400))
		}
		return fmt.Sprintf("It was waiting on a %s whose text could not be read.", kind)
	}
	out := fmt.Sprintf("It was waiting on a %s", kind)
	if text != "" {
		out += fmt.Sprintf(": %q", clip(text, 400))
	}
	if len(options) > 0 {
		out += " (options: " + strings.Join(options, "; ") + ")"
	}
	return out + "."
}

// screenTail is the last n non-blank screen lines, whitespace collapsed.
func screenTail(screen string, n int) string {
	var lines []string
	for _, line := range strings.Split(screen, "\n") {
		if line = strings.Join(strings.Fields(line), " "); line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, " / ")
}

func clip(text string, max int) string {
	if r := []rune(text); len(r) > max {
		return string(r[:max]) + "…"
	}
	return text
}

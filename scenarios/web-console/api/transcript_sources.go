package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"web-console/backends/codex"
	"web-console/backends/grok"
	"web-console/internal/sessionstore"
	"web-console/internal/tailer"
)

type transcriptCheckpointAdapter struct {
	store AgentTranscriptCheckpointStore
}

func (a transcriptCheckpointAdapter) Get(ctx context.Context, source, key string) (tailer.Checkpoint, bool, error) {
	if a.store == nil {
		return tailer.Checkpoint{}, false, nil
	}
	cp, ok, err := a.store.Get(ctx, source, key)
	if err != nil || !ok {
		return tailer.Checkpoint{}, ok, err
	}
	return tailer.Checkpoint{
		Source: cp.Source, SourceKey: cp.SourceKey, SessionID: cp.SessionID,
		Cursor: cp.Cursor, UpdatedAt: cp.UpdatedAt,
	}, true, nil
}

func (a transcriptCheckpointAdapter) Save(ctx context.Context, cp tailer.Checkpoint) error {
	if a.store == nil {
		return nil
	}
	return a.store.Save(ctx, AgentTranscriptCheckpoint{
		Source: cp.Source, SourceKey: cp.SourceKey, SessionID: cp.SessionID,
		Cursor: cp.Cursor, UpdatedAt: cp.UpdatedAt,
	})
}

type claudeTranscriptSource struct{ tailer *ClaudeTailer }

func (s claudeTranscriptSource) DiscoverFiles(ctx context.Context) ([]tailer.FileRef, error) {
	if s.tailer == nil || s.tailer.server == nil || s.tailer.server.sessionStore == nil {
		return nil, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	rows, err := s.tailer.server.sessionStore.List(ctx)
	if err != nil {
		return nil, err
	}
	files := make([]tailer.FileRef, 0, len(rows))
	for _, row := range rows {
		if row.Status != sessionstore.StatusLive || row.AgentType != sessionstore.AgentClaude || row.AgentSessionID == "" || row.CWD == "" {
			continue
		}
		path := claudeTranscriptPath(home, row.CWD, row.AgentSessionID)
		if _, statErr := os.Stat(path); statErr != nil {
			continue
		}
		files = append(files, tailer.FileRef{Path: path, SessionID: row.ID})
	}
	return files, nil
}

func (claudeTranscriptSource) DecodeLine(_ string, line []byte) ([]tailer.Event, error) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return nil, nil
	}
	var entry claudeTranscriptEntry
	if err := json.Unmarshal(line, &entry); err != nil {
		return nil, nil
	}
	blocks, ok := claudeTextBlocks(entry.Message.Content)
	if !ok {
		return nil, nil
	}
	if entry.Type == "user" {
		for _, block := range blocks {
			if block.Type == "tool_result" {
				return nil, nil
			}
		}
	}
	events := make([]tailer.Event, 0, len(blocks))
	for _, block := range blocks {
		if block.Type != "text" || strings.TrimSpace(block.Text) == "" {
			continue
		}
		if entry.Type == "user" {
			events = append(events, tailer.Event{Role: "user", Text: block.Text, Commit: true})
		} else if entry.Type == "assistant" {
			events = append(events, tailer.Event{Role: "assistant", Text: block.Text, Commit: true})
		}
	}
	return events, nil
}

func (claudeTranscriptSource) CaptureAgentInfo(string, string) {}

type codexTranscriptSource struct{ tailer *CodexTailer }

func (s codexTranscriptSource) DiscoverFiles(context.Context) ([]tailer.FileRef, error) {
	if s.tailer == nil || s.tailer.server == nil || s.tailer.server.sessions == nil {
		return nil, nil
	}
	now := time.Now()
	files := make([]tailer.FileRef, 0)
	for _, sess := range s.tailer.server.sessions.List() {
		baseDir := sessionCodexSessionsDir(sess.ID)
		for day := 0; day >= -1; day-- {
			date := now.AddDate(0, 0, day)
			dir := filepath.Join(baseDir, date.Format("2006"), date.Format("01"), date.Format("02"))
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, entry := range entries {
				if entry.IsDir() || !strings.HasPrefix(entry.Name(), "rollout-") || !strings.HasSuffix(entry.Name(), ".jsonl") {
					continue
				}
				files = append(files, tailer.FileRef{Path: filepath.Join(dir, entry.Name()), SessionID: sess.ID})
			}
		}
	}
	return files, nil
}

func (s codexTranscriptSource) DecodeLine(_ string, line []byte) ([]tailer.Event, error) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return nil, nil
	}
	if text := codex.ExtractAssistantText(line); text != "" {
		return []tailer.Event{{Role: "assistant", Text: text, Commit: true}}, nil
	}
	if text := codex.ExtractUserText(line); text != "" {
		return []tailer.Event{{Role: "user", Text: text, Commit: true}}, nil
	}
	return nil, nil
}

func (s codexTranscriptSource) CaptureAgentInfo(path, sessionID string) {
	if s.tailer != nil {
		s.tailer.captureAgentInfo(path, sessionID)
	}
}

type grokTranscriptSource struct {
	tailer *GrokTailer
	mu     sync.Mutex
	turns  map[string]grok.TurnAccumulator
}

func (s *grokTranscriptSource) DiscoverFiles(context.Context) ([]tailer.FileRef, error) {
	if s == nil || s.tailer == nil || s.tailer.server == nil || s.tailer.server.sessions == nil {
		return nil, nil
	}
	files := make([]tailer.FileRef, 0)
	for _, sess := range s.tailer.server.sessions.List() {
		matches, err := filepath.Glob(filepath.Join(sessionGrokSessionsDir(sess.ID), "*", "*", "updates.jsonl"))
		if err != nil {
			continue
		}
		for _, path := range matches {
			files = append(files, tailer.FileRef{Path: path, SessionID: sess.ID})
		}
	}
	return files, nil
}

func (s *grokTranscriptSource) DecodeLine(path string, line []byte) ([]tailer.Event, error) {
	record, ok := grok.ParseUpdateLine(line)
	if !ok {
		return nil, nil
	}
	s.mu.Lock()
	if s.turns == nil {
		s.turns = make(map[string]grok.TurnAccumulator)
	}
	acc := s.turns[path]
	turn, done := acc.Add(record)
	s.turns[path] = acc
	s.mu.Unlock()
	if !done {
		return nil, nil
	}
	events := make([]tailer.Event, 0, 2)
	if turn.User != "" {
		events = append(events, tailer.Event{Role: "user", Text: turn.User, Commit: true})
	}
	if turn.Assistant != "" {
		events = append(events, tailer.Event{Role: "assistant", Text: turn.Assistant, Commit: true})
	}
	if len(events) == 0 {
		return []tailer.Event{{Commit: true}}, nil
	}
	return events, nil
}

func (s *grokTranscriptSource) CaptureAgentInfo(path, sessionID string) {
	if s != nil && s.tailer != nil {
		s.tailer.captureAgentInfo(path, sessionID)
	}
}

func newClaudeTranscriptEngine(ct *ClaudeTailer) *tailer.Engine {
	return tailer.New(tailer.Config{
		Name:         claudeTailerSource,
		Source:       claudeTranscriptSource{tailer: ct},
		Checkpoints:  transcriptCheckpointAdapter{store: ct.checkpoints},
		PollInterval: claudeTailerPollInterval,
		TailInterval: 500 * time.Millisecond,
		Dispatch: func(event tailer.Event, sessionID string) {
			if event.Role == "user" {
				ct.server.AppendUser(event.Text, sessionID, claudeTailerSource)
			} else {
				ct.server.AppendAssistant(event.Text, sessionID, claudeTailerSource)
			}
		},
	})
}

func newCodexTranscriptEngine(ct *CodexTailer) *tailer.Engine {
	return tailer.New(tailer.Config{
		Name:         codexTailerSource,
		Source:       codexTranscriptSource{tailer: ct},
		Checkpoints:  transcriptCheckpointAdapter{store: ct.checkpoints},
		PollInterval: codexPollInterval,
		TailInterval: codexTailInterval,
		StaleTimeout: codexStaleTimeout,
		Dispatch: func(event tailer.Event, sessionID string) {
			if event.Role == "user" {
				ct.server.AppendUser(event.Text, sessionID, codexTailerSource)
			} else {
				ct.server.AppendAssistant(event.Text, sessionID, codexTailerSource)
			}
		},
	})
}

func newGrokTranscriptEngine(gt *GrokTailer) *tailer.Engine {
	source := &grokTranscriptSource{tailer: gt, turns: make(map[string]grok.TurnAccumulator)}
	return tailer.New(tailer.Config{
		Name:         grokSource,
		Source:       source,
		Checkpoints:  transcriptCheckpointAdapter{store: gt.checkpoints},
		PollInterval: grokPollInterval,
		TailInterval: grokTailInterval,
		StaleTimeout: grokStaleTimeout,
		Dispatch: func(event tailer.Event, sessionID string) {
			if event.Role == "user" {
				logGrokAppend("user", sessionID, gt.server.AppendUser(event.Text, sessionID, grokSource))
			} else {
				logGrokAppend("assistant", sessionID, gt.server.AppendAssistant(event.Text, sessionID, grokSource))
			}
		},
	})
}

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/cli-core/agenttranscript"
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

type claudeTranscriptSource struct {
	tailer  *ClaudeTailer
	homeDir func() (string, error)
}

func (s claudeTranscriptSource) DiscoverFiles(ctx context.Context) ([]tailer.FileRef, error) {
	if s.tailer == nil || s.tailer.server == nil || s.tailer.server.sessionStore == nil {
		return nil, nil
	}
	homeDir := s.homeDir
	if homeDir == nil {
		homeDir = os.UserHomeDir
	}
	home, err := homeDir()
	if err != nil {
		return nil, err
	}
	// Identify before discovering. Claude cannot name its own transcript, so a
	// pane whose Stop hook has not fired carries no session id and would be
	// skipped below forever. The resolver adopts what it can prove and leaves
	// anything ambiguous alone; it is cheap when there is nothing to do.
	s.tailer.resolveIdentities(ctx)

	rows, err := s.tailer.server.sessionStore.List(ctx)
	if err != nil {
		return nil, err
	}
	files := make([]tailer.FileRef, 0, len(rows))
	for _, row := range rows {
		if row.Status != sessionstore.StatusLive || row.AgentType != sessionstore.AgentClaude || !row.ArchivedAt.IsZero() {
			continue
		}
		if row.AgentSessionID == "" || row.CWD == "" {
			// Not yet identified. This is a reportable state rather than a
			// silent skip — see conversationAdapter.CaptureStatus, which turns
			// it into an explanation the Messages view can render.
			s.tailer.noteUndiscovered(row.ID)
			continue
		}
		path := claudeTranscriptPath(home, row.CWD, row.AgentSessionID)
		if _, statErr := os.Stat(path); statErr != nil {
			s.tailer.noteUndiscovered(row.ID)
			continue
		}
		s.tailer.noteDiscovered(row.ID)
		files = append(files, tailer.FileRef{Path: path, SessionID: row.ID})
		// Subagents and in-process teammates write their own transcripts
		// beside the session's; their rows land in the session under their
		// own thread so the Messages view can group them.
		for _, subagent := range agenttranscript.ClaudeSubagentFiles(path) {
			files = append(files, tailer.FileRef{Path: subagent, SessionID: row.ID})
		}
	}
	return files, nil
}

func (s claudeTranscriptSource) DecodeLine(path string, line []byte) ([]tailer.Event, error) {
	var thread agenttranscript.ThreadRef
	if s.tailer != nil {
		thread = s.tailer.threads.lookup(path, agenttranscript.ClaudeSubagentThread)
	}
	messages := agenttranscript.ParseMessages(agenttranscript.HarnessClaude, line)
	events := make([]tailer.Event, 0, len(messages))
	for _, message := range messages {
		message.InThread(thread)
		events = append(events, tailer.Event{Role: message.Role, Text: message.Text, OccurredAt: message.OccurredAt, Class: messageClassification(message), Commit: true})
	}
	return events, nil
}

// messageClassification carries a parsed message's native classification to
// the store.
func messageClassification(message agenttranscript.Message) *agenttranscript.Classification {
	return &agenttranscript.Classification{
		Kind: message.Kind, Author: message.Author, AuthorName: message.AuthorName, Label: message.Label, Thread: message.Thread,
	}
}

// transcriptThreads caches the agent thread each transcript file records (read
// once from the file's own metadata), so decoding a line never re-reads it.
type transcriptThreads struct {
	mu    sync.Mutex
	paths map[string]agenttranscript.ThreadRef
}

func (t *transcriptThreads) lookup(path string, read func(string) (agenttranscript.ThreadRef, bool)) agenttranscript.ThreadRef {
	if t == nil {
		return agenttranscript.ThreadRef{}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if thread, ok := t.paths[path]; ok {
		return thread
	}
	thread, ok := read(path)
	if !ok {
		return agenttranscript.ThreadRef{}
	}
	if t.paths == nil {
		t.paths = make(map[string]agenttranscript.ThreadRef)
	}
	t.paths[path] = thread
	return thread
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

func (s codexTranscriptSource) DecodeLine(path string, line []byte) ([]tailer.Event, error) {
	messages := agenttranscript.ParseMessages(agenttranscript.HarnessCodex, line)
	if len(messages) == 0 {
		return nil, nil
	}
	// Every rollout in a session's Codex home is tailed into the session;
	// subagent and approval-review rollouts are filed under their thread.
	var thread agenttranscript.ThreadRef
	if s.tailer != nil {
		thread = s.tailer.threads.lookup(path, agenttranscript.CodexRolloutThread)
	}
	events := make([]tailer.Event, 0, len(messages))
	for _, message := range messages {
		message.InThread(thread)
		if message.Role == "user" {
			input := codex.ParseUserInput(message.Text)
			if input.Text == "" {
				continue
			}
			source := codexTailerSource
			if input.VoiceHandoff {
				source = codexVoiceHandoffSource
			}
			events = append(events, tailer.Event{Role: message.Role, Text: input.Text, Source: source, OccurredAt: message.OccurredAt, Class: messageClassification(message), Commit: true})
		} else {
			events = append(events, tailer.Event{Role: message.Role, Text: message.Text, OccurredAt: message.OccurredAt, Class: messageClassification(message), Commit: true})
		}
	}
	return events, nil
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
	events := make([]tailer.Event, 0, 2+len(turn.ToolCalls))
	if turn.User != "" {
		events = append(events, tailer.Event{Role: "user", Text: turn.User, Commit: true})
	}
	for _, toolCall := range turn.ToolCalls {
		events = append(events, tailer.Event{Role: "assistant", Text: toolCall, Commit: true})
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
		Name:             claudeTailerSource,
		Source:           claudeTranscriptSource{tailer: ct},
		StopUndiscovered: true,
		Checkpoints:      transcriptCheckpointAdapter{store: ct.checkpoints},
		PollInterval:     claudeTailerPollInterval,
		TailInterval:     500 * time.Millisecond,
		Dispatch: func(event tailer.Event, sessionID string) bool {
			return ct.server.AppendTranscriptAt(context.Background(), transcriptRole(event.Role), event.Text, sessionID, claudeTailerSource, event.OccurredAt, event.Class).Appended
		},
	})
}

func newCodexTranscriptEngine(ct *CodexTailer, staleTimeout time.Duration) *tailer.Engine {
	return tailer.New(tailer.Config{
		Name:         codexTailerSource,
		Source:       codexTranscriptSource{tailer: ct},
		Checkpoints:  transcriptCheckpointAdapter{store: ct.checkpoints},
		PollInterval: codexPollInterval,
		TailInterval: codexTailInterval,
		StaleTimeout: staleTimeout,
		Dispatch: func(event tailer.Event, sessionID string) bool {
			source := codexTailerSource
			if event.Role == "user" && event.Source == codexVoiceHandoffSource {
				source = codexVoiceHandoffSource
			}
			return ct.server.AppendTranscriptAt(context.Background(), transcriptRole(event.Role), event.Text, sessionID, source, event.OccurredAt, event.Class).Appended
		},
	})
}

func transcriptRole(role string) ConversationRole {
	if role == "user" {
		return ConversationRoleUser
	}
	return ConversationRoleAssistant
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
		Dispatch: func(event tailer.Event, sessionID string) bool {
			if event.Role == "user" {
				result := gt.server.AppendUser(context.Background(), event.Text, sessionID, grokSource)
				logGrokAppend("user", sessionID, result)
				return result.Appended
			} else {
				result := gt.server.AppendAssistant(context.Background(), event.Text, sessionID, grokSource)
				logGrokAppend("assistant", sessionID, result)
				return result.Appended
			}
		},
	})
}

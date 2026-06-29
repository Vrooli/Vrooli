package main

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"web-console/backends/grok"
	"web-console/internal/sessionstore"
)

const (
	grokPollInterval = 2 * time.Second
	grokTailInterval = 500 * time.Millisecond
	grokStaleTimeout = 1 * time.Hour
	grokSource       = "grok_tailer"
)

// GrokTailer watches per-session GROK_HOME transcript trees and routes the
// user/assistant text from grok's append-only updates.jsonl back to the owning
// web-console session. It mirrors CodexTailer's lifecycle (one watcher goroutine
// per discovered transcript file, byte-offset checkpoints for restart/backfill)
// but parses grok's ACP turn structure and only advances the persisted
// checkpoint at turn-completion boundaries so a mid-turn restart re-accumulates
// without duplicating.
//
// DOC: docs/guides/CONVERSATION_TRACKING.md
// DOC: docs/internal/TEMPORAL-FLOWS.md
type GrokTailer struct {
	server       *Server
	checkpoints  AgentTranscriptCheckpointStore
	staleTimeout time.Duration
	mu           sync.Mutex
	watchers     map[string]string // updates.jsonl path -> session id
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

func NewGrokTailer(server *Server) *GrokTailer {
	return &GrokTailer{
		server:      server,
		checkpoints: server.agentCheckpointStore,
		watchers:    make(map[string]string),
		stopCh:      make(chan struct{}),
	}
}

func (gt *GrokTailer) Start() {
	gt.wg.Add(1)
	go gt.pollLoop()
}

func (gt *GrokTailer) Stop() {
	close(gt.stopCh)
	gt.wg.Wait()
}

func (gt *GrokTailer) pollLoop() {
	defer gt.wg.Done()
	ticker := time.NewTicker(grokPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-gt.stopCh:
			return
		case <-ticker.C:
			gt.scanForNewFiles()
		}
	}
}

// scanForNewFiles discovers updates.jsonl files under each live session's
// GROK_HOME transcript root. grok lays them out as
// <sessionsDir>/<url-encoded-cwd>/<session-id>/updates.jsonl, so we glob two
// levels deep. Attribution is by construction: the sessionsDir is per pane.
func (gt *GrokTailer) scanForNewFiles() {
	for _, session := range gt.server.sessions.List() {
		baseDir := sessionGrokSessionsDir(session.ID)
		matches, err := filepath.Glob(filepath.Join(baseDir, "*", "*", "updates.jsonl"))
		if err != nil {
			continue
		}
		for _, path := range matches {
			gt.mu.Lock()
			_, known := gt.watchers[path]
			if !known {
				gt.watchers[path] = session.ID
				gt.wg.Add(1)
			}
			gt.mu.Unlock()
			if !known {
				go gt.captureAgentInfo(path, session.ID)
				go gt.tailFile(path, session.ID)
			}
		}
	}
}

func (gt *GrokTailer) loadCheckpointOffset(path string) int64 {
	if gt.checkpoints == nil {
		return 0
	}
	cp, ok, err := gt.checkpoints.Get(grokSource, path)
	if err != nil {
		log.Printf("grok-tailer: checkpoint load failed for %s: %v", path, err)
		return 0
	}
	if !ok {
		return 0
	}
	offset, err := strconv.ParseInt(cp.Cursor, 10, 64)
	if err != nil {
		return 0
	}
	return offset
}

func (gt *GrokTailer) saveCheckpoint(path, sessionID string, offset int64) {
	if gt.checkpoints == nil {
		return
	}
	if err := gt.checkpoints.Save(AgentTranscriptCheckpoint{
		Source:    grokSource,
		SourceKey: path,
		SessionID: sessionID,
		Cursor:    strconv.FormatInt(offset, 10),
		UpdatedAt: time.Now().UTC(),
	}); err != nil {
		log.Printf("grok-tailer: checkpoint save failed for %s: %v", path, err)
	}
}

func (gt *GrokTailer) tailFile(path, sessionID string) {
	defer gt.wg.Done()
	defer func() {
		gt.mu.Lock()
		delete(gt.watchers, path)
		gt.mu.Unlock()
	}()

	f, err := os.Open(path)
	if err != nil {
		log.Printf("grok-tailer: failed to open %s: %v", path, err)
		return
	}
	defer f.Close()

	startOffset := gt.loadCheckpointOffset(path)
	if stat, err := f.Stat(); err == nil && startOffset > stat.Size() {
		startOffset = 0
	}
	if _, err := f.Seek(startOffset, io.SeekStart); err != nil {
		log.Printf("grok-tailer: seek failed for %s: %v", path, err)
		return
	}

	reader := bufio.NewReader(f)
	// boundaryOffset is the byte offset of the last turn boundary we have fully
	// emitted (and the only offset we ever persist). currentOffset tracks the
	// raw read position; the gap between them is the in-flight, not-yet-emitted
	// turn held in acc.
	boundaryOffset := startOffset
	currentOffset := startOffset
	var acc grok.TurnAccumulator

	ticker := time.NewTicker(grokTailInterval)
	defer ticker.Stop()

	timeout := grokStaleTimeout
	if gt.staleTimeout > 0 {
		timeout = gt.staleTimeout
	}
	staleTimer := time.NewTimer(timeout)
	defer staleTimer.Stop()

	resetStaleTimer := func() {
		if !staleTimer.Stop() {
			select {
			case <-staleTimer.C:
			default:
			}
		}
		staleTimer.Reset(timeout)
	}

	processAvailable := func() bool {
		processed := false
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				// No complete line available (partial trailing write or EOF).
				// Leave currentOffset where it is; the partial bytes are
				// re-read once the newline lands.
				return processed
			}
			processed = true
			currentOffset += int64(len(line))
			rec, ok := grok.ParseUpdateLine(line)
			if !ok {
				// Malformed or irrelevant complete line — skip it. The persisted
				// checkpoint does not advance until the next turn boundary.
				continue
			}
			turn, done := acc.Add(rec)
			if !done {
				continue
			}
			if turn.User != "" {
				gt.server.AppendUser(turn.User, sessionID, grokSource)
			}
			if turn.Assistant != "" {
				gt.server.AppendAssistant(turn.Assistant, sessionID, grokSource)
			}
			boundaryOffset = currentOffset
			gt.saveCheckpoint(path, sessionID, boundaryOffset)
		}
	}

	if processAvailable() {
		resetStaleTimer()
	}

	sessionAlive := func() bool {
		if gt.server == nil || gt.server.sessions == nil {
			return true
		}
		for _, s := range gt.server.sessions.List() {
			if s.ID == sessionID {
				return true
			}
		}
		return false
	}

	for {
		select {
		case <-gt.stopCh:
			return
		case <-staleTimer.C:
			if _, err := os.Stat(path); err != nil {
				log.Printf("grok-tailer: transcript %s vanished; stopping watcher", path)
				return
			}
			if !sessionAlive() {
				log.Printf("grok-tailer: session %s is gone; stopping watcher for %s", sessionID, path)
				return
			}
			staleTimer.Reset(timeout)
		case <-ticker.C:
			if processAvailable() {
				resetStaleTimer()
			}
		}
	}
}

// captureAgentInfo reads the session summary.json sibling of updates.jsonl and
// records grok's session id + cwd on the owning web-console session so orphan
// recovery can reattach with `grok --resume <id>`. Tolerant of missing/garbage
// data — failures log nothing fatal and never touch existing fields.
func (gt *GrokTailer) captureAgentInfo(updatesPath, sessionID string) {
	if gt.server == nil || gt.server.sessionStore == nil {
		return
	}
	summaryPath := filepath.Join(filepath.Dir(updatesPath), "summary.json")
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		// summary.json may not exist yet at first sight; the next scan re-tries.
		return
	}
	var summary struct {
		Info struct {
			ID  string `json:"id"`
			CWD string `json:"cwd"`
		} `json:"info"`
	}
	if err := json.Unmarshal(data, &summary); err != nil || summary.Info.ID == "" {
		return
	}
	info := sessionstore.AgentInfo{
		AgentType:       sessionstore.AgentGrok,
		AgentSessionID:  summary.Info.ID,
		CWD:             summary.Info.CWD,
		LastRolloutPath: updatesPath,
		LastActivityAt:  time.Now(),
	}
	if err := gt.server.sessionStore.UpdateAgentInfo(sessionID, info); err != nil {
		log.Printf("grok-tailer: UpdateAgentInfo for %s: %v", sessionID, err)
	}
}

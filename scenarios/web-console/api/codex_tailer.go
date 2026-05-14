package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"web-console/internal/sessionstore"
)

const (
	codexPollInterval = 2 * time.Second
	codexTailInterval = 500 * time.Millisecond
	codexStaleTimeout = 1 * time.Hour
)

// CodexTailer watches per-session CODEX_HOME rollout files and routes
// assistant responses back to the owning web-console session.
type CodexTailer struct {
	server       *Server
	checkpoints  CodexCheckpointStore
	staleTimeout time.Duration
	mu           sync.Mutex
	watchers     map[string]string // rollout path -> session id
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

func NewCodexTailer(server *Server) *CodexTailer {
	return &CodexTailer{
		server:      server,
		checkpoints: server.codexCheckpointStore,
		watchers:    make(map[string]string),
		stopCh:      make(chan struct{}),
	}
}

func (ct *CodexTailer) Start() {
	ct.wg.Add(1)
	go ct.pollLoop()
}

func (ct *CodexTailer) Stop() {
	close(ct.stopCh)
	ct.wg.Wait()
}

func (ct *CodexTailer) pollLoop() {
	defer ct.wg.Done()
	ticker := time.NewTicker(codexPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ct.stopCh:
			return
		case <-ticker.C:
			ct.scanForNewFiles()
		}
	}
}

func (ct *CodexTailer) scanForNewFiles() {
	now := time.Now()
	for _, session := range ct.server.sessions.List() {
		baseDir := sessionCodexSessionsDir(session.ID)
		for _, offset := range []int{0, -1} {
			d := now.AddDate(0, 0, offset)
			dir := filepath.Join(baseDir, d.Format("2006"), d.Format("01"), d.Format("02"))
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, entry := range entries {
				if entry.IsDir() || !strings.HasPrefix(entry.Name(), "rollout-") || !strings.HasSuffix(entry.Name(), ".jsonl") {
					continue
				}
				path := filepath.Join(dir, entry.Name())
				ct.mu.Lock()
				_, known := ct.watchers[path]
				if !known {
					ct.watchers[path] = session.ID
					ct.wg.Add(1)
				}
				ct.mu.Unlock()
				if !known {
					// One-shot agent-identity capture: parse the rollout's
					// session_meta line so the orphan recovery flow can
					// reattach this codex session by id without grovelling
					// rollouts at recovery time. Cheap (rollouts are
					// line-oriented; the first line is < 4KB) and only runs
					// once per (session, rollout) pair.
					go ct.captureAgentInfo(path, session.ID)
					go ct.tailFile(path, session.ID)
				}
			}
		}
	}
}

func (ct *CodexTailer) tailFile(path, sessionID string) {
	defer ct.wg.Done()
	defer func() {
		ct.mu.Lock()
		delete(ct.watchers, path)
		ct.mu.Unlock()
	}()

	f, err := os.Open(path)
	if err != nil {
		log.Printf("codex-tailer: failed to open %s: %v", path, err)
		return
	}
	defer f.Close()

	startOffset := int64(0)
	if ct.checkpoints != nil {
		if checkpoint, ok, err := ct.checkpoints.Get(path); err != nil {
			log.Printf("codex-tailer: checkpoint load failed for %s: %v", path, err)
		} else if ok {
			startOffset = checkpoint.Offset
		}
	}
	if stat, err := f.Stat(); err == nil && startOffset > stat.Size() {
		startOffset = 0
	}
	if _, err := f.Seek(startOffset, io.SeekStart); err != nil {
		log.Printf("codex-tailer: seek failed for %s: %v", path, err)
		return
	}

	reader := bufio.NewReader(f)
	currentOffset := startOffset
	ticker := time.NewTicker(codexTailInterval)
	defer ticker.Stop()

	timeout := codexStaleTimeout
	if ct.staleTimeout > 0 {
		timeout = ct.staleTimeout
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
				return processed
			}
			processed = true
			currentOffset += int64(len(line))
			line = bytes.TrimSpace(line)
			if len(line) == 0 {
				ct.saveCheckpoint(path, sessionID, currentOffset)
				continue
			}
			if text := ExtractAssistantText(line); text != "" {
				ct.server.appendConversationEvent(text, sessionID, "codex_tailer")
			} else if text := ExtractUserText(line); text != "" {
				ct.server.appendUserConversationEvent(text, sessionID, "codex_tailer")
			}
			ct.saveCheckpoint(path, sessionID, currentOffset)
		}
	}

	if processAvailable() {
		resetStaleTimer()
	}

	// sessionAlive reports whether the owning session is still known to the
	// server. We keep the watcher alive for as long as the session exists —
	// agents can go quiet for hours and resume. The previous hard-timeout
	// exit killed watchers mid-session and caused the "messages stop
	// updating until I reopen web-console" bug.
	sessionAlive := func() bool {
		if ct.server == nil || ct.server.sessions == nil {
			// No session manager to consult (tests); assume alive so the
			// watcher remains in effect until Stop() or deletion of file.
			return true
		}
		for _, s := range ct.server.sessions.List() {
			if s.ID == sessionID {
				return true
			}
		}
		return false
	}

	for {
		select {
		case <-ct.stopCh:
			return
		case <-staleTimer.C:
			// If the file is gone or the session has been deleted, stop.
			// Otherwise keep watching indefinitely — quiet agents are
			// normal and must not silently lose the side-channel.
			if _, err := os.Stat(path); err != nil {
				log.Printf("codex-tailer: rollout %s vanished; stopping watcher", path)
				return
			}
			if !sessionAlive() {
				log.Printf("codex-tailer: session %s is gone; stopping watcher for %s", sessionID, path)
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

// captureAgentInfo reads the first line of a codex rollout and writes the
// codex session id, cwd, and rollout path into the owning web-console
// session's metadata row. Tolerant of missing/garbage data — failures log
// once and do not stop the tailer or touch existing fields.
func (ct *CodexTailer) captureAgentInfo(path, sessionID string) {
	if ct.server == nil || ct.server.sessionStore == nil {
		return
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	// Read up to ~64KB on the first line (codex's session_meta carries the
	// embedded base_instructions blob, which can be ~25KB; pad for safety).
	r := bufio.NewReaderSize(f, 128*1024)
	line, err := r.ReadBytes('\n')
	if err != nil && err != io.EOF {
		return
	}
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return
	}
	var meta struct {
		Type    string `json:"type"`
		Payload struct {
			ID  string `json:"id"`
			CWD string `json:"cwd"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(line, &meta); err != nil {
		return
	}
	if meta.Type != "session_meta" || meta.Payload.ID == "" {
		return
	}
	info := sessionstore.AgentInfo{
		AgentType:       sessionstore.AgentCodex,
		AgentSessionID:  meta.Payload.ID,
		CWD:             meta.Payload.CWD,
		LastRolloutPath: path,
		LastActivityAt:  time.Now(),
	}
	if err := ct.server.sessionStore.UpdateAgentInfo(sessionID, info); err != nil {
		log.Printf("codex-tailer: UpdateAgentInfo for %s: %v", sessionID, err)
	}
}

func (ct *CodexTailer) saveCheckpoint(path, sessionID string, offset int64) {
	if ct.checkpoints == nil {
		return
	}
	if err := ct.checkpoints.Save(CodexRolloutCheckpoint{
		Path:      path,
		SessionID: sessionID,
		Offset:    offset,
		UpdatedAt: time.Now().UTC(),
	}); err != nil {
		log.Printf("codex-tailer: checkpoint save failed for %s: %v", path, err)
	}
}

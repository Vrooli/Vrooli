package main

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
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

	processAvailable := func() {
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				return
			}
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

	processAvailable()

	for {
		select {
		case <-ct.stopCh:
			return
		case <-staleTimer.C:
			log.Printf("codex-tailer: stopping stale watcher for %s", path)
			return
		case <-ticker.C:
			processAvailable()
		}
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

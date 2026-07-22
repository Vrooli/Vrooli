package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"web-console/internal/sessionstore"
)

const (
	claudeTailerSource       = "claude_tailer"
	claudeTailerPollInterval = 2 * time.Second
)

// ClaudeTailer reads Claude Code's append-only project transcripts directly.
// It deliberately remains active alongside Claude's hook transport: hooks are
// low-latency, while this cursor-backed reader protects resumed sessions and
// future hook regressions. ConversationStore deduplicates identical text
// across sources inside its short replay window.
type ClaudeTailer struct {
	server      *Server
	checkpoints AgentTranscriptCheckpointStore
	stopCh      chan struct{}
	wg          sync.WaitGroup
	mu          sync.Mutex
	missing     map[string]bool
}

func NewClaudeTailer(server *Server) *ClaudeTailer {
	return &ClaudeTailer{
		server:      server,
		checkpoints: server.agentCheckpointStore,
		stopCh:      make(chan struct{}),
		missing:     make(map[string]bool),
	}
}

func (ct *ClaudeTailer) Start() {
	ct.wg.Add(1)
	go ct.pollLoop()
}

func (ct *ClaudeTailer) Stop() {
	close(ct.stopCh)
	ct.wg.Wait()
}

func (ct *ClaudeTailer) pollLoop() {
	defer ct.wg.Done()
	ticker := time.NewTicker(claudeTailerPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ct.stopCh:
			return
		case <-ticker.C:
			ct.scan()
		}
	}
}

func claudeTranscriptPath(home, cwd, agentSessionID string) string {
	key := strings.ReplaceAll(filepath.Clean(cwd), string(filepath.Separator), "-")
	return filepath.Join(home, ".claude", "projects", key, agentSessionID+".jsonl")
}

func (ct *ClaudeTailer) scan() {
	if ct.server == nil || ct.server.sessionStore == nil {
		return
	}
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("claude-tailer: resolve home directory: %v", err)
		return
	}
	rows, err := ct.server.sessionStore.List()
	if err != nil {
		log.Printf("claude-tailer: list sessions: %v", err)
		return
	}
	for _, row := range rows {
		if row.Status != sessionstore.StatusLive || row.AgentType != sessionstore.AgentClaude || row.AgentSessionID == "" || row.CWD == "" {
			continue
		}
		path := claudeTranscriptPath(home, row.CWD, row.AgentSessionID)
		if _, err := os.Stat(path); err != nil {
			ct.logMissingOnce(path, row.ID)
			continue
		}
		ct.clearMissing(path)
		ct.tailAvailable(path, row.ID)
	}
}

func (ct *ClaudeTailer) logMissingOnce(path, sessionID string) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	if ct.missing[path] {
		return
	}
	ct.missing[path] = true
	log.Printf("claude-tailer: transcript missing for session %s: %s", sessionID, path)
}

func (ct *ClaudeTailer) clearMissing(path string) {
	ct.mu.Lock()
	delete(ct.missing, path)
	ct.mu.Unlock()
}

func (ct *ClaudeTailer) tailAvailable(path, sessionID string) {
	f, err := os.Open(path)
	if err != nil {
		ct.logMissingOnce(path, sessionID)
		return
	}
	defer f.Close()

	offset := ct.loadOffset(path)
	if info, err := f.Stat(); err == nil && offset > info.Size() {
		offset = 0
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		log.Printf("claude-tailer: seek %s: %v", path, err)
		return
	}
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("claude-tailer: read %s: %v", path, err)
			}
			return
		}
		offset += int64(len(line))
		ct.dispatchLine(bytes.TrimSpace(line), sessionID)
		ct.saveOffset(path, sessionID, offset)
	}
}

func (ct *ClaudeTailer) loadOffset(path string) int64 {
	if ct.checkpoints == nil {
		return 0
	}
	cp, ok, err := ct.checkpoints.Get(claudeTailerSource, path)
	if err != nil || !ok {
		return 0
	}
	offset, err := strconv.ParseInt(cp.Cursor, 10, 64)
	if err != nil {
		return 0
	}
	return offset
}

func (ct *ClaudeTailer) saveOffset(path, sessionID string, offset int64) {
	if ct.checkpoints == nil {
		return
	}
	if err := ct.checkpoints.Save(AgentTranscriptCheckpoint{
		Source: claudeTailerSource, SourceKey: path, SessionID: sessionID,
		Cursor: strconv.FormatInt(offset, 10), UpdatedAt: time.Now().UTC(),
	}); err != nil {
		log.Printf("claude-tailer: checkpoint save %s: %v", path, err)
	}
}

type claudeTranscriptEntry struct {
	Type    string `json:"type"`
	Message struct {
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

type claudeContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func claudeTextBlocks(content json.RawMessage) ([]claudeContentBlock, bool) {
	var blocks []claudeContentBlock
	if err := json.Unmarshal(content, &blocks); err != nil {
		return nil, false
	}
	return blocks, true
}

func (ct *ClaudeTailer) dispatchLine(line []byte, sessionID string) {
	if len(line) == 0 || ct.server == nil {
		return
	}
	var entry claudeTranscriptEntry
	if err := json.Unmarshal(line, &entry); err != nil {
		return
	}
	blocks, ok := claudeTextBlocks(entry.Message.Content)
	if !ok {
		return
	}
	if entry.Type == "user" {
		for _, block := range blocks {
			if block.Type == "tool_result" {
				return
			}
		}
		for _, block := range blocks {
			if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
				ct.server.AppendUser(block.Text, sessionID, claudeTailerSource)
			}
		}
		return
	}
	if entry.Type != "assistant" {
		return
	}
	for _, block := range blocks {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			ct.server.AppendAssistant(block.Text, sessionID, claudeTailerSource)
		}
	}
}

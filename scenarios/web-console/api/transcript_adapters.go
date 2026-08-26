package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"web-console/internal/sessionstore"
	"web-console/internal/tailer"
)

const (
	claudeTailerSource       = "claude_tailer"
	claudeTailerPollInterval = 2 * time.Second
	codexTailerSource        = "codex_tailer"
	codexPollInterval        = 2 * time.Second
	codexTailInterval        = 500 * time.Millisecond
	codexStaleTimeout        = time.Hour
	grokPollInterval         = 2 * time.Second
	grokTailInterval         = 500 * time.Millisecond
	grokStaleTimeout         = time.Hour
	grokSource               = "grok_tailer"
)

// The three public adapter types remain as narrow compatibility seams for
// server wiring and package-local fixtures. Polling, file reads, decoding,
// checkpointing, and shutdown are owned by internal/tailer.Engine.
type ClaudeTailer struct {
	server      *Server
	checkpoints AgentTranscriptCheckpointStore
	engine      *tailer.Engine
	mu          sync.Mutex
	missing     map[string]bool
}

type CodexTailer struct {
	server       *Server
	checkpoints  AgentTranscriptCheckpointStore
	staleTimeout time.Duration // retained for fixture source compatibility
	engine       *tailer.Engine
	mu           sync.Mutex
	watchers     map[string]string
}

type GrokTailer struct {
	server       *Server
	checkpoints  AgentTranscriptCheckpointStore
	staleTimeout time.Duration // retained for fixture source compatibility
	engine       *tailer.Engine
	mu           sync.Mutex
	watchers     map[string]string
}

func NewClaudeTailer(server *Server) *ClaudeTailer {
	ct := &ClaudeTailer{server: server, missing: make(map[string]bool)}
	if server != nil {
		ct.checkpoints = server.agentCheckpointStore
	}
	ct.engine = newClaudeTranscriptEngine(ct)
	return ct
}

func NewCodexTailer(server *Server) *CodexTailer {
	ct := &CodexTailer{server: server, watchers: make(map[string]string)}
	if server != nil {
		ct.checkpoints = server.agentCheckpointStore
	}
	ct.engine = newCodexTranscriptEngine(ct)
	return ct
}

func NewGrokTailer(server *Server) *GrokTailer {
	gt := &GrokTailer{server: server, watchers: make(map[string]string)}
	if server != nil {
		gt.checkpoints = server.agentCheckpointStore
	}
	gt.engine = newGrokTranscriptEngine(gt)
	return gt
}

func (ct *ClaudeTailer) Start() { ct.engine.Start() }
func (ct *CodexTailer) Start()  { ct.engine.Start() }
func (gt *GrokTailer) Start()   { gt.engine.Start() }

func (ct *ClaudeTailer) Stop() { ct.engine.Stop() }
func (ct *CodexTailer) Stop()  { ct.engine.Stop() }
func (gt *GrokTailer) Stop()   { gt.engine.Stop() }

// These scan methods are retained for package-local tests that need one
// deterministic discovery pass without waiting for the engine ticker.
func (ct *ClaudeTailer) scan() { ct.engine.Scan(context.Background()) }

func claudeTranscriptPath(home, cwd, agentSessionID string) string {
	key := strings.ReplaceAll(filepath.Clean(cwd), string(filepath.Separator), "-")
	return filepath.Join(home, ".claude", "projects", key, agentSessionID+".jsonl")
}

func (ct *CodexTailer) scanForNewFiles() {
	if ct.engine == nil {
		return
	}
	refs, _ := (codexTranscriptSource{tailer: ct}).DiscoverFiles(context.Background())
	ct.mu.Lock()
	for _, ref := range refs {
		ct.watchers[ref.Path] = ref.SessionID
	}
	ct.mu.Unlock()
	ct.engine.Scan(context.Background())
}

func (gt *GrokTailer) scanForNewFiles() { gt.engine.Scan(context.Background()) }

func (ct *CodexTailer) captureAgentInfo(path, sessionID string) {
	if ct.server == nil || ct.server.sessionStore == nil {
		return
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
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
	if err := json.Unmarshal(line, &meta); err != nil || meta.Type != "session_meta" || meta.Payload.ID == "" {
		return
	}
	info := sessionstore.AgentInfo{
		AgentType:       sessionstore.AgentCodex,
		AgentSessionID:  meta.Payload.ID,
		CWD:             meta.Payload.CWD,
		LastRolloutPath: path,
		LastActivityAt:  time.Now(),
	}
	if err := ct.server.sessionStore.UpdateAgentInfo(context.Background(), sessionID, info); err != nil {
		log.Printf("codex-tailer: UpdateAgentInfo for %s: %v", sessionID, err)
	}
}

func (gt *GrokTailer) captureAgentInfo(updatesPath, sessionID string) {
	if gt.server == nil || gt.server.sessionStore == nil {
		return
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(updatesPath), "summary.json"))
	if err != nil {
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
	if err := gt.server.sessionStore.UpdateAgentInfo(context.Background(), sessionID, info); err != nil {
		log.Printf("grok-tailer: UpdateAgentInfo for %s: %v", sessionID, err)
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

func logGrokAppend(role, sessionID string, result ConversationAppendResult) {
	if result.Appended {
		log.Printf("grok-tailer: appended %s message for session %s (seq=%d)", role, sessionID, result.Sequence)
		return
	}
	log.Printf("grok-tailer: DROPPED %s message for session %s: code=%s reason=%q", role, sessionID, result.Code, result.Reason)
}

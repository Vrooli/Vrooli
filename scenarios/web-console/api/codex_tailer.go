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
	// codexPollInterval is how often we scan for new rollout files.
	codexPollInterval = 2 * time.Second
	// codexTailInterval is how often we poll an open file for new lines.
	codexTailInterval = 500 * time.Millisecond
	// codexStaleTimeout is how long we watch a single rollout file before
	// assuming it's stale. Codex sessions rarely exceed 1 hour.
	codexStaleTimeout = 1 * time.Hour
)

// CodexTailer watches for new Codex rollout JSONL files and extracts
// assistant responses for TTS delivery.
type CodexTailer struct {
	server       *Server       // for deliverTTS access
	baseDir      string        // ~/.codex/sessions
	staleTimeout time.Duration // override codexStaleTimeout for testing; 0 = use default
	mu           sync.Mutex
	watchers     map[string]struct{} // tracked rollout file paths
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

// NewCodexTailer creates a tailer. Does NOT start polling yet.
func NewCodexTailer(server *Server) *CodexTailer {
	home, _ := os.UserHomeDir()
	return &CodexTailer{
		server:   server,
		baseDir:  filepath.Join(home, ".codex", "sessions"),
		watchers: make(map[string]struct{}),
		stopCh:   make(chan struct{}),
	}
}

// Start begins the polling loop that watches for new rollout files.
func (ct *CodexTailer) Start() {
	ct.wg.Add(1)
	go ct.pollLoop()
}

// Stop signals the tailer to shut down and waits for goroutines.
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
	// Walk today's and yesterday's date directories for rollout files.
	now := time.Now()
	for _, offset := range []int{0, -1} {
		d := now.AddDate(0, 0, offset)
		dir := filepath.Join(ct.baseDir, d.Format("2006"), d.Format("01"), d.Format("02"))
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
				ct.watchers[path] = struct{}{}
				ct.wg.Add(1)
			}
			ct.mu.Unlock()
			if !known {
				go ct.tailFile(path)
			}
		}
	}
}

func (ct *CodexTailer) tailFile(path string) {
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

	// Seek to end — only process new lines.
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		log.Printf("codex-tailer: seek failed for %s: %v", path, err)
		return
	}

	reader := bufio.NewReader(f)
	ticker := time.NewTicker(codexTailInterval)
	defer ticker.Stop()

	timeout := codexStaleTimeout
	if ct.staleTimeout > 0 {
		timeout = ct.staleTimeout
	}
	staleTimer := time.NewTimer(timeout)
	defer staleTimer.Stop()

	for {
		select {
		case <-ct.stopCh:
			return
		case <-staleTimer.C:
			log.Printf("codex-tailer: stopping stale watcher for %s", path)
			return
		case <-ticker.C:
			for {
				line, err := reader.ReadBytes('\n')
				if err != nil {
					break // No more data right now
				}
				line = bytes.TrimSpace(line)
				if len(line) == 0 {
					continue
				}
				text := ExtractAssistantText(line)
				if text != "" {
					ct.server.deliverTTS(text, "", "codex_tailer")
				}
			}
		}
	}
}

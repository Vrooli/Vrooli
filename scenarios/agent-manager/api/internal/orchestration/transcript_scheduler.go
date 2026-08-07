// Transcript scheduling owns the periodic, idempotent adoption of external
// harness transcripts into the agent-manager evidence corpus.
package orchestration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
)

// TranscriptImportSummary describes one scheduled corpus sweep.
type TranscriptImportSummary struct {
	Scanned  int      `json:"scanned"`
	Imported int      `json:"imported"`
	Existing int      `json:"existing"`
	Skipped  int      `json:"skipped"`
	Failures []string `json:"failures,omitempty"`
}

// TranscriptImportScheduler keeps external harness evidence current. It uses
// ImportTranscript's source-session idempotency check, so repeated sweeps are
// safe and never create duplicate runs.
type TranscriptImportScheduler struct {
	orchestrator *Orchestrator
	interval     time.Duration
	cancel       context.CancelFunc
	done         chan struct{}
	mu           sync.Mutex
}

func NewTranscriptImportScheduler(orchestrator *Orchestrator, interval time.Duration) *TranscriptImportScheduler {
	if interval <= 0 {
		interval = 6 * time.Hour
	}
	return &TranscriptImportScheduler{orchestrator: orchestrator, interval: interval}
}

func (s *TranscriptImportScheduler) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil || s.orchestrator == nil {
		return
	}
	workerCtx, cancel := context.WithCancel(ctx)
	s.cancel, s.done = cancel, make(chan struct{})
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if _, err := s.RunOnce(workerCtx); err != nil {
					obs.Component("transcript-importer").Warn("scheduled transcript import failed", obs.KeyError, err.Error())
				}
			case <-workerCtx.Done():
				return
			}
		}
	}()
}

func (s *TranscriptImportScheduler) Stop() {
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	s.cancel, s.done = nil, nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

// RunOnce scans only governed durable harness roots. It is exported so the
// lifecycle can be tested without waiting six hours and so operators can use
// the same idempotent operation when diagnosing a scheduler.
func (s *TranscriptImportScheduler) RunOnce(ctx context.Context) (TranscriptImportSummary, error) {
	var summary TranscriptImportSummary
	if s == nil || s.orchestrator == nil {
		return summary, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return summary, err
	}
	roots := []struct {
		path    string
		harness string
		runner  domain.RunnerType
	}{
		{filepath.Join(home, ".codex", "sessions"), "codex", domain.RunnerTypeCodex},
		{filepath.Join(home, ".claude", "projects"), "claude-code", domain.RunnerTypeClaudeCode},
	}
	for _, root := range roots {
		_ = filepath.WalkDir(root.path, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry == nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
				return nil
			}
			summary.Scanned++
			sessionID := transcriptSessionIDFromPath(path)
			if sessionID == "" {
				summary.Skipped++
				return nil
			}
			_, importErr := s.orchestrator.ImportTranscript(ctx, ImportTranscriptRequest{Path: path, RunnerType: root.runner, SourceHarness: root.harness, SourceSessionID: sessionID})
			if importErr != nil {
				if strings.Contains(importErr.Error(), "already") {
					summary.Existing++
				} else if existing, lookupErr := s.orchestrator.runs.GetByImportProvenance(ctx, root.harness, sessionID); lookupErr == nil && existing != nil {
					summary.Existing++
				} else {
					summary.Failures = append(summary.Failures, path+": "+importErr.Error())
				}
				return nil
			}
			summary.Imported++
			return nil
		})
	}
	return summary, nil
}

func transcriptSessionIDFromPath(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	return transcriptSessionID(file)
}

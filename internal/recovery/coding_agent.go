package recovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// CodingAgentRecoveryWindow is the durable per-scenario admission window.
	// A failed launch still consumes the window: repeated launches are not a
	// recovery strategy when the underlying credential or runner is unavailable.
	CodingAgentRecoveryWindow = time.Hour
	// CodingAgentRecoveryTimeout bounds one asynchronous investigation. The
	// autoheal API remains available while the coding agent is working.
	CodingAgentRecoveryTimeout = 10 * time.Minute
	codingAgentRecordMode      = 0o600
	codingAgentDirectoryMode   = 0o755
)

var DefaultCodingAgentRunners = []string{"opencode", "codex", "claude", "grok"}

// CodingAgentRequest is the non-secret task envelope passed to a recovery
// runner. Credentials remain in the runner's configured stores and never enter
// this durable ledger.
type CodingAgentRequest struct {
	Scenario   string
	Reason     string
	Requester  string
	WorkingDir string
	Claim      string
}

// CodingAgentRecord is the durable admission and outcome receipt for one
// investigation attempt.
type CodingAgentRecord struct {
	ID         string    `json:"id"`
	Scenario   string    `json:"scenario"`
	Reason     string    `json:"reason"`
	Requester  string    `json:"requester"`
	Runner     string    `json:"runner,omitempty"`
	Outcome    string    `json:"outcome"`
	Error      string    `json:"error,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
}

// CodingAgentRunner launches one named coding-agent runner. The implementation
// in main.go delegates to the existing `vrooli agent launch` command, which
// already prefers Agent Manager attachment and falls back to the native runner.
type CodingAgentRunner func(context.Context, string, CodingAgentRequest) error

type CodingAgentBroker struct {
	mu      sync.Mutex
	path    string
	window  time.Duration
	timeout time.Duration
	now     func() time.Time
	runners []string
	records []CodingAgentRecord
	run     CodingAgentRunner
}

func NewCodingAgentBroker(path string, run CodingAgentRunner) (*CodingAgentBroker, error) {
	if run == nil {
		return nil, errors.New("coding-agent recovery runner is required")
	}
	b := &CodingAgentBroker{
		path:    path,
		window:  CodingAgentRecoveryWindow,
		timeout: CodingAgentRecoveryTimeout,
		now:     time.Now,
		runners: append([]string(nil), DefaultCodingAgentRunners...),
		run:     run,
	}
	if strings.TrimSpace(path) == "" {
		return b, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return b, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read coding-agent recovery records: %w", err)
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &b.records); err != nil {
			return nil, fmt.Errorf("decode coding-agent recovery records: %w", err)
		}
	}
	return b, nil
}

// Submit admits at most one investigation for a scenario in the rolling
// window. A duplicate returns the existing receipt and does not start another
// process, which makes repeated health ticks and API restarts idempotent.
func (b *CodingAgentBroker) Submit(req CodingAgentRequest) (CodingAgentRecord, error) {
	req.Scenario = strings.TrimSpace(req.Scenario)
	req.Requester = strings.TrimSpace(req.Requester)
	if req.Scenario == "" || req.Requester == "" {
		return CodingAgentRecord{}, errors.New("coding-agent recovery scenario and requester are required")
	}
	now := b.now().UTC()
	b.mu.Lock()
	defer b.mu.Unlock()
	cutoff := now.Add(-b.window)
	for _, record := range b.records {
		if record.Scenario == req.Scenario && record.CreatedAt.After(cutoff) {
			return record, nil
		}
	}
	record := CodingAgentRecord{
		ID:        uuid.NewString(),
		Scenario:  req.Scenario,
		Reason:    truncateCodingAgentString(req.Reason),
		Requester: req.Requester,
		Outcome:   "scheduled",
		CreatedAt: now,
	}
	b.records = append(b.records, record)
	if err := b.persistLocked(); err != nil {
		b.records = b.records[:len(b.records)-1]
		return CodingAgentRecord{}, fmt.Errorf("persist coding-agent recovery admission: %w", err)
	}
	go b.execute(record.ID, req)
	return record, nil
}

func (b *CodingAgentBroker) execute(id string, req CodingAgentRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()
	b.update(id, func(record *CodingAgentRecord) {
		record.Outcome = "running"
	})
	var lastErr error
	for _, runner := range b.runners {
		if err := b.run(ctx, runner, req); err == nil {
			b.update(id, func(record *CodingAgentRecord) {
				record.Runner = runner
				record.Outcome = "succeeded"
				record.FinishedAt = b.now().UTC()
			})
			return
		} else {
			lastErr = err
		}
		if ctx.Err() != nil {
			break
		}
	}
	b.update(id, func(record *CodingAgentRecord) {
		record.Outcome = "failed"
		record.FinishedAt = b.now().UTC()
		record.Error = truncateCodingAgentText(lastErr)
	})
}

func (b *CodingAgentBroker) update(id string, mutate func(*CodingAgentRecord)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i := range b.records {
		if b.records[i].ID != id {
			continue
		}
		mutate(&b.records[i])
		if err := b.persistLocked(); err != nil {
			// The admission receipt already exists. Keep the in-memory outcome
			// authoritative and let the next status read expose the failure.
			return
		}
		return
	}
}

func (b *CodingAgentBroker) Records() []CodingAgentRecord {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]CodingAgentRecord(nil), b.records...)
}

func (b *CodingAgentBroker) persistLocked() error {
	if strings.TrimSpace(b.path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(b.path), codingAgentDirectoryMode); err != nil {
		return err
	}
	data, err := json.MarshalIndent(b.records, "", "  ")
	if err != nil {
		return err
	}
	tmp := b.path + ".tmp"
	if err := os.WriteFile(tmp, data, codingAgentRecordMode); err != nil {
		return err
	}
	return os.Rename(tmp, b.path)
}

func truncateCodingAgentText(err error) string {
	if err == nil {
		return ""
	}
	value := strings.Join(strings.Fields(err.Error()), " ")
	if len(value) > 512 {
		return value[:512] + "..."
	}
	return value
}

func truncateCodingAgentString(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > 512 {
		return value[:512] + "..."
	}
	return value
}

// DefaultCodingAgentRecordPath returns the separate durable ledger for coding
// agent investigations. The existing lifecycle broker retains its own records
// and budget semantics for compatibility.
func DefaultCodingAgentRecordPath() (string, error) {
	path, err := DefaultRecordPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(path), "coding-agent-records.json"), nil
}

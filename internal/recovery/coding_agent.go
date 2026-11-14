package recovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/config"
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

// ErrCodingAgentUnavailable proves the runner was not started. Only this
// classification permits trying another runner in the same investigation.
var ErrCodingAgentUnavailable = errors.New("coding-agent runner unavailable before launch")

// CodingAgentRequest is the non-secret task envelope passed to a recovery
// runner. Credentials remain in the runner's configured stores and never enter
// this durable ledger.
type CodingAgentRequest struct {
	Scenario   string `json:"scenario"`
	Reason     string `json:"reason"`
	Requester  string `json:"requester"`
	WorkingDir string `json:"working_dir"`
	Claim      string `json:"claim"`
}

// CodingAgentRecord is the durable admission and outcome receipt for one
// investigation attempt.
type CodingAgentRecord struct {
	ID                   string     `json:"id"`
	Scenario             string     `json:"scenario"`
	Reason               string     `json:"reason"`
	Requester            string     `json:"requester"`
	Runner               string     `json:"runner,omitempty"`
	Outcome              string     `json:"outcome"`
	Error                string     `json:"error,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	FinishedAt           time.Time  `json:"finished_at,omitempty"`
	DeadlineAt           time.Time  `json:"deadline_at,omitempty"`
	ReconciledAt         *time.Time `json:"reconciled_at,omitempty"`
	ReconciliationReason string     `json:"reconciliation_reason,omitempty"`
	ObservationEndedAt   *time.Time `json:"observation_ended_at,omitempty"`
}

// CodingAgentRunner launches one named coding-agent runner. The implementation
// in main.go delegates to the existing `vrooli agent launch` command, which
// already prefers Agent Manager attachment and falls back to the native runner.
// Returned errors must be safe for a durable/public diagnostic: no raw child
// output, prompt argv or resolved credential values.
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
	native  *codingAgentNativeOwner
}

// NewCodingAgentBroker opens the shared ledger. A nil runner permits read-only
// inspection; Submit still requires an execution callback.
func NewCodingAgentBroker(path string, run CodingAgentRunner) (*CodingAgentBroker, error) {
	b := &CodingAgentBroker{
		path:    path,
		window:  CodingAgentRecoveryWindow,
		timeout: CodingAgentRecoveryTimeout,
		now:     time.Now,
		runners: append([]string(nil), DefaultCodingAgentRunners...),
		run:     run,
	}
	_, err := b.Snapshot()
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (b *CodingAgentBroker) loadLocked() error {
	if strings.TrimSpace(b.path) == "" {
		return nil
	}
	records, err := readCodingAgentRecords(b.path)
	if errors.Is(err, os.ErrNotExist) {
		if b.path != "" {
			retained, readErr := boundedCodingInputEntries(filepath.Join(filepath.Dir(b.path), "coding-agent-executions"), codingAgentMaxRecords)
			if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
				return fmt.Errorf("native recovery inventory is unreadable while admission ledger is absent")
			}
			if len(retained) > 0 {
				return fmt.Errorf("coding-agent recovery admission ledger is absent while retained native inputs exist")
			}
		}
		b.records = nil
		return nil
	}
	if err != nil {
		return fmt.Errorf("read coding-agent recovery records: %w", err)
	}
	b.records = records
	return nil
}

// lockLedger serializes read-modify-write transactions across brokers and
// processes. The in-process mutex alone cannot protect the durable budget.
func (b *CodingAgentBroker) lockLedger(ctx context.Context) (func(), error) {
	if strings.TrimSpace(b.path) == "" {
		return func() {}, nil
	}
	if err := os.MkdirAll(filepath.Dir(b.path), codingAgentDirectoryMode); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return platform.AcquireFileLockContext(ctx, b.path+".lock")
}

// Submit admits at most one investigation for a scenario in the rolling
// window. A duplicate returns the existing receipt and does not start another
// process, which makes repeated health ticks and API restarts idempotent.
func (b *CodingAgentBroker) Submit(req CodingAgentRequest) (CodingAgentRecord, error) {
	return b.SubmitContext(context.Background(), req)
}

func (b *CodingAgentBroker) SubmitContext(ctx context.Context, req CodingAgentRequest) (CodingAgentRecord, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return CodingAgentRecord{}, err
	}
	if b.run == nil && b.native == nil {
		return CodingAgentRecord{}, errors.New("coding-agent recovery runner is required")
	}
	req.Scenario = strings.TrimSpace(req.Scenario)
	req.Requester = strings.TrimSpace(req.Requester)
	if !codingScenarioName.MatchString(req.Scenario) || req.Requester == "" || len(req.Requester) > 512 || !utf8.ValidString(req.Requester) || strings.ContainsRune(req.Requester, '\x00') {
		return CodingAgentRecord{}, errors.New("coding-agent recovery scenario and requester are required")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	unlock, err := b.lockLedger(ctx)
	if err != nil {
		return CodingAgentRecord{}, fmt.Errorf("lock coding-agent recovery admission: %w", err)
	}
	defer unlock()
	if err := b.loadLocked(); err != nil {
		return CodingAgentRecord{}, err
	}
	if err := ctx.Err(); err != nil {
		return CodingAgentRecord{}, err
	}
	now := b.now().UTC()
	cutoff := now.Add(-b.window)
	for i := len(b.records) - 1; i >= 0; i-- {
		record := b.records[i]
		// A timeout or API restart does not prove an agent or its descendants
		// stopped. Never overlap unresolved work merely because an hour passed.
		pending := record.Outcome == "scheduled" || record.Outcome == "running" || record.Outcome == "unconfirmed"
		if record.Scenario == req.Scenario && (pending || record.CreatedAt.After(cutoff)) {
			return record, nil
		}
	}
	record := CodingAgentRecord{
		ID:         uuid.NewString(),
		Scenario:   req.Scenario,
		Reason:     truncateCodingAgentString(req.Reason),
		Requester:  req.Requester,
		Outcome:    "scheduled",
		CreatedAt:  now,
		DeadlineAt: now.Add(b.timeout),
	}
	var execution CodingAgentExecution
	if b.native != nil {
		// Establish an empty ledger before preparing the first sidecar. Later
		// ledger absence with retained inputs is loss, never a fresh budget.
		if _, err := os.Lstat(b.path); errors.Is(err, os.ErrNotExist) {
			if err := b.persistLocked(); err != nil {
				return CodingAgentRecord{}, err
			}
		} else if err != nil {
			return CodingAgentRecord{}, err
		}
		execution, err = b.prepareNative(ctx, record, req)
		if err != nil {
			return CodingAgentRecord{}, err
		}
	}
	b.records = append(b.records, record)
	if err := b.persistLocked(); err != nil {
		b.records = b.records[:len(b.records)-1]
		return CodingAgentRecord{}, fmt.Errorf("persist coding-agent recovery admission: %w", err)
	}
	if b.native != nil {
		// Persisted native intent is once-only. Even an ambiguous dispatch
		// returns this admitted identity; no goroutine owns the repair lifetime.
		if err := b.dispatchNative(ctx, &execution); err != nil {
			return record, fmt.Errorf("coding-agent recovery %s admitted; native dispatch requires inspection: %w", record.ID, err)
		}
	} else {
		go b.execute(record.ID, req)
	}
	return record, nil
}

func (b *CodingAgentBroker) execute(id string, req CodingAgentRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()
	b.executeContext(ctx, id, req)
}

func (b *CodingAgentBroker) executeContext(ctx context.Context, id string, req CodingAgentRequest) {
	if err := b.update(id, func(record *CodingAgentRecord) {
		record.Outcome = "running"
	}); err != nil {
		log.Printf("coding-agent recovery %s not launched: %v", id, err)
		return
	}
	var lastErr error
	for _, runner := range b.runners {
		if err := b.update(id, func(record *CodingAgentRecord) { record.Runner = runner }); err != nil {
			log.Printf("coding-agent recovery %s runner not launched: %v", id, err)
			return
		}
		if err := b.run(ctx, runner, req); err == nil {
			if ctx.Err() != nil {
				lastErr = ctx.Err()
				break
			}
			if err := b.update(id, func(record *CodingAgentRecord) {
				record.Runner = runner
				record.Outcome = "succeeded"
				record.FinishedAt = b.now().UTC()
			}); err != nil {
				log.Printf("coding-agent recovery %s completion not persisted: %v", id, err)
			}
			return
		} else {
			lastErr = err
		}
		if ctx.Err() != nil || !errors.Is(lastErr, ErrCodingAgentUnavailable) {
			break
		}
	}
	if err := b.update(id, func(record *CodingAgentRecord) {
		record.Outcome = "failed"
		if ctx.Err() != nil || (lastErr != nil && !errors.Is(lastErr, ErrCodingAgentUnavailable)) {
			record.Outcome = "unconfirmed"
		}
		if ctx.Err() != nil {
			lastErr = ctx.Err()
		}
		observedAt := b.now().UTC()
		if record.Outcome == "unconfirmed" {
			record.ObservationEndedAt = &observedAt
			record.FinishedAt = time.Time{}
		} else {
			record.FinishedAt = observedAt
		}
		record.Error = truncateCodingAgentText(lastErr)
	}); err != nil {
		log.Printf("coding-agent recovery %s completion not persisted: %v", id, err)
	}
}

func (b *CodingAgentBroker) update(id string, mutate func(*CodingAgentRecord)) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	unlock, err := b.lockLedger(context.Background())
	if err != nil {
		return err
	}
	defer unlock()
	if err := b.loadLocked(); err != nil {
		return err
	}
	for i := range b.records {
		if b.records[i].ID != id {
			continue
		}
		before := b.records[i]
		mutate(&b.records[i])
		if err := b.persistLocked(); err != nil {
			b.records[i] = before
			return err
		}
		return nil
	}
	return fmt.Errorf("coding-agent recovery receipt %s is missing", id)
}

// Snapshot reads the current durable ledger, including updates made by another
// process. Read failures are explicit; a cached success is not a durable receipt.
func (b *CodingAgentBroker) Snapshot() ([]CodingAgentRecord, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	unlock, err := b.lockLedger(context.Background())
	if err != nil {
		return nil, err
	}
	defer unlock()
	if err := b.loadLocked(); err != nil {
		return nil, err
	}
	return append([]CodingAgentRecord(nil), b.records...), nil
}

func (b *CodingAgentBroker) persistLocked() error {
	if strings.TrimSpace(b.path) == "" {
		return nil
	}
	if len(b.records) > codingAgentMaxRecords {
		return fmt.Errorf("coding-agent ledger record capacity exhausted")
	}
	data, err := json.MarshalIndent(b.records, "", "  ")
	if err != nil {
		return err
	}
	if len(data) > codingAgentMaxLedgerBytes {
		return fmt.Errorf("coding-agent ledger byte capacity exhausted")
	}
	return config.WriteOwnedFileDurable(b.path, data, codingAgentRecordMode)
}

func truncateCodingAgentText(err error) string {
	if err == nil {
		return ""
	}
	return truncateCodingAgentString(err.Error())
}

func truncateCodingAgentString(value string) string {
	value = strings.Join(strings.Fields(strings.ToValidUTF8(value, "\uFFFD")), " ")
	if len(value) > 512 {
		end := 512
		for !utf8.ValidString(value[:end]) {
			end--
		}
		return value[:end] + "..."
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

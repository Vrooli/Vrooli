// Package recovery owns the control-plane seam for bounded agent-spawn
// recovery. Scenarios may request recovery, but they do not own the ladder,
// budget, child environment, or durable record.
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
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/envkit-go"
)

const (
	DefaultBudget                     = 3
	DefaultWindow                     = time.Hour
	RecoveryTierOne                   = "governed-attach"
	RecoveryTierTwo                   = "fresh-agent-run"
	RecoveryTierThree                 = "native-runner"
	RecoveryTierFour                  = "operator-escalation"
	recoveryDirectoryMode os.FileMode = 0o755
	recoveryRecordMode    os.FileMode = 0o600
)

type Request struct {
	Scenario  string `json:"scenario"`
	Reason    string `json:"reason"`
	Requester string `json:"requester"`
}

type Record struct {
	ID              string    `json:"id"`
	Scenario        string    `json:"scenario"`
	Reason          string    `json:"reason"`
	Requester       string    `json:"requester"`
	TierReached     string    `json:"tier_reached"`
	BudgetRemaining int       `json:"budget_remaining"`
	Outcome         string    `json:"outcome"`
	CreatedAt       time.Time `json:"created_at"`
}

type TierRunner func(context.Context, string, string, []string) error

type Broker struct {
	mu      sync.Mutex
	path    string
	budget  int
	window  time.Duration
	now     func() time.Time
	records []Record
	lastUse map[string][]time.Time
	runTier TierRunner
}

func New(path string, run TierRunner) (*Broker, error) {
	b := &Broker{path: path, budget: DefaultBudget, window: DefaultWindow, now: time.Now, lastUse: make(map[string][]time.Time), runTier: run}
	if strings.TrimSpace(path) != "" {
		if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
			if err := json.Unmarshal(data, &b.records); err != nil {
				return nil, fmt.Errorf("decode recovery records: %w", err)
			}
			cutoff := b.now().Add(-b.window)
			for _, record := range b.records {
				if record.CreatedAt.After(cutoff) && record.Outcome != "budget-exhausted" {
					b.lastUse[record.Scenario] = append(b.lastUse[record.Scenario], record.CreatedAt)
				}
			}
		} else if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("read recovery records: %w", err)
		}
	}
	return b, nil
}

func (b *Broker) Recover(ctx context.Context, req Request, parent envkit.Env) (Record, error) {
	if b.runTier == nil {
		return Record{}, errors.New("recovery tier runner is required")
	}
	if req.Scenario == "" || req.Requester == "" {
		return Record{}, errors.New("recovery scenario and requester are required")
	}
	now := b.now().UTC()
	b.mu.Lock()
	uses := b.lastUse[req.Scenario]
	cutoff := now.Add(-b.window)
	kept := uses[:0]
	for _, used := range uses {
		if used.After(cutoff) {
			kept = append(kept, used)
		}
	}
	if len(kept) >= b.budget {
		record := b.newRecord(req, RecoveryTierFour, b.budget-len(kept), "budget-exhausted", now)
		b.records = append(b.records, record)
		if err := b.persistLocked(); err != nil {
			b.mu.Unlock()
			return record, fmt.Errorf("persist recovery exhaustion record: %w", err)
		}
		b.mu.Unlock()
		return record, fmt.Errorf("recovery budget exhausted for %s", req.Scenario)
	}
	kept = append(kept, now)
	b.lastUse[req.Scenario] = kept
	b.mu.Unlock()

	child := []string(envkit.WithOverlay(parent, envkit.ForeignScenario, nil))
	tiers := []string{RecoveryTierOne, RecoveryTierTwo, RecoveryTierThree, RecoveryTierFour}
	var lastErr error
	for _, tier := range tiers {
		if tier == RecoveryTierFour {
			break
		}
		if err := b.runTier(ctx, tier, req.Scenario, child); err == nil {
			record := b.newRecord(req, tier, b.budget-len(kept), "succeeded", now)
			return b.append(record)
		} else {
			lastErr = err
		}
	}
	record := b.newRecord(req, RecoveryTierFour, b.budget-len(kept), "failed: "+lastErr.Error(), now)
	return b.append(record)
}

// DefaultRecordPath resolves the durable recovery ledger through api-core's
// storage authority rather than constructing a home-directory path here.
func DefaultRecordPath() (string, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		return "", err
	}
	return resolver.Path(storage.Options{ScenarioID: "agent-recovery"}, storage.ClassState, "records.json")
}

func (b *Broker) Records() []Record {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]Record(nil), b.records...)
}

func (b *Broker) newRecord(req Request, tier string, remaining int, outcome string, now time.Time) Record {
	return Record{ID: uuid.NewString(), Scenario: req.Scenario, Reason: req.Reason, Requester: req.Requester, TierReached: tier, BudgetRemaining: remaining, Outcome: outcome, CreatedAt: now}
}

func (b *Broker) append(record Record) (Record, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.records = append(b.records, record)
	if err := b.persistLocked(); err != nil {
		return record, err
	}
	if record.Outcome != "succeeded" {
		return record, errors.New(record.Outcome)
	}
	return record, nil
}

func (b *Broker) persistLocked() error {
	if strings.TrimSpace(b.path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(b.path), recoveryDirectoryMode); err != nil {
		return err
	}
	data, err := json.MarshalIndent(b.records, "", "  ")
	if err != nil {
		return err
	}
	tmp := b.path + ".tmp"
	if err := os.WriteFile(tmp, data, recoveryRecordMode); err != nil {
		return err
	}
	return os.Rename(tmp, b.path)
}

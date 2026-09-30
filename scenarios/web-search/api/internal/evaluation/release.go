package evaluation

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

type MethodRevision struct {
	ID          string
	ProgramHash string
	ConfigHash  string
}

type EvaluationReceipt struct {
	ID            string
	CandidateHash string
	ReportHash    string
	Accepted      bool
}

type Release struct {
	Revision       MethodRevision
	EvaluationID   string
	EvaluationHash string
	RollbackOf     string
}

// MethodRegistry is an owner-local release selector. Promotion and rollback
// are compare-and-swap operations so a stale evaluator cannot overwrite a
// newer selection.
type MethodRegistry struct {
	mu        sync.Mutex
	current   *Release
	history   []Release
	suspended map[string]string
	store     *releaseStore
}

type dbExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type releaseStore struct{ db dbExecutor }

type persistedReleaseState struct {
	Current   *Release
	History   []Release
	Suspended map[string]string
}

// NewSQLiteMethodRegistry restores release state from the scenario-owned
// database. The zero-value registry remains valid for isolated tests.
func NewSQLiteMethodRegistry(db dbExecutor) (*MethodRegistry, error) {
	r := &MethodRegistry{store: &releaseStore{db: db}, suspended: map[string]string{}}
	var raw string
	err := db.QueryRowContext(context.Background(), `SELECT state_json FROM method_release_state WHERE id = 1`).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return r, nil
	}
	if err != nil {
		return nil, fmt.Errorf("restore method release state: %w", err)
	}
	var state persistedReleaseState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return nil, fmt.Errorf("decode method release state: %w", err)
	}
	r.current, r.history, r.suspended = state.Current, state.History, state.Suspended
	if r.suspended == nil {
		r.suspended = map[string]string{}
	}
	return r, nil
}

func (r *MethodRegistry) persistLocked() error {
	if r.store == nil {
		return nil
	}
	b, err := json.Marshal(persistedReleaseState{Current: r.current, History: r.history, Suspended: r.suspended})
	if err != nil {
		return err
	}
	_, err = r.store.db.ExecContext(context.Background(), `INSERT INTO method_release_state (id, state_json) VALUES (1, ?) ON CONFLICT(id) DO UPDATE SET state_json = excluded.state_json`, string(b))
	return err
}

func (r *MethodRegistry) Current() (Release, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.current == nil {
		return Release{}, false
	}
	return *r.current, true
}

func RevisionHash(revision MethodRevision) string {
	b, _ := json.Marshal(revision)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func ReportHash(report Report) string {
	b, _ := json.Marshal(report)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (r *MethodRegistry) Promote(expectedCurrentHash string, revision MethodRevision, receipt EvaluationReceipt, report Report, grant string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if grant == "" {
		return fmt.Errorf("promotion grant is required")
	}
	if !report.Accepted || !receipt.Accepted {
		return fmt.Errorf("method evaluation did not pass promotion gates")
	}
	if receipt.CandidateHash != RevisionHash(revision) || receipt.ReportHash != ReportHash(report) {
		return fmt.Errorf("evaluation receipt does not match candidate report")
	}
	if reason, suspended := r.suspended[RevisionHash(revision)]; suspended {
		return fmt.Errorf("method revision is suspended: %s", reason)
	}
	if r.current != nil && RevisionHash(r.current.Revision) != expectedCurrentHash {
		return fmt.Errorf("current method changed")
	}
	release := Release{Revision: revision, EvaluationID: receipt.ID, EvaluationHash: receipt.ReportHash}
	if r.current != nil {
		release.RollbackOf = RevisionHash(r.current.Revision)
	}
	if r.current != nil {
		r.history = append(r.history, *r.current)
	}
	r.current = &release
	if err := r.persistLocked(); err != nil {
		return fmt.Errorf("persist method promotion: %w", err)
	}
	return nil
}

// Suspend invalidates a revision when supporting evidence is corrected. Its
// evaluation and release history remain readable for audit.
func (r *MethodRegistry) Suspend(revisionHash, reason string) error {
	if revisionHash == "" || reason == "" {
		return fmt.Errorf("revision hash and suspension reason are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.suspended == nil {
		r.suspended = map[string]string{}
	}
	r.suspended[revisionHash] = reason
	if err := r.persistLocked(); err != nil {
		return fmt.Errorf("persist method suspension: %w", err)
	}
	return nil
}

func (r *MethodRegistry) Rollback(expectedCurrentHash, targetHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.current == nil || RevisionHash(r.current.Revision) != expectedCurrentHash {
		return fmt.Errorf("current method changed")
	}
	for i := len(r.history) - 1; i >= 0; i-- {
		if RevisionHash(r.history[i].Revision) != targetHash {
			continue
		}
		target := r.history[i]
		target.RollbackOf = expectedCurrentHash
		r.history = append(r.history, *r.current)
		r.current = &target
		if err := r.persistLocked(); err != nil {
			return fmt.Errorf("persist method rollback: %w", err)
		}
		return nil
	}
	return fmt.Errorf("rollback target is unavailable")
}

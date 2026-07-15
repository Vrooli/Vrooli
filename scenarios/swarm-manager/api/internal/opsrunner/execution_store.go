package opsrunner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/agentops"
)

// ExecutionSnapshot is the immutable, self-contained record of one operation
// execution. It pins BOTH the provenance digests and the exact bytes those
// digests summarize (the compiled mode bundle, the prompt-catalog projection,
// and the validated caller-input snapshot), so a historical execution stays
// fully reproducible after the live mode/prompt/policy sources change — nothing
// about a past run is read back from the mutable catalog. A digest recomputed
// from the persisted bytes that no longer matches the pinned provenance digest
// is a typed tamper/corruption signal (ErrDigestMismatch), never a silent
// divergence.
type ExecutionSnapshot struct {
	Provenance      agentops.ExecutionProvenance `json:"provenance"`
	CompiledMode    json.RawMessage              `json:"compiled_mode"`
	PromptCatalog   json.RawMessage              `json:"prompt_catalog"`
	EffectiveInputs json.RawMessage              `json:"effective_inputs"`
	Outcome         string                       `json:"outcome,omitempty"`
	Result          json.RawMessage              `json:"result,omitempty"`
	// PolicyID pins the transition policy resolved for this execution so the
	// async CommitResult path can fire the same policy the Invoke resolved,
	// without re-resolving the binding. Empty when the operation had no policy.
	PolicyID   string `json:"policy_id,omitempty"`
	RecordedAt string `json:"recorded_at"`
}

// ExecutionStore persists execution snapshots beside the domain entity that owns
// the workflow ("<agentops dir>/executions/<execution-id>.json"), so the record
// is domain-owned rather than living in an unbounded central store.
type ExecutionStore struct {
	loc DomainLocator
}

// NewExecutionStore constructs a store over a domain locator.
func NewExecutionStore(loc DomainLocator) *ExecutionStore { return &ExecutionStore{loc: loc} }

const executionsSubdir = "executions"

func (s *ExecutionStore) path(kind agentops.TargetKind, id, executionID string) (string, error) {
	if err := validateDomainToken(executionID); err != nil {
		return "", err
	}
	if strings.ContainsAny(executionID, "/\\") {
		return "", fmt.Errorf("execution id %q must not contain path separators", executionID)
	}
	dir, err := s.loc.AgentOpsDir(kind, id)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, executionsSubdir, executionID+".json"), nil
}

// SaveExecution writes a snapshot under an explicit execution id. It validates
// the provenance one final time so a corrupt record can never be persisted.
func (s *ExecutionStore) SaveExecution(kind agentops.TargetKind, id, executionID string, snap ExecutionSnapshot) error {
	raw, err := json.Marshal(snap.Provenance)
	if err != nil {
		return err
	}
	if err := agentops.ValidateProvenance(raw); err != nil {
		return fmt.Errorf("refusing to persist invalid provenance: %w", err)
	}
	if snap.RecordedAt == "" {
		snap.RecordedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	path, err := s.path(kind, id, executionID)
	if err != nil {
		return err
	}
	body, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(path, body)
}

// Load returns the persisted snapshot for an execution.
func (s *ExecutionStore) Load(kind agentops.TargetKind, id, executionID string) (ExecutionSnapshot, bool, error) {
	path, err := s.path(kind, id, executionID)
	if err != nil {
		return ExecutionSnapshot{}, false, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ExecutionSnapshot{}, false, nil
		}
		return ExecutionSnapshot{}, false, err
	}
	var snap ExecutionSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return ExecutionSnapshot{}, false, err
	}
	return snap, true, nil
}

// StoredExecution pairs a persisted snapshot with its execution id (the file
// basename), for history listings.
type StoredExecution struct {
	ID       string
	Snapshot ExecutionSnapshot
}

// List returns every persisted execution snapshot for a target, ordered
// newest-first by RecordedAt (ties and unparsable timestamps fall back to id
// order for determinism). A missing executions directory is an empty history,
// not an error; a corrupt snapshot file is a fail-closed error naming it.
func (s *ExecutionStore) List(kind agentops.TargetKind, id string) ([]StoredExecution, error) {
	dir, err := s.loc.AgentOpsDir(kind, id)
	if err != nil {
		return nil, err
	}
	execDir := filepath.Join(dir, executionsSubdir)
	entries, err := os.ReadDir(execDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []StoredExecution
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		path := filepath.Join(execDir, e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var snap ExecutionSnapshot
		if err := json.Unmarshal(raw, &snap); err != nil {
			return nil, fmt.Errorf("corrupt execution snapshot %s: %w", path, err)
		}
		out = append(out, StoredExecution{ID: strings.TrimSuffix(e.Name(), ".json"), Snapshot: snap})
	}
	sort.Slice(out, func(i, j int) bool {
		ti, ei := time.Parse(time.RFC3339Nano, out[i].Snapshot.RecordedAt)
		tj, ej := time.Parse(time.RFC3339Nano, out[j].Snapshot.RecordedAt)
		switch {
		case ei == nil && ej == nil && !ti.Equal(tj):
			return ti.After(tj)
		case ei == nil && ej != nil:
			return true
		case ei != nil && ej == nil:
			return false
		default:
			return out[i].ID > out[j].ID
		}
	})
	return out, nil
}

// Reproduce loads a historical execution and re-verifies its reproducibility:
// it recomputes the compiled-mode, prompt-catalog, and caller-input digests from
// the persisted bytes and confirms they still equal the pinned provenance
// digests. A mismatch — a tampered snapshot or a corrupted record — is
// ErrDigestMismatch. Because the snapshot is self-contained, this holds even
// after the live mode/prompt/policy definitions have changed.
func (s *ExecutionStore) Reproduce(kind agentops.TargetKind, id, executionID string) (ExecutionSnapshot, error) {
	snap, found, err := s.Load(kind, id, executionID)
	if err != nil {
		return ExecutionSnapshot{}, err
	}
	if !found {
		return ExecutionSnapshot{}, fmt.Errorf("execution %q not found for %s/%s", executionID, kind, id)
	}
	if err := verifySnapshotDigests(snap); err != nil {
		return ExecutionSnapshot{}, err
	}
	return snap, nil
}

// verifySnapshotDigests recomputes and compares each pinned digest.
func verifySnapshotDigests(snap ExecutionSnapshot) error {
	checks := []struct {
		name   string
		bytes  json.RawMessage
		pinned string
	}{
		{"compiled_mode", snap.CompiledMode, snap.Provenance.CompiledModeDigest},
		{"prompt_catalog", snap.PromptCatalog, snap.Provenance.PromptCatalogDigest},
		{"caller_input", snap.EffectiveInputs, snap.Provenance.CallerInputDigest},
	}
	for _, c := range checks {
		got, err := agentops.CanonicalDigest(c.bytes)
		if err != nil {
			return fmt.Errorf("recompute %s digest: %w", c.name, err)
		}
		if got != c.pinned {
			return fmt.Errorf("%w: %s digest is %s, provenance pinned %s", ErrDigestMismatch, c.name, got, c.pinned)
		}
	}
	return nil
}

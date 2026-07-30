// Package transitionrun provides the durable, subject-neutral correlation
// journal shared by declared transition consumers.
package transitionrun

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"swarm-manager/internal/storage"
)

const (
	ApplyStateClaimed   = "claimed"
	ApplyStateComplete  = "complete"
	CompletionSucceeded = "succeeded"
)

// Correlation is the durable hand-off between an Agent Manager workflow and a
// subject-specific apply function. It intentionally contains no domain types.
type Correlation struct {
	TransitionKey        string          `json:"transition_key"`
	SubjectKind          string          `json:"subject_kind"`
	SubjectRef           string          `json:"subject_ref"`
	ExecutionID          string          `json:"execution_id"`
	WorkflowKey          string          `json:"workflow_key"`
	DefinitionDigest     string          `json:"definition_digest"`
	EntityVersion        string          `json:"entity_version"`
	FrontierDigest       string          `json:"frontier_digest"`
	ApplyState           string          `json:"apply_state"`
	Outcome              string          `json:"outcome"`
	TerminalCode         string          `json:"terminal_code,omitempty"`
	BudgetName           string          `json:"budget_name,omitempty"`
	Result               json.RawMessage `json:"result,omitempty"`
	Attempts             []Attempt       `json:"attempts,omitempty"`
	ApprovalActor        string          `json:"approval_actor,omitempty"`
	ApprovalTime         string          `json:"approval_time,omitempty"`
	ApplyAttemptCount    int             `json:"apply_attempt_count,omitempty"`
	LastApplyAttemptTime string          `json:"last_apply_attempt_time,omitempty"`
	LastApplyError       string          `json:"last_apply_error,omitempty"`
	AppliedTime          string          `json:"applied_time,omitempty"`
	DeclaredOutcomes     []string        `json:"declared_outcomes"`
}

// Attempt preserves enough workflow provenance for an operator to correlate a
// terminal result back to its source invocation without importing a domain.
type Attempt struct {
	NodeID          string `json:"node_id,omitempty"`
	Ordinal         int32  `json:"ordinal,omitempty"`
	Strategy        string `json:"strategy,omitempty"`
	RunID           string `json:"run_id,omitempty"`
	ConversationID  string `json:"conversation_id,omitempty"`
	SourceAttemptID string `json:"source_attempt_id,omitempty"`
	ProfileIdentity string `json:"profile_identity,omitempty"`
}

// Completion is the immutable workflow terminal snapshot used by CanApply.
type Completion struct {
	ExecutionID      string
	DefinitionDigest string
	EntityVersion    string
	FrontierDigest   string
	Status           string
	Outcome          string
}

type AlreadyCompleteError struct{ ExecutionID string }

func (e *AlreadyCompleteError) Error() string {
	return fmt.Sprintf("transition %q is already applied", e.ExecutionID)
}

type DigestMismatchError struct{ Expected, Actual string }

func (e *DigestMismatchError) Error() string {
	return "workflow definition digest does not match correlation"
}

type StatusNotSucceededError struct{ Status string }

func (e *StatusNotSucceededError) Error() string {
	return fmt.Sprintf("workflow status %q is not succeeded", e.Status)
}

type EntityVersionChangedError struct{ Expected, Actual string }

func (e *EntityVersionChangedError) Error() string {
	return "subject entity version changed while workflow was running"
}

type OutcomeNotDeclaredError struct{ Outcome string }

func (e *OutcomeNotDeclaredError) Error() string {
	return fmt.Sprintf("workflow outcome %q is not declared by transition", e.Outcome)
}

type FrontierChangedError struct{ Expected, Actual string }

func (e *FrontierChangedError) Error() string {
	return "subject frontier changed while workflow was running"
}

// CanApply checks that a terminal workflow snapshot still belongs to, and is
// permitted to mutate, this correlation. Each rejection has a typed error so
// callers can map it without parsing strings.
func CanApply(c Correlation, completion Completion) error {
	if c.ApplyState == ApplyStateComplete {
		return &AlreadyCompleteError{ExecutionID: c.ExecutionID}
	}
	if c.ExecutionID != completion.ExecutionID || c.DefinitionDigest != completion.DefinitionDigest {
		return &DigestMismatchError{Expected: c.DefinitionDigest, Actual: completion.DefinitionDigest}
	}
	if completion.Status != CompletionSucceeded {
		return &StatusNotSucceededError{Status: completion.Status}
	}
	if c.EntityVersion != completion.EntityVersion {
		return &EntityVersionChangedError{Expected: c.EntityVersion, Actual: completion.EntityVersion}
	}
	if c.FrontierDigest != completion.FrontierDigest {
		return &FrontierChangedError{Expected: c.FrontierDigest, Actual: completion.FrontierDigest}
	}
	for _, allowed := range c.DeclaredOutcomes {
		if allowed == completion.Outcome {
			return nil
		}
	}
	return &OutcomeNotDeclaredError{Outcome: completion.Outcome}
}

// Store persists correlations independently of their subject domain.
type Store interface {
	Put(Correlation) error
	Get(string) (Correlation, error)
	FindBySubject(transitionKey, subjectRef string) (Correlation, error)
	ListUnapplied() ([]Correlation, error)
	Delete(string) error
}

// FindBySubject returns the durable correlation for one declared transition
// subject. It scans both applied and unapplied entries because a projection
// must be able to explain historical rounds after application.
func (s *FileStore) FindBySubject(transitionKey, subjectRef string) (Correlation, error) {
	entries, err := os.ReadDir(s.root)
	if errors.Is(err, os.ErrNotExist) {
		return Correlation{}, os.ErrNotExist
	}
	if err != nil {
		return Correlation{}, err
	}
	var matches []Correlation
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		correlation, err := s.Get(strings.TrimSuffix(entry.Name(), ".json"))
		if err != nil {
			return Correlation{}, err
		}
		if correlation.TransitionKey == transitionKey && correlation.SubjectRef == subjectRef {
			matches = append(matches, correlation)
		}
	}
	if len(matches) == 0 {
		return Correlation{}, os.ErrNotExist
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].ExecutionID > matches[j].ExecutionID })
	return matches[0], nil
}

// FileStore stores one JSON document per workflow execution under root.
type FileStore struct{ root string }

func NewFileStore(root string) *FileStore { return &FileStore{root: root} }
func (s *FileStore) path(id string) (string, error) {
	if strings.TrimSpace(id) == "" || filepath.Base(id) != id {
		return "", fmt.Errorf("invalid correlation execution id %q", id)
	}
	return filepath.Join(s.root, id+".json"), nil
}

func (s *FileStore) Put(c Correlation) error {
	p, err := s.path(c.ExecutionID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return fmt.Errorf("create transition correlation directory: %w", err)
	}
	return storage.WriteJSONAtomic(p, c)
}

func (s *FileStore) Get(id string) (Correlation, error) {
	p, err := s.path(id)
	if err != nil {
		return Correlation{}, err
	}
	var c Correlation
	exists, err := storage.ReadJSON(p, &c)
	if err != nil {
		return Correlation{}, err
	}
	if !exists {
		return Correlation{}, os.ErrNotExist
	}
	return c, nil
}

func (s *FileStore) ListUnapplied() ([]Correlation, error) {
	entries, err := os.ReadDir(s.root)
	if errors.Is(err, os.ErrNotExist) {
		return []Correlation{}, nil
	}
	if err != nil {
		return nil, err
	}
	correlations := make([]Correlation, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		c, err := s.Get(strings.TrimSuffix(entry.Name(), ".json"))
		if err != nil {
			return nil, err
		}
		if c.ApplyState != ApplyStateComplete {
			correlations = append(correlations, c)
		}
	}
	sort.Slice(correlations, func(i, j int) bool { return correlations[i].ExecutionID < correlations[j].ExecutionID })
	return correlations, nil
}

func (s *FileStore) Delete(id string) error {
	p, err := s.path(id)
	if err != nil {
		return err
	}
	err = os.Remove(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

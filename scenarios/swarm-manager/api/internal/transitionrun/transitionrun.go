// Package transitionrun provides the durable, subject-neutral correlation
// journal shared by declared transition consumers.
package transitionrun

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"swarm-manager/internal/storage"
	"swarm-manager/internal/workflowcontract"
)

const (
	ApplyStateClaimed   = "claimed"
	ApplyStateComplete  = "complete"
	CompletionSucceeded = "succeeded"
)

// Correlation is the durable hand-off between an Agent Manager workflow and a
// subject-specific apply function. It intentionally contains no domain types.
type Correlation struct {
	TransitionKey              string          `json:"transition_key"`
	SubjectKind                string          `json:"subject_kind"`
	SubjectRef                 string          `json:"subject_ref"`
	ExecutionID                string          `json:"execution_id"`
	WorkflowKey                string          `json:"workflow_key"`
	DefinitionDigest           string          `json:"definition_digest"`
	EntityVersion              string          `json:"entity_version"`
	FrontierDigest             string          `json:"frontier_digest"`
	ApplyState                 string          `json:"apply_state"`
	Outcome                    string          `json:"outcome"`
	TerminalCode               string          `json:"terminal_code,omitempty"`
	BudgetName                 string          `json:"budget_name,omitempty"`
	Result                     json.RawMessage `json:"result,omitempty"`
	Attempts                   []Attempt       `json:"attempts,omitempty"`
	ApprovalActor              string          `json:"approval_actor,omitempty"`
	ApprovalTime               string          `json:"approval_time,omitempty"`
	EntityVersionRebasedBy     string          `json:"entity_version_rebased_by,omitempty"`
	EntityVersionRebasedAt     string          `json:"entity_version_rebased_at,omitempty"`
	EntityVersionRebasedReason string          `json:"entity_version_rebased_reason,omitempty"`
	ApplyAttemptCount          int             `json:"apply_attempt_count,omitempty"`
	LastApplyAttemptTime       string          `json:"last_apply_attempt_time,omitempty"`
	LastApplyError             string          `json:"last_apply_error,omitempty"`
	AppliedTime                string          `json:"applied_time,omitempty"`
	DeclaredOutcomes           []string        `json:"declared_outcomes"`
}

// DispatchIntent records the original owner lookup before its start RPC. An
// absent lookup response cannot erase this possibly submitted reservation.
type DispatchIntent struct {
	State          DispatchState           `json:"state"`
	LastError      string                  `json:"last_error,omitempty"`
	Correlation    Correlation             `json:"correlation"`
	Owner          string                  `json:"owner"`
	IdempotencyKey string                  `json:"idempotency_key"`
	InputDigest    string                  `json:"input_digest"`
	ApprovalDigest string                  `json:"approval_digest,omitempty"`
	GrantDigest    string                  `json:"grant_digest,omitempty"`
	Grant          *workflowcontract.Grant `json:"grant,omitempty"`
	CreatedAt      string                  `json:"created_at"`
}

// DispatchState describes the durable reservation around an owner start RPC.
// Unknown means the RPC may have been accepted but its acknowledgement was
// lost; recovery may only join it to that original owner, never resubmit it.
type DispatchState string

const (
	DispatchSubmitting   DispatchState = "submitting"
	DispatchUnknown      DispatchState = "unknown"
	DispatchAcknowledged DispatchState = "acknowledged"
)

var ErrInvalidDispatchTransition = errors.New("invalid dispatch intent transition")

// Advance validates and applies one lifecycle edge. The same-state unknown
// edge is allowed because a later read-only recovery attempt can update its
// diagnostic without changing the reservation's meaning.
func (i *DispatchIntent) Advance(next DispatchState, lastError string) error {
	if i == nil {
		return fmt.Errorf("%w: nil dispatch intent", ErrInvalidDispatchTransition)
	}
	if !validDispatchTransition(i.State, next) {
		return fmt.Errorf("%w: %q -> %q", ErrInvalidDispatchTransition, i.State, next)
	}
	i.State, i.LastError = next, lastError
	return nil
}

func validDispatchTransition(from, to DispatchState) bool {
	switch from {
	case DispatchSubmitting:
		return to == DispatchUnknown || to == DispatchAcknowledged
	case DispatchUnknown:
		return to == DispatchUnknown || to == DispatchAcknowledged
	case DispatchAcknowledged:
		return false
	default:
		return false
	}
}

func knownDispatchState(state DispatchState) bool {
	switch state {
	case DispatchSubmitting, DispatchUnknown, DispatchAcknowledged:
		return true
	default:
		return false
	}
}

type DispatchStore interface {
	PutDispatch(DispatchIntent) error
	GetDispatch(transitionKey, subjectRef string) (DispatchIntent, error)
	ListUnresolvedDispatches() ([]DispatchIntent, error)
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
	if c.EntityVersion != completion.EntityVersion {
		return &EntityVersionChangedError{Expected: c.EntityVersion, Actual: completion.EntityVersion}
	}
	if c.FrontierDigest != completion.FrontierDigest {
		return &FrontierChangedError{Expected: c.FrontierDigest, Actual: completion.FrontierDigest}
	}
	// A declared terminal outcome applies through the typed apply. The succeeded
	// status carries the worker's structured outcome; blocked, abstained and
	// budget_exhausted map to their declared outcome by status. failed and
	// cancelled remain reconcile-owned and are rejected here.
	switch completion.Status {
	case CompletionSucceeded:
		for _, allowed := range c.DeclaredOutcomes {
			if allowed == completion.Outcome {
				return nil
			}
		}
		return &OutcomeNotDeclaredError{Outcome: completion.Outcome}
	case "blocked", "abstained", "budget_exhausted":
		mapped := declaredOutcomeForStatus(completion.Status)
		for _, allowed := range c.DeclaredOutcomes {
			if allowed == mapped {
				return nil
			}
		}
		return &OutcomeNotDeclaredError{Outcome: mapped}
	default:
		return &StatusNotSucceededError{Status: completion.Status}
	}
}

// declaredOutcomeForStatus maps a non-succeeded workflow terminal status to the
// transition outcome it declares.
func declaredOutcomeForStatus(status string) string {
	switch status {
	case "blocked":
		return "needs_attention"
	case "abstained":
		return "abstained"
	case "budget_exhausted":
		return "budget_exhausted"
	default:
		return ""
	}
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

func (s *FileStore) dispatchPath(transitionKey, subjectRef string) string {
	sum := sha256.Sum256([]byte(transitionKey + "\n" + subjectRef))
	return filepath.Join(s.root, "dispatch", hex.EncodeToString(sum[:])+".json")
}

func (s *FileStore) PutDispatch(intent DispatchIntent) error {
	if err := validateDispatchIntent(intent); err != nil {
		return err
	}
	path := s.dispatchPath(intent.Correlation.TransitionKey, intent.Correlation.SubjectRef)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var previous DispatchIntent
	exists, err := storage.ReadJSON(path, &previous)
	if err != nil {
		return err
	}
	if !exists {
		if intent.State != DispatchSubmitting {
			return fmt.Errorf("%w: first state must be %q", ErrInvalidDispatchTransition, DispatchSubmitting)
		}
	} else {
		if !sameDispatchReservation(previous, intent) {
			return errors.New("dispatch intent immutable identity changed")
		}
		if !validDispatchTransition(previous.State, intent.State) {
			return fmt.Errorf("%w: %q -> %q", ErrInvalidDispatchTransition, previous.State, intent.State)
		}
	}
	return storage.WriteJSONAtomic(path, intent)
}

func sameDispatchReservation(previous, next DispatchIntent) bool {
	previous.State, next.State = "", ""
	previous.LastError, next.LastError = "", ""
	// The owner reply binds execution identity and run provenance after the
	// reservation is written. Those fields are deliberately allowed to arrive
	// only with the acknowledged edge.
	previous.Correlation.ExecutionID, next.Correlation.ExecutionID = "", ""
	previous.Correlation.DefinitionDigest, next.Correlation.DefinitionDigest = "", ""
	previous.Correlation.Attempts, next.Correlation.Attempts = nil, nil
	return reflect.DeepEqual(previous, next)
}

func validateDispatchIntent(intent DispatchIntent) error {
	if intent.Correlation.TransitionKey == "" || intent.Correlation.SubjectRef == "" || intent.IdempotencyKey == "" || intent.InputDigest == "" || intent.Owner == "" || intent.CreatedAt == "" {
		return errors.New("dispatch intent requires immutable subject, owner, input and idempotency identities")
	}
	if !knownDispatchState(intent.State) {
		return fmt.Errorf("%w: unknown state %q", ErrInvalidDispatchTransition, intent.State)
	}
	if intent.State == DispatchAcknowledged && (intent.Correlation.ExecutionID == "" || intent.Correlation.DefinitionDigest == "") {
		return fmt.Errorf("%w: acknowledged dispatch must retain its original execution and definition", ErrInvalidDispatchTransition)
	}
	if intent.State != DispatchAcknowledged && (intent.Correlation.ExecutionID != "" || intent.Correlation.DefinitionDigest != "") {
		return fmt.Errorf("%w: unresolved dispatch cannot contain an acknowledged execution", ErrInvalidDispatchTransition)
	}
	return nil
}

func (s *FileStore) GetDispatch(transitionKey, subjectRef string) (DispatchIntent, error) {
	var intent DispatchIntent
	exists, err := storage.ReadJSON(s.dispatchPath(transitionKey, subjectRef), &intent)
	if err != nil {
		return intent, err
	}
	if !exists {
		return intent, os.ErrNotExist
	}
	if intent.Correlation.TransitionKey != transitionKey || intent.Correlation.SubjectRef != subjectRef || intent.IdempotencyKey == "" || intent.InputDigest == "" {
		return DispatchIntent{}, errors.New("stored dispatch intent identity is invalid")
	}
	if err := validateDispatchIntent(intent); err != nil {
		return DispatchIntent{}, err
	}
	return intent, nil
}

func (s *FileStore) ListUnresolvedDispatches() ([]DispatchIntent, error) {
	entries, err := os.ReadDir(filepath.Join(s.root, "dispatch"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var intents []DispatchIntent
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var intent DispatchIntent
		if _, err := storage.ReadJSON(filepath.Join(s.root, "dispatch", entry.Name()), &intent); err != nil {
			return nil, err
		}
		if err := validateDispatchIntent(intent); err != nil {
			return nil, err
		}
		if intent.Correlation.ExecutionID == "" {
			intents = append(intents, intent)
		}
	}
	return intents, nil
}
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

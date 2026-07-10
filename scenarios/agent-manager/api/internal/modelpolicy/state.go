package modelpolicy

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/domain"
)

const DiagnosticCodeCatalogInvalid = "MODEL_POLICY_CATALOG_INVALID"

// Requirement explains whether agent-manager may operate without an active
// catalog revision. The reason is operator-facing and should name the consumer
// that makes the catalog mandatory.
type Requirement struct {
	Required bool   `json:"required"`
	Reason   string `json:"reason,omitempty"`
}

// Diagnostic preserves an actionable catalog load or reload failure.
type Diagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Cause   string `json:"cause,omitempty"`
}

// ReloadAttempt describes the most recent validation-and-activation attempt.
type ReloadAttempt struct {
	AttemptedAt time.Time   `json:"attemptedAt"`
	Succeeded   bool        `json:"succeeded"`
	Digest      string      `json:"digest,omitempty"`
	Diagnostic  *Diagnostic `json:"diagnostic,omitempty"`
}

// Status is an immutable operator view of catalog activation state.
type Status struct {
	Path              string         `json:"path"`
	Requirement       Requirement    `json:"requirement"`
	Ready             bool           `json:"ready"`
	ActiveDigest      string         `json:"activeDigest,omitempty"`
	ActivatedAt       *time.Time     `json:"activatedAt,omitempty"`
	LastReloadAttempt *ReloadAttempt `json:"lastReloadAttempt,omitempty"`
}

// State owns the single active model-policy revision. Candidate revisions are
// fully loaded and validated before the lock is acquired, making activation a
// single pointer swap. Readers therefore observe either the old complete
// revision or the new complete revision, never partial state.
type State struct {
	mu sync.RWMutex

	path        string
	requirement Requirement
	active      *Revision
	activatedAt time.Time
	lastAttempt *ReloadAttempt
	now         func() time.Time
}

// NewState loads the initial revision while preserving a state object when the
// load fails. Callers must keep the returned state so readiness and status can
// expose the resolved path and exact diagnostic.
func NewState(path string, requirement Requirement) (*State, error) {
	return newState(path, requirement, time.Now)
}

func newState(path string, requirement Requirement, now func() time.Time) (*State, error) {
	if now == nil {
		now = time.Now
	}
	state := &State{
		path: strings.TrimSpace(path),
		requirement: Requirement{
			Required: requirement.Required,
			Reason:   strings.TrimSpace(requirement.Reason),
		},
		now: now,
	}
	_, err := state.Reload()
	return state, err
}

// Validate loads and validates the configured path without changing active
// state. The returned revision is immutable.
func (s *State) Validate() (*Revision, error) {
	if s == nil {
		return nil, fmt.Errorf("model policy state is not configured")
	}
	return Load(s.path)
}

// Reload validates a candidate revision before atomically activating it. A
// failed reload updates the diagnostic but never discards a valid active
// revision.
func (s *State) Reload() (*Revision, error) {
	if s == nil {
		return nil, fmt.Errorf("model policy state is not configured")
	}

	candidate, err := Load(s.path)
	attemptedAt := s.now().UTC()
	if err != nil {
		diagnostic := diagnosticFromError(err)
		s.mu.Lock()
		s.lastAttempt = &ReloadAttempt{
			AttemptedAt: attemptedAt,
			Succeeded:   false,
			Diagnostic:  diagnostic,
		}
		s.mu.Unlock()
		return nil, err
	}

	s.mu.Lock()
	s.active = candidate
	s.activatedAt = attemptedAt
	s.lastAttempt = &ReloadAttempt{
		AttemptedAt: attemptedAt,
		Succeeded:   true,
		Digest:      candidate.Digest(),
	}
	s.mu.Unlock()
	return candidate, nil
}

// SetRequirement updates whether the active catalog is a readiness dependency.
// It does not load or activate configuration.
func (s *State) SetRequirement(requirement Requirement) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.requirement = Requirement{
		Required: requirement.Required,
		Reason:   strings.TrimSpace(requirement.Reason),
	}
	s.mu.Unlock()
}

// Active returns the immutable active revision, if one exists.
func (s *State) Active() *Revision {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.active
}

// ModelIDs returns a copy of one runner's active static model inventory.
func (s *State) ModelIDs(runnerType domain.RunnerType) []string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.active == nil || s.active.catalog == nil {
		return nil
	}
	return s.active.catalog.ModelIDs(runnerType)
}

// IterModels implements the health probe's catalog snapshot seam without
// coupling modelpolicy to the health package.
func (s *State) IterModels(yield func(runnerType string, modelIDs []string) bool) {
	if s == nil || yield == nil {
		return
	}
	s.mu.RLock()
	if s.active == nil || s.active.catalog == nil {
		s.mu.RUnlock()
		return
	}
	catalog := s.active.catalog.Clone()
	s.mu.RUnlock()

	for runnerType := range catalog.Runners {
		if !yield(string(runnerType), catalog.ModelIDs(runnerType)) {
			return
		}
	}
}

// Status returns a deep copy suitable for API and health surfaces.
func (s *State) Status() Status {
	if s == nil {
		return Status{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := Status{
		Path:        s.path,
		Requirement: s.requirement,
		Ready:       !s.requirement.Required || s.active != nil,
	}
	if s.active != nil {
		status.ActiveDigest = s.active.Digest()
		activatedAt := s.activatedAt
		status.ActivatedAt = &activatedAt
	}
	status.LastReloadAttempt = cloneReloadAttempt(s.lastAttempt)
	return status
}

// ReadinessError returns an actionable error only when the catalog is required
// and no validated revision has ever been activated.
func (s *State) ReadinessError() error {
	status := s.Status()
	if status.Ready {
		return nil
	}
	reason := status.Requirement.Reason
	if reason == "" {
		reason = "model policy resolution is required"
	}
	message := "no validated catalog revision is active"
	if status.LastReloadAttempt != nil && status.LastReloadAttempt.Diagnostic != nil {
		message = status.LastReloadAttempt.Diagnostic.Message
	}
	return fmt.Errorf("required model policy catalog at %s is not ready (%s): %s", status.Path, reason, message)
}

func diagnosticFromError(err error) *Diagnostic {
	if err == nil {
		return nil
	}
	return &Diagnostic{
		Code:    DiagnosticCodeCatalogInvalid,
		Message: err.Error(),
		Cause:   rootCause(err).Error(),
	}
}

// DiagnosticForError converts validation/reload failures into the stable
// operator contract without exposing domain error implementation details.
func DiagnosticForError(err error) *Diagnostic {
	return diagnosticFromError(err)
}

func rootCause(err error) error {
	for {
		next := errors.Unwrap(err)
		if next == nil {
			return err
		}
		err = next
	}
}

func cloneReloadAttempt(attempt *ReloadAttempt) *ReloadAttempt {
	if attempt == nil {
		return nil
	}
	clone := *attempt
	if attempt.Diagnostic != nil {
		diagnostic := *attempt.Diagnostic
		clone.Diagnostic = &diagnostic
	}
	return &clone
}

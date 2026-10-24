package rolepolicy

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const DiagnosticCodeCatalogInvalid = "ROLE_POLICY_CATALOG_INVALID"

type Requirement struct {
	Required bool   `json:"required"`
	Reason   string `json:"reason,omitempty"`
}

type Diagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Cause   string `json:"cause,omitempty"`
}

type ReloadAttempt struct {
	AttemptedAt time.Time   `json:"attemptedAt"`
	Succeeded   bool        `json:"succeeded"`
	Digest      string      `json:"digest,omitempty"`
	Diagnostic  *Diagnostic `json:"diagnostic,omitempty"`
}

type Status struct {
	Path              string         `json:"path"`
	Requirement       Requirement    `json:"requirement"`
	Ready             bool           `json:"ready"`
	ActiveDigest      string         `json:"activeDigest,omitempty"`
	ActivatedAt       *time.Time     `json:"activatedAt,omitempty"`
	LastReloadAttempt *ReloadAttempt `json:"lastReloadAttempt,omitempty"`
}

// State atomically activates complete role catalog revisions. Failed reloads
// record diagnostics but retain the previously active revision.
type State struct {
	mu          sync.RWMutex
	path        string
	requirement Requirement
	active      *Revision
	activatedAt time.Time
	lastAttempt *ReloadAttempt
	now         func() time.Time
}

func NewState(path string, requirement Requirement) (*State, error) {
	return newState(path, requirement, time.Now)
}

func newState(path string, requirement Requirement, now func() time.Time) (*State, error) {
	if now == nil {
		now = time.Now
	}
	state := &State{path: strings.TrimSpace(path), requirement: Requirement{Required: requirement.Required, Reason: strings.TrimSpace(requirement.Reason)}, now: now}
	_, err := state.Reload()
	return state, err
}

func (s *State) Validate() (*Revision, error) {
	if s == nil {
		return nil, fmt.Errorf("role policy state is not configured")
	}
	return Load(s.path)
}

func (s *State) Reload() (*Revision, error) {
	if s == nil {
		return nil, fmt.Errorf("role policy state is not configured")
	}
	candidate, err := Load(s.path)
	attemptedAt := s.now().UTC()
	if err != nil {
		s.mu.Lock()
		s.lastAttempt = &ReloadAttempt{AttemptedAt: attemptedAt, Diagnostic: diagnosticFromError(err)}
		s.mu.Unlock()
		return nil, err
	}
	s.mu.Lock()
	s.active = candidate
	s.activatedAt = attemptedAt
	s.lastAttempt = &ReloadAttempt{AttemptedAt: attemptedAt, Succeeded: true, Digest: candidate.Digest()}
	s.mu.Unlock()
	return candidate, nil
}

func (s *State) Active() *Revision {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.active
}

func (s *State) Status() Status {
	if s == nil {
		return Status{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	status := Status{Path: s.path, Requirement: s.requirement, Ready: !s.requirement.Required || s.active != nil}
	if s.active != nil {
		status.ActiveDigest = s.active.Digest()
		activatedAt := s.activatedAt
		status.ActivatedAt = &activatedAt
	}
	if s.lastAttempt != nil {
		attempt := *s.lastAttempt
		if attempt.Diagnostic != nil {
			diagnostic := *attempt.Diagnostic
			attempt.Diagnostic = &diagnostic
		}
		status.LastReloadAttempt = &attempt
	}
	return status
}

func (s *State) ReadinessError() error {
	status := s.Status()
	if status.Ready {
		return nil
	}
	reason := status.Requirement.Reason
	if reason == "" {
		reason = "role resolution is required"
	}
	message := "no validated catalog revision is active"
	if status.LastReloadAttempt != nil && status.LastReloadAttempt.Diagnostic != nil {
		message = status.LastReloadAttempt.Diagnostic.Message
	}
	return fmt.Errorf("required role policy catalog at %s is not ready (%s): %s", status.Path, reason, message)
}

func diagnosticFromError(err error) *Diagnostic {
	if err == nil {
		return nil
	}
	return &Diagnostic{Code: DiagnosticCodeCatalogInvalid, Message: err.Error(), Cause: rootCause(err).Error()}
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

package documents

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"

	"persona/internal/handoffs"
	"persona/internal/journal"
	"persona/internal/personas"
)

var (
	ErrMissingPersona               = errors.New("persona_id is required")
	ErrMissingDocument              = errors.New("document_id is required")
	ErrMissingHandoff               = errors.New("handoff_id is required")
	ErrBindingNotFound              = errors.New("document binding not found")
	ErrDocumentAuthorityUnavailable = errors.New("document-manager is unreachable")
	ErrHandoffMismatch              = errors.New("document is not required by the named handoff")
	ErrBindingExists                = errors.New("document is already bound to this persona")
)

type (
	Binding struct {
		ID, PersonaID, DocumentID, DocumentKind string
		ValidUntil, CreatedAt                   time.Time
	}
	Release struct {
		ID, HandoffID, DocumentID string
		ReleasedAt                time.Time
	}
	HealthFinding   = personas.HealthFinding
	PersonaResolver interface {
		Get(context.Context, string) (personas.Persona, error)
	}
	HandoffResolver interface {
		Get(context.Context, string) (handoffs.Handoff, error)
	}
	Authority interface {
		Check(context.Context) error
		Release(context.Context, string, string) (string, error)
	}
	Service interface {
		Bind(context.Context, BindingInput) (Binding, error)
		List(context.Context, string) ([]Binding, error)
		CheckHealth(context.Context, string) ([]personas.HealthFinding, error)
		ReleaseIntoHandoff(context.Context, ReleaseInput) (Release, error)
	}
	BindingInput struct {
		PersonaID, DocumentID, DocumentKind string
		ValidUntil                          time.Time
	}
	ReleaseInput struct{ PersonaID, DocumentID, HandoffID string }
	service      struct {
		repo      Repository
		personas  PersonaResolver
		handoffs  HandoffResolver
		authority Authority
		journal   journal.Service
		clock     schedule.Clock
	}
)

func NewService(repo Repository, p PersonaResolver, h HandoffResolver, authority Authority, j journal.Service, clock schedule.Clock) Service {
	if clock == nil {
		clock = schedule.System()
	}
	return &service{repo: repo, personas: p, handoffs: h, authority: authority, journal: j, clock: clock}
}

var _ Service = (*service)(nil)

func (s *service) Bind(ctx context.Context, in BindingInput) (Binding, error) {
	if strings.TrimSpace(in.PersonaID) == "" {
		return Binding{}, ErrMissingPersona
	}
	if strings.TrimSpace(in.DocumentID) == "" {
		return Binding{}, ErrMissingDocument
	}
	if _, err := s.personas.Get(ctx, in.PersonaID); err != nil {
		return Binding{}, err
	}
	if err := s.authority.Check(ctx); err != nil {
		s.record(ctx, in.PersonaID, "document_bind_refused", "refused", "document_authority_unreachable", map[string]string{"document_id": in.DocumentID})
		return Binding{}, fmt.Errorf("%w: %v", ErrDocumentAuthorityUnavailable, err)
	}
	binding, err := s.repo.Create(ctx, Binding{PersonaID: in.PersonaID, DocumentID: in.DocumentID, DocumentKind: in.DocumentKind, ValidUntil: in.ValidUntil})
	if err == nil {
		s.record(ctx, in.PersonaID, "document_bound", "granted", "", map[string]string{"document_id": binding.DocumentID})
	}
	return binding, err
}

func (s *service) List(ctx context.Context, personaID string) ([]Binding, error) {
	if strings.TrimSpace(personaID) == "" {
		return nil, ErrMissingPersona
	}
	return s.repo.List(ctx, personaID)
}

func (s *service) CheckHealth(ctx context.Context, personaID string) ([]personas.HealthFinding, error) {
	if strings.TrimSpace(personaID) == "" {
		return nil, ErrMissingPersona
	}
	bindings, err := s.repo.List(ctx, personaID)
	if err != nil {
		return nil, err
	}
	findings := make([]personas.HealthFinding, 0)
	now := s.clock.Now().UTC()
	for _, binding := range bindings {
		if !binding.ValidUntil.IsZero() && !binding.ValidUntil.After(now) {
			findings = append(findings, personas.HealthFinding{Code: "document_expired", Message: "Document binding " + binding.DocumentID + " is past its validity date.", Blocking: true})
		}
	}
	if err := s.authority.Check(ctx); err != nil {
		findings = append(findings, personas.HealthFinding{Code: "document_authority_unreachable", Message: err.Error(), Blocking: true})
	}
	return findings, nil
}

func (s *service) ReleaseIntoHandoff(ctx context.Context, in ReleaseInput) (Release, error) {
	if strings.TrimSpace(in.PersonaID) == "" {
		return Release{}, ErrMissingPersona
	}
	if strings.TrimSpace(in.DocumentID) == "" {
		return Release{}, ErrMissingDocument
	}
	if strings.TrimSpace(in.HandoffID) == "" {
		return Release{}, ErrMissingHandoff
	}
	binding, err := s.repo.Get(ctx, in.PersonaID, in.DocumentID)
	if err != nil {
		return Release{}, err
	}
	h, err := s.handoffs.Get(ctx, in.HandoffID)
	if err != nil {
		return Release{}, err
	}
	if h.PersonaID != in.PersonaID || !contains(h.Checkpoint.RequiredDocumentIDs, in.DocumentID) {
		s.record(ctx, in.PersonaID, "document_release_refused", "refused", "handoff_mismatch", map[string]string{"document_id": in.DocumentID, "handoff_id": in.HandoffID})
		return Release{}, ErrHandoffMismatch
	}
	if err := s.authority.Check(ctx); err != nil {
		s.record(ctx, in.PersonaID, "document_release_refused", "refused", "document_authority_unreachable", map[string]string{"document_id": in.DocumentID, "handoff_id": in.HandoffID})
		return Release{}, fmt.Errorf("%w: %v", ErrDocumentAuthorityUnavailable, err)
	}
	releaseID, err := s.authority.Release(ctx, binding.DocumentID, h.ID)
	if err != nil {
		s.record(ctx, in.PersonaID, "document_release_refused", "refused", "document_authority_release_failed", map[string]string{"document_id": in.DocumentID, "handoff_id": in.HandoffID})
		return Release{}, err
	}
	release := Release{ID: releaseID, HandoffID: h.ID, DocumentID: binding.DocumentID, ReleasedAt: s.clock.Now().UTC()}
	s.record(ctx, in.PersonaID, "document_released", "granted", "", map[string]string{"document_id": release.DocumentID, "handoff_id": release.HandoffID})
	return release, nil
}

func (s *service) record(ctx context.Context, personaID, verb, outcome, constraint string, details map[string]string) {
	if s.journal == nil {
		return
	}
	_, _ = s.journal.Append(ctx, journal.Entry{PersonaID: personaID, Actor: "agent", Verb: verb, Outcome: outcome, Constraint: constraint, Details: details})
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
func Schema() string { return schemaSQL }

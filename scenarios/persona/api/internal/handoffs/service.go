package handoffs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"

	"persona/internal/journal"
	"persona/internal/personas"
)

var (
	ErrMissingPersona    = errors.New("persona_id is required")
	ErrMissingHandoff    = errors.New("handoff_id is required")
	ErrInvalidHandoff    = errors.New("handoff kind, human action, and checkpoint are required")
	ErrNotFound          = errors.New("handoff not found")
	ErrInvalidTransition = errors.New("handoff transition is not allowed")
	ErrExpired           = errors.New("handoff is expired")
	ErrProposalDenied    = errors.New("handoff proposal is not authorised")
)

type State string

const (
	StateOpen          State = "open"
	StateDelivered     State = "delivered"
	StateAwaitingHuman State = "awaiting_human"
	StateCompleted     State = "completed"
	StateExpired       State = "expired"
	StateCancelled     State = "cancelled"
	StateResumed       State = "resumed"
)

type (
	Field      struct{ Name, Value string }
	Checkpoint struct {
		CompletedFields     []Field
		RequiredDocumentIDs []string
		ResumeToken         string
	}
	Handoff struct {
		ID, PersonaID, Kind, Title, HumanAction string
		Checkpoint                              Checkpoint
		State                                   State
		OpenedByRunID, AuthorisingHuman         string
		Deadline, CreatedAt, UpdatedAt          time.Time
		RelayState                              string
	}
)

type EnrolmentField struct {
	Name, Value string
	HumanOnly   bool
}

type (
	PersonaResolver interface {
		Get(context.Context, string) (personas.Persona, error)
	}
	ProposalAuthorizer interface {
		AuthorizeProposal(context.Context, string, string) (string, string, error)
	}
	Service interface {
		Open(context.Context, OpenInput) (Handoff, error)
		Get(context.Context, string) (Handoff, error)
		List(context.Context, string, int) ([]Handoff, error)
		Complete(context.Context, string, string) (Handoff, error)
		Cancel(context.Context, string, string) (Handoff, error)
		Resume(context.Context, string, string) (Handoff, error)
		PrepareEnrolment(context.Context, EnrolmentInput) ([]EnrolmentField, Handoff, error)
	}
	Relay interface {
		Deliver(context.Context, Handoff) error
	}
)

type (
	OpenInput struct {
		PersonaID, Kind, Title, HumanAction, OpenedByRunID, AuthorisingHuman, IdentityToken string
		Checkpoint                                                                          Checkpoint
		Deadline                                                                            time.Time
	}
	EnrolmentInput struct {
		PersonaID, Target, IdentityToken string
		RequiredFields                   []string
	}
)

type service struct {
	repo     Repository
	personas PersonaResolver
	journal  journal.Service
	relay    Relay
	proposer ProposalAuthorizer
	clock    schedule.Clock
}

func NewService(repo Repository, resolver PersonaResolver, actionJournal journal.Service, clock schedule.Clock) Service {
	return NewServiceWithRelayAndAuthorizer(repo, resolver, actionJournal, nil, nil, clock)
}

func NewServiceWithRelay(repo Repository, resolver PersonaResolver, actionJournal journal.Service, relay Relay, clock schedule.Clock) Service {
	return NewServiceWithRelayAndAuthorizer(repo, resolver, actionJournal, relay, nil, clock)
}

func NewServiceWithRelayAndAuthorizer(repo Repository, resolver PersonaResolver, actionJournal journal.Service, relay Relay, proposer ProposalAuthorizer, clock schedule.Clock) Service {
	if clock == nil {
		clock = schedule.System()
	}
	return &service{repo: repo, personas: resolver, journal: actionJournal, relay: relay, proposer: proposer, clock: clock}
}

var _ Service = (*service)(nil)

// AllowedTransitions is the complete state machine. Any edge not listed here
// is rejected; terminal states have no outgoing edge except Completed→Resumed.
var AllowedTransitions = map[State]map[State]bool{
	StateOpen:          {StateDelivered: true},
	StateDelivered:     {StateAwaitingHuman: true},
	StateAwaitingHuman: {StateCompleted: true, StateExpired: true, StateCancelled: true},
	StateCompleted:     {StateResumed: true},
	StateExpired:       {}, StateCancelled: {}, StateResumed: {},
}

func (s *service) Open(ctx context.Context, in OpenInput) (Handoff, error) {
	if strings.TrimSpace(in.PersonaID) == "" {
		return Handoff{}, ErrMissingPersona
	}
	if strings.TrimSpace(in.Kind) == "" || strings.TrimSpace(in.HumanAction) == "" || (len(in.Checkpoint.CompletedFields) == 0 && len(in.Checkpoint.RequiredDocumentIDs) == 0 && strings.TrimSpace(in.Checkpoint.ResumeToken) == "") {
		return Handoff{}, ErrInvalidHandoff
	}
	if _, err := s.personas.Get(ctx, in.PersonaID); err != nil {
		return Handoff{}, err
	}
	if s.proposer != nil {
		runID, human, err := s.proposer.AuthorizeProposal(ctx, in.PersonaID, in.IdentityToken)
		if err != nil {
			s.record(ctx, Handoff{PersonaID: in.PersonaID, OpenedByRunID: in.OpenedByRunID, AuthorisingHuman: in.AuthorisingHuman}, "handoff_open_refused", "refused", "proposal_not_authorised")
			return Handoff{}, fmt.Errorf("%w: %v", ErrProposalDenied, err)
		}
		in.OpenedByRunID = runID
		in.AuthorisingHuman = human
	}
	now := s.clock.Now().UTC()
	deadline := in.Deadline.UTC()
	if deadline.IsZero() {
		deadline = now.Add(24 * time.Hour)
	}
	h, err := s.repo.Create(ctx, Handoff{PersonaID: in.PersonaID, Kind: in.Kind, Title: in.Title, HumanAction: in.HumanAction, Checkpoint: in.Checkpoint, State: StateOpen, OpenedByRunID: in.OpenedByRunID, AuthorisingHuman: in.AuthorisingHuman, Deadline: deadline})
	if err != nil {
		return Handoff{}, err
	}
	s.record(ctx, h, "handoff_opened", "granted", "")
	h, err = s.repo.UpdateState(ctx, h.ID, StateDelivered, "system", "")
	if err != nil {
		return Handoff{}, err
	}
	s.record(ctx, h, "handoff_delivered", "granted", "")
	h, err = s.repo.UpdateState(ctx, h.ID, StateAwaitingHuman, "system", "")
	if err != nil {
		return Handoff{}, err
	}
	s.record(ctx, h, "handoff_awaiting_human", "granted", "")
	if s.relay != nil {
		if err := s.relay.Deliver(ctx, h); err != nil {
			h.RelayState = "deferred"
			if updated, updateErr := s.repo.SetRelayState(ctx, h.ID, h.RelayState); updateErr == nil {
				h = updated
			}
			s.record(ctx, h, "handoff_relay_deferred", "refused", "notification_hub_unavailable")
		} else {
			h.RelayState = "delivered"
			if updated, updateErr := s.repo.SetRelayState(ctx, h.ID, h.RelayState); updateErr == nil {
				h = updated
			}
			s.record(ctx, h, "handoff_relay_delivered", "granted", "")
		}
	}
	return h, nil
}

func (s *service) Get(ctx context.Context, id string) (Handoff, error) {
	if strings.TrimSpace(id) == "" {
		return Handoff{}, ErrMissingHandoff
	}
	h, err := s.repo.Get(ctx, id)
	if err != nil {
		return Handoff{}, err
	}
	if (h.State == StateOpen || h.State == StateDelivered || h.State == StateAwaitingHuman) && !h.Deadline.After(s.clock.Now()) {
		h, err = s.repo.UpdateState(ctx, h.ID, StateExpired, "system", "deadline")
		if err == nil {
			s.record(ctx, h, "handoff_expired", "refused", "deadline")
		}
	}
	return h, err
}

func (s *service) List(ctx context.Context, personaID string, limit int) ([]Handoff, error) {
	if strings.TrimSpace(personaID) == "" {
		return nil, ErrMissingPersona
	}
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	return s.repo.List(ctx, personaID, limit)
}

func (s *service) Complete(ctx context.Context, id, completedBy string) (Handoff, error) {
	h, err := s.Get(ctx, id)
	if err != nil {
		return Handoff{}, err
	}
	if h.State != StateAwaitingHuman {
		return Handoff{}, fmt.Errorf("%w: %s to completed", ErrInvalidTransition, h.State)
	}
	h, err = s.repo.UpdateState(ctx, id, StateCompleted, completedBy, "")
	if err == nil {
		s.record(ctx, h, "handoff_completed", "granted", "")
	}
	return h, err
}

func (s *service) Cancel(ctx context.Context, id, cancelledBy string) (Handoff, error) {
	h, err := s.Get(ctx, id)
	if err != nil {
		return Handoff{}, err
	}
	if h.State != StateAwaitingHuman {
		return Handoff{}, fmt.Errorf("%w: %s to cancelled", ErrInvalidTransition, h.State)
	}
	h, err = s.repo.UpdateState(ctx, id, StateCancelled, cancelledBy, "")
	if err == nil {
		s.record(ctx, h, "handoff_cancelled", "refused", "operator_cancelled")
	}
	return h, err
}

func (s *service) Resume(ctx context.Context, id, runID string) (Handoff, error) {
	h, err := s.Get(ctx, id)
	if err != nil {
		return Handoff{}, err
	}
	if h.State != StateCompleted {
		return Handoff{}, fmt.Errorf("%w: %s to resumed", ErrInvalidTransition, h.State)
	}
	h, err = s.repo.UpdateState(ctx, id, StateResumed, runID, "")
	if err == nil {
		s.record(ctx, h, "handoff_resumed", "granted", "")
	}
	return h, err
}

func (s *service) PrepareEnrolment(ctx context.Context, in EnrolmentInput) ([]EnrolmentField, Handoff, error) {
	if strings.TrimSpace(in.PersonaID) == "" {
		return nil, Handoff{}, ErrMissingPersona
	}
	if strings.TrimSpace(in.Target) == "" || len(in.RequiredFields) == 0 {
		return nil, Handoff{}, ErrInvalidHandoff
	}
	if _, err := s.personas.Get(ctx, in.PersonaID); err != nil {
		return nil, Handoff{}, err
	}
	fields := make([]EnrolmentField, 0, len(in.RequiredFields))
	names := make([]string, 0, len(in.RequiredFields))
	for _, name := range in.RequiredFields {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		fields = append(fields, EnrolmentField{Name: name, HumanOnly: true})
		names = append(names, name)
	}
	h, err := s.Open(ctx, OpenInput{PersonaID: in.PersonaID, Kind: "identity_enrolment", Title: "Complete " + in.Target, HumanAction: "Complete the human-only identity steps for this enrolment.", IdentityToken: in.IdentityToken, Checkpoint: Checkpoint{ResumeToken: strings.Join(names, ",")}})
	if err != nil {
		return nil, Handoff{}, err
	}
	s.record(ctx, h, "enrolment_prepared", "granted", "")
	return fields, h, nil
}

func (s *service) record(ctx context.Context, h Handoff, verb, outcome, constraint string) {
	if s.journal == nil {
		return
	}
	_, _ = s.journal.Append(ctx, journal.Entry{PersonaID: h.PersonaID, Actor: "operator", Verb: verb, RunID: h.OpenedByRunID, AuthorisingHuman: h.AuthorisingHuman, Outcome: outcome, Constraint: constraint, Details: map[string]string{"handoff_id": h.ID, "state": string(h.State)}})
}

func Schema() string { return schemaSQL }

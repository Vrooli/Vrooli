package handoffs

import (
	"context"
	"errors"
	"testing"
	"time"

	"persona/internal/journal"
	"persona/internal/personas"
)

type handoffPersonaFunc func(context.Context, string) (personas.Persona, error)

func (f handoffPersonaFunc) Get(ctx context.Context, id string) (personas.Persona, error) {
	return f(ctx, id)
}

type handoffJournal struct{ entries []journal.Entry }

func (j *handoffJournal) Append(_ context.Context, entry journal.Entry) (journal.Entry, error) {
	j.entries = append(j.entries, entry)
	return entry, nil
}
func (j *handoffJournal) List(context.Context, string, int) ([]journal.Entry, error) {
	return j.entries, nil
}

type handoffRelay struct{ err error }

func (r handoffRelay) Deliver(context.Context, Handoff) error { return r.err }

type handoffRepository struct {
	handoff Handoff
}

func (r *handoffRepository) Create(_ context.Context, h Handoff) (Handoff, error) {
	h.ID = "handoff-1"
	h.CreatedAt = time.Now().UTC()
	h.UpdatedAt = h.CreatedAt
	r.handoff = h
	return h, nil
}

func (r *handoffRepository) Get(_ context.Context, id string) (Handoff, error) {
	if id != r.handoff.ID {
		return Handoff{}, ErrNotFound
	}
	return r.handoff, nil
}
func (r *handoffRepository) List(context.Context, string, int) ([]Handoff, error) {
	return []Handoff{r.handoff}, nil
}
func (r *handoffRepository) UpdateState(_ context.Context, id string, state State, actor, reason string) (Handoff, error) {
	if id != r.handoff.ID {
		return Handoff{}, ErrNotFound
	}
	r.handoff.State = state
	r.handoff.UpdatedAt = time.Now().UTC()
	if state == StateExpired {
		r.handoff.RelayState = reason
	}
	return r.handoff, nil
}

func (r *handoffRepository) SetRelayState(_ context.Context, id, relayState string) (Handoff, error) {
	if id != r.handoff.ID {
		return Handoff{}, ErrNotFound
	}
	r.handoff.RelayState = relayState
	return r.handoff, nil
}

func handoffPersona() personas.Persona {
	return personas.Persona{ID: "persona-1", Kind: personas.KindPersonal, Status: personas.StatusActive}
}

func TestHandoffCarriesCheckpointAndResumesWithoutOriginatingRun(t *testing.T) {
	// [REQ:PSN-P0-007] a human wall is a resumable checkpoint, not a dropped error.
	repo := &handoffRepository{}
	j := &handoffJournal{}
	service := NewService(repo, handoffPersonaFunc(func(context.Context, string) (personas.Persona, error) { return handoffPersona(), nil }), j, nil)
	handoff, err := service.Open(context.Background(), OpenInput{
		PersonaID: "persona-1", Kind: "identity_verification", Title: "Verify identity", HumanAction: "Upload the government ID and finish the review.", OpenedByRunID: "run-that-exited",
		Checkpoint: Checkpoint{CompletedFields: []Field{{Name: "email", Value: "persona@example.test"}}, RequiredDocumentIDs: []string{"document-1"}, ResumeToken: "checkpoint-1"},
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if handoff.State != StateAwaitingHuman || handoff.HumanAction == "" || len(handoff.Checkpoint.CompletedFields) != 1 || len(handoff.Checkpoint.RequiredDocumentIDs) != 1 {
		t.Fatalf("handoff = %#v", handoff)
	}
	completed, err := service.Complete(context.Background(), handoff.ID, "human-1")
	if err != nil || completed.State != StateCompleted {
		t.Fatalf("Complete() = %#v, %v", completed, err)
	}
	resumed, err := service.Resume(context.Background(), handoff.ID, "different-run")
	if err != nil || resumed.State != StateResumed {
		t.Fatalf("Resume() = %#v, %v", resumed, err)
	}
	if resumed.OpenedByRunID != "run-that-exited" {
		t.Fatalf("originating run was rewritten during resume: %#v", resumed)
	}
}

func TestHandoffRejectsUnnamedTransitionsAndRecordsRelayDegradation(t *testing.T) {
	// [REQ:PSN-P0-007] and [REQ:PSN-P1-004] require a strict state table and an optional relay.
	repo := &handoffRepository{}
	j := &handoffJournal{}
	service := NewServiceWithRelay(repo, handoffPersonaFunc(func(context.Context, string) (personas.Persona, error) { return handoffPersona(), nil }), j, handoffRelay{err: errors.New("notification-hub down")}, nil)
	handoff, err := service.Open(context.Background(), OpenInput{PersonaID: "persona-1", Kind: "captcha", HumanAction: "Complete the CAPTCHA in the counterparty console.", Checkpoint: Checkpoint{ResumeToken: "checkpoint-1"}})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if handoff.State != StateAwaitingHuman || handoff.RelayState != "deferred" || len(j.entries) != 4 || j.entries[3].Verb != "handoff_relay_deferred" || j.entries[3].Outcome != "refused" {
		t.Fatalf("relay degradation journal = %#v", j.entries)
	}
	if _, err := service.Resume(context.Background(), handoff.ID, "run-2"); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("resume before completion error = %v", err)
	}
	if _, err := service.Complete(context.Background(), handoff.ID, "human-1"); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if _, err := service.Complete(context.Background(), handoff.ID, "human-1"); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("complete after terminal transition error = %v", err)
	}

	// [REQ:PSN-P0-007] the terminal states deliberately have no implicit retry edge.
	if len(AllowedTransitions[StateExpired]) != 0 || len(AllowedTransitions[StateCancelled]) != 0 || len(AllowedTransitions[StateResumed]) != 0 {
		t.Fatal("terminal handoff state has an outgoing transition")
	}
}

func TestExpiredHandoffRemainsReadable(t *testing.T) {
	// [REQ:PSN-P0-007] expiry is a terminal, inspectable state rather than deletion.
	repo := &handoffRepository{}
	service := NewService(repo, handoffPersonaFunc(func(context.Context, string) (personas.Persona, error) { return handoffPersona(), nil }), nil, nil)
	opened, err := service.Open(context.Background(), OpenInput{PersonaID: "persona-1", Kind: "review", HumanAction: "Complete the operator review.", Checkpoint: Checkpoint{ResumeToken: "checkpoint-1"}, Deadline: time.Now().Add(-time.Minute)})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	expired, err := service.Get(context.Background(), opened.ID)
	if err != nil || expired.State != StateExpired || !expired.Deadline.Equal(opened.Deadline) {
		t.Fatalf("expired handoff = %#v, %v", expired, err)
	}
}

func TestPrepareEnrolmentUsesOneHumanOnlyHandoff(t *testing.T) {
	// [REQ:PSN-P1-008] all human-only fields are assembled into one resumable handoff.
	repo := &handoffRepository{}
	service := NewService(repo, handoffPersonaFunc(func(context.Context, string) (personas.Persona, error) { return handoffPersona(), nil }), nil, nil)
	fields, handoff, err := service.PrepareEnrolment(context.Background(), EnrolmentInput{PersonaID: "persona-1", Target: "counterparty", RequiredFields: []string{"government_id", "captcha", "biometric_review"}})
	if err != nil {
		t.Fatalf("PrepareEnrolment() error = %v", err)
	}
	if len(fields) != 3 || len(handoff.Checkpoint.CompletedFields) != 0 || handoff.Checkpoint.ResumeToken != "government_id,captcha,biometric_review" {
		t.Fatalf("prepared enrolment = fields %#v handoff %#v", fields, handoff)
	}
	for _, field := range fields {
		if !field.HumanOnly || field.Value != "" {
			t.Fatalf("enrolment field was not human-only and blank: %#v", field)
		}
	}
}

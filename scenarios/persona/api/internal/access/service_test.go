package access

import (
	"context"
	"errors"
	"testing"
	"time"

	"persona/internal/journal"
	"persona/internal/personas"
)

type accessVerifierFunc func(context.Context, string) (*Claims, error)

func (f accessVerifierFunc) Verify(ctx context.Context, token string) (*Claims, error) {
	return f(ctx, token)
}

type accessPersonaFunc func(context.Context, string) (personas.Persona, error)

func (f accessPersonaFunc) Get(ctx context.Context, id string) (personas.Persona, error) {
	return f(ctx, id)
}

type accessJournal struct{ entries []journal.Entry }

func (j *accessJournal) Append(_ context.Context, entry journal.Entry) (journal.Entry, error) {
	j.entries = append(j.entries, entry)
	return entry, nil
}

func (j *accessJournal) List(context.Context, string, int) ([]journal.Entry, error) {
	return append([]journal.Entry(nil), j.entries...), nil
}

type accessRepository struct {
	grants []Grant
}

func (r *accessRepository) CreateGrant(_ context.Context, grant Grant) (Grant, error) {
	grant.ID = "grant-1"
	r.grants = append(r.grants, grant)
	return grant, nil
}

func (r *accessRepository) ListGrants(context.Context, string) ([]Grant, error) {
	return append([]Grant(nil), r.grants...), nil
}

func (r *accessRepository) GetGrant(_ context.Context, id string) (Grant, error) {
	for _, grant := range r.grants {
		if grant.ID == id {
			return grant, nil
		}
	}
	return Grant{}, ErrGrantNotFound
}

func (r *accessRepository) RemoveGrant(_ context.Context, id string) error {
	for i, grant := range r.grants {
		if grant.ID == id {
			r.grants = append(r.grants[:i], r.grants[i+1:]...)
			return nil
		}
	}
	return ErrGrantNotFound
}

func accessPersona() personas.Persona {
	return personas.Persona{ID: "persona-1", Kind: personas.KindPersonal, DisplayName: "Ada's identity", LegalBasis: personas.LegalBasis{SubjectID: "legal-1", SubjectName: "Ada", BasisType: "authorised"}, Status: personas.StatusActive}
}

func accessClaims() *Claims {
	return &Claims{RunID: "run-1", Subject: "human-1", Scopes: []string{"persona.act-as:persona-1", "persona.resolve.display", "persona.resolve.legal"}, ExpiresAt: time.Now().Add(time.Hour)}
}

func newAccessService(repo *accessRepository, verifier Verifier, j *accessJournal) Service {
	return NewService(repo, accessPersonaFunc(func(context.Context, string) (personas.Persona, error) { return accessPersona(), nil }), j, verifier, ServiceOptions{Secret: []byte("test-signing-key"), KeyID: "test-key"})
}

func TestActAsVerifiesLiveAndRecordsGrantAndRefusal(t *testing.T) {
	ctx := context.Background()
	repo := &accessRepository{grants: []Grant{{ID: "grant-1", PersonaID: "persona-1", HumanSubject: "human-1", Level: GrantAct, Source: "prompt-manager"}}}
	j := &accessJournal{}
	service := newAccessService(repo, accessVerifierFunc(func(context.Context, string) (*Claims, error) { return accessClaims(), nil }), j)

	// [REQ:PSN-P0-003] and [REQ:PSN-P0-004] require live verification plus a server-side act grant.
	session, err := service.ActAs(ctx, "persona-1", "token", "send_message")
	if err != nil {
		t.Fatalf("ActAs() error = %v", err)
	}
	if session.RunID != "run-1" || session.AuthorisingHuman != "human-1" {
		t.Fatalf("session attribution = %#v", session)
	}
	if len(j.entries) != 1 || j.entries[0].Verb != "act_as_granted" || j.entries[0].RunID != "run-1" {
		t.Fatalf("grant journal = %#v", j.entries)
	}

	j.entries = nil
	service = newAccessService(repo, accessVerifierFunc(func(context.Context, string) (*Claims, error) { return nil, errors.New("agent-manager down") }), j)
	_, err = service.ActAs(ctx, "persona-1", "token", "send_message")
	if !errors.Is(err, ErrAuthorityUnreachable) {
		t.Fatalf("unreachable ActAs() error = %v", err)
	}
	if len(j.entries) != 1 || j.entries[0].Outcome != "refused" || j.entries[0].Constraint != "authority_unreachable" {
		t.Fatalf("refusal journal = %#v", j.entries)
	}
}

func TestProposeGrantCannotAct(t *testing.T) {
	// [REQ:PSN-P0-004] propose is not an implicit act permission.
	repo := &accessRepository{grants: []Grant{{ID: "grant-1", PersonaID: "persona-1", HumanSubject: "human-1", Level: GrantPropose}}}
	j := &accessJournal{}
	service := newAccessService(repo, accessVerifierFunc(func(context.Context, string) (*Claims, error) { return accessClaims(), nil }), j)
	_, err := service.ActAs(context.Background(), "persona-1", "token", "dangerous_action")
	if !errors.Is(err, ErrProposeOnly) {
		t.Fatalf("propose ActAs() error = %v", err)
	}
	if len(j.entries) != 1 || j.entries[0].Constraint != "propose_only" || j.entries[0].Outcome != "refused" {
		t.Fatalf("propose refusal journal = %#v", j.entries)
	}
}

func TestGrantSourceIsRecordedForOrganisationalPolicy(t *testing.T) {
	// [REQ:PSN-P1-006] prompt-manager may supply the organisational grant; the source is auditable.
	repo := &accessRepository{}
	service := newAccessService(repo, accessVerifierFunc(func(context.Context, string) (*Claims, error) { return accessClaims(), nil }), &accessJournal{})
	grant, err := service.CreateGrant(context.Background(), GrantInput{PersonaID: "persona-1", HumanSubject: "human-1", Level: GrantAct, Source: "prompt-manager"})
	if err != nil || grant.Source != "prompt-manager" {
		t.Fatalf("CreateGrant() = %#v, %v", grant, err)
	}
}

func TestResolvePersonaReturnsOnlyEntitledFields(t *testing.T) {
	// [REQ:PSN-P0-012] and [REQ:PSN-P0-013] require a named persona and centralised attenuation.
	j := &accessJournal{}
	service := newAccessService(&accessRepository{}, accessVerifierFunc(func(context.Context, string) (*Claims, error) { return accessClaims(), nil }), j)
	resolution, err := service.ResolvePersona(context.Background(), "persona-1", "token", []string{"display_name", "legal_subject_id", "document_bytes", "credential_value"})
	if err != nil {
		t.Fatalf("ResolvePersona() error = %v", err)
	}
	if resolution.DisplayName == "" || resolution.LegalSubjectID == "" {
		t.Fatalf("entitled resolution = %#v", resolution)
	}
	if resolution.ControlledEmail != "" || len(resolution.AddressIDs) != 0 {
		t.Fatalf("resolution leaked unrequested sensitive fields = %#v", resolution)
	}
	for _, field := range resolution.ReturnedFields {
		if field == "document_bytes" || field == "credential_value" {
			t.Fatalf("resolution returned forbidden field %q", field)
		}
	}
	if _, err := service.ResolvePersona(context.Background(), "", "token", nil); !errors.Is(err, ErrMissingPersona) {
		t.Fatalf("unnamed resolution error = %v", err)
	}
}

func TestIssueAttestationRequiresConfiguredSigner(t *testing.T) {
	// [REQ:PSN-P1-007] an attestation must be cryptographically signed, never an empty placeholder.
	j := &accessJournal{}
	repo := &accessRepository{grants: []Grant{{ID: "grant-1", PersonaID: "persona-1", HumanSubject: "human-1", Level: GrantAct}}}
	service := NewService(repo, accessPersonaFunc(func(context.Context, string) (personas.Persona, error) { return accessPersona(), nil }), j, accessVerifierFunc(func(context.Context, string) (*Claims, error) { return accessClaims(), nil }), ServiceOptions{})
	_, err := service.IssueAttestation(context.Background(), "persona-1", "token", "counterparty", time.Now().Add(time.Minute))
	if !errors.Is(err, ErrAttestationSigner) {
		t.Fatalf("unconfigured signer error = %v", err)
	}

	service = newAccessService(repo, accessVerifierFunc(func(context.Context, string) (*Claims, error) { return accessClaims(), nil }), j)
	attestation, err := service.IssueAttestation(context.Background(), "persona-1", "token", "counterparty", time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("IssueAttestation() error = %v", err)
	}
	if len(attestation.Signature) == 0 || attestation.PersonaID != "persona-1" || attestation.AccountSubject != "human-1" {
		t.Fatalf("attestation = %#v", attestation)
	}
}

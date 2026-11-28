package access

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"testing"
	"time"

	accessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/access"

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

func (r *accessRepository) UpdateGrant(_ context.Context, grant Grant) (Grant, error) {
	for i := range r.grants {
		if r.grants[i].ID == grant.ID {
			r.grants[i] = grant
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

func TestActAsWithoutGrantRefuses(t *testing.T) {
	// [REQ:PSN-P0-004] a verified subject without a persona grant is refused.
	j := &accessJournal{}
	service := newAccessService(&accessRepository{}, accessVerifierFunc(func(context.Context, string) (*Claims, error) { return accessClaims(), nil }), j)
	_, err := service.ActAs(context.Background(), "persona-1", "token", "send_message")
	if !errors.Is(err, ErrGrantMissing) {
		t.Fatalf("no-grant ActAs() error = %v", err)
	}
	if len(j.entries) != 1 || j.entries[0].Verb != "act_as_refused" || j.entries[0].Constraint != "grant_missing" {
		t.Fatalf("no-grant refusal journal = %#v", j.entries)
	}
}

func TestProposeGrantAuthorizesHandoffProposal(t *testing.T) {
	// [REQ:PSN-P0-004] a propose grant authorizes a human handoff, not act-as.
	claims := accessClaims()
	claims.Scopes = []string{"persona.propose:persona-1"}
	repo := &accessRepository{grants: []Grant{{ID: "grant-1", PersonaID: "persona-1", HumanSubject: "human-1", Level: GrantPropose}}}
	service := newAccessService(repo, accessVerifierFunc(func(context.Context, string) (*Claims, error) { return claims, nil }), &accessJournal{})
	runID, human, err := service.AuthorizeProposal(context.Background(), "persona-1", "token")
	if err != nil {
		t.Fatalf("AuthorizeProposal() error = %v", err)
	}
	if runID != "run-1" || human != "human-1" {
		t.Fatalf("proposal attribution = run %q human %q", runID, human)
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

func TestGrantLifecycleIsJournaled(t *testing.T) {
	// [REQ:PSN-P0-004] create, change, and remove each leave an operator journal row.
	repo := &accessRepository{}
	j := &accessJournal{}
	service := newAccessService(repo, accessVerifierFunc(func(context.Context, string) (*Claims, error) { return accessClaims(), nil }), j)
	grant, err := service.CreateGrant(context.Background(), GrantInput{PersonaID: "persona-1", HumanSubject: "human-1", Level: GrantPropose, Source: "operator"})
	if err != nil {
		t.Fatalf("CreateGrant() error = %v", err)
	}
	if _, err := service.ChangeGrant(context.Background(), GrantChangeInput{GrantID: grant.ID, Level: GrantAct, Source: "operator-change"}); err != nil {
		t.Fatalf("ChangeGrant() error = %v", err)
	}
	if err := service.RemoveGrant(context.Background(), grant.ID); err != nil {
		t.Fatalf("RemoveGrant() error = %v", err)
	}
	if len(j.entries) != 3 || j.entries[0].Verb != "grant_created" || j.entries[1].Verb != "grant_changed" || j.entries[2].Verb != "grant_removed" {
		t.Fatalf("grant lifecycle journal = %#v", j.entries)
	}
	for _, entry := range j.entries {
		if entry.Actor != "operator" || entry.Outcome != "granted" {
			t.Fatalf("grant journal attribution = %#v", entry)
		}
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

func TestResolvePersonaEntitlementLevels(t *testing.T) {
	// [REQ:PSN-P0-012] the same typed RPC returns a narrower or wider field set
	// based on the verified scopes, never on a consumer-side policy.
	claims := accessClaims()
	claims.Scopes = nil
	service := newAccessService(&accessRepository{}, accessVerifierFunc(func(context.Context, string) (*Claims, error) { return claims, nil }), &accessJournal{})
	low, err := service.ResolvePersona(context.Background(), "persona-1", "token", []string{"display_name", "legal_subject_id"})
	if err != nil {
		t.Fatalf("low-entitlement ResolvePersona() error = %v", err)
	}
	if got, want := low.ReturnedFields, []string{"kind", "persona_id"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("low-entitlement fields = %v, want %v", got, want)
	}

	claims.Scopes = []string{"persona.resolve.display", "persona.resolve.legal"}
	high, err := service.ResolvePersona(context.Background(), "persona-1", "token", []string{"display_name", "legal_subject_id"})
	if err != nil {
		t.Fatalf("high-entitlement ResolvePersona() error = %v", err)
	}
	if got, want := high.ReturnedFields, []string{"display_name", "kind", "legal_subject_id", "persona_id"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("high-entitlement fields = %v, want %v", got, want)
	}
}

func TestResolvePersonaDescriptorHasNoSensitivePayloadFields(t *testing.T) {
	// [REQ:PSN-P0-013] the wire contract contains no document bytes or
	// credential values for any caller or scope.
	service := accessv1.File_persona_v1_access_access_proto.Services().ByName("AccessService")
	method := service.Methods().ByName("ResolvePersona")
	response := method.Output().Fields().ByName("persona").Message()
	fields := make([]string, 0, response.Fields().Len())
	for i := 0; i < response.Fields().Len(); i++ {
		fields = append(fields, string(response.Fields().Get(i).Name()))
	}
	sort.Strings(fields)
	want := []string{"address_ids", "controlled_email", "display_name", "kind", "legal_subject_id", "persona_id", "returned_fields"}
	if !reflect.DeepEqual(fields, want) {
		t.Fatalf("ResolvePersona response fields = %v, want %v", fields, want)
	}
	for _, field := range fields {
		if field == "document_bytes" || field == "credential_value" || field == "secret" {
			t.Fatalf("sensitive payload field %q is present", field)
		}
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

package personas

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestCreateRequiresLegalBasisAndKindSpecificIdentifier(t *testing.T) {
	ctx := context.Background()
	var created Persona
	service := NewService(FakeRepository{
		CreateFunc: func(_ context.Context, p Persona) (Persona, error) {
			created = p
			p.ID = "persona-1"
			return p, nil
		},
	})

	// [REQ:PSN-P0-001] a legal basis and an identifier are creation invariants.
	_, err := service.Create(ctx, CreateInput{Kind: KindPersonal, Identifiers: []Identifier{{Type: "passport", Value: "P-1"}}})
	if !errors.Is(err, ErrMissingLegal) {
		t.Fatalf("Create without legal basis error = %v, want %v", err, ErrMissingLegal)
	}
	_, err = service.Create(ctx, CreateInput{Kind: KindPersonal, LegalBasis: LegalBasis{SubjectID: "human-1", SubjectName: "Ada", BasisType: "authorised"}})
	if !errors.Is(err, ErrMissingIdentity) {
		t.Fatalf("Create without identifier error = %v, want %v", err, ErrMissingIdentity)
	}
	_, err = service.Create(ctx, CreateInput{Kind: KindPersonal, LegalBasis: LegalBasis{SubjectID: "human-1", SubjectName: "Ada", BasisType: "authorised"}, Identifiers: []Identifier{{Type: "business_registration", Value: "BR-1"}}})
	if !errors.Is(err, ErrInvalidIdentity) {
		t.Fatalf("cross-kind identifier error = %v, want %v", err, ErrInvalidIdentity)
	}

	// [REQ:PSN-P0-010] personal and business identities use disjoint identifier sets.
	if _, err := service.Create(ctx, CreateInput{Kind: KindPersonal, LegalBasis: LegalBasis{SubjectID: "human-1", SubjectName: "Ada", BasisType: "authorised"}, Identifiers: []Identifier{{Type: "passport", Value: "P-1"}}}); err != nil {
		t.Fatalf("personal Create() error = %v", err)
	}
	if created.Kind != KindPersonal || created.Identifiers[0].Type != "passport" {
		t.Fatalf("created personal identity = %#v", created)
	}
	if _, err := service.Create(ctx, CreateInput{Kind: KindBusiness, LegalBasis: LegalBasis{SubjectID: "company-1", SubjectName: "Acme", BasisType: "authorised"}, Identifiers: []Identifier{{Type: "business_registration", Value: "BR-1"}}}); err != nil {
		t.Fatalf("business Create() error = %v", err)
	}
}

func TestPersonaServiceHasNoLegalBasisMutationVerb(t *testing.T) {
	// [REQ:PSN-P0-001] legal basis is immutable: the service contract has no update path.
	serviceType := reflect.TypeOf((*Service)(nil)).Elem()
	for i := 0; i < serviceType.NumMethod(); i++ {
		if serviceType.Method(i).Name == "Update" || serviceType.Method(i).Name == "UpdateLegalBasis" || serviceType.Method(i).Name == "Delete" {
			t.Fatalf("mutable persona service method exposed: %s", serviceType.Method(i).Name)
		}
	}
}

func TestCheckHealthSurfacesArchivedPersona(t *testing.T) {
	// [REQ:PSN-P1-005] a stale/retired persona is surfaced before a flow uses it.
	service := NewService(FakeRepository{
		GetFunc: func(context.Context, string) (Persona, error) {
			return Persona{ID: "persona-1", Kind: KindPersonal, Status: StatusArchived, Identifiers: []Identifier{{Type: "passport", Value: "P-1"}}}, nil
		},
	})
	findings, err := service.CheckHealth(context.Background(), "persona-1")
	if err != nil {
		t.Fatalf("CheckHealth() error = %v", err)
	}
	if len(findings) != 1 || findings[0].Code != "persona_archived" || !findings[0].Blocking {
		t.Fatalf("health findings = %#v", findings)
	}
}

func TestCheckHealthSurfacesDependencyFindings(t *testing.T) {
	// [REQ:PSN-P1-005] document, mailbox, and OTP checks are composed before a flow starts.
	service := NewServiceWithHealth(FakeRepository{
		GetFunc: func(context.Context, string) (Persona, error) {
			return Persona{ID: "persona-1", Kind: KindPersonal, Status: StatusActive, Identifiers: []Identifier{{Type: "passport", Value: "P-1"}}}, nil
		},
	}, HealthProviderFunc(func(context.Context, Persona) ([]HealthFinding, error) {
		return []HealthFinding{
			{Code: "document_expired", Message: "passport validity has elapsed", Blocking: true},
			{Code: "mailbox_unreachable", Message: "controlled mailbox did not authenticate", Blocking: true},
			{Code: "otp_route_unreachable", Message: "registered OTP route is unavailable", Blocking: true},
		}, nil
	}))
	findings, err := service.CheckHealth(context.Background(), "persona-1")
	if err != nil || len(findings) != 3 {
		t.Fatalf("dependency health = %#v, %v", findings, err)
	}
}

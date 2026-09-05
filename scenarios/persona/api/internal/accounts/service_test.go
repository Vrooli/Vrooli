package accounts

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"persona/internal/handoffs"
	"persona/internal/journal"
	"persona/internal/personas"
)

type accountPersonaFunc func(context.Context, string) (personas.Persona, error)

func (f accountPersonaFunc) Get(ctx context.Context, id string) (personas.Persona, error) {
	return f(ctx, id)
}

type accountHandoffFunc func(context.Context, string) (handoffs.Handoff, error)

func (f accountHandoffFunc) Get(ctx context.Context, id string) (handoffs.Handoff, error) {
	return f(ctx, id)
}

type accountJournal struct{ entries []journal.Entry }

func (j *accountJournal) Append(_ context.Context, entry journal.Entry) (journal.Entry, error) {
	j.entries = append(j.entries, entry)
	return entry, nil
}

func (j *accountJournal) List(context.Context, string, int) ([]journal.Entry, error) {
	return j.entries, nil
}

func accountPersona() personas.Persona {
	return personas.Persona{ID: "persona-1", Kind: personas.KindPersonal, Status: personas.StatusActive}
}

func accountRepository() *FakeRepository {
	return &FakeRepository{
		LinkFunc: func(_ context.Context, link AccountLink) (AccountLink, error) {
			link.ID = "account-1"
			return link, nil
		},
		AddAddressFunc: func(_ context.Context, address Address) (Address, error) {
			address.ID = "address-1"
			return address, nil
		},
		GetAddressFunc: func(_ context.Context, personaID, id string) (Address, error) {
			return Address{ID: id, PersonaID: personaID, Label: "shipping", Country: "US"}, nil
		},
		AddObligationFunc: func(_ context.Context, obligation Obligation) (Obligation, error) {
			obligation.ID = "obligation-1"
			return obligation, nil
		},
		CancelObligationFunc: func(_ context.Context, id string) (Obligation, error) {
			return Obligation{ID: id, PersonaID: "persona-1", Cancelled: true}, nil
		},
	}
}

func TestAccountLinkRequiresRecoverySeamAndRecordsLinkage(t *testing.T) {
	// [REQ:PSN-P1-001] a linked account is useful for retirement only when its recovery seam is explicit.
	j := &accountJournal{}
	service := NewService(accountRepository(), accountPersonaFunc(func(context.Context, string) (personas.Persona, error) { return accountPersona(), nil }), accountHandoffFunc(func(context.Context, string) (handoffs.Handoff, error) { return handoffs.Handoff{}, nil }), j, nil)
	if _, err := service.Link(context.Background(), AccountInput{PersonaID: "persona-1", Site: "example.test", LoginSeam: "email"}); !errors.Is(err, ErrInvalidAccount) {
		t.Fatalf("incomplete account link error = %v", err)
	}
	link, err := service.Link(context.Background(), AccountInput{PersonaID: "persona-1", Site: "example.test", LoginSeam: "email", RecoveryPath: "controlled mailbox"})
	if err != nil || link.ID != "account-1" || link.RecoveryPath == "" {
		t.Fatalf("Link() = %#v, %v", link, err)
	}
	if len(j.entries) != 1 || j.entries[0].Verb != "account_linked" {
		t.Fatalf("linkage journal = %#v", j.entries)
	}
}

func TestAddressReleaseRequiresNamedSamePersonaTarget(t *testing.T) {
	// [REQ:PSN-P1-002] address release cannot infer a destination or cross persona boundary.
	repo := accountRepository()
	j := &accountJournal{}
	handoffResolver := accountHandoffFunc(func(_ context.Context, id string) (handoffs.Handoff, error) {
		if id == "other-persona-handoff" {
			return handoffs.Handoff{ID: id, PersonaID: "persona-2"}, nil
		}
		return handoffs.Handoff{ID: id, PersonaID: "persona-1"}, nil
	})
	service := NewService(repo, accountPersonaFunc(func(context.Context, string) (personas.Persona, error) { return accountPersona(), nil }), handoffResolver, j, nil)
	address, err := service.AddAddress(context.Background(), AddressInput{PersonaID: "persona-1", Address: Address{Label: "shipping", Country: "US"}})
	if err != nil {
		t.Fatalf("AddAddress() error = %v", err)
	}
	if _, err := service.ReleaseAddress(context.Background(), AddressReleaseInput{PersonaID: "persona-1", AddressID: address.ID}); !errors.Is(err, ErrAddressReleaseTarget) {
		t.Fatalf("unnamed release error = %v", err)
	}
	if _, err := service.ReleaseAddress(context.Background(), AddressReleaseInput{PersonaID: "persona-1", AddressID: address.ID, TargetKind: "handoff", TargetID: "other-persona-handoff"}); !errors.Is(err, ErrAddressReleaseTarget) {
		t.Fatalf("cross-persona release error = %v", err)
	}
	if _, err := service.ReleaseAddress(context.Background(), AddressReleaseInput{PersonaID: "persona-1", AddressID: address.ID, TargetKind: "resolution", TargetID: "resolution-1"}); err != nil {
		t.Fatalf("named resolution release error = %v", err)
	}
}

func TestObligationCarriesRenewalAndCancelPathWithoutMoney(t *testing.T) {
	// [REQ:PSN-P1-003] obligations track renewal/cancellation, while treasury owns money.
	service := NewService(accountRepository(), accountPersonaFunc(func(context.Context, string) (personas.Persona, error) { return accountPersona(), nil }), accountHandoffFunc(func(context.Context, string) (handoffs.Handoff, error) { return handoffs.Handoff{}, nil }), nil, nil)
	if _, err := service.AddObligation(context.Background(), ObligationInput{PersonaID: "persona-1", Description: "developer membership", CancelPath: "operator portal"}); !errors.Is(err, ErrInvalidObligation) {
		t.Fatalf("incomplete obligation error = %v", err)
	}
	obligation, err := service.AddObligation(context.Background(), ObligationInput{PersonaID: "persona-1", Description: "developer membership", RenewalAt: time.Now().Add(30 * 24 * time.Hour), CancelPath: "operator portal"})
	if err != nil || obligation.ID != "obligation-1" || obligation.RenewalAt.IsZero() {
		t.Fatalf("AddObligation() = %#v, %v", obligation, err)
	}
	cancelled, err := service.CancelObligation(context.Background(), obligation.ID)
	if err != nil || !cancelled.Cancelled {
		t.Fatalf("CancelObligation() = %#v, %v", cancelled, err)
	}
	if strings.Contains(strings.ToLower(Schema()), "amount") || strings.Contains(strings.ToLower(Schema()), "currency") || strings.Contains(strings.ToLower(Schema()), "budget") || strings.Contains(strings.ToLower(Schema()), "mandate") {
		t.Fatal("accounts schema contains treasury-owned money fields")
	}
}

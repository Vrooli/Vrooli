package accounts

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"
	"persona/internal/handoffs"
	"persona/internal/journal"
	"persona/internal/personas"
)

var (
	ErrMissingPersona       = errors.New("persona_id is required")
	ErrMissingAddress       = errors.New("address_id is required")
	ErrMissingObligation    = errors.New("obligation_id is required")
	ErrInvalidAccount       = errors.New("site, login seam, and recovery path are required")
	ErrInvalidAddress       = errors.New("address label and country are required")
	ErrInvalidObligation    = errors.New("description, renewal date, and cancel path are required")
	ErrAddressReleaseTarget = errors.New("address release requires a named handoff or resolution")
)

type (
	AccountLink struct {
		ID, PersonaID, Site, LoginSeam, RecoveryPath string
		CreatedAt                                    time.Time
	}
	Address struct {
		ID, PersonaID, Label, Line1, Line2, City, Region, PostalCode, Country string
		CreatedAt                                                             time.Time
	}
	Obligation struct {
		ID, PersonaID, AccountLinkID, Description string
		RenewalAt                                 time.Time
		CancelPath                                string
		Cancelled                                 bool
		CreatedAt                                 time.Time
	}
	PersonaResolver interface {
		Get(context.Context, string) (personas.Persona, error)
	}
	HandoffResolver interface {
		Get(context.Context, string) (handoffs.Handoff, error)
	}
	Service interface {
		Link(context.Context, AccountInput) (AccountLink, error)
		ListAccounts(context.Context, string) ([]AccountLink, error)
		AddAddress(context.Context, AddressInput) (Address, error)
		ListAddresses(context.Context, string) ([]Address, error)
		ReleaseAddress(context.Context, AddressReleaseInput) (Address, error)
		AddObligation(context.Context, ObligationInput) (Obligation, error)
		ListObligations(context.Context, string) ([]Obligation, error)
		CancelObligation(context.Context, string) (Obligation, error)
	}
	AccountInput struct{ PersonaID, Site, LoginSeam, RecoveryPath string }
	AddressInput struct {
		PersonaID string
		Address   Address
	}
	AddressReleaseInput struct{ PersonaID, AddressID, TargetKind, TargetID string }
	ObligationInput     struct {
		PersonaID, AccountLinkID, Description string
		RenewalAt                             time.Time
		CancelPath                            string
	}
	service struct {
		repo     Repository
		personas PersonaResolver
		handoffs HandoffResolver
		journal  journal.Service
		clock    schedule.Clock
	}
)

func NewService(repo Repository, p PersonaResolver, h HandoffResolver, j journal.Service, clock schedule.Clock) Service {
	if clock == nil {
		clock = schedule.System()
	}
	return &service{repo: repo, personas: p, handoffs: h, journal: j, clock: clock}
}

var _ Service = (*service)(nil)

func (s *service) Link(ctx context.Context, in AccountInput) (AccountLink, error) {
	if strings.TrimSpace(in.PersonaID) == "" {
		return AccountLink{}, ErrMissingPersona
	}
	if strings.TrimSpace(in.Site) == "" || strings.TrimSpace(in.LoginSeam) == "" || strings.TrimSpace(in.RecoveryPath) == "" {
		return AccountLink{}, ErrInvalidAccount
	}
	if _, err := s.personas.Get(ctx, in.PersonaID); err != nil {
		return AccountLink{}, err
	}
	link, err := s.repo.Link(ctx, AccountLink{PersonaID: in.PersonaID, Site: in.Site, LoginSeam: in.LoginSeam, RecoveryPath: in.RecoveryPath})
	if err == nil {
		s.record(ctx, in.PersonaID, "account_linked", map[string]string{"account_id": link.ID, "site": link.Site})
	}
	return link, err
}

func (s *service) ListAccounts(ctx context.Context, id string) ([]AccountLink, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrMissingPersona
	}
	return s.repo.ListAccounts(ctx, id)
}

func (s *service) AddAddress(ctx context.Context, in AddressInput) (Address, error) {
	if strings.TrimSpace(in.PersonaID) == "" {
		return Address{}, ErrMissingPersona
	}
	if strings.TrimSpace(in.Address.Label) == "" || strings.TrimSpace(in.Address.Country) == "" {
		return Address{}, ErrInvalidAddress
	}
	if _, err := s.personas.Get(ctx, in.PersonaID); err != nil {
		return Address{}, err
	}
	in.Address.PersonaID = in.PersonaID
	address, err := s.repo.AddAddress(ctx, in.Address)
	if err == nil {
		s.record(ctx, in.PersonaID, "address_added", map[string]string{"address_id": address.ID})
	}
	return address, err
}

func (s *service) ListAddresses(ctx context.Context, id string) ([]Address, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrMissingPersona
	}
	return s.repo.ListAddresses(ctx, id)
}

func (s *service) ReleaseAddress(ctx context.Context, in AddressReleaseInput) (Address, error) {
	if strings.TrimSpace(in.PersonaID) == "" {
		return Address{}, ErrMissingPersona
	}
	if strings.TrimSpace(in.AddressID) == "" {
		return Address{}, ErrMissingAddress
	}
	if (in.TargetKind != "handoff" && in.TargetKind != "resolution") || strings.TrimSpace(in.TargetID) == "" {
		return Address{}, ErrAddressReleaseTarget
	}
	address, err := s.repo.GetAddress(ctx, in.PersonaID, in.AddressID)
	if err != nil {
		return Address{}, err
	}
	if in.TargetKind == "handoff" {
		h, err := s.handoffs.Get(ctx, in.TargetID)
		if err != nil {
			return Address{}, err
		}
		if h.PersonaID != in.PersonaID {
			return Address{}, ErrAddressReleaseTarget
		}
	}
	s.record(ctx, in.PersonaID, "address_released", map[string]string{"address_id": address.ID, "target_kind": in.TargetKind, "target_id": in.TargetID})
	return address, nil
}

func (s *service) AddObligation(ctx context.Context, in ObligationInput) (Obligation, error) {
	if strings.TrimSpace(in.PersonaID) == "" {
		return Obligation{}, ErrMissingPersona
	}
	if strings.TrimSpace(in.Description) == "" || in.RenewalAt.IsZero() || strings.TrimSpace(in.CancelPath) == "" {
		return Obligation{}, ErrInvalidObligation
	}
	if _, err := s.personas.Get(ctx, in.PersonaID); err != nil {
		return Obligation{}, err
	}
	obligation, err := s.repo.AddObligation(ctx, Obligation{PersonaID: in.PersonaID, AccountLinkID: in.AccountLinkID, Description: in.Description, RenewalAt: in.RenewalAt, CancelPath: in.CancelPath})
	if err == nil {
		s.record(ctx, in.PersonaID, "obligation_added", map[string]string{"obligation_id": obligation.ID})
	}
	return obligation, err
}

func (s *service) ListObligations(ctx context.Context, id string) ([]Obligation, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrMissingPersona
	}
	return s.repo.ListObligations(ctx, id)
}

func (s *service) CancelObligation(ctx context.Context, id string) (Obligation, error) {
	if strings.TrimSpace(id) == "" {
		return Obligation{}, ErrMissingObligation
	}
	obligation, err := s.repo.CancelObligation(ctx, id)
	if err == nil {
		s.record(ctx, obligation.PersonaID, "obligation_cancelled", map[string]string{"obligation_id": id})
	}
	return obligation, err
}

func (s *service) record(ctx context.Context, personaID, verb string, details map[string]string) {
	if s.journal == nil {
		return
	}
	_, _ = s.journal.Append(ctx, journal.Entry{PersonaID: personaID, Actor: "operator", Verb: verb, Outcome: "granted", Details: details})
}
func Schema() string { return schemaSQL }

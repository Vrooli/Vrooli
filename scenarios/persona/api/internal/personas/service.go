package personas

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const defaultListLimit = 100

var (
	ErrMissingID       = errors.New("persona_id is required")
	ErrNotFound        = errors.New("persona not found")
	ErrMissingLegal    = errors.New("legal_basis is required")
	ErrInvalidKind     = errors.New("persona kind must be personal or business")
	ErrMissingIdentity = errors.New("persona requires at least one kind-specific identifier")
	ErrInvalidIdentity = errors.New("persona identifier is not valid for its kind")
	ErrImmutableBasis  = errors.New("legal basis is immutable")
	ErrAlreadyArchived = errors.New("persona is already archived")
)

type Kind string

const (
	KindPersonal Kind = "personal"
	KindBusiness Kind = "business"
)

type LegalBasis struct {
	SubjectID   string
	SubjectName string
	BasisType   string
}

type Identifier struct {
	Type  string
	Value string
}

type Status string

const (
	StatusActive   Status = "active"
	StatusArchived Status = "archived"
)

type Persona struct {
	ID          string
	Kind        Kind
	LegalBasis  LegalBasis
	DisplayName string
	Identifiers []Identifier
	Status      Status
	CreatedAt   time.Time
	ArchivedAt  *time.Time
}

type HealthFinding struct {
	Code     string
	Message  string
	Blocking bool
}

type HealthProvider interface {
	CheckHealth(context.Context, Persona) ([]HealthFinding, error)
}

type HealthProviderFunc func(context.Context, Persona) ([]HealthFinding, error)

func (f HealthProviderFunc) CheckHealth(ctx context.Context, persona Persona) ([]HealthFinding, error) {
	return f(ctx, persona)
}

type CreateInput struct {
	Kind        Kind
	LegalBasis  LegalBasis
	DisplayName string
	Identifiers []Identifier
}

type Service interface {
	Create(context.Context, CreateInput) (Persona, error)
	Get(context.Context, string) (Persona, error)
	List(context.Context, bool, int) ([]Persona, error)
	Archive(context.Context, string) (Persona, error)
	CheckHealth(context.Context, string) ([]HealthFinding, error)
}

type service struct {
	repo           Repository
	healthProvider HealthProvider
}

func NewService(repo Repository) Service { return NewServiceWithHealth(repo, nil) }

func NewServiceWithHealth(repo Repository, healthProvider HealthProvider) Service {
	return &service{repo: repo, healthProvider: healthProvider}
}

var _ Service = (*service)(nil)

func (s *service) Create(ctx context.Context, in CreateInput) (Persona, error) {
	if in.Kind != KindPersonal && in.Kind != KindBusiness {
		return Persona{}, ErrInvalidKind
	}
	if strings.TrimSpace(in.LegalBasis.SubjectID) == "" || strings.TrimSpace(in.LegalBasis.SubjectName) == "" || strings.TrimSpace(in.LegalBasis.BasisType) == "" {
		return Persona{}, ErrMissingLegal
	}
	if err := validateIdentifiers(in.Kind, in.Identifiers); err != nil {
		return Persona{}, err
	}
	if strings.TrimSpace(in.DisplayName) == "" {
		in.DisplayName = in.LegalBasis.SubjectName
	}
	return s.repo.Create(ctx, Persona{Kind: in.Kind, LegalBasis: in.LegalBasis, DisplayName: strings.TrimSpace(in.DisplayName), Identifiers: in.Identifiers, Status: StatusActive})
}

func (s *service) Get(ctx context.Context, id string) (Persona, error) {
	if strings.TrimSpace(id) == "" {
		return Persona{}, ErrMissingID
	}
	return s.repo.Get(ctx, id)
}

func (s *service) List(ctx context.Context, includeArchived bool, limit int) ([]Persona, error) {
	if limit <= 0 || limit > 1000 {
		limit = defaultListLimit
	}
	return s.repo.List(ctx, includeArchived, limit)
}

func (s *service) Archive(ctx context.Context, id string) (Persona, error) {
	if strings.TrimSpace(id) == "" {
		return Persona{}, ErrMissingID
	}
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return Persona{}, err
	}
	if p.Status == StatusArchived {
		return Persona{}, fmt.Errorf("%w: %s", ErrAlreadyArchived, id)
	}
	return s.repo.Archive(ctx, id)
}

func (s *service) CheckHealth(ctx context.Context, id string) ([]HealthFinding, error) {
	p, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	findings := make([]HealthFinding, 0)
	if p.Status == StatusArchived {
		findings = append(findings, HealthFinding{Code: "persona_archived", Message: "Persona is archived and cannot be used for new actions.", Blocking: true})
	}
	if err := validateIdentifiers(p.Kind, p.Identifiers); err != nil {
		findings = append(findings, HealthFinding{Code: "identifier_invalid", Message: err.Error(), Blocking: true})
	}
	if s.healthProvider != nil {
		dependencyFindings, err := s.healthProvider.CheckHealth(ctx, p)
		if err != nil {
			findings = append(findings, HealthFinding{Code: "health_provider_unavailable", Message: err.Error(), Blocking: true})
		} else {
			findings = append(findings, dependencyFindings...)
		}
	}
	return findings, nil
}

func validateIdentifiers(kind Kind, identifiers []Identifier) error {
	if len(identifiers) == 0 {
		return ErrMissingIdentity
	}
	allowed := map[string]bool{}
	switch kind {
	case KindPersonal:
		allowed = map[string]bool{"government_id": true, "national_id": true, "passport": true}
	case KindBusiness:
		allowed = map[string]bool{"business_registration": true, "tax_id": true, "duns": true}
	default:
		return ErrInvalidKind
	}
	for _, identifier := range identifiers {
		if strings.TrimSpace(identifier.Type) == "" || strings.TrimSpace(identifier.Value) == "" || !allowed[strings.ToLower(strings.TrimSpace(identifier.Type))] {
			return ErrInvalidIdentity
		}
	}
	return nil
}

// Schema returns the persona domain's SQL contribution.
func Schema() string { return schemaSQL }

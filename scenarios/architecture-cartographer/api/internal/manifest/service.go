package manifest

import (
	"context"
	"errors"
	"strings"
)

func errorsAsInvalid(err error, target *ErrInvalidManifest) bool {
	return errors.As(err, target)
}

// Service is the application-layer surface for manifest operations.
// ValidateManifest accepts a pre-parsed ManifestDefinition (used by
// internal callers and fixtures); ValidateSource is the Connect-facing
// entry point that handles raw YAML/JSON bytes.
type Service interface {
	// ValidateManifest accepts an already-parsed ManifestDefinition and
	// runs structural validation. Returns the manifest + diagnostics.
	// Successful structural validation persists the manifest via the
	// Repository so GetManifest can serve it.
	ValidateManifest(ctx context.Context, in ManifestDefinition) (ManifestDefinition, []Diagnostic, error)

	// ValidateSource parses raw manifest bytes, runs structural
	// validation, and persists the result on success. hint may be
	// blank to trigger detection from the first non-whitespace byte.
	// scenario, when non-empty, overrides any scenario field in the
	// source.
	ValidateSource(ctx context.Context, scenario string, source []byte, hint ContentType) (ManifestDefinition, []Diagnostic, error)

	GetManifest(ctx context.Context, scenario string) (ManifestDefinition, error)
	ListDomains(ctx context.Context, scenario string) ([]DomainSpec, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

var _ Service = (*service)(nil)

func (s *service) ValidateManifest(ctx context.Context, in ManifestDefinition) (ManifestDefinition, []Diagnostic, error) {
	diagnostics := Validate(in)
	hasError := false
	for _, d := range diagnostics {
		if d.Severity == DiagnosticSeverityError {
			hasError = true
			break
		}
	}
	if hasError {
		return in, diagnostics, ErrInvalidManifest{Diagnostics: diagnostics}
	}
	// Default version when caller left it blank.
	if in.Version == ManifestVersionUnspecified {
		in.Version = ManifestVersionV1
	}
	persisted, err := s.repo.SaveManifest(ctx, in)
	if err != nil {
		return in, diagnostics, err
	}
	return persisted, diagnostics, nil
}

func (s *service) ValidateSource(ctx context.Context, scenario string, source []byte, hint ContentType) (ManifestDefinition, []Diagnostic, error) {
	parsed, _, parseDiags, parseErr := Parse(source, hint)
	if parseErr != nil {
		return ManifestDefinition{}, parseDiags, ErrInvalidManifest{Diagnostics: parseDiags}
	}
	if scenario = strings.TrimSpace(scenario); scenario != "" {
		parsed.Scenario = scenario
	}
	m, structDiags, err := s.ValidateManifest(ctx, parsed)
	all := append(append([]Diagnostic(nil), parseDiags...), structDiags...)
	if err != nil {
		var typed ErrInvalidManifest
		if errorsAsInvalid(err, &typed) {
			return m, all, ErrInvalidManifest{Diagnostics: all}
		}
		return m, all, err
	}
	return m, all, nil
}

func (s *service) GetManifest(ctx context.Context, scenario string) (ManifestDefinition, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return ManifestDefinition{}, ErrManifestNotFound{Scenario: scenario}
	}
	return s.repo.GetManifest(ctx, scenario)
}

func (s *service) ListDomains(ctx context.Context, scenario string) ([]DomainSpec, error) {
	m, err := s.GetManifest(ctx, scenario)
	if err != nil {
		return nil, err
	}
	out := make([]DomainSpec, len(m.Domains))
	copy(out, m.Domains)
	return out, nil
}

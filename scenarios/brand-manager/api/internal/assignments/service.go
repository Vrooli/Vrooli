package assignments

import (
	"context"
	"errors"
	"log"
	"strings"
)

// Service is the application-layer surface the assignments handlers depend on.
// Owns validation, brand-version resolution (via the BrandResolver seam), and
// idempotent unassign semantics. The handler is intentionally thin around it:
// decode → call service → translate errors.
type Service interface {
	// List returns assignments, optionally filtered by brandID, newest-applied
	// first.
	List(ctx context.Context, brandID string) ([]Assignment, error)

	// Assign validates the input (brand_id + scenario_name required), resolves
	// the brand's current version (rejecting an unknown brand), and upserts the
	// link keyed by scenario_name. Re-assigning replaces the prior link and
	// re-pins the version.
	Assign(ctx context.Context, in AssignInput) (Assignment, error)

	// ScenarioStatus returns the scenario's branding status. A scenario with no
	// assignment yields HasBrand=false (not an error).
	ScenarioStatus(ctx context.Context, scenarioName string) (ScenarioStatus, error)

	// Unassign removes a scenario's brand assignment. Idempotent: unassigning a
	// scenario with no brand returns nil.
	Unassign(ctx context.Context, scenarioName string) error
}

type service struct {
	repo   Repository
	brands BrandResolver
	logger *log.Logger
}

// NewService constructs the production Service. brands resolves the brand
// version at assignment time. A nil logger defaults to log.Default().
func NewService(repo Repository, brands BrandResolver, logger *log.Logger) Service {
	if logger == nil {
		logger = log.Default()
	}
	return &service{repo: repo, brands: brands, logger: logger}
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) List(ctx context.Context, brandID string) ([]Assignment, error) {
	return s.repo.ListByBrand(ctx, strings.TrimSpace(brandID))
}

func (s *service) Assign(ctx context.Context, in AssignInput) (Assignment, error) {
	brandID := strings.TrimSpace(in.BrandID)
	if brandID == "" {
		return Assignment{}, ErrInvalidAssignment{Field: "brand_id", Reason: "required"}
	}
	scenario := strings.TrimSpace(in.ScenarioName)
	if scenario == "" {
		return Assignment{}, ErrInvalidAssignment{Field: "scenario_name", Reason: "required"}
	}

	version, ok, err := s.brands.BrandVersion(ctx, brandID)
	if err != nil {
		return Assignment{}, err
	}
	if !ok {
		return Assignment{}, ErrInvalidAssignment{Field: "brand_id", Reason: "brand not found"}
	}

	return s.repo.Upsert(ctx, Assignment{
		BrandID:      brandID,
		ScenarioName: scenario,
		BrandVersion: version,
		Elements:     normalizeElements(in.Elements),
	})
}

func (s *service) ScenarioStatus(ctx context.Context, scenarioName string) (ScenarioStatus, error) {
	scenario := strings.TrimSpace(scenarioName)
	if scenario == "" {
		return ScenarioStatus{}, ErrInvalidAssignment{Field: "scenario_name", Reason: "required"}
	}
	a, err := s.repo.GetByScenario(ctx, scenario)
	if err != nil {
		var notFound ErrAssignmentNotFound
		if errors.As(err, &notFound) {
			return StatusUnassigned(scenario), nil
		}
		return ScenarioStatus{}, err
	}
	return StatusFromAssignment(a), nil
}

func (s *service) Unassign(ctx context.Context, scenarioName string) error {
	scenario := strings.TrimSpace(scenarioName)
	if scenario == "" {
		return ErrInvalidAssignment{Field: "scenario_name", Reason: "required"}
	}
	err := s.repo.DeleteByScenario(ctx, scenario)
	if err == nil {
		return nil
	}
	// Idempotent: unassigning a scenario with no brand is a success.
	var notFound ErrAssignmentNotFound
	if errors.As(err, &notFound) {
		return nil
	}
	return err
}

// normalizeElements trims, drops empties, and returns a fresh slice (never the
// caller's backing array). A nil/empty input yields nil so the JSON column
// stores an empty array consistently.
func normalizeElements(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, e := range in {
		if trimmed := strings.TrimSpace(e); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

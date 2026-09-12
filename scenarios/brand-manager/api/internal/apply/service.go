package apply

import (
	"context"
	"log"
	"path"
	"strings"
)

// Managed file locations within a scenario's source tree. Stable so a re-apply
// overwrites the same files (convergent, no accumulation).
const (
	brandCSSPath = "ui/src/styles/brand.css"
	manifestPath = "ui/public/manifest.json"
	publicDir    = "ui/public"
)

// Service is the application-layer surface the apply handlers depend on. It
// validates input, plans the per-element file writes, and — for Apply — performs
// them and records the assignment. The handler is intentionally thin around it:
// decode → call service → translate errors.
type Service interface {
	// Preview computes exactly what Apply would write WITHOUT touching the
	// filesystem or recording an assignment. The returned Result has DryRun=true.
	Preview(ctx context.Context, in Request) (Result, error)

	// Apply writes the requested brand elements into the scenario's source tree
	// and, when anything was written, records the brand↔scenario assignment. The
	// returned Result has DryRun=false.
	Apply(ctx context.Context, in Request) (Result, error)
}

type service struct {
	brands      BrandStore
	assets      AssetStore
	assignments AssignmentRecorder
	workspace   Workspace
	logger      *log.Logger
}

// NewService constructs the production Service. A nil logger defaults to
// log.Default().
func NewService(brands BrandStore, assets AssetStore, assignments AssignmentRecorder, workspace Workspace, logger *log.Logger) Service {
	if logger == nil {
		logger = log.Default()
	}
	return &service{brands: brands, assets: assets, assignments: assignments, workspace: workspace, logger: logger}
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) Preview(ctx context.Context, in Request) (Result, error) {
	return s.run(ctx, in, false)
}

func (s *service) Apply(ctx context.Context, in Request) (Result, error) {
	return s.run(ctx, in, true)
}

// run is the shared plan-and-maybe-write core. write=false is a pure preview
// (no filesystem mutation, no assignment); write=true performs the writes and
// records the assignment when at least one element was applied.
func (s *service) run(ctx context.Context, in Request, write bool) (Result, error) {
	brandID := strings.TrimSpace(in.BrandID)
	if brandID == "" {
		return Result{}, ErrInvalidApply{Field: "brand_id", Reason: "required"}
	}
	scenario := strings.TrimSpace(in.Scenario)
	if scenario == "" {
		return Result{}, ErrInvalidApply{Field: "scenario_name", Reason: "required"}
	}

	brand, err := s.brands.Get(ctx, brandID)
	if err != nil {
		return Result{}, err
	}

	exists, err := s.workspace.ScenarioExists(ctx, scenario)
	if err != nil {
		return Result{}, err
	}
	if !exists {
		return Result{}, ErrScenarioNotFound{Scenario: scenario}
	}

	elements := normalizeElements(in.Elements)

	result := Result{
		Scenario:     scenario,
		BrandID:      brand.ID,
		BrandVersion: brand.Version,
		DryRun:       !write,
	}
	var appliedElements []string

	for _, elem := range elements {
		actions, skip, err := s.applyElement(ctx, brand, scenario, elem, write)
		if err != nil {
			return Result{}, err
		}
		switch {
		case skip != nil:
			result.Skipped = append(result.Skipped, *skip)
		case len(actions) > 0:
			result.Applied = append(result.Applied, actions...)
			appliedElements = append(appliedElements, elem)
		}
	}

	if write && len(appliedElements) > 0 {
		if err := s.assignments.Record(ctx, brand.ID, scenario, appliedElements); err != nil {
			return Result{}, err
		}
	}

	return result, nil
}

// applyElement plans (and, when write, performs) a single element's writes.
// Returns either a non-empty action list, a non-nil skip, or an error for a
// genuine filesystem/IO failure (a missing facet/asset is a skip, not an error).
// Most elements produce exactly one action; ElementIcons can produce several.
func (s *service) applyElement(ctx context.Context, brand BrandView, scenario, element string, write bool) ([]Action, *Skip, error) {
	switch element {
	case ElementColors:
		return one(s.applyColors(ctx, brand, scenario, write))
	case ElementTypography:
		return one(s.applyTypography(ctx, brand, scenario, write))
	case ElementIdentity:
		return one(s.applyIdentity(ctx, brand, scenario, write))
	case ElementIcons:
		return s.applyIcons(ctx, brand, scenario, write)
	case ElementFavicon, ElementLogo:
		return one(s.applyAsset(ctx, brand, scenario, element, write))
	default:
		return nil, &Skip{Element: element, Reason: "unknown element"}, nil
	}
}

// one adapts a single-action element helper to the []Action shape.
func one(action *Action, skip *Skip, err error) ([]Action, *Skip, error) {
	if err != nil || skip != nil || action == nil {
		return nil, skip, err
	}
	return []Action{*action}, nil, nil
}

func (s *service) applyColors(ctx context.Context, brand BrandView, scenario string, write bool) (*Action, *Skip, error) {
	if !brand.Colors.HasAny() {
		return nil, &Skip{Element: ElementColors, Reason: "no colors defined"}, nil
	}
	if write {
		if err := s.workspace.WriteFile(ctx, scenario, brandCSSPath, []byte(generateColorCSS(brand.Colors))); err != nil {
			return nil, nil, err
		}
	}
	return &Action{Type: ActionCSS, File: brandCSSPath, Element: ElementColors}, nil, nil
}

func (s *service) applyTypography(ctx context.Context, brand BrandView, scenario string, write bool) (*Action, *Skip, error) {
	if !brand.Typography.HasAny() {
		return nil, &Skip{Element: ElementTypography, Reason: "no typography defined"}, nil
	}
	if write {
		// Append to the managed CSS file so colors (written first in AllElements
		// order) and typography share one :root stylesheet.
		existing, err := s.workspace.ReadFile(ctx, scenario, brandCSSPath)
		if err != nil {
			return nil, nil, err
		}
		combined := string(existing)
		if combined != "" && !strings.HasSuffix(combined, "\n") {
			combined += "\n"
		}
		combined += generateTypographyCSS(brand.Typography)
		if err := s.workspace.WriteFile(ctx, scenario, brandCSSPath, []byte(combined)); err != nil {
			return nil, nil, err
		}
	}
	return &Action{Type: ActionCSS, File: brandCSSPath, Element: ElementTypography}, nil, nil
}

func (s *service) applyIdentity(ctx context.Context, brand BrandView, scenario string, write bool) (*Action, *Skip, error) {
	if brand.DisplayName == "" && brand.Tagline == "" {
		return nil, &Skip{Element: ElementIdentity, Reason: "no identity defined"}, nil
	}
	if write {
		existing, err := s.workspace.ReadFile(ctx, scenario, manifestPath)
		if err != nil {
			return nil, nil, err
		}
		merged, err := mergeManifest(existing, brand)
		if err != nil {
			return nil, nil, err
		}
		if err := s.workspace.WriteFile(ctx, scenario, manifestPath, merged); err != nil {
			return nil, nil, err
		}
	}
	return &Action{Type: ActionJSON, File: manifestPath, Element: ElementIdentity}, nil, nil
}

func (s *service) applyAsset(ctx context.Context, brand BrandView, scenario, kind string, write bool) (*Action, *Skip, error) {
	content, found, err := s.assets.Read(ctx, brand.ID, kind)
	if err != nil {
		return nil, nil, err
	}
	if !found {
		return nil, &Skip{Element: kind, Reason: "no " + kind + " asset"}, nil
	}
	rel := path.Join(publicDir, content.Filename)
	if write {
		if err := s.workspace.WriteFile(ctx, scenario, rel, content.Bytes); err != nil {
			return nil, nil, err
		}
	}
	return &Action{Type: ActionAsset, File: rel, Element: kind}, nil, nil
}

// normalizeElements lower-cases and trims the requested elements, dropping
// empties. An empty request expands to AllElements (apply everything).
func normalizeElements(in []string) []string {
	if len(in) == 0 {
		return append([]string(nil), AllElements...)
	}
	out := make([]string, 0, len(in))
	for _, e := range in {
		if trimmed := strings.ToLower(strings.TrimSpace(e)); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return append([]string(nil), AllElements...)
	}
	return out
}

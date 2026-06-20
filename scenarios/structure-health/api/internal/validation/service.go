// Package validation is structure-health's core engine. It reconciles a
// scenario's code-facts ground truth (actual surfaces/languages/frameworks)
// against its declared service.json intent and emits profile-aware skeleton and
// lifecycle-wiring findings.
//
// Phase 2 wires real profile detection + intent parsing + the reconcile model;
// the rule packs that turn the reconcile into findings land in later phases.
package validation

import (
	"context"
	"fmt"

	"structure-health/internal/autofix"
	"structure-health/internal/intent"
	"structure-health/internal/packs"
	"structure-health/internal/packs/scan"
	"structure-health/internal/profile"
	"structure-health/internal/reconcile"
	"structure-health/internal/rules"

	"github.com/vrooli/maturity-go/assessment"
)

// Request is the engine input.
type Request struct {
	Scenario         string
	Path             string
	IncludeExecution bool
}

// DetectedProfile is the language/framework shape detected from code facts.
type DetectedProfile struct {
	ID              string
	BackendLanguage string
	UIFramework     string
	Recognized      bool
	Evidence        []string
}

// SurfaceReconcile pairs a surface's declared intent with its detected actual.
type SurfaceReconcile struct {
	Surface        string
	Kind           string
	Declared       bool
	Actual         bool
	DeclaredDetail string
	ActualDetail   string
}

// Finding is structure-health's native finding shape.
type Finding struct {
	Code             string
	Severity         string
	Title            string
	Message          string
	Location         string
	Remediation      string
	Surface          string
	AutofixAvailable bool
	FixClass         string
}

// Response is the engine output.
type Response struct {
	RunID          string
	Status         string
	Summary        string
	Scenario       string
	TargetPath     string
	DegradedReason string
	Profile        DetectedProfile
	Surfaces       []SurfaceReconcile
	Findings       []Finding
	NextSteps      []string
}

// Service holds the engine's collaborators.
type Service struct {
	// Spec is the loaded .vrooli/maturity.json maturity spec.
	Spec *assessment.Spec
	// Facts is the code-facts intake seam; defaults to a live CodeFactsClient.
	Facts profile.Describer
}

// New constructs a Service with defaults.
func New() *Service { return &Service{} }

// Validate reconciles the target scenario and returns its structure findings.
func (s *Service) Validate(ctx context.Context, req Request) (Response, error) {
	describer := s.Facts
	if describer == nil {
		describer = profile.CodeFactsClient{}
	}
	facts, err := describer.Describe(ctx, req.Scenario, req.Path)
	if err != nil {
		return Response{}, err
	}
	p := profile.Derive(facts)

	resp := Response{
		Scenario:       facts.Scenario,
		TargetPath:     facts.RootPath,
		DegradedReason: facts.DegradedReason,
		Profile: DetectedProfile{
			ID:              p.ID,
			BackendLanguage: p.BackendLanguage,
			UIFramework:     p.UIFramework,
			Recognized:      p.Recognized,
			Evidence:        p.Evidence,
		},
	}

	in, ierr := intent.Load(facts.RootPath)
	serviceJSONReadable := ierr == nil
	if ierr != nil {
		resp.DegradedReason = appendReason(resp.DegradedReason, fmt.Sprintf("service.json unreadable: %v", ierr))
	}
	model := reconcile.Build(facts.Scenario, facts.RootPath, in, p)
	for _, st := range model.Surfaces {
		resp.Surfaces = append(resp.Surfaces, SurfaceReconcile{
			Surface:        st.Surface,
			Kind:           st.Kind,
			Declared:       st.Declared,
			Actual:         st.Actual,
			DeclaredDetail: st.DeclaredDetail,
			ActualDetail:   st.ActualDetail,
		})
	}

	findings := rules.Evaluate(rules.Input{Model: model, ServiceJSONReadable: serviceJSONReadable})
	// Profile-keyed conformance packs (migrated scenario-auditor structure/
	// config/ui rules). The default profile enforces; an unrecognized profile
	// gets advisory findings only.
	if sc, scanErr := scan.Build(facts.Scenario, facts.RootPath); scanErr == nil {
		findings = append(findings, packs.Evaluate(p.ID, p.Recognized, sc)...)
	} else {
		resp.DegradedReason = appendReason(resp.DegradedReason, fmt.Sprintf("conformance scan failed: %v", scanErr))
	}
	errCount, warnCount := 0, 0
	for _, f := range findings {
		switch f.Severity {
		case "error":
			errCount++
		case "warning", "warn":
			warnCount++
		}
		fixClass := autofix.FixClassFor(f.Code)
		resp.Findings = append(resp.Findings, Finding{
			Code:             f.Code,
			Severity:         f.Severity,
			Title:            f.Title,
			Message:          f.Message,
			Location:         f.Location,
			Remediation:      f.Remediation,
			Surface:          f.Surface,
			FixClass:         string(fixClass),
			AutofixAvailable: fixClass.Autofixable() && autofix.CanFix(facts.RootPath, f.Code, f.Location),
		})
	}

	resp.Status = "PASSED"
	if errCount > 0 {
		resp.Status = "FAILED"
	}
	resp.Summary = fmt.Sprintf("profile %s; %d surface(s); %d error(s), %d warning(s)", p.ID, len(resp.Surfaces), errCount, warnCount)
	if len(findings) == 0 {
		resp.NextSteps = []string{"structure and lifecycle wiring are conformant"}
	} else {
		resp.NextSteps = []string{"run `structure-health fix-config run " + facts.Scenario + "` to preview auto-fixable remediations (add --apply to write them)"}
	}
	return resp, nil
}

// PreviewFix resolves the target scenario root and returns the deterministic
// auto-fix candidates for the requested rules (all auto-fixable when empty)
// without writing anything.
func (s *Service) PreviewFix(ctx context.Context, scenario, path string, ruleIDs []string) (string, []autofix.Candidate, error) {
	root, resolved, err := s.resolveRoot(ctx, scenario, path)
	if err != nil {
		return "", nil, err
	}
	candidates, err := autofix.Preview(root, ruleIDs)
	return resolved, candidates, err
}

// ApplyFix resolves the target scenario root and applies the deterministic
// auto-fix candidates for the requested rules, returning what changed.
func (s *Service) ApplyFix(ctx context.Context, scenario, path string, ruleIDs []string) (string, []autofix.Candidate, error) {
	root, resolved, err := s.resolveRoot(ctx, scenario, path)
	if err != nil {
		return "", nil, err
	}
	candidates, err := autofix.Apply(root, ruleIDs)
	return resolved, candidates, err
}

// resolveRoot resolves the scenario's on-disk root via the code-facts seam,
// returning the root path and the resolved scenario slug.
func (s *Service) resolveRoot(ctx context.Context, scenario, path string) (root, resolvedScenario string, err error) {
	describer := s.Facts
	if describer == nil {
		describer = profile.CodeFactsClient{}
	}
	facts, err := describer.Describe(ctx, scenario, path)
	if err != nil {
		return "", "", err
	}
	if facts.RootPath == "" {
		return "", "", fmt.Errorf("could not resolve scenario root for %q", scenario)
	}
	return facts.RootPath, facts.Scenario, nil
}

func appendReason(existing, addition string) string {
	if existing == "" {
		return addition
	}
	return existing + "; " + addition
}

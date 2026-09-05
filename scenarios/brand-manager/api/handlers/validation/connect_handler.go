// Package validation exposes brand-manager's ScenarioValidationService handler:
// the served Connect-RPC surface test-genie's `branding` delegated phase calls.
// It wraps the transport-agnostic branding scan (internal/validation) in the
// shared scenario-validation contract + maturity assessment.
package validation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/maturity-go/assessment"
	repocontract "github.com/vrooli/repo-contract-go"
	brandingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/validation"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/structpb"

	bscan "brand-manager/internal/validation"
)

// Handler implements scenariovalidationconnect.ScenarioValidationServiceHandler.
type Handler struct {
	// repoRoot is the resolved repository root used to locate target scenarios
	// and brand-manager's own maturity spec. Resolved once at construction.
	repoRoot string
}

// NewHandler resolves the repo root and returns a branding validation handler.
// A resolution failure is non-fatal: the handler still serves but reports an
// error per request (so a misconfigured environment degrades, not crashes).
func NewHandler() *Handler {
	root := ""
	if _, repoRoot, err := repocontract.LoadDefaultFromEnvOrCWD(); err == nil {
		root = repoRoot
	}
	return &Handler{repoRoot: root}
}

// resolveTarget turns a (scenario, path) request into a concrete scenario name
// and readable root directory.
func (h *Handler) resolveTarget(scenario, path string) (string, string, error) {
	scenario = strings.TrimSpace(scenario)
	path = strings.TrimSpace(path)
	if scenario == "" && path == "" {
		return "", "", fmt.Errorf("scenario or path is required")
	}
	if path != "" {
		if scenario == "" {
			scenario = filepath.Base(filepath.Clean(path))
		}
		return scenario, path, nil
	}
	if h.repoRoot == "" {
		return "", "", fmt.Errorf("repository root could not be resolved to locate scenario %q", scenario)
	}
	target := filepath.Join(h.repoRoot, "scenarios", scenario)
	if info, err := os.Stat(target); err != nil || !info.IsDir() {
		return "", "", fmt.Errorf("scenario %q not found at %s", scenario, target)
	}
	return scenario, target, nil
}

func (h *Handler) ValidateScenario(
	ctx context.Context,
	req *connect.Request[scenariovalidationv1.ValidateScenarioRequest],
) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	scenario, root, err := h.resolveTarget(req.Msg.GetScenario(), req.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	spec, err := h.loadSpec()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	collector := metrics.Start()
	scan := bscan.ScanScenario(scenario, root)
	native := scanToProto(scan)

	maturity, err := bscan.BuildMaturityAssessment(scenario, scan.Findings, *spec)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build branding assessment: %w", err))
	}
	native.Assessment = maturity

	execMetrics := collector.Stop()
	resp, err := assessment.BuildValidationResponse(scenario, maturity, native, execMetrics)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build validation response: %w", err))
	}
	return connect.NewResponse(resp), nil
}

func (h *Handler) PreviewFix(
	ctx context.Context,
	req *connect.Request[scenariovalidationv1.FixRequest],
) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return h.fix(req.Msg, false)
}

func (h *Handler) ApplyFix(
	ctx context.Context,
	req *connect.Request[scenariovalidationv1.FixRequest],
) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return h.fix(req.Msg, true)
}

func (h *Handler) fix(req *scenariovalidationv1.FixRequest, apply bool) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	scenario, root, err := h.resolveTarget(req.GetScenario(), req.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	candidates, messages, err := bscan.BuildFixCandidates(root, req.GetRuleIds(), apply)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*scenariovalidationv1.FixCandidate, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, &scenariovalidationv1.FixCandidate{
			RuleId:      c.RuleID,
			FilePath:    c.FilePath,
			Description: c.Description,
			Before:      c.Before,
			After:       c.After,
			Applied:     c.Applied,
		})
	}
	return connect.NewResponse(&scenariovalidationv1.FixResponse{
		Scenario: scenario,
		// A successful ApplyFix call is not itself evidence of a write. The shared
		// contract reserves applied=true for responses that carry at least one
		// candidate actually written to disk.
		Applied:    apply && len(out) > 0,
		Candidates: out,
		Messages:   messages,
	}), nil
}

func (h *Handler) loadSpec() (*assessment.Spec, error) {
	if h.repoRoot == "" {
		return nil, errors.New("repository root could not be resolved to load the branding maturity spec")
	}
	return assessment.LoadSpecFromScenario(filepath.Join(h.repoRoot, "scenarios", "brand-manager"))
}

// scanToProto packs a branding scan into its native proto detail (assessment is
// attached by the caller after it is built).
func scanToProto(scan *bscan.ScanResult) *brandingv1.BrandingScanResponse {
	status := "passed"
	if len(scan.Findings) > 0 {
		status = "issues_found"
	}
	var summary brandingv1.BrandingScanSummary
	out := make([]*brandingv1.BrandingFinding, 0, len(scan.Findings))
	for _, f := range scan.Findings {
		summary.TotalFindings++
		if f.AutofixAvailable {
			summary.Autofixable++
		}
		switch f.Severity {
		case bscan.SeverityError:
			summary.Errors++
		case bscan.SeverityWarning:
			summary.Warnings++
		case bscan.SeverityInfo:
			summary.Infos++
		}
		out = append(out, &brandingv1.BrandingFinding{
			Id:                     findingID(scan.Scenario, f.RuleID, f.FilePath),
			RuleId:                 f.RuleID,
			Scenario:               scan.Scenario,
			FilePath:               f.FilePath,
			Severity:               string(f.Severity),
			Title:                  f.Title,
			Description:            f.Description,
			Evidence:               evidenceToStruct(f.Evidence),
			WhyItMatters:           f.WhyItMatters,
			RecommendedRemediation: f.RecommendedRemediation,
			AutofixAvailable:       f.AutofixAvailable,
		})
	}
	return &brandingv1.BrandingScanResponse{
		Scenario: scan.Scenario,
		Status:   status,
		Findings: out,
		Summary:  &summary,
	}
}

func findingID(scenario, rule, loc string) string {
	return fmt.Sprintf("branding:%s:%s:%s", scenario, rule, loc)
}

func evidenceToStruct(evidence map[string]any) *structpb.Struct {
	if len(evidence) == 0 {
		return &structpb.Struct{}
	}
	payload, err := json.Marshal(evidence)
	if err != nil {
		return &structpb.Struct{}
	}
	var normalized map[string]any
	if err := json.Unmarshal(payload, &normalized); err != nil {
		return &structpb.Struct{}
	}
	s, err := structpb.NewStruct(normalized)
	if err != nil {
		return &structpb.Struct{}
	}
	return s
}

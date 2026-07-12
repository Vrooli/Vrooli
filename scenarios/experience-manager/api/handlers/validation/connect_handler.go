package validation

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/metrics"

	localassessment "experience-manager/internal/assessment"
	"experience-manager/internal/attestation"
	localautofix "experience-manager/internal/autofix"
	"experience-manager/internal/checks"
	"experience-manager/internal/envx"
	"experience-manager/internal/fleet"
	"experience-manager/internal/spec"

	maturity "github.com/vrooli/maturity-go/assessment"
	"github.com/vrooli/maturity-go/autofix"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	contractv1 "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// Engine is the validation engine seam implemented by internal/checks.Engine.
type Engine interface {
	ValidateScenario(ctx context.Context, scenario, path string) (spec.Report, error)
}

type Deps struct {
	Logger                *log.Logger
	Engine                Engine
	Builder               *localassessment.Builder
	Env                   envx.Env
	Fixers                *autofix.Registry
	RepoRoot              string
	Environment           *commonv1.CaptureEnvironment
	AttestationRepository attestation.Repository
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	if d.Env == nil {
		d.Env = envx.OS{}
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) listFleet(ctx context.Context, repoRoot string) (*contractv1.ListFleetResponse, error) {
	root := strings.TrimSpace(repoRoot)
	if root == "" {
		root = h.deps.RepoRoot
	}
	summary, err := fleet.Sweep(ctx, root)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("sweep experience fleet: %w", err))
	}
	resp := &contractv1.ListFleetResponse{
		ScenarioCount:       int32(summary.ScenarioCount),
		WithExperienceCount: int32(summary.WithExperienceCount),
		TotalPages:          int32(summary.TotalPages),
	}
	for _, row := range summary.Scenarios {
		resp.Scenarios = append(resp.Scenarios, &contractv1.FleetScenario{
			Scenario:      row.Scenario,
			HasExperience: row.HasExperience,
			MaxDepth:      fmt.Sprintf("L%d", row.MaxDepth),
			MaxDepthValue: int32(row.MaxDepth),
			PageCount:     int32(row.PageCount),
			FindingCount:  int32(row.FindingCount),
			DebtScore:     int32(row.DebtScore),
			Status:        row.Status,
		})
	}
	return resp, nil
}

func (h *connectHandler) appendAttestation(ctx context.Context, req *contractv1.AppendAttestationRequest) (*contractv1.AppendAttestationResponse, error) {
	if h.deps.AttestationRepository == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("attestation repository is not configured"))
	}
	now := time.Now().UTC()
	a := attestation.Attestation{
		ID:        fmt.Sprintf("att-%s-%s-%s-%d", req.GetScenario(), req.GetPage(), req.GetClaim(), now.UnixNano()),
		Scenario:  strings.TrimSpace(req.GetScenario()),
		PageID:    strings.TrimSpace(req.GetPage()),
		ClaimID:   strings.TrimSpace(req.GetClaim()),
		Author:    strings.TrimSpace(req.GetAuthor()),
		Rationale: strings.TrimSpace(req.GetRationale()),
		ExpiresAt: strings.TrimSpace(req.GetExpiresAt()),
		CreatedAt: now.Format(time.RFC3339Nano),
	}
	if _, err := time.Parse(time.RFC3339, a.ExpiresAt); err != nil {
		if _, nanoErr := time.Parse(time.RFC3339Nano, a.ExpiresAt); nanoErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("expires_at must be RFC3339: %w", err))
		}
	}
	if err := h.deps.AttestationRepository.AppendAttestation(ctx, a); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return &contractv1.AppendAttestationResponse{Attestation: protoAttestation(a)}, nil
}

func (h *connectHandler) scaffoldCases(req *contractv1.ScaffoldCasesRequest) (*contractv1.ScaffoldCasesResponse, error) {
	root, scenario, err := h.resolveFixTarget(req.GetScenario(), req.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	reg := h.deps.Fixers
	if reg == nil {
		reg = localautofix.NewRegistry()
	}
	candidates, err := reg.Preview(root, []string{localautofix.RuleCaseScaffold})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("preview BAS case scaffolds: %w", err))
	}
	applied := false
	if !req.GetDryRun() {
		candidates, err = localautofix.ApplySequential(reg, root, []string{localautofix.RuleCaseScaffold})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("apply BAS case scaffolds: %w", err))
		}
		applied = true
	}
	resp := &contractv1.ScaffoldCasesResponse{
		Scenario: scenario,
		Applied:  applied,
	}
	for _, c := range candidates {
		action := "update"
		if c.Before == "" {
			action = "create"
		}
		resp.Diffs = append(resp.Diffs, &contractv1.FileDiff{
			Path:   c.FilePath,
			Action: action,
			Before: c.Before,
			After:  c.After,
		})
		resp.Messages = append(resp.Messages, c.Description)
	}
	if len(candidates) == 0 {
		resp.Messages = append(resp.Messages, "No BAS case scaffolds needed.")
	}
	return resp, nil
}

// validate is the single shared pipeline wrapper for the native and delegated
// service mounts.
func (h *connectHandler) validate(ctx context.Context, scenario, path string) (spec.Report, *commonv1.MaturityAssessment, *commonv1.ExecutionMetrics, error) {
	collector := metrics.Start()
	if h.deps.Engine == nil {
		return spec.Report{}, nil, nil, connect.NewError(connect.CodeInternal, fmt.Errorf("experience validation engine is not configured"))
	}
	report, err := h.deps.Engine.ValidateScenario(ctx, scenario, path)
	if err != nil {
		return report, nil, nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	builder := h.deps.Builder
	if builder == nil {
		var buildErr error
		builder, buildErr = localassessment.NewBuilder(localassessment.DefaultSpec())
		if buildErr != nil {
			return report, nil, nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build maturity mapper: %w", buildErr))
		}
	}
	assessment, err := builder.Build(report.Scenario, report.Findings)
	if err != nil {
		return report, nil, nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build maturity assessment: %w", err))
	}
	return report, assessment, collector.Stop(), nil
}

// ValidateScenario implements the shared ScenarioValidationService mount.
func (h *connectHandler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	report, maturityAssessment, execMetrics, err := h.validate(ctx, req.Msg.GetScenario(), req.Msg.GetPath())
	if err != nil {
		return nil, err
	}
	native := nativeReport(report)
	resp, err := maturity.BuildValidationResponse(report.Scenario, maturityAssessment, native, execMetrics, maturity.WithValidationStatus(h.sharedStatus(report.Findings)))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build shared validation response: %w", err))
	}
	return connect.NewResponse(resp), nil
}

// PreviewFix implements the shared Fix RPC (dry-run).
func (h *connectHandler) PreviewFix(_ context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	root, scenario, err := h.resolveFixTarget(req.Msg.GetScenario(), req.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	reg := h.deps.Fixers
	if reg == nil {
		reg = localautofix.NewRegistry()
	}
	resp, err := reg.PreviewFixResponse(scenario, root, req.Msg.GetRuleIds())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("preview experience fixes: %w", err))
	}
	return connect.NewResponse(resp), nil
}

// ApplyFix implements the shared Fix RPC (writes).
func (h *connectHandler) ApplyFix(_ context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	root, scenario, err := h.resolveFixTarget(req.Msg.GetScenario(), req.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	reg := h.deps.Fixers
	if reg == nil {
		reg = localautofix.NewRegistry()
	}
	candidates, err := localautofix.ApplySequential(reg, root, req.Msg.GetRuleIds())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("apply experience fixes: %w", err))
	}
	return connect.NewResponse(autofix.BuildFixResponse(scenario, true, candidates)), nil
}

func (h *connectHandler) resolveFixTarget(scenario, path string) (string, string, error) {
	if strings.TrimSpace(path) != "" {
		clean := filepath.Clean(path)
		if _, err := os.Stat(clean); err != nil {
			return "", "", fmt.Errorf("target path %q: %w", clean, err)
		}
		if scenario == "" {
			scenario = filepath.Base(clean)
		}
		return clean, scenario, nil
	}
	if scenario == "" {
		return "", "", fmt.Errorf("scenario is required")
	}
	root := h.deps.RepoRoot
	if root == "" {
		var err error
		root, err = os.Getwd()
		if err != nil {
			return "", "", fmt.Errorf("resolve repo root: %w", err)
		}
	}
	target := filepath.Join(root, "scenarios", scenario)
	if _, err := os.Stat(target); err != nil {
		return "", "", fmt.Errorf("scenario %q not found under %s: %w", scenario, filepath.Join(root, "scenarios"), err)
	}
	return target, scenario, nil
}

func (h *connectHandler) validateNative(ctx context.Context, scenario, path string) (*contractv1.ValidateScenarioResponse, error) {
	report, maturityAssessment, _, err := h.validate(ctx, scenario, path)
	if err != nil {
		return nil, err
	}
	return &contractv1.ValidateScenarioResponse{
		Scenario:       report.Scenario,
		Status:         nativeStatus(report.Findings),
		TargetPath:     report.TargetPath,
		DegradedReason: report.DegradedReason,
		Report:         nativeReport(report),
		Assessment:     maturityAssessment,
	}, nil
}

func nativeStatus(findings []spec.Finding) string {
	for _, finding := range findings {
		if finding.Severity == spec.SeverityError {
			return "FAILED"
		}
	}
	return "PASSED"
}

func nativeReport(report spec.Report) *contractv1.ExperienceContractReport {
	out := &contractv1.ExperienceContractReport{}
	for _, f := range checks.CapSeverity(report.Findings) {
		out.Findings = append(out.Findings, &contractv1.ExperienceFinding{
			Code:        f.Code,
			Severity:    f.Severity,
			Message:     f.Message,
			Location:    firstLocation(f.Locations),
			Remediation: f.Suggestion,
		})
	}
	return out
}

func firstLocation(locations []string) string {
	if len(locations) == 0 {
		return ""
	}
	return locations[0]
}

func protoAttestation(a attestation.Attestation) *contractv1.ManualAttestation {
	return &contractv1.ManualAttestation{
		Id:        a.ID,
		Scenario:  a.Scenario,
		Page:      a.PageID,
		Claim:     a.ClaimID,
		Author:    a.Author,
		Rationale: a.Rationale,
		ExpiresAt: a.ExpiresAt,
		CreatedAt: a.CreatedAt,
	}
}

func (h *connectHandler) sharedStatus(findings []spec.Finding) scenariovalidationv1.ValidationStatus {
	// Validation truth is independent from Test Genie's gate policy. Returning
	// PASSED for a required error made the provider success-shaped and prevented
	// downstream operators from seeing a failed experience contract. Consumers
	// may classify this failure as advisory/non-gating, but they must not rewrite
	// the provider's underlying truth.
	for _, finding := range findings {
		if finding.Severity == spec.SeverityError {
			return scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
		}
	}
	return scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_UNSPECIFIED
}

// Package validation hosts workflow-health's shared ScenarioValidationService
// provider surface for Test Genie and provider-contract checks.
package validation

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"connectrpc.com/connect"

	"workflow-health/internal/artifacts"
	"workflow-health/internal/execution"
	internalvalidation "workflow-health/internal/validation"
	workflowrun "workflow-health/internal/validationrun"
	"workflow-health/internal/workflows"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/maturity-go/assessment"
	"github.com/vrooli/maturity-go/autofix"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

const basScenario = "browser-automation-studio"

type Deps struct {
	Logger       *log.Logger
	Engine       *internalvalidation.Engine
	MaturitySpec *assessment.Spec
	RepoRoot     string
	Environment  *commonv1.CaptureEnvironment
	BASClient    execution.BASClient
	Ledger       workflowrun.Repository
}

type connectHandler struct {
	deps    Deps
	mu      sync.Mutex
	cancels map[string]context.CancelFunc
	notices map[string]chan struct{}
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	if d.Engine == nil {
		d.Engine = internalvalidation.NewEngine()
	}
	return &connectHandler{deps: d, cancels: map[string]context.CancelFunc{}, notices: map[string]chan struct{}{}}
}

func (h *connectHandler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	if req.Msg.GetIncludeExecution() {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("include_execution is retired; start a durable validation run and wait for its terminal result"))
	}
	return h.staticResponse(ctx, req.Msg.GetScenario(), req.Msg.GetPath())
}

func (h *connectHandler) staticResponse(ctx context.Context, scenario, path string) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	if strings.TrimSpace(scenario) == "" && strings.TrimSpace(path) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario or path is required"))
	}
	collector := metrics.Start(metrics.WithEnvironment(h.deps.Environment))
	report, err := h.run(ctx, scenario, path, execution.Options{})
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if h.deps.MaturitySpec == nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("maturity spec is required"))
	}
	maturity, err := internalvalidation.BuildMaturityAssessment(report.Scenario, report.Findings, *h.deps.MaturitySpec)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build maturity assessment: %w", err))
	}
	native, err := nativeDetail(report)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build native detail: %w", err))
	}
	execMetrics := collector.Stop()
	options := validationStatusOptions(report)
	resp, err := assessment.BuildValidationResponse(report.Scenario, maturity, native, execMetrics, options...)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build shared validation response: %w", err))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) PreviewFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return h.fix(ctx, req.Msg, false)
}

func (h *connectHandler) ApplyFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return h.fix(ctx, req.Msg, true)
}

func (h *connectHandler) fix(ctx context.Context, req *scenariovalidationv1.FixRequest, apply bool) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	report, err := h.deps.Engine.ValidateScenario(ctx, req.GetScenario(), req.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	fixers := h.deps.Engine.Fixers
	if fixers == nil {
		fixers = internalvalidation.NewFixRegistry()
	}
	var candidates []autofixCandidate
	if apply {
		applied, err := fixers.Apply(report.TargetPath, req.GetRuleIds())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		candidates = wrapCandidates(applied)
	} else {
		preview, err := fixers.Preview(report.TargetPath, req.GetRuleIds())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		candidates = wrapCandidates(preview)
	}
	resp := &scenariovalidationv1.FixResponse{
		Scenario: report.Scenario,
		Applied:  apply,
	}
	for _, c := range candidates {
		resp.Candidates = append(resp.Candidates, &scenariovalidationv1.FixCandidate{
			RuleId:      c.RuleID,
			FilePath:    c.FilePath,
			Description: c.Description,
			Before:      c.Before,
			After:       c.After,
			Applied:     apply,
		})
	}
	if len(resp.Candidates) == 0 {
		resp.Messages = append(resp.Messages, "no deterministic workflow fixes available")
	}
	return connect.NewResponse(resp), nil
}

type providerReport struct {
	Scenario  string
	Findings  []internalvalidation.Finding
	Catalog   *workflows.ScenarioWorkflowCatalog
	Runs      []execution.WorkflowRun
	Summary   execution.Summary
	Isolation execution.IsolationEvidence
}

func (h *connectHandler) run(ctx context.Context, scenario, path string, opts execution.Options) (providerReport, error) {
	if !opts.IncludeExecution {
		report, err := h.deps.Engine.ValidateScenario(ctx, scenario, path)
		if err != nil {
			return providerReport{}, err
		}
		return providerReport{Scenario: report.Scenario, Findings: report.Findings, Catalog: report.Catalog}, nil
	}
	client, err := h.basClient(ctx)
	if err != nil {
		return providerReport{}, err
	}
	svc := execution.NewService(client)
	svc.Validator = h.deps.Engine
	report, err := svc.RunScenario(ctx, scenario, path, opts)
	if err != nil {
		return providerReport{}, err
	}
	return providerReport{
		Scenario:  report.Scenario,
		Findings:  report.Findings,
		Catalog:   report.Catalog,
		Runs:      report.Runs,
		Summary:   report.Summary,
		Isolation: report.Isolation,
	}, nil
}

func (h *connectHandler) basClient(ctx context.Context) (execution.BASClient, error) {
	if h.deps.BASClient != nil {
		return h.deps.BASClient, nil
	}
	if base := strings.TrimSpace(os.Getenv("BROWSER_AUTOMATION_STUDIO_API_BASE")); base != "" {
		return execution.NewConnectClient(base, nil), nil
	}
	base, err := discovery.ResolveScenarioURLDefault(ctx, basScenario)
	if err != nil {
		return nil, fmt.Errorf("resolve %s URL for workflow execution: %w", basScenario, err)
	}
	return execution.NewConnectClient(base, nil), nil
}

func validationStatusOptions(report providerReport) []assessment.ValidationResponseOption {
	if report.Summary.Failed > 0 || report.Summary.Refused > 0 {
		return []assessment.ValidationResponseOption{
			assessment.WithValidationStatus(scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED),
		}
	}
	return nil
}

func nativeDetail(report providerReport) (*structpb.Struct, error) {
	payload := map[string]any{
		"catalog": catalogSummary(report.Catalog),
		"execution": map[string]any{
			"selected": report.Summary.Selected,
			"executed": report.Summary.Executed,
			"refused":  report.Summary.Refused,
			"skipped":  report.Summary.Skipped,
			"failed":   report.Summary.Failed,
			"passed":   report.Summary.Passed,
			"runs":     runSummaries(report.Runs),
			"isolation": map[string]any{
				"installed":                            report.Isolation.Installed,
				"lease_id":                             report.Isolation.LeaseID,
				"install_error":                        report.Isolation.InstallError,
				"heartbeat_error":                      report.Isolation.HeartbeatError,
				"clear_error":                          report.Isolation.ClearError,
				"test_pool_requests":                   report.Isolation.TestPoolRequests,
				"primary_during_test_mode_requests":    report.Isolation.PrimaryDuringTestModeRequests,
				"test_root_writes":                     report.Isolation.TestRootWrites,
				"primary_root_writes_during_test_mode": report.Isolation.PrimaryRootWritesDuringTestMode,
			},
		},
	}
	return structpb.NewStruct(payload)
}

func catalogSummary(catalog *workflows.ScenarioWorkflowCatalog) map[string]any {
	if catalog == nil {
		return map[string]any{}
	}
	return map[string]any{
		"scenario":             catalog.Scenario,
		"assets":               len(catalog.Assets),
		"cases":                len(catalog.Cases),
		"flows":                len(catalog.Flows),
		"actions":              len(catalog.Actions),
		"seeds":                len(catalog.Seeds),
		"registry_exists":      catalog.Registry.Exists,
		"registry_stale_paths": stringValues(catalog.RegistryOnlyPaths),
	}
}

func stringValues(values []string) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

func runSummaries(runs []execution.WorkflowRun) []any {
	out := make([]any, 0, len(runs))
	for _, run := range runs {
		out = append(out, map[string]any{
			"asset_id":     run.Asset.ID,
			"asset_path":   run.Asset.Path,
			"execution_id": run.ExecutionID,
			"status":       run.Status,
			"success":      run.Success,
			"refused":      run.Refused,
			"skipped":      run.Skipped,
			"dry_run":      run.DryRun,
			"error":        run.Error,
			"artifact": map[string]any{
				"dir":        run.Artifact.Dir,
				"workflow":   run.Artifact.Workflow,
				"latest":     run.Artifact.Latest,
				"timeline":   run.Artifact.Timeline,
				"references": artifactReferenceMaps(run.Artifact.References),
			},
		})
	}
	return out
}

func artifactReferenceMaps(references []artifacts.Reference) []any {
	out := make([]any, 0, len(references))
	for _, reference := range references {
		out = append(out, map[string]any{
			"id":         reference.ID,
			"kind":       reference.Kind,
			"uri":        reference.URI,
			"media_type": reference.MediaType,
			"checksum":   reference.Checksum,
			"redacted":   reference.Redacted,
		})
	}
	return out
}

type autofixCandidate struct {
	RuleID      string
	FilePath    string
	Description string
	Before      string
	After       string
}

func wrapCandidates(input []autofix.Candidate) []autofixCandidate {
	out := make([]autofixCandidate, 0, len(input))
	for _, c := range input {
		out = append(out, autofixCandidate{
			RuleID:      c.RuleID,
			FilePath:    c.FilePath,
			Description: c.Description,
			Before:      c.Before,
			After:       c.After,
		})
	}
	return out
}

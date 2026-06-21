// Package validation mounts performance-health's native ReadinessService and the
// shared ScenarioValidationService (dual-mount) over one readiness engine. The
// native service serves the richer tier+infra shape the performance-health
// UI/CLI consume; the shared service is the contract test-genie delegates axes
// ①/③ to in P9.
package validation

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"connectrpc.com/connect"

	phassessment "performance-health/internal/assessment"
	"performance-health/internal/autofix"
	"performance-health/internal/budgets"
	"performance-health/internal/readiness"

	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/maturity-go/assessment"
	autofixcore "github.com/vrooli/maturity-go/autofix"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/readiness"
	readinessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/readiness/readiness_v1connect"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// BudgetChecker evaluates a scenario's latest measurements against its declared
// performance budget. The budgets service satisfies it; it is optional (nil =>
// no budget gating folded into validation). This is the seam that turns a perf
// budget breach into a validation failure (and therefore a baseline-diff exit-1)
// exactly like any other health regression.
type BudgetChecker interface {
	Check(ctx context.Context, scenario string) (passed bool, violations []budgets.Violation, err error)
}

// Deps are the handler's collaborators.
type Deps struct {
	Readiness    *readiness.Service
	Autofix      *autofix.Service
	Budgets      BudgetChecker
	Logger       *log.Logger
	MaturitySpec *assessment.Spec
	// RepoRoot resolves a scenario's filesystem path for autofix.
	RepoRoot string
	// Environment is the host CaptureEnvironment captured once at module init.
	Environment *commonv1.CaptureEnvironment
	// Execution runs the deterministic perf producers (benchmark + bundle,
	// Lighthouse-if-UI) and folds breaches into findings when a caller requests
	// execution-mode validation (include_execution=true). Optional (nil =>
	// execution mode degrades to readiness + budget-on-existing-trend).
	Execution ExecutionRunner
}

// Handler implements the generated native ReadinessServiceHandler.
type Handler struct {
	readinessconnect.UnimplementedReadinessServiceHandler
	readiness *readiness.Service
	autofix   *autofix.Service
	budgets   BudgetChecker
	logger    *log.Logger
	spec      *assessment.Spec
	repoRoot  string
	env       *commonv1.CaptureEnvironment
	execution ExecutionRunner
}

// NewHandlerWithDeps builds a Handler, defaulting nil collaborators.
func NewHandlerWithDeps(deps Deps) *Handler {
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	return &Handler{
		readiness: deps.Readiness,
		autofix:   deps.Autofix,
		budgets:   deps.Budgets,
		logger:    deps.Logger,
		spec:      deps.MaturitySpec,
		repoRoot:  deps.RepoRoot,
		env:       deps.Environment,
		execution: deps.Execution,
	}
}

var _ readinessconnect.ReadinessServiceHandler = (*Handler)(nil)

// ValidateReadiness reports the reachable capture tier and the perf-build infra
// findings for a scenario without writing. It is readiness-only (no process is
// spawned): execution-mode measurement is reached through the shared
// ScenarioValidationService with include_execution=true.
func (h *Handler) ValidateReadiness(ctx context.Context, req *connect.Request[readinessv1.ValidateReadinessRequest]) (*connect.Response[readinessv1.ValidateReadinessResponse], error) {
	return h.validate(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), false)
}

// validate is the shared core for both the readiness-only RPC and the shared
// ScenarioValidationService. When includeExecution is true and an execution
// runner is wired, it runs the deterministic producers (which persist a fresh
// perf_samples row) BEFORE evaluating budgets, then folds the execution
// threshold-breach findings into the assessment alongside readiness + budget
// findings. When false it is exactly the prior readiness + budget-on-existing-
// trend behavior (no process spawned).
func (h *Handler) validate(ctx context.Context, scenario, path string, includeExecution bool) (*connect.Response[readinessv1.ValidateReadinessResponse], error) {
	if scenario == "" && path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario or path is required"))
	}
	if h.readiness == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("readiness engine not wired"))
	}
	// Execution mode runs the producers first so the budget gate below reads the
	// freshly persisted sample. Its findings (build-over-threshold, broken build,
	// Lighthouse error-threshold) are folded into the assessment.
	var executionFindings []phassessment.Finding
	if includeExecution && h.execution != nil {
		executionFindings = h.execution.Run(ctx, scenario, path)
	}
	res, err := h.readiness.Validate(ctx, scenario, path)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	budgetFindings := h.budgetFindings(ctx, res.Scenario)
	extraFindings := append(executionFindings, budgetFindings...)
	maturity, err := buildAssessment(res, extraFindings, h.spec)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build maturity assessment: %w", err))
	}
	out := &readinessv1.ValidateReadinessResponse{
		Scenario:         res.Scenario,
		Status:           assessment.DeriveValidationStatus(maturity),
		Tier:             readiness.TierToProto(res.Tier),
		UiFramework:      res.UIFramework,
		Surfaces:         res.Surfaces,
		Assessment:       maturity,
		AutofixableCount: int32(res.AutofixableCount()),
		DegradedReason:   res.DegradedReason,
	}
	if out.Status == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_UNSPECIFIED {
		out.Status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED
	}
	return connect.NewResponse(out), nil
}

// PreviewReadinessFix returns the format-preserving edits readiness could apply,
// without writing.
func (h *Handler) PreviewReadinessFix(ctx context.Context, req *connect.Request[readinessv1.ReadinessFixRequest]) (*connect.Response[readinessv1.ReadinessFixResponse], error) {
	return h.fix(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetRuleIds(), false)
}

// ApplyReadinessFix applies the format-preserving edits and reports what changed.
func (h *Handler) ApplyReadinessFix(ctx context.Context, req *connect.Request[readinessv1.ReadinessFixRequest]) (*connect.Response[readinessv1.ReadinessFixResponse], error) {
	return h.fix(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetRuleIds(), true)
}

func (h *Handler) fix(_ context.Context, scenario, path string, ruleIDs []string, apply bool) (*connect.Response[readinessv1.ReadinessFixResponse], error) {
	if scenario == "" && path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario or path is required"))
	}
	if h.autofix == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("autofix engine not wired"))
	}
	root := h.resolveRoot(scenario, path)
	var (
		candidates []autofix.Candidate
		err        error
	)
	if apply {
		candidates, err = h.autofix.Apply(root, ruleIDs)
	} else {
		candidates, err = h.autofix.Preview(root, ruleIDs)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out := &readinessv1.ReadinessFixResponse{Scenario: firstNonEmpty(scenario, path), Applied: apply}
	for _, c := range candidates {
		out.Candidates = append(out.Candidates, &scenariovalidationv1.FixCandidate{
			RuleId:      c.RuleID,
			FilePath:    c.FilePath,
			Description: c.Description,
			Before:      c.Before,
			After:       c.After,
			Applied:     c.Applied,
		})
	}
	if len(candidates) == 0 {
		out.Messages = append(out.Messages, "No auto-fixable readiness findings are available.")
	}
	return connect.NewResponse(out), nil
}

func (h *Handler) resolveRoot(scenario, path string) string {
	if strings.TrimSpace(path) != "" {
		return path
	}
	if h.repoRoot != "" && scenario != "" {
		return strings.TrimRight(h.repoRoot, "/") + "/scenarios/" + scenario
	}
	return scenario
}

// budgetFindings runs the budget gate for a scenario and projects any breach
// into performance-health findings (ERROR severity). A nil checker, or any
// transient error, yields no findings — the budget gate never breaks readiness;
// it only ADDS a failure when a budget is breached. The budget breach finding is
// what drives the shared validation status to FAILED (baseline-diff exit 1).
func (h *Handler) budgetFindings(ctx context.Context, scenario string) []phassessment.Finding {
	if h.budgets == nil || scenario == "" {
		return nil
	}
	passed, violations, err := h.budgets.Check(ctx, scenario)
	if err != nil {
		h.logger.Printf("validation: budget check for %s degraded: %v", scenario, err)
		return nil
	}
	if passed || len(violations) == 0 {
		return nil
	}
	out := make([]phassessment.Finding, 0, len(violations))
	for _, f := range budgets.Findings(scenario, violations) {
		out = append(out, phassessment.Finding{
			Code:             f.Code,
			Severity:         f.Severity,
			Title:            f.Title,
			Message:          f.Message,
			Location:         f.Location,
			AutofixAvailable: f.AutofixAvailable,
		})
	}
	return out
}

func buildAssessment(res readiness.Result, extraFindings []phassessment.Finding, spec *assessment.Spec) (*commonv1.MaturityAssessment, error) {
	findings := make([]phassessment.Finding, 0, len(res.Findings)+len(extraFindings))
	for _, f := range res.Findings {
		findings = append(findings, phassessment.Finding{
			Code:             f.Code,
			Severity:         f.Severity,
			Title:            f.Code,
			Message:          f.Message,
			AutofixAvailable: f.Autofixable,
		})
	}
	// Dedupe extra findings by code: the native build-time floor and a declared
	// budget can both emit e.g. PERF_BUDGET_BREACH_GO_BUILD; the gate only needs
	// one. First occurrence (execution floor) wins.
	seen := make(map[string]struct{}, len(findings))
	for _, f := range findings {
		seen[f.Code] = struct{}{}
	}
	for _, f := range extraFindings {
		if _, dup := seen[f.Code]; dup {
			continue
		}
		seen[f.Code] = struct{}{}
		findings = append(findings, f)
	}
	return phassessment.Build(res.Scenario, spec, findings)
}

// SharedHandler adapts the native readiness RPC to the shared
// ScenarioValidationService contract consumed by test-genie.
type SharedHandler struct {
	handler *Handler
}

// NewSharedHandler wraps a native Handler in the shared adapter.
func NewSharedHandler(handler *Handler) *SharedHandler { return &SharedHandler{handler: handler} }

// ValidateScenario runs the native engine and returns the shared response with
// the native readiness detail packed into native_detail and metrics attached.
func (h *SharedHandler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	if h == nil || h.handler == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("readiness handler not wired"))
	}
	collector := metrics.Start(metrics.WithEnvironment(h.handler.env))
	// include_execution honored end-to-end: true => run the deterministic
	// producers (benchmark + bundle, Lighthouse-if-UI), persist a fresh sample,
	// then gate on budgets + native thresholds; false => readiness + budget-on-
	// existing-trend only (no process spawned). The metrics collector wraps the
	// whole path so the resource envelope covers the builds.
	native, err := h.handler.validate(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetIncludeExecution())
	if err != nil {
		collector.Stop()
		return nil, err
	}
	execMetrics := collector.Stop()
	resp, err := assessment.BuildValidationResponse(
		native.Msg.GetScenario(),
		native.Msg.GetAssessment(),
		native.Msg,
		execMetrics,
		assessment.WithValidationStatus(native.Msg.GetStatus()),
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build shared validation response: %w", err))
	}
	return connect.NewResponse(resp), nil
}

// PreviewFix exposes readiness's deterministic fixes through the shared Fix RPC.
func (h *SharedHandler) PreviewFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return h.sharedFix(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetRuleIds(), false)
}

// ApplyFix applies readiness's deterministic fixes through the shared Fix RPC.
func (h *SharedHandler) ApplyFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return h.sharedFix(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetRuleIds(), true)
}

func (h *SharedHandler) sharedFix(_ context.Context, scenario, path string, ruleIDs []string, apply bool) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	if h == nil || h.handler == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("readiness handler not wired"))
	}
	if scenario == "" && path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario or path is required"))
	}
	if h.handler.autofix == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("autofix engine not wired"))
	}
	root := h.handler.resolveRoot(scenario, path)
	var (
		candidates []autofixcore.Candidate
		err        error
	)
	if apply {
		candidates, err = h.handler.autofix.Apply(root, ruleIDs)
	} else {
		candidates, err = h.handler.autofix.Preview(root, ruleIDs)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(autofixcore.BuildFixResponse(firstNonEmpty(scenario, path), apply, candidates)), nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

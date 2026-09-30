// Package validation exposes quality-health through the shared
// ScenarioValidationService contract consumed by Test Genie.
package validation

import (
	"context"
	"errors"
	"fmt"
	"log"

	"connectrpc.com/connect"

	auditH "quality-health/handlers/audit"
	internalaudit "quality-health/internal/audit"
	"quality-health/internal/surfaces"

	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/maturity-go/assessment"
	autofixcore "github.com/vrooli/maturity-go/autofix"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/quality-health/v1/audit"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// Auditor is the subset of quality-health's audit service needed by the shared
// validation adapter, including the deterministic config fixers exposed through
// the shared scenario-validation Fix RPC.
type Auditor interface {
	Audit(context.Context, internalaudit.Request) (internalaudit.Response, error)
	PreviewFix(ctx context.Context, scenario, path string, ruleIDs []string) (surfaces.Inventory, []autofixcore.Candidate, error)
	ApplyFix(ctx context.Context, scenario, path string, ruleIDs []string) (surfaces.Inventory, []autofixcore.Candidate, error)
}

// Deps wires the shared validation handler.
type Deps struct {
	Auditor      Auditor
	Logger       *log.Logger
	MaturitySpec *assessment.Spec
	// Environment is the host CaptureEnvironment captured once at module init
	// (os/arch/cpu/mem/present-GPUs). nil is safe — the metrics collector
	// backfills os/arch/num_cpu from the stdlib.
	Environment *commonv1.CaptureEnvironment
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// ValidateScenario is the scenario-shaped alias. It delegates to validate with
// the scenario kind, which is what the request could only ever have meant.
func (h *connectHandler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	return h.validate(ctx, req.Msg, "scenario", nil)
}

// validate runs one audit against a target of the given kind. targetKind is the
// repo-contract governance kind; target, when set, additionally scopes the
// maturity assessment so capabilities that do not apply to this kind are not
// scored.
func (h *connectHandler) validate(
	ctx context.Context,
	msg *scenariovalidationv1.ValidateScenarioRequest,
	targetKind string,
	target *commonv1.ValidationTarget,
) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	if h.deps.Auditor == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("quality validation auditor not wired"))
	}
	if msg.GetScenario() == "" && msg.GetPath() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario or path is required"))
	}
	collector := metrics.Start(metrics.WithEnvironment(h.deps.Environment))
	report, err := h.deps.Auditor.Audit(internalaudit.WithMetrics(ctx, collector), internalaudit.Request{
		Scenario:                msg.GetScenario(),
		Path:                    msg.GetPath(),
		ValidationTargetKind:    targetKind,
		IncludeCommandExecution: msg.GetIncludeExecution(),
		UseCache:                true,
	})
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	native, err := auditResponseToProto(report, h.deps.MaturitySpec, target)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build quality native detail: %w", err))
	}
	execMetrics := collector.Stop()
	resp, err := assessment.BuildValidationResponse(native.GetScenario(), native.GetAssessment(), native, execMetrics, statusOverride(native)...)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build shared validation response: %w", err))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ValidateTarget(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateTargetRequest]) (*connect.Response[scenariovalidationv1.ValidateTargetResponse], error) {
	target := req.Msg.GetTarget()
	if target == nil || target.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("target is required"))
	}
	path := req.Msg.GetPath()
	if path == "" {
		path = target.GetRoot()
	}
	// Carry the kind through. Downgrading to ValidateScenario without it made
	// every target look scenario-shaped, so scenario-contract rules — Makefile
	// gates, coverage/testing.json policy — fired against packages that cannot
	// have either, and one of them is an ERROR that failed the run.
	legacy, err := h.validate(ctx, &scenariovalidationv1.ValidateScenarioRequest{
		Scenario:         target.GetId(),
		Path:             path,
		IncludeExecution: req.Msg.GetIncludeExecution(),
	}, assessment.TargetKindName(target.GetKind()), target)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&scenariovalidationv1.ValidateTargetResponse{Target: target, Status: legacy.Msg.GetStatus(), Assessment: legacy.Msg.GetAssessment(), NativeDetail: legacy.Msg.GetNativeDetail(), Metrics: legacy.Msg.GetMetrics()}), nil
}

// PreviewFix reports the deterministic config edits quality-health could apply
// for the requested scenario without writing anything (shared Fix RPC).
func (h *connectHandler) PreviewFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	if h.deps.Auditor == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("quality validation auditor not wired"))
	}
	if req.Msg.GetScenario() == "" && req.Msg.GetPath() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario or path is required"))
	}
	inv, candidates, err := h.deps.Auditor.PreviewFix(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetRuleIds())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(autofixcore.BuildFixResponse(fixScenario(req.Msg, inv), false, candidates)), nil
}

// ApplyFix applies quality-health's deterministic config edits for the requested
// scenario and reports what changed (shared Fix RPC).
func (h *connectHandler) ApplyFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	if h.deps.Auditor == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("quality validation auditor not wired"))
	}
	if req.Msg.GetScenario() == "" && req.Msg.GetPath() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario or path is required"))
	}
	inv, candidates, err := h.deps.Auditor.ApplyFix(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetRuleIds())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(autofixcore.BuildFixResponse(fixScenario(req.Msg, inv), true, candidates)), nil
}

func fixScenario(req *scenariovalidationv1.FixRequest, inv surfaces.Inventory) string {
	if s := inv.Scenario; s != "" {
		return s
	}
	return req.GetScenario()
}

func auditResponseToProto(in internalaudit.Response, spec *assessment.Spec, target *commonv1.ValidationTarget) (*auditv1.AuditQualityResponse, error) {
	return auditH.ResponseToProto(in, spec, target)
}

func statusOverride(resp *auditv1.AuditQualityResponse) []assessment.ValidationResponseOption {
	switch resp.GetStatus() {
	case "degraded":
		return []assessment.ValidationResponseOption{assessment.WithValidationStatus(scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED)}
	case "error":
		return []assessment.ValidationResponseOption{assessment.WithValidationStatus(scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR)}
	default:
		return nil
	}
}

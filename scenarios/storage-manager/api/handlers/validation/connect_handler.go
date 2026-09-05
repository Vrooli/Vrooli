// Package validation hosts the shared ScenarioValidationService handler for
// storage-manager. The handler delegates real work to the internal validation
// Service and returns the common scenario-validation response shape that
// test-genie's delegated `storage` phase consumes.
package validation

import (
	"context"
	"fmt"
	"log"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"

	"storage-manager/internal/autofix"
	"storage-manager/internal/validation"

	"github.com/vrooli/api-core/metrics"
	corestorage "github.com/vrooli/api-core/storage"
	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// Validator is the slice of internal/validation.Service the handler exercises.
// An interface so handler tests can stub it without running real analyzers.
type Validator interface {
	ValidateScenario(ctx context.Context, scenario string) (validation.Report, error)
}

type ownerValidator interface {
	ValidateOwner(ctx context.Context, kind corestorage.OwnerKind, id string, requested corestorage.Platform) (validation.Report, error)
}

// Deps wires the seams the Connect validation handler needs.
type Deps struct {
	Logger       *log.Logger
	Validator    Validator
	MaturitySpec *assessment.Spec
	// RepoRoot is the resolved repository root. PreviewFix/ApplyFix resolve a
	// request's scenario to scenarios/<scenario> beneath it so the deterministic
	// autofix registry operates on the right tree. Empty disables the Fix RPCs.
	RepoRoot string
	// Environment is the host CaptureEnvironment captured once at module init.
	// nil is safe — the metrics collector backfills os/arch/num_cpu.
	Environment *commonv1.CaptureEnvironment
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the shared validation handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	if h.deps.Validator == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("validation.ValidateScenario: validator not wired"))
	}
	collector := metrics.Start(metrics.WithEnvironment(h.deps.Environment))
	report, err := h.deps.Validator.ValidateScenario(validation.WithMetrics(ctx, collector), req.Msg.GetScenario())
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	maturityAssessment, err := buildMaturityAssessment(report, h.deps.MaturitySpec)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build maturity assessment: %w", err))
	}
	execMetrics := collector.Stop()
	nativeDetail, err := structpb.NewStruct(map[string]any{
		"file_persisting": report.HasEngine(validation.EngineFile),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build storage native detail: %w", err))
	}
	resp, err := assessment.BuildValidationResponse(report.Scenario, maturityAssessment, nativeDetail, execMetrics)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build shared validation response: %w", err))
	}
	return connect.NewResponse(resp), nil
}

// ValidateTarget is the generalized contract's compatibility path. Storage
// adoption for resource/tool/safeguard targets is wired by the fleet phase;
// until then this provider truthfully accepts only scenario targets.
func (h *connectHandler) ValidateTarget(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateTargetRequest]) (*connect.Response[scenariovalidationv1.ValidateTargetResponse], error) {
	if req == nil || req.Msg == nil || req.Msg.GetTarget() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("target is required"))
	}
	target := req.Msg.GetTarget()
	if target.GetKind() != commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO {
		owner, ok := ownerKind(target.GetKind())
		if !ok {
			return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("storage-manager does not support target kind %s", target.GetKind()))
		}
		validator, ok := h.deps.Validator.(ownerValidator)
		if !ok {
			return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("storage-manager owner validation is not wired"))
		}
		collector := metrics.Start(metrics.WithEnvironment(h.deps.Environment))
		report, err := validator.ValidateOwner(validation.WithMetrics(ctx, collector), owner, target.GetId(), corestorage.HostPlatform())
		if err != nil {
			collector.Stop()
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		for i := range report.Findings {
			if report.Findings[i].Subject == nil {
				report.Findings[i].Subject = target
			}
		}
		maturityAssessment, err := buildMaturityAssessment(report, h.deps.MaturitySpec)
		if err != nil {
			collector.Stop()
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build maturity assessment: %w", err))
		}
		execMetrics := collector.Stop()
		nativeDetail, err := structpb.NewStruct(map[string]any{"owner_kind": string(owner), "owner_id": target.GetId()})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		shared, err := assessment.BuildValidationResponse(report.Scenario, maturityAssessment, nativeDetail, execMetrics)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&scenariovalidationv1.ValidateTargetResponse{Target: target, Status: shared.GetStatus(), Assessment: shared.GetAssessment(), NativeDetail: shared.GetNativeDetail(), Metrics: shared.GetMetrics()}), nil
	}
	legacy, err := h.ValidateScenario(ctx, connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario:         target.GetId(),
		Path:             req.Msg.GetPath(),
		IncludeExecution: req.Msg.GetIncludeExecution(),
	}))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&scenariovalidationv1.ValidateTargetResponse{
		Target:       req.Msg.GetTarget(),
		Status:       legacy.Msg.GetStatus(),
		Assessment:   legacy.Msg.GetAssessment(),
		NativeDetail: legacy.Msg.GetNativeDetail(),
		Metrics:      legacy.Msg.GetMetrics(),
	}), nil
}

func buildMaturityAssessment(rep validation.Report, spec *assessment.Spec) (*commonv1.MaturityAssessment, error) {
	if spec == nil {
		return nil, fmt.Errorf("maturity spec is required")
	}
	findings := make([]assessment.Finding, 0, len(rep.Findings))
	for _, f := range rep.Findings {
		// A finding is auto-fixable when the autofix registry has a fixer for its
		// Code. The registry is the single source of truth for what storage-manager
		// can deterministically remediate, so the assessment's AutofixableCount
		// reflects exactly the registry's coverage.
		findings = append(findings, assessment.Finding{
			Code:             f.Code,
			Severity:         f.Severity.Token(),
			Title:            f.Title,
			Message:          f.Message,
			Location:         f.Location,
			Remediation:      f.Remediation,
			Phase:            spec.Phase,
			AutofixAvailable: f.AutofixAvailable || autofix.CoveredCodes[f.Code],
			Subject:          f.Subject,
		})
	}
	return assessment.BuildProtoAssessment(assessment.BuildInput{
		Scenario: rep.Scenario,
		Spec:     *spec,
		Findings: findings,
	})
}

func ownerKind(kind commonv1.ValidationTargetKind) (corestorage.OwnerKind, bool) {
	switch kind {
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_RESOURCE:
		return corestorage.OwnerResource, true
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_TOOL:
		return corestorage.OwnerTool, true
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SAFEGUARD:
		return corestorage.OwnerSafeguard, true
	default:
		return "", false
	}
}

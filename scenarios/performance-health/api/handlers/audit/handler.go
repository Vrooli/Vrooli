// Package audit mounts performance-health's AuditService — the capture
// orchestration conductor (profile-mode restart → BAS perf capture → restore),
// with a clean skip when capture is impossible.
package audit

import (
	"context"
	"log"

	"connectrpc.com/connect"

	"performance-health/internal/capture"
	"performance-health/internal/readiness"

	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/audit"
	auditconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/audit/audit_v1connect"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/readiness"
)

// Tierer decides the reachable capture tier for a scenario. The readiness
// engine satisfies this; tests drive a fake.
type Tierer interface {
	Validate(ctx context.Context, scenario, path string) (readiness.Result, error)
}

// Handler implements the generated AuditServiceHandler.
type Handler struct {
	auditconnect.UnimplementedAuditServiceHandler
	svc    *capture.Service
	tierer Tierer
	logger *log.Logger
}

// NewHandler builds an audit Handler.
func NewHandler(svc *capture.Service, tierer Tierer, logger *log.Logger) *Handler {
	if logger == nil {
		logger = log.Default()
	}
	return &Handler{svc: svc, tierer: tierer, logger: logger}
}

var _ auditconnect.AuditServiceHandler = (*Handler)(nil)

// RunAudit decides the scenario's reachable tier via readiness, then orchestrates
// the capture (profile-mode for Tier 1, direct for Tier 0), cleanly skipping when
// capture is impossible.
func (h *Handler) RunAudit(ctx context.Context, req *connect.Request[auditv1.RunAuditRequest]) (*connect.Response[auditv1.RunAuditResponse], error) {
	scenario := req.Msg.GetScenario()
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalid("scenario is required"))
	}
	arm, err := layoutCacheArmFromProto(req.Msg.GetLayoutCacheArm())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalid(err.Error()))
	}
	workflow := req.Msg.GetWorkflow()
	if err := capture.ValidateLayoutCacheArm(scenario, workflow, arm); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalid(err.Error()))
	}

	tier := readiness.Tier0
	if h.tierer != nil {
		readyRes, rerr := h.tierer.Validate(ctx, scenario, req.Msg.GetPath())
		if rerr != nil {
			// Readiness failure should not hard-fail the audit; fall back to the
			// safe Tier-0 path so a capture is still attempted (or cleanly skipped).
			h.logger.Printf("audit.RunAudit(%s): tier detection degraded: %v", scenario, rerr)
		} else {
			tier = readyRes.Tier
		}
	}

	res, err := h.svc.OrchestrateWithLayoutCacheArm(ctx, scenario, workflow, tier, arm)
	if err != nil {
		h.logger.Printf("audit.RunAudit(%s): %v", scenario, err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&auditv1.RunAuditResponse{
		Scenario:               res.Scenario,
		Outcome:                outcomeToProto(res.Outcome),
		Tier:                   tierToProto(res.Tier),
		TraceArtifact:          res.TraceArtifact,
		WebVitalsArtifact:      res.WebVitalsArtifact,
		ProcessTreeArtifact:    res.ProcessTreeArtifact,
		ProcessTreeReference:   res.ProcessTreeReference,
		ProcessTreeSessionId:   res.ProcessTreeSessionID,
		ProcessTreeLeaseSha256: res.ProcessTreeLeaseSHA256,
		Reason:                 res.Reason,
		LayoutCacheArm:         layoutCacheArmToProto(res.LayoutCacheArm),
	}), nil
}

func layoutCacheArmFromProto(arm auditv1.LayoutCacheArm) (capture.LayoutCacheArm, error) {
	switch arm {
	case auditv1.LayoutCacheArm_LAYOUT_CACHE_ARM_UNSPECIFIED:
		return capture.LayoutCacheArmUnspecified, nil
	case auditv1.LayoutCacheArm_LAYOUT_CACHE_ARM_ENABLED:
		return capture.LayoutCacheArmEnabled, nil
	case auditv1.LayoutCacheArm_LAYOUT_CACHE_ARM_DISABLED:
		return capture.LayoutCacheArmDisabled, nil
	default:
		return capture.LayoutCacheArmUnspecified, errInvalid("layout cache arm is unknown")
	}
}

func layoutCacheArmToProto(arm capture.LayoutCacheArm) auditv1.LayoutCacheArm {
	switch arm {
	case capture.LayoutCacheArmEnabled:
		return auditv1.LayoutCacheArm_LAYOUT_CACHE_ARM_ENABLED
	case capture.LayoutCacheArmDisabled:
		return auditv1.LayoutCacheArm_LAYOUT_CACHE_ARM_DISABLED
	default:
		return auditv1.LayoutCacheArm_LAYOUT_CACHE_ARM_UNSPECIFIED
	}
}

func outcomeToProto(o capture.Outcome) auditv1.AuditOutcome {
	switch o {
	case capture.OutcomeCaptured:
		return auditv1.AuditOutcome_AUDIT_OUTCOME_CAPTURED
	case capture.OutcomeSkipped:
		return auditv1.AuditOutcome_AUDIT_OUTCOME_SKIPPED
	case capture.OutcomeFailed:
		return auditv1.AuditOutcome_AUDIT_OUTCOME_FAILED
	case capture.OutcomeUnavailable:
		return auditv1.AuditOutcome_AUDIT_OUTCOME_UNAVAILABLE
	default:
		return auditv1.AuditOutcome_AUDIT_OUTCOME_UNSPECIFIED
	}
}

func tierToProto(t readiness.Tier) readinessv1.CaptureTier { return readiness.TierToProto(t) }

type invalidArg string

func (e invalidArg) Error() string { return string(e) }

func errInvalid(msg string) error { return invalidArg(msg) }

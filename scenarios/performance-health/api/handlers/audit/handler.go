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

	res, err := h.svc.Orchestrate(ctx, scenario, req.Msg.GetWorkflow(), tier)
	if err != nil {
		h.logger.Printf("audit.RunAudit(%s): %v", scenario, err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&auditv1.RunAuditResponse{
		Scenario:          res.Scenario,
		Outcome:           outcomeToProto(res.Outcome),
		Tier:              tierToProto(res.Tier),
		TraceArtifact:     res.TraceArtifact,
		WebVitalsArtifact: res.WebVitalsArtifact,
		Reason:            res.Reason,
	}), nil
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

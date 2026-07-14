package onboard

import (
	"context"
	"time"

	"vrooli-bridge/internal/onboard"
	"vrooli-bridge/internal/pairing"
	"vrooli-bridge/internal/presence"

	"google.golang.org/protobuf/types/known/timestamppb"

	onboardv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/onboard"
)

// This file is the single translation point between the proto-free onboard
// domain (its seams + DTOs) and proto, plus the concrete pairing / presence
// services. The onboard domain never imports proto or a sibling domain; these
// adapters do.

// ---- proto <-> domain translations ----

func domainOpToProto(op onboard.Op) *onboardv1.OnboardingOp {
	out := &onboardv1.OnboardingOp{
		Id:             op.ID,
		Host:           op.Host,
		Port:           int32(op.Port),
		User:           op.User,
		NodeName:       op.NodeName,
		TargetRevision: op.TargetRevision,
		RepoUrl:        op.RepoURL,
		State:          stateToProto(op.State),
		NodeId:         op.NodeID,
		FailureReason:  string(op.FailureReason),
		ExitCode:       op.ExitCode,
		CreatedAt:      timestamppb.New(op.CreatedAt),
	}
	if !op.StartedAt.IsZero() {
		out.StartedAt = timestamppb.New(op.StartedAt)
	}
	if !op.FinishedAt.IsZero() {
		out.FinishedAt = timestamppb.New(op.FinishedAt)
	}
	return out
}

func stateToProto(s onboard.State) onboardv1.OnboardingState {
	switch s {
	case onboard.StatePending:
		return onboardv1.OnboardingState_ONBOARDING_STATE_PENDING
	case onboard.StateSSHSetup:
		return onboardv1.OnboardingState_ONBOARDING_STATE_SSH_SETUP
	case onboard.StatePushingScript:
		return onboardv1.OnboardingState_ONBOARDING_STATE_PUSHING_SCRIPT
	case onboard.StateBootstrapping:
		return onboardv1.OnboardingState_ONBOARDING_STATE_BOOTSTRAPPING
	case onboard.StateVerifying:
		return onboardv1.OnboardingState_ONBOARDING_STATE_VERIFYING
	case onboard.StateSucceeded:
		return onboardv1.OnboardingState_ONBOARDING_STATE_SUCCEEDED
	case onboard.StateFailed:
		return onboardv1.OnboardingState_ONBOARDING_STATE_FAILED
	case onboard.StateCancelled:
		return onboardv1.OnboardingState_ONBOARDING_STATE_CANCELLED
	default:
		return onboardv1.OnboardingState_ONBOARDING_STATE_UNSPECIFIED
	}
}

func domainStepEventToProto(ev onboard.StepEvent) *onboardv1.OnboardingStepEvent {
	out := &onboardv1.OnboardingStepEvent{
		OpId:     ev.OpID,
		Sequence: ev.Sequence,
		StepId:   ev.StepID,
		Status:   stepStatusToProto(ev.Status),
		Detail:   ev.Detail,
	}
	if !ev.EmittedAt.IsZero() {
		out.EmittedAt = timestamppb.New(ev.EmittedAt)
	}
	return out
}

func stepStatusToProto(s onboard.StepStatus) onboardv1.OnboardingStepStatus {
	switch s {
	case onboard.StepStatusStarted:
		return onboardv1.OnboardingStepStatus_ONBOARDING_STEP_STATUS_STARTED
	case onboard.StepStatusOK:
		return onboardv1.OnboardingStepStatus_ONBOARDING_STEP_STATUS_OK
	case onboard.StepStatusSkipped:
		return onboardv1.OnboardingStepStatus_ONBOARDING_STEP_STATUS_SKIPPED
	case onboard.StepStatusFailed:
		return onboardv1.OnboardingStepStatus_ONBOARDING_STEP_STATUS_FAILED
	default:
		return onboardv1.OnboardingStepStatus_ONBOARDING_STEP_STATUS_UNSPECIFIED
	}
}

// ---- seam adapters (proto-free domain <-> concrete services) ----

// codeIssuerAdapter mints a single-use pairing code server-side via the pairing
// service, returning it as an owned []byte the orchestrator zeroes after use.
type codeIssuerAdapter struct {
	svc *pairing.Service
}

var _ onboard.CodeIssuer = codeIssuerAdapter{}

func (a codeIssuerAdapter) Issue(ctx context.Context, p onboard.IssueParams) ([]byte, error) {
	issued, err := a.svc.IssueCode(ctx, p.NodeName, p.Scopes, 0)
	if err != nil {
		return nil, err
	}
	return []byte(issued.Code), nil
}

// presencePoller is the narrow presence read the online confirmer needs.
type presencePoller interface {
	IsOnline(nodeID string) bool
}

var _ presencePoller = (*presence.Hub)(nil)

// onlineConfirmerAdapter confirms a freshly-onboarded node is ONLINE by polling
// the presence hub until the node holds a live dial-out channel or the budget
// elapses. A node online in the hub has completed the signed-frame handshake, so
// its control-plane key is pinned (the agent refuses to connect otherwise).
type onlineConfirmerAdapter struct {
	presence presencePoller
	interval time.Duration
}

var _ onboard.OnlineConfirmer = onlineConfirmerAdapter{}

func newOnlineConfirmer(p presencePoller) onlineConfirmerAdapter {
	return onlineConfirmerAdapter{presence: p, interval: time.Second}
}

func (a onlineConfirmerAdapter) ConfirmOnline(ctx context.Context, nodeID string, timeout time.Duration) (bool, error) {
	if a.presence.IsOnline(nodeID) {
		return true, nil
	}
	interval := a.interval
	if interval <= 0 {
		interval = time.Second
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-deadline.C:
			return a.presence.IsOnline(nodeID), nil
		case <-ticker.C:
			if a.presence.IsOnline(nodeID) {
				return true, nil
			}
		}
	}
}

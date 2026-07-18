package onboard

import (
	"context"
	"errors"
	"time"

	"vrooli-bridge/internal/machines"
	"vrooli-bridge/internal/onboard"
	"vrooli-bridge/internal/pairing"
	"vrooli-bridge/internal/presence"
	"vrooli-bridge/internal/registry"

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
		FailureDetail:  op.FailureDetail,
		ExitCode:       op.ExitCode,
		CreatedAt:      timestamppb.New(op.CreatedAt),

		SourceMode:        sourceModeToProto(op.SourceMode),
		BaseRevision:      op.BaseRevision,
		WorkingTreeDigest: op.WorkingTreeDigest,
		ControlPlaneUrl:   op.ControlPlaneURL,
		ReachabilityMode:  op.ReachabilityMode,
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

func sourceModeToProto(m onboard.SourceMode) onboardv1.SourceMode {
	switch m {
	case onboard.SourceModeWorkingTree:
		return onboardv1.SourceMode_SOURCE_MODE_WORKING_TREE
	default:
		return onboardv1.SourceMode_SOURCE_MODE_PINNED_REVISION
	}
}

// sourceModeFromProto maps the wire source mode to the domain, defaulting an
// unspecified value to pinned (the fleet-safe default).
func sourceModeFromProto(m onboardv1.SourceMode) onboard.SourceMode {
	switch m {
	case onboardv1.SourceMode_SOURCE_MODE_WORKING_TREE:
		return onboard.SourceModeWorkingTree
	default:
		return onboard.SourceModePinned
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

var _ onboard.EnrollmentResolver = codeIssuerAdapter{}

func (a codeIssuerAdapter) Issue(ctx context.Context, p onboard.IssueParams) ([]byte, error) {
	issued, err := a.svc.IssueCodeForEnrollment(ctx, p.NodeName, p.Scopes, 0, p.CorrelationID)
	if err != nil {
		return nil, err
	}
	return []byte(issued.Code), nil
}

func (a codeIssuerAdapter) ResolveEnrollment(ctx context.Context, correlationID string) (string, bool, error) {
	return a.svc.ResolveEnrollment(ctx, correlationID)
}

type machineLinkerAdapter struct {
	attempts onboard.AttemptStore
	machines machines.Service
}

var _ onboard.MachineLinker = machineLinkerAdapter{}

func (a machineLinkerAdapter) LinkCorrelatedNode(ctx context.Context, correlationID, nodeID string) error {
	attempt, err := a.attempts.GetAttemptByCorrelation(ctx, correlationID)
	if err != nil {
		var absent onboard.ErrOpNotFound
		if errors.As(err, &absent) {
			// The legacy operation did not originate from a Machine attempt; do
			// not guess a relationship from host/name/locator data.
			return nil
		}
		return err
	}
	_, err = a.machines.LinkNode(ctx, attempt.MachineID, nodeID, correlationID)
	return err
}

func (a machineLinkerAdapter) RecordCorrelatedTrust(ctx context.Context, correlationID string, conn onboard.Conn) error {
	attempt, err := a.attempts.GetAttemptByCorrelation(ctx, correlationID)
	if err != nil {
		var absent onboard.ErrOpNotFound
		if errors.As(err, &absent) {
			return nil
		}
		return err
	}
	state := machines.HostKeyUnverified
	if conn.HostKeyFingerprint != "" {
		state = machines.HostKeyVerified
	}
	_, err = a.machines.UpsertTrust(ctx, machines.TrustRecord{MachineID: attempt.MachineID, ClientKeyRef: conn.ClientKeyRef, ClientKeyFingerprint: conn.ClientKeyFingerprint, HostKeyFingerprint: conn.HostKeyFingerprint, HostKeyState: state})
	return err
}

// nodeRevisionRecorderAdapter stamps a node's provenance revision after
// onboarding verifies it ONLINE, via a read-modify-write over the registry (the
// registry Update overwrites the whole editable surface, so it reloads the node
// and rewrites only the revision). Best-effort at the call site: a failure is a
// non-fatal note on the op, never a failed onboarding.
type nodeRevisionRecorderAdapter struct {
	svc registry.Service
}

var _ onboard.NodeRevisionRecorder = nodeRevisionRecorderAdapter{}

// NewNodeRevisionRecorder builds the onboard NodeRevisionRecorder over the
// registry service so main.go can wire it without reaching into the domain.
func NewNodeRevisionRecorder(svc registry.Service) onboard.NodeRevisionRecorder {
	return nodeRevisionRecorderAdapter{svc: svc}
}

func (a nodeRevisionRecorderAdapter) RecordRevision(ctx context.Context, nodeID, revision string) error {
	node, err := a.svc.Get(ctx, nodeID)
	if err != nil {
		return err
	}
	_, err = a.svc.Update(ctx, registry.UpdateInput{
		ID:           node.ID,
		Name:         node.Name,
		Endpoint:     node.Endpoint,
		Capabilities: node.Capabilities,
		Scopes:       node.Scopes,
		Revision:     revision,
	})
	return err
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

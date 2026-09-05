package supervision

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

type ActionController interface {
	GetRun(context.Context, uuid.UUID) (*domain.Run, error)
	ContinueRun(context.Context, uuid.UUID, string, string) error
	StopRun(context.Context, uuid.UUID) error
	ParkRun(context.Context, uuid.UUID, string, *time.Time) error
	WakeRun(context.Context, uuid.UUID, string) error
}

type ActionService struct {
	repo       *Repository
	controller ActionController
	now        func() time.Time
}

func NewActionService(repo *Repository, controller ActionController) *ActionService {
	return &ActionService{repo: repo, controller: controller, now: time.Now}
}

func (s *ActionService) Request(ctx context.Context, req *domainpb.RequestCohortWatchActionRequest) (*domainpb.RequestCohortWatchActionResponse, error) {
	if req == nil {
		return nil, errors.New("action request is required")
	}
	request := proto.Clone(req).(*domainpb.RequestCohortWatchActionRequest)
	applyActionDefaults(request)
	action, replay, err := s.repo.RequestAction(ctx, request)
	if err != nil {
		return nil, err
	}
	watch, _, err := s.repo.Get(ctx, request.GetWatchId())
	if err != nil {
		return nil, err
	}
	if replay {
		return &domainpb.RequestCohortWatchActionResponse{Action: action, Watch: watch, IdempotentReplay: true}, nil
	}
	if reason := validateActionRequest(watch, request); reason != "" {
		action, err = s.repo.TransitionAction(ctx, action.GetActionId(), domainpb.WatchActionState_WATCH_ACTION_STATE_REQUESTED, domainpb.WatchActionState_WATCH_ACTION_STATE_REJECTED, reason)
		return &domainpb.RequestCohortWatchActionResponse{Action: action, Watch: watch}, err
	}
	action, err = s.repo.TransitionAction(ctx, action.GetActionId(), domainpb.WatchActionState_WATCH_ACTION_STATE_REQUESTED, domainpb.WatchActionState_WATCH_ACTION_STATE_ACCEPTED, "authorized")
	if err != nil {
		return nil, err
	}
	if action.GetState() != domainpb.WatchActionState_WATCH_ACTION_STATE_ACCEPTED {
		return &domainpb.RequestCohortWatchActionResponse{Action: action, Watch: watch}, nil
	}
	action, err = s.apply(ctx, watch, action)
	return &domainpb.RequestCohortWatchActionResponse{Action: action, Watch: watch}, err
}

func (s *ActionService) List(ctx context.Context, watchID string, limit uint32) (*domainpb.ListCohortWatchActionsResponse, error) {
	actions, err := s.repo.ListActions(ctx, strings.TrimSpace(watchID), int(limit))
	return &domainpb.ListCohortWatchActionsResponse{Actions: actions}, err
}

// RecoverPending retries accepted actions after restart. Nudge and continue
// actions remain accepted while the target is actively streaming and are
// delivered only after it reaches a safe resting turn boundary.
func (s *ActionService) RecoverPending(ctx context.Context) (int, error) {
	actions, err := s.repo.ListPendingActions(ctx, 100)
	if err != nil {
		return 0, err
	}
	applied := 0
	for _, action := range actions {
		watch, _, err := s.repo.Get(ctx, action.GetWatchId())
		if err != nil {
			continue
		}
		if action.GetState() == domainpb.WatchActionState_WATCH_ACTION_STATE_REQUESTED {
			continue // startup never invents an authorization decision.
		}
		updated, err := s.apply(ctx, watch, action)
		if err == nil && updated.GetState() == domainpb.WatchActionState_WATCH_ACTION_STATE_APPLIED {
			applied++
		}
	}
	return applied, nil
}

func (s *ActionService) apply(ctx context.Context, watch *domainpb.CohortWatch, action *domainpb.WatchAction) (*domainpb.WatchAction, error) {
	// The emergency switch also governs already accepted interventions.
	disabled, _, err := NewPolicyStore(s.repo.db, nil).Disabled(ctx)
	if err != nil {
		return action, err
	}
	if disabled {
		return s.rejectAccepted(ctx, action, "supervision is disabled")
	}
	terminalWake := watch.GetStatus() == domainpb.WatchStatus_WATCH_STATUS_TERMINAL && (action.GetKind() == domainpb.WatchActionKind_WATCH_ACTION_KIND_WAKE_PARENT || action.GetKind() == domainpb.WatchActionKind_WATCH_ACTION_KIND_ESCALATE)
	if watch.GetStatus() != domainpb.WatchStatus_WATCH_STATUS_ACTIVE && !terminalWake {
		return s.rejectAccepted(ctx, action, "watch no longer authorizes intervention")
	}
	if reason := validateActionRequest(watch, &domainpb.RequestCohortWatchActionRequest{Kind: action.GetKind(), TargetRunId: action.GetTargetRunId(), Authority: action.GetAuthority(), RequestedBy: action.GetRequestedBy()}); reason != "" {
		return s.rejectAccepted(ctx, action, reason)
	}

	if action.GetKind() == domainpb.WatchActionKind_WATCH_ACTION_KIND_OBSERVE {
		return s.repo.TransitionAction(ctx, action.GetActionId(), domainpb.WatchActionState_WATCH_ACTION_STATE_ACCEPTED, domainpb.WatchActionState_WATCH_ACTION_STATE_APPLIED, "observation recorded")
	}
	if action.GetKind() == domainpb.WatchActionKind_WATCH_ACTION_KIND_ESCALATE && strings.TrimSpace(action.GetTargetRunId()) == "" {
		return s.repo.TransitionAction(ctx, action.GetActionId(), domainpb.WatchActionState_WATCH_ACTION_STATE_ACCEPTED, domainpb.WatchActionState_WATCH_ACTION_STATE_APPLIED, "escalation recorded")
	}
	if s.controller == nil {
		return action, errors.New("run action controller is unavailable")
	}
	targetID, err := uuid.Parse(action.GetTargetRunId())
	if err != nil {
		return s.rejectAccepted(ctx, action, "target run id is invalid")
	}
	run, err := s.controller.GetRun(ctx, targetID)
	if err != nil {
		return action, err
	}
	switch action.GetKind() {
	case domainpb.WatchActionKind_WATCH_ACTION_KIND_NUDGE, domainpb.WatchActionKind_WATCH_ACTION_KIND_CONTINUE:
		if run.Status.IsActive() {
			return action, nil // safe-turn delivery remains durably accepted.
		}
		if run.Status != domain.RunStatusNeedsReview {
			return s.rejectAccepted(ctx, action, "target is not at a resumable non-terminal turn boundary")
		}
		if err := s.controller.ContinueRun(ctx, targetID, boundedActionMessage(action), action.GetIdempotencyKey()); err != nil {
			return action, err
		}
	case domainpb.WatchActionKind_WATCH_ACTION_KIND_STOP:
		if allowed, reason := domain.CanStopRun(run); !allowed {
			return s.rejectAccepted(ctx, action, reason)
		}
		if err := s.controller.StopRun(ctx, targetID); err != nil {
			return action, err
		}
	case domainpb.WatchActionKind_WATCH_ACTION_KIND_PARK:
		if targetID.String() != watch.GetSpec().GetParentRunId() {
			return s.rejectAccepted(ctx, action, "only the family parent can be parked for supervision")
		}
		if allowed, reason := domain.CanParkRun(run); !allowed {
			return s.rejectAccepted(ctx, action, reason)
		}
		deadline := watch.GetSpec().GetTriggers().GetDeadline()
		var deadlineTime *time.Time
		if deadline.IsValid() {
			value := deadline.AsTime()
			deadlineTime = &value
		}
		if err := s.controller.ParkRun(ctx, targetID, watch.GetWatchId(), deadlineTime); err != nil {
			return action, err
		}
	case domainpb.WatchActionKind_WATCH_ACTION_KIND_ESCALATE, domainpb.WatchActionKind_WATCH_ACTION_KIND_WAKE_PARENT:
		if targetID.String() != watch.GetSpec().GetParentRunId() {
			return s.rejectAccepted(ctx, action, "wake target must be the family parent")
		}
		if run.Status != domain.RunStatusParked {
			return s.rejectAccepted(ctx, action, "family parent is not parked")
		}
		if err := s.controller.WakeRun(ctx, targetID, boundedActionMessage(action)); err != nil {
			return action, err
		}
	default:
		return s.rejectAccepted(ctx, action, "unsupported action kind")
	}
	return s.repo.TransitionAction(ctx, action.GetActionId(), domainpb.WatchActionState_WATCH_ACTION_STATE_ACCEPTED, domainpb.WatchActionState_WATCH_ACTION_STATE_APPLIED, "action applied")
}

func (s *ActionService) rejectAccepted(ctx context.Context, action *domainpb.WatchAction, reason string) (*domainpb.WatchAction, error) {
	return s.repo.TransitionAction(ctx, action.GetActionId(), domainpb.WatchActionState_WATCH_ACTION_STATE_ACCEPTED, domainpb.WatchActionState_WATCH_ACTION_STATE_REJECTED, reason)
}

func applyActionDefaults(request *domainpb.RequestCohortWatchActionRequest) {
	if request.MaximumCount == 0 {
		request.MaximumCount = 1
		if request.GetKind() == domainpb.WatchActionKind_WATCH_ACTION_KIND_NUDGE {
			request.MaximumCount = 3
		}
	}
	if request.Cooldown == nil && request.GetKind() == domainpb.WatchActionKind_WATCH_ACTION_KIND_NUDGE {
		request.Cooldown = durationpb.New(5 * time.Minute)
	}
}

func validateActionRequest(watch *domainpb.CohortWatch, request *domainpb.RequestCohortWatchActionRequest) string {
	if request.GetWaiveHardValidationGate() {
		return "hard validation gates cannot be waived by supervision"
	}
	if request.GetAuthority() == domainpb.WatchAuthority_WATCH_AUTHORITY_UNSPECIFIED || strings.TrimSpace(request.GetRequestedBy()) == "" {
		return "requester identity and authority are required"
	}
	if request.GetAuthority() == domainpb.WatchAuthority_WATCH_AUTHORITY_FAMILY_PARENT && request.GetRequestedBy() != watch.GetSpec().GetParentRunId() {
		return "family-parent authority does not match the persisted parent run"
	}
	if request.GetAuthority() == domainpb.WatchAuthority_WATCH_AUTHORITY_SYSTEM {
		switch request.GetKind() {
		case domainpb.WatchActionKind_WATCH_ACTION_KIND_OBSERVE, domainpb.WatchActionKind_WATCH_ACTION_KIND_PARK, domainpb.WatchActionKind_WATCH_ACTION_KIND_ESCALATE, domainpb.WatchActionKind_WATCH_ACTION_KIND_WAKE_PARENT:
		default:
			return "system authority cannot directly mutate child execution"
		}
	}
	if request.GetKind() == domainpb.WatchActionKind_WATCH_ACTION_KIND_OBSERVE {
		return ""
	}
	target := strings.TrimSpace(request.GetTargetRunId())
	if target == "" && request.GetKind() != domainpb.WatchActionKind_WATCH_ACTION_KIND_ESCALATE {
		return "target run id is required"
	}
	if target == watch.GetSpec().GetParentRunId() {
		if request.GetKind() != domainpb.WatchActionKind_WATCH_ACTION_KIND_PARK && request.GetKind() != domainpb.WatchActionKind_WATCH_ACTION_KIND_WAKE_PARENT && request.GetKind() != domainpb.WatchActionKind_WATCH_ACTION_KIND_ESCALATE {
			return "only park, escalate, or wake-parent may target the family parent"
		}
		return ""
	}
	for _, subject := range watch.GetSpec().GetSubjects() {
		if subject.GetRunId() == target {
			return ""
		}
	}
	return fmt.Sprintf("target run %q is not a member of the watched cohort", target)
}

func boundedActionMessage(action *domainpb.WatchAction) string {
	message := strings.TrimSpace(action.GetMessage())
	if message == "" {
		message = strings.TrimSpace(action.GetRationale())
	}
	if len(message) > 3500 {
		message = message[:3500]
	}
	return fmt.Sprintf("[cohort supervision action=%s evidence=%s]\n%s", action.GetKind(), action.GetActionId(), message)
}

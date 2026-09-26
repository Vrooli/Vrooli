package goals

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"swarm-manager/internal/attempt"
	"swarm-manager/internal/attemptstore"
	"swarm-manager/internal/storage"
	"swarm-manager/internal/transitionrunner"

	"google.golang.org/protobuf/types/known/structpb"
)

// SetTransitionRunner installs the shared lifecycle owner. Goal adapters keep
// only immutable snapshot construction and proposal projection.
func (h *Handler) SetTransitionRunner(runner *transitionrunner.Runner) { h.transitionRunner = runner }

// RegisterTransitionAdapter contributes the three declared goal transitions.
func (h *Handler) RegisterTransitionAdapter(registrar transitionrunner.Registrar) {
	registrar.RegisterInput("goal.plan", h.buildGoalInput)
	registrar.RegisterInput("goal.discover", h.buildGoalInput)
	registrar.RegisterInput("milestone.review", h.buildMilestoneInput)
	registrar.RegisterApply("apply_goal_proposal", h.applyGoalProposal)
	registrar.RegisterApply("apply_milestone_review", h.applyMilestoneReview)
}

func (h *Handler) buildGoalInput(_ context.Context, subjectRef string) (transitionrunner.Snapshot, error) {
	goal, err := h.service.Get(strings.TrimSpace(subjectRef))
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	snapshot, err := workflowSnapshot(goal)
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	input, err := structpb.NewValue(map[string]any{"entity": map[string]any{"kind": "goal", "name": goal.Goal.Name, "version": goal.Goal.Updated}, "snapshot": snapshot, "supported_ops": h.supportedOps()})
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	return transitionrunner.SnapshotFromSubject(input, goal.Goal, nil)
}

func (h *Handler) buildMilestoneInput(ctx context.Context, subjectRef string) (transitionrunner.Snapshot, error) {
	parts := strings.SplitN(strings.TrimSpace(subjectRef), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return transitionrunner.Snapshot{}, fmt.Errorf("invalid milestone subject reference %q", subjectRef)
	}
	goal, err := h.service.Get(parts[0])
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	found := false
	for _, milestone := range goal.Goal.Milestones {
		if milestone.Name == parts[1] {
			found = true
			break
		}
	}
	if !found {
		return transitionrunner.Snapshot{}, fmt.Errorf("milestone not found")
	}
	snapshot, err := workflowSnapshot(goal)
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	input, err := structpb.NewValue(map[string]any{"entity": map[string]any{"kind": "milestone", "name": parts[1], "goalName": goal.Goal.Name, "version": goal.Goal.Updated}, "snapshot": snapshot, "supported_ops": h.supportedOps()})
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	return transitionrunner.SnapshotFromSubject(input, milestoneSnapshotSubject{Goal: goal.Goal, Milestone: parts[1]}, nil)
}

func (h *Handler) applyGoalProposal(ctx context.Context, subjectRef string, outcome transitionrunner.Outcome) error {
	return h.applyGoalOutcome(ctx, strings.TrimSpace(subjectRef), "", outcome)
}

func (h *Handler) applyMilestoneReview(ctx context.Context, subjectRef string, outcome transitionrunner.Outcome) error {
	parts := strings.SplitN(strings.TrimSpace(subjectRef), "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid milestone subject reference %q", subjectRef)
	}
	return h.applyGoalOutcome(ctx, parts[0], parts[1], outcome)
}

func (h *Handler) applyGoalOutcome(ctx context.Context, goalName, milestone string, outcome transitionrunner.Outcome) error {
	if h.proposalRecorder == nil {
		return fmt.Errorf("goal workflow proposal recorder is not configured")
	}
	goal, err := h.service.Get(goalName)
	if err != nil {
		return err
	}
	currentVersion, err := goalEntityVersion(goal.Goal, milestone)
	if err != nil {
		return err
	}
	if currentVersion != outcome.EntityVersion {
		return fmt.Errorf("goal changed after workflow start; re-run against the current snapshot")
	}
	value := &structpb.Value{}
	if err := value.UnmarshalJSON([]byte(`{"result":` + string(outcome.Result) + `}`)); err != nil {
		return err
	}
	result, err := decodeWorkflowResult(value, outcome.TransitionKey)
	if err != nil {
		return err
	}
	var existing goalTransitionReceipt
	if found, err := storage.ReadJSON(h.transitionReceiptPath(goalName, outcome.ExecutionID), &existing); err != nil {
		return err
	} else if found {
		return nil
	}
	if milestone != "" && result.Outcome == "delivered" {
		if _, err := h.service.MarkMilestoneDeliveredWithVerdicts(goalName, milestone, result.CriterionVerdicts); err != nil {
			return err
		}
	}
	if err := h.saveWorkflowAttempt(goalName, milestone, outcome, result); err != nil {
		return err
	}
	receipt, err := h.proposalRecorder.RecordGoalWorkflowProposals(ctx, GoalWorkflowProposal{GoalName: goalName, GoalVersion: outcome.EntityVersion, Title: outcome.TransitionKey + " for " + goal.Goal.Title, Summary: result.Summary, ExecutionID: outcome.ExecutionID, WorkflowKey: outcome.TransitionKey, Payloads: result.Payloads})
	if err != nil {
		return err
	}
	return storage.WriteJSONAtomic(h.transitionReceiptPath(goalName, outcome.ExecutionID), newGoalTransitionReceipt(outcome.ExecutionID, receipt))
}

func (h *Handler) saveWorkflowAttempt(goalName, milestone string, outcome transitionrunner.Outcome, result decodedWorkflowResult) error {
	// Execution identity is storage provenance, not payload lifecycle state. It
	// makes replay after a crash idempotent while Attempt itself remains a pure
	// projection joined with transitionrun at read time.
	relativeDir := filepath.Join("attempts", outcome.TransitionKey)
	if milestone != "" {
		relativeDir = filepath.Join(relativeDir, milestone)
	}
	relativeDir = filepath.Join(relativeDir, outcome.ExecutionID)
	root := h.service.GoalDir(goalName)
	round, err := attemptstore.NextRoundNumber(root, relativeDir)
	if err != nil {
		return err
	}
	proposals := make([]attempt.Proposal, 0, len(result.Payloads))
	for index, payload := range result.Payloads {
		proposals = append(proposals, attempt.Proposal{ID: fmt.Sprintf("proposal-%d", index+1), Type: "workflow_proposal", Payload: payload})
	}
	subjectKind, subjectRef := "goal", goalName
	if milestone != "" {
		subjectKind, subjectRef = "milestone", goalName+"/"+milestone
	}
	return attemptstore.SaveRound(root, relativeDir, attempt.Attempt{
		SubjectKind: subjectKind, SubjectRef: subjectRef, TransitionKey: outcome.TransitionKey,
		RoundNum: round, Status: "complete", GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Assessment: result.Summary, Verdict: result.Outcome, Evidence: result.Evidence, Proposals: proposals,
	})
}

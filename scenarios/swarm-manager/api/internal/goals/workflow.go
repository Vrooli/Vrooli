package goals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/attempt"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/storage"
	"swarm-manager/internal/transitionrun"
	"swarm-manager/internal/transitionrunner"

	"google.golang.org/protobuf/types/known/structpb"
)

// GoalWorkflowProposal is the composed boundary between a transition result
// and the durable agent-session proposal inbox.
type GoalWorkflowProposal struct {
	GoalName, GoalVersion, Title, Summary, ExecutionID, RunID, WorkflowKey string
	Payloads                                                               []string
}

type GoalWorkflowProposalReceipt struct {
	SessionID   string
	ProposalIDs []string
}

// PendingWorkflow is an operator-facing projection of the shared transition
// journal. It intentionally owns no lifecycle state.
type PendingWorkflow struct {
	GoalName      string `json:"goal_name"`
	ExecutionID   string `json:"execution_id"`
	Transition    string `json:"transition"`
	Milestone     string `json:"milestone,omitempty"`
	GoalVersion   string `json:"goal_version"`
	Stale         bool   `json:"stale"`
	Attempts      int    `json:"attempts,omitempty"`
	LastAttemptAt string `json:"last_attempt_at,omitempty"`
	LastError     string `json:"last_error,omitempty"`
}

type milestoneSnapshotSubject struct {
	Goal      Goal   `json:"goal"`
	Milestone string `json:"milestone"`
}

func goalEntityVersion(goal Goal, milestone string) (string, error) {
	if milestone == "" {
		return transitionrunner.DigestJSON("entity", goal)
	}
	return transitionrunner.DigestJSON("entity", milestoneSnapshotSubject{Goal: goal, Milestone: milestone})
}

// goalTransitionReceipt is provenance for the external proposal-recording
// side effect. Workflow lifecycle ownership remains solely in transitionrun.
type goalTransitionReceipt struct {
	ExecutionID string   `json:"execution_id"`
	SessionID   string   `json:"session_id"`
	ProposalIDs []string `json:"proposal_ids"`
	AppliedAt   string   `json:"applied_at"`
}

func (h *Handler) transitionReceiptPath(goalName, executionID string) string {
	return filepath.Join(h.service.GoalDir(goalName), "transition-receipts", executionID+".json")
}

func goalTransitionSubject(c transitionrun.Correlation) (goalName, milestone string, ok bool) {
	switch c.TransitionKey {
	case "goal.plan", "goal.discover":
		return strings.TrimSpace(c.SubjectRef), "", strings.TrimSpace(c.SubjectRef) != ""
	case "milestone.review":
		parts := strings.SplitN(strings.TrimSpace(c.SubjectRef), "/", 2)
		return parts[0], parts[1], len(parts) == 2 && parts[0] != "" && parts[1] != ""
	default:
		return "", "", false
	}
}

// ListPendingWorkflows projects every unapplied goal correlation from the
// shared journal. The journal, retry diagnostics, and sweeper are generic.
func (h *Handler) ListPendingWorkflows() ([]PendingWorkflow, error) {
	if h.transitionRunner == nil {
		return nil, errors.New("transition runner is not configured")
	}
	goalsList, err := h.service.LoadAllRaw()
	if err != nil {
		return nil, err
	}
	versions := make(map[string]string, len(goalsList))
	for _, goal := range goalsList {
		version, versionErr := goalEntityVersion(goal, "")
		if versionErr != nil {
			return nil, versionErr
		}
		versions[goal.Name] = version
	}
	correlations, err := h.transitionRunner.ListUnapplied()
	if err != nil {
		return nil, err
	}
	out := make([]PendingWorkflow, 0, len(correlations))
	for _, correlation := range correlations {
		goalName, milestone, ok := goalTransitionSubject(correlation)
		if !ok {
			continue
		}
		version, exists := versions[goalName]
		if milestone != "" {
			for _, goal := range goalsList {
				if goal.Name != goalName {
					continue
				}
				version, err = goalEntityVersion(goal, milestone)
				if err != nil {
					return nil, err
				}
				break
			}
		}
		out = append(out, PendingWorkflow{
			GoalName: goalName, ExecutionID: correlation.ExecutionID, Transition: correlation.TransitionKey,
			Milestone: milestone, GoalVersion: correlation.EntityVersion, Stale: !exists || version != correlation.EntityVersion,
			Attempts: correlation.ApplyAttemptCount, LastAttemptAt: correlation.LastApplyAttemptTime, LastError: correlation.LastApplyError,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].GoalName != out[j].GoalName {
			return out[i].GoalName < out[j].GoalName
		}
		return out[i].ExecutionID < out[j].ExecutionID
	})
	return out, nil
}

type AppliedWorkflow struct {
	ExecutionID    string   `json:"execution_id"`
	SessionID      string   `json:"session_id"`
	ProposalIDs    []string `json:"proposal_ids"`
	Outcome        string   `json:"outcome,omitempty"`
	AlreadyApplied bool     `json:"already_applied,omitempty"`
}

func (h *Handler) ApplyWorkflowResult(ctx context.Context, goalName, executionID string) (AppliedWorkflow, error) {
	if h.transitionRunner == nil {
		return AppliedWorkflow{}, errors.New("transition runner is not configured")
	}
	if strings.TrimSpace(executionID) == "" {
		return AppliedWorkflow{}, errors.New("workflow execution id is required")
	}
	var prior goalTransitionReceipt
	already, err := storage.ReadJSON(h.transitionReceiptPath(goalName, executionID), &prior)
	if err != nil {
		return AppliedWorkflow{}, err
	}
	correlation, err := h.transitionRunner.ApplyExecution(ctx, executionID)
	if err != nil {
		return AppliedWorkflow{}, err
	}
	correlationGoal, _, ok := goalTransitionSubject(correlation)
	if !ok || correlationGoal != goalName {
		return AppliedWorkflow{}, errors.New("goal workflow belongs to a different goal")
	}
	var receipt goalTransitionReceipt
	found, err := storage.ReadJSON(h.transitionReceiptPath(goalName, executionID), &receipt)
	if err != nil || !found {
		return AppliedWorkflow{}, errors.New("transition completed without a goal proposal receipt")
	}
	return AppliedWorkflow{ExecutionID: executionID, SessionID: receipt.SessionID, ProposalIDs: receipt.ProposalIDs, Outcome: correlation.Outcome, AlreadyApplied: already}, nil
}

type decodedWorkflowResult struct {
	Outcome, Summary  string
	Payloads          []string
	CriterionVerdicts []CriterionVerdict
	Evidence          []attempt.Evidence
}

func decodeWorkflowResult(output *structpb.Value, transition string) (decodedWorkflowResult, error) {
	if output == nil {
		return decodedWorkflowResult{}, errors.New("goal workflow did not return an output")
	}
	payload, ok := output.AsInterface().(map[string]any)
	if !ok {
		return decodedWorkflowResult{}, errors.New("goal workflow output is not an object")
	}
	result, ok := payload["result"].(map[string]any)
	if !ok {
		return decodedWorkflowResult{}, errors.New("goal workflow output omitted result")
	}
	outcome, _ := result["outcome"].(string)
	summary, _ := result["summary"].(string)
	if transition == "milestone.review" {
		outcome, _ = result["verdict"].(string)
		summary, _ = result["assessment"].(string)
	}
	if strings.TrimSpace(outcome) == "" || strings.TrimSpace(summary) == "" {
		return decodedWorkflowResult{}, errors.New("goal workflow result omitted outcome or summary")
	}
	values, _ := result["proposals"].([]any)
	payloads := make([]string, 0, len(values))
	for _, value := range values {
		encoded, err := json.Marshal(value)
		if err != nil {
			return decodedWorkflowResult{}, fmt.Errorf("encode workflow proposal: %w", err)
		}
		payloads = append(payloads, string(encoded))
	}
	verdicts := make([]CriterionVerdict, 0)
	if transition == "milestone.review" {
		if raw, ok := result["criterion_verdicts"].([]any); ok {
			encoded, _ := json.Marshal(raw)
			_ = json.Unmarshal(encoded, &verdicts)
		}
	}
	evidence := make([]attempt.Evidence, 0)
	if raw, ok := result["evidence"].([]any); ok {
		encoded, marshalErr := json.Marshal(raw)
		if marshalErr != nil {
			return decodedWorkflowResult{}, fmt.Errorf("encode workflow evidence: %w", marshalErr)
		}
		if err := json.Unmarshal(encoded, &evidence); err != nil {
			return decodedWorkflowResult{}, fmt.Errorf("decode workflow evidence: %w", err)
		}
	}
	return decodedWorkflowResult{Outcome: outcome, Summary: summary, Payloads: payloads, CriterionVerdicts: verdicts, Evidence: evidence}, nil
}

func (h *Handler) StartMilestoneReviewsForTerminalItem(ctx context.Context, kind, name string, status backlog.BacklogStatus) {
	if !backlog.IsTerminalStatus(status) {
		return
	}
	itemRef := strings.TrimSpace(kind) + "/" + strings.TrimSpace(name)
	go func() {
		if err := h.startEligibleMilestoneReviews(ctx, itemRef); err != nil {
			slog.Warn("[goals] milestone review auto-trigger failed", "item", itemRef, "err", err)
		}
	}()
}

func (h *Handler) startEligibleMilestoneReviews(ctx context.Context, itemRef string) error {
	if h.transitionRunner == nil {
		return errors.New("transition runner is not configured")
	}
	goalsList, err := h.service.List()
	if err != nil {
		return err
	}
	for _, listed := range goalsList {
		goal, err := h.service.Get(listed.Goal.Name)
		if err != nil {
			return err
		}
		for _, milestone := range goal.Goal.Milestones {
			if milestone.ArchivedAt != nil || len(milestone.AcceptanceCriteria) == 0 || !containsRef(milestone.Items, itemRef) || !allMilestoneItemsTerminal(goal, milestone) {
				continue
			}
			if _, err := h.transitionRunner.StartWith(ctx, "milestone.review", goal.Goal.Name+"/"+milestone.Name, transitionrunner.PreparedInput{FirstRunNodeID: "review", Activity: &transitionrunner.Activity{OwnerType: "milestone", OwnerKind: "goal", OwnerName: goal.Goal.Name + "/" + milestone.Name, Purpose: "milestone_review"}}); err != nil {
				return err
			}
		}
	}
	return nil
}

func allMilestoneItemsTerminal(goal *GoalWithScope, milestone Milestone) bool {
	if len(milestone.Items) == 0 || goal.ScopeEntities == nil {
		return false
	}
	for _, ref := range milestone.Items {
		item, ok := goal.ScopeEntities.Items[ref]
		if !ok || !backlog.IsTerminalStatus(item.Status) {
			return false
		}
	}
	return true
}

func containsRef(refs []string, want string) bool {
	for _, ref := range refs {
		if ref == want {
			return true
		}
	}
	return false
}

func newGoalTransitionReceipt(executionID string, receipt GoalWorkflowProposalReceipt) goalTransitionReceipt {
	return goalTransitionReceipt{ExecutionID: executionID, SessionID: receipt.SessionID, ProposalIDs: receipt.ProposalIDs, AppliedAt: time.Now().UTC().Format(time.RFC3339Nano)}
}

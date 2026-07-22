package goals

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/storage"

	"google.golang.org/protobuf/types/known/structpb"
)

// GoalWorkflowProposal is the composed boundary between the goal workflow
// projection and the durable agent-session proposal inbox.
type GoalWorkflowProposal struct {
	GoalName, GoalVersion, Title, Summary, ExecutionID, RunID, WorkflowKey string
	Payloads                                                               []string
}

type GoalWorkflowProposalReceipt struct {
	SessionID   string
	ProposalIDs []string
}

type workflowPending struct {
	ExecutionID      string `json:"execution_id"`
	DefinitionDigest string `json:"definition_digest"`
	Transition       string `json:"transition"`
	GoalVersion      string `json:"goal_version"`
	Milestone        string `json:"milestone,omitempty"`
}

type workflowApplied struct {
	ExecutionID string   `json:"execution_id"`
	SessionID   string   `json:"session_id"`
	ProposalIDs []string `json:"proposal_ids"`
	AppliedAt   string   `json:"applied_at"`
}

func (h *Handler) workflowPendingPath(goalName, executionID string) string {
	return filepath.Join(h.service.GoalDir(goalName), "workflow-pending", executionID+".json")
}

func (h *Handler) workflowAppliedPath(goalName, executionID string) string {
	return filepath.Join(h.service.GoalDir(goalName), "workflow-applied", executionID+".json")
}

func (h *Handler) writeWorkflowPending(goalName string, pending workflowPending) error {
	if strings.TrimSpace(pending.ExecutionID) == "" || strings.TrimSpace(pending.DefinitionDigest) == "" {
		return errors.New("workflow correlation is incomplete")
	}
	return storage.WriteJSONAtomic(h.workflowPendingPath(goalName, pending.ExecutionID), pending)
}

func (h *Handler) readWorkflowPending(goalName, executionID string) (workflowPending, error) {
	var pending workflowPending
	found, err := storage.ReadJSON(h.workflowPendingPath(goalName, executionID), &pending)
	if err != nil {
		return workflowPending{}, err
	}
	if !found || pending.ExecutionID != executionID {
		return workflowPending{}, errors.New("goal workflow is not pending for this goal")
	}
	return pending, nil
}

func (h *Handler) applyWorkflow(ctx context.Context, goalName, executionID string) (map[string]any, error) {
	if h.workflow == nil || h.proposalRecorder == nil {
		return nil, errors.New("goal workflow application is not configured")
	}
	if strings.TrimSpace(executionID) == "" {
		return nil, errors.New("workflow execution id is required")
	}
	var existing workflowApplied
	if found, err := storage.ReadJSON(h.workflowAppliedPath(goalName, executionID), &existing); err == nil && found {
		return map[string]any{"execution_id": executionID, "session_id": existing.SessionID, "proposal_ids": existing.ProposalIDs, "already_applied": true}, nil
	} else if err != nil {
		return nil, err
	}
	pending, err := h.readWorkflowPending(goalName, executionID)
	if err != nil {
		return nil, err
	}
	completion, err := h.workflow.CollectWorkflow(ctx, executionID)
	if err != nil {
		return nil, err
	}
	if completion.ExecutionID != executionID || completion.DefinitionDigest != pending.DefinitionDigest || !completion.Succeeded {
		return nil, errors.New("goal workflow terminal snapshot is not applicable")
	}
	if !workflowInputMatches(completion.Input, goalName, pending) {
		return nil, errors.New("goal workflow input does not match pending correlation")
	}
	goal, err := h.service.Get(goalName)
	if err != nil {
		return nil, err
	}
	if goal.Goal.Updated != pending.GoalVersion {
		return nil, errors.New("goal changed after workflow start; re-run against the current snapshot")
	}
	result, err := decodeWorkflowResult(completion.Output, pending.Transition)
	if err != nil {
		return nil, err
	}
	receipt, err := h.proposalRecorder.RecordGoalWorkflowProposals(ctx, GoalWorkflowProposal{GoalName: goalName, GoalVersion: pending.GoalVersion, Title: pending.Transition + " for " + goal.Goal.Title, Summary: result.Summary, ExecutionID: executionID, WorkflowKey: pending.Transition, Payloads: result.Payloads})
	if err != nil {
		return nil, err
	}
	applied := workflowApplied{ExecutionID: executionID, SessionID: receipt.SessionID, ProposalIDs: receipt.ProposalIDs, AppliedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	if err := storage.WriteJSONAtomic(h.workflowAppliedPath(goalName, executionID), applied); err != nil {
		return nil, err
	}
	if err := os.Remove(h.workflowPendingPath(goalName, executionID)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return map[string]any{"execution_id": executionID, "session_id": receipt.SessionID, "proposal_ids": receipt.ProposalIDs, "outcome": result.Outcome}, nil
}

func workflowInputMatches(input *structpb.Value, goalName string, pending workflowPending) bool {
	if input == nil {
		return false
	}
	payload, ok := input.AsInterface().(map[string]any)
	if !ok {
		return false
	}
	entity, ok := payload["entity"].(map[string]any)
	if !ok || entity["version"] != pending.GoalVersion {
		return false
	}
	if pending.Transition == "milestone.review" {
		return entity["kind"] == "milestone" && entity["goalName"] == goalName && entity["name"] == pending.Milestone
	}
	return entity["kind"] == "goal" && entity["name"] == goalName
}

type decodedWorkflowResult struct {
	Outcome, Summary string
	Payloads         []string
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
		encoded, marshalErr := json.Marshal(value)
		if marshalErr != nil {
			return decodedWorkflowResult{}, fmt.Errorf("encode workflow proposal: %w", marshalErr)
		}
		payloads = append(payloads, string(encoded))
	}
	return decodedWorkflowResult{Outcome: outcome, Summary: summary, Payloads: payloads}, nil
}

// StartMilestoneReviewsForTerminalItem is the short request-path callback used
// by backlog review-decision. Expensive workflow startup is self-dispatched so
// a terminal item response never waits on Agent Manager.
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
	if h.workflow == nil {
		return errors.New("goal workflow service is unavailable")
	}
	goalsList, err := h.service.List()
	if err != nil {
		return err
	}
	for _, listed := range goalsList {
		goal, getErr := h.service.Get(listed.Goal.Name)
		if getErr != nil {
			return getErr
		}
		for _, milestone := range goal.Goal.Milestones {
			if milestone.ArchivedAt != nil || len(milestone.AcceptanceCriteria) == 0 || !containsRef(milestone.Items, itemRef) || !allMilestoneItemsTerminal(goal, milestone) {
				continue
			}
			if err := h.startMilestoneReview(ctx, goal, milestone); err != nil {
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

func (h *Handler) startMilestoneReview(ctx context.Context, goal *GoalWithScope, milestone Milestone) error {
	locator, err := h.registry.ResolveWorkflow("milestone.review")
	if err != nil {
		return err
	}
	snapshot, err := workflowSnapshot(goal)
	if err != nil {
		return err
	}
	input, err := structpb.NewValue(map[string]any{"entity": map[string]any{"kind": "milestone", "name": milestone.Name, "goalName": goal.Goal.Name, "version": goal.Goal.Updated}, "snapshot": snapshot})
	if err != nil {
		return err
	}
	start, err := h.workflow.StartWorkflow(ctx, WorkflowInvocation{Owner: locator.Owner, WorkflowKey: locator.Key, Input: input, IdempotencyKey: "milestone-review/" + goal.Goal.Name + "/" + milestone.Name + "/" + milestoneMemberSetDigest(milestone.Items), FirstRunNodeID: "review"})
	if err != nil {
		return err
	}
	return h.writeWorkflowPending(goal.Goal.Name, workflowPending{ExecutionID: start.ExecutionID, DefinitionDigest: start.DefinitionDigest, Transition: "milestone.review", GoalVersion: goal.Goal.Updated, Milestone: milestone.Name})
}

func milestoneMemberSetDigest(refs []string) string {
	copyRefs := append([]string(nil), refs...)
	sort.Strings(copyRefs)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(copyRefs, "\n"))))
}

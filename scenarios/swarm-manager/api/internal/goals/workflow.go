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

// ErrWorkflowNotReady marks the one apply failure that is expected and
// transient: the run has not finished yet. The sweeper retries these silently,
// so anything that is *not* this sentinel is worth an operator's attention.
var ErrWorkflowNotReady = errors.New("goal workflow result is not ready")

// ErrWorkflowUnavailable marks the other transient failure: the workflow engine
// could not be reached. It is deliberately distinct from ErrWorkflowNotReady —
// "the run is still going" and "we cannot ask" are different facts — but the
// sweeper treats both as retry-later, so an agent-manager outage never stamps a
// permanent-looking failure onto a healthy correlation record.
var ErrWorkflowUnavailable = errors.New("goal workflow engine is unavailable")

type workflowPending struct {
	ExecutionID      string `json:"execution_id"`
	DefinitionDigest string `json:"definition_digest"`
	Transition       string `json:"transition"`
	GoalVersion      string `json:"goal_version"`
	Milestone        string `json:"milestone,omitempty"`
	// Attempts, LastAttemptAt and LastError are apply diagnostics. They live on
	// the correlation record rather than in sweeper memory so a stuck workflow
	// explains itself across restarts, and so `goals workflow-pending` can say
	// why a record is stuck instead of only that it is.
	Attempts      int    `json:"attempts,omitempty"`
	LastAttemptAt string `json:"last_attempt_at,omitempty"`
	LastError     string `json:"last_error,omitempty"`
}

// PendingWorkflow is one goal workflow result that has not been applied. It is
// the operator-facing projection of workflowPending.
type PendingWorkflow struct {
	GoalName    string `json:"goal_name"`
	ExecutionID string `json:"execution_id"`
	Transition  string `json:"transition"`
	Milestone   string `json:"milestone,omitempty"`
	GoalVersion string `json:"goal_version"`
	// Stale reports that the goal changed after the workflow started. Apply
	// refuses these by design, so the result can never land and the run has to
	// be repeated against the current snapshot.
	Stale         bool   `json:"stale"`
	Attempts      int    `json:"attempts,omitempty"`
	LastAttemptAt string `json:"last_attempt_at,omitempty"`
	LastError     string `json:"last_error,omitempty"`
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

func (h *Handler) workflowPendingDir(goalName string) string {
	return filepath.Join(h.service.GoalDir(goalName), "workflow-pending")
}

// listGoalPendingWorkflows reads every unapplied correlation for one goal.
// A goal with no pending directory is the normal case, not an error.
func (h *Handler) listGoalPendingWorkflows(goalName, goalVersion string) ([]PendingWorkflow, error) {
	entries, err := os.ReadDir(h.workflowPendingDir(goalName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]PendingWorkflow, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		executionID := strings.TrimSuffix(entry.Name(), ".json")
		pending, readErr := h.readWorkflowPending(goalName, executionID)
		if readErr != nil {
			// A correlation we cannot parse is not a reason to hide the rest.
			slog.Warn("[goals] unreadable workflow correlation", "goal", goalName, "execution_id", executionID, "err", readErr)
			continue
		}
		out = append(out, PendingWorkflow{
			GoalName: goalName, ExecutionID: pending.ExecutionID, Transition: pending.Transition,
			Milestone: pending.Milestone, GoalVersion: pending.GoalVersion,
			Stale:         goalVersion != "" && goalVersion != pending.GoalVersion,
			Attempts:      pending.Attempts,
			LastAttemptAt: pending.LastAttemptAt,
			LastError:     pending.LastError,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ExecutionID < out[j].ExecutionID })
	return out, nil
}

// ListPendingWorkflows reports every goal workflow result awaiting application,
// across all goals. This is the surface that makes a stalled apply hop visible;
// before it existed, results accumulated on disk with nothing reporting them.
func (h *Handler) ListPendingWorkflows() ([]PendingWorkflow, error) {
	goalsList, err := h.service.LoadAllRaw()
	if err != nil {
		return nil, err
	}
	out := make([]PendingWorkflow, 0)
	for _, goal := range goalsList {
		pending, listErr := h.listGoalPendingWorkflows(goal.Name, goal.Updated)
		if listErr != nil {
			return nil, listErr
		}
		out = append(out, pending...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].GoalName != out[j].GoalName {
			return out[i].GoalName < out[j].GoalName
		}
		return out[i].ExecutionID < out[j].ExecutionID
	})
	return out, nil
}

// recordApplyFailure stamps the apply diagnostics onto the correlation record.
// Best-effort: a diagnostics write must never mask the apply error itself.
func (h *Handler) recordApplyFailure(goalName, executionID string, cause error) {
	pending, err := h.readWorkflowPending(goalName, executionID)
	if err != nil {
		return
	}
	pending.Attempts++
	pending.LastAttemptAt = time.Now().UTC().Format(time.RFC3339Nano)
	pending.LastError = cause.Error()
	if err := storage.WriteJSONAtomic(h.workflowPendingPath(goalName, executionID), pending); err != nil {
		slog.Warn("[goals] could not record apply failure", "goal", goalName, "execution_id", executionID, "err", err)
	}
}

// AppliedWorkflow is the outcome of one apply attempt.
type AppliedWorkflow struct {
	ExecutionID string   `json:"execution_id"`
	SessionID   string   `json:"session_id"`
	ProposalIDs []string `json:"proposal_ids"`
	Outcome     string   `json:"outcome,omitempty"`
	// AlreadyApplied marks the idempotent replay: the result had landed before,
	// and this call changed nothing.
	AlreadyApplied bool `json:"already_applied,omitempty"`
}

// ApplyWorkflowResult applies one terminal goal workflow result. It is the
// single entry point shared by the REST handler, the Connect service, and the
// sweeper, so the apply rules cannot diverge per caller.
func (h *Handler) ApplyWorkflowResult(ctx context.Context, goalName, executionID string) (AppliedWorkflow, error) {
	return h.applyWorkflow(ctx, goalName, executionID)
}

func (h *Handler) applyWorkflow(ctx context.Context, goalName, executionID string) (AppliedWorkflow, error) {
	if h.workflow == nil || h.proposalRecorder == nil {
		return AppliedWorkflow{}, errors.New("goal workflow application is not configured")
	}
	if strings.TrimSpace(executionID) == "" {
		return AppliedWorkflow{}, errors.New("workflow execution id is required")
	}
	var existing workflowApplied
	if found, err := storage.ReadJSON(h.workflowAppliedPath(goalName, executionID), &existing); err == nil && found {
		return AppliedWorkflow{ExecutionID: executionID, SessionID: existing.SessionID, ProposalIDs: existing.ProposalIDs, AlreadyApplied: true}, nil
	} else if err != nil {
		return AppliedWorkflow{}, err
	}
	pending, err := h.readWorkflowPending(goalName, executionID)
	if err != nil {
		return AppliedWorkflow{}, err
	}
	completion, err := h.workflow.CollectWorkflow(ctx, executionID)
	if err != nil {
		return AppliedWorkflow{}, err
	}
	if completion.ExecutionID != executionID || completion.DefinitionDigest != pending.DefinitionDigest || !completion.Succeeded {
		return AppliedWorkflow{}, errors.New("goal workflow terminal snapshot is not applicable")
	}
	if !workflowInputMatches(completion.Input, goalName, pending) {
		return AppliedWorkflow{}, errors.New("goal workflow input does not match pending correlation")
	}
	goal, err := h.service.Get(goalName)
	if err != nil {
		return AppliedWorkflow{}, err
	}
	if goal.Goal.Updated != pending.GoalVersion {
		return AppliedWorkflow{}, errors.New("goal changed after workflow start; re-run against the current snapshot")
	}
	result, err := decodeWorkflowResult(completion.Output, pending.Transition)
	if err != nil {
		return AppliedWorkflow{}, err
	}
	if pending.Transition == "milestone.review" && result.Outcome == "delivered" {
		if _, err := h.service.MarkMilestoneDelivered(goalName, pending.Milestone); err != nil {
			return AppliedWorkflow{}, err
		}
	}
	receipt, err := h.proposalRecorder.RecordGoalWorkflowProposals(ctx, GoalWorkflowProposal{GoalName: goalName, GoalVersion: pending.GoalVersion, Title: pending.Transition + " for " + goal.Goal.Title, Summary: result.Summary, ExecutionID: executionID, WorkflowKey: pending.Transition, Payloads: result.Payloads})
	if err != nil {
		return AppliedWorkflow{}, err
	}
	applied := workflowApplied{ExecutionID: executionID, SessionID: receipt.SessionID, ProposalIDs: receipt.ProposalIDs, AppliedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	if err := storage.WriteJSONAtomic(h.workflowAppliedPath(goalName, executionID), applied); err != nil {
		return AppliedWorkflow{}, err
	}
	if err := os.Remove(h.workflowPendingPath(goalName, executionID)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return AppliedWorkflow{}, err
	}
	return AppliedWorkflow{ExecutionID: executionID, SessionID: receipt.SessionID, ProposalIDs: receipt.ProposalIDs, Outcome: result.Outcome}, nil
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
	input, err := structpb.NewValue(map[string]any{"entity": map[string]any{"kind": "milestone", "name": milestone.Name, "goalName": goal.Goal.Name, "version": goal.Goal.Updated}, "snapshot": snapshot, "supported_ops": h.supportedOps()})
	if err != nil {
		return err
	}
	start, err := h.workflow.StartWorkflow(ctx, WorkflowInvocation{
		Owner: locator.Owner, WorkflowKey: locator.Key, Input: input,
		IdempotencyKey: "milestone-review/" + goal.Goal.Name + "/" + milestone.Name + "/" + milestoneMemberSetDigest(milestone.Items), FirstRunNodeID: "review",
		ActivityOwnerType: "milestone", ActivityOwnerKind: "goal", ActivityOwnerName: goal.Goal.Name + "/" + milestone.Name, ActivityPurpose: "milestone_review",
	})
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

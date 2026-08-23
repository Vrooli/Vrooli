package captures

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/proposals"
	"swarm-manager/internal/transitionrun"
	"swarm-manager/internal/transitionrunner"

	"github.com/gorilla/mux"
	"google.golang.org/protobuf/types/known/structpb"
)

// classifyFailureReason maps workflow transport failures to a user-facing category.
func classifyFailureReason(err error) string {
	if errors.Is(err, agentmanager.ErrNotAvailable) {
		return FailDependencyUnavailable
	}
	return FailAgentError
}

// Classify starts a declared workflow for an immutable snapshot of a capture.
// The workflow is deliberately unable to write capture files; ApplyClassification
// is the only consumer-side mutation point.
func (h *Handler) Classify(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	cap, err := h.loadCapture(id)
	if err != nil {
		if os.IsNotExist(err) {
			apierr.MapError(w, "[captures] classify", apierr.NotFound("capture not found"))
			return
		}
		apierr.MapError(w, "[captures] classify", apierr.Internal("failed to load capture"))
		return
	}

	cap.Status, cap.FailureReason = "classifying", ""
	if err := h.writeCapture(cap); err != nil {
		apierr.MapError(w, "[captures] classify", apierr.Internal("failed to update capture"))
		return
	}
	if h.transitionRunner == nil {
		apierr.MapError(w, "[captures] classify", apierr.Unavailable("agent-manager is not available — try again once it's running"))
		return
	}
	correlation, err := h.transitionRunner.Start(r.Context(), "capture.classify", cap.ID)
	if err != nil {
		cap.Status, cap.FailureReason = "failed", classifyFailureReason(err)
		_ = h.writeCapture(cap)
		if errors.Is(err, agentmanager.ErrNotAvailable) {
			apierr.MapError(w, "[captures] classify", apierr.Unavailable("agent-manager is not available — try again once it's running"))
			return
		}
		apierr.MapError(w, "[captures] classify", apierr.Internal("failed to start classification workflow"))
		return
	}
	h.invalidateTopologyGraph()
	_ = httputil.JSON(w, map[string]any{"capture": cap, "workflow_execution_id": correlation.ExecutionID, "workflow_definition_digest": correlation.DefinitionDigest, "created": time.Now().UTC().Format(time.RFC3339)})
}

func (h *Handler) buildClassificationInput(ctx context.Context, id string) (transitionrunner.Snapshot, error) {
	cap, err := h.loadCapture(id)
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	attachments := make([]any, len(cap.Attachments))
	for i, attachment := range cap.Attachments {
		// Agent Manager bindings are text-only and the capture is stored in the
		// cache class. Pass an absolute, readable path so the read-only agent
		// workspace can inspect the actual image rather than a project-root
		// relative path that does not exist there.
		attachments[i] = filepath.Join(h.captureDir(cap.ID), filepath.FromSlash(attachment))
	}
	grounding := map[string]any{"status": "degraded", "reason": "grounding provider is not configured"}
	if h.groundingProvider != nil {
		if candidate, groundingErr := h.groundingProvider.BuildCaptureGrounding(ctx, cap.Text); groundingErr != nil {
			grounding = map[string]any{"status": "degraded", "reason": groundingErr.Error()}
		} else if candidate != nil {
			grounding = candidate
		}
	}
	groundingValue, err := boundedStructValue(grounding)
	if err != nil {
		return transitionrunner.Snapshot{}, fmt.Errorf("classification grounding: %w", err)
	}
	input, err := structpb.NewValue(map[string]any{
		"capture":   map[string]any{"id": cap.ID, "text": cap.Text, "attachments": attachments, "note": cap.Note, "version": captureVersion(cap)},
		"grounding": groundingValue,
	})
	if err != nil {
		return transitionrunner.Snapshot{}, fmt.Errorf("classification input: %w", err)
	}
	return transitionrunner.SnapshotFromSubject(input, cap, grounding)
}

// normalizeStructValue converts Go's typed slices and structs into the JSON
// shaped map/slice values accepted by structpb.NewValue. Grounding is assembled
// from several domain packages, so keeping this boundary explicit prevents a
// harmless provider model change from making capture intake fail to spawn.
func normalizeStructValue(value any) (any, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var normalized any
	if err := json.Unmarshal(encoded, &normalized); err != nil {
		return nil, err
	}
	return normalized, nil
}

const maxGroundingBytes = 48 * 1024

// boundedStructValue keeps the grounding packet useful while respecting the
// workflow input envelope. A project can have hundreds of active goals and
// search payloads are intentionally rich; sending all of that to one agent
// would make a healthy capture fail before the workflow starts.
func boundedStructValue(value any) (any, error) {
	normalized, err := normalizeStructValue(value)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}
	if len(encoded) <= maxGroundingBytes {
		return normalized, nil
	}
	compact := compactStructValue(normalized, 0)
	if packet, ok := compact.(map[string]any); ok {
		packet["status"] = "degraded"
		packet["degraded_reasons"] = appendStringReason(packet["degraded_reasons"], "grounding packet compacted to fit workflow input budget")
	}
	encoded, err = json.Marshal(compact)
	if err != nil {
		return nil, err
	}
	if len(encoded) <= maxGroundingBytes {
		return compact, nil
	}
	return map[string]any{
		"status":           "degraded",
		"degraded_reasons": []any{"grounding packet exceeded workflow input budget; detailed context omitted"},
	}, nil
}

func compactStructValue(value any, depth int) any {
	if depth >= 4 {
		return nil
	}
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		if len(keys) > 20 {
			keys = keys[:20]
		}
		result := make(map[string]any, len(keys))
		for _, key := range keys {
			result[key] = compactStructValue(typed[key], depth+1)
		}
		return result
	case []any:
		limit := len(typed)
		if limit > 12 {
			limit = 12
		}
		result := make([]any, 0, limit)
		for _, item := range typed[:limit] {
			result = append(result, compactStructValue(item, depth+1))
		}
		return result
	case string:
		if len(typed) > 600 {
			return typed[:600] + "…"
		}
		return typed
	default:
		return value
	}
}

func appendStringReason(existing any, reason string) []any {
	result := []any{}
	if reasons, ok := existing.([]any); ok {
		result = append(result, reasons...)
	}
	return append(result, reason)
}

// ApplyClassification validates a terminal workflow snapshot and makes its
// typed result visible to the capture UI. It is safe to retry after success.
func (h *Handler) ApplyClassification(w http.ResponseWriter, r *http.Request) {
	h.classificationMu.Lock()
	defer h.classificationMu.Unlock()

	id, executionID := mux.Vars(r)["id"], mux.Vars(r)["executionID"]
	_, err := h.loadCapture(id)
	if err != nil && !os.IsNotExist(err) {
		apierr.MapError(w, "[captures] apply classification", apierr.Internal("failed to load capture"))
		return
	}
	// A completed projection is the durable idempotency marker. The original
	// terminal result stays inspectable in Agent Manager; Swarm does not apply it
	// again merely because delivery was retried after a restart.
	if h.transitionRunner == nil {
		apierr.MapError(w, "[captures] apply classification", apierr.Unavailable("agent-manager is not available"))
		return
	}
	correlation, err := h.transitionRunner.Apply(r.Context(), "capture.classify", executionID)
	if err != nil {
		if errors.Is(err, agentmanager.ErrWorkflowNotReady) {
			apierr.MapError(w, "[captures] apply classification", apierr.Conflict("classification workflow is not terminal"))
			return
		}
		var changed *transitionrun.EntityVersionChangedError
		if errors.As(err, &changed) {
			apierr.MapError(w, "[captures] apply classification", apierr.Conflict("capture changed after classification started"))
			return
		}
		apierr.MapError(w, "[captures] apply classification", apierr.Internal("failed to collect classification workflow"))
		return
	}
	if correlation.SubjectRef != id {
		apierr.MapError(w, "[captures] apply classification", apierr.Conflict("classification workflow does not match capture"))
		return
	}
	cap, err := h.loadCapture(id)
	if err != nil {
		if os.IsNotExist(err) {
			_ = httputil.JSON(w, map[string]any{"capture_id": id, "capture_deleted": true, "outcome": correlation.Outcome})
			return
		}
		apierr.MapError(w, "[captures] apply classification", apierr.Internal("failed to load applied capture"))
		return
	}
	_ = httputil.JSON(w, map[string]any{"capture": cap, "outcome": correlation.Outcome})
}

func (h *Handler) applyClassificationOutcome(ctx context.Context, id string, outcome transitionrunner.Outcome) error {
	cap, err := h.loadCapture(id)
	if err != nil {
		return err
	}
	var result classificationResult
	if err := json.Unmarshal(outcome.Result, &result); err != nil {
		return fmt.Errorf("decode classification result: %w", err)
	}
	if result.Outcome != outcome.Name {
		return fmt.Errorf("classification outcome does not match result")
	}
	if h.proposalRecorder == nil {
		return fmt.Errorf("capture proposal recorder is not configured")
	}
	if err := normalizeClassificationResult(&result); err != nil {
		return err
	}
	cap.FailureReason = ""
	switch result.Outcome {
	case "suggested", "research":
		resultBytes, err := captureProposalPayload(id, result)
		if err != nil {
			return err
		}
		if err := h.proposalRecorder.RecordCaptureProposals(ctx, id, truncate(cap.Text, 100), result.Reason, captureVersion(cap), outcome.ExecutionID, classificationRunID(outcome), resultBytes); err != nil {
			return err
		}
	case "discarded", "abstained":
		resultBytes, err := captureProposalPayload(id, result)
		if err != nil {
			return err
		}
		if err := h.proposalRecorder.RecordCaptureProposals(ctx, id, truncate(cap.Text, 100), result.Reason, captureVersion(cap), outcome.ExecutionID, classificationRunID(outcome), resultBytes); err != nil {
			return err
		}
	default:
		return fmt.Errorf("classification workflow emitted an unknown outcome")
	}
	if err := os.RemoveAll(h.captureDir(id)); err != nil {
		return fmt.Errorf("remove triaged capture: %w", err)
	}
	if h.aiIndexer != nil {
		if err := h.aiIndexer.DeleteCapture(ctx, id); err != nil {
			slog.Debug("captures: semantic index delete failed", "capture_id", id, "err", err)
		}
	}
	h.invalidateTopologyGraph()
	return nil
}

// normalizeClassificationResult enforces the conservative landing contract at
// the mutation boundary. A research outcome is exactly one investigation, even
// when a model redundantly emits both research_item and duplicate items.
func normalizeClassificationResult(result *classificationResult) error {
	if result == nil || result.Outcome != "research" {
		return nil
	}
	var selected *classificationItem
	if result.ResearchItem != nil {
		item := *result.ResearchItem
		selected = &item
	} else {
		for _, item := range result.Items {
			if strings.EqualFold(strings.TrimSpace(item.Kind), "research") {
				candidate := item
				selected = &candidate
				break
			}
		}
	}
	if selected == nil {
		return fmt.Errorf("research classification emitted no research item")
	}
	selected.Kind = "research"
	result.Items = []classificationItem{*selected}
	result.ResearchItem = nil
	return nil
}

func classificationRunID(outcome transitionrunner.Outcome) string {
	for i := len(outcome.Attempts) - 1; i >= 0; i-- {
		if runID := strings.TrimSpace(outcome.Attempts[i].RunID); runID != "" {
			return runID
		}
	}
	return strings.TrimSpace(outcome.ExecutionID)
}

type classificationResult struct {
	Outcome      string               `json:"outcome"`
	Items        []classificationItem `json:"items"`
	Goal         *classificationGoal  `json:"goal,omitempty"`
	ResearchItem *classificationItem  `json:"research_item,omitempty"`
	Reason       string               `json:"reason"`
}

func captureProposalPayload(captureID string, result classificationResult) ([]byte, error) {
	items := append([]classificationItem(nil), result.Items...)
	if result.ResearchItem != nil {
		research := *result.ResearchItem
		research.Kind = "research"
		items = append(items, research)
	}
	mutations := make([]proposals.Mutation, 0, len(items)+1)
	itemRefs := make([]string, 0, len(items))
	for index, item := range items {
		name := sanitizeCaptureItemName(item.Name)
		if name == "" {
			name = sanitizeCaptureItemName(item.Title)
		}
		if name == "" {
			name = "capture-item-" + captureID
		}
		if index > 0 {
			name = fmt.Sprintf("%s-%d", name, index+1)
		}
		kind := strings.ToLower(strings.TrimSpace(item.Kind))
		ref := kind + "/" + name
		itemRefs = append(itemRefs, ref)
		mutations = append(mutations, proposals.Mutation{ID: fmt.Sprintf("capture-item-%d", index+1), Op: proposals.OpAddItem, Item: &proposals.ItemSpec{
			Kind: kind, Name: name, Title: strings.TrimSpace(item.Title), Description: item.Description, Priority: item.Priority,
			Tags: deduplicateTags(item.Tags), DependsOn: append([]string(nil), item.DependsOn...), Milestone: item.Milestone, Effort: item.Effort,
			AcceptanceAllow: append([]string(nil), item.AcceptanceAllow...), AcceptanceDeny: append([]string(nil), item.AcceptanceDeny...), SpawnedFrom: captureID,
		}})
	}
	if result.Goal != nil {
		goal := &proposals.GoalSpec{Name: result.Goal.Name, Title: result.Goal.Title, Description: result.Goal.Description, Priority: result.Goal.Priority, Targets: append([]string(nil), result.Goal.Targets...), SpawnedFrom: captureID}
		for _, milestone := range result.Goal.Milestones {
			owned := proposals.GoalMilestone{Name: milestone.Name, Title: milestone.Title, Description: milestone.Description, AcceptanceCriteria: append([]string(nil), milestone.AcceptanceCriteria...), DependsOn: append([]string(nil), milestone.DependsOn...), SpawnedFrom: captureID}
			for index, item := range items {
				if item.Milestone == milestone.Name && index < len(itemRefs) {
					owned.Items = append(owned.Items, itemRefs[index])
				}
			}
			goal.Milestones = append(goal.Milestones, owned)
		}
		mutations = append(mutations, proposals.Mutation{ID: "capture-goal", Op: proposals.OpCreateGoal, Goal: goal})
	}
	payload := proposals.Proposal{Form: proposals.FormMutationList, Rationale: strings.TrimSpace(result.Reason), BaseVersion: captureID, Mutations: mutations}
	return json.Marshal(payload)
}

func deduplicateTags(tags []string) []string {
	result := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		result = append(result, tag)
	}
	return result
}

func captureVersion(cap *capture) string {
	payload, _ := json.Marshal(struct {
		ID          string   `json:"id"`
		Text        string   `json:"text"`
		Attachments []string `json:"attachments"`
		Note        string   `json:"note"`
	}{cap.ID, cap.Text, cap.Attachments, cap.Note})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

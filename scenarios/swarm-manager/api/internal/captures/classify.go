package captures

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
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

	cap.Status, cap.FailureReason, cap.Classification = "classifying", "", nil
	cap.WorkflowExecutionID, cap.WorkflowDefinitionDigest = "", ""
	cap.WorkflowEntityVersion = captureVersion(cap)
	if err := h.writeCapture(cap); err != nil {
		apierr.MapError(w, "[captures] classify", apierr.Internal("failed to update capture"))
		return
	}
	if rmErr := os.Remove(h.classificationPath(id)); rmErr != nil && !os.IsNotExist(rmErr) {
		apierr.MapError(w, "[captures] classify", apierr.Internal("failed to clear prior classification"))
		return
	}

	start, err := h.startClassificationWorkflow(r, cap)
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
	cap.WorkflowExecutionID, cap.WorkflowDefinitionDigest = start.ExecutionID, start.DefinitionDigest
	if err := h.writeCapture(cap); err != nil {
		apierr.MapError(w, "[captures] classify", apierr.Internal("failed to persist classification workflow"))
		return
	}
	h.invalidateTopologyGraph()
	_ = httputil.JSON(w, map[string]any{"capture": cap, "workflow_execution_id": start.ExecutionID, "workflow_definition_digest": start.DefinitionDigest, "created": time.Now().UTC().Format(time.RFC3339)})
}

func (h *Handler) startClassificationWorkflow(r *http.Request, cap *capture) (agentmanager.WorkflowStart, error) {
	if h.classificationWorkflow == nil {
		return agentmanager.WorkflowStart{}, agentmanager.ErrNotAvailable
	}
	attachments := make([]any, len(cap.Attachments))
	for i, attachment := range cap.Attachments {
		attachments[i] = attachment
	}
	input, err := structpb.NewValue(map[string]any{"capture": map[string]any{"id": cap.ID, "text": cap.Text, "attachments": attachments, "version": cap.WorkflowEntityVersion}})
	if err != nil {
		return agentmanager.WorkflowStart{}, fmt.Errorf("classification input: %w", err)
	}
	return h.classificationWorkflow.StartWorkflow(r.Context(), agentmanager.Invocation{
		Owner: "swarm-manager", WorkflowKey: "swarm-manager/capture-classify", Input: input,
		IdempotencyKey: "capture-classify/" + cap.ID + "/" + cap.WorkflowEntityVersion, FirstRunNodeID: "classify",
	})
}

// ApplyClassification validates a terminal workflow snapshot and makes its
// typed result visible to the capture UI. It is safe to retry after success.
func (h *Handler) ApplyClassification(w http.ResponseWriter, r *http.Request) {
	h.classificationMu.Lock()
	defer h.classificationMu.Unlock()

	id, executionID := mux.Vars(r)["id"], mux.Vars(r)["executionID"]
	cap, err := h.loadCapture(id)
	if err != nil {
		if os.IsNotExist(err) {
			apierr.MapError(w, "[captures] apply classification", apierr.NotFound("capture not found"))
			return
		}
		apierr.MapError(w, "[captures] apply classification", apierr.Internal("failed to load capture"))
		return
	}
	if cap.WorkflowExecutionID == "" || cap.WorkflowExecutionID != executionID {
		apierr.MapError(w, "[captures] apply classification", apierr.Conflict("classification workflow does not match capture"))
		return
	}
	// A completed projection is the durable idempotency marker. The original
	// terminal result stays inspectable in Agent Manager; Swarm does not apply it
	// again merely because delivery was retried after a restart.
	if cap.Status == "classified" && cap.Classification != nil {
		_ = httputil.JSON(w, map[string]any{"capture": cap, "already_applied": true})
		return
	}
	if h.classificationWorkflow == nil {
		apierr.MapError(w, "[captures] apply classification", apierr.Unavailable("agent-manager is not available"))
		return
	}
	completion, err := h.classificationWorkflow.CollectWorkflow(r.Context(), executionID)
	if err != nil {
		if errors.Is(err, agentmanager.ErrWorkflowNotReady) {
			apierr.MapError(w, "[captures] apply classification", apierr.Conflict("classification workflow is not terminal"))
			return
		}
		apierr.MapError(w, "[captures] apply classification", apierr.Internal("failed to collect classification workflow"))
		return
	}
	if completion.DefinitionDigest != cap.WorkflowDefinitionDigest || completion.Status != domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED {
		apierr.MapError(w, "[captures] apply classification", apierr.Conflict("classification workflow terminal snapshot is not applicable"))
		return
	}
	if err := validateClassificationInput(completion.Input, cap); err != nil {
		apierr.MapError(w, "[captures] apply classification", apierr.Conflict("%s", err))
		return
	}
	result, err := parseClassificationResult(completion.Output)
	if err != nil {
		apierr.MapError(w, "[captures] apply classification", apierr.Conflict("classification workflow emitted an invalid result"))
		return
	}
	cap.FailureReason = ""
	switch result.Outcome {
	case "suggested":
		cap.Status = "classified"
		cap.Classification = &classification{Items: result.Items, ClassifiedAt: time.Now().UTC().Format(time.RFC3339)}
		data, _ := json.MarshalIndent(cap.Classification, "", "  ")
		if err := os.WriteFile(h.classificationPath(id), data, 0o600); err != nil {
			apierr.MapError(w, "[captures] apply classification", apierr.Internal("failed to persist classification"))
			return
		}
	case "discarded", "abstained":
		cap.Status, cap.Classification = "classified", &classification{Items: []classificationItem{}, ClassifiedAt: time.Now().UTC().Format(time.RFC3339)}
	default:
		apierr.MapError(w, "[captures] apply classification", apierr.Conflict("classification workflow emitted an unknown outcome"))
		return
	}
	if err := h.writeCapture(cap); err != nil {
		apierr.MapError(w, "[captures] apply classification", apierr.Internal("failed to update capture"))
		return
	}
	h.invalidateTopologyGraph()
	_ = httputil.JSON(w, map[string]any{"capture": cap, "outcome": result.Outcome, "reason": result.Reason})
}

type classificationResult struct {
	Outcome string               `json:"outcome"`
	Items   []classificationItem `json:"items"`
	Reason  string               `json:"reason"`
}

func parseClassificationResult(output *structpb.Value) (classificationResult, error) {
	if output == nil {
		return classificationResult{}, errors.New("missing output")
	}
	payload, ok := output.AsInterface().(map[string]any)
	if !ok {
		return classificationResult{}, errors.New("output must be an object")
	}
	raw, ok := payload["result"]
	if !ok {
		return classificationResult{}, errors.New("missing result")
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return classificationResult{}, err
	}
	var result classificationResult
	if err := json.Unmarshal(data, &result); err != nil {
		return classificationResult{}, err
	}
	if result.Outcome == "suggested" && len(result.Items) == 0 {
		return classificationResult{}, errors.New("suggested outcome requires items")
	}
	if (result.Outcome == "discarded" || result.Outcome == "abstained") && strings.TrimSpace(result.Reason) == "" {
		return classificationResult{}, errors.New("terminal outcome requires reason")
	}
	return result, nil
}

func validateClassificationInput(input *structpb.Value, cap *capture) error {
	if input == nil {
		return errors.New("classification workflow omitted input")
	}
	payload, ok := input.AsInterface().(map[string]any)
	if !ok {
		return errors.New("classification workflow input is invalid")
	}
	raw, ok := payload["capture"].(map[string]any)
	if !ok {
		return errors.New("classification workflow input omitted capture")
	}
	if raw["id"] != cap.ID || raw["version"] != cap.WorkflowEntityVersion {
		return errors.New("capture changed since classification workflow started")
	}
	return nil
}

func captureVersion(cap *capture) string {
	payload, _ := json.Marshal(struct {
		ID          string   `json:"id"`
		Text        string   `json:"text"`
		Attachments []string `json:"attachments"`
	}{cap.ID, cap.Text, cap.Attachments})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

package captures

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
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

	cap.Status, cap.FailureReason, cap.Classification = "classifying", "", nil
	if err := h.writeCapture(cap); err != nil {
		apierr.MapError(w, "[captures] classify", apierr.Internal("failed to update capture"))
		return
	}
	if rmErr := os.Remove(h.classificationPath(id)); rmErr != nil && !os.IsNotExist(rmErr) {
		apierr.MapError(w, "[captures] classify", apierr.Internal("failed to clear prior classification"))
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

func (h *Handler) buildClassificationInput(_ context.Context, id string) (transitionrunner.Snapshot, error) {
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
	version := captureVersion(cap)
	input, err := structpb.NewValue(map[string]any{"capture": map[string]any{"id": cap.ID, "text": cap.Text, "attachments": attachments, "note": cap.Note, "version": version}})
	if err != nil {
		return transitionrunner.Snapshot{}, fmt.Errorf("classification input: %w", err)
	}
	return transitionrunner.Snapshot{Input: input, EntityVersion: version}, nil
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
	// A completed projection is the durable idempotency marker. The original
	// terminal result stays inspectable in Agent Manager; Swarm does not apply it
	// again merely because delivery was retried after a restart.
	if cap.Status == "classified" && cap.Classification != nil {
		_ = httputil.JSON(w, map[string]any{"capture": cap, "already_applied": true})
		return
	}
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
	cap, err = h.loadCapture(id)
	if err != nil {
		apierr.MapError(w, "[captures] apply classification", apierr.Internal("failed to load applied capture"))
		return
	}
	_ = httputil.JSON(w, map[string]any{"capture": cap, "outcome": correlation.Outcome})
}

func (h *Handler) applyClassificationOutcome(_ context.Context, id string, outcome transitionrunner.Outcome) error {
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
	cap.FailureReason = ""
	switch result.Outcome {
	case "suggested":
		cap.Status = "classified"
		cap.Classification = &classification{Items: result.Items, ClassifiedAt: time.Now().UTC().Format(time.RFC3339)}
		data, _ := json.MarshalIndent(cap.Classification, "", "  ")
		if err := os.WriteFile(h.classificationPath(id), data, 0o600); err != nil {
			return err
		}
	case "discarded", "abstained":
		cap.Status, cap.Classification = "classified", &classification{Items: []classificationItem{}, ClassifiedAt: time.Now().UTC().Format(time.RFC3339)}
	default:
		return fmt.Errorf("classification workflow emitted an unknown outcome")
	}
	if err := h.writeCapture(cap); err != nil {
		return err
	}
	h.invalidateTopologyGraph()
	return nil
}

type classificationResult struct {
	Outcome string               `json:"outcome"`
	Items   []classificationItem `json:"items"`
	Reason  string               `json:"reason"`
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

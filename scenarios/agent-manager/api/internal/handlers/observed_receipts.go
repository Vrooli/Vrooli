package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"agent-manager/internal/runreport"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/eventbus"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	eventspb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// GetObservedReceipts reads durable, post-response observations from Vrooli
// Events. Receipt absence is represented explicitly as an empty observation
// set; it is never interpreted as a failed or incomplete Agent Manager run.
func (h *Handler) GetObservedReceipts(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}
	if _, err := h.svc.GetRun(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			writeSimpleError(w, r, "limit", "limit must be between 1 and 100")
			return
		}
		limit = parsed
	}
	if !h.receipts.Enabled() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": runreport.AvailabilityUnavailable, "observations": []any{}, "reason": "vrooli-events observations are not configured"})
		return
	}
	observations, err := h.receipts.ReceiptQuery(r.Context(), id.String(), limit)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": runreport.AvailabilityDegraded, "observations": []any{}, "reason": "vrooli-events observations unavailable"})
		return
	}
	observations = verifiedReceiptObservations(observations, id.String())
	availability := runreport.Availability{State: runreport.AvailabilityAvailable}
	message := ""
	if len(observations) == 0 {
		availability = h.emptyReceiptAvailability(r.Context())
		message = availability.Reason
	}
	payload := map[string]any{"status": availability.State, "observations": observations}
	if message != "" {
		payload["reason"] = message
	}
	writeJSON(w, http.StatusOK, payload)
}

func (h *Handler) emptyReceiptAvailability(ctx context.Context) runreport.Availability {
	if h.receiptAvailability == nil {
		return runreport.Availability{State: runreport.AvailabilityUnobserved}
	}
	availability := h.receiptAvailability(ctx)
	if availability.State == "" {
		return runreport.Availability{State: runreport.AvailabilityUnobserved}
	}
	return availability
}

// verifiedReceiptObservations keeps the raw receipt payload behind the explicit
// drill-down boundary while refusing unverified or incorrectly correlated data.
func verifiedReceiptObservations(items []json.RawMessage, runID string) []json.RawMessage {
	verified := make([]json.RawMessage, 0, len(items))
	for _, item := range items {
		env := &eventspb.EventEnvelope{}
		if protojson.Unmarshal(item, env) != nil || env.EventType != eventbus.ReceiptEventType || env.Attribution == nil || !env.Attribution.Verified || env.Correlation == nil || env.Correlation.AgentRunId != runID {
			continue
		}
		verified = append(verified, item)
	}
	return verified
}

// attachObservedReceipts adds platform evidence to the normal result model
// without changing agent-declared output. Events outage is represented
// explicitly and never changes the run's own terminal result.
func (h *Handler) attachObservedReceipts(ctx context.Context, runID string, result *domainpb.RunResult) {
	if result == nil {
		return
	}
	observations := &domainpb.ReceiptObservations{}
	if !h.receipts.Enabled() {
		observations.State = domainpb.ReceiptObservationState_RECEIPT_OBSERVATION_STATE_UNAVAILABLE
		observations.Reason = "vrooli-events is not configured"
		result.Observations = observations
		return
	}
	raw, err := h.receipts.ReceiptQuery(ctx, runID, 100)
	if err != nil {
		observations.State = domainpb.ReceiptObservationState_RECEIPT_OBSERVATION_STATE_DEGRADED
		observations.Reason = "vrooli-events query unavailable"
		result.Observations = observations
		return
	}
	for _, item := range raw {
		env := &eventspb.EventEnvelope{}
		if protojson.Unmarshal(item, env) != nil || env.EventType != "vrooli.events.receipt.v1" || env.Attribution == nil || !env.Attribution.Verified {
			continue
		}
		data := &eventspb.ReceiptData{}
		if env.Data == nil || anypb.UnmarshalTo(env.Data, data, proto.UnmarshalOptions{}) != nil {
			continue
		}
		receipt := &domainpb.ObservedReceipt{EventId: env.EventId, Outcome: data.Outcome, StatusCode: data.StatusCode, Projection: data.Projection}
		if env.Target != nil {
			receipt.TargetScenario = env.Target.Scenario
			receipt.Operation = env.Target.Operation
		}
		if env.Correlation != nil {
			receipt.AgentRunId = env.Correlation.AgentRunId
			receipt.WorkflowExecutionId = env.Correlation.WorkflowExecutionId
			receipt.WorkflowNodeId = env.Correlation.WorkflowNodeId
			receipt.Attempt = env.Correlation.Attempt
		}
		if env.Attribution != nil {
			receipt.AttributionVerified = env.Attribution.Verified
		}
		if receipt.AgentRunId == runID {
			observations.Receipts = append(observations.Receipts, receipt)
		}
	}
	if len(observations.Receipts) == 0 {
		observations.State = domainpb.ReceiptObservationState_RECEIPT_OBSERVATION_STATE_UNOBSERVED
	} else {
		observations.State = domainpb.ReceiptObservationState_RECEIPT_OBSERVATION_STATE_AVAILABLE
	}
	result.Observations = observations
}

func (h *Handler) workflowObservedReceipts(ctx context.Context, runIDs []string) *domainpb.ReceiptObservations {
	observations := &domainpb.ReceiptObservations{}
	if !h.receipts.Enabled() {
		observations.State = domainpb.ReceiptObservationState_RECEIPT_OBSERVATION_STATE_UNAVAILABLE
		observations.Reason = "vrooli-events is not configured"
		return observations
	}
	wanted := make(map[string]bool, len(runIDs))
	for _, id := range runIDs {
		wanted[id] = true
	}
	for _, runID := range runIDs {
		raw, err := h.receipts.ReceiptQuery(ctx, runID, 100)
		if err != nil {
			observations.State = domainpb.ReceiptObservationState_RECEIPT_OBSERVATION_STATE_DEGRADED
			observations.Reason = "vrooli-events query unavailable"
			return observations
		}
		for _, item := range raw {
			env := &eventspb.EventEnvelope{}
			if protojson.Unmarshal(item, env) != nil || env.EventType != "vrooli.events.receipt.v1" || env.Attribution == nil || !env.Attribution.Verified || env.Correlation == nil || !wanted[env.Correlation.AgentRunId] {
				continue
			}
			data := &eventspb.ReceiptData{}
			if env.Data == nil || anypb.UnmarshalTo(env.Data, data, proto.UnmarshalOptions{}) != nil {
				continue
			}
			receipt := &domainpb.ObservedReceipt{EventId: env.EventId, AgentRunId: env.Correlation.AgentRunId, WorkflowExecutionId: env.Correlation.WorkflowExecutionId, WorkflowNodeId: env.Correlation.WorkflowNodeId, Attempt: env.Correlation.Attempt, Outcome: data.Outcome, StatusCode: data.StatusCode, Projection: data.Projection}
			if env.Target != nil {
				receipt.TargetScenario = env.Target.Scenario
				receipt.Operation = env.Target.Operation
			}
			if env.Attribution != nil {
				receipt.AttributionVerified = env.Attribution.Verified
			}
			observations.Receipts = append(observations.Receipts, receipt)
		}
	}
	if len(observations.Receipts) == 0 {
		observations.State = domainpb.ReceiptObservationState_RECEIPT_OBSERVATION_STATE_UNOBSERVED
	} else {
		observations.State = domainpb.ReceiptObservationState_RECEIPT_OBSERVATION_STATE_AVAILABLE
	}
	return observations
}

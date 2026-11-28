package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"agent-manager/internal/model_eval"
)

type modelEvalBatchRequest struct {
	DecisionID string `json:"decision_id"`
	MaxCases   int    `json:"max_cases,omitempty"`
}

type modelEvalBatchResponse struct {
	State       string              `json:"state"`
	FailureCode string              `json:"failure_code,omitempty"`
	Receipt     *model_eval.Receipt `json:"receipt,omitempty"`
}

// RunModelEvalBatch dispatches only a source-selected, approval-gated batch.
// The request has no prompt, output, or caller-supplied event body field.
func (h *Handler) RunModelEvalBatch(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.modelEvalRunner.Approval == nil {
		writeJSON(w, http.StatusServiceUnavailable, modelEvalBatchResponse{State: "unavailable", FailureCode: "owner_path_unavailable"})
		return
	}
	var request modelEvalBatchRequest
	decoder := json.NewDecoder(io.LimitReader(r.Body, 4097))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, modelEvalBatchResponse{State: "refused", FailureCode: "request_invalid"})
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, modelEvalBatchResponse{State: "refused", FailureCode: "request_invalid"})
		return
	}
	if request.DecisionID != model_eval.AdmissionDecisionID || (request.MaxCases != 0 && (request.MaxCases < 1 || request.MaxCases > model_eval.MaximumCases)) {
		writeJSON(w, http.StatusBadRequest, modelEvalBatchResponse{State: "refused", FailureCode: "admission_scope_invalid"})
		return
	}
	receipt, err := h.modelEvalRunner.Execute(r.Context(), model_eval.Request{DecisionID: request.DecisionID, MaxCases: request.MaxCases})
	if err != nil {
		status := http.StatusBadGateway
		state, code := "failed", "batch_failed"
		if errors.Is(err, model_eval.ErrAdmissionRefused) {
			status, state, code = http.StatusPreconditionFailed, "refused", "admission_or_safety_gate_refused"
		}
		var receiptPtr *model_eval.Receipt
		if receipt.Denominator > 0 {
			receiptPtr = &receipt
		}
		writeJSON(w, status, modelEvalBatchResponse{State: state, FailureCode: code, Receipt: receiptPtr})
		return
	}
	writeJSON(w, http.StatusOK, modelEvalBatchResponse{State: "succeeded", Receipt: &receipt})
}

package agentaccess

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type verifyEvidenceProofRequest struct {
	RunID        string `json:"run_id"`
	GrantReceipt string `json:"grant_receipt"`
	ProofRef     string `json:"proof_reference"`
}

type consumeEvidenceProofRequest struct {
	RunID    string `json:"run_id"`
	ProofRef string `json:"proof_reference"`
}

func (h *HTTPHandler) verifyEvidenceProof(w http.ResponseWriter, r *http.Request) {
	var request verifyEvidenceProofRequest
	if !decodeEvidenceProofJSON(w, r, &request) {
		return
	}
	runID, err := parseEvidenceRunAndProof(request.RunID, request.ProofRef)
	if err != nil || strings.TrimSpace(request.GrantReceipt) == "" || len(request.GrantReceipt) > 256 {
		writeErrorStatus(w, http.StatusBadRequest, errors.New("valid run_id, grant_receipt, and bounded proof_reference are required"))
		return
	}
	if h.Broker == nil {
		writeErrorStatus(w, http.StatusServiceUnavailable, errors.New("browser broker is unavailable"))
		return
	}
	observation, err := h.Broker.VerifyBrowserEvidenceProof(r.Context(), runID, request.GrantReceipt, request.ProofRef)
	writeResponse(w, map[string]any{"observation": observation}, err)
}

func (h *HTTPHandler) consumeEvidenceProof(w http.ResponseWriter, r *http.Request) {
	var request consumeEvidenceProofRequest
	if !decodeEvidenceProofJSON(w, r, &request) {
		return
	}
	runID, err := parseEvidenceRunAndProof(request.RunID, request.ProofRef)
	if err != nil {
		writeErrorStatus(w, http.StatusBadRequest, err)
		return
	}
	if h.Broker == nil {
		writeErrorStatus(w, http.StatusServiceUnavailable, errors.New("browser broker is unavailable"))
		return
	}
	if err := h.Broker.ConsumeBrowserEvidenceProof(r.Context(), runID, request.ProofRef); err != nil {
		writeError(w, err)
		return
	}
	writeResponse(w, map[string]any{"consumed": true}, nil)
}

func decodeEvidenceProofJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeErrorStatus(w, http.StatusBadRequest, errors.New("invalid browser evidence proof request"))
		return false
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeErrorStatus(w, http.StatusBadRequest, errors.New("browser evidence proof request must contain one JSON value"))
		return false
	}
	return true
}

func parseEvidenceRunAndProof(runIDValue, proofRef string) (uuid.UUID, error) {
	runID, err := uuid.Parse(strings.TrimSpace(runIDValue))
	if err != nil || runID == uuid.Nil {
		return uuid.Nil, errors.New("valid run_id is required")
	}
	proofRef = strings.TrimSpace(proofRef)
	if proofRef == "" || len(proofRef) > 256 || strings.ContainsAny(proofRef, "\r\n") {
		return uuid.Nil, errors.New("bounded opaque proof_reference is required")
	}
	return runID, nil
}

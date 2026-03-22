package deployments

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// ApprovalsHandler handles HTTP endpoints for deployment approval gating.
type ApprovalsHandler struct {
	repo ApprovalsRepository
	log  func(string, map[string]interface{})
}

// NewApprovalsHandler creates a new approvals handler.
func NewApprovalsHandler(repo ApprovalsRepository, log func(string, map[string]interface{})) *ApprovalsHandler {
	return &ApprovalsHandler{repo: repo, log: log}
}

// Create handles POST /api/v1/profiles/{id}/approvals
func (h *ApprovalsHandler) Create(w http.ResponseWriter, r *http.Request) {
	profileID := mux.Vars(r)["id"]

	var req CreateApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON: %v"}`, err), http.StatusBadRequest)
		return
	}

	if req.GitCommitHash == "" {
		http.Error(w, `{"error":"git_commit_hash is required"}`, http.StatusBadRequest)
		return
	}
	if req.Platform == "" {
		http.Error(w, `{"error":"platform is required"}`, http.StatusBadRequest)
		return
	}

	approval := &DeploymentApproval{
		ID:            generateApprovalID(),
		ProfileID:     profileID,
		GitCommitHash: req.GitCommitHash,
		Platform:      req.Platform,
		Status:        ApprovalStatusPending,
		ValidationID:  req.ValidationID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := h.repo.Create(r.Context(), approval); err != nil {
		h.log("failed to create approval", map[string]interface{}{"error": err.Error()})
		http.Error(w, fmt.Sprintf(`{"error":"failed to create approval: %v"}`, err), http.StatusInternalServerError)
		return
	}

	h.log("approval created", map[string]interface{}{
		"approval_id": approval.ID,
		"profile_id":  profileID,
		"platform":    req.Platform,
		"commit":      req.GitCommitHash,
	})

	writeJSON(w, http.StatusCreated, approval)
}

// ListByProfile handles GET /api/v1/profiles/{id}/approvals
func (h *ApprovalsHandler) ListByProfile(w http.ResponseWriter, r *http.Request) {
	profileID := mux.Vars(r)["id"]
	commit := r.URL.Query().Get("commit")

	var (
		approvals []*DeploymentApproval
		err       error
	)

	if commit != "" {
		approvals, err = h.repo.ListByCommit(r.Context(), profileID, commit)
	} else {
		approvals, err = h.repo.ListByProfile(r.Context(), profileID, 50)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
		return
	}

	if approvals == nil {
		approvals = []*DeploymentApproval{}
	}

	writeJSON(w, http.StatusOK, approvals)
}

// Get handles GET /api/v1/approvals/{id}
func (h *ApprovalsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	approval, err := h.repo.Get(r.Context(), id)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, approval)
}

// Decide handles POST /api/v1/approvals/{id}/decide
func (h *ApprovalsHandler) Decide(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var req ApprovalDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON: %v"}`, err), http.StatusBadRequest)
		return
	}

	if req.Decision != ApprovalStatusApproved && req.Decision != ApprovalStatusRejected {
		http.Error(w, `{"error":"decision must be 'approved' or 'rejected'"}`, http.StatusBadRequest)
		return
	}
	if req.Reviewer == "" {
		http.Error(w, `{"error":"reviewer is required"}`, http.StatusBadRequest)
		return
	}

	if err := h.repo.UpdateDecision(r.Context(), id, req.Decision, req.Reviewer, req.Notes); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
		return
	}

	h.log("approval decided", map[string]interface{}{
		"approval_id": id,
		"decision":    req.Decision,
		"reviewer":    req.Reviewer,
	})

	// Return the updated approval
	approval, err := h.repo.Get(r.Context(), id)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, approval)
}

// CheckReleaseGate handles GET /api/v1/profiles/{id}/release-gate?commit={hash}
func (h *ApprovalsHandler) CheckReleaseGate(w http.ResponseWriter, r *http.Request) {
	profileID := mux.Vars(r)["id"]
	commit := r.URL.Query().Get("commit")

	if commit == "" {
		http.Error(w, `{"error":"commit query parameter is required"}`, http.StatusBadRequest)
		return
	}

	gate, err := h.repo.CheckReleaseGate(r.Context(), profileID, commit)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, gate)
}

// SetRequiredPlatforms handles PUT /api/v1/profiles/{id}/required-platforms
func (h *ApprovalsHandler) SetRequiredPlatforms(w http.ResponseWriter, r *http.Request) {
	profileID := mux.Vars(r)["id"]

	var req SetRequiredPlatformsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON: %v"}`, err), http.StatusBadRequest)
		return
	}

	if err := h.repo.SetRequiredPlatforms(r.Context(), profileID, req.Platforms); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
		return
	}

	h.log("required platforms set", map[string]interface{}{
		"profile_id": profileID,
		"platforms":  req.Platforms,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"profile_id": profileID,
		"platforms":  req.Platforms,
	})
}

// GetRequiredPlatforms handles GET /api/v1/profiles/{id}/required-platforms
func (h *ApprovalsHandler) GetRequiredPlatforms(w http.ResponseWriter, r *http.Request) {
	profileID := mux.Vars(r)["id"]

	platforms, err := h.repo.GetRequiredPlatforms(r.Context(), profileID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
		return
	}

	if platforms == nil {
		platforms = []string{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"profile_id": profileID,
		"platforms":  platforms,
	})
}

// writeJSON sends a JSON response.
func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// generateApprovalID creates a unique approval ID.
func generateApprovalID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("approval-%s", hex.EncodeToString(b))
}

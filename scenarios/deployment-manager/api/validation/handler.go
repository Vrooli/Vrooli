package validation

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"deployment-manager/deployments"

	"github.com/gorilla/mux"
)

// Handler provides HTTP endpoints for visual validation management.
type Handler struct {
	repo          Repository
	sqlRepo       *SQLRepository                      // concrete type for WithTx; nil in unit tests
	approvalsRepo *deployments.SQLApprovalsRepository // nil when bridging is not configured
	db            *sql.DB                             // nil in unit tests
	videoDir      string
	log           func(string, map[string]interface{})
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("val-%x", b[:8])
}

func generateApprovalID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("approval-%s", hex.EncodeToString(b))
}

// NewHandler creates a new validation handler with approval bridging support.
func NewHandler(repo *SQLRepository, approvalsRepo *deployments.SQLApprovalsRepository, db *sql.DB, videoDir string, log func(string, map[string]interface{})) *Handler {
	return &Handler{repo: repo, sqlRepo: repo, approvalsRepo: approvalsRepo, db: db, videoDir: videoDir, log: log}
}

// NewHandlerForTest creates a handler backed by any Repository implementation.
// Bridging is disabled (approvalsRepo and db are nil).
func NewHandlerForTest(repo Repository, videoDir string, log func(string, map[string]interface{})) *Handler {
	return &Handler{repo: repo, videoDir: videoDir, log: log}
}

// Create initiates a new visual validation.
// POST /api/v1/validations
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON: %v"}`, err), http.StatusBadRequest)
		return
	}

	if req.ProfileID == "" {
		http.Error(w, `{"error":"profile_id is required"}`, http.StatusBadRequest)
		return
	}
	if req.GitCommitHash == "" {
		http.Error(w, `{"error":"git_commit_hash is required"}`, http.StatusBadRequest)
		return
	}

	record := &Record{
		ID:            generateID(),
		ProfileID:     req.ProfileID,
		Status:        "pending",
		Platform:      req.Platform,
		GitCommitHash: req.GitCommitHash,
		CreatedAt:     time.Now(),
	}

	if err := h.repo.Create(r.Context(), record); err != nil {
		h.log("error", map[string]interface{}{"msg": "create validation failed", "error": err.Error()})
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	h.log("info", map[string]interface{}{"msg": "validation created", "id": record.ID, "profile_id": req.ProfileID})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(record)
}

// Get returns a validation record by ID.
// GET /api/v1/validations/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	record, err := h.repo.Get(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	if record == nil {
		http.Error(w, `{"error":"validation not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(record)
}

// StreamVideo serves the recorded video file with Range header support.
// GET /api/v1/validations/{id}/video
func (h *Handler) StreamVideo(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	record, err := h.repo.Get(r.Context(), id)
	if err != nil || record == nil {
		http.Error(w, `{"error":"validation not found"}`, http.StatusNotFound)
		return
	}

	if record.VideoURL == "" {
		http.Error(w, `{"error":"no video available"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", "inline")
	http.ServeFile(w, r, record.VideoURL)
}

// SubmitReview records an approve/reject decision and, when the validation
// has a git_commit_hash, atomically creates/updates a deployment approval.
// POST /api/v1/validations/{id}/review
func (h *Handler) SubmitReview(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var req ReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON: %v"}`, err), http.StatusBadRequest)
		return
	}

	if req.Decision != "approved" && req.Decision != "rejected" {
		http.Error(w, `{"error":"decision must be 'approved' or 'rejected'"}`, http.StatusBadRequest)
		return
	}

	// Default reviewer to "operator" when not provided.
	reviewer := req.Reviewer
	if reviewer == "" {
		reviewer = "operator"
	}

	// Look up the validation to get profile_id, platform, git_commit_hash.
	record, err := h.repo.Get(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	if record == nil {
		http.Error(w, `{"error":"validation not found"}`, http.StatusNotFound)
		return
	}

	resp := ReviewResponse{
		Status:   "ok",
		Decision: req.Decision,
	}

	// If the validation has a commit hash and bridging is configured,
	// bridge the review into a deployment approval atomically.
	if record.GitCommitHash != "" && h.approvalsRepo != nil && h.sqlRepo != nil {
		tx, err := h.db.BeginTx(r.Context(), nil)
		if err != nil {
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
		defer tx.Rollback() //nolint:errcheck

		// Update validation review within tx.
		if err := h.sqlRepo.WithTx(tx).UpdateReview(r.Context(), id, req.Decision, reviewer, req.Notes); err != nil {
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}

		// Create bridged deployment approval within same tx.
		approvalID := generateApprovalID()
		now := time.Now()
		approval := &deployments.DeploymentApproval{
			ID:            approvalID,
			ProfileID:     record.ProfileID,
			GitCommitHash: record.GitCommitHash,
			Platform:      record.Platform,
			Status:        req.Decision, // "approved" or "rejected"
			ApprovedBy:    reviewer,
			ApprovedAt:    &now,
			Notes:         req.Notes,
			ValidationID:  id,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := h.approvalsRepo.WithTx(tx).Create(r.Context(), approval); err != nil {
			h.log("error", map[string]interface{}{"msg": "bridged approval failed", "error": err.Error()})
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}

		resp.ApprovalID = approvalID
		resp.ApprovalStatus = req.Decision
	} else {
		// No bridging — just update the review directly.
		if err := h.repo.UpdateReview(r.Context(), id, req.Decision, reviewer, req.Notes); err != nil {
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
	}

	h.log("info", map[string]interface{}{"msg": "review submitted", "id": id, "decision": req.Decision, "approval_id": resp.ApprovalID})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// ListByProfile returns all validations for a profile.
// GET /api/v1/profiles/{id}/validations
func (h *Handler) ListByProfile(w http.ResponseWriter, r *http.Request) {
	profileID := mux.Vars(r)["id"]
	records, err := h.repo.ListByProfile(r.Context(), profileID)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	if records == nil {
		records = []*Record{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(records)
}

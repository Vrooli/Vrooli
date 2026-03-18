package validation

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Handler provides HTTP endpoints for visual validation management.
type Handler struct {
	repo     Repository
	videoDir string
	log      func(string, map[string]interface{})
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("val-%x", b[:8])
}

// NewHandler creates a new validation handler.
func NewHandler(repo Repository, videoDir string, log func(string, map[string]interface{})) *Handler {
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

	record := &Record{
		ID:        generateID(),
		ProfileID: req.ProfileID,
		Status:    "pending",
		Platform:  req.Platform,
		CreatedAt: time.Now(),
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

// SubmitReview records an approve/reject decision.
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

	if err := h.repo.UpdateReview(r.Context(), id, req.Decision, req.Reviewer, req.Notes); err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	h.log("info", map[string]interface{}{"msg": "review submitted", "id": id, "decision": req.Decision})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "decision": req.Decision})
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

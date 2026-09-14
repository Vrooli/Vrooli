package storagehealth

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	coredb "github.com/vrooli/api-core/database"
)

// Authorizer verifies the owner-maintenance credential and returns its subject,
// or the HTTP status that refuses the request.
type Authorizer func(*http.Request) (subject string, status int, err error)

// Handler serves the storage contract:
//
//	GET  /api/v1/storage/health   measurement, thresholds, latest compaction
//	POST /api/v1/storage/reclaim  {"dry_run":bool} online reclaim (storage-manager)
//	POST /api/v1/storage/compact  {"reason":"..."} fenced compaction (owner only)
type Handler struct {
	svc       *Service
	authorize Authorizer
}

func NewHandler(svc *Service, authorize Authorizer) (*Handler, error) {
	if svc == nil || authorize == nil {
		return nil, errors.New("storage handler requires the storage service and owner authorizer")
	}
	return &Handler{svc: svc, authorize: authorize}, nil
}

type healthView struct {
	Storage    Stats            `json:"storage"`
	Policy     Policy           `json:"policy"`
	Guidance   string           `json:"guidance,omitempty"`
	Compaction CompactionStatus `json:"compaction"`
}

// refuseTestMode keeps test traffic away from production: every operation
// here reads or rewrites the production file, never a routed test pool.
func refuseTestMode(w http.ResponseWriter, r *http.Request) bool {
	if coredb.IsTestMode(r.Context()) {
		writeError(w, http.StatusConflict, "storage maintenance operates on the production database and is unavailable to test-mode requests")
		return true
	}
	return false
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if refuseTestMode(w, r) {
		return
	}
	st, err := h.svc.Stats(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, healthView{Storage: st, Policy: h.svc.Policy(), Guidance: guidance(st.Action), Compaction: h.svc.CompactionStatus()})
}

func (h *Handler) Reclaim(w http.ResponseWriter, r *http.Request) {
	if refuseTestMode(w, r) {
		return
	}
	var input struct {
		DryRun bool `json:"dry_run"`
	}
	if !decodeOptional(w, r, &input) {
		return
	}
	receipt, err := h.svc.Reclaim(r.Context(), input.DryRun)
	switch {
	case errors.Is(err, ErrBusy):
		writeError(w, http.StatusConflict, err.Error())
	case err != nil:
		writeError(w, http.StatusServiceUnavailable, err.Error())
	default:
		writeJSON(w, http.StatusOK, receipt)
	}
}

func (h *Handler) Compact(w http.ResponseWriter, r *http.Request) {
	if refuseTestMode(w, r) {
		return
	}
	subject, status, err := h.authorize(r)
	if err != nil {
		writeError(w, status, err.Error())
		return
	}
	var input struct {
		Reason string `json:"reason"`
	}
	if !decodeOptional(w, r, &input) {
		return
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" || len(reason) > 512 {
		writeError(w, http.StatusBadRequest, "reason must contain 1-512 bytes")
		return
	}
	started, err := h.svc.StartCompaction(r.Context(), subject, reason)
	switch {
	case errors.Is(err, ErrFenceNotDrained), errors.Is(err, ErrBusy):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrInsufficientSpace):
		writeError(w, http.StatusInsufficientStorage, err.Error())
	case err != nil:
		writeError(w, http.StatusServiceUnavailable, err.Error())
	default:
		writeJSON(w, http.StatusAccepted, started)
	}
}

// decodeOptional accepts an empty body as the zero request and rejects
// unknown fields or trailing documents.
func decodeOptional(w http.ResponseWriter, r *http.Request, into any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(into); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid storage request")
		return false
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "one storage request is required")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// AssignRunWorkload writes an operator-declared attribution for an existing
// run. The run identity and lifecycle remain untouched.
func (h *Handler) AssignRunWorkload(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeSimpleError(w, r, "id", "valid run ID is required")
		return
	}
	var body struct {
		Kind     domain.WorkloadKind `json:"kind"`
		Key      string              `json:"key"`
		Instance string              `json:"instance,omitempty"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil || !body.Kind.IsValid() || strings.TrimSpace(body.Key) == "" {
		writeSimpleError(w, r, "workload", "kind must be a supported workload kind and key must be non-empty")
		return
	}
	service, ok := h.svc.RunService.(orchestration.RunWorkloadAssignmentService)
	if !ok {
		writeSimpleError(w, r, "service", "workload assignment is unavailable")
		return
	}
	run, err := service.AssignRunWorkload(r.Context(), id, domain.WorkloadRef{Kind: body.Kind, Key: body.Key, Instance: body.Instance})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

// BackfillImportedRunLabels is a narrow maintenance endpoint. The optional
// service assertion keeps label repair separate from normal run mutation.
func (h *Handler) BackfillImportedRunLabels(w http.ResponseWriter, r *http.Request) {
	service, ok := h.svc.RunService.(orchestration.RunLabelMaintenanceService)
	if !ok {
		writeSimpleError(w, r, "service", "run label backfill is unavailable")
		return
	}
	result, err := service.BackfillImportedRunLabels(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) BackfillRunSubjects(w http.ResponseWriter, r *http.Request) {
	service, ok := h.svc.RunService.(orchestration.RunSubjectMaintenanceService)
	if !ok {
		writeSimpleError(w, r, "service", "run subject backfill is unavailable")
		return
	}
	result, err := service.BackfillRunSubjects(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ImportTranscriptSweep runs the scheduled importer's sweep on demand. It is
// the same idempotent operation the scheduler runs, so an operator diagnosing a
// stale corpus never has to wait out the interval or reach for a second import
// path that could drift from the scheduled one.
func (h *Handler) ImportTranscriptSweep(w http.ResponseWriter, r *http.Request) {
	if h.transcriptImporter == nil {
		writeSimpleError(w, r, "service", "transcript import sweep is unavailable")
		return
	}
	summary, err := h.transcriptImporter.RunOnce(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

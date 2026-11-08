package validationmatrix

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"google.golang.org/protobuf/proto"
)

// Handler exposes the durable orchestration seam without making the control
// plane depend on a provider-specific workflow implementation.
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// Create delegates one typed matrix selection to the durable service. Scenario
// owners can normalize provider-specific selections before entering this shared
// orchestration seam.
func (h *Handler) Create(selection MatrixSelection) (*MatrixRun, error) {
	if h == nil || h.service == nil {
		return nil, errors.New("validation matrix service is unavailable")
	}
	return h.service.Create(selection)
}

func (h *Handler) RegisterRoutes(router *mux.Router) {
	if h == nil || h.service == nil {
		return
	}
	router.HandleFunc("/api/v1/validation/matrices", h.create).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/validation/matrices", h.list).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/validation/catalog", h.catalog).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/validation/profiles", h.profiles).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/validation/matrices/{run_id}", h.get).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/validation/matrices/{run_id}/compare/{prior_run_id}", h.compare).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/validation/matrices/{run_id}/start", h.start).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/validation/matrices/{run_id}/wait", h.wait).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/validation/matrices/{run_id}/abort", h.abort).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/validation/matrices/{run_id}/rerun", h.rerun).Methods(http.MethodPost)
}

func (h *Handler) profiles(w http.ResponseWriter, _ *http.Request) {
	type profileResponse struct {
		Value                int32    `json:"value"`
		Name                 string   `json:"name"`
		Label                string   `json:"label"`
		Category             string   `json:"category"`
		Executable           bool     `json:"executable"`
		RequiredCapabilities []string `json:"required_capabilities,omitempty"`
	}
	contracts := ProfileContracts()
	profiles := make([]profileResponse, 0, len(contracts))
	for _, contract := range contracts {
		capabilities := make([]string, 0, len(contract.RequiredCapabilities))
		for _, capability := range contract.RequiredCapabilities {
			capabilities = append(capabilities, capability.String())
		}
		profiles = append(profiles, profileResponse{
			Value:                int32(contract.Profile),
			Name:                 contract.Profile.String(),
			Label:                contract.Label,
			Category:             contract.Category,
			Executable:           contract.Executable,
			RequiredCapabilities: capabilities,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"profiles": profiles})
}

func (h *Handler) catalog(w http.ResponseWriter, r *http.Request) {
	scenario := strings.TrimSpace(r.URL.Query().Get("scenario"))
	if scenario == "" {
		writeError(w, http.StatusBadRequest, "scenario is required")
		return
	}
	catalog, err := h.service.Catalog(r.Context(), scenario)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, catalog)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	runs := h.service.List(r.URL.Query().Get("scenario"))
	views := make([]*MatrixRun, 0, len(runs))
	for _, run := range runs {
		views = append(views, PublicRun(run))
	}
	writeJSON(w, http.StatusOK, views)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var selection MatrixSelection
	if err := json.NewDecoder(r.Body).Decode(&selection); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	run, err := h.service.Create(selection)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, PublicRun(run))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	run, ok := h.service.store.Get(mux.Vars(r)["run_id"])
	if !ok {
		writeError(w, http.StatusNotFound, "matrix run not found")
		return
	}
	writeJSON(w, http.StatusOK, PublicRun(run))
}

func (h *Handler) compare(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	comparison, err := h.service.Compare(vars["run_id"], vars["prior_run_id"])
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, comparison)
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	run, err := h.service.Start(mux.Vars(r)["run_id"])
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, PublicRun(run))
}

func (h *Handler) wait(w http.ResponseWriter, r *http.Request) {
	run, err := h.service.Wait(r.Context(), mux.Vars(r)["run_id"])
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, PublicRun(run))
}

func (h *Handler) abort(w http.ResponseWriter, r *http.Request) {
	runID := mux.Vars(r)["run_id"]
	run, err := h.service.AbortAndWait(r.Context(), runID)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			if current, ok := h.service.store.Get(runID); ok {
				writeJSON(w, http.StatusAccepted, PublicRun(current))
				return
			}
		}
		status := http.StatusNotFound
		if !strings.Contains(err.Error(), "not found") {
			status = http.StatusInternalServerError
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, PublicRun(run))
}

func (h *Handler) rerun(w http.ResponseWriter, r *http.Request) {
	var selector RerunSelector
	if err := json.NewDecoder(r.Body).Decode(&selector); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	run, err := h.service.Rerun(mux.Vars(r)["run_id"], selector)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, PublicRun(run))
}

// PublicRun creates the path-free representation used by HTTP consumers.
func PublicRun(run *MatrixRun) *MatrixRun {
	if run == nil {
		return nil
	}
	view := *run
	view.Selection = cloneSelection(run.Selection)
	view.Selection.ArtifactPath = ""
	view.Selection.ArtifactPaths = nil
	view.Selection.Metadata = redactPathFields(view.Selection.Metadata)
	if run.Matrix != nil {
		view.Matrix = proto.Clone(run.Matrix).(*domainv1.ValidationMatrix)
		for _, cell := range view.Matrix.GetCells() {
			redactEvidenceURIs(cell)
		}
	}
	view.Cells = make([]*CellRecord, len(run.Cells))
	for i, cell := range run.Cells {
		if cell == nil {
			continue
		}
		publicCell := *cell
		if cell.Cell != nil {
			publicCell.Cell = proto.Clone(cell.Cell).(*domainv1.ValidationCell)
			redactEvidenceURIs(publicCell.Cell)
		}
		publicCell.Report = redactPathFields(cell.Report)
		view.Cells[i] = &publicCell
	}
	return &view
}

func redactEvidenceURIs(cell *domainv1.ValidationCell) {
	if cell == nil {
		return
	}
	for _, evidence := range cell.GetEvidence() {
		if evidence != nil && isLocalEvidenceURI(evidence.GetUri()) {
			evidence.Uri = "validation://local-evidence/" + evidence.GetEvidenceId()
		}
	}
}

func isLocalEvidenceURI(uri string) bool {
	uri = strings.TrimSpace(uri)
	if uri == "" || strings.HasPrefix(strings.ToLower(uri), "file:") || filepath.IsAbs(uri) || strings.HasPrefix(uri, `\\`) {
		return uri != ""
	}
	return len(uri) >= 3 && ((uri[0] >= 'a' && uri[0] <= 'z') || (uri[0] >= 'A' && uri[0] <= 'Z')) && uri[1] == ':' && (uri[2] == '/' || uri[2] == '\\')
}

func redactPathFields(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	redacted := make(map[string]string, len(values))
	for key, value := range values {
		if strings.Contains(strings.ToLower(key), "path") {
			continue
		}
		redacted[key] = value
	}
	return redacted
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

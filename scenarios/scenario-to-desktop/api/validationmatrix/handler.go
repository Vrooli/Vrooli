package validationmatrix

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// Handler exposes the durable orchestration seam without making the control
// plane depend on a provider-specific workflow implementation.
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(router *mux.Router) {
	if h == nil || h.service == nil {
		return
	}
	router.HandleFunc("/api/v1/validation/matrices", h.create).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/validation/matrices", h.list).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/validation/catalog", h.catalog).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/validation/matrices/{run_id}", h.get).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/validation/matrices/{run_id}/compare/{prior_run_id}", h.compare).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/validation/matrices/{run_id}/start", h.start).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/validation/matrices/{run_id}/wait", h.wait).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/validation/matrices/{run_id}/abort", h.abort).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/validation/matrices/{run_id}/rerun", h.rerun).Methods(http.MethodPost)
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
	writeJSON(w, http.StatusOK, h.service.List(r.URL.Query().Get("scenario")))
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
	writeJSON(w, http.StatusCreated, run)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	run, ok := h.service.store.Get(mux.Vars(r)["run_id"])
	if !ok {
		writeError(w, http.StatusNotFound, "matrix run not found")
		return
	}
	writeJSON(w, http.StatusOK, run)
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
	writeJSON(w, http.StatusAccepted, run)
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
	writeJSON(w, http.StatusOK, run)
}

func (h *Handler) abort(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Abort(mux.Vars(r)["run_id"]); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	run, _ := h.service.store.Get(mux.Vars(r)["run_id"])
	writeJSON(w, http.StatusOK, run)
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
	writeJSON(w, http.StatusCreated, run)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

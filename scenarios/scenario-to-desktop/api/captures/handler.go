package captures

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"

	httputil "scenario-to-desktop-api/shared/http"
)

// Handler holds HTTP handlers for the captures domain.
type Handler struct {
	service *Service
}

// NewHandler creates a new captures handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all captures API routes.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/captures/{scenario}", h.listCaptures).Methods("GET")
	r.HandleFunc("/api/v1/captures/{scenario}/summary", h.summary).Methods("GET")
	r.HandleFunc("/api/v1/captures/{scenario}/{id}/file", h.serveFile).Methods("GET")
	r.HandleFunc("/api/v1/captures/{scenario}/{id}", h.deleteCapture).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/api/v1/captures/{scenario}", h.deleteAll).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/api/v1/captures/{scenario}/download", h.download).Methods("GET")
}

func (h *Handler) listCaptures(w http.ResponseWriter, r *http.Request) {
	scenario := mux.Vars(r)["scenario"]
	caps, err := h.service.Store().List(scenario)
	if err != nil {
		httputil.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	httputil.WriteJSON(w, http.StatusOK, caps)
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	scenario := mux.Vars(r)["scenario"]
	s, err := h.service.Store().Summary(scenario)
	if err != nil {
		httputil.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	httputil.WriteJSON(w, http.StatusOK, s)
}

func (h *Handler) serveFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenario := vars["scenario"]
	captureID := vars["id"]

	path, err := h.service.CaptureFilePath(scenario, captureID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			httputil.WriteJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		httputil.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	http.ServeFile(w, r, path)
}

func (h *Handler) deleteCapture(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenario := vars["scenario"]
	captureID := vars["id"]

	if err := h.service.DeleteCapture(scenario, captureID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			httputil.WriteJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		httputil.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) deleteAll(w http.ResponseWriter, r *http.Request) {
	scenario := mux.Vars(r)["scenario"]
	if err := h.service.CleanAll(scenario); err != nil {
		httputil.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
	scenario := mux.Vars(r)["scenario"]
	idsParam := r.URL.Query().Get("ids")
	if idsParam == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "ids query parameter is required"})
		return
	}

	ids := strings.Split(idsParam, ",")

	// Single file: serve directly
	if len(ids) == 1 {
		path, err := h.service.CaptureFilePath(scenario, ids[0])
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				httputil.WriteJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			httputil.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		http.ServeFile(w, r, path)
		return
	}

	// Multiple files: stream as zip
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="captures-%s.zip"`, scenario))

	zw := zip.NewWriter(w)
	defer zw.Close()

	for _, id := range ids {
		path, err := h.service.CaptureFilePath(scenario, id)
		if err != nil {
			continue // skip missing files
		}
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		info, err := f.Stat()
		if err != nil {
			f.Close()
			continue
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			f.Close()
			continue
		}
		header.Method = zip.Deflate
		writer, err := zw.CreateHeader(header)
		if err != nil {
			f.Close()
			continue
		}
		_, _ = io.Copy(writer, f)
		f.Close()
	}
}

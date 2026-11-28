package control

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func (h *handler) audit(w http.ResponseWriter, r *http.Request) {
	write(w, http.StatusOK, map[string]any{"records": h.service.AuditContext(r.Context())})
}

func (h *handler) artifact(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	data, kind, err := h.service.ArtifactContext(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "artifact_not_found", err.Error())
		return
	}
	contentType := http.DetectContentType(data)
	switch kind {
	case "image":
		if strings.HasPrefix(contentType, "application/octet-stream") {
			contentType = "image/png"
		}
	case "video":
		contentType = "video/mp4"
	case "log":
		contentType = "text/plain; charset=utf-8"
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *handler) listAnchors(w http.ResponseWriter, _ *http.Request) {
	write(w, http.StatusOK, map[string]any{"anchors": h.anchors.List()})
}

func (h *handler) createAnchor(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name       string    `json:"name"`
		Target     string    `json:"target"`
		Bounds     []float64 `json:"bounds"`
		Confidence float64   `json:"confidence"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	a, err := h.anchors.Create(in.Name, in.Target, in.Bounds, in.Confidence)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_anchor", err.Error())
		return
	}
	write(w, http.StatusCreated, map[string]any{"anchor": a})
}

func (h *handler) deleteAnchor(w http.ResponseWriter, r *http.Request) {
	if err := h.anchors.Delete(mux.Vars(r)["id"]); err != nil {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

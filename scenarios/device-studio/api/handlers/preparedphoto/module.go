package preparedphoto

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"device-studio/internal/module"
	"device-studio/internal/photographic"
	"github.com/gorilla/mux"
)

type Source interface {
	PreparedPhotograph(context.Context, string) ([]byte, error)
}

type Handler struct {
	source  Source
	library photographic.Publisher
	scenes  photographic.SceneSaver
}

type request struct {
	PreparationID string `json:"preparation_id"`
	LicenseID     string `json:"license_id"`
	Attribution   string `json:"attribution"`
	SourceLabel   string `json:"source_label"`
}

type response struct {
	SceneID            string  `json:"scene_id"`
	SceneRevision      uint32  `json:"scene_revision"`
	SceneSHA256        string  `json:"scene_sha256"`
	LibraryBundleID    string  `json:"library_bundle_id"`
	LibraryRevision    uint32  `json:"library_revision"`
	LibraryManifestSHA string  `json:"library_manifest_sha256"`
	ScaleStatus        string  `json:"scale_status"`
	ScaleEstimateM     float64 `json:"estimated_scale_m"`
}

var Endpoints = []module.EndpointDescriptor{{
	ID: "prepared_photo_to_scene_candidate", Path: "/api/v1/photographic-scenes/from-preparation", Method: http.MethodPost,
	Summary:     "Convert an accepted immutable photograph preparation into an edit-only scene candidate",
	Description: "Fetches a content-addressed Image Tools preparation by ID, validates its exact retained review/geometry/source bytes, publishes geometry and clean-plate assets through Library, then saves a SceneCandidate. It never grants render rights or claims measured scale.", Category: "scenes",
	Request:       &module.Schema{Type: "object", Properties: map[string]string{"preparation_id": "64-character content SHA-256", "license_id": "explicit caller claim", "attribution": "explicit caller claim", "source_label": "explicit caller claim"}},
	Response:      &module.Schema{Type: "object", Properties: map[string]string{"scene_id": "string", "scene_revision": "integer", "scene_sha256": "SHA-256", "library_bundle_id": "string", "scale_status": "estimated_monocular_scale_not_measured"}},
	RESTException: &module.RESTException{Reason: module.RESTReasonBrowserSurface, Note: "The workbench submits an immutable preparation ID and explicit source-rights claims; the server fetches exact retained bytes from Image Tools and publishes through canonical Library and SceneCandidate owners."},
}, {
	ID: "pending_preparation_geometry_qa", Path: "/api/v1/photographic-scenes/preparations/{preparation_id}/geometry-qa", Method: http.MethodGet,
	Summary:     "Inspect geometry for a pending EMPTY_SCENE preparation",
	Description: "Runs strict outer bundle, inner ZIP/NPY and camera-projection checks and returns a diagnostic-only ZIP with source/ROI/depth/plane overlays. It cannot accept or publish a preparation.", Category: "scenes",
	Response:      &module.Schema{Type: "application/zip", Properties: map[string]string{"geometry-qa.json": "pending-only numerical validation report", "geometry-qa.png": "source ROI/inlier/reject, depth and plane-residual panels"}},
	RESTException: &module.RESTException{Reason: module.RESTReasonBrowserSurface, Note: "This bounded inspection download is read-only and requires the content-addressed pending preparation ID."},
}}

func Module(source Source, libraryOwner photographic.Publisher, sceneOwner photographic.SceneSaver) module.Module {
	h := &Handler{source: source, library: libraryOwner, scenes: sceneOwner}
	return module.Module{Name: "preparedphoto", Endpoints: Endpoints, Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/photographic-scenes/from-preparation", h.convert).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/photographic-scenes/preparations/{preparation_id}/geometry-qa", h.geometryQA).Methods(http.MethodGet)
	}}
}

func (h *Handler) geometryQA(w http.ResponseWriter, r *http.Request) {
	if h.source == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "photographic preparation inspection is unavailable"})
		return
	}
	id := mux.Vars(r)["preparation_id"]
	if len(id) != 64 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "preparation_id must be a content SHA-256"})
		return
	}
	raw, err := h.source.PreparedPhotograph(r.Context(), id)
	if err != nil {
		var ownerErr photographic.OwnerHTTPError
		status := http.StatusBadGateway
		if errors.As(err, &ownerErr) && ownerErr.Status == http.StatusNotFound {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": "Image Tools preparation could not be verified"})
		return
	}
	qa, preview, err := photographic.InspectPending(raw)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": fmt.Sprintf("pending preparation geometry QA failed: %v", err)})
		return
	}
	archive, err := photographic.GeometryQAZIP(qa, preview)
	if err != nil || len(archive) > 32<<20 {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "geometry QA archive could not be produced"})
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Length", fmt.Sprint(len(archive)))
	w.Header().Set("X-Geometry-QA-Inspection-Only", "true")
	w.Header().Set("X-Geometry-QA-Accepted", "false")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(archive)
}

func (h *Handler) convert(w http.ResponseWriter, r *http.Request) {
	if h.source == nil || h.library == nil || h.scenes == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "photographic preparation conversion is unavailable"})
		return
	}
	var in request
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&in); err != nil || strings.TrimSpace(in.PreparationID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid preparation conversion request"})
		return
	}
	if decoder.Decode(new(any)) != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request must contain one JSON value"})
		return
	}
	raw, err := h.source.PreparedPhotograph(r.Context(), in.PreparationID)
	if err != nil {
		var ownerErr photographic.OwnerHTTPError
		status := http.StatusBadGateway
		if errors.As(err, &ownerErr) && ownerErr.Status == http.StatusNotFound {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": "Image Tools preparation could not be verified"})
		return
	}
	result, err := photographic.PublishCandidate(r.Context(), raw, photographic.PublishClaims{LicenseID: in.LicenseID, Attribution: in.Attribution, SourceLabel: in.SourceLabel}, h.library, h.scenes)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, response{SceneID: result.Scene.SceneID, SceneRevision: result.Scene.Revision, SceneSHA256: result.Scene.SHA256,
		LibraryBundleID: result.Library.Manifest.BundleID, LibraryRevision: result.Library.Manifest.Revision, LibraryManifestSHA: result.Library.ManifestSHA256,
		ScaleStatus: result.Conversion.ScaleStatus, ScaleEstimateM: result.Conversion.EstimatedScaleMeters})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

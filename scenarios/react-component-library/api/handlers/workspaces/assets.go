// Asset transport for the workspace studio.
//
// Composition travels over Connect-RPC, but reference images do not: the
// browser renders them with <img>/CSS and uploads a data URL. Those routes are
// REST for one reason (binary bytes) while still answering failures in the
// canonical JSON error envelope. Metadata is small JSON on the same routes so a
// client never needs a second transport for one concept.
package workspaces

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"react-component-library/internal/httprest"
	"react-component-library/internal/module"
	domain "react-component-library/internal/workspaces"
)

// maxAssetRequestBytes bounds one JSON upload: the base64 envelope is ~4/3 of
// the decoded bytes, plus the small metadata fields.
const maxAssetRequestBytes = (domain.MaxAssetBytes/3+1)*4 + (1 << 20)

type assetRequest struct {
	Name       string `json:"name"`
	Mime       string `json:"mime"`
	DataBase64 string `json:"dataBase64"`
	Fidelity   string `json:"fidelity"`
}

// mountAssets registers the workspace asset routes on the module router.
func mountAssets(router *mux.Router, handler *Handler) {
	router.HandleFunc("/api/workspaces/{id}/assets", handler.listAssets).Methods(http.MethodGet)
	router.HandleFunc("/api/workspaces/{id}/assets", handler.createAsset).Methods(http.MethodPost)
	router.HandleFunc("/api/workspaces/{id}/assets/{assetId}", handler.getAssetData).Methods(http.MethodGet)
	router.HandleFunc("/api/workspaces/{id}/assets/{assetId}", handler.removeAsset).Methods(http.MethodDelete)
	router.HandleFunc("/api/workspace-assets", handler.listAllAssets).Methods(http.MethodGet)
}

func (h *Handler) listAllAssets(w http.ResponseWriter, r *http.Request) {
	assets, err := h.repository.ListAllWorkspaceAssets(r.Context())
	if err != nil {
		h.assetRepositoryError(w, "list all workspace assets", err)
		return
	}
	httprest.WriteJSON(w, http.StatusOK, map[string]any{"assets": assets})
}

func (h *Handler) listAssets(w http.ResponseWriter, r *http.Request) {
	assets, err := h.repository.ListWorkspaceAssets(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		h.assetRepositoryError(w, "list workspace assets", err)
		return
	}
	httprest.WriteJSON(w, http.StatusOK, map[string]any{"assets": assets})
}

func (h *Handler) createAsset(w http.ResponseWriter, r *http.Request) {
	var body assetRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxAssetRequestBytes))
	if err := decoder.Decode(&body); err != nil {
		httprest.WriteError(w, http.StatusBadRequest, "invalid_argument", "request body must be a JSON asset upload")
		return
	}
	data, err := httprest.DecodeBase64Data(body.DataBase64)
	if err != nil {
		httprest.WriteError(w, http.StatusBadRequest, "invalid_argument", err.Error())
		return
	}
	asset, err := h.repository.CreateWorkspaceAsset(r.Context(), domain.AssetInput{
		WorkspaceID: mux.Vars(r)["id"],
		Name:        body.Name,
		Mime:        body.Mime,
		Data:        data,
		Fidelity:    body.Fidelity,
	})
	if err != nil {
		h.assetRepositoryError(w, "create workspace asset", err)
		return
	}
	httprest.WriteJSON(w, http.StatusCreated, map[string]any{"asset": asset})
}

func (h *Handler) getAssetData(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	asset, data, err := h.repository.GetWorkspaceAsset(r.Context(), vars["id"], vars["assetId"])
	if err != nil {
		h.assetRepositoryError(w, "get workspace asset", err)
		return
	}
	// The id is stable and the bytes for an id never change, so it is also the
	// entity tag. A 304 keeps a large reference image out of a reload.
	etag := asset.ID
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	if httprest.EtagMatches(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", asset.Mime)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *Handler) removeAsset(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := h.repository.DeleteWorkspaceAsset(r.Context(), vars["id"], vars["assetId"]); err != nil {
		h.assetRepositoryError(w, "delete workspace asset", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// assetRepositoryError maps the two asset failures onto the same envelope the
// Connect routes produce.
func (h *Handler) assetRepositoryError(w http.ResponseWriter, operation string, err error) {
	var invalid *domain.InvalidError
	switch {
	case errors.As(err, &invalid):
		httprest.WriteError(w, http.StatusBadRequest, "invalid_argument", strings.TrimPrefix(err.Error(), domain.ErrInvalid.Error()+": "))
	case errors.Is(err, domain.ErrCurated):
		httprest.WriteError(w, http.StatusConflict, "failed_precondition",
			"curated templates are immutable; fork one to add references")
	case errors.Is(err, domain.ErrNotFound):
		httprest.WriteError(w, http.StatusNotFound, "not_found", "workspace or asset not found")
	default:
		h.logger.Printf("workspaces: %s: %v", operation, err)
		httprest.WriteError(w, http.StatusInternalServerError, "internal", "workspace asset storage failure")
	}
}

func assetREST(request module.RESTPayload) *module.RESTException {
	return &module.RESTException{
		Reason: module.RESTReasonBrowserSurface,
		Note:   "Reference images are rendered directly by the browser and uploaded as a data URL. The list/create metadata travels with the same routes so one client owns uploads; failures still use the canonical JSON error envelope.",
		ProtoPayloads: &module.RESTProtoPayloads{
			Request:  request,
			Response: module.RESTPayload{Transport: "json", Conformance: "external_shape"},
			Error:    module.RESTPayload{Transport: "json", Conformance: "external_shape"},
		},
	}
}

var assetEndpoints = []module.EndpointDescriptor{
	{
		ID:            "workspaces_assets_list",
		Path:          "/api/workspaces/{id}/assets",
		Method:        http.MethodGet,
		Summary:       "List a workspace's reference assets",
		Description:   "Returns upload metadata newest-first. Bytes are served by the per-asset route. A missing workspace is 404.",
		Category:      "workspaces",
		Response:      &module.Schema{Type: "object", Properties: map[string]string{"assets": "array<WorkspaceAsset>"}},
		Errors:        []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No workspace with that id"}},
		RESTException: assetREST(module.RESTPayload{Transport: "none", Conformance: "none"}),
	},
	{
		ID:            "workspaces_assets_create",
		Path:          "/api/workspaces/{id}/assets",
		Method:        http.MethodPost,
		Summary:       "Upload a reference asset",
		Description:   "Stores one image as a workspace reference. name is required; mime must be png, jpeg, webp, gif or svg+xml; decoded bytes are capped at 8 MiB; fidelity is reference (default) or captured. Dimensions are read from the header when the type carries them.",
		Category:      "workspaces",
		Request:       &module.Schema{Type: "object", Properties: map[string]string{"name": "string (required)", "mime": "string (required)", "dataBase64": "string (required; bare base64 or data URL)", "fidelity": "\"reference\"|\"captured\" (optional)"}},
		Response:      &module.Schema{Type: "object", Properties: map[string]string{"asset": "WorkspaceAsset"}},
		Errors:        []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Missing or invalid name/mime/dataBase64/fidelity, or bytes over 8 MiB"}, {Status: 404, Code: "not_found", Description: "No workspace with that id"}},
		RESTException: assetREST(module.RESTPayload{Transport: "json", Conformance: "external_shape"}),
	},
	{
		ID:            "workspaces_assets_list_all",
		Path:          "/api/workspace-assets",
		Method:        http.MethodGet,
		Summary:       "List uploads across every workspace",
		Description:   "Returns metadata for uploads in any workspace, newest-first and bounded, so a reference already uploaded elsewhere can be found. Bytes remain scoped to the owning workspace.",
		Category:      "workspaces",
		Response:      &module.Schema{Type: "object", Properties: map[string]string{"assets": "array<WorkspaceAsset>"}},
		RESTException: assetREST(module.RESTPayload{Transport: "json", Conformance: "external_shape"}),
	},
	{
		ID:          "workspaces_assets_get",
		Path:        "/api/workspaces/{id}/assets/{assetId}",
		Method:      http.MethodGet,
		Summary:     "Read a reference asset's bytes",
		Description: "Serves the stored image with its Content-Type, Cache-Control: private, max-age=3600 and an ETag equal to the asset id. If-None-Match returns 304.",
		Category:    "workspaces",
		Response:    &module.Schema{Type: "binary", Properties: map[string]string{"body": "image bytes"}},
		Errors:      []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No asset with that id for that workspace"}},
		RESTException: &module.RESTException{
			Reason: module.RESTReasonBrowserSurface,
			Note:   "The byte route is browser-native: it sets Content-Type, Cache-Control and an ETag and answers If-None-Match with 304. Wrapping the bytes in Connect JSON would force base64 on every image render.",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
				Response: module.RESTPayload{Transport: "binary", Conformance: "external_shape"},
				Error:    module.RESTPayload{Transport: "json", Conformance: "external_shape"},
			},
		},
	},
	{
		ID:            "workspaces_assets_delete",
		Path:          "/api/workspaces/{id}/assets/{assetId}",
		Method:        http.MethodDelete,
		Summary:       "Delete a reference asset",
		Description:   "Deletes the asset scoped to that workspace. Idempotent for an existing workspace; an asset belonging to another workspace is untouched.",
		Category:      "workspaces",
		Response:      &module.Schema{Type: "object", Properties: map[string]string{"status": "204 No Content"}},
		Errors:        []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No workspace with that id"}},
		RESTException: assetREST(module.RESTPayload{Transport: "json", Conformance: "external_shape"}),
	},
}

// Endpoints is the full module surface: Connect RPCs first, then the REST asset
// routes. internal/modules reads this slice, so both transports are described
// in one place.
var Endpoints = append(append([]module.EndpointDescriptor{}, connectEndpoints...), assetEndpoints...)

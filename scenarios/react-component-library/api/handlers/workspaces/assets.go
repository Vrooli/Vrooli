// Asset byte transport for the workspace studio. Metadata and mutations travel
// over WorkspaceService; this route remains REST so browser image rendering can
// use the stored bytes directly with cache validation.
package workspaces

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"react-component-library/internal/httprest"
	"react-component-library/internal/module"
	domain "react-component-library/internal/workspaces"
)

func mountAssets(router *mux.Router, handler *Handler) {
	router.HandleFunc("/api/workspaces/{id}/assets/{assetId}", handler.getAssetData).Methods(http.MethodGet)
}

func (h *Handler) getAssetData(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	asset, data, err := h.repository.GetWorkspaceAsset(r.Context(), vars["id"], vars["assetId"])
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "workspace or asset not found", http.StatusNotFound)
		} else {
			h.logger.Printf("workspaces: get workspace asset: %v", err)
			http.Error(w, "workspace asset storage failure", http.StatusInternalServerError)
		}
		return
	}
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

var assetEndpoints = []module.EndpointDescriptor{{
	ID: "workspaces_assets_get", Path: "/api/workspaces/{id}/assets/{assetId}", Method: http.MethodGet,
	Summary:     "Read a reference asset's bytes",
	Description: "Serves the stored image with its Content-Type, private cache headers and an ETag equal to the asset id. If-None-Match returns 304.",
	Category:    "workspaces", Response: &module.Schema{Type: "binary", Properties: map[string]string{"body": "image bytes"}},
	Errors:        []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No asset with that id for that workspace"}},
	RESTException: &module.RESTException{Reason: module.RESTReasonBrowserSurface, Note: "The browser renders image bytes directly with Content-Type, Cache-Control and ETag support; Connect JSON would base64 encode each image render.", ProtoPayloads: &module.RESTProtoPayloads{Request: module.RESTPayload{Transport: "none", Conformance: "none"}, Response: module.RESTPayload{Transport: "binary", Conformance: "external_shape"}, Error: module.RESTPayload{Transport: "text", Conformance: "external_shape"}}},
}}

var Endpoints = append(
	append(
		append(
			append([]module.EndpointDescriptor{}, connectEndpoints...),
			visualConnectEndpoints...,
		),
		assetEndpoints...,
	),
	append(captureEndpoints, comparisonEndpoints...)...,
)

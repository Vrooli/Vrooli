package main

import (
	"context"
	"errors"
	"net/http"
	"strings"

	internalassets "brand-manager/internal/assets"
	internalbrands "brand-manager/internal/brands"
	"brand-manager/internal/candidates"
)

// logoAssetsHandler serves the Logo page's read-only binary endpoints. It
// returns the stored bytes verbatim (an SVG candidate stays an SVG, so the
// browser renders it crisply at any size), which is why the thumbnail is one
// HTTP route rather than a rasterizing RPC: the bytes are already the mark.
type logoAssetsHandler struct {
	candidates *candidates.Service
	assets     internalassets.Service
	brands     internalbrands.Service
}

// registerLogoAssetRoutes mounts the two endpoints on the root mux. They are
// exact paths, so the catch-all Connect handler still owns everything else.
func registerLogoAssetRoutes(mux *http.ServeMux, h logoAssetsHandler) {
	mux.HandleFunc("/api/v1/brand-manager/candidates/thumbnail", h.thumbnail)
	mux.HandleFunc("/api/v1/brand-manager/brands/mark", h.mark)
}

// thumbnail returns the bytes of a candidate's asset. Query: id=<candidate-id>.
func (h logoAssetsHandler) thumbnail(w http.ResponseWriter, r *http.Request) {
	candidateID := strings.TrimSpace(r.URL.Query().Get("id"))
	if candidateID == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	candidate, err := h.candidates.Get(r.Context(), candidateID)
	if err != nil {
		status := http.StatusInternalServerError
		var notFound candidates.ErrNotFound
		if errors.As(err, &notFound) {
			status = http.StatusNotFound
		}
		http.Error(w, "candidate not found", status)
		return
	}
	h.writeAsset(w, r.Context(), candidate.AssetID)
}

// mark returns the bytes of a brand's picked mark. Query: brand_id=<brand-id>.
func (h logoAssetsHandler) mark(w http.ResponseWriter, r *http.Request) {
	brandID := strings.TrimSpace(r.URL.Query().Get("brand_id"))
	if brandID == "" {
		http.Error(w, "brand_id is required", http.StatusBadRequest)
		return
	}
	brand, err := h.brands.Get(r.Context(), brandID)
	if err != nil {
		http.Error(w, "brand not found", http.StatusNotFound)
		return
	}
	if strings.TrimSpace(brand.MarkAssetID) == "" {
		http.Error(w, "brand has no picked mark", http.StatusNotFound)
		return
	}
	h.writeAsset(w, r.Context(), brand.MarkAssetID)
}

func (h logoAssetsHandler) writeAsset(w http.ResponseWriter, ctx context.Context, assetID string) {
	if strings.TrimSpace(assetID) == "" {
		http.Error(w, "asset not found", http.StatusNotFound)
		return
	}
	content, err := h.assets.Download(ctx, assetID)
	if err != nil {
		http.Error(w, "asset not found", http.StatusNotFound)
		return
	}
	mime := strings.TrimSpace(content.MimeType)
	if mime == "" {
		mime = "application/octet-stream"
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(content.Bytes)
}

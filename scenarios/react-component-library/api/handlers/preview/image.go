package preview

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"

	"react-component-library/internal/components"
)

// previewImagePath resolves a version preview through the manifest's own
// category. The library is organized by category (`components`, `primitives`,
// `foundations`, …); assuming every renderable asset lived in `components/`
// returned a 404 for every captured primitive preview.
func previewImagePath(libraryRoot string, component components.Component, version string) string {
	assetDir := filepath.Join(component.Slug)
	if manifest := strings.TrimSpace(component.ManifestPath); manifest != "" {
		if relative := filepath.Dir(filepath.FromSlash(manifest)); relative != "." {
			assetDir = relative
		}
	}
	return filepath.Join(libraryRoot, assetDir, "versions", version, "preview.png")
}

// ServePreviewImage exposes the durable version preview through the same
// library identity resolver as the harness. The browse UI can therefore show
// what was captured without reading the repository directly.
func (h *HarnessHandler) ServePreviewImage(w http.ResponseWriter, r *http.Request) {
	rawID := strings.TrimSpace(mux.Vars(r)["id"])
	if rawID == "" || h.components == nil {
		http.Error(w, "missing component id", http.StatusBadRequest)
		return
	}
	id, err := h.resolveComponentID(r, rawID)
	if err != nil {
		writeHarnessError(w, h.logger, rawID, err)
		return
	}
	component, err := h.components.Get(r.Context(), id)
	if err != nil {
		writeHarnessError(w, h.logger, rawID, err)
		return
	}
	version := strings.TrimSpace(r.URL.Query().Get("version"))
	if version == "" {
		version = component.LatestVersion
	}
	if version == "" || filepath.Base(version) != version || strings.Contains(version, "-") {
		http.Error(w, "invalid preview version", http.StatusBadRequest)
		return
	}
	root := filepath.Join(h.repoRoot, "scenarios", "react-component-library", "library")
	path := previewImagePath(root, component, version)
	if _, err := filepath.Rel(root, path); err != nil || !strings.HasPrefix(filepath.Clean(path), filepath.Clean(root)+string(filepath.Separator)) {
		http.Error(w, "preview path escapes library root", http.StatusBadRequest)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=60")
	http.ServeFile(w, r, path)
}

package api

import (
	"context"
	"encoding/json"
	"net/http"

	component "github.com/vrooli/vrooli/packages/desktop-component"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/components"
)

// FeatureResolution is returned only after the selected provider is usable.
type FeatureResolution struct {
	URL             string `json:"url"`
	Source          string `json:"source"`
	ComponentID     string `json:"component_id,omitempty"`
	Owner           string `json:"owner"`
	OwnerVersion    string `json:"owner_version"`
	Feature         string `json:"feature"`
	RuntimeInstance string `json:"runtime_instance"`
}

func (s *Server) handleFeature(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	scenario, feature := r.URL.Query().Get("scenario"), r.URL.Query().Get("feature")
	if !component.ValidID(scenario) || !component.ValidID(feature) {
		http.Error(w, "invalid feature", http.StatusBadRequest)
		return
	}
	source, ok := s.runtime.(interface {
		ResolveFeature(context.Context, string, string, string) (FeatureResolution, error)
	})
	if !ok {
		http.Error(w, "features unavailable", http.StatusNotImplemented)
		return
	}
	result, err := source.ResolveFeature(r.Context(), scenario, feature, r.URL.Query().Get("version_range"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

type ComponentSource interface {
	ComponentCatalog(context.Context) ([]component.SignedEntry, error)
	ComponentInstallations(context.Context) ([]components.Installation, error)
	InstallComponent(context.Context, string) (components.Installation, error)
	RemoveComponent(context.Context, string, bool) error
}

func (s *Server) handleComponents(w http.ResponseWriter, r *http.Request) {
	source, ok := s.runtime.(ComponentSource)
	if !ok {
		http.Error(w, "components unavailable", http.StatusNotImplemented)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		catalog, catalogErr := source.ComponentCatalog(r.Context())
		installed, err := source.ComponentInstallations(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		result := map[string]any{"available": catalog, "installed": installed}
		if catalogErr != nil {
			result["available"] = []component.SignedEntry{}
			result["catalog_error"] = "The dependency catalog is unavailable. Installed dependencies can still be removed."
		}
		_ = json.NewEncoder(w).Encode(result)
	case http.MethodPost, http.MethodDelete:
		var input struct {
			ID       string `json:"id"`
			KeepData *bool  `json:"keep_data"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&input) != nil || !component.ValidID(input.ID) {
			http.Error(w, "invalid component ID", http.StatusBadRequest)
			return
		}
		if r.Method == http.MethodDelete {
			if input.KeepData == nil {
				http.Error(w, "choose whether to keep component data", http.StatusBadRequest)
				return
			}
			if err := source.RemoveComponent(r.Context(), input.ID, *input.KeepData); err != nil {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"state": "removed"})
			return
		}
		row, err := source.InstallComponent(r.Context(), input.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		_ = json.NewEncoder(w).Encode(row)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"runtime"
	"sort"

	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
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

// OptionalDependencyRelease joins one app's shared manifest declaration to
// the signed component release and local installation state that can satisfy
// it. Resource dependencies stay on their resource-owned lifecycle path.
type OptionalDependencyRelease struct {
	DependencyID       string   `json:"dependency_id"`
	DependencySlug     string   `json:"dependency_slug"`
	DependencyKind     string   `json:"dependency_kind"`
	Enabled            bool     `json:"enabled"`
	MatchedFeatures    []string `json:"matched_features,omitempty"`
	ComponentID        string   `json:"component_id,omitempty"`
	Owner              string   `json:"owner,omitempty"`
	AvailableVersion   string   `json:"available_version,omitempty"`
	InstalledVersion   string   `json:"installed_version,omitempty"`
	State              string   `json:"state"`
	CanInstallOrUpdate bool     `json:"can_install_or_update"`
	CanRemove          bool     `json:"can_remove"`
	Reason             string   `json:"reason,omitempty"`
}

type optionalDependencyDeclarationSource interface {
	OptionalDependencyDeclarations(context.Context) ([]capabilityregistry.Def, error)
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
		manifest := s.runtime.Manifest()
		if catalogErr == nil && len(catalog) == 0 && (manifest == nil || manifest.Components == nil || manifest.Components.CatalogURL == "") {
			catalogErr = errors.New("signed component catalog source is not configured")
		}
		result := map[string]any{"available": catalog, "installed": installed}
		if manifest != nil && manifest.App.Scenario != "" {
			// This matches OptionalDependencyScenario.Name from the generated all-app catalog.
			result["app_scenario"] = manifest.App.Scenario
		}
		if catalogErr != nil {
			result["available"] = []component.SignedEntry{}
			result["catalog_error"] = "The dependency catalog is unavailable. Installed dependencies can still be removed."
		}
		if declarationSource, ok := s.runtime.(optionalDependencyDeclarationSource); ok {
			declarations, declarationErr := declarationSource.OptionalDependencyDeclarations(r.Context())
			if declarationErr != nil {
				result["dependency_catalog_error"] = "The app's service manifest is unavailable; dependency declarations could not be projected."
				result["dependencies"] = []OptionalDependencyRelease{}
			} else {
				publicKey := ""
				if manifest != nil && manifest.Components != nil {
					publicKey = manifest.Components.PublicKeyPEM
				}
				hostVersion := ""
				if manifest != nil {
					hostVersion = manifest.App.Version
				}
				result["dependencies"] = projectOptionalDependencyReleases(declarations, catalog, installed, catalogErr, publicKey, hostVersion)
			}
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

func projectOptionalDependencyReleases(declarations []capabilityregistry.Def, available []component.SignedEntry, installed []components.Installation, catalogErr error, publicKey, hostVersion string) []OptionalDependencyRelease {
	availableByID := make(map[string]component.SignedEntry, len(available))
	for _, release := range available {
		availableByID[release.Entry.ID] = release
	}
	installedByID := make(map[string]components.Installation, len(installed))
	for _, row := range installed {
		if row.Signed.Entry.ID != "" && row.Signed.Verify(publicKey) == nil {
			installedByID[row.Signed.Entry.ID] = row
		}
	}

	ids := make(map[string]bool, len(availableByID)+len(installedByID))
	for id := range availableByID {
		ids[id] = true
	}
	for id := range installedByID {
		ids[id] = true
	}
	componentIDs := make([]string, 0, len(ids))
	for id := range ids {
		componentIDs = append(componentIDs, id)
	}
	sort.Strings(componentIDs)

	result := make([]OptionalDependencyRelease, 0, len(declarations))
	for _, definition := range declarations {
		if definition.Required {
			continue
		}
		if !definition.Enabled {
			result = append(result, OptionalDependencyRelease{
				DependencyID: definition.ID, DependencySlug: definition.DependencySlug,
				DependencyKind: string(definition.DependencyKind), Enabled: false, State: "disabled",
				Reason: "This optional dependency is disabled in the staged service manifest; no install, update, or removal action is offered.",
			})
			continue
		}
		if definition.DependencyKind == capabilityregistry.DependencyResource {
			result = append(result, OptionalDependencyRelease{
				DependencyID: definition.ID, DependencySlug: definition.DependencySlug,
				DependencyKind: string(definition.DependencyKind), Enabled: true, State: "resource_managed",
				Reason: "Resource dependencies use the resource lifecycle and shared broker, not the signed scenario-component release catalog.",
			})
			continue
		}
		if definition.DependencyKind == capabilityregistry.DependencyControlPlane {
			result = append(result, OptionalDependencyRelease{
				DependencyID: definition.ID, DependencySlug: definition.DependencySlug,
				DependencyKind: string(definition.DependencyKind), Enabled: true, State: "control_plane_managed",
				Reason: "Control-plane dependencies are managed by the Vrooli control plane, not the signed scenario-component release catalog.",
			})
			continue
		}
		if definition.DependencyKind != capabilityregistry.DependencyScenario {
			result = append(result, OptionalDependencyRelease{
				DependencyID: definition.ID, DependencySlug: definition.DependencySlug,
				DependencyKind: string(definition.DependencyKind), Enabled: true, State: "unsupported_dependency_kind",
				Reason: "This optional dependency kind has no signed scenario-component lifecycle mapping.",
			})
			continue
		}

		matched := make([]struct {
			id       string
			features []string
		}, 0)
		for _, id := range componentIDs {
			entry := availableByID[id].Entry
			if entry.ID == "" {
				entry = installedByID[id].Signed.Entry
			}
			if entry.Owner != definition.DependencySlug {
				continue
			}
			var features []string
			for _, provided := range entry.Provides {
				for _, use := range definition.Uses {
					if provided.ID == use.ID {
						features = append(features, provided.ID)
						break
					}
				}
			}
			if len(features) > 0 {
				matched = append(matched, struct {
					id       string
					features []string
				}{id: id, features: features})
			}
		}
		if len(matched) == 0 {
			reason := "No signed release matches this scenario dependency owner and declared feature IDs."
			state := "unavailable"
			if catalogErr != nil {
				reason = "The signed component release catalog is unavailable, so no release match can be confirmed."
				state = "catalog_unavailable"
			}
			result = append(result, OptionalDependencyRelease{
				DependencyID: definition.ID, DependencySlug: definition.DependencySlug,
				DependencyKind: string(definition.DependencyKind), Enabled: true, State: state, Reason: reason,
			})
			continue
		}

		for _, match := range matched {
			availableRelease, hasAvailable := availableByID[match.id]
			installedRow, hasInstalled := installedByID[match.id]
			entry := availableRelease.Entry
			if !hasAvailable {
				entry = installedRow.Signed.Entry
			}
			row := OptionalDependencyRelease{
				DependencyID: definition.ID, DependencySlug: definition.DependencySlug,
				DependencyKind: string(definition.DependencyKind), Enabled: true, MatchedFeatures: match.features,
				ComponentID: entry.ID, Owner: entry.Owner, State: "available",
				CanInstallOrUpdate: hasAvailable && entry.Compatible(hostVersion, runtime.GOOS, runtime.GOARCH),
				CanRemove:          hasInstalled,
			}
			if hasAvailable {
				row.AvailableVersion = entry.Version
				if !row.CanInstallOrUpdate {
					row.State = "out_of_range"
					row.Reason = "The signed release does not declare compatibility with this app version and host platform."
				}
			}
			if hasInstalled {
				row.InstalledVersion = installedRow.Signed.Entry.Version
				if hasAvailable && !row.CanInstallOrUpdate {
					row.State = "out_of_range"
				} else if hasAvailable && row.AvailableVersion != row.InstalledVersion {
					row.State = "update_available"
				} else if hasAvailable && installedRow.State == components.NeedsUpdate {
					row.CanInstallOrUpdate = true
					row.State = components.NeedsUpdate
				} else {
					row.CanInstallOrUpdate = false
					row.State = installedRow.State
				}
				if !hasAvailable {
					if catalogErr != nil {
						row.Reason = "The signed release catalog is unavailable; the installed component remains removable."
					} else {
						row.Reason = "The installed signed release is no longer in the current catalog; removal remains available."
					}
				}
			}
			result = append(result, row)
		}
	}
	return result
}

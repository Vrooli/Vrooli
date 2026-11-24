package scenario

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"

	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
	httputil "scenario-to-desktop-api/shared/http"
	pathutil "scenario-to-desktop-api/shared/path"
)

const desktopEligibilitySource = "scenario-to-desktop platforms/electron discovery plus successful shared service-manifest projection"

// ListOptionalDependencyCatalog retains every row from the existing scenario
// status catalog. Current desktop eligibility is defined by the same
// platforms/electron discovery rule used by ListDesktopStatus; the service
// manifest remains the authority for optional dependency identity and
// requiredness.
func (h *Handler) ListOptionalDependencyCatalog() (*OptionalDependencyCatalog, error) {
	statuses, err := h.ListDesktopStatus()
	if err != nil {
		return nil, err
	}

	root := h.vrooliRoot
	if root == "" {
		root = pathutil.DetectVrooliRoot()
	}
	catalog := &OptionalDependencyCatalog{
		Source:    "service.json scenario directories + scenario-to-desktop desktop discovery + capability-registry-go.ProjectManifest",
		Scenarios: make([]OptionalDependencyScenario, 0, len(statuses.Scenarios)),
	}
	for _, status := range statuses.Scenarios {
		manifestPath := filepath.Join(root, "scenarios", status.Name, ".vrooli", "service.json")
		if _, statErr := os.Stat(manifestPath); os.IsNotExist(statErr) {
			continue
		}
		row := OptionalDependencyScenario{
			Name:              status.Name,
			Eligible:          status.HasDesktop,
			EligibilitySource: desktopEligibilitySource,
			Dependencies:      []capabilityregistry.Def{},
		}
		if !status.HasDesktop {
			row.EligibilityReasons = []string{"No platforms/electron directory is present, so the scenario status catalog does not recognize a current desktop app entrypoint."}
		}

		definitions, projectErr := capabilityregistry.ProjectManifest(manifestPath, nil)
		if projectErr != nil {
			row.Eligible = false
			row.EligibilityReasons = append(row.EligibilityReasons, "The service manifest is missing or cannot be projected by the shared capability registry, so its optional dependency catalog is incomplete.")
			row.DependencyCatalogNote = "Optional dependencies are unavailable because the service manifest could not be projected."
		} else {
			for _, definition := range definitions {
				if !definition.Required {
					row.Dependencies = append(row.Dependencies, definition)
				}
			}
		}
		catalog.Scenarios = append(catalog.Scenarios, row)
	}
	return catalog, nil
}

// OptionalDependencyCatalogHandler serves the generated catalog to clients
// that need one all-scenario eligibility view.
func (h *Handler) OptionalDependencyCatalogHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			httputil.WriteErrorWithStatus(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
			return
		}
		catalog, err := h.ListOptionalDependencyCatalog()
		if err != nil {
			httputil.WriteErrorWithStatus(w, http.StatusInternalServerError, errors.New("could not generate optional dependency catalog"))
			return
		}
		httputil.WriteJSONOK(w, catalog)
	})
}

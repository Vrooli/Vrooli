package validators

import (
	"os"
	"path/filepath"
	"strings"
)

// ScenarioComponents represents which components exist in a scenario
type ScenarioComponents struct {
	HasAPI bool
	HasUI  bool
}

// DetectScenarioComponents detects which components (API, UI) exist in a scenario
func DetectScenarioComponents(scenarioRoot string) ScenarioComponents {
	components := ScenarioComponents{}

	// Check for API component
	apiDir := filepath.Join(scenarioRoot, "api")
	if info, err := os.Stat(apiDir); err == nil && info.IsDir() {
		entries, err := os.ReadDir(apiDir)
		if err == nil {
			for _, entry := range entries {
				if strings.HasSuffix(entry.Name(), ".go") {
					components.HasAPI = true
					break
				}
			}
		}
	}

	// Check for UI component
	uiDir := filepath.Join(scenarioRoot, "ui")
	if info, err := os.Stat(uiDir); err == nil && info.IsDir() {
		pkgJSON := filepath.Join(uiDir, "package.json")
		if _, err := os.Stat(pkgJSON); err == nil {
			components.HasUI = true
		}
	}

	return components
}

// GetApplicableLayers returns which validation layers apply to a scenario
func GetApplicableLayers(components ScenarioComponents) map[string]bool {
	layers := map[string]bool{
		"E2E": true, // E2E always applicable
	}

	if components.HasAPI {
		layers["API"] = true
	}
	if components.HasUI {
		layers["UI"] = true
	}

	return layers
}

// ComponentsToSet converts ScenarioComponents to a set of component names
func (c ScenarioComponents) ToSet() map[string]bool {
	set := make(map[string]bool)
	if c.HasAPI {
		set["API"] = true
	}
	if c.HasUI {
		set["UI"] = true
	}
	return set
}

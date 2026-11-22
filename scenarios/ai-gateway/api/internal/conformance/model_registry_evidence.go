package conformance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// governedModelRegistryIDs returns model IDs only when a scenario has the
// Audio Tools-style typed and validated registry contract. The exception is
// intentionally narrow: a seed path by itself is never enough. The loader
// must validate the embedded seed, generated docs must have drift coverage,
// and a registry test must compare summarize roles/fallbacks with
// resource-ollama policy. This lets a provider catalog refer to known
// registry IDs without turning arbitrary executable literals into policy.
type governedModelRegistry struct {
	ids                     map[string]struct{}
	providerCatalogChecked  bool
	providerCatalogRendered bool
}

func governedModelRegistryIDs(root string) governedModelRegistry {
	const seedPath = "api/internal/modelcatalog/models.seed.json"
	seedBytes, ok := readScenarioFile(root, seedPath)
	if !ok {
		return governedModelRegistry{}
	}
	var seed struct {
		SchemaVersion string `json:"schema_version"`
		Models        []struct {
			ID string `json:"id"`
		} `json:"models"`
	}
	if json.Unmarshal(seedBytes, &seed) != nil || seed.SchemaVersion == "" || len(seed.Models) == 0 {
		return governedModelRegistry{}
	}
	ids := make(map[string]struct{}, len(seed.Models))
	for _, model := range seed.Models {
		id := strings.TrimSpace(model.ID)
		if id == "" {
			return governedModelRegistry{}
		}
		ids[id] = struct{}{}
	}
	if len(ids) != len(seed.Models) {
		return governedModelRegistry{}
	}

	loader, loaderOK := readScenarioFile(root, "api/internal/modelcatalog/catalog.go")
	validator, validatorOK := readScenarioFile(root, "api/internal/modelcatalog/validate.go")
	tests, testsOK := readScenarioFile(root, "api/internal/modelcatalog/catalog_test.go")
	if !loaderOK || !validatorOK || !testsOK {
		return governedModelRegistry{}
	}
	loaderText, validatorText, testText := string(loader), string(validator), string(tests)
	if !strings.Contains(loaderText, "func Load(raw []byte)") ||
		!strings.Contains(loaderText, `SupportedSchemaVersion = "`+seed.SchemaVersion+`"`) ||
		!strings.Contains(loaderText, "c.Validate()") ||
		!strings.Contains(loaderText, "go:generate") ||
		!strings.Contains(validatorText, "func (c *Catalog) ValidationIssues()") ||
		!strings.Contains(testText, "TestEmbeddedSeedIsValid") ||
		!strings.Contains(testText, "TestDocsMatchRenderer") ||
		!strings.Contains(testText, "TestSummarizeRecordsMirrorResourceOllamaPolicy") ||
		!strings.Contains(testText, "model-policy.json") ||
		!strings.Contains(testText, "resource-ollama") ||
		!strings.Contains(testText, "Fallbacks []string") ||
		!strings.Contains(testText, "Model     string") ||
		!strings.Contains(testText, "summarize.default") {
		return governedModelRegistry{}
	}
	return governedModelRegistry{
		ids:                     ids,
		providerCatalogChecked:  providerCatalogValidationEvidence(root),
		providerCatalogRendered: providerCatalogRenderEvidence(root),
	}
}

func providerCatalogValidationEvidence(root string) bool {
	loader, loaderOK := readScenarioFile(root, "api/internal/providercatalog/catalog.go")
	tests, testsOK := readScenarioFile(root, "api/internal/providercatalog/conformance_test.go")
	if !loaderOK || !testsOK {
		return false
	}
	return strings.Contains(string(loader), "func Load()") &&
		strings.Contains(string(loader), "r.validate()") &&
		strings.Contains(string(loader), "func (r *Registry) validate() error") &&
		strings.Contains(string(tests), "TestEveryProviderModelReferenceResolves") &&
		strings.Contains(string(tests), "models.ByID(modelID)")
}

func providerCatalogRenderEvidence(root string) bool {
	renderer, rendererOK := readScenarioFile(root, "api/internal/providercatalog/render.go")
	tests, testsOK := readScenarioFile(root, "api/internal/providercatalog/render_test.go")
	if !rendererOK || !testsOK {
		return false
	}
	return strings.Contains(string(renderer), "func RenderTS()") &&
		strings.Contains(string(renderer), "catalog.seed.json") &&
		strings.Contains(string(tests), "TestGeneratedTSProviderCatalogIsNotStale") &&
		strings.Contains(string(tests), "RenderTS()")
}

// readScenarioFile refuses symlinked evidence so a scenario cannot borrow
// validation markers or model IDs from outside the scanned tree.
func readScenarioFile(root, rel string) ([]byte, bool) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, false
	}
	path := filepath.Join(rootAbs, filepath.FromSlash(rel))
	checked, err := filepath.Rel(rootAbs, path)
	if err != nil || checked == ".." || strings.HasPrefix(checked, ".."+string(filepath.Separator)) || filepath.IsAbs(checked) {
		return nil, false
	}
	current := rootAbs
	for _, part := range strings.Split(checked, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return nil, false
		}
	}
	data, err := os.ReadFile(path)
	return data, err == nil
}

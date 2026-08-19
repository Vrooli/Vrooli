package targetpack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/vrooli/api-core/scopecatalog"
	"structure-health/internal/rules"
)

var concreteScopeLiteral = regexp.MustCompile(`\b[a-z0-9][a-z0-9-]*:(?:read|write|destructive)\b`)

// projectScopeVocabularyRules keeps concrete command scopes enforced by a
// relying party subordinate to the CLI-derived catalog. Wildcard grants and
// transport capabilities are deliberately outside this concrete-scope rule.
func projectScopeVocabularyRules(root string) []rules.Finding {
	if _, err := os.Stat(filepath.Join(root, "cli", "manifest.json")); err != nil {
		return nil
	}
	catalog, err := scopecatalog.Build(root)
	if err != nil {
		return []rules.Finding{finding("PROJECT_SCOPE_VOCABULARY", "error", fmt.Sprintf("build scope catalog: %v", err), filepath.ToSlash(filepath.Join("cli", "manifest.json")), "Repair the CLI manifests so the repository scope vocabulary can be derived.")}
	}

	var out []rules.Finding
	for _, base := range []string{
		"packages/api-core/discovery",
		"scenarios/agent-manager/api/internal/orchestration/phases",
		"scenarios/vrooli-bridge/api/internal/dispatch",
		"scenarios/vrooli-bridge/api/internal/registry",
	} {
		_ = filepath.WalkDir(filepath.Join(root, filepath.FromSlash(base)), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			for _, scope := range concreteScopeLiteral.FindAllString(string(raw), -1) {
				if catalog.HasScope(scope) {
					continue
				}
				rel, _ := filepath.Rel(root, path)
				out = append(out, finding("PROJECT_SCOPE_VOCABULARY", "error", fmt.Sprintf("relying party enforces unknown concrete scope %q", scope), filepath.ToSlash(rel), "Declare command authority in CLI governance metadata or use an existing catalog scope."))
			}
			return nil
		})
	}
	return out
}

// projectCLIManifestSchemaRules validates each CLI manifest at its owning
// project/scenario boundary. Catalog construction also validates manifests,
// but this rule preserves the individual owner and path in structure-health
// evidence instead of collapsing the first malformed manifest into a catalog
// error.
func projectCLIManifestSchemaRules(root string) []rules.Finding {
	paths := make([]string, 0)
	projectManifest := filepath.Join(root, "cli", "manifest.json")
	if _, err := os.Stat(projectManifest); err == nil {
		paths = append(paths, projectManifest)
	}
	_ = filepath.WalkDir(filepath.Join(root, "scenarios"), func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || entry.Name() != "manifest.json" || filepath.Base(filepath.Dir(path)) != "cli" {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if len(paths) == 0 {
		return nil
	}

	schemaPath := filepath.Join(root, ".vrooli", "schemas", "cli-manifest.schema.json")
	schemaRaw, err := os.ReadFile(schemaPath)
	if err != nil {
		return []rules.Finding{finding("PROJECT_CLI_MANIFEST_SCHEMA", "error", fmt.Sprintf("read CLI manifest schema: %v", err), ".vrooli/schemas/cli-manifest.schema.json", "Restore the canonical CLI manifest schema before validating project and scenario manifests.")}
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("cli-manifest.schema.json", bytes.NewReader(schemaRaw)); err != nil {
		return []rules.Finding{finding("PROJECT_CLI_MANIFEST_SCHEMA", "error", fmt.Sprintf("compile CLI manifest schema: %v", err), ".vrooli/schemas/cli-manifest.schema.json", "Repair the canonical CLI manifest schema.")}
	}
	schema, err := compiler.Compile("cli-manifest.schema.json")
	if err != nil {
		return []rules.Finding{finding("PROJECT_CLI_MANIFEST_SCHEMA", "error", fmt.Sprintf("compile CLI manifest schema: %v", err), ".vrooli/schemas/cli-manifest.schema.json", "Repair the canonical CLI manifest schema.")}
	}

	var out []rules.Finding
	for _, path := range paths {
		rel, _ := filepath.Rel(root, path)
		location := filepath.ToSlash(rel)
		owner := "project"
		if strings.HasPrefix(location, "scenarios/") {
			parts := strings.Split(location, "/")
			if len(parts) > 1 && parts[1] != "" {
				owner = "scenario/" + parts[1]
			}
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			out = append(out, finding("PROJECT_CLI_MANIFEST_SCHEMA", "error", fmt.Sprintf("%s CLI manifest is unreadable: %v", owner, err), location, "Restore a readable cli/manifest.json for the owning project or scenario."))
			continue
		}
		var document any
		if err := json.Unmarshal(raw, &document); err != nil {
			out = append(out, finding("PROJECT_CLI_MANIFEST_SCHEMA", "error", fmt.Sprintf("%s CLI manifest is invalid JSON: %v", owner, err), location, "Repair cli/manifest.json as valid JSON matching the canonical schema."))
			continue
		}
		if err := schema.Validate(document); err != nil {
			out = append(out, finding("PROJECT_CLI_MANIFEST_SCHEMA", "error", fmt.Sprintf("%s CLI manifest fails schema validation: %v", owner, err), location, "Repair cli/manifest.json against .vrooli/schemas/cli-manifest.schema.json."))
		}
	}
	return out
}

func unknownConcreteScopes(catalog scopecatalog.Catalog, source string) []string {
	var unknown []string
	for _, scope := range concreteScopeLiteral.FindAllString(source, -1) {
		if !catalog.HasScope(scope) {
			unknown = append(unknown, scope)
		}
	}
	return unknown
}

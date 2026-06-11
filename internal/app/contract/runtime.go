package contractapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/repocontractcheck"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
)

func NewDefaultService() Service {
	return Service{
		ResolveRootFn:     ResolveRoot,
		ValidateFn:        Validate,
		ShowFn:            LoadShowOutput,
		ResolveScenarioFn: ResolveScenario,
		MatchGlobFn:       MatchGlob,
	}
}

func ResolveRoot() (string, error) {
	return repocontract.FindRepoRootFromEnvOrCWD()
}

func RunSchemaValidation(root string) (string, bool) {
	if err := validateSchema(root); err != nil {
		return err.Error(), false
	}
	return "ok", true
}

func Validate(root string) (ValidationOutput, error) {
	schemaMessage, schemaPassed := RunSchemaValidation(root)
	report, err := repocontractcheck.Run(root)
	if err != nil {
		return ValidationOutput{}, fmt.Errorf("run repo-contract checks: %w", err)
	}

	return ValidationOutput{
		Success: schemaPassed && report.Success,
		Root:    root,
		Schema: ValidationCheck{
			Passed:  schemaPassed,
			Message: schemaMessage,
		},
		Report: report,
	}, nil
}

func LoadShowOutput() (ShowOutput, error) {
	contract, root, err := repocontract.LoadDefaultFromEnvOrCWD()
	if err != nil {
		return ShowOutput{}, err
	}
	return ShowOutput{
		Success:      true,
		Root:         root,
		ContractPath: repocontractmeta.ContractPath(root),
		Schema:       contract.Schema(),
		Version:      contract.Version(),
		Platform:     contract.Platform(),
		Markers:      contract.RootMarkers(),
		Layout:       contract.Layout(),
		Scenario:     contract.Scenario(),
		Resource:     contract.Resource(),
		Globs:        contract.Globs(),
		Environment:  contract.EnvironmentVariables(),
		Sandbox: ShowSandbox{
			FullRepoScopes:      contract.SandboxFullRepoScopes(),
			ScenarioScopePrefix: contract.SandboxScenarioScopePrefix(),
		},
		Profiles: LoadProfiles(contract),
	}, nil
}

func ResolveScenario(root, scenarioName, fileKey string) (ResolveScenarioOutput, error) {
	var (
		resolved string
		err      error
	)
	if fileKey == "" {
		resolved, err = repocontract.ResolveScenarioPath(root, scenarioName)
	} else {
		resolved, err = repocontract.ResolveScenarioFile(root, scenarioName, fileKey)
	}
	if err != nil {
		return ResolveScenarioOutput{}, err
	}
	return ResolveScenarioOutput{
		Success:  true,
		Root:     root,
		Scenario: scenarioName,
		File:     fileKey,
		Path:     resolved,
	}, nil
}

func MatchGlob(pattern, path string) (MatchGlobOutput, error) {
	matched, err := repocontract.MatchRepoGlob(pattern, path)
	if err != nil {
		return MatchGlobOutput{}, err
	}
	return MatchGlobOutput{
		Success: true,
		Pattern: pattern,
		Path:    path,
		Matched: matched,
	}, nil
}

func LoadProfiles(contract *repocontract.Contract) map[string]repocontract.Profile {
	names := []string{repocontractmeta.MiniBundleProfile}
	profiles := make(map[string]repocontract.Profile, len(names))
	for _, name := range names {
		profile, err := contract.Profile(name)
		if err == nil {
			profiles[name] = profile
		}
	}
	return profiles
}

func SortedProfileNames(profiles map[string]repocontract.Profile) []string {
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func validateSchema(root string) error {
	schemaDir := filepath.Join(root, ".vrooli", "schemas")
	schemaPath := filepath.Join(schemaDir, repocontractmeta.SchemaFilename)
	commonPath := filepath.Join(schemaDir, repocontractmeta.CommonSchemaFilename)
	contractPath := repocontractmeta.ContractPath(root)

	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("repo contract schema is invalid: read %s: %w", schemaPath, err)
	}
	commonBytes, err := os.ReadFile(commonPath)
	if err != nil {
		return fmt.Errorf("repo contract schema is invalid: read %s: %w", commonPath, err)
	}
	contractBytes, err := os.ReadFile(contractPath)
	if err != nil {
		return fmt.Errorf("repo contract validation failed: read %s: %w", contractPath, err)
	}
	schemaBytes, err = normalizeContractSchemaPatterns(schemaBytes)
	if err != nil {
		return fmt.Errorf("repo contract schema is invalid: %v", err)
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(repocontractmeta.CommonSchemaFilename, bytes.NewReader(commonBytes)); err != nil {
		return fmt.Errorf("repo contract schema is invalid: %v", err)
	}
	if err := compiler.AddResource("https://vrooli.com/schemas/"+repocontractmeta.CommonSchemaFilename, bytes.NewReader(commonBytes)); err != nil {
		return fmt.Errorf("repo contract schema is invalid: %v", err)
	}
	if err := compiler.AddResource(repocontractmeta.SchemaFilename, bytes.NewReader(schemaBytes)); err != nil {
		return fmt.Errorf("repo contract schema is invalid: %v", err)
	}
	if err := compiler.AddResource("https://vrooli.com/schemas/"+repocontractmeta.SchemaFilename, bytes.NewReader(schemaBytes)); err != nil {
		return fmt.Errorf("repo contract schema is invalid: %v", err)
	}

	schema, err := compiler.Compile("https://vrooli.com/schemas/" + repocontractmeta.SchemaFilename)
	if err != nil {
		return fmt.Errorf("repo contract schema is invalid: %v", err)
	}

	var payload any
	if err := json.Unmarshal(contractBytes, &payload); err != nil {
		return fmt.Errorf("repo contract validation failed: %v", err)
	}
	if err := schema.Validate(payload); err != nil {
		return fmt.Errorf("repo contract validation failed: %v", err)
	}
	if err := validateRepoRelativePaths(payload); err != nil {
		return fmt.Errorf("repo contract validation failed: %v", err)
	}
	return nil
}

func normalizeContractSchemaPatterns(schemaBytes []byte) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(schemaBytes, &doc); err != nil {
		return nil, fmt.Errorf("decode %s: %w", repocontractmeta.SchemaFilename, err)
	}
	definitions, _ := doc["definitions"].(map[string]any)
	for _, name := range []string{"slashPath", "slashGlob"} {
		definition, _ := definitions[name].(map[string]any)
		if definition == nil {
			return nil, fmt.Errorf("missing %s definition", name)
		}
		// The checked-in schema uses lookaheads that Python accepts but RE2 rejects.
		// Keep the string/non-empty/backslash-free portion in-schema, then enforce the
		// remaining repository-relative constraints in Go after validation.
		definition["pattern"] = `^[^\\]+$`
	}
	normalized, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	return normalized, nil
}

func validateRepoRelativePaths(payload any) error {
	doc, _ := payload.(map[string]any)
	if doc == nil {
		return fmt.Errorf("repo contract payload must be an object")
	}

	for _, value := range stringListAt(doc, "root", "markers", "required_dirs") {
		if err := validateRepoRelativePath(value); err != nil {
			return fmt.Errorf("root.markers.required_dirs: %w", err)
		}
	}
	for _, value := range stringListAt(doc, "root", "markers", "required_files") {
		if err := validateRepoRelativePath(value); err != nil {
			return fmt.Errorf("root.markers.required_files: %w", err)
		}
	}
	for _, key := range []string{"project_config_dir", "scenario_dir", "resource_dir", "template_dir", "package_dir", "command_dir", "internal_dir", "docs_dir"} {
		if err := validateRepoRelativePath(stringAt(doc, "layout", key)); err != nil {
			return fmt.Errorf("layout.%s: %w", key, err)
		}
	}
	for _, value := range stringListAt(doc, "scenario", "required_files") {
		if err := validateRepoRelativePath(value); err != nil {
			return fmt.Errorf("scenario.required_files: %w", err)
		}
	}
	for _, key := range []string{"service", "docs", "requirements", "api", "ui", "cli", "initialization"} {
		if err := validateRepoRelativePath(stringAt(doc, "scenario", "well_known_paths", key)); err != nil {
			return fmt.Errorf("scenario.well_known_paths.%s: %w", key, err)
		}
	}
	if err := validateRepoRelativePath(stringAt(doc, "resource", "manifest")); err != nil {
		return fmt.Errorf("resource.manifest: %w", err)
	}
	for _, key := range []string{"docs", "initialization"} {
		if err := validateRepoRelativePath(stringAt(doc, "resource", "well_known_paths", key)); err != nil {
			return fmt.Errorf("resource.well_known_paths.%s: %w", key, err)
		}
	}
	for key, rawEntry := range mapAt(doc, "runtime_home", "entries") {
		entry, _ := rawEntry.(map[string]any)
		if entry == nil {
			continue
		}
		if err := validateRepoRelativePath(stringAt(entry, "path")); err != nil {
			return fmt.Errorf("runtime_home.entries.%s.path: %w", key, err)
		}
	}

	profiles := mapAt(doc, "profiles")
	for profileName, rawProfile := range profiles {
		profile, _ := rawProfile.(map[string]any)
		for _, value := range stringList(profile["include"]) {
			if err := validateRepoRelativePath(value); err != nil {
				return fmt.Errorf("profiles.%s.include: %w", profileName, err)
			}
		}
		for _, value := range stringList(profile["optional_include"]) {
			if err := validateRepoRelativePath(value); err != nil {
				return fmt.Errorf("profiles.%s.optional_include: %w", profileName, err)
			}
		}
		for _, value := range stringList(profile["exclude"]) {
			if err := validateRepoRelativePath(value); err != nil {
				return fmt.Errorf("profiles.%s.exclude: %w", profileName, err)
			}
		}
	}
	return nil
}

var windowsDrivePattern = regexp.MustCompile(`^[A-Za-z]:`)

func validateRepoRelativePath(value string) error {
	value = strings.TrimSpace(value)
	switch {
	case value == "":
		return fmt.Errorf("path cannot be empty")
	case strings.Contains(value, `\`):
		return fmt.Errorf("path %q must be slash-normalized", value)
	case strings.HasPrefix(value, "/"):
		return fmt.Errorf("path %q must be repository-relative", value)
	case windowsDrivePattern.MatchString(value):
		return fmt.Errorf("path %q must not use a Windows drive prefix", value)
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == ".." {
			return fmt.Errorf("path %q must not contain '..' segments", value)
		}
	}
	return nil
}

func stringAt(value any, keys ...string) string {
	current := value
	for _, key := range keys {
		next, _ := current.(map[string]any)
		if next == nil {
			return ""
		}
		current = next[key]
	}
	text, _ := current.(string)
	return text
}

func mapAt(value any, keys ...string) map[string]any {
	current := value
	for _, key := range keys {
		next, _ := current.(map[string]any)
		if next == nil {
			return nil
		}
		current = next[key]
	}
	result, _ := current.(map[string]any)
	return result
}

func stringListAt(value any, keys ...string) []string {
	current := value
	for _, key := range keys {
		next, _ := current.(map[string]any)
		if next == nil {
			return nil
		}
		current = next[key]
	}
	return stringList(current)
}

func stringList(value any) []string {
	items, _ := value.([]any)
	result := make([]string, 0, len(items))
	for _, item := range items {
		text, _ := item.(string)
		result = append(result, text)
	}
	return result
}

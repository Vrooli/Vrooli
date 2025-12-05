package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	sharedartifacts "test-genie/internal/shared/artifacts"
)

// NativeResolver resolves workflows using pure Go (no Python dependency).
type NativeResolver struct {
	scenarioDir string
	fixturesDir string
}

// NewNativeResolver creates a new native workflow resolver.
func NewNativeResolver(scenarioDir string) *NativeResolver {
	return &NativeResolver{
		scenarioDir: scenarioDir,
		fixturesDir: filepath.Join(scenarioDir, "test", "playbooks", "__subflows"),
	}
}

// ResolveWorkflow loads and fully resolves a workflow definition.
func (r *NativeResolver) ResolveWorkflow(workflowPath string) (map[string]any, error) {
	// Ensure absolute path
	if !filepath.IsAbs(workflowPath) {
		workflowPath = filepath.Join(r.scenarioDir, filepath.FromSlash(workflowPath))
	}

	// Load workflow
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow: %w", err)
	}

	var workflow map[string]any
	if err := json.Unmarshal(data, &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse workflow JSON: %w", err)
	}

	// Load fixtures
	fixtures, err := r.loadFixtures()
	if err != nil {
		return nil, fmt.Errorf("failed to load fixtures: %w", err)
	}

	// Load seed state
	seedState, err := r.loadSeedState()
	if err != nil {
		return nil, fmt.Errorf("failed to load seed state: %w", err)
	}

	// Resolve workflow
	fixtureReqs := make(map[string]bool)
	if err := r.resolveDefinition(workflow, fixtures, seedState, fixtureReqs, ""); err != nil {
		return nil, err
	}

	// Add fixture requirements to metadata
	r.annotateFixtureRequirements(workflow, fixtureReqs)

	// Resolve selectors
	if err := r.resolveSelectors(workflow); err != nil {
		return nil, err
	}

	return workflow, nil
}

// FixtureDefinition represents a loaded fixture.
type FixtureDefinition struct {
	Definition   map[string]any
	Source       string
	FixtureID    string
	Description  string
	Requirements []string
	Parameters   []FixtureParameter
	Reset        string
}

// FixtureParameter defines a fixture parameter.
type FixtureParameter struct {
	Name        string
	Type        string   // string, number, boolean, enum
	Required    bool
	Default     any
	EnumValues  []string
	Description string
}

// loadFixtures loads all fixture definitions from the fixtures directory.
func (r *NativeResolver) loadFixtures() (map[string]*FixtureDefinition, error) {
	fixtures := make(map[string]*FixtureDefinition)

	if _, err := os.Stat(r.fixturesDir); os.IsNotExist(err) {
		return fixtures, nil
	}

	err := filepath.WalkDir(r.fixturesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read fixture %s: %w", path, err)
		}

		var doc map[string]any
		if err := json.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("failed to parse fixture %s: %w", path, err)
		}

		fixture, err := r.parseFixture(doc, path)
		if err != nil {
			return err
		}

		if _, exists := fixtures[fixture.FixtureID]; exists {
			return fmt.Errorf("duplicate fixture ID %q in %s", fixture.FixtureID, path)
		}

		fixtures[fixture.FixtureID] = fixture
		return nil
	})

	return fixtures, err
}

// parseFixture extracts fixture metadata and definition from a document.
func (r *NativeResolver) parseFixture(doc map[string]any, source string) (*FixtureDefinition, error) {
	metadata, _ := doc["metadata"].(map[string]any)
	if metadata == nil {
		return nil, fmt.Errorf("fixture %s missing metadata", source)
	}

	fixtureID, _ := metadata["fixture_id"].(string)
	if fixtureID == "" {
		// Try camelCase variant
		fixtureID, _ = metadata["fixtureId"].(string)
	}
	if fixtureID == "" {
		return nil, fmt.Errorf("fixture %s missing fixture_id", source)
	}

	fixture := &FixtureDefinition{
		Definition:  make(map[string]any),
		Source:      source,
		FixtureID:   fixtureID,
		Description: getString(metadata, "description"),
		Reset:       getString(metadata, "reset"),
	}

	// Extract requirements
	if reqs, ok := metadata["requirements"].([]any); ok {
		for _, r := range reqs {
			if s, ok := r.(string); ok {
				fixture.Requirements = append(fixture.Requirements, s)
			}
		}
	}

	// Extract parameters
	if params, ok := metadata["parameters"].([]any); ok {
		for _, p := range params {
			if pm, ok := p.(map[string]any); ok {
				param := FixtureParameter{
					Name:        getString(pm, "name"),
					Type:        strings.ToLower(getStringDefault(pm, "type", "string")),
					Required:    getBool(pm, "required"),
					Default:     pm["default"],
					Description: getString(pm, "description"),
				}
				if enumVals, ok := pm["enumValues"].([]any); ok {
					for _, v := range enumVals {
						param.EnumValues = append(param.EnumValues, fmt.Sprint(v))
					}
				}
				fixture.Parameters = append(fixture.Parameters, param)
			}
		}
	}

	// Copy definition (everything except metadata)
	for k, v := range doc {
		if k != "metadata" {
			fixture.Definition[k] = v
		}
	}

	// Ensure nodes exist
	if _, ok := fixture.Definition["nodes"].([]any); !ok {
		return nil, fmt.Errorf("fixture %s must define nodes array", source)
	}

	return fixture, nil
}

// loadSeedState loads the seed state file if it exists.
func (r *NativeResolver) loadSeedState() (map[string]any, error) {
	seedPath := sharedartifacts.SeedStatePath(r.scenarioDir)
	data, err := os.ReadFile(seedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]any), nil
		}
		return nil, err
	}

	var state map[string]any
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse seed state: %w", err)
	}
	return state, nil
}

var (
	fixtureRefPattern = regexp.MustCompile(`^@fixture/([A-Za-z0-9_.-]+)(\s*\(.*\))?$`)
	seedRefPattern    = regexp.MustCompile(`^@seed/([A-Za-z0-9_.-]+)$`)
	templatePattern   = regexp.MustCompile(`\$\{fixture\.([A-Za-z0-9_]+)\}`)
)

// resolveDefinition recursively resolves fixture references in a workflow definition.
func (r *NativeResolver) resolveDefinition(
	definition map[string]any,
	fixtures map[string]*FixtureDefinition,
	seedState map[string]any,
	fixtureReqs map[string]bool,
	pointer string,
) error {
	nodes, ok := definition["nodes"].([]any)
	if !ok {
		return nil
	}

	var aggregatedReset string

	for i, node := range nodes {
		nodeMap, ok := node.(map[string]any)
		if !ok {
			continue
		}
		nodePointer := fmt.Sprintf("%s/nodes/%d", pointer, i)

		data, ok := nodeMap["data"].(map[string]any)
		if !ok {
			continue
		}

		workflowID, _ := data["workflowId"].(string)
		if strings.HasPrefix(workflowID, "@fixture/") {
			// Parse fixture reference
			slug, args, err := parseFixtureReference(workflowID)
			if err != nil {
				return fmt.Errorf("at %s: %w", nodePointer, err)
			}

			fixture, ok := fixtures[slug]
			if !ok {
				return fmt.Errorf("at %s: unknown fixture %q", nodePointer, slug)
			}

			// Resolve parameters
			params, err := r.resolveFixtureParameters(fixture, args, seedState, nodePointer)
			if err != nil {
				return err
			}

			// Clone and process fixture definition
			nested := cloneMap(fixture.Definition)
			r.applyFixtureTemplates(nested, params)

			// Add fixture requirements
			for _, req := range fixture.Requirements {
				fixtureReqs[req] = true
			}

			// Recursively resolve nested workflow
			if err := r.resolveDefinition(nested, fixtures, seedState, fixtureReqs, nodePointer); err != nil {
				return err
			}

			// Replace workflowId with workflowDefinition
			data["workflowDefinition"] = nested
			delete(data, "workflowId")

			// Merge reset values
			if fixture.Reset != "" {
				aggregatedReset = mergeResetValues(aggregatedReset, fixture.Reset)
			}
		} else if nestedDef, ok := data["workflowDefinition"].(map[string]any); ok {
			// Recursively resolve existing nested definition
			if err := r.resolveDefinition(nestedDef, fixtures, seedState, fixtureReqs, nodePointer); err != nil {
				return err
			}
		}
	}

	// Update reset in metadata
	r.updateResetMetadata(definition, aggregatedReset)

	return nil
}

// parseFixtureReference parses "@fixture/slug(args)" into components.
func parseFixtureReference(ref string) (slug string, args map[string]string, err error) {
	match := fixtureRefPattern.FindStringSubmatch(strings.TrimSpace(ref))
	if match == nil {
		return "", nil, fmt.Errorf("invalid fixture reference: %s", ref)
	}

	slug = match[1]
	args = make(map[string]string)

	if match[2] != "" {
		argsStr := strings.TrimSpace(match[2])
		if strings.HasPrefix(argsStr, "(") && strings.HasSuffix(argsStr, ")") {
			inner := argsStr[1 : len(argsStr)-1]
			for _, part := range splitArgs(inner) {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				eqIdx := strings.Index(part, "=")
				if eqIdx == -1 {
					return "", nil, fmt.Errorf("invalid fixture argument: %s", part)
				}
				key := strings.TrimSpace(part[:eqIdx])
				value := strings.TrimSpace(part[eqIdx+1:])
				// Remove quotes if present
				if len(value) >= 2 {
					if (value[0] == '"' && value[len(value)-1] == '"') ||
						(value[0] == '\'' && value[len(value)-1] == '\'') {
						value = value[1 : len(value)-1]
					}
				}
				args[key] = value
			}
		}
	}

	return slug, args, nil
}

// splitArgs splits argument string respecting quotes.
func splitArgs(s string) []string {
	var parts []string
	var current strings.Builder
	var inQuote rune

	for _, ch := range s {
		switch {
		case inQuote != 0:
			current.WriteRune(ch)
			if ch == inQuote {
				inQuote = 0
			}
		case ch == '"' || ch == '\'':
			inQuote = ch
			current.WriteRune(ch)
		case ch == ',':
			parts = append(parts, current.String())
			current.Reset()
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

// resolveFixtureParameters validates and coerces fixture parameters.
func (r *NativeResolver) resolveFixtureParameters(
	fixture *FixtureDefinition,
	provided map[string]string,
	seedState map[string]any,
	pointer string,
) (map[string]any, error) {
	resolved := make(map[string]any)

	// Check for unknown parameters
	known := make(map[string]bool)
	for _, p := range fixture.Parameters {
		known[p.Name] = true
	}
	for name := range provided {
		if !known[name] {
			return nil, fmt.Errorf("at %s: fixture %q received unknown parameter %q",
				pointer, fixture.FixtureID, name)
		}
	}

	// Process each defined parameter
	for _, param := range fixture.Parameters {
		rawValue, hasProvided := provided[param.Name]

		if !hasProvided {
			if param.Default != nil {
				// Use default, resolving seed references
				resolved[param.Name] = r.resolveSeedReference(param.Default, seedState)
			} else if param.Required {
				return nil, fmt.Errorf("at %s: fixture %q missing required parameter %q",
					pointer, fixture.FixtureID, param.Name)
			}
			continue
		}

		// Resolve seed references in provided value
		value := r.resolveSeedReferenceStr(rawValue, seedState)

		// Coerce to expected type
		coerced, err := coerceParameterValue(param, value)
		if err != nil {
			return nil, fmt.Errorf("at %s: fixture %q parameter %q: %w",
				pointer, fixture.FixtureID, param.Name, err)
		}
		resolved[param.Name] = coerced
	}

	return resolved, nil
}

// resolveSeedReference resolves @seed/ tokens in a value.
func (r *NativeResolver) resolveSeedReference(value any, seedState map[string]any) any {
	if s, ok := value.(string); ok {
		return r.resolveSeedReferenceStr(s, seedState)
	}
	return value
}

// resolveSeedReferenceStr resolves @seed/ tokens in a string.
func (r *NativeResolver) resolveSeedReferenceStr(s string, seedState map[string]any) any {
	match := seedRefPattern.FindStringSubmatch(strings.TrimSpace(s))
	if match == nil {
		return s
	}
	key := match[1]
	if val, ok := seedState[key]; ok {
		return val
	}
	return s // Return original if key not found
}

// coerceParameterValue coerces a string value to the expected parameter type.
func coerceParameterValue(param FixtureParameter, value any) (any, error) {
	strVal := fmt.Sprint(value)

	switch param.Type {
	case "string":
		return strVal, nil
	case "number":
		// Check if it contains a decimal point
		if strings.Contains(strVal, ".") {
			var floatVal float64
			if _, err := fmt.Sscanf(strVal, "%f", &floatVal); err == nil {
				return floatVal, nil
			}
		} else {
			var intVal int64
			if _, err := fmt.Sscanf(strVal, "%d", &intVal); err == nil {
				return intVal, nil
			}
		}
		return nil, fmt.Errorf("must be numeric, got %q", strVal)
	case "boolean":
		lower := strings.ToLower(strVal)
		switch lower {
		case "true", "1", "yes", "on":
			return true, nil
		case "false", "0", "no", "off":
			return false, nil
		}
		return nil, fmt.Errorf("must be boolean, got %q", strVal)
	case "enum":
		for _, allowed := range param.EnumValues {
			if strVal == allowed {
				return strVal, nil
			}
		}
		return nil, fmt.Errorf("must be one of %v, got %q", param.EnumValues, strVal)
	default:
		return strVal, nil
	}
}

// applyFixtureTemplates substitutes ${fixture.param} templates in a definition.
func (r *NativeResolver) applyFixtureTemplates(definition any, params map[string]any) {
	switch v := definition.(type) {
	case map[string]any:
		for key, val := range v {
			v[key] = r.substituteTemplates(val, params)
		}
	case []any:
		for i, val := range v {
			v[i] = r.substituteTemplates(val, params)
		}
	}
}

// substituteTemplates replaces ${fixture.param} patterns in values.
func (r *NativeResolver) substituteTemplates(value any, params map[string]any) any {
	switch v := value.(type) {
	case string:
		return templatePattern.ReplaceAllStringFunc(v, func(match string) string {
			paramName := templatePattern.FindStringSubmatch(match)[1]
			if val, ok := params[paramName]; ok {
				return fmt.Sprint(val)
			}
			return match
		})
	case map[string]any:
		for key, val := range v {
			v[key] = r.substituteTemplates(val, params)
		}
		return v
	case []any:
		for i, val := range v {
			v[i] = r.substituteTemplates(val, params)
		}
		return v
	default:
		return value
	}
}

// annotateFixtureRequirements adds collected fixture requirements to workflow metadata.
func (r *NativeResolver) annotateFixtureRequirements(workflow map[string]any, reqs map[string]bool) {
	if len(reqs) == 0 {
		return
	}

	metadata, _ := workflow["metadata"].(map[string]any)
	if metadata == nil {
		metadata = make(map[string]any)
		workflow["metadata"] = metadata
	}

	var reqList []string
	for req := range reqs {
		reqList = append(reqList, req)
	}
	sort.Strings(reqList)

	// Convert to []any for consistent JSON handling
	reqAny := make([]any, len(reqList))
	for i, r := range reqList {
		reqAny[i] = r
	}
	metadata["requirementsFromFixtures"] = reqAny
}

// updateResetMetadata updates the reset value in workflow metadata.
func (r *NativeResolver) updateResetMetadata(definition map[string]any, aggregatedReset string) {
	metadata, _ := definition["metadata"].(map[string]any)
	if metadata == nil {
		metadata = make(map[string]any)
		definition["metadata"] = metadata
	}

	currentReset, _ := metadata["reset"].(string)
	finalReset := mergeResetValues(currentReset, aggregatedReset)
	if finalReset == "" {
		finalReset = "none"
	}
	metadata["reset"] = finalReset
}

// mergeResetValues returns the higher-priority reset value.
func mergeResetValues(a, b string) string {
	priority := map[string]int{"none": 0, "full": 1}
	if priority[b] > priority[a] {
		return b
	}
	return a
}

// cloneMap creates a deep copy of a map.
func cloneMap(m map[string]any) map[string]any {
	data, _ := json.Marshal(m)
	var result map[string]any
	json.Unmarshal(data, &result)
	return result
}

// Helper functions for safe type extraction.
func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getStringDefault(m map[string]any, key, def string) string {
	if v, ok := m[key].(string); ok && v != "" {
		return v
	}
	return def
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

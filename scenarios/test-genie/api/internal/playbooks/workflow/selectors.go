package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	selectorManifestFile = "selectors.manifest.json"
)

var (
	selectorPattern    = regexp.MustCompile(`@selector/([A-Za-z0-9_.-]+)(\([^)]*\))?`)
	dataTestIDPattern  = regexp.MustCompile(`\[data-testid[^\]]*\]`)
	selectorDupPattern = regexp.MustCompile(`\s*/\*dup-\d+\*/`)
	templateVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)
)

// SelectorManifest represents the structure of selectors.manifest.json.
type SelectorManifest struct {
	Selectors        map[string]SelectorEntry   `json:"selectors"`
	DynamicSelectors map[string]DynamicSelector `json:"dynamicSelectors"`
}

// SelectorEntry represents a static selector entry.
type SelectorEntry struct {
	Selector string `json:"selector"`
}

// DynamicSelector represents a parameterized selector.
type DynamicSelector struct {
	SelectorPattern string          `json:"selectorPattern"`
	Params          []SelectorParam `json:"params"`
}

// SelectorParam represents a dynamic selector parameter.
type SelectorParam struct {
	Name   string   `json:"name"`
	Type   string   `json:"type"`
	Values []string `json:"values"` // For enum type
}

// resolveSelectors resolves all @selector/ tokens in a workflow definition.
func (r *NativeResolver) resolveSelectors(workflow map[string]any) error {
	manifest, manifestPath, err := r.loadSelectorManifest()
	if err != nil {
		return err
	}
	if manifest == nil {
		// No manifest, nothing to resolve
		return nil
	}

	return r.injectSelectors(workflow, manifest, manifestPath, "")
}

// loadSelectorManifest loads the selector manifest from the scenario's UI directory.
func (r *NativeResolver) loadSelectorManifest() (*SelectorManifest, string, error) {
	// Try both possible locations
	candidates := []string{
		filepath.Join(r.scenarioDir, "ui", "src", "consts", selectorManifestFile),
		filepath.Join(r.scenarioDir, "ui", "src", "constants", selectorManifestFile),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, "", fmt.Errorf("failed to read selector manifest: %w", err)
			}

			var manifest SelectorManifest
			if err := json.Unmarshal(data, &manifest); err != nil {
				return nil, "", fmt.Errorf("failed to parse selector manifest: %w", err)
			}

			return &manifest, path, nil
		}
	}

	return nil, "", nil // No manifest found is OK
}

// injectSelectors recursively resolves @selector/ tokens in a definition.
func (r *NativeResolver) injectSelectors(
	value any,
	manifest *SelectorManifest,
	manifestPath string,
	pointer string,
) error {
	switch v := value.(type) {
	case map[string]any:
		for key, val := range v {
			nextPointer := fmt.Sprintf("%s/%s", pointer, key)
			resolved, err := r.resolveValue(val, manifest, manifestPath, nextPointer)
			if err != nil {
				return err
			}
			v[key] = resolved
		}
	case []any:
		for i, val := range v {
			nextPointer := fmt.Sprintf("%s/%d", pointer, i)
			resolved, err := r.resolveValue(val, manifest, manifestPath, nextPointer)
			if err != nil {
				return err
			}
			v[i] = resolved
		}
	}
	return nil
}

// resolveValue resolves @selector/ tokens in a single value.
func (r *NativeResolver) resolveValue(
	value any,
	manifest *SelectorManifest,
	manifestPath string,
	pointer string,
) (any, error) {
	switch v := value.(type) {
	case string:
		// Check for raw data-testid (should use @selector instead)
		if manifest != nil && dataTestIDPattern.MatchString(v) {
			return nil, fmt.Errorf("raw data-testid selector detected at %s; use @selector/<key> instead", pointer)
		}

		// Replace @selector/ tokens
		var lastErr error
		result := selectorPattern.ReplaceAllStringFunc(v, func(match string) string {
			submatch := selectorPattern.FindStringSubmatch(match)
			selectorKey := strings.TrimSpace(submatch[1])
			argsSegment := submatch[2]

			resolved, err := r.resolveSelector(selectorKey, argsSegment, manifest, manifestPath, pointer)
			if err != nil {
				lastErr = err
				return match
			}
			return resolved
		})

		if lastErr != nil {
			return nil, lastErr
		}

		// Strip /*dup-N*/ suffixes
		result = selectorDupPattern.ReplaceAllString(result, "")

		return result, nil

	case map[string]any:
		if err := r.injectSelectors(v, manifest, manifestPath, pointer); err != nil {
			return nil, err
		}
		return v, nil

	case []any:
		if err := r.injectSelectors(v, manifest, manifestPath, pointer); err != nil {
			return nil, err
		}
		return v, nil

	default:
		return value, nil
	}
}

// resolveSelector resolves a single @selector/ token.
func (r *NativeResolver) resolveSelector(
	selectorKey string,
	argsSegment string,
	manifest *SelectorManifest,
	manifestPath string,
	pointer string,
) (string, error) {
	if selectorKey == "" {
		return "", fmt.Errorf("empty selector reference at %s", pointer)
	}

	// Try static selector first
	if entry, ok := manifest.Selectors[selectorKey]; ok {
		if argsSegment != "" && strings.TrimSpace(argsSegment) != "()" {
			return "", fmt.Errorf("selector @selector/%s does not accept parameters (%s)", selectorKey, pointer)
		}
		return entry.Selector, nil
	}

	// Try dynamic selector
	dynamic, ok := manifest.DynamicSelectors[selectorKey]
	if !ok {
		return "", r.selectorNotFoundError(selectorKey, manifest, manifestPath, pointer)
	}

	return r.resolveDynamicSelector(selectorKey, argsSegment, dynamic, pointer)
}

// resolveDynamicSelector resolves a parameterized selector.
func (r *NativeResolver) resolveDynamicSelector(
	selectorKey string,
	argsSegment string,
	dynamic DynamicSelector,
	pointer string,
) (string, error) {
	// Parse provided arguments
	hasArgs := argsSegment != "" && strings.TrimSpace(argsSegment) != "()"
	provided := make(map[string]string)

	if hasArgs {
		args := strings.TrimSpace(argsSegment)
		if !strings.HasPrefix(args, "(") || !strings.HasSuffix(args, ")") {
			return "", fmt.Errorf("selector parameters must use parentheses syntax at %s", pointer)
		}
		inner := args[1 : len(args)-1]

		for _, part := range strings.Split(inner, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			eqIdx := strings.Index(part, "=")
			if eqIdx == -1 {
				return "", fmt.Errorf("invalid selector parameter %q at %s; expected key=value", part, pointer)
			}
			key := strings.TrimSpace(part[:eqIdx])
			val := strings.TrimSpace(part[eqIdx+1:])
			// Remove quotes
			if len(val) >= 2 {
				if (val[0] == '"' && val[len(val)-1] == '"') ||
					(val[0] == '\'' && val[len(val)-1] == '\'') {
					val = val[1 : len(val)-1]
				}
			}
			provided[key] = val
		}
	}

	// No parameters defined
	if len(dynamic.Params) == 0 {
		if len(provided) > 0 {
			return "", fmt.Errorf("selector @selector/%s does not accept parameters (%s)", selectorKey, pointer)
		}
		// Check if template has variables
		if templateVarPattern.MatchString(dynamic.SelectorPattern) {
			return "", fmt.Errorf("selector @selector/%s requires parameters (%s)", selectorKey, pointer)
		}
		return dynamic.SelectorPattern, nil
	}

	// Parameters required
	if !hasArgs {
		return "", fmt.Errorf("selector @selector/%s requires parameters (%s)", selectorKey, pointer)
	}

	// Build expected parameter map
	expected := make(map[string]SelectorParam)
	for _, p := range dynamic.Params {
		expected[p.Name] = p
	}

	// Check for missing parameters
	var missing []string
	for name := range expected {
		if _, ok := provided[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return "", fmt.Errorf("selector @selector/%s missing parameter(s): %s (%s)",
			selectorKey, strings.Join(missing, ", "), pointer)
	}

	// Check for extra parameters
	var extra []string
	for name := range provided {
		if _, ok := expected[name]; !ok {
			extra = append(extra, name)
		}
	}
	if len(extra) > 0 {
		return "", fmt.Errorf("selector @selector/%s received unknown parameter(s): %s (%s)",
			selectorKey, strings.Join(extra, ", "), pointer)
	}

	// Validate parameter types
	for name, value := range provided {
		param := expected[name]
		if param.Type == "number" {
			var num float64
			if _, err := fmt.Sscanf(value, "%f", &num); err != nil {
				return "", fmt.Errorf("selector @selector/%s parameter %q must be numeric (%s)",
					selectorKey, name, pointer)
			}
		}
		if param.Type == "enum" {
			valid := false
			for _, allowed := range param.Values {
				if value == allowed {
					valid = true
					break
				}
			}
			if !valid {
				return "", fmt.Errorf("selector @selector/%s parameter %q must be one of: %s (%s)",
					selectorKey, name, strings.Join(param.Values, ", "), pointer)
			}
		}
	}

	// Substitute template variables
	result := templateVarPattern.ReplaceAllStringFunc(dynamic.SelectorPattern, func(match string) string {
		varName := templateVarPattern.FindStringSubmatch(match)[1]
		if val, ok := provided[varName]; ok {
			return val
		}
		return match
	})

	return result, nil
}

// selectorNotFoundError generates a helpful error message for missing selectors.
func (r *NativeResolver) selectorNotFoundError(
	selectorKey string,
	manifest *SelectorManifest,
	manifestPath string,
	pointer string,
) error {
	// Find similar selectors for suggestions
	var suggestions []string
	parts := strings.Split(selectorKey, ".")

	for key := range manifest.Selectors {
		keyParts := strings.Split(key, ".")
		for _, part := range parts {
			for _, kp := range keyParts {
				if part == kp {
					suggestions = append(suggestions, key)
					break
				}
			}
		}
	}
	for key := range manifest.DynamicSelectors {
		keyParts := strings.Split(key, ".")
		for _, part := range parts {
			for _, kp := range keyParts {
				if part == kp {
					suggestions = append(suggestions, key)
					break
				}
			}
		}
	}

	msg := fmt.Sprintf("selector @selector/%s not found at %s", selectorKey, pointer)
	if len(suggestions) > 0 {
		// Deduplicate and limit suggestions
		seen := make(map[string]bool)
		var unique []string
		for _, s := range suggestions {
			if !seen[s] {
				seen[s] = true
				unique = append(unique, "@selector/"+s)
			}
			if len(unique) >= 5 {
				break
			}
		}
		msg += fmt.Sprintf("\n  Did you mean: %s", strings.Join(unique, ", "))
	}
	msg += fmt.Sprintf("\n  Register the selector in %s", manifestPath)

	return fmt.Errorf("%s", msg)
}

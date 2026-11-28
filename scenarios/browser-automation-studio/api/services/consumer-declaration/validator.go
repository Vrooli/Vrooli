// Package consumerdeclaration validates the public, scenario-owned BAS adoption contract.
// It deliberately has no dependency on any scenario that consumes BAS.
package consumerdeclaration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const SchemaVersion = "browser-automation-studio.consumer-declaration/v1"

var (
	profileKey = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)
	variable   = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,63}$`)
)

// Declaration contains source-controlled, non-secret BAS preferences only.
// Runtime profile IDs and browser state never appear here.
type Declaration struct {
	SchemaVersion string    `json:"schemaVersion"`
	Profiles      []Profile `json:"profiles"`
}
type Profile struct {
	Key              string         `json:"key"`
	WorkflowRef      string         `json:"workflowRef"`
	AllowedVariables []string       `json:"allowedVariables,omitempty"`
	Preferences      map[string]any `json:"preferences,omitempty"`
}

// Result reports deterministic validation failures without retaining input secrets.
type Result struct{ Issues []string }

func (r Result) Valid() bool { return len(r.Issues) == 0 }

// Validate parses a declaration strictly and rejects secret-bearing or runtime state fields.
func Validate(raw []byte) (Declaration, Result) {
	var envelope map[string]json.RawMessage
	var result Result
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return Declaration{}, Result{Issues: []string{"invalid JSON: " + err.Error()}}
	}
	for key := range envelope {
		if key != "schemaVersion" && key != "profiles" {
			result.Issues = append(result.Issues, "unknown top-level field: "+key)
		}
	}
	if containsForbidden(raw, []string{"cookie", "credential", "password", "secret", "token", "storageState", "runtimeProfile", "profileId", "proxy"}) {
		result.Issues = append(result.Issues, "declaration contains a forbidden secret or runtime-state field")
	}
	var declaration Declaration
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&declaration); err != nil {
		result.Issues = append(result.Issues, "invalid declaration shape: "+err.Error())
		return declaration, normalize(result)
	}
	if declaration.SchemaVersion != SchemaVersion {
		result.Issues = append(result.Issues, "schemaVersion must be "+SchemaVersion)
	}
	if len(declaration.Profiles) == 0 {
		result.Issues = append(result.Issues, "profiles must contain at least one entry")
	}
	keys := map[string]bool{}
	for i, profile := range declaration.Profiles {
		prefix := fmt.Sprintf("profiles[%d]", i)
		if !profileKey.MatchString(profile.Key) {
			result.Issues = append(result.Issues, prefix+".key is invalid")
		}
		if keys[profile.Key] {
			result.Issues = append(result.Issues, prefix+".key must be unique")
		}
		keys[profile.Key] = true
		if strings.TrimSpace(profile.WorkflowRef) == "" {
			result.Issues = append(result.Issues, prefix+".workflowRef is required")
		}
		variables := map[string]bool{}
		for _, name := range profile.AllowedVariables {
			if !variable.MatchString(name) {
				result.Issues = append(result.Issues, prefix+".allowedVariables contains invalid name")
			}
			if variables[name] {
				result.Issues = append(result.Issues, prefix+".allowedVariables must be unique")
			}
			variables[name] = true
		}
		for key, value := range profile.Preferences {
			if !safePreference(value) {
				result.Issues = append(result.Issues, prefix+".preferences."+key+" must be a scalar non-secret preference")
			}
		}
	}
	return declaration, normalize(result)
}
func containsForbidden(raw []byte, forbidden []string) bool {
	var walk func(any) bool
	walk = func(value any) bool {
		switch v := value.(type) {
		case map[string]any:
			for k, child := range v {
				lower := strings.ToLower(k)
				for _, word := range forbidden {
					if strings.Contains(lower, strings.ToLower(word)) {
						return true
					}
				}
				if walk(child) {
					return true
				}
			}
		case []any:
			for _, child := range v {
				if walk(child) {
					return true
				}
			}
		}
		return false
	}
	var value any
	return json.Unmarshal(raw, &value) == nil && walk(value)
}
func safePreference(value any) bool {
	switch value.(type) {
	case string, bool, float64, nil:
		return true
	default:
		return false
	}
}
func normalize(result Result) Result {
	sort.Strings(result.Issues)
	result.Issues = slicesCompact(result.Issues)
	return result
}
func slicesCompact(values []string) []string {
	out := values[:0]
	for _, value := range values {
		if len(out) == 0 || out[len(out)-1] != value {
			out = append(out, value)
		}
	}
	return out
}

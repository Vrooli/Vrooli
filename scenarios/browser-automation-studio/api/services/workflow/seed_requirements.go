package workflow

import (
	"fmt"
	"strings"

	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

const (
	seedRequiredLabel = "seed_required"
	seedKeysLabel     = "seed_keys"
	seedAppliedKey    = "seed_applied"
)

// SeedRequirementError indicates missing seed data for workflows that declare a seed dependency.
type SeedRequirementError struct {
	MissingKeys []string
}

func (e *SeedRequirementError) Error() string {
	if len(e.MissingKeys) == 0 {
		return "seed data required but not provided"
	}
	return fmt.Sprintf("seed data required; missing %s", strings.Join(e.MissingKeys, ", "))
}

func validateSeedRequirements(
	flow *basworkflows.WorkflowDefinitionV2,
	store map[string]any,
	params map[string]any,
	env map[string]any,
) error {
	if flow == nil {
		return nil
	}

	labels := flow.GetMetadata().GetLabels()
	if !seedRequired(labels) {
		return nil
	}

	if seedApplied(store, params, env) {
		return nil
	}

	required := parseSeedKeys(labels[seedKeysLabel])
	if len(required) == 0 {
		required = []string{"projectId", "workflowId"}
	}

	missing := requiredKeysMissing(required, store, params, env)
	if len(missing) == 0 {
		return nil
	}

	return &SeedRequirementError{MissingKeys: missing}
}

func seedRequired(labels map[string]string) bool {
	if labels == nil {
		return false
	}
	raw := strings.TrimSpace(strings.ToLower(labels[seedRequiredLabel]))
	return raw == "true" || raw == "1" || raw == "yes" || raw == "required"
}

func seedApplied(store map[string]any, params map[string]any, env map[string]any) bool {
	return truthySeedFlag(env) || truthySeedFlag(params) || truthySeedFlag(store)
}

func truthySeedFlag(source map[string]any) bool {
	if source == nil {
		return false
	}
	val, ok := source[seedAppliedKey]
	if !ok {
		return false
	}
	switch typed := val.(type) {
	case bool:
		return typed
	case string:
		raw := strings.TrimSpace(strings.ToLower(typed))
		return raw == "true" || raw == "1" || raw == "yes"
	default:
		return false
	}
}

func parseSeedKeys(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	keys := make([]string, 0, len(parts))
	for _, part := range parts {
		key := strings.TrimSpace(part)
		if key == "" {
			continue
		}
		keys = append(keys, key)
	}
	return keys
}

func requiredKeysMissing(required []string, store map[string]any, params map[string]any, env map[string]any) []string {
	var missing []string
	for _, key := range required {
		if key == "" {
			continue
		}
		if hasKey(params, key) || hasKey(env, key) || hasKey(store, key) {
			continue
		}
		missing = append(missing, key)
	}
	return missing
}

func hasKey(source map[string]any, key string) bool {
	if source == nil {
		return false
	}
	_, ok := source[key]
	return ok
}

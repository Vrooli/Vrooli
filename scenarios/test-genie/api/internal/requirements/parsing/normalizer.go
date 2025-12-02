package parsing

import (
	"strings"

	"test-genie/internal/requirements/types"
)

// Normalizer normalizes requirement data to canonical forms.
type Normalizer struct {
	// Configuration options can be added here
}

// NewNormalizer creates a new Normalizer.
func NewNormalizer() *Normalizer {
	return &Normalizer{}
}

// NormalizeModule normalizes all fields in a requirement module.
func (n *Normalizer) NormalizeModule(module *types.RequirementModule) {
	if module == nil {
		return
	}

	// Normalize metadata
	n.normalizeMetadata(&module.Metadata)

	// Normalize each requirement
	for i := range module.Requirements {
		n.NormalizeRequirement(&module.Requirements[i])
	}
}

// normalizeMetadata normalizes module metadata.
func (n *Normalizer) normalizeMetadata(meta *types.ModuleMetadata) {
	if meta == nil {
		return
	}

	meta.Module = strings.TrimSpace(meta.Module)
	meta.ModuleName = strings.TrimSpace(meta.ModuleName)
	meta.Description = strings.TrimSpace(meta.Description)
	meta.PRDRef = strings.TrimSpace(meta.PRDRef)
	meta.Priority = strings.TrimSpace(meta.Priority)
	meta.SchemaVersion = strings.TrimSpace(meta.SchemaVersion)
}

// NormalizeRequirement normalizes a single requirement.
func (n *Normalizer) NormalizeRequirement(req *types.Requirement) {
	if req == nil {
		return
	}

	// Trim string fields
	req.ID = strings.TrimSpace(req.ID)
	req.Title = strings.TrimSpace(req.Title)
	req.PRDRef = strings.TrimSpace(req.PRDRef)
	req.Category = strings.TrimSpace(req.Category)
	req.Description = strings.TrimSpace(req.Description)

	// Normalize status
	req.Status = types.NormalizeDeclaredStatus(string(req.Status))

	// Normalize criticality
	if req.Criticality != "" {
		req.Criticality = types.NormalizeCriticality(string(req.Criticality))
	}

	// Normalize array fields
	req.Tags = normalizeStringSlice(req.Tags)
	req.Children = normalizeStringSlice(req.Children)
	req.DependsOn = normalizeStringSlice(req.DependsOn)
	req.Blocks = normalizeStringSlice(req.Blocks)

	// Normalize validations
	for i := range req.Validations {
		n.NormalizeValidation(&req.Validations[i])
	}
}

// NormalizeValidation normalizes a single validation.
func (n *Normalizer) NormalizeValidation(val *types.Validation) {
	if val == nil {
		return
	}

	// Trim string fields
	val.Ref = strings.TrimSpace(val.Ref)
	val.WorkflowID = strings.TrimSpace(val.WorkflowID)
	val.Phase = strings.TrimSpace(val.Phase)
	val.Notes = strings.TrimSpace(val.Notes)
	val.Scenario = strings.TrimSpace(val.Scenario)
	val.Folder = strings.TrimSpace(val.Folder)

	// Normalize type
	val.Type = types.NormalizeValidationType(string(val.Type))

	// Normalize status
	val.Status = types.NormalizeValidationStatus(string(val.Status))

	// Normalize phase name
	val.Phase = normalizePhase(val.Phase)
}

// normalizeStringSlice trims whitespace and removes empty strings.
func normalizeStringSlice(slice []string) []string {
	if len(slice) == 0 {
		return nil
	}

	result := make([]string, 0, len(slice))
	for _, s := range slice {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// normalizePhase normalizes a phase name to lowercase.
func normalizePhase(phase string) string {
	normalized := strings.ToLower(strings.TrimSpace(phase))

	// Map common aliases
	aliases := map[string]string{
		"unit-test":        "unit",
		"unit_test":        "unit",
		"unittest":         "unit",
		"integration-test": "integration",
		"integration_test": "integration",
		"integrationtest":  "integration",
		"e2e":              "integration",
		"business-logic":   "business",
		"business_logic":   "business",
		"struct":           "structure",
		"deps":             "dependencies",
		"perf":             "performance",
		"playbook":         "playbooks",
	}

	if mapped, ok := aliases[normalized]; ok {
		return mapped
	}

	return normalized
}

// NormalizeID normalizes a requirement ID for consistent comparison.
func NormalizeID(id string) string {
	return strings.ToUpper(strings.TrimSpace(id))
}

// NormalizeRef normalizes a validation reference path.
func NormalizeRef(ref string) string {
	trimmed := strings.TrimSpace(ref)
	// Normalize path separators
	return strings.ReplaceAll(trimmed, "\\", "/")
}

// ExtractPhaseFromRef attempts to infer the phase from a validation ref path.
func ExtractPhaseFromRef(ref string) string {
	normalized := strings.ToLower(ref)

	// Test file patterns
	if strings.HasSuffix(normalized, ".test.ts") ||
		strings.HasSuffix(normalized, ".test.tsx") ||
		strings.HasSuffix(normalized, ".test.js") ||
		strings.HasSuffix(normalized, "_test.go") ||
		strings.HasSuffix(normalized, ".spec.ts") {
		return "unit"
	}

	// Integration test patterns
	if strings.Contains(normalized, "integration") ||
		strings.Contains(normalized, "_integration_") ||
		strings.HasSuffix(normalized, "_integration_test.go") {
		return "integration"
	}

	// Playbook patterns
	if strings.Contains(normalized, "playbook") ||
		strings.Contains(normalized, "test/playbooks/") {
		return "playbooks"
	}

	// BATS test patterns
	if strings.HasSuffix(normalized, ".bats") {
		// Could be unit or integration depending on location
		if strings.Contains(normalized, "integration") {
			return "integration"
		}
		return "business"
	}

	return ""
}

// ExtractTypeFromRef attempts to infer the validation type from a ref path.
func ExtractTypeFromRef(ref string) types.ValidationType {
	normalized := strings.ToLower(ref)

	// Test file patterns
	if strings.HasSuffix(normalized, ".test.ts") ||
		strings.HasSuffix(normalized, ".test.tsx") ||
		strings.HasSuffix(normalized, ".test.js") ||
		strings.HasSuffix(normalized, "_test.go") ||
		strings.HasSuffix(normalized, ".spec.ts") ||
		strings.HasSuffix(normalized, ".bats") {
		return types.ValTypeTest
	}

	// Playbook/workflow patterns
	if strings.Contains(normalized, "playbook") ||
		strings.Contains(normalized, "workflow") ||
		strings.HasSuffix(normalized, ".yaml") ||
		strings.HasSuffix(normalized, ".yml") {
		return types.ValTypeAutomation
	}

	// Default to test type
	return types.ValTypeTest
}

// DeduplicateValidations removes duplicate validations based on ref/workflow_id.
func DeduplicateValidations(validations []types.Validation) []types.Validation {
	if len(validations) <= 1 {
		return validations
	}

	seen := make(map[string]bool)
	result := make([]types.Validation, 0, len(validations))

	for _, v := range validations {
		key := v.Key()
		if key == "" {
			// Keep validations without a unique key
			result = append(result, v)
			continue
		}

		if !seen[key] {
			seen[key] = true
			result = append(result, v)
		}
	}

	return result
}

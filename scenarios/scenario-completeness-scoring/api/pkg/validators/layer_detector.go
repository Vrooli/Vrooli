package validators

import (
	"regexp"
	"strings"
)

// ValidationLayerPatterns defines patterns for each validation layer
// Note: Patterns should NOT use $ anchors to allow ::function_name suffixes
// in requirement refs (e.g., "api/foo_test.go::TestBar")
var ValidationLayerPatterns = map[string]struct {
	Patterns    []*regexp.Regexp
	Description string
}{
	"API": {
		Patterns: []*regexp.Regexp{
			// Match Go test files, allowing optional ::TestName suffix
			regexp.MustCompile(`(?i)^/?api/.*_test\.go(::.*)?$`),
			regexp.MustCompile(`(?i)^/?api/.*/tests/`),
		},
		Description: "API unit tests (Go)",
	},
	"UI": {
		Patterns: []*regexp.Regexp{
			// Match UI test files, allowing optional ::TestName suffix
			regexp.MustCompile(`(?i)^/?ui/src/.*\.test\.(ts|tsx|js|jsx)(::.*)?$`),
		},
		Description: "UI unit tests (Vitest/Jest)",
	},
	"E2E": {
		Patterns: []*regexp.Regexp{
			// Match BAS playbooks (JSON/YAML) and BATS test files
			regexp.MustCompile(`(?i)^/?bas/cases/.*\.(json|yaml)(::.*)?$`),
			regexp.MustCompile(`(?i)^/?bas/cases/.*\.bats(::.*)?$`),
		},
		Description: "End-to-end automation (BAS playbooks and BATS tests)",
	},
}

// DeriveRequirementCriticality derives criticality from PRD reference
func DeriveRequirementCriticality(req Requirement) string {
	prdRef := req.PRDRef
	if prdRef == "" {
		prdRef = req.OperationalTargetID
	}

	pattern := regexp.MustCompile(`(?i)OT-([Pp][0-2])-\d{3}`)
	match := pattern.FindStringSubmatch(prdRef)

	if len(match) >= 2 {
		return strings.ToUpper(match[1])
	}

	// Check priority field
	if req.Priority != "" {
		return strings.ToUpper(req.Priority)
	}

	// Default to P2 if no reference
	return "P2"
}

// DetectValidationLayersBasic detects which layers a requirement is tested in (without quality checks)
func DetectValidationLayersBasic(req Requirement) map[string]bool {
	layers := make(map[string]bool)

	for _, v := range req.Validation {
		ref := strings.ToLower(v.Ref)

		// Check API layer
		for _, pattern := range ValidationLayerPatterns["API"].Patterns {
			if pattern.MatchString(ref) {
				layers["API"] = true
				break
			}
		}

		// Check UI layer
		for _, pattern := range ValidationLayerPatterns["UI"].Patterns {
			if pattern.MatchString(ref) {
				layers["UI"] = true
				break
			}
		}

		// Check E2E layer
		for _, pattern := range ValidationLayerPatterns["E2E"].Patterns {
			if pattern.MatchString(ref) {
				layers["E2E"] = true
				break
			}
		}

		// Check manual layer
		if v.Type == "manual" {
			layers["MANUAL"] = true
		}
	}

	return layers
}

// DetectValidationLayers detects validation layers with quality filtering
func DetectValidationLayers(req Requirement, scenarioRoot string) ValidationLayerAnalysis {
	automatedLayers := make(map[string]bool)
	hasManual := false

	for _, v := range req.Validation {
		refOriginal := v.Ref
		ref := strings.ToLower(refOriginal)

		// Track manual validations separately
		if v.Type == "manual" {
			hasManual = true
			continue
		}

		// Skip empty refs
		if refOriginal == "" && v.WorkflowID == "" {
			continue
		}

			// Reject unsupported test/ directories
			if strings.HasPrefix(refOriginal, "test/") {
				continue
			}

		// Quality check for test type validations
		if v.Type == "test" && refOriginal != "" {
			// Strip ::function_name suffix for file-based checks
			refForFileCheck := refOriginal
			if idx := strings.Index(refOriginal, "::"); idx > 0 {
				refForFileCheck = refOriginal[:idx]
			}

			isActualTestFile := strings.HasSuffix(refForFileCheck, "_test.go") ||
				strings.Contains(refForFileCheck, "/tests/") ||
				regexp.MustCompile(`\.test\.(ts|tsx|js|jsx)$`).MatchString(refForFileCheck) ||
				strings.HasSuffix(refForFileCheck, ".bats")

			if isActualTestFile {
				quality := AnalyzeTestFileQuality(refForFileCheck, scenarioRoot)
				if !quality.IsMeaningful {
					continue
				}
			} else {
				continue
			}
		}

		// Quality check for automation type validations (playbooks)
		if v.Type == "automation" && refOriginal != "" {
			// Strip ::function_name suffix for file-based checks
			refForFileCheck := refOriginal
			if idx := strings.Index(refOriginal, "::"); idx > 0 {
				refForFileCheck = refOriginal[:idx]
			}

			if regexp.MustCompile(`\.(json|yaml|bats)$`).MatchString(refForFileCheck) {
				quality := AnalyzePlaybookQuality(refForFileCheck, scenarioRoot)
				if !quality.IsMeaningful {
					continue
				}
			}
		}

		// Check API layer
		for _, pattern := range ValidationLayerPatterns["API"].Patterns {
			if pattern.MatchString(ref) {
				automatedLayers["API"] = true
				break
			}
		}

		// Check UI layer
		for _, pattern := range ValidationLayerPatterns["UI"].Patterns {
			if pattern.MatchString(ref) {
				automatedLayers["UI"] = true
				break
			}
		}

		// Check E2E layer
		for _, pattern := range ValidationLayerPatterns["E2E"].Patterns {
			if pattern.MatchString(ref) {
				automatedLayers["E2E"] = true
				break
			}
		}
	}

	// Convert map to slice
	var layers []string
	for layer := range automatedLayers {
		layers = append(layers, layer)
	}

	return ValidationLayerAnalysis{
		AutomatedLayers: layers,
		HasManual:       hasManual,
	}
}

// CheckDiversityRequirement checks if a requirement meets diversity requirements
func CheckDiversityRequirement(req Requirement, applicableLayers map[string]bool, criticality string) (bool, []string, int) {
	minLayers := 1
	if criticality == "P0" || criticality == "P1" {
		minLayers = 2
	}

	layers := DetectValidationLayersBasic(req)

	// Filter to applicable layers only
	var applicable []string
	for layer := range layers {
		if applicableLayers[layer] && layer != "MANUAL" {
			applicable = append(applicable, layer)
		}
	}

	return len(applicable) >= minLayers, applicable, minLayers
}

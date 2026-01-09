package requirements

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"test-genie/internal/requirements/refresolver"
	"test-genie/internal/requirements/types"
)

// PhaseInspectResult mirrors the legacy phase-inspect JSON structure.
type PhaseInspectResult struct {
	Scenario          string                    `json:"scenario,omitempty"`
	Phase             string                    `json:"phase"`
	Requirements      []PhaseInspectRequirement `json:"requirements"`
	MissingReferences []string                  `json:"missingReferences,omitempty"`
	// MissingDetails provides detailed information about missing refs with suggestions.
	// This helps agents and developers fix broken validation references.
	MissingDetails []MissingReferenceDetail `json:"missingDetails,omitempty"`
}

// MissingReferenceDetail provides detailed information about a missing reference.
type MissingReferenceDetail struct {
	// Ref is the original reference string.
	Ref string `json:"ref"`
	// FilePath is the file path component (before ::).
	FilePath string `json:"filePath"`
	// TestFunc is the test function name (after ::), if specified.
	TestFunc string `json:"testFunc,omitempty"`
	// Issue describes what's wrong.
	Issue string `json:"issue"`
	// SearchedPath is the absolute path that was checked.
	SearchedPath string `json:"searchedPath"`
	// Suggestion contains fix guidance.
	Suggestion *RefSuggestion `json:"suggestion,omitempty"`
}

// RefSuggestion provides actionable guidance for fixing a broken ref.
type RefSuggestion struct {
	// FoundFile is the path where a matching file was found.
	FoundFile string `json:"foundFile,omitempty"`
	// CorrectRef is the suggested corrected ref string.
	CorrectRef string `json:"correctRef,omitempty"`
	// FunctionExists indicates if the test function was found in the suggested file.
	FunctionExists bool `json:"functionExists,omitempty"`
	// AvailableFunctions lists test functions found in the file.
	AvailableFunctions []string `json:"availableFunctions,omitempty"`
	// SimilarFunctions lists functions with similar names.
	SimilarFunctions []string `json:"similarFunctions,omitempty"`
	// Hint is a human-readable explanation.
	Hint string `json:"hint"`
}

// PhaseInspectRequirement contains validations for a single requirement.
type PhaseInspectRequirement struct {
	ID          string                   `json:"id"`
	Criticality string                   `json:"criticality,omitempty"`
	Validations []PhaseInspectValidation `json:"validations,omitempty"`
}

// PhaseInspectValidation mirrors the legacy validation fields.
type PhaseInspectValidation struct {
	Type       string         `json:"type"`
	Ref        string         `json:"ref,omitempty"`
	WorkflowID string         `json:"workflow_id,omitempty"`
	Reference  string         `json:"reference,omitempty"`
	Status     string         `json:"status,omitempty"`
	Exists     bool           `json:"exists"`
	Scenario   string         `json:"scenario,omitempty"`
	Folder     string         `json:"folder,omitempty"`
	Notes      string         `json:"notes,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	// Suggestion provides fix guidance when Exists is false.
	Suggestion *RefSuggestion `json:"suggestion,omitempty"`
}

// PhaseInspect returns validations for a specific phase, mirroring the legacy Node output.
func (s *Service) PhaseInspect(ctx context.Context, scenarioDir, phase string) (*PhaseInspectResult, error) {
	if strings.TrimSpace(phase) == "" {
		return nil, errors.New("phase is required")
	}

	files, err := s.discoverer.Discover(ctx, scenarioDir)
	if err != nil {
		return nil, fmt.Errorf("discovery: %w", err)
	}
	if len(files) == 0 {
		return &PhaseInspectResult{Phase: strings.ToLower(strings.TrimSpace(phase))}, nil
	}

	index, err := s.parser.ParseAll(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("parsing: %w", err)
	}

	normalizedPhase := strings.ToLower(strings.TrimSpace(phase))
	result := &PhaseInspectResult{Phase: normalizedPhase}

	// Create resolver for smart path suggestions
	resolver := refresolver.NewResolver(scenarioDir)

	// Track missing refs with details
	missingRefs := make(map[string]MissingReferenceDetail)

	for _, module := range index.Modules {
		for _, req := range module.Requirements {
			validations := collectPhaseValidations(&req, scenarioDir, normalizedPhase, resolver, missingRefs)
			if len(validations) == 0 {
				continue
			}
			result.Requirements = append(result.Requirements, PhaseInspectRequirement{
				ID:          req.ID,
				Criticality: string(req.Criticality),
				Validations: validations,
			})
		}
	}

	// Populate missing references (legacy format for backwards compatibility)
	if len(missingRefs) > 0 {
		result.MissingReferences = make([]string, 0, len(missingRefs))
		result.MissingDetails = make([]MissingReferenceDetail, 0, len(missingRefs))
		for ref, detail := range missingRefs {
			result.MissingReferences = append(result.MissingReferences, ref)
			result.MissingDetails = append(result.MissingDetails, detail)
		}
	}

	return result, nil
}

func collectPhaseValidations(req *types.Requirement, scenarioDir, phase string, resolver *refresolver.Resolver, missingRefs map[string]MissingReferenceDetail) []PhaseInspectValidation {
	if req == nil {
		return nil
	}

	var out []PhaseInspectValidation
	for _, val := range req.Validations {
		if !matchesPhase(val, phase) {
			continue
		}

		validation := PhaseInspectValidation{
			Type:       string(val.Type),
			Ref:        val.Ref,
			WorkflowID: val.WorkflowID,
			Reference:  firstNonEmpty(val.Ref, val.WorkflowID),
			Status:     string(val.Status),
			Exists:     true, // Assume exists unless proven otherwise
			Scenario:   val.Scenario,
			Folder:     val.Folder,
			Notes:      val.Notes,
			Metadata:   val.Metadata,
		}

		// Resolve the reference if present
		if val.Ref != "" {
			resolution := resolver.Resolve(val.Ref)

			// Check if file exists (considering the :: separator)
			validation.Exists = resolution.FileExists
			if resolution.Ref.TestFunc != "" {
				// If function was specified, both file and function must exist
				validation.Exists = resolution.FileExists && resolution.FunctionExists
			}

			// Add suggestion if there's an issue
			if !validation.Exists && resolution.Suggestion != nil {
				validation.Suggestion = convertSuggestion(resolution.Suggestion)

				// Track missing ref with details
				if _, exists := missingRefs[val.Ref]; !exists {
					missingRefs[val.Ref] = MissingReferenceDetail{
						Ref:          val.Ref,
						FilePath:     resolution.Ref.FilePath,
						TestFunc:     resolution.Ref.TestFunc,
						Issue:        string(resolution.Issue),
						SearchedPath: resolution.SearchedPath,
						Suggestion:   validation.Suggestion,
					}
				}
			}
		}

		out = append(out, validation)
	}

	return out
}

// convertSuggestion converts a refresolver.Suggestion to a RefSuggestion.
func convertSuggestion(s *refresolver.Suggestion) *RefSuggestion {
	if s == nil {
		return nil
	}
	return &RefSuggestion{
		FoundFile:          s.FoundFile,
		CorrectRef:         s.CorrectRef,
		AvailableFunctions: s.AvailableFunctions,
		SimilarFunctions:   s.SimilarFunctions,
		Hint:               s.Hint,
	}
}

func matchesPhase(val types.Validation, phase string) bool {
	explicit := strings.ToLower(strings.TrimSpace(val.Phase))
	if explicit != "" && explicit == phase {
		return true
	}

	switch detectPhaseFromValidation(val) {
	case phase:
		return true
	}

	ref := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(val.Ref), "\\", "/"))
	if ref != "" && ref == fmt.Sprintf("coverage/phases/test-%s.sh", phase) {
		return true
	}

	return false
}

func detectPhaseFromValidation(val types.Validation) string {
	ref := strings.ToLower(strings.TrimSpace(val.Ref))
	ref = strings.ReplaceAll(ref, "\\", "/")
	workflow := strings.ToLower(strings.TrimSpace(val.WorkflowID))

	if val.Type == types.ValTypeTest {
		if strings.HasPrefix(ref, "ui/src/") && (strings.HasSuffix(ref, ".test.ts") || strings.HasSuffix(ref, ".test.tsx") || strings.HasSuffix(ref, ".test.js") || strings.HasSuffix(ref, ".test.jsx")) {
			return "unit"
		}
		if strings.HasSuffix(ref, "_test.go") || strings.Contains(ref, "/tests/") {
			return "unit"
		}
	}

	if val.Type == types.ValTypeAutomation {
		if ref != "" {
			return strings.TrimSuffix(filepath.Base(ref), filepath.Ext(ref))
		}
		if workflow != "" {
			return workflow
		}
	}

	if val.Type == types.ValTypeManual {
		return "manual"
	}

	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

package requirements

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"test-genie/internal/requirements/types"
)

// PhaseInspectResult mirrors the legacy phase-inspect JSON structure.
type PhaseInspectResult struct {
	Scenario          string                    `json:"scenario,omitempty"`
	Phase             string                    `json:"phase"`
	Requirements      []PhaseInspectRequirement `json:"requirements"`
	MissingReferences []string                  `json:"missingReferences,omitempty"`
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
	missing := make(map[string]struct{})

	for _, module := range index.Modules {
		for _, req := range module.Requirements {
			validations := collectPhaseValidations(&req, scenarioDir, normalizedPhase, s.reader, missing)
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

	if len(missing) > 0 {
		result.MissingReferences = make([]string, 0, len(missing))
		for ref := range missing {
			result.MissingReferences = append(result.MissingReferences, ref)
		}
	}

	return result, nil
}

func collectPhaseValidations(req *types.Requirement, scenarioDir, phase string, reader Reader, missing map[string]struct{}) []PhaseInspectValidation {
	if req == nil {
		return nil
	}

	var out []PhaseInspectValidation
	for _, val := range req.Validations {
		if !matchesPhase(val, phase) {
			continue
		}

		refExists := true
		if val.Ref != "" {
			resolved := filepath.Join(scenarioDir, filepath.FromSlash(val.Ref))
			if _, err := reader.Stat(resolved); err != nil {
				refExists = false
				missing[val.Ref] = struct{}{}
			}
		}

		out = append(out, PhaseInspectValidation{
			Type:       string(val.Type),
			Ref:        val.Ref,
			WorkflowID: val.WorkflowID,
			Reference:  firstNonEmpty(val.Ref, val.WorkflowID),
			Status:     string(val.Status),
			Exists:     refExists,
			Scenario:   val.Scenario,
			Folder:     val.Folder,
			Notes:      val.Notes,
			Metadata:   val.Metadata,
		})
	}

	return out
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
	if ref != "" && ref == fmt.Sprintf("test/phases/test-%s.sh", phase) {
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

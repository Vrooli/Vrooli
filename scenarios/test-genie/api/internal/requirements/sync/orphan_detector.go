package sync

import (
	"context"
	"path/filepath"

	"test-genie/internal/requirements/parsing"
	"test-genie/internal/requirements/types"
)

// OrphanDetector detects orphaned validations.
type OrphanDetector struct {
	reader Reader
}

// NewOrphanDetector creates a new OrphanDetector.
func NewOrphanDetector(reader Reader) *OrphanDetector {
	return &OrphanDetector{reader: reader}
}

// Orphan represents an orphaned validation.
type Orphan struct {
	FilePath      string
	RequirementID string
	ValidationRef string
	Reason        string
}

// DetectOrphans finds validations that reference non-existent files.
func (d *OrphanDetector) DetectOrphans(ctx context.Context, index *parsing.ModuleIndex, scenarioRoot string) ([]Orphan, error) {
	if index == nil || scenarioRoot == "" {
		return nil, nil
	}

	var orphans []Orphan

	for _, module := range index.Modules {
		select {
		case <-ctx.Done():
			return orphans, ctx.Err()
		default:
		}

		for _, req := range module.Requirements {
			for _, val := range req.Validations {
				if orphan := d.checkValidation(val, req.ID, module.FilePath, scenarioRoot); orphan != nil {
					orphans = append(orphans, *orphan)
				}
			}
		}
	}

	return orphans, nil
}

// checkValidation checks if a validation reference is orphaned.
func (d *OrphanDetector) checkValidation(val types.Validation, reqID, modulePath, scenarioRoot string) *Orphan {
	// Only check file-based validations
	if val.Ref == "" {
		return nil
	}

	// Skip manual validations
	if val.Type == types.ValTypeManual {
		return nil
	}

	// Check if referenced file exists
	refPath := resolveRefPath(val.Ref, scenarioRoot)
	if d.reader.Exists(refPath) {
		return nil // File exists, not orphaned
	}

	return &Orphan{
		FilePath:      modulePath,
		RequirementID: reqID,
		ValidationRef: val.Ref,
		Reason:        "referenced file does not exist",
	}
}

// resolveRefPath resolves a validation ref to an absolute path.
func resolveRefPath(ref, scenarioRoot string) string {
	// If ref is already absolute, use it
	if filepath.IsAbs(ref) {
		return ref
	}

	// Try common base paths
	candidates := []string{
		filepath.Join(scenarioRoot, ref),
		filepath.Join(scenarioRoot, "api", ref),
		filepath.Join(scenarioRoot, "ui", ref),
		filepath.Join(scenarioRoot, "test", ref),
	}

	// Return the first candidate (we'll check existence in the caller)
	return candidates[0]
}

// DetectOrphanedByPhase finds validations for phases that didn't run.
func (d *OrphanDetector) DetectOrphanedByPhase(ctx context.Context, index *parsing.ModuleIndex, executedPhases []string) ([]Orphan, error) {
	if index == nil {
		return nil, nil
	}

	executedMap := make(map[string]bool)
	for _, phase := range executedPhases {
		executedMap[phase] = true
	}

	var orphans []Orphan

	for _, module := range index.Modules {
		select {
		case <-ctx.Done():
			return orphans, ctx.Err()
		default:
		}

		for _, req := range module.Requirements {
			for _, val := range req.Validations {
				if val.Phase != "" && !executedMap[val.Phase] {
					orphans = append(orphans, Orphan{
						FilePath:      module.FilePath,
						RequirementID: req.ID,
						ValidationRef: val.Ref,
						Reason:        "phase not executed: " + val.Phase,
					})
				}
			}
		}
	}

	return orphans, nil
}

// RemoveOrphan removes a validation from its requirement.
func RemoveOrphan(module *types.RequirementModule, reqID, validationRef string) bool {
	if module == nil {
		return false
	}

	for i := range module.Requirements {
		req := &module.Requirements[i]
		if req.ID != reqID {
			continue
		}

		for j := len(req.Validations) - 1; j >= 0; j-- {
			if req.Validations[j].Ref == validationRef {
				req.Validations = append(req.Validations[:j], req.Validations[j+1:]...)
				return true
			}
		}
	}

	return false
}

// IsOrphanedRef checks if a validation ref points to a non-existent file.
func IsOrphanedRef(ref, scenarioRoot string, reader Reader) bool {
	if ref == "" {
		return false
	}

	refPath := resolveRefPath(ref, scenarioRoot)
	return !reader.Exists(refPath)
}

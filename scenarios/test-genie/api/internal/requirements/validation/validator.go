// Package validation provides structural validation for requirement files.
package validation

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"test-genie/internal/requirements/parsing"
	"test-genie/internal/requirements/types"
)

// Reader abstracts file reading operations.
type Reader interface {
	ReadFile(path string) ([]byte, error)
	ReadDir(path string) ([]fs.DirEntry, error)
	Exists(path string) bool
}

// osReader implements Reader using the os package.
type osReader struct{}

func (r *osReader) ReadFile(path string) ([]byte, error)       { return os.ReadFile(path) }
func (r *osReader) ReadDir(path string) ([]fs.DirEntry, error) { return os.ReadDir(path) }
func (r *osReader) Exists(path string) bool                    { _, err := os.Stat(path); return err == nil }

// Validator validates requirement structure.
type Validator interface {
	// Validate checks all requirements for structural issues.
	Validate(ctx context.Context, index *parsing.ModuleIndex, scenarioRoot string) *types.ValidationResult
}

// validator implements Validator.
type validator struct {
	reader Reader
	rules  []Rule
}

// New creates a Validator with default rules.
func New(reader Reader) Validator {
	return &validator{
		reader: reader,
		rules:  DefaultRules(),
	}
}

// NewDefault creates a Validator using the real file system.
func NewDefault() Validator {
	return New(&osReader{})
}

// Validate checks all requirements for structural issues.
func (v *validator) Validate(ctx context.Context, index *parsing.ModuleIndex, scenarioRoot string) *types.ValidationResult {
	result := types.NewValidationResult()

	if index == nil {
		return result
	}

	// Run each rule
	for _, rule := range v.rules {
		select {
		case <-ctx.Done():
			return result
		default:
		}

		ruleCtx := RuleContext{
			Index:        index,
			ScenarioRoot: scenarioRoot,
			Reader:       v.reader,
		}

		issues := rule.Check(ctx, ruleCtx)
		result.Issues = append(result.Issues, issues...)
	}

	return result
}

// Rule represents a validation rule.
type Rule interface {
	// Name returns the rule name.
	Name() string

	// Check runs the rule and returns any issues found.
	Check(ctx context.Context, rctx RuleContext) []types.ValidationIssue
}

// RuleContext provides context for rule execution.
type RuleContext struct {
	Index        *parsing.ModuleIndex
	ScenarioRoot string
	Reader       Reader
}

// DefaultRules returns the default set of validation rules.
func DefaultRules() []Rule {
	return []Rule{
		&DuplicateIDRule{},
		&MissingIDRule{},
		&MissingTitleRule{},
		&InvalidReferenceRule{},
		&CycleDetectionRule{},
		&OrphanedChildRule{},
		&InvalidStatusRule{},
	}
}

// DuplicateIDRule checks for duplicate requirement IDs.
type DuplicateIDRule struct{}

func (r *DuplicateIDRule) Name() string { return "duplicate_id" }

func (r *DuplicateIDRule) Check(ctx context.Context, rctx RuleContext) []types.ValidationIssue {
	var issues []types.ValidationIssue
	seen := make(map[string]string) // id -> first file

	for _, module := range rctx.Index.Modules {
		for _, req := range module.Requirements {
			normalizedID := strings.ToUpper(strings.TrimSpace(req.ID))
			if firstFile, ok := seen[normalizedID]; ok {
				issues = append(issues, types.ValidationIssue{
					FilePath:      module.FilePath,
					RequirementID: req.ID,
					Field:         "id",
					Message:       "duplicate requirement ID (first seen in " + firstFile + ")",
					Severity:      types.SeverityError,
				})
			} else {
				seen[normalizedID] = module.FilePath
			}
		}
	}

	return issues
}

// MissingIDRule checks for requirements without IDs.
type MissingIDRule struct{}

func (r *MissingIDRule) Name() string { return "missing_id" }

func (r *MissingIDRule) Check(ctx context.Context, rctx RuleContext) []types.ValidationIssue {
	var issues []types.ValidationIssue

	for _, module := range rctx.Index.Modules {
		for i, req := range module.Requirements {
			if strings.TrimSpace(req.ID) == "" {
				issues = append(issues, types.ValidationIssue{
					FilePath:      module.FilePath,
					RequirementID: "",
					Field:         "id",
					Message:       "requirement at index " + string(rune('0'+i)) + " is missing ID",
					Severity:      types.SeverityError,
				})
			}
		}
	}

	return issues
}

// MissingTitleRule checks for requirements without titles.
type MissingTitleRule struct{}

func (r *MissingTitleRule) Name() string { return "missing_title" }

func (r *MissingTitleRule) Check(ctx context.Context, rctx RuleContext) []types.ValidationIssue {
	var issues []types.ValidationIssue

	for _, module := range rctx.Index.Modules {
		for _, req := range module.Requirements {
			if strings.TrimSpace(req.Title) == "" {
				issues = append(issues, types.ValidationIssue{
					FilePath:      module.FilePath,
					RequirementID: req.ID,
					Field:         "title",
					Message:       "requirement is missing title",
					Severity:      types.SeverityWarning,
				})
			}
		}
	}

	return issues
}

// InvalidReferenceRule checks for validations referencing non-existent files.
type InvalidReferenceRule struct{}

func (r *InvalidReferenceRule) Name() string { return "invalid_reference" }

// ParsedRef represents a parsed validation reference.
// Refs can be in format "file/path.go" or "file/path.go::SymbolName"
type ParsedRef struct {
	FilePath string // The file path portion (before ::)
	Symbol   string // The symbol name (after ::), empty if not specified
	Raw      string // The original raw ref string
}

// ParseRef parses a validation ref string into its components.
// Supports formats:
//   - "path/to/file.go" -> FilePath="path/to/file.go", Symbol=""
//   - "path/to/file.go::TestFunction" -> FilePath="path/to/file.go", Symbol="TestFunction"
//   - "path/to/file.bats::test name with spaces" -> FilePath="path/to/file.bats", Symbol="test name with spaces"
//
// Edge cases handled gracefully:
//   - Empty ref -> empty FilePath and Symbol
//   - "::symbol" (no file) -> FilePath="", Symbol="symbol"
//   - "file::" (empty symbol) -> FilePath="file", Symbol=""
//   - "file::sym::bol" (multiple ::) -> FilePath="file", Symbol="sym::bol"
func ParseRef(ref string) ParsedRef {
	parsed := ParsedRef{Raw: ref}

	if ref == "" {
		return parsed
	}

	// Find the first :: separator
	idx := strings.Index(ref, "::")
	if idx == -1 {
		// No separator, entire ref is the file path
		parsed.FilePath = ref
		return parsed
	}

	// Split at the first :: (allows :: in symbol names for edge cases)
	parsed.FilePath = ref[:idx]
	if idx+2 < len(ref) {
		parsed.Symbol = ref[idx+2:]
	}

	return parsed
}

func (r *InvalidReferenceRule) Check(ctx context.Context, rctx RuleContext) []types.ValidationIssue {
	var issues []types.ValidationIssue

	if rctx.ScenarioRoot == "" || rctx.Reader == nil {
		return issues
	}

	for _, module := range rctx.Index.Modules {
		for _, req := range module.Requirements {
			for _, val := range req.Validations {
				if val.Ref == "" {
					continue
				}

				// Skip manual validations
				if val.Type == types.ValTypeManual {
					continue
				}

				// Parse the ref to extract file path and optional symbol
				parsed := ParseRef(val.Ref)

				// If no file path (e.g., "::Symbol"), that's a malformed ref
				if parsed.FilePath == "" {
					issues = append(issues, types.ValidationIssue{
						FilePath:      module.FilePath,
						RequirementID: req.ID,
						Field:         "validation.ref",
						Message:       "validation ref is malformed (missing file path): " + val.Ref,
						Severity:      types.SeverityWarning,
					})
					continue
				}

				// Check if file exists using the extracted file path
				refPath := filepath.Join(rctx.ScenarioRoot, parsed.FilePath)
				if !rctx.Reader.Exists(refPath) {
					// Try alternative paths
					altPaths := []string{
						filepath.Join(rctx.ScenarioRoot, "api", parsed.FilePath),
						filepath.Join(rctx.ScenarioRoot, "ui", parsed.FilePath),
						filepath.Join(rctx.ScenarioRoot, "test", parsed.FilePath),
					}

					found := false
					for _, alt := range altPaths {
						if rctx.Reader.Exists(alt) {
							found = true
							break
						}
					}

					if !found {
						issues = append(issues, types.ValidationIssue{
							FilePath:      module.FilePath,
							RequirementID: req.ID,
							Field:         "validation.ref",
							Message:       "validation references non-existent file: " + parsed.FilePath,
							Severity:      types.SeverityWarning,
						})
					}
				}
			}
		}
	}

	return issues
}

// CycleDetectionRule checks for cycles in requirement hierarchy.
type CycleDetectionRule struct{}

func (r *CycleDetectionRule) Name() string { return "cycle_detection" }

func (r *CycleDetectionRule) Check(ctx context.Context, rctx RuleContext) []types.ValidationIssue {
	var issues []types.ValidationIssue

	cycles := rctx.Index.DetectCycles()
	for _, cycle := range cycles {
		issues = append(issues, types.ValidationIssue{
			FilePath:      "",
			RequirementID: cycle[0],
			Field:         "children",
			Message:       "cycle detected in requirement hierarchy: " + strings.Join(cycle, " -> "),
			Severity:      types.SeverityError,
		})
	}

	return issues
}

// OrphanedChildRule checks for children that reference non-existent requirements.
type OrphanedChildRule struct{}

func (r *OrphanedChildRule) Name() string { return "orphaned_child" }

func (r *OrphanedChildRule) Check(ctx context.Context, rctx RuleContext) []types.ValidationIssue {
	var issues []types.ValidationIssue

	for _, module := range rctx.Index.Modules {
		for _, req := range module.Requirements {
			for _, childID := range req.Children {
				if rctx.Index.GetRequirement(childID) == nil {
					issues = append(issues, types.ValidationIssue{
						FilePath:      module.FilePath,
						RequirementID: req.ID,
						Field:         "children",
						Message:       "references non-existent child requirement: " + childID,
						Severity:      types.SeverityError,
					})
				}
			}

			for _, depID := range req.DependsOn {
				if rctx.Index.GetRequirement(depID) == nil {
					issues = append(issues, types.ValidationIssue{
						FilePath:      module.FilePath,
						RequirementID: req.ID,
						Field:         "depends_on",
						Message:       "references non-existent dependency: " + depID,
						Severity:      types.SeverityWarning,
					})
				}
			}
		}
	}

	return issues
}

// InvalidStatusRule checks for invalid status values.
type InvalidStatusRule struct{}

func (r *InvalidStatusRule) Name() string { return "invalid_status" }

func (r *InvalidStatusRule) Check(ctx context.Context, rctx RuleContext) []types.ValidationIssue {
	var issues []types.ValidationIssue

	validStatuses := map[types.DeclaredStatus]bool{
		types.StatusPending:        true,
		types.StatusPlanned:        true,
		types.StatusInProgress:     true,
		types.StatusComplete:       true,
		types.StatusNotImplemented: true,
	}

	for _, module := range rctx.Index.Modules {
		for _, req := range module.Requirements {
			if req.Status != "" && !validStatuses[req.Status] {
				issues = append(issues, types.ValidationIssue{
					FilePath:      module.FilePath,
					RequirementID: req.ID,
					Field:         "status",
					Message:       "invalid status value: " + string(req.Status),
					Severity:      types.SeverityWarning,
				})
			}
		}
	}

	return issues
}

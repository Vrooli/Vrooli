// DOC: docs/concepts/ARCHITECTURE.md#domain-modules
// DOC: docs/internal/SEAMS.md#change-axes
//
// [REQ:REQ-P0-007] Structural Validation Engine
package validation

import (
	"development-toolchain-validator/domain/expectation"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StructuralChecker validates structural expectations against a reference scenario.
//
// [REQ:REQ-P0-007] Structural Validation Engine
type StructuralChecker struct {
	basePath string
}

// NewStructuralChecker creates a checker for the given reference path.
func NewStructuralChecker(basePath string) *StructuralChecker {
	return &StructuralChecker{basePath: basePath}
}

// ValidateExpectation checks a single structural expectation against the filesystem.
//
// [REQ:REQ-P0-007] Folder and File Validation Logic
func (c *StructuralChecker) ValidateExpectation(exp *expectation.StructuralExpectation) *ExpectationResult {
	result := &ExpectationResult{
		ExpectationID: exp.ID,
		Expectation:   exp,
		ValidatedAt:   time.Now(),
	}

	switch exp.Type {
	case expectation.TypeFolder:
		c.checkFolder(exp, result)
	case expectation.TypeFile:
		c.checkFile(exp, result)
	case expectation.TypeContentSnippet:
		c.checkContentSnippet(exp, result)
	default:
		result.Status = StatusError
		result.Message = "unknown expectation type: " + string(exp.Type)
	}

	return result
}

// ValidateAll validates multiple expectations and returns all results.
func (c *StructuralChecker) ValidateAll(expectations []*expectation.StructuralExpectation) []*ExpectationResult {
	results := make([]*ExpectationResult, 0, len(expectations))
	for _, exp := range expectations {
		results = append(results, c.ValidateExpectation(exp))
	}
	return results
}

// checkFolder validates a folder existence expectation.
func (c *StructuralChecker) checkFolder(exp *expectation.StructuralExpectation, result *ExpectationResult) {
	pattern := filepath.Join(c.basePath, exp.Pattern)

	// Handle glob patterns
	matches, err := filepath.Glob(pattern)
	if err != nil {
		result.Status = StatusError
		result.Message = "invalid glob pattern: " + err.Error()
		return
	}

	// Filter to only directories
	var dirMatches []string
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		if info.IsDir() {
			// Store path relative to base
			relPath, _ := filepath.Rel(c.basePath, match)
			dirMatches = append(dirMatches, relPath)
		}
	}

	if len(dirMatches) > 0 {
		result.Status = StatusPassed
		result.MatchedPaths = dirMatches
		result.Message = "folder(s) found"
	} else if exp.Required {
		result.Status = StatusFailed
		result.MissingPaths = []string{exp.Pattern}
		result.Message = "required folder not found"
	} else {
		result.Status = StatusSkipped
		result.Message = "optional folder not found"
	}
}

// checkFile validates a file existence expectation.
func (c *StructuralChecker) checkFile(exp *expectation.StructuralExpectation, result *ExpectationResult) {
	pattern := filepath.Join(c.basePath, exp.Pattern)

	// Handle glob patterns
	matches, err := filepath.Glob(pattern)
	if err != nil {
		result.Status = StatusError
		result.Message = "invalid glob pattern: " + err.Error()
		return
	}

	// Filter to only regular files
	var fileMatches []string
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(c.basePath, match)
			fileMatches = append(fileMatches, relPath)
		}
	}

	if len(fileMatches) > 0 {
		result.Status = StatusPassed
		result.MatchedPaths = fileMatches
		result.Message = "file(s) found"
	} else if exp.Required {
		result.Status = StatusFailed
		result.MissingPaths = []string{exp.Pattern}
		result.Message = "required file not found"
	} else {
		result.Status = StatusSkipped
		result.Message = "optional file not found"
	}
}

// checkContentSnippet validates that a file contains expected content.
//
// [REQ:REQ-P0-007a] Content Validation and Result Aggregation
func (c *StructuralChecker) checkContentSnippet(exp *expectation.StructuralExpectation, result *ExpectationResult) {
	fullPath := filepath.Join(c.basePath, exp.Pattern)

	// Check if file exists
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			if exp.Required {
				result.Status = StatusFailed
				result.MissingPaths = []string{exp.Pattern}
				result.Message = "file not found for content check"
			} else {
				result.Status = StatusSkipped
				result.Message = "optional file not found"
			}
			return
		}
		result.Status = StatusError
		result.Message = "error accessing file: " + err.Error()
		return
	}

	if info.IsDir() {
		result.Status = StatusError
		result.Message = "path is a directory, expected file"
		return
	}

	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		result.Status = StatusError
		result.Message = "error reading file: " + err.Error()
		return
	}

	// Check if content contains expected snippet
	relPath, _ := filepath.Rel(c.basePath, fullPath)
	result.MatchedPaths = []string{relPath}

	if strings.Contains(string(content), exp.ExpectedContent) {
		result.Status = StatusPassed
		result.ContentMatch = true
		result.Message = "content snippet found"
	} else if exp.Required {
		result.Status = StatusFailed
		result.ContentMatch = false
		result.Message = "required content snippet not found in file"
	} else {
		result.Status = StatusSkipped
		result.ContentMatch = false
		result.Message = "optional content snippet not found"
	}
}

// CountResults returns pass/fail/skip/error counts from a slice of results.
func CountResults(results []*ExpectationResult) (pass, fail, skip, errCount int) {
	for _, r := range results {
		switch r.Status {
		case StatusPassed:
			pass++
		case StatusFailed:
			fail++
		case StatusSkipped:
			skip++
		case StatusError:
			errCount++
		}
	}
	return
}

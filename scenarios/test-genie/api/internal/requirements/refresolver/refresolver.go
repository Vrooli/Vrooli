// Package refresolver provides functionality for parsing, validating, and resolving
// validation reference strings in requirement modules.
//
// A validation ref has the format: "path/to/file.ext::TestFunctionName"
// where the "::TestFunctionName" suffix is optional.
//
// This package helps:
// - Parse refs into file path and test function components
// - Check if referenced files exist
// - Find likely matches when files aren't found at the expected path
// - Verify test functions exist in files
// - Generate helpful suggestions for fixing broken refs
package refresolver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RefSeparator is the delimiter between file path and test function name.
const RefSeparator = "::"

// ParsedRef represents a parsed validation reference.
type ParsedRef struct {
	// Original is the original ref string before parsing.
	Original string `json:"original"`
	// FilePath is the file path component (before ::).
	FilePath string `json:"filePath"`
	// TestFunc is the test function name (after ::), empty if not specified.
	TestFunc string `json:"testFunc,omitempty"`
}

// Parse splits a validation ref into its file path and test function components.
func Parse(ref string) ParsedRef {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ParsedRef{Original: ref}
	}

	parts := strings.SplitN(ref, RefSeparator, 2)
	parsed := ParsedRef{
		Original: ref,
		FilePath: strings.TrimSpace(parts[0]),
	}
	if len(parts) > 1 {
		parsed.TestFunc = strings.TrimSpace(parts[1])
	}
	return parsed
}

// IssueType categorizes what's wrong with a ref.
type IssueType string

const (
	// IssueNone means no issue - ref is valid.
	IssueNone IssueType = ""
	// IssueFileNotFound means the file doesn't exist at the specified path.
	IssueFileNotFound IssueType = "file_not_found"
	// IssueFunctionNotFound means the file exists but the test function wasn't found.
	IssueFunctionNotFound IssueType = "function_not_found"
	// IssueEmptyRef means the ref was empty or whitespace-only.
	IssueEmptyRef IssueType = "empty_ref"
)

// Resolution contains the result of resolving a validation ref.
type Resolution struct {
	// Ref is the parsed reference.
	Ref ParsedRef `json:"ref"`
	// Issue indicates what's wrong (empty if ref is valid).
	Issue IssueType `json:"issue,omitempty"`
	// FileExists indicates whether the file was found.
	FileExists bool `json:"fileExists"`
	// FunctionExists indicates whether the test function was found in the file.
	// Only set if FileExists is true and TestFunc was specified.
	FunctionExists bool `json:"functionExists,omitempty"`
	// SearchedPath is the absolute path that was checked.
	SearchedPath string `json:"searchedPath,omitempty"`
	// Suggestion contains fix suggestions if there's an issue.
	Suggestion *Suggestion `json:"suggestion,omitempty"`
}

// IsValid returns true if the ref resolved successfully.
func (r *Resolution) IsValid() bool {
	return r.Issue == IssueNone
}

// Suggestion provides actionable guidance for fixing a broken ref.
type Suggestion struct {
	// FoundFile is the path where a matching file was found, if any.
	FoundFile string `json:"foundFile,omitempty"`
	// CorrectRef is the suggested corrected ref string.
	CorrectRef string `json:"correctRef,omitempty"`
	// AvailableFunctions lists test functions found in the file.
	// Only populated if file exists but function wasn't found.
	AvailableFunctions []string `json:"availableFunctions,omitempty"`
	// SimilarFunctions lists functions with similar names to the requested one.
	SimilarFunctions []string `json:"similarFunctions,omitempty"`
	// Hint is a human-readable explanation of the issue and fix.
	Hint string `json:"hint"`
}

// Resolver resolves validation refs to check existence and generate suggestions.
type Resolver struct {
	finder   *Finder
	detector TestFuncDetector
}

// NewResolver creates a new Resolver with the given scenario directory.
func NewResolver(scenarioDir string) *Resolver {
	return &Resolver{
		finder:   NewFinder(scenarioDir),
		detector: NewCompositeDetector(),
	}
}

// Resolve checks if a validation ref is valid and generates suggestions if not.
func (r *Resolver) Resolve(ref string) Resolution {
	parsed := Parse(ref)

	if parsed.FilePath == "" {
		return Resolution{
			Ref:   parsed,
			Issue: IssueEmptyRef,
			Suggestion: &Suggestion{
				Hint: "Validation ref is empty. Provide a file path like 'api/handlers_test.go' or 'api/handlers_test.go::TestFoo'.",
			},
		}
	}

	result := Resolution{
		Ref:          parsed,
		SearchedPath: r.finder.AbsolutePath(parsed.FilePath),
	}

	// Check if file exists at the expected path
	if r.finder.FileExists(parsed.FilePath) {
		result.FileExists = true

		// If a test function was specified, check if it exists
		if parsed.TestFunc != "" {
			absPath := r.finder.AbsolutePath(parsed.FilePath)
			funcs, err := r.detector.DetectFunctions(absPath)
			if err == nil {
				result.FunctionExists = containsFunc(funcs, parsed.TestFunc)
				if !result.FunctionExists {
					result.Issue = IssueFunctionNotFound
					result.Suggestion = r.suggestFunction(parsed.TestFunc, funcs)
				}
			}
		}
		return result
	}

	// File not found - try to find it
	result.Issue = IssueFileNotFound
	result.Suggestion = r.suggestFile(parsed)
	return result
}

// suggestFile generates suggestions when a file isn't found.
func (r *Resolver) suggestFile(parsed ParsedRef) *Suggestion {
	basename := filepath.Base(parsed.FilePath)
	matches := r.finder.FindByBasename(basename)

	if len(matches) == 0 {
		return &Suggestion{
			Hint: fmt.Sprintf("File '%s' not found. No files matching '%s' were found in the scenario directory.", parsed.FilePath, basename),
		}
	}

	// Find the best match by path similarity
	bestMatch := r.finder.BestMatch(parsed.FilePath, matches)

	suggestion := &Suggestion{
		FoundFile: bestMatch,
		Hint:      fmt.Sprintf("File not found at '%s'. Found similar file at '%s'.", parsed.FilePath, bestMatch),
	}

	// Generate corrected ref
	if parsed.TestFunc != "" {
		suggestion.CorrectRef = bestMatch + RefSeparator + parsed.TestFunc

		// Check if function exists in the found file
		absPath := r.finder.AbsolutePath(bestMatch)
		funcs, err := r.detector.DetectFunctions(absPath)
		if err == nil {
			if containsFunc(funcs, parsed.TestFunc) {
				suggestion.Hint = fmt.Sprintf("File exists at '%s' (not '%s'). Function '%s' found. Update the ref to: %s",
					bestMatch, parsed.FilePath, parsed.TestFunc, suggestion.CorrectRef)
			} else {
				suggestion.AvailableFunctions = funcs
				suggestion.SimilarFunctions = findSimilar(parsed.TestFunc, funcs, 3)
				if len(suggestion.SimilarFunctions) > 0 {
					suggestion.Hint = fmt.Sprintf("File exists at '%s' but function '%s' not found. Similar functions: %v",
						bestMatch, parsed.TestFunc, suggestion.SimilarFunctions)
				}
			}
		}
	} else {
		suggestion.CorrectRef = bestMatch
	}

	return suggestion
}

// suggestFunction generates suggestions when a function isn't found in a file.
func (r *Resolver) suggestFunction(funcName string, availableFuncs []string) *Suggestion {
	suggestion := &Suggestion{
		AvailableFunctions: availableFuncs,
		SimilarFunctions:   findSimilar(funcName, availableFuncs, 3),
	}

	if len(suggestion.SimilarFunctions) > 0 {
		suggestion.Hint = fmt.Sprintf("Function '%s' not found. Did you mean: %v?", funcName, suggestion.SimilarFunctions)
	} else if len(availableFuncs) > 0 {
		preview := availableFuncs
		if len(preview) > 5 {
			preview = preview[:5]
		}
		suggestion.Hint = fmt.Sprintf("Function '%s' not found. Available functions: %v...", funcName, preview)
	} else {
		suggestion.Hint = fmt.Sprintf("Function '%s' not found and no test functions detected in file.", funcName)
	}

	return suggestion
}

// containsFunc checks if funcName is in the list.
func containsFunc(funcs []string, funcName string) bool {
	for _, f := range funcs {
		if f == funcName {
			return true
		}
	}
	return false
}

// findSimilar returns functions with names similar to target.
func findSimilar(target string, funcs []string, max int) []string {
	target = strings.ToLower(target)
	var similar []string

	for _, f := range funcs {
		lower := strings.ToLower(f)
		// Simple similarity: contains or is contained
		if strings.Contains(lower, target) || strings.Contains(target, lower) {
			similar = append(similar, f)
			if len(similar) >= max {
				break
			}
			continue
		}
		// Check for common prefix
		minLen := len(target)
		if len(lower) < minLen {
			minLen = len(lower)
		}
		commonPrefix := 0
		for i := 0; i < minLen; i++ {
			if target[i] == lower[i] {
				commonPrefix++
			} else {
				break
			}
		}
		if commonPrefix >= minLen/2 && commonPrefix >= 4 {
			similar = append(similar, f)
			if len(similar) >= max {
				break
			}
		}
	}

	return similar
}

// Reader is an interface for file system operations (for testing).
type Reader interface {
	Stat(path string) (os.FileInfo, error)
	ReadFile(path string) ([]byte, error)
	Glob(pattern string) ([]string, error)
}

// OSReader implements Reader using the real file system.
type OSReader struct{}

func (OSReader) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func (OSReader) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (OSReader) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}

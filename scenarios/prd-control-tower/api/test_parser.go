package main

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Pattern to match requirement IDs in test comments: [REQ:ID] or // REQ:ID or /* REQ:ID */
var testReqPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\[REQ:([A-Z]+-[A-Z]+-[A-Z0-9]+)\]`),                  // [REQ:BAS-EXEC-TELEMETRY-STREAM]
	regexp.MustCompile(`//\s*REQ:([A-Z]+-[A-Z]+-[A-Z0-9]+)`),                 // // REQ:BAS-EXEC-TELEMETRY-STREAM
	regexp.MustCompile(`/\*\s*REQ:([A-Z]+-[A-Z]+-[A-Z0-9]+)\s*\*/`),          // /* REQ:BAS-EXEC-TELEMETRY-STREAM */
	regexp.MustCompile(`it\(["'].*?\[REQ:([A-Z]+-[A-Z]+-[A-Z0-9]+)\]`),       // it("test [REQ:ID]"
	regexp.MustCompile(`describe\(["'].*?\[REQ:([A-Z]+-[A-Z]+-[A-Z0-9]+)\]`), // describe("suite [REQ:ID]"
	regexp.MustCompile(`t\.Run\(["'].*?\[REQ:([A-Z]+-[A-Z]+-[A-Z0-9]+)\]`),   // t.Run("test [REQ:ID]"
}

// TestFileReference represents a test file that references a requirement
type TestFileReference struct {
	FilePath      string   `json:"file_path"`
	RequirementID string   `json:"requirement_id"`
	Lines         []int    `json:"lines"`      // Line numbers where requirement is mentioned
	TestNames     []string `json:"test_names"` // Names of test functions/cases
}

// scanTestFilesForRequirement scans test files in an entity's directory for requirement ID references
func scanTestFilesForRequirement(entityType, entityName, requirementID string) ([]TestFileReference, error) {
	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		return nil, err
	}

	entityDir := filepath.Join(vrooliRoot, entityType+"s", entityName)
	var references []TestFileReference

	// Common test file patterns
	testPatterns := []string{
		"**/*_test.go",
		"**/*.test.ts",
		"**/*.test.tsx",
		"**/*.test.js",
		"**/*.test.jsx",
		"**/test*.go",
		"**/test*.ts",
		"**/test*.sh",
		"test/**/*.sh",
		"test/**/*.bats",
	}

	for _, pattern := range testPatterns {
		matches, err := filepath.Glob(filepath.Join(entityDir, pattern))
		if err != nil {
			continue
		}

		for _, match := range matches {
			refs := scanFileForRequirement(match, requirementID)
			if len(refs.Lines) > 0 {
				references = append(references, refs)
			}
		}
	}

	// Also scan subdirectories for nested test files
	err = filepath.Walk(entityDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible files
		}
		if info.IsDir() {
			// Skip common non-test directories
			basename := filepath.Base(path)
			if basename == "node_modules" || basename == ".git" || basename == "dist" || basename == "build" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if it's a test file
		name := info.Name()
		isTestFile := strings.Contains(name, "test") || strings.Contains(name, "spec") ||
			strings.HasSuffix(name, ".bats")

		if isTestFile {
			refs := scanFileForRequirement(path, requirementID)
			if len(refs.Lines) > 0 {
				// Make path relative to entity directory for cleaner display
				relPath, err := filepath.Rel(entityDir, path)
				if err == nil {
					refs.FilePath = relPath
				}
				references = append(references, refs)
			}
		}

		return nil
	})

	if err != nil {
		return references, err
	}

	return references, nil
}

// scanFileForRequirement scans a single file for requirement ID references
func scanFileForRequirement(filePath, requirementID string) TestFileReference {
	ref := TestFileReference{
		FilePath:      filePath,
		RequirementID: requirementID,
		Lines:         []int{},
		TestNames:     []string{},
	}

	file, err := os.Open(filePath)
	if err != nil {
		return ref
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	var currentTestName string

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Check if this line mentions the requirement ID
		foundReq := false
		for _, pattern := range testReqPatterns {
			matches := pattern.FindStringSubmatch(line)
			if len(matches) > 1 && matches[1] == requirementID {
				foundReq = true
				ref.Lines = append(ref.Lines, lineNum)
				break
			}
		}

		// Also check for plain requirement ID mention (case-insensitive)
		if !foundReq && strings.Contains(strings.ToUpper(line), strings.ToUpper(requirementID)) {
			ref.Lines = append(ref.Lines, lineNum)
		}

		// Extract test function names
		if strings.Contains(line, "func Test") || strings.Contains(line, "func Benchmark") {
			// Go test: func TestFoo(t *testing.T)
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				testName := strings.TrimSuffix(parts[1], "(")
				currentTestName = testName
			}
		} else if strings.Contains(line, "it(") || strings.Contains(line, "test(") || strings.Contains(line, "describe(") {
			// JS/TS test: it("should do thing", ...
			startQuote := strings.Index(line, "\"")
			if startQuote == -1 {
				startQuote = strings.Index(line, "'")
			}
			if startQuote != -1 {
				endQuote := strings.Index(line[startQuote+1:], line[startQuote:startQuote+1])
				if endQuote != -1 {
					currentTestName = line[startQuote+1 : startQuote+1+endQuote]
				}
			}
		} else if strings.Contains(line, "t.Run(") {
			// Go subtest: t.Run("test name", func(t *testing.T) {
			startQuote := strings.Index(line, "\"")
			if startQuote != -1 {
				endQuote := strings.Index(line[startQuote+1:], "\"")
				if endQuote != -1 {
					currentTestName = line[startQuote+1 : startQuote+1+endQuote]
				}
			}
		}

		// If this line mentions the requirement and we have a current test name, record it
		if len(ref.Lines) > 0 && ref.Lines[len(ref.Lines)-1] == lineNum && currentTestName != "" {
			// Add test name if not already present
			found := false
			for _, name := range ref.TestNames {
				if name == currentTestName {
					found = true
					break
				}
			}
			if !found {
				ref.TestNames = append(ref.TestNames, currentTestName)
			}
		}
	}

	return ref
}

// getAllTestReferencesForEntity scans all test files in an entity for all requirement IDs
func getAllTestReferencesForEntity(entityType, entityName string, requirements []RequirementRecord) map[string][]TestFileReference {
	result := make(map[string][]TestFileReference)

	for _, req := range requirements {
		refs, err := scanTestFilesForRequirement(entityType, entityName, req.ID)
		if err == nil && len(refs) > 0 {
			result[req.ID] = refs
		}
	}

	return result
}

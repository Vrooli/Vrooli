package refresolver

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// TestFuncDetector detects test function names from source files.
type TestFuncDetector interface {
	// CanHandle returns true if this detector can handle the given file.
	CanHandle(filePath string) bool
	// DetectFunctions returns test function names found in the file.
	DetectFunctions(filePath string) ([]string, error)
}

// CompositeDetector combines multiple language-specific detectors.
type CompositeDetector struct {
	detectors []TestFuncDetector
}

// NewCompositeDetector creates a detector that handles Go, JS/TS, and BATS files.
func NewCompositeDetector() *CompositeDetector {
	return &CompositeDetector{
		detectors: []TestFuncDetector{
			&GoDetector{},
			&JSDetector{},
			&BATSDetector{},
			&PythonDetector{},
		},
	}
}

// CanHandle returns true if any detector can handle the file.
func (c *CompositeDetector) CanHandle(filePath string) bool {
	for _, d := range c.detectors {
		if d.CanHandle(filePath) {
			return true
		}
	}
	return false
}

// DetectFunctions delegates to the appropriate detector.
func (c *CompositeDetector) DetectFunctions(filePath string) ([]string, error) {
	for _, d := range c.detectors {
		if d.CanHandle(filePath) {
			return d.DetectFunctions(filePath)
		}
	}
	return nil, nil
}

// AddDetector adds a custom detector to the composite.
func (c *CompositeDetector) AddDetector(d TestFuncDetector) {
	c.detectors = append(c.detectors, d)
}

// GoDetector detects Go test functions.
type GoDetector struct{}

var goTestFuncPattern = regexp.MustCompile(`^func\s+(Test[A-Z]\w*)\s*\(`)

func (GoDetector) CanHandle(filePath string) bool {
	return strings.HasSuffix(filePath, "_test.go")
}

func (GoDetector) DetectFunctions(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var funcs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if matches := goTestFuncPattern.FindStringSubmatch(line); len(matches) > 1 {
			funcs = append(funcs, matches[1])
		}
	}

	return funcs, scanner.Err()
}

// JSDetector detects JavaScript/TypeScript test functions.
type JSDetector struct{}

var (
	// Matches: test("name", ...), it("name", ...), describe("name", ...)
	jsTestPattern = regexp.MustCompile(`(?:test|it|describe)\s*\(\s*["'\x60]([^"'\x60]+)["'\x60]`)
	// Matches: test.only("name", ...), it.skip("name", ...), etc.
	jsTestModPattern = regexp.MustCompile(`(?:test|it|describe)\s*\.\s*(?:only|skip|todo)\s*\(\s*["'\x60]([^"'\x60]+)["'\x60]`)
)

func (JSDetector) CanHandle(filePath string) bool {
	ext := filepath.Ext(filePath)
	base := filepath.Base(filePath)

	// Check for test file patterns
	testPatterns := []string{".test.", ".spec.", "_test.", "_spec."}
	for _, pattern := range testPatterns {
		if strings.Contains(base, pattern) {
			return ext == ".js" || ext == ".jsx" || ext == ".ts" || ext == ".tsx" || ext == ".mjs"
		}
	}
	return false
}

func (JSDetector) DetectFunctions(filePath string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	text := string(content)
	funcSet := make(map[string]struct{})

	// Find regular test/it/describe blocks
	for _, matches := range jsTestPattern.FindAllStringSubmatch(text, -1) {
		if len(matches) > 1 {
			funcSet[matches[1]] = struct{}{}
		}
	}

	// Find modified test blocks (test.only, it.skip, etc.)
	for _, matches := range jsTestModPattern.FindAllStringSubmatch(text, -1) {
		if len(matches) > 1 {
			funcSet[matches[1]] = struct{}{}
		}
	}

	var funcs []string
	for f := range funcSet {
		funcs = append(funcs, f)
	}
	return funcs, nil
}

// BATSDetector detects BATS test functions.
type BATSDetector struct{}

// Matches: @test "test name" {
var batsTestPattern = regexp.MustCompile(`@test\s+["']([^"']+)["']`)

func (BATSDetector) CanHandle(filePath string) bool {
	return strings.HasSuffix(filePath, ".bats")
}

func (BATSDetector) DetectFunctions(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var funcs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if matches := batsTestPattern.FindStringSubmatch(line); len(matches) > 1 {
			funcs = append(funcs, matches[1])
		}
	}

	return funcs, scanner.Err()
}

// PythonDetector detects Python test functions (pytest/unittest style).
type PythonDetector struct{}

var (
	// Matches: def test_something(
	pytestFuncPattern = regexp.MustCompile(`^(?:\s*)def\s+(test_\w+)\s*\(`)
	// Matches: class TestSomething(
	pytestClassPattern = regexp.MustCompile(`^(?:\s*)class\s+(Test\w+)\s*[:\(]`)
)

func (PythonDetector) CanHandle(filePath string) bool {
	base := filepath.Base(filePath)
	ext := filepath.Ext(filePath)

	if ext != ".py" {
		return false
	}

	// Check for test file patterns
	return strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py")
}

func (PythonDetector) DetectFunctions(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var funcs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if matches := pytestFuncPattern.FindStringSubmatch(line); len(matches) > 1 {
			funcs = append(funcs, matches[1])
		}
		if matches := pytestClassPattern.FindStringSubmatch(line); len(matches) > 1 {
			funcs = append(funcs, matches[1])
		}
	}

	return funcs, scanner.Err()
}

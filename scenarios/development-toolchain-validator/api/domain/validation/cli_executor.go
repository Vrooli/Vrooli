// DOC: docs/concepts/ARCHITECTURE.md#domain-modules
// DOC: docs/internal/SEAMS.md#change-axes
//
// [REQ:REQ-P0-008] Secure CLI Command Execution
// [REQ:REQ-P0-008a] JSON Parsing and Assertion Evaluation
package validation

import (
	"bytes"
	"context"
	"development-toolchain-validator/domain/expectation"
	"development-toolchain-validator/internal/validation"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Default timeout for CLI command execution.
const DefaultCommandTimeout = 30 * time.Second

// CLI execution errors.
var (
	ErrCommandTimeout    = errors.New("command timed out")
	ErrDangerousCommand  = errors.New("command contains dangerous patterns")
	ErrJSONParseFailed   = errors.New("failed to parse command output as JSON")
	ErrJSONPathNotFound  = errors.New("JSONPath did not match any value")
	ErrInvalidOperator   = errors.New("invalid assertion operator")
	ErrTypeNotComparable = errors.New("values are not comparable")
)

// CLIExecutor validates CLI assertions by executing commands and checking output.
//
// [REQ:REQ-P0-008] Secure CLI Command Execution
type CLIExecutor struct {
	workingDir string
	timeout    time.Duration
}

// ExecutorOption configures the CLIExecutor.
type ExecutorOption func(*CLIExecutor)

// WithTimeout sets a custom timeout for command execution.
func WithTimeout(d time.Duration) ExecutorOption {
	return func(e *CLIExecutor) {
		e.timeout = d
	}
}

// NewCLIExecutor creates an executor for the given working directory.
func NewCLIExecutor(workingDir string, opts ...ExecutorOption) *CLIExecutor {
	e := &CLIExecutor{
		workingDir: workingDir,
		timeout:    DefaultCommandTimeout,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// ValidateAssertion executes a CLI command and evaluates the assertion.
//
// [REQ:REQ-P0-008] Command execution with timeout controls
// [REQ:REQ-P0-008a] JSON parsing and assertion evaluation
func (e *CLIExecutor) ValidateAssertion(ctx context.Context, assertion *expectation.CLIAssertion) *AssertionResult {
	result := &AssertionResult{
		AssertionID:   assertion.ID,
		Assertion:     assertion,
		ExpectedValue: assertion.ExpectedValue,
		ValidatedAt:   time.Now(),
	}

	// Security: Check command safety before execution
	safetyResult := validation.ValidateCommandSafety(assertion.Command)
	if !safetyResult.IsSafe {
		result.Status = StatusError
		result.Message = fmt.Sprintf("dangerous command blocked: pattern '%s' detected", safetyResult.DangerousPattern)
		return result
	}

	// Execute command with timeout
	output, err := e.executeCommand(ctx, assertion.Command)
	if err != nil {
		result.Status = StatusError
		result.CommandError = err.Error()
		if errors.Is(err, context.DeadlineExceeded) {
			result.Message = "command timed out after " + e.timeout.String()
		} else {
			result.Message = "command execution failed: " + err.Error()
		}
		return result
	}
	result.CommandOutput = output

	// Parse JSON output
	var jsonData interface{}
	if err := json.Unmarshal([]byte(output), &jsonData); err != nil {
		result.Status = StatusError
		result.Message = "failed to parse JSON output: " + err.Error()
		return result
	}

	// Extract value using JSONPath
	actualValue, err := extractJSONPath(jsonData, assertion.JSONPath)
	if err != nil {
		result.Status = StatusError
		result.Message = "JSONPath extraction failed: " + err.Error()
		return result
	}
	result.ActualValue = actualValue

	// Evaluate assertion
	passed, err := evaluateAssertion(actualValue, assertion.Operator, assertion.ExpectedValue)
	if err != nil {
		result.Status = StatusError
		result.Message = "assertion evaluation failed: " + err.Error()
		return result
	}

	if passed {
		result.Status = StatusPassed
		result.Message = "assertion passed"
	} else {
		result.Status = StatusFailed
		result.Message = fmt.Sprintf("assertion failed: actual %v %s %v", actualValue, assertion.Operator, assertion.ExpectedValue)
	}

	return result
}

// ValidateAll executes multiple CLI assertions and returns all results.
func (e *CLIExecutor) ValidateAll(ctx context.Context, assertions []*expectation.CLIAssertion) []*AssertionResult {
	results := make([]*AssertionResult, 0, len(assertions))
	for _, assertion := range assertions {
		results = append(results, e.ValidateAssertion(ctx, assertion))
	}
	return results
}

// executeCommand runs a shell command with timeout.
//
// [REQ:REQ-P0-008] Secure CLI Command Execution
func (e *CLIExecutor) executeCommand(ctx context.Context, command string) (string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Execute using sh -c for proper shell parsing
	// Note: Command safety is validated before this function is called
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = e.workingDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", ErrCommandTimeout
		}
		// Include stderr in error for debugging
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%w: %s", err, stderr.String())
		}
		return "", err
	}

	return stdout.String(), nil
}

// extractJSONPath extracts a value from JSON data using a JSONPath expression.
//
// Supported formats:
//   - $ (root)
//   - $.field
//   - $.field.nested
//   - $[0]
//   - $.field[0].nested
//   - $[*] (array all elements)
//
// [REQ:REQ-P0-008a] JSON Parsing and Assertion Evaluation
func extractJSONPath(data interface{}, path string) (interface{}, error) {
	if path == "$" {
		return data, nil
	}

	// Remove leading $
	if !strings.HasPrefix(path, "$") {
		return nil, fmt.Errorf("JSONPath must start with $")
	}
	path = path[1:]

	current := data
	for len(path) > 0 {
		if strings.HasPrefix(path, ".") {
			// Field access: .fieldName
			path = path[1:] // Remove leading dot
			end := strings.IndexAny(path, ".[")
			var field string
			if end == -1 {
				field = path
				path = ""
			} else {
				field = path[:end]
				path = path[end:]
			}

			obj, ok := current.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("cannot access field '%s' on non-object", field)
			}
			val, exists := obj[field]
			if !exists {
				return nil, fmt.Errorf("field '%s' not found", field)
			}
			current = val

		} else if strings.HasPrefix(path, "[") {
			// Array access: [index] or [*]
			end := strings.Index(path, "]")
			if end == -1 {
				return nil, fmt.Errorf("unclosed bracket in path")
			}
			indexStr := path[1:end]
			path = path[end+1:]

			if indexStr == "*" {
				// Return all array elements
				arr, ok := current.([]interface{})
				if !ok {
					return nil, fmt.Errorf("cannot use [*] on non-array")
				}
				return arr, nil
			}

			index, err := strconv.Atoi(indexStr)
			if err != nil {
				return nil, fmt.Errorf("invalid array index: %s", indexStr)
			}

			arr, ok := current.([]interface{})
			if !ok {
				return nil, fmt.Errorf("cannot index non-array")
			}
			if index < 0 || index >= len(arr) {
				return nil, fmt.Errorf("array index %d out of bounds (len=%d)", index, len(arr))
			}
			current = arr[index]

		} else {
			return nil, fmt.Errorf("unexpected character in path: %s", path)
		}
	}

	return current, nil
}

// evaluateAssertion compares actual vs expected using the given operator.
//
// [REQ:REQ-P0-008a] All assertion types supported
func evaluateAssertion(actual interface{}, op expectation.AssertionOperator, expected interface{}) (bool, error) {
	switch op {
	case expectation.OpExists:
		return actual != nil, nil

	case expectation.OpEq:
		return deepEqual(actual, expected), nil

	case expectation.OpNeq:
		return !deepEqual(actual, expected), nil

	case expectation.OpContains:
		return containsValue(actual, expected)

	case expectation.OpMatches:
		return matchesRegex(actual, expected)

	case expectation.OpGt, expectation.OpGte, expectation.OpLt, expectation.OpLte:
		return compareNumeric(actual, expected, op)

	case expectation.OpBetween:
		return checkBetween(actual, expected)

	default:
		return false, ErrInvalidOperator
	}
}

// deepEqual compares two values for equality, handling JSON type quirks.
func deepEqual(a, b interface{}) bool {
	// Handle numeric comparison (JSON numbers are float64)
	aNum, aIsNum := toFloat64(a)
	bNum, bIsNum := toFloat64(b)
	if aIsNum && bIsNum {
		return aNum == bNum
	}

	return reflect.DeepEqual(a, b)
}

// containsValue checks if actual contains the expected value.
// For strings: substring check
// For arrays: element presence check
func containsValue(actual, expected interface{}) (bool, error) {
	// String contains
	if actualStr, ok := actual.(string); ok {
		expectedStr, ok := expected.(string)
		if !ok {
			return false, fmt.Errorf("expected string for contains on string")
		}
		return strings.Contains(actualStr, expectedStr), nil
	}

	// Array contains
	if actualArr, ok := actual.([]interface{}); ok {
		for _, elem := range actualArr {
			if deepEqual(elem, expected) {
				return true, nil
			}
		}
		return false, nil
	}

	return false, fmt.Errorf("contains only works on strings and arrays")
}

// matchesRegex checks if actual matches the expected regex pattern.
func matchesRegex(actual, expected interface{}) (bool, error) {
	actualStr, ok := actual.(string)
	if !ok {
		return false, fmt.Errorf("matches requires string value")
	}

	pattern, ok := expected.(string)
	if !ok {
		return false, fmt.Errorf("matches requires string pattern")
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, fmt.Errorf("invalid regex pattern: %w", err)
	}

	return re.MatchString(actualStr), nil
}

// compareNumeric performs numeric comparison.
func compareNumeric(actual, expected interface{}, op expectation.AssertionOperator) (bool, error) {
	actualNum, ok := toFloat64(actual)
	if !ok {
		return false, fmt.Errorf("actual value is not numeric: %v", actual)
	}

	expectedNum, ok := toFloat64(expected)
	if !ok {
		return false, fmt.Errorf("expected value is not numeric: %v", expected)
	}

	switch op {
	case expectation.OpGt:
		return actualNum > expectedNum, nil
	case expectation.OpGte:
		return actualNum >= expectedNum, nil
	case expectation.OpLt:
		return actualNum < expectedNum, nil
	case expectation.OpLte:
		return actualNum <= expectedNum, nil
	default:
		return false, ErrInvalidOperator
	}
}

// checkBetween checks if actual is between expected[0] and expected[1].
func checkBetween(actual, expected interface{}) (bool, error) {
	actualNum, ok := toFloat64(actual)
	if !ok {
		return false, fmt.Errorf("actual value is not numeric")
	}

	// Expected should be an array of two values [min, max]
	expectedArr, ok := expected.([]interface{})
	if !ok || len(expectedArr) != 2 {
		return false, fmt.Errorf("between requires [min, max] array")
	}

	minVal, ok := toFloat64(expectedArr[0])
	if !ok {
		return false, fmt.Errorf("min value is not numeric")
	}

	maxVal, ok := toFloat64(expectedArr[1])
	if !ok {
		return false, fmt.Errorf("max value is not numeric")
	}

	return actualNum >= minVal && actualNum <= maxVal, nil
}

// toFloat64 converts a value to float64 if possible.
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	case string:
		f, err := strconv.ParseFloat(val, 64)
		return f, err == nil
	case json.Number:
		f, err := val.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

// CountAssertionResults returns pass/fail/skip/error counts from assertion results.
func CountAssertionResults(results []*AssertionResult) (pass, fail, skip, errCount int) {
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

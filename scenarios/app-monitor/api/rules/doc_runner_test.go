package rules_test

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"app-monitor-api/rules"
	_ "app-monitor-api/rules/interop" // trigger init() registrations
)

// =============================================================================
// Shared doc test runner: loads <test-case> blocks from embedded rule sources
// =============================================================================

var testCasePattern = regexp.MustCompile(
	`(?s)<test-case\s+id="([^"]+)"\s+should-fail="(true|false)">\s*` +
		`<description>(.*?)</description>\s*` +
		`<input>\s*(.*?)\s*</input>` +
		`(?:\s*<expected-violations>(\d+)</expected-violations>)?` +
		`(?:\s*<expected-message>(.*?)</expected-message>)?` +
		`\s*</test-case>`,
)

var fileMarkerPattern = regexp.MustCompile(`(?m)^\s*\[([^\]]+)\]\s*$`)

type docTestCase struct {
	id                 string
	description        string
	shouldFail         bool
	input              string
	expectedViolations int
	expectedMessage    string
}

func parseDocTestCases(src string) []docTestCase {
	matches := testCasePattern.FindAllStringSubmatch(src, -1)
	cases := make([]docTestCase, 0, len(matches))
	for _, m := range matches {
		shouldFail := m[2] == "true"
		expectedViolations := 0
		if m[5] != "" {
			expectedViolations, _ = strconv.Atoi(m[5])
		}
		cases = append(cases, docTestCase{
			id:                 m[1],
			description:        strings.TrimSpace(m[3]),
			shouldFail:         shouldFail,
			input:              m[4],
			expectedViolations: expectedViolations,
			expectedMessage:    strings.TrimSpace(m[6]),
		})
	}
	return cases
}

// createTestDir builds a temp directory from [filepath] markers in the input.
func createTestDir(t *testing.T, input string) string {
	t.Helper()
	root := t.TempDir()

	parts := fileMarkerPattern.Split(input, -1)
	markers := fileMarkerPattern.FindAllStringSubmatch(input, -1)

	// Skip anything before the first marker.
	for i, marker := range markers {
		relPath := strings.TrimSpace(marker[1])
		content := ""
		if i+1 < len(parts) {
			content = strings.TrimSpace(parts[i+1])
		}
		fullPath := filepath.Join(root, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("MkdirAll for %s: %v", relPath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile for %s: %v", relPath, err)
		}
	}

	return root
}

// TestAllDocCases iterates all registered rules, extracts <test-case> blocks
// from their embedded source, creates temp dirs, runs the check, and asserts.
func TestAllDocCases(t *testing.T) {
	allRules := rules.All()
	if len(allRules) == 0 {
		t.Fatal("no rules registered")
	}

	for _, r := range allRules {
		r := r
		// Find the embedded source for this rule to extract test cases.
		src := findRuleSource(t, r.Def.ID)
		if src == "" {
			continue // meta-test will catch missing doc cases
		}

		cases := parseDocTestCases(src)
		for _, tc := range cases {
			tc := tc
			t.Run(fmt.Sprintf("%s/%s", r.Def.ID, tc.id), func(t *testing.T) {
				root := createTestDir(t, tc.input)
				ctx := rules.CheckContext{
					ScenarioRoot: root,
					TechStack:    []string{"*"}, // run all rules
					ScenarioName: "test-scenario",
				}

				result := r.Check(ctx)

				if tc.shouldFail {
					if result.Passed {
						t.Errorf("expected failure but rule passed: %s", result.Message)
					}
					if tc.expectedViolations > 0 && len(result.Violations) != tc.expectedViolations {
						t.Errorf("expected %d violations, got %d", tc.expectedViolations, len(result.Violations))
					}
					if tc.expectedMessage != "" {
						found := strings.Contains(result.Message, tc.expectedMessage)
						if !found {
							for _, v := range result.Violations {
								if strings.Contains(v.Description, tc.expectedMessage) || strings.Contains(v.Title, tc.expectedMessage) {
									found = true
									break
								}
							}
						}
						if !found {
							t.Errorf("expected message containing %q, got %q (violations: %v)", tc.expectedMessage, result.Message, result.Violations)
						}
					}
				} else {
					if !result.Passed && !result.Skipped {
						t.Errorf("expected pass but rule failed: %s", result.Message)
					}
				}
			})
		}
	}
}

// findRuleSource searches all registered embed.FS files for the source containing
// the given rule ID.
func findRuleSource(t *testing.T, ruleID string) string {
	t.Helper()

	// Access the registered embed.FS files through the rules package.
	// We need to find the source file that contains this rule's ID.
	allRules := rules.All()
	for _, r := range allRules {
		if r.Def.ID != ruleID {
			continue
		}
		// The source is in the embedded FS. Search all registered FSes.
		for _, fsys := range rules.EmbedFSes() {
			var found string
			_ = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() || found != "" {
					return err
				}
				if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
					return nil
				}
				data, err := fs.ReadFile(fsys, path)
				if err != nil {
					return nil
				}
				src := string(data)
				if strings.Contains(src, fmt.Sprintf("ID: %s", ruleID)) {
					found = src
				}
				return nil
			})
			if found != "" {
				return found
			}
		}
	}
	return ""
}

// =============================================================================
// Meta-tests
// =============================================================================

// TestAllRulesHaveMetadata verifies every registered rule has required fields.
func TestAllRulesHaveMetadata(t *testing.T) {
	for _, r := range rules.All() {
		if r.Def.ID == "" {
			t.Error("rule registered with empty ID")
		}
		if r.Def.Name == "" {
			t.Errorf("rule %s has empty Name", r.Def.ID)
		}
		if r.Def.Severity == "" {
			t.Errorf("rule %s has empty Severity", r.Def.ID)
		}
		if r.Def.Category == "" {
			t.Errorf("rule %s has empty Category", r.Def.ID)
		}
	}
}

// TestRegistryIdsMatchDocIds verifies the ID in Register() matches the ID: field
// in the docstring.
func TestRegistryIdsMatchDocIds(t *testing.T) {
	for _, r := range rules.All() {
		if r.Def.ID == "" {
			continue
		}
		// The ID was set by Register() and should match the docstring.
		// Since build() marries them by ID, a mismatch means the rule
		// would get a minimal def (no Name, etc).
		if r.Def.Name == "" && r.Def.Category == "" {
			t.Errorf("rule %s: Register() ID has no matching docstring ID", r.Def.ID)
		}
	}
}

// TestAllRulesHaveDocCases verifies every rule has at least one pass and one fail test case.
func TestAllRulesHaveDocCases(t *testing.T) {
	for _, r := range rules.All() {
		src := findRuleSource(t, r.Def.ID)
		if src == "" {
			t.Errorf("rule %s: no embedded source found", r.Def.ID)
			continue
		}
		cases := parseDocTestCases(src)
		hasPass := false
		hasFail := false
		for _, tc := range cases {
			if tc.shouldFail {
				hasFail = true
			} else {
				hasPass = true
			}
		}
		if !hasPass {
			t.Errorf("rule %s: no passing test case in docstring", r.Def.ID)
		}
		if !hasFail {
			t.Errorf("rule %s: no failing test case in docstring", r.Def.ID)
		}
	}
}

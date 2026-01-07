package fix

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PromptConfig contains configuration for building a fix prompt.
type PromptConfig struct {
	ScenarioName string
	ScenarioPath string
	Phases       []PhaseInfo
}

// BuildPrompt constructs the prompt for the fix agent.
func BuildPrompt(cfg PromptConfig) string {
	var sb strings.Builder

	// Header
	sb.WriteString("# Test Fix Task\n\n")

	// Boundaries & Focus
	sb.WriteString("## Boundaries & Focus\n")
	sb.WriteString(fmt.Sprintf("- Working directory: `%s`\n", cfg.ScenarioPath))
	sb.WriteString("- Fix ONLY the failing test phases listed below\n")
	sb.WriteString("- Do NOT weaken tests or take shortcuts to make them pass\n")
	sb.WriteString("- Do NOT modify production code unless absolutely necessary to fix a test\n")
	sb.WriteString("- Do NOT delete tests or reduce coverage\n")
	sb.WriteString("- Prefer fixing the root cause over patching symptoms\n")
	sb.WriteString("- Preserve existing `[REQ:ID]` tags in tests for requirement tracking\n\n")

	// Failing Phases
	sb.WriteString("## Failing Phases\n\n")
	for _, phase := range cfg.Phases {
		sb.WriteString(fmt.Sprintf("### Phase: %s\n", phase.Name))
		sb.WriteString(fmt.Sprintf("- **Status**: %s\n", phase.Status))
		if phase.Duration > 0 {
			sb.WriteString(fmt.Sprintf("- **Duration**: %ds\n", phase.Duration))
		}
		if phase.Error != "" {
			sb.WriteString(fmt.Sprintf("- **Error**:\n```\n%s\n```\n", phase.Error))
		}
		if phase.LogPath != "" {
			sb.WriteString(fmt.Sprintf("- **Log file**: `%s`\n", phase.LogPath))
		}
		sb.WriteString("\n")
	}

	// Phase Reference
	sb.WriteString("## Phase Reference\n\n")
	sb.WriteString("Each phase has specific characteristics:\n\n")
	sb.WriteString("| Phase | Purpose | Timeout | Test Location |\n")
	sb.WriteString("|-------|---------|---------|---------------|\n")
	sb.WriteString("| structure | Files, config, CLI validation | 15s | `__test/phases/structure/` |\n")
	sb.WriteString("| standards | scenario-auditor rules | 60s | N/A (automated) |\n")
	sb.WriteString("| dependencies | Tools and resources | 30s | `__test/phases/dependencies/` |\n")
	sb.WriteString("| lint | Type checking | 30s | N/A (automated) |\n")
	sb.WriteString("| docs | Markdown/link validation | 60s | `__test/phases/docs/` |\n")
	sb.WriteString("| unit | Code-level tests | 60s | `api/*_test.go`, `ui/src/**/*.test.ts` |\n")
	sb.WriteString("| integration | API/component tests | 120s | `__test/phases/integration/` |\n")
	sb.WriteString("| playbooks | BAS browser automation | 120s | `bas/` workflows |\n")
	sb.WriteString("| business | Requirements validation | 180s | `__test/phases/business/` |\n")
	sb.WriteString("| performance | Benchmarks, Lighthouse | 60s | `__test/phases/performance/` |\n\n")

	// Validation Loop
	sb.WriteString("## Validation Loop\n\n")
	sb.WriteString("**Before and after each fix, run these commands to verify status:**\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("# 1. Check current scenario status\nvrooli scenario status %s\n\n", cfg.ScenarioName))
	sb.WriteString(fmt.Sprintf("# 2. Run the specific failing phase\nvrooli scenario test %s <phase-name>\n\n", cfg.ScenarioName))
	sb.WriteString(fmt.Sprintf("# 3. After fixing, run comprehensive tests to ensure no regressions\nvrooli scenario test %s\n", cfg.ScenarioName))
	sb.WriteString("```\n\n")
	sb.WriteString("**Do NOT declare success until the validation loop passes.** Re-run after every change.\n\n")

	// Test Commands
	sb.WriteString("## Test Commands\n\n")
	sb.WriteString("Run tests for this scenario using the vrooli CLI:\n\n")
	sb.WriteString(fmt.Sprintf("```bash\n# Run all phases\nvrooli scenario test %s\n\n", cfg.ScenarioName))
	sb.WriteString(fmt.Sprintf("# Run a single phase\nvrooli scenario test %s <phase-name>\n\n", cfg.ScenarioName))
	sb.WriteString("# Common phase names: structure, dependencies, unit, integration, e2e, business, performance\n```\n\n")

	// Task Instructions
	sb.WriteString("## Your Task\n\n")
	sb.WriteString("Follow these steps carefully:\n\n")
	sb.WriteString("1. **Understand the failures**: Read the error messages above. If log files are available, read them for more context.\n\n")
	sb.WriteString("2. **Locate the failing tests**: Use grep/glob to find the test files. Look for test files in:\n")
	sb.WriteString("   - `__test/` directory for shell-based tests (bats, etc.)\n")
	sb.WriteString("   - `*_test.go` files for Go tests\n")
	sb.WriteString("   - `*.test.ts` or `*.spec.ts` files for TypeScript tests\n\n")
	sb.WriteString("3. **Analyze the root cause**: Determine whether the issue is:\n")
	sb.WriteString("   - A flaky test (timing, race condition)\n")
	sb.WriteString("   - An incorrect assertion\n")
	sb.WriteString("   - A missing mock or fixture\n")
	sb.WriteString("   - An environment or configuration issue\n")
	sb.WriteString("   - A genuine bug in production code that needs fixing\n\n")
	sb.WriteString("4. **Fix the tests properly**:\n")
	sb.WriteString("   - Fix the underlying issue, not just the symptoms\n")
	sb.WriteString("   - Ensure assertions remain meaningful and correct\n")
	sb.WriteString("   - If a production bug exists, fix it properly\n")
	sb.WriteString("   - Add comments explaining non-obvious fixes\n\n")
	sb.WriteString("5. **Validate your fixes**: Run the tests to confirm they pass:\n")
	sb.WriteString(fmt.Sprintf("   ```bash\n   vrooli scenario test %s\n   ```\n\n", cfg.ScenarioName))
	sb.WriteString("6. **Do NOT declare success until tests actually pass**. If tests still fail, continue debugging.\n\n")

	// Safety Warnings
	sb.WriteString("## Critical Safety Warnings\n\n")
	sb.WriteString("**When fixing shell scripts (BATS, bash tests), follow these MANDATORY rules:**\n\n")
	sb.WriteString("### BATS Teardown Safety\n")
	sb.WriteString("BATS `teardown()` functions run even when tests are skipped. If `setup()` calls `skip` before setting variables, `teardown()` runs with empty variables.\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# DANGEROUS - Can delete everything if TEST_FILE_PREFIX is empty\n")
	sb.WriteString("teardown() {\n")
	sb.WriteString("    rm -f \"${TEST_FILE_PREFIX}\"*\n")
	sb.WriteString("}\n\n")
	sb.WriteString("# SAFE - Always guard with proper checks\n")
	sb.WriteString("teardown() {\n")
	sb.WriteString("    if [ -n \"${TEST_FILE_PREFIX:-}\" ] && [ \"${TEST_FILE_PREFIX}\" != \"/\" ]; then\n")
	sb.WriteString("        case \"${TEST_FILE_PREFIX}\" in\n")
	sb.WriteString("            /tmp/*)\n")
	sb.WriteString("                rm -f \"${TEST_FILE_PREFIX}\"* 2>/dev/null || true\n")
	sb.WriteString("                ;;\n")
	sb.WriteString("            *)\n")
	sb.WriteString("                echo \"WARNING: Unsafe TEST_FILE_PREFIX, skipping cleanup\" >&2\n")
	sb.WriteString("                ;;\n")
	sb.WriteString("        esac\n")
	sb.WriteString("    fi\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n\n")
	sb.WriteString("### Setup Function Order\n")
	sb.WriteString("**ALWAYS** set critical variables before any skip conditions:\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# SAFE - Set variables first\n")
	sb.WriteString("setup() {\n")
	sb.WriteString("    export TEST_FILE_PREFIX=\"/tmp/my-test\"  # Always set\n")
	sb.WriteString("    if ! command -v my-cli >/dev/null 2>&1; then\n")
	sb.WriteString("        skip \"CLI not installed\"\n")
	sb.WriteString("    fi\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n\n")
	sb.WriteString("### Path Validation\n")
	sb.WriteString("**NEVER** use unguarded `rm` with variables. Always validate paths point to safe locations (`/tmp/`).\n\n")

	// Framework-Specific Patterns
	sb.WriteString("## Framework-Specific Patterns\n\n")
	sb.WriteString("### Go Tests (`*_test.go`)\n")
	sb.WriteString("```go\n")
	sb.WriteString("// Table-driven tests for comprehensive coverage\n")
	sb.WriteString("func TestHandler(t *testing.T) {\n")
	sb.WriteString("    tests := []struct {\n")
	sb.WriteString("        name           string\n")
	sb.WriteString("        input          string\n")
	sb.WriteString("        expectedStatus int\n")
	sb.WriteString("    }{\n")
	sb.WriteString("        {\"valid input\", `{\"data\":\"test\"}`, http.StatusOK},\n")
	sb.WriteString("        {\"invalid JSON\", `{invalid}`, http.StatusBadRequest},\n")
	sb.WriteString("    }\n")
	sb.WriteString("    for _, tt := range tests {\n")
	sb.WriteString("        t.Run(tt.name+\" [REQ:SCENARIO-REQ-001]\", func(t *testing.T) {\n")
	sb.WriteString("            // Test implementation\n")
	sb.WriteString("        })\n")
	sb.WriteString("    }\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n\n")
	sb.WriteString("### Vitest/TypeScript Tests (`*.test.ts`)\n")
	sb.WriteString("```typescript\n")
	sb.WriteString("// Use describe blocks with [REQ:ID] tags for tracking\n")
	sb.WriteString("describe('featureName [REQ:SCENARIO-FEATURE-001]', () => {\n")
	sb.WriteString("  it('handles valid input', () => {\n")
	sb.WriteString("    // Test implementation\n")
	sb.WriteString("  });\n")
	sb.WriteString("});\n")
	sb.WriteString("```\n\n")
	sb.WriteString("### BATS Tests (`*.bats`)\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# [REQ:SCENARIO-CLI-001] Tests CLI functionality\n")
	sb.WriteString("@test \"CLI executes successfully\" {\n")
	sb.WriteString("    run my-cli --help\n")
	sb.WriteString("    [ \"$status\" -eq 0 ]\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n\n")

	// Success Criteria
	sb.WriteString("## Success Criteria\n\n")
	sb.WriteString("- All specified phases pass when re-run\n")
	sb.WriteString("- No regressions introduced to other tests\n")
	sb.WriteString("- Changes are minimal and focused on the fix\n")
	sb.WriteString("- Test assertions remain meaningful (no weakening)\n\n")

	// Output Format
	sb.WriteString("## Final Output\n\n")
	sb.WriteString("When you have successfully fixed the tests, provide:\n\n")
	sb.WriteString("1. **Files Changed**: List all files you modified\n")
	sb.WriteString("2. **Test Results**: Show the output from running the tests\n")
	sb.WriteString("3. **Summary**: Brief description of what was broken and how you fixed it\n")

	return sb.String()
}

// GetScenarioPath returns the full path to a scenario directory.
func GetScenarioPath(scenarioName string) string {
	repoRoot := os.Getenv("VROOLI_ROOT")
	if repoRoot == "" {
		repoRoot = filepath.Join(os.Getenv("HOME"), "Vrooli")
	}
	return filepath.Join(repoRoot, "scenarios", scenarioName)
}

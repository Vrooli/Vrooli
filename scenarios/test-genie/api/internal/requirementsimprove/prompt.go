package requirementsimprove

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PromptConfig contains configuration for building an improve prompt.
type PromptConfig struct {
	ScenarioName string
	ScenarioPath string
	Requirements []RequirementInfo
	ActionType   ActionType
}

// BuildPrompt constructs the prompt for the requirements improve agent.
func BuildPrompt(cfg PromptConfig) string {
	var sb strings.Builder

	// Header
	sb.WriteString("# Requirements Improvement Task\n\n")

	// Task Overview
	sb.WriteString("## Task Overview\n\n")
	switch cfg.ActionType {
	case ActionWriteTests:
		sb.WriteString("Your goal is to **write or fix tests** for the selected requirements to achieve passing status.\n\n")
	case ActionUpdateReqs:
		sb.WriteString("Your goal is to **update requirement files** to accurately reflect the current implementation state.\n\n")
	case ActionBoth:
		sb.WriteString("Your goal is to both **write/fix tests** AND **update requirement files** for the selected requirements.\n\n")
	}

	// Boundaries & Focus
	sb.WriteString("## Boundaries & Focus\n")
	sb.WriteString(fmt.Sprintf("- Working directory: `%s`\n", cfg.ScenarioPath))
	sb.WriteString("- Focus ONLY on the requirements listed below\n")
	sb.WriteString("- Do NOT weaken tests or reduce coverage\n")
	sb.WriteString("- Ensure tests are properly tagged with `[REQ:ID]` comments\n")
	sb.WriteString("- Follow existing test patterns and conventions in this scenario\n")
	sb.WriteString("- Do NOT modify production code unless absolutely necessary\n\n")

	// Validation Loop
	sb.WriteString("## Validation Loop\n\n")
	sb.WriteString("**Run these commands before and after each change to verify progress:**\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("# 1. Check scenario status and failing checks\nvrooli scenario status %s\n\n", cfg.ScenarioName))
	sb.WriteString(fmt.Sprintf("# 2. Run tests (triggers auto-sync of requirement coverage)\nvrooli scenario test %s\n\n", cfg.ScenarioName))
	sb.WriteString(fmt.Sprintf("# 3. View updated requirements coverage\nvrooli scenario requirements report %s --format markdown\n", cfg.ScenarioName))
	sb.WriteString("```\n\n")
	sb.WriteString("**Do NOT declare success until the validation loop shows passing tests and updated coverage.**\n\n")

	// Auto-Sync Explanation
	sb.WriteString("## How Auto-Sync Works\n\n")
	sb.WriteString("When you run `vrooli scenario test`, the system automatically:\n\n")
	sb.WriteString("1. **Discovers** all tests with `[REQ:ID]` tags\n")
	sb.WriteString("2. **Collects evidence** from:\n")
	sb.WriteString("   - Go test results (`*_test.go`)\n")
	sb.WriteString("   - Vitest results (`ui/coverage/vitest-requirements.json`)\n")
	sb.WriteString("   - BATS test results\n")
	sb.WriteString("3. **Updates** requirement files with live status:\n")
	sb.WriteString("   - `passed` → validation status becomes `implemented`\n")
	sb.WriteString("   - `failed` → validation status becomes `failing`\n")
	sb.WriteString("   - Requirement status auto-updates based on all validations\n\n")
	sb.WriteString("**You do NOT need to manually update requirement files after tests pass** - auto-sync handles it.\n\n")

	// Requirements to address
	sb.WriteString("## Requirements to Address\n\n")
	for _, req := range cfg.Requirements {
		sb.WriteString(fmt.Sprintf("### %s: %s\n", req.ID, req.Title))
		sb.WriteString(fmt.Sprintf("- **Declared Status**: %s\n", req.Status))
		sb.WriteString(fmt.Sprintf("- **Live Status**: %s\n", req.LiveStatus))
		if req.Criticality != "" {
			sb.WriteString(fmt.Sprintf("- **Criticality**: %s\n", req.Criticality))
		}
		sb.WriteString(fmt.Sprintf("- **Requirements File**: `%s`\n", req.ModulePath))
		if req.Description != "" {
			sb.WriteString(fmt.Sprintf("- **Description**: %s\n", req.Description))
		}

		if len(req.Validations) > 0 {
			sb.WriteString("\n**Current Validations:**\n")
			for _, val := range req.Validations {
				phaseInfo := ""
				if val.Phase != "" {
					phaseInfo = fmt.Sprintf("/%s", val.Phase)
				}
				sb.WriteString(fmt.Sprintf("  - `%s` (%s%s): %s\n",
					val.Ref, val.Type, phaseInfo, val.LiveStatus))
			}
		} else {
			sb.WriteString("\n**No validations defined yet**\n")
		}
		sb.WriteString("\n")
	}

	// Status Definitions
	sb.WriteString("## Status Definitions\n\n")
	sb.WriteString("### Requirement Status\n")
	sb.WriteString("| Status | Meaning |\n")
	sb.WriteString("|--------|--------|\n")
	sb.WriteString("| `pending` | Requirement identified but not yet planned |\n")
	sb.WriteString("| `in_progress` | At least one test exists (auto-detected by sync) |\n")
	sb.WriteString("| `complete` | All validations passing (auto-updated by sync) |\n")
	sb.WriteString("| `not_implemented` | Requirement deprioritized or deferred |\n\n")
	sb.WriteString("### Validation Status\n")
	sb.WriteString("| Status | Meaning |\n")
	sb.WriteString("|--------|--------|\n")
	sb.WriteString("| `not_implemented` | Test not written yet |\n")
	sb.WriteString("| `implemented` | Test exists and passing |\n")
	sb.WriteString("| `failing` | Test exists but currently failing |\n\n")
	sb.WriteString("### Live Status (from test runs)\n")
	sb.WriteString("| Status | Meaning |\n")
	sb.WriteString("|--------|--------|\n")
	sb.WriteString("| `passed` | Test executed and passed |\n")
	sb.WriteString("| `failed` | Test executed and failed |\n")
	sb.WriteString("| `skipped` | Test was skipped |\n")
	sb.WriteString("| `not_run` | No evidence found for this validation |\n\n")
	sb.WriteString("### Criticality Levels\n")
	sb.WriteString("| Level | Meaning | CI/CD Impact |\n")
	sb.WriteString("|-------|---------|-------------|\n")
	sb.WriteString("| **P0** | Critical - core functionality required for MVP | Blocks releases |\n")
	sb.WriteString("| **P1** | Important - significantly improves UX | Tracked, may not block |\n")
	sb.WriteString("| **P2** | Nice-to-have - polish and expansion | Tracked only |\n\n")

	// Test tagging convention
	if cfg.ActionType == ActionWriteTests || cfg.ActionType == ActionBoth {
		sb.WriteString("## Test Tagging Convention\n\n")
		sb.WriteString("When writing tests, add a comment with the requirement ID. This enables automatic tracking:\n\n")
		sb.WriteString("**Go tests:**\n")
		sb.WriteString("```go\n// [REQ:TESTGENIE-ORCH-P0] Tests scenario-local orchestrator\nfunc TestOrchestratorExecution(t *testing.T) { ... }\n```\n\n")
		sb.WriteString("**TypeScript/JavaScript tests:**\n")
		sb.WriteString("```typescript\n// [REQ:TESTGENIE-UI-001] Tests requirements panel rendering\ntest('renders requirements tree', () => { ... });\n```\n\n")
		sb.WriteString("**Bash/Bats tests:**\n")
		sb.WriteString("```bash\n# [REQ:TESTGENIE-CLI-001] Tests CLI execution\n@test \"CLI executes tests\" { ... }\n```\n\n")
	}

	// Validation phases
	if cfg.ActionType == ActionWriteTests || cfg.ActionType == ActionBoth {
		sb.WriteString("## Validation Phases\n\n")
		sb.WriteString("Tests can belong to different validation phases:\n\n")
		sb.WriteString("| Phase | Purpose | Test Location |\n")
		sb.WriteString("|-------|---------|---------------|\n")
		sb.WriteString("| unit | Fast, isolated tests | `api/*_test.go`, `ui/src/**/*.test.ts` |\n")
		sb.WriteString("| integration | Component interactions | `__test/phases/integration/` |\n")
		sb.WriteString("| business | End-to-end workflow validation | `__test/phases/business/` |\n")
		sb.WriteString("| performance | Benchmarks, load tests | `__test/phases/performance/` |\n")
		sb.WriteString("| structure | File structure validation | `__test/phases/structure/` |\n\n")

		// Framework Setup
		sb.WriteString("## Framework-Specific Setup\n\n")
		sb.WriteString("### Vitest (UI Tests)\n\n")
		sb.WriteString("For `[REQ:ID]` tags to be tracked in TypeScript tests, ensure the custom reporter is configured:\n\n")
		sb.WriteString("```typescript\n")
		sb.WriteString("// vite.config.ts\n")
		sb.WriteString("import RequirementReporter from '@vrooli/vitest-requirement-reporter';\n\n")
		sb.WriteString("export default defineConfig({\n")
		sb.WriteString("  test: {\n")
		sb.WriteString("    reporters: [\n")
		sb.WriteString("      'default',\n")
		sb.WriteString("      new RequirementReporter({\n")
		sb.WriteString("        outputFile: 'coverage/vitest-requirements.json',\n")
		sb.WriteString("        emitStdout: true,  // Required for phase integration\n")
		sb.WriteString("        verbose: true,\n")
		sb.WriteString("      }),\n")
		sb.WriteString("    ],\n")
		sb.WriteString("  },\n")
		sb.WriteString("});\n")
		sb.WriteString("```\n\n")
		sb.WriteString("If this config doesn't exist, add it. The reporter extracts `[REQ:ID]` tags from test names.\n\n")

		// Safety Warnings for Shell Tests
		sb.WriteString("## Critical Safety Warnings\n\n")
		sb.WriteString("**When writing BATS or shell tests, follow these MANDATORY rules:**\n\n")
		sb.WriteString("### BATS Teardown Safety\n")
		sb.WriteString("BATS `teardown()` runs even when tests are skipped. Guard all cleanup:\n\n")
		sb.WriteString("```bash\n")
		sb.WriteString("# DANGEROUS\n")
		sb.WriteString("teardown() {\n")
		sb.WriteString("    rm -f \"${TEST_FILE_PREFIX}\"*  # Can delete everything if empty!\n")
		sb.WriteString("}\n\n")
		sb.WriteString("# SAFE\n")
		sb.WriteString("teardown() {\n")
		sb.WriteString("    if [ -n \"${TEST_FILE_PREFIX:-}\" ] && [[ \"${TEST_FILE_PREFIX}\" == /tmp/* ]]; then\n")
		sb.WriteString("        rm -f \"${TEST_FILE_PREFIX}\"* 2>/dev/null || true\n")
		sb.WriteString("    fi\n")
		sb.WriteString("}\n")
		sb.WriteString("```\n\n")
		sb.WriteString("### Setup Order\n")
		sb.WriteString("**ALWAYS** set variables before skip conditions:\n\n")
		sb.WriteString("```bash\n")
		sb.WriteString("setup() {\n")
		sb.WriteString("    export TEST_FILE_PREFIX=\"/tmp/my-test\"  # Set first!\n")
		sb.WriteString("    if ! command -v my-cli >/dev/null 2>&1; then\n")
		sb.WriteString("        skip \"CLI not installed\"\n")
		sb.WriteString("    fi\n")
		sb.WriteString("}\n")
		sb.WriteString("```\n\n")
	}

	// Requirements file format
	if cfg.ActionType == ActionUpdateReqs || cfg.ActionType == ActionBoth {
		sb.WriteString("## Requirements File Format\n\n")
		sb.WriteString("Requirements are in JSON format with the following structure:\n\n")
		sb.WriteString("```json\n")
		sb.WriteString("{\n")
		sb.WriteString("  \"requirements\": [\n")
		sb.WriteString("    {\n")
		sb.WriteString("      \"id\": \"REQ-001\",\n")
		sb.WriteString("      \"title\": \"Requirement title\",\n")
		sb.WriteString("      \"status\": \"complete\",        // \"pending\" | \"in_progress\" | \"complete\"\n")
		sb.WriteString("      \"validation\": [\n")
		sb.WriteString("        {\n")
		sb.WriteString("          \"type\": \"test\",          // \"test\" | \"automation\" | \"manual\"\n")
		sb.WriteString("          \"ref\": \"path/to/test.go\", // Path to test file\n")
		sb.WriteString("          \"phase\": \"unit\",          // \"unit\" | \"integration\" | \"business\" | \"performance\"\n")
		sb.WriteString("          \"status\": \"implemented\"   // \"pending\" | \"implemented\" | \"passing\" | \"failing\"\n")
		sb.WriteString("        }\n")
		sb.WriteString("      ]\n")
		sb.WriteString("    }\n")
		sb.WriteString("  ]\n")
		sb.WriteString("}\n")
		sb.WriteString("```\n\n")
		sb.WriteString("**Important:**\n")
		sb.WriteString("- Only set `status: \"complete\"` if the requirement is fully implemented\n")
		sb.WriteString("- Set validation `status: \"implemented\"` when the test exists and passes\n")
		sb.WriteString("- The `ref` should point to the actual test file path\n\n")
	}

	// Test Commands
	sb.WriteString("## Test Commands\n\n")
	sb.WriteString("Run tests for this scenario using the vrooli CLI:\n\n")
	sb.WriteString(fmt.Sprintf("```bash\n# Run all test phases\nvrooli scenario test %s\n\n", cfg.ScenarioName))
	sb.WriteString(fmt.Sprintf("# Run a specific phase\nvrooli scenario test %s <phase-name>\n\n", cfg.ScenarioName))
	sb.WriteString("# Common phase names: structure, dependencies, unit, integration, e2e, business, performance\n```\n\n")
	sb.WriteString("**Note:** Running the full test suite triggers automatic requirement coverage sync.\n\n")

	// Task Instructions
	sb.WriteString("## Your Task\n\n")
	sb.WriteString("Follow these steps carefully:\n\n")

	if cfg.ActionType == ActionWriteTests || cfg.ActionType == ActionBoth {
		sb.WriteString("### Writing/Fixing Tests\n\n")
		sb.WriteString("1. **Analyze each requirement**: Understand what needs to be validated\n")
		sb.WriteString("2. **Find existing tests**: Search for tests tagged with `[REQ:ID]` or related test files\n")
		sb.WriteString("3. **Create or fix tests**: Write tests that properly validate the requirement\n")
		sb.WriteString("4. **Tag tests properly**: Add `[REQ:ID]` comments to link tests to requirements\n")
		sb.WriteString("5. **Run tests**: Verify all tests pass using `vrooli scenario test`\n\n")
	}

	if cfg.ActionType == ActionUpdateReqs || cfg.ActionType == ActionBoth {
		sb.WriteString("### Updating Requirements\n\n")
		sb.WriteString("1. **Review implementation**: Check if features are actually implemented\n")
		sb.WriteString("2. **Update status**: Set to \"complete\" only if fully implemented\n")
		sb.WriteString("3. **Add validations**: Link to test files that validate the requirement\n")
		sb.WriteString("4. **Set validation status**: \"implemented\" if test exists and passes\n\n")
	}

	// Success Criteria
	sb.WriteString("## Success Criteria\n\n")
	if cfg.ActionType == ActionWriteTests || cfg.ActionType == ActionBoth {
		sb.WriteString("- Tests pass when running `vrooli scenario test`\n")
		sb.WriteString("- Tests are properly tagged with `[REQ:ID]` comments\n")
	}
	if cfg.ActionType == ActionUpdateReqs || cfg.ActionType == ActionBoth {
		sb.WriteString("- Requirements files accurately reflect implementation state\n")
		sb.WriteString("- Validations reference actual test files\n")
	}
	sb.WriteString("- No regressions in existing tests\n")
	sb.WriteString("- Changes are minimal and focused\n\n")

	// Output Format
	sb.WriteString("## Final Output\n\n")
	sb.WriteString("When you have successfully completed the task, provide:\n\n")
	sb.WriteString("1. **Files Changed**: List all files you modified or created\n")
	sb.WriteString("2. **Test Results**: Show the output from running `vrooli scenario test`\n")
	sb.WriteString("3. **Summary**: Brief description of what was done and the improvements made\n")

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

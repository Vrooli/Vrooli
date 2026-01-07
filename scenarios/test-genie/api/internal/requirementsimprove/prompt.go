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
		sb.WriteString("- **unit**: Fast, isolated tests (Go `_test.go`, Jest, Vitest, etc.)\n")
		sb.WriteString("- **integration**: Tests that exercise multiple components together\n")
		sb.WriteString("- **business**: End-to-end business workflow validation\n")
		sb.WriteString("- **performance**: Load, stress, or timing tests\n")
		sb.WriteString("- **structure**: Code structure and dependency validation\n\n")
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

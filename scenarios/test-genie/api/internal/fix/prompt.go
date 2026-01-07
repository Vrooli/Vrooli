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
	sb.WriteString("- Prefer fixing the root cause over patching symptoms\n\n")

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

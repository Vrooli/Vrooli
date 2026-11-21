package manifestschema

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// CheckScenarioShellInvocations rejects shell interpreters and shell files in
// the two declared execution surfaces: lifecycle argv and component argv.
// Shell files elsewhere in a scenario can still be fixtures, examples, or
// operator tools; the lifecycle contract is the boundary this rule owns.
func CheckScenarioShellInvocations(content []byte, filePath string) []Violation {
	var document map[string]any
	if json.Unmarshal(content, &document) != nil {
		return nil
	}

	var out []Violation
	lifecycle, _ := document["lifecycle"].(map[string]any)
	for phaseName, rawPhase := range lifecycle {
		phase, _ := rawPhase.(map[string]any)
		steps, _ := phase["steps"].([]any)
		for index, rawStep := range steps {
			step, _ := rawStep.(map[string]any)
			if argv, ok := shellStringArray(step["exec"]); ok && declaredShell(argv) {
				out = append(out, shellViolation(filePath, fmt.Sprintf("lifecycle.%s.steps[%d].exec", phaseName, index)))
			}
		}
	}

	components, _ := document["components"].(map[string]any)
	for name, rawComponent := range components {
		component, _ := rawComponent.(map[string]any)
		run, _ := component["run"].(map[string]any)
		if argv, ok := shellStringArray(run["argv"]); ok && declaredShell(argv) {
			out = append(out, shellViolation(filePath, "components."+name+".run.argv"))
		}
	}
	return out
}

func shellStringArray(value any) ([]string, bool) {
	raw, ok := value.([]any)
	if !ok || len(raw) == 0 {
		return nil, false
	}
	values := make([]string, 0, len(raw))
	for _, item := range raw {
		value, ok := item.(string)
		if !ok {
			return nil, false
		}
		values = append(values, value)
	}
	return values, true
}

func declaredShell(argv []string) bool {
	for _, argument := range argv {
		base := filepath.Base(strings.TrimSpace(strings.ToLower(argument)))
		if base == "sh" || base == "bash" || base == "zsh" || base == "dash" ||
			strings.HasSuffix(base, ".sh") || strings.HasSuffix(base, ".bash") {
			return true
		}
	}
	return false
}

func shellViolation(filePath, location string) Violation {
	return Violation{
		Type:           "config_scenario_shell",
		Severity:       "high",
		Title:          "Scenario shell invocation",
		Description:    "declared scenario work invokes a shell interpreter or shell file",
		FilePath:       filePath,
		LineNumber:     1,
		Recommendation: "Replace " + location + " with component metadata, data_dirs, argv-native execution, or a Go CLI subcommand.",
	}
}

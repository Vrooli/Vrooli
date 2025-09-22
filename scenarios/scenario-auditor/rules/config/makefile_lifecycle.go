package config

import (
	"fmt"
	"regexp"
	"strings"
)

/*
Rule: Makefile Lifecycle Commands
Description: Ensures lifecycle targets call the Vrooli CLI with the canonical messaging
Reason: Keeps lifecycle orchestration consistent and prevents direct execution regressions
Category: makefile
Severity: high
Targets: makefile

<test-case id="lifecycle-valid" should-fail="false">
  <description>Lifecycle commands use vrooli scenario CLI</description>
  <input language="make">
start:
	@echo "$(BLUE)🚀 Starting demo scenario...$(RESET)"
	@vrooli scenario start demo

stop:
	@echo "$(YELLOW)⏹️  Stopping demo scenario...$(RESET)"
	@vrooli scenario stop demo

test:
	@echo "$(BLUE)🧪 Testing demo scenario...$(RESET)"
	@vrooli scenario test demo

logs:
	@echo "$(BLUE)📜 Logs for demo:$(RESET)"
	@vrooli scenario logs demo --tail 50

status:
	@echo "$(BLUE)📊 Status of demo:$(RESET)"
	@vrooli scenario status demo
  </input>
</test-case>

<test-case id="lifecycle-legacy-run" should-fail="true">
  <description>Start command uses legacy run</description>
  <input language="make">
start:
	@echo "$(BLUE)🚀 Starting demo scenario...$(RESET)"
	@vrooli scenario run demo
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>execute 'vrooli scenario start</expected-message>
</test-case>

*/

type MakefileLifecycleViolation struct {
	Severity string
	Message  string
	FilePath string
	Line     int
}

type lifecycleMakefileData struct {
	lines   []string
	targets map[string][]string
}

var lifecycleTargetRegexp = regexp.MustCompile(`^([A-Za-z0-9_.-]+)\s*:(.*)$`)

// CheckMakefileLifecycle validates lifecycle targets adhere to the expected implementation.
func CheckMakefileLifecycle(content string, filepath string) ([]MakefileLifecycleViolation, error) {
	data := parseLifecycleMakefile(content)
	var violations []MakefileLifecycleViolation

	violations = append(violations, lifecycleValidateTarget(data, filepath, "start", "🚀 Starting", "vrooli scenario start $(SCENARIO_NAME)")...)
	violations = append(violations, lifecycleValidateTarget(data, filepath, "stop", "⏹️  Stopping", "vrooli scenario stop $(SCENARIO_NAME)")...)
	violations = append(violations, lifecycleValidateTarget(data, filepath, "test", "🧪 Testing", "vrooli scenario test $(SCENARIO_NAME)")...)
	violations = append(violations, lifecycleValidateTarget(data, filepath, "logs", "📜 Logs", "vrooli scenario logs $(SCENARIO_NAME) --tail 50")...)
	violations = append(violations, lifecycleValidateTarget(data, filepath, "status", "📊 Status", "vrooli scenario status $(SCENARIO_NAME)")...)

	return violations, nil
}

func lifecycleValidateTarget(data lifecycleMakefileData, path, target, messageFragment, command string) []MakefileLifecycleViolation {
	var violations []MakefileLifecycleViolation
	recipe := lifecycleNormalize(data.targets[target])
	if len(recipe) == 0 {
		violations = append(violations, MakefileLifecycleViolation{
			Severity: "high",
			Message:  fmt.Sprintf("%s target missing", target),
			FilePath: path,
			Line:     lifecycleFindLine(data.lines, target+":"),
		})
		return violations
	}

	if !lifecycleContains(recipe, messageFragment) {
		violations = append(violations, MakefileLifecycleViolation{
			Severity: "high",
			Message:  fmt.Sprintf("%s target must echo message containing '%s'", target, messageFragment),
			FilePath: path,
			Line:     lifecycleFindLine(data.lines, target+":"),
		})
	}

	if !lifecycleContains(recipe, command) {
		violations = append(violations, MakefileLifecycleViolation{
			Severity: "high",
			Message:  fmt.Sprintf("%s target must execute '%s'", target, command),
			FilePath: path,
			Line:     lifecycleFindLine(data.lines, target+":"),
		})
	}

	return violations
}

func parseLifecycleMakefile(content string) lifecycleMakefileData {
	lines := strings.Split(content, "\n")
	data := lifecycleMakefileData{
		lines:   lines,
		targets: make(map[string][]string),
	}

	var currentTarget string
	for _, raw := range lines {
		trimmed := strings.TrimLeft(raw, "\t ")

		if strings.HasPrefix(raw, "\t") && currentTarget != "" {
			data.targets[currentTarget] = append(data.targets[currentTarget], raw)
			continue
		}

		matches := lifecycleTargetRegexp.FindStringSubmatch(trimmed)
		if len(matches) == 3 {
			currentTarget = matches[1]
			if _, ok := data.targets[currentTarget]; !ok {
				data.targets[currentTarget] = []string{}
			}
			remainder := strings.TrimSpace(matches[2])
			if remainder != "" {
				data.targets[currentTarget] = append(data.targets[currentTarget], "\t"+remainder)
			}
			continue
		}

		currentTarget = ""
	}

	return data
}

func lifecycleNormalize(lines []string) []string {
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		normalized = append(normalized, strings.TrimSpace(line))
	}
	return normalized
}

func lifecycleContains(lines []string, fragment string) bool {
	for _, line := range lines {
		if strings.Contains(line, fragment) {
			return true
		}
	}
	return false
}

func lifecycleFindLine(lines []string, needle string) int {
	for idx, line := range lines {
		if strings.Contains(line, needle) {
			return idx + 1
		}
	}
	return 1
}

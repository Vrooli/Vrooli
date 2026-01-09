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
  <input language="make"><![CDATA[
SCENARIO_NAME := $(notdir $(CURDIR))

start:
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME)

stop:
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

test:
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

logs:
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

status:
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario status $(SCENARIO_NAME)
  ]]></input>
</test-case>

<test-case id="lifecycle-legacy-run" should-fail="true">
  <description>Start command uses legacy run</description>
  <input language="make"><![CDATA[
SCENARIO_NAME := $(notdir $(CURDIR))

start:
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario run $(SCENARIO_NAME)

stop:
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

test:
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

logs:
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

status:
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario status $(SCENARIO_NAME)
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>execute 'vrooli scenario start</expected-message>
</test-case>

<test-case id="lifecycle-run-masked" should-fail="true">
  <description>Start command echoes canonical message but invokes non-compliant command</description>
  <input language="make"><![CDATA[
SCENARIO_NAME := $(notdir $(CURDIR))

start:
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@echo "Running vrooli scenario start $(SCENARIO_NAME)"
	@vrooli scenario run $(SCENARIO_NAME)

stop:
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

test:
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

logs:
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

status:
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario status $(SCENARIO_NAME)
  ]]></input>
  <expected-violations>1</expected-violations>
<expected-message>execute 'vrooli scenario start</expected-message>
</test-case>

<test-case id="lifecycle-start-message-mismatch" should-fail="true">
  <description>Start echo does not match canonical wording</description>
  <input language="make"><![CDATA[
SCENARIO_NAME := $(notdir $(CURDIR))

start:
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME)...$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME)

stop:
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

test:
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

logs:
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

status:
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario status $(SCENARIO_NAME)
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>start target must echo 'echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"'</expected-message>
</test-case>

<test-case id="lifecycle-start-extra-flags" should-fail="true">
  <description>Start command includes unsupported flags</description>
  <input language="make"><![CDATA[
SCENARIO_NAME := $(notdir $(CURDIR))

start:
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME) --dev

stop:
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

test:
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

logs:
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

status:
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario status $(SCENARIO_NAME)
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>execute 'vrooli scenario start</expected-message>
</test-case>

<test-case id="lifecycle-multiline" should-fail="false">
  <description>Lifecycle commands use multiline continuation and quoting</description>
  <input language="make"><![CDATA[
SCENARIO_NAME := $(notdir $(CURDIR))

start:
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario start \
		"$(SCENARIO_NAME)"

stop:
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop "$(SCENARIO_NAME)"

test:
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test "$(SCENARIO_NAME)"

logs:
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs "$(SCENARIO_NAME)" \
		--tail 50

status:
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario status "$(SCENARIO_NAME)"
  ]]></input>
</test-case>

<test-case id="lifecycle-recursive-prefix" should-fail="false">
  <description>Lifecycle commands allow make's recursive '+' prefix</description>
  <input language="make"><![CDATA[
SCENARIO_NAME := $(notdir $(CURDIR))

start:
	@+echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@+vrooli scenario start $(SCENARIO_NAME)

stop:
	@+echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@+vrooli scenario stop $(SCENARIO_NAME)

test:
	@+echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@+vrooli scenario test $(SCENARIO_NAME)

logs:
	@+echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@+vrooli scenario logs $(SCENARIO_NAME) --tail 50

status:
	@+echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
	@+vrooli scenario status $(SCENARIO_NAME)
  ]]></input>
</test-case>

<test-case id="lifecycle-logs-tail-equals" should-fail="true">
  <description>Logs target uses non-canonical tail syntax</description>
  <input language="make"><![CDATA[
SCENARIO_NAME := $(notdir $(CURDIR))

start:
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME)

stop:
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

test:
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

logs:
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail=50

status:
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario status $(SCENARIO_NAME)
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>execute 'vrooli scenario logs $(SCENARIO_NAME) --tail 50'</expected-message>
</test-case>

*/

type MakefileLifecycleViolation struct {
	Severity       string
	Type           string
	Standard       string
	Message        string
	Recommendation string
	FilePath       string
	Line           int
	LineNumber     int
}

const lifecycleStandard = "configuration-v1"

type lifecycleMakefileData struct {
	lines   []string
	targets map[string][]string
}

var lifecycleTargetRegexp = regexp.MustCompile(`^([A-Za-z0-9_.-]+)\s*:(.*)$`)

// CheckMakefileLifecycle validates lifecycle targets adhere to the expected implementation.
func CheckMakefileLifecycle(content string, filepath string) ([]MakefileLifecycleViolation, error) {
	data := parseLifecycleMakefile(content)
	var violations []MakefileLifecycleViolation

	violations = append(violations, lifecycleValidateTarget(data, filepath, "start", canonicalMessageForTarget("start"), lifecycleMatchStartCommand)...)
	violations = append(violations, lifecycleValidateTarget(data, filepath, "stop", canonicalMessageForTarget("stop"), lifecycleMatchStopCommand)...)
	violations = append(violations, lifecycleValidateTarget(data, filepath, "test", canonicalMessageForTarget("test"), lifecycleMatchTestCommand)...)
	violations = append(violations, lifecycleValidateTarget(data, filepath, "logs", canonicalMessageForTarget("logs"), lifecycleMatchLogsCommand)...)
	violations = append(violations, lifecycleValidateTarget(data, filepath, "status", canonicalMessageForTarget("status"), lifecycleMatchStatusCommand)...)

	return violations, nil
}

func lifecycleValidateTarget(data lifecycleMakefileData, path, target string, canonicalMessage string, matcher lifecycleCommandMatcher) []MakefileLifecycleViolation {
	var violations []MakefileLifecycleViolation
	rawRecipe := data.targets[target]
	recipe := lifecycleNormalize(rawRecipe)
	canonical := canonicalCommandForTarget(target)
	if len(recipe) == 0 {
		line := lifecycleFindLine(data.lines, target+":")
		recommendation := fmt.Sprintf("Define the %s target with the canonical echo and '%s'.", target, canonical)
		violations = append(violations, newLifecycleViolation(
			path,
			line,
			target,
			fmt.Sprintf("%s target missing", target),
			"makefile_lifecycle_missing_target",
			recommendation,
		))
		return violations
	}

	expectedEcho := canonicalEchoForMessage(canonicalMessage)
	if ok, line, observed := lifecycleHasCanonicalMessage(data, rawRecipe, expectedEcho); !ok {
		recommendation := fmt.Sprintf("Replace the echo line with '%s'.", expectedEcho)
		if line == 0 {
			switch {
			case observed != "":
				line = lifecycleFindLine(data.lines, observed)
			default:
				line = lifecycleFindRecipeLineContaining(data, target, rawRecipe, "echo")
			}
		}

		message := fmt.Sprintf("%s target must echo '%s'", target, expectedEcho)
		if observed != "" {
			message = fmt.Sprintf("%s target must echo '%s' (found '%s')", target, expectedEcho, observed)
		}

		violations = append(violations, newLifecycleViolation(
			path,
			line,
			target,
			message,
			"makefile_lifecycle_message",
			recommendation,
		))
	}

	if matcher != nil {
		hasCommand, observed := lifecycleHasCommand(recipe, matcher)
		if !hasCommand {
			recommendation := fmt.Sprintf("Replace the command with '%s'.", canonical)

			line := lifecycleFindRecipeLineContaining(data, target, rawRecipe, "vrooli", "scenario")
			if observed != "" {
				if candidate := lifecycleFindLine(data.lines, observed); candidate != 1 {
					line = candidate
				}
			}

			message := fmt.Sprintf("%s target must execute '%s'", target, canonical)
			if observed != "" {
				message = fmt.Sprintf("%s target must execute '%s' (found '%s')", target, canonical, observed)
			}

			violations = append(violations, newLifecycleViolation(
				path,
				line,
				target,
				message,
				"makefile_lifecycle_command",
				recommendation,
			))
		}
	}

	return violations
}

type lifecycleCommandMatcher func(tokens []string) bool

func lifecycleHasCommand(recipe []string, matcher lifecycleCommandMatcher) (bool, string) {
	var observed string

	for _, line := range lifecycleFlattenCommands(recipe) {
		sanitized := lifecycleTrimRecipePrefixes(line)
		if sanitized == "" {
			continue
		}

		if strings.HasPrefix(sanitized, "#") {
			continue
		}

		tokens := lifecycleCommandTokens(sanitized)
		if len(tokens) == 0 {
			continue
		}

		lower := strings.ToLower(sanitized)
		if strings.Contains(lower, "vrooli") {
			observed = sanitized
		} else if observed == "" {
			observed = sanitized
		}
		if matcher(tokens) {
			return true, sanitized
		}
	}

	return false, observed
}

func lifecycleFlattenCommands(lines []string) []string {
	var flattened []string
	var builder strings.Builder

	flush := func() {
		if builder.Len() == 0 {
			return
		}
		flattened = append(flattened, strings.TrimSpace(builder.String()))
		builder.Reset()
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if builder.Len() > 0 {
			builder.WriteByte(' ')
		}

		current := trimmed
		if strings.HasSuffix(current, "\\") {
			current = strings.TrimSpace(strings.TrimSuffix(current, "\\"))
			builder.WriteString(current)
			continue
		}

		builder.WriteString(current)
		flush()
	}

	flush()
	return flattened
}

func lifecycleCommandTokens(line string) []string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil
	}

	for len(trimmed) > 0 {
		switch trimmed[0] {
		case '@', '-', '+':
			trimmed = strings.TrimSpace(trimmed[1:])
			continue
		}
		break
	}

	if trimmed == "" {
		return nil
	}

	tokens := strings.Fields(trimmed)
	if len(tokens) == 0 {
		return nil
	}

	idx := 0
	for idx < len(tokens) {
		tok := tokens[idx]
		if lifecycleLooksLikeAssignment(tok) {
			idx++
			continue
		}
		break
	}

	return tokens[idx:]
}

func lifecycleLooksLikeAssignment(token string) bool {
	if !strings.Contains(token, "=") {
		return false
	}
	if strings.HasPrefix(token, "--") {
		return false
	}
	return true
}

func canonicalMessageForTarget(target string) string {
	switch target {
	case "start":
		return "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	case "stop":
		return "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	case "test":
		return "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	case "logs":
		return "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	case "status":
		return "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
	default:
		return target
	}
}

func canonicalCommandForTarget(target string) string {
	switch target {
	case "start":
		return "vrooli scenario start $(SCENARIO_NAME)"
	case "stop":
		return "vrooli scenario stop $(SCENARIO_NAME)"
	case "test":
		return "vrooli scenario test $(SCENARIO_NAME)"
	case "logs":
		return "vrooli scenario logs $(SCENARIO_NAME) --tail 50"
	case "status":
		return "vrooli scenario status $(SCENARIO_NAME)"
	default:
		return target
	}
}

func lifecycleMatchStartCommand(tokens []string) bool {
	return lifecycleMatchScenarioVerb(tokens, "start")
}

func lifecycleMatchStopCommand(tokens []string) bool {
	return lifecycleMatchScenarioVerb(tokens, "stop")
}

func lifecycleMatchTestCommand(tokens []string) bool {
	return lifecycleMatchScenarioVerb(tokens, "test")
}

func lifecycleMatchStatusCommand(tokens []string) bool {
	return lifecycleMatchScenarioVerb(tokens, "status")
}

func lifecycleMatchLogsCommand(tokens []string) bool {
	if len(tokens) != 6 {
		return false
	}
	if tokens[0] != "vrooli" || tokens[1] != "scenario" || tokens[2] != "logs" {
		return false
	}
	if !lifecycleMatchesScenarioToken(tokens[3]) {
		return false
	}
	if tokens[4] != "--tail" {
		return false
	}
	return tokens[5] == "50"
}

func lifecycleMatchScenarioVerb(tokens []string, verb string) bool {
	if len(tokens) != 4 {
		return false
	}
	if tokens[0] != "vrooli" || tokens[1] != "scenario" || tokens[2] != verb {
		return false
	}
	if !lifecycleMatchesScenarioToken(tokens[3]) {
		return false
	}
	return true
}

func lifecycleMatchesScenarioToken(token string) bool {
	trimmed := strings.Trim(token, "\"'")
	return trimmed == "$(SCENARIO_NAME)"
}

func lifecycleHasCanonicalMessage(data lifecycleMakefileData, rawRecipe []string, expectedEcho string) (bool, int, string) {
	var firstEchoLine int
	var firstEchoText string

	for _, raw := range rawRecipe {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		sanitized := lifecycleTrimRecipePrefixes(trimmed)

		if strings.HasPrefix(sanitized, "echo ") && firstEchoText == "" {
			firstEchoText = sanitized
			firstEchoLine = lifecycleFindLine(data.lines, trimmed)
		}

		if sanitized == expectedEcho {
			return true, lifecycleFindLine(data.lines, trimmed), sanitized
		}
	}

	if firstEchoLine == 0 && firstEchoText != "" {
		firstEchoLine = lifecycleFindLine(data.lines, firstEchoText)
	}

	return false, firstEchoLine, firstEchoText
}

func canonicalEchoForMessage(message string) string {
	return fmt.Sprintf("echo \"%s\"", message)
}

func lifecycleTrimRecipePrefixes(line string) string {
	trimmed := strings.TrimSpace(line)
	for len(trimmed) > 0 {
		switch trimmed[0] {
		case '@', '+', '-':
			trimmed = strings.TrimSpace(trimmed[1:])
			continue
		}
		break
	}
	return trimmed
}

func lifecycleNormalize(lines []string) []string {
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		normalized = append(normalized, strings.TrimSpace(line))
	}
	return normalized
}

func lifecycleFindRecipeLineContaining(data lifecycleMakefileData, target string, rawRecipe []string, substrings ...string) int {
	for _, raw := range rawRecipe {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		matches := true
		for _, sub := range substrings {
			if !strings.Contains(trimmed, sub) {
				matches = false
				break
			}
		}
		if matches {
			return lifecycleFindLine(data.lines, trimmed)
		}
	}
	return lifecycleFindLine(data.lines, target+":")
}

func newLifecycleViolation(path string, line int, target, message, violationType, recommendation string) MakefileLifecycleViolation {
	if line <= 0 {
		line = 1
	}
	return MakefileLifecycleViolation{
		Severity:       "high",
		Type:           violationType,
		Standard:       lifecycleStandard,
		Message:        message,
		Recommendation: recommendation,
		FilePath:       path,
		Line:           line,
		LineNumber:     line,
	}
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

func lifecycleFindLine(lines []string, needle string) int {
	for idx, line := range lines {
		if strings.Contains(line, needle) {
			return idx + 1
		}
	}
	return 1
}

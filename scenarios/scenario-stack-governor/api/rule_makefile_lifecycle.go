package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

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

// RunMakefileLifecycle runs the lifecycle check rule against scenario Makefiles.
func RunMakefileLifecycle(ctx context.Context, repoRoot, scenarioName string) (result RuleResult) {
	start := time.Now()
	result = RuleResult{
		RuleID:    "MAKEFILE_LIFECYCLE",
		StartedAt: start,
	}
	defer func() {
		result.FinishedAt = time.Now()
		result.Passed = !hasActionableFindings(result.Findings)
	}()

	paths, err := locateMakefiles(repoRoot, scenarioName)
	if err != nil {
		result.Findings = append(result.Findings, Finding{
			Level:   "error",
			Message: fmt.Sprintf("failed to locate Makefiles: %v", err),
		})
		return result
	}
	if len(paths) == 0 {
		result.Findings = append(result.Findings, Finding{
			Level:   "warn",
			Message: "no Makefiles found",
		})
		return result
	}

	for _, path := range paths {
		scenSlug := filepath.Base(filepath.Dir(path))
		content, err := os.ReadFile(path)
		if err != nil {
			result.Findings = append(result.Findings, Finding{
				Level:        "error",
				Message:      fmt.Sprintf("failed to read %s: %v", path, err),
				ScenarioName: scenSlug,
				Evidence: []Evidence{
					{Type: "file", Ref: path},
				},
			})
			continue
		}

		violations, err := CheckMakefileLifecycle(string(content), path)
		if err != nil {
			result.Findings = append(result.Findings, Finding{
				Level:        "error",
				Message:      fmt.Sprintf("failed to check lifecycle in %s: %v", path, err),
				ScenarioName: scenSlug,
				Evidence: []Evidence{
					{Type: "file", Ref: path},
				},
			})
			continue
		}
		for _, v := range violations {
			result.Findings = append(result.Findings, Finding{
				Level:        "error",
				Message:      v.Message,
				ScenarioName: scenSlug,
				Evidence: []Evidence{
					{Type: "file", Ref: v.FilePath},
					{Type: "note", Detail: v.Recommendation},
				},
			})
		}
	}

	return result
}

// CheckMakefileLifecycle validates lifecycle targets adhere to the expected implementation.
//
// The rule only enforces that each lifecycle target ultimately invokes the canonical
// `vrooli scenario <verb>` command. User-facing output (banners, status messages,
// colors) is owned by the CLI itself, not by per-scenario Makefiles, so we no
// longer require a specific @echo line in the recipe.
func CheckMakefileLifecycle(content string, filepath string) ([]MakefileLifecycleViolation, error) {
	data := parseLifecycleMakefile(content)
	var violations []MakefileLifecycleViolation

	violations = append(violations, lifecycleValidateTarget(data, filepath, "start", lifecycleMatchStartCommand)...)
	violations = append(violations, lifecycleValidateTarget(data, filepath, "stop", lifecycleMatchStopCommand)...)
	violations = append(violations, lifecycleValidateTarget(data, filepath, "restart", lifecycleMatchRestartCommand)...)
	violations = append(violations, lifecycleValidateTarget(data, filepath, "test", lifecycleMatchTestCommand)...)
	violations = append(violations, lifecycleValidateTarget(data, filepath, "logs", lifecycleMatchLogsCommand)...)
	violations = append(violations, lifecycleValidateTarget(data, filepath, "status", lifecycleMatchStatusCommand)...)

	return violations, nil
}

func lifecycleValidateTarget(data lifecycleMakefileData, path, target string, matcher lifecycleCommandMatcher) []MakefileLifecycleViolation {
	var violations []MakefileLifecycleViolation
	rawRecipe := data.targets[target]
	recipe := lifecycleNormalize(rawRecipe)
	canonical := canonicalCommandForTarget(target)
	if len(recipe) == 0 {
		line := lifecycleFindLine(data.lines, target+":")
		recommendation := fmt.Sprintf("Define the %s target with '%s'.", target, canonical)
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

	if matcher == nil {
		return violations
	}

	hasCommand, observed := lifecycleHasCommand(recipe, matcher)
	if hasCommand {
		return violations
	}

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

func canonicalCommandForTarget(target string) string {
	switch target {
	case "start":
		return "vrooli scenario start $(SCENARIO_NAME)"
	case "stop":
		return "vrooli scenario stop $(SCENARIO_NAME)"
	case "restart":
		return "vrooli scenario restart $(SCENARIO_NAME)"
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

func lifecycleMatchRestartCommand(tokens []string) bool {
	return lifecycleMatchScenarioVerb(tokens, "restart")
}

func lifecycleMatchTestCommand(tokens []string) bool {
	return lifecycleMatchScenarioVerb(tokens, "test")
}

func lifecycleMatchStatusCommand(tokens []string) bool {
	return lifecycleMatchScenarioVerb(tokens, "status")
}

func lifecycleMatchLogsCommand(tokens []string) bool {
	// Require at least 5 tokens: vrooli scenario logs $(SCENARIO_NAME) --tail=50
	// or 6 tokens: vrooli scenario logs $(SCENARIO_NAME) --tail 50
	// Allow additional flags (e.g. --follow) after the required ones.
	if len(tokens) < 5 {
		return false
	}
	if tokens[0] != "vrooli" || tokens[1] != "scenario" || tokens[2] != "logs" {
		return false
	}
	if !lifecycleMatchesScenarioToken(tokens[3]) {
		return false
	}
	// Accept both "--tail 50" (two tokens) and "--tail=50" (one token).
	if tokens[4] == "--tail=50" {
		return true
	}
	if len(tokens) >= 6 && tokens[4] == "--tail" && tokens[5] == "50" {
		return true
	}
	return false
}

func lifecycleMatchScenarioVerb(tokens []string, verb string) bool {
	// Require at least 4 tokens: vrooli scenario <verb> $(SCENARIO_NAME)
	// Allow additional flags (e.g. --detach) after the required ones.
	if len(tokens) < 4 {
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
	return trimmed == "$(SCENARIO_NAME)" || trimmed == "${SCENARIO_NAME}"
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

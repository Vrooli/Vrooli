package services

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"app-monitor-api/logger"
)

// =============================================================================
// Log Fetching
// =============================================================================

func (s *AppService) runScenarioLogsCommand(ctx context.Context, appName string, extraArgs ...string) (string, error) {
	args := []string{"scenario", "logs", appName}
	if len(extraArgs) > 0 {
		args = append(args, extraArgs...)
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctxWithTimeout, "vrooli", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outStr := string(output)
		lower := strings.ToLower(outStr)
		if strings.Contains(lower, "not found") || strings.Contains(lower, "no such") {
			return outStr, nil
		}
		return "", fmt.Errorf("failed to execute %s: %w (output: %s)", strings.Join(cmd.Args, " "), err, strings.TrimSpace(outStr))
	}

	return string(output), nil
}

// GetAppLogs retrieves lifecycle and background logs for an application.
func (s *AppService) GetAppLogs(ctx context.Context, appName string, logType string) (*AppLogsResult, error) {
	normalized := strings.ToLower(strings.TrimSpace(logType))
	if normalized == "" || normalized == "all" {
		normalized = "both"
	}

	primaryOutput, err := s.runScenarioLogsCommand(ctx, appName)
	if err != nil {
		return nil, err
	}

	trimmedOutput := strings.TrimSpace(primaryOutput)
	lifecycle := make([]string, 0)
	if trimmedOutput != "" {
		lifecycle = strings.Split(trimmedOutput, "\n")
	}
	if len(lifecycle) == 0 {
		lifecycle = []string{fmt.Sprintf("No logs available for scenario '%s'", appName)}
	}

	result := &AppLogsResult{}
	if normalized != "background" {
		result.Lifecycle = lifecycle
	}

	if normalized == "lifecycle" {
		return result, nil
	}

	backgroundCandidates := parseBackgroundLogCandidates(primaryOutput)
	if len(backgroundCandidates) == 0 {
		return result, nil
	}

	backgroundLogs := make([]BackgroundLog, 0, len(backgroundCandidates))
	for _, candidate := range backgroundCandidates {
		stepOutput, stepErr := s.runScenarioLogsCommand(ctx, appName, "--step", candidate.Step)
		if stepErr != nil {
			logger.Warn(fmt.Sprintf("failed to fetch background logs for %s/%s", appName, candidate.Step), stepErr)
			continue
		}

		trimmed := strings.TrimSpace(stepOutput)
		lines := make([]string, 0)
		if trimmed != "" {
			lines = strings.Split(trimmed, "\n")
		}
		if len(lines) == 0 {
			lines = []string{fmt.Sprintf("No logs captured for background step '%s'", candidate.Step)}
		}

		backgroundLogs = append(backgroundLogs, BackgroundLog{
			Step:    candidate.Step,
			Phase:   candidate.Phase,
			Label:   candidate.Label,
			Command: candidate.Command,
			Lines:   lines,
		})
	}

	if len(backgroundLogs) > 0 {
		result.Background = backgroundLogs
	}

	return result, nil
}

// =============================================================================
// Background Log Parsing
// =============================================================================

func parseBackgroundLogCandidates(raw string) []backgroundLogCandidate {
	lines := strings.Split(raw, "\n")
	if len(lines) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	results := make([]backgroundLogCandidate, 0)

	inSection := false
	lastLabel := ""
	lastAvailable := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if strings.Contains(trimmed, "BACKGROUND STEP LOGS AVAILABLE") {
			inSection = true
			lastLabel = ""
			lastAvailable = false
			continue
		}

		if !inSection {
			continue
		}

		if strings.HasPrefix(trimmed, "💡") {
			break
		}

		switch {
		case strings.HasPrefix(trimmed, "✅"):
			lastLabel = trimmed
			lastAvailable = true
			continue
		case strings.HasPrefix(trimmed, "⚠️"), strings.HasPrefix(trimmed, "⚠"):
			lastLabel = trimmed
			lastAvailable = false
			continue
		case strings.HasPrefix(trimmed, "❌"):
			lastLabel = trimmed
			lastAvailable = false
			continue
		case strings.HasPrefix(trimmed, "View:"):
			matches := backgroundViewCommandRegex.FindStringSubmatch(trimmed)
			if len(matches) != 3 {
				continue
			}
			if !lastAvailable {
				continue
			}

			step := strings.TrimSpace(matches[2])
			if step == "" {
				continue
			}

			label, phase := normalizeBackgroundLabel(lastLabel)
			key := step + "|" + phase
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}

			candidate := backgroundLogCandidate{
				Step:    step,
				Phase:   phase,
				Label:   label,
				Command: fmt.Sprintf("vrooli scenario logs %s --step %s", strings.TrimSpace(matches[1]), step),
			}
			results = append(results, candidate)
		}
	}

	return results
}

func normalizeBackgroundLabel(raw string) (string, string) {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return "", ""
	}

	iconPrefixes := []string{"✅", "⚠️", "⚠", "❌", "•"}
	for _, prefix := range iconPrefixes {
		clean = strings.TrimPrefix(clean, prefix)
		clean = strings.TrimPrefix(clean, strings.TrimSpace(prefix))
	}
	clean = strings.TrimSpace(clean)

	if idx := strings.Index(clean, " - "); idx > -1 {
		clean = strings.TrimSpace(clean[:idx])
	}

	label := clean
	phase := ""
	if open := strings.LastIndex(clean, "("); open > -1 && strings.HasSuffix(clean, ")") {
		phase = strings.TrimSpace(clean[open+1 : len(clean)-1])
		name := strings.TrimSpace(clean[:open])
		if name != "" {
			label = fmt.Sprintf("%s (%s)", name, phase)
		}
	}

	return label, phase
}

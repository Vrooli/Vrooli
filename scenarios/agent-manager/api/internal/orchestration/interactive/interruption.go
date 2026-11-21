// Responsibility: classify resumable provider screen interruptions.
package interactive

import "strings"

// ResumableInterruption is a provider/runtime condition that stopped a turn
// but can be retried without creating a new Web Console session.
type ResumableInterruption struct {
	Kind    string
	Message string
}

// ClassifyScreenInterruption recognizes stable, user-visible provider failure
// shapes. Keep this table additive and provider-neutral: harness-specific
// screen readers can call it after obtaining their plain-text screen.
func ClassifyScreenInterruption(screen string) (ResumableInterruption, bool) {
	lower := strings.ToLower(screen)
	if hasCodexStalledGoalFooter(screen) {
		return ResumableInterruption{Kind: "goal_stalled", Message: "Goal stalled (/goal resume)"}, true
	}
	markers := []struct {
		kind   string
		marker string
	}{
		{"model_capacity", "selected model is at capacity"},
		{"model_capacity", "model is at capacity"},
		{"model_capacity", "temporarily at capacity"},
		{"rate_limit", "rate limit exceeded"},
		{"rate_limit", "too many requests"},
		{"provider_unavailable", "service unavailable"},
		{"provider_unavailable", "temporarily unavailable"},
	}
	for _, candidate := range markers {
		if strings.Contains(lower, candidate.marker) {
			return ResumableInterruption{Kind: candidate.kind, Message: candidate.marker}, true
		}
	}
	return ResumableInterruption{}, false
}

// hasCodexStalledGoalFooter recognizes the actionable status at the bottom of
// the current Codex screen. The terminal screen may include scrollback, so a
// matching phrase in earlier output must not schedule a resume. Require the
// prompt, blank separator, stalled status with its explicit resume command,
// and shortcuts footer as the final nonblank screen rows.
func hasCodexStalledGoalFooter(screen string) bool {
	lines := strings.Split(strings.ReplaceAll(screen, "\r\n", "\n"), "\n")
	last := len(lines) - 1
	for last >= 0 && strings.TrimSpace(lines[last]) == "" {
		last--
	}
	if last < 3 {
		return false
	}

	prompt := codexMainPaneText(lines[last-3])
	separator := codexMainPaneText(lines[last-2])
	status := strings.ToLower(codexMainPaneText(lines[last-1]))
	shortcuts := strings.ToLower(codexMainPaneText(lines[last]))
	marker := "goal stalled (/goal resume)"
	return strings.HasPrefix(prompt, "›") && separator == "" &&
		strings.HasSuffix(status, marker) && len(status) > len(marker) &&
		strings.HasPrefix(shortcuts, "? for shortcuts")
}

func codexMainPaneText(line string) string {
	if divider := strings.IndexRune(line, '│'); divider >= 0 {
		line = line[:divider]
	}
	return strings.TrimSpace(line)
}

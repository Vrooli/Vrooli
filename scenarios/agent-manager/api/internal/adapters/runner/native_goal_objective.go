package runner

import "strings"

// NativeGoalObjective applies the exact one-line transform used when installing a native goal.
// It replaces only CRLF, LF, and CR line breaks with spaces, then trims the ends.
func NativeGoalObjective(objective string) string {
	objective = strings.ReplaceAll(objective, "\r\n", " ")
	objective = strings.ReplaceAll(objective, "\n", " ")
	objective = strings.ReplaceAll(objective, "\r", " ")
	return strings.TrimSpace(objective)
}

// NativeGoalObjectiveMatches correlates observed native objective text to the configured objective.
// The configured text may be observed raw or after NativeGoalObjective, and either form may have
// the existing newline-delimited prompt suffix. Observed interior whitespace is never normalized.
func NativeGoalObjectiveMatches(observed, configured string) bool {
	observed = strings.TrimSpace(observed)
	raw := strings.TrimSpace(configured)
	installed := NativeGoalObjective(configured)
	if observed == "" || (raw == "" && installed == "") {
		return false
	}
	for _, expected := range []string{installed, raw} {
		if expected != "" && (observed == expected || strings.HasPrefix(observed, expected+"\n")) {
			return true
		}
	}
	return false
}

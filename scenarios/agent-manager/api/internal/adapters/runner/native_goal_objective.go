package runner

import (
	"strings"

	"agent-manager/internal/domain"
)

// NativeGoalDelegation is the native goal Agent Manager installs when a run's
// `until` is longer than its harness keeps verbatim. The task prompt then
// carries the full `until` as its engine-owned completion contract.
const NativeGoalDelegation = "Follow the complete engine-owned completion contract included in the task prompt exactly. Do not stop until it is satisfied; return blocked, abstained, or an operator-decision approval request when required."

// NativeGoalVerbatimLimit is the longest `until`, in characters, that Agent
// Manager installs verbatim as the harness's native goal. A longer `until`
// gets NativeGoalDelegation instead. Zero means the harness has no measured
// limit and always gets the delegation.
//
// The limit is what survives the harness's own context management, not only
// what its goal field accepts:
//   - Codex keeps a native goal's objective through compaction only when it is
//     short; a longer one becomes "User set the goal: [objective omitted;
//     exceeds the evidence limit]" until the next goal-turn start. Across
//     every October–November 2026 Codex rollout on the reference host, the
//     longest retained objective had 669 characters and the shortest omitted
//     one 711 (268 compactions; docs/agent-system/LONG_EFFORT_DESIGN.md §4a).
//     600 is the kernel bound that document's rule D3 sets, safely below both.
//   - Claude Code rejects /goal conditions over 4,000 characters, and its
//     separate evaluator keeps the condition through compaction.
func NativeGoalVerbatimLimit(runnerType domain.RunnerType) int {
	switch runnerType {
	case domain.RunnerTypeCodex:
		return 600
	case domain.RunnerTypeClaudeCode:
		return 4000
	default:
		return 0
	}
}

// NativeGoalObjectiveFor is the native goal objective Agent Manager installs
// for a run of runnerType with this `until`: the `until` itself when it fits
// the harness's verbatim limit, otherwise NativeGoalDelegation.
func NativeGoalObjectiveFor(runnerType domain.RunnerType, until string) string {
	until = strings.TrimSpace(until)
	if until == "" {
		return ""
	}
	if len([]rune(until)) <= NativeGoalVerbatimLimit(runnerType) {
		return until
	}
	return NativeGoalDelegation
}

// NativeGoalObjective applies the exact one-line transform used when installing a native goal.
// It replaces only CRLF, LF, and CR line breaks with spaces, then trims the ends.
func NativeGoalObjective(objective string) string {
	objective = strings.ReplaceAll(objective, "\r\n", " ")
	objective = strings.ReplaceAll(objective, "\n", " ")
	objective = strings.ReplaceAll(objective, "\r", " ")
	return strings.TrimSpace(objective)
}

// NativeGoalObjectiveMatches correlates an observed native objective with a
// run's configured `until`. The `until` may be observed raw or after
// NativeGoalObjective, and either form may carry the existing
// newline-delimited prompt suffix. Observed interior whitespace is never
// normalized. When the `until` is longer than the harness's verbatim limit,
// the delegation Agent Manager installed in its place matches too; the full
// `until` still matches so that sessions launched under an earlier, longer
// limit keep correlating.
func NativeGoalObjectiveMatches(runnerType domain.RunnerType, observed, configured string) bool {
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
	return observed == NativeGoalDelegation && NativeGoalObjectiveFor(runnerType, raw) == NativeGoalDelegation
}

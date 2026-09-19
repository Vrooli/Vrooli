package report

import (
	"fmt"
	"strings"

	execTypes "test-genie/cli/internal/execute"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// maxScorecardGaps bounds how many blocking codes the concise inline scorecard
// lists; the full set is in --json and `runs findings`.
const maxScorecardGaps = 2

// PrintPhaseStanding renders the concise per-phase maturity scorecard (Phase
// Capability Contract) beneath a phase's result line. It is a tight, bounded
// block: rung/ceiling, top gaps, the single highest-unlock next move, and a
// runnable doc-search line. At maximum maturity the next-move and doc lines are
// suppressed. No-op for phases with no standing (native / degraded), which keep
// their plain status row.
func (p *Printer) PrintPhaseStanding(phase execTypes.Phase) {
	st := phase.PhasePresentation
	if st == nil {
		if phase.PresentationState == "legacy_maturity_standing" {
			fmt.Fprintln(p.w, "     presentation: historical maturity standing (not canonical v1)")
		}
		return
	}
	fmt.Fprintf(p.w, "     %s %s\n", p.color.Bold("standing:"), scorecardRung(st))
	if ns := strings.TrimSpace(st.GetNorthStar()); ns != "" {
		fmt.Fprintf(p.w, "     North Star: %s\n", ns)
	}
	if st.GetAtMaximum() {
		return
	}
	if gaps := topGaps(st.GetBlockingFindingCodes(), maxScorecardGaps); gaps != "" {
		fmt.Fprintf(p.w, "     gaps: %s\n", gaps)
	}
	if move := strings.TrimSpace(st.GetNextAction()); move != "" {
		line := "     next: " + move
		if capLabel := strings.TrimSpace(st.GetFocusCapabilityLabel()); capLabel != "" {
			line += "  [→ " + capLabel + "]"
		}
		fmt.Fprintln(p.w, line)
	}
	if len(st.GetDocumentationTopics()) > 0 {
		fmt.Fprintf(p.w, "     docs: search-hub query %q --type doc\n", st.GetDocumentationTopics()[0])
	}
}

func (p *Printer) printTopPriority(priority *execTypes.RunTopPriority) {
	if priority == nil {
		return
	}
	fmt.Fprintln(p.w, p.color.Bold("RUN TOP PRIORITY:"))
	line := fmt.Sprintf("  • %s: %s", priority.Phase, priority.NextMove)
	if capLabel := strings.TrimSpace(priority.PriorityCapabilityLabel); capLabel != "" {
		line += "  [→ " + capLabel + "]"
	}
	fmt.Fprintln(p.w, line)
	if topic := strings.TrimSpace(priority.DocSearchTopic); topic != "" {
		fmt.Fprintf(p.w, "  • docs: search-hub query %q --type doc\n", topic)
	}
	fmt.Fprintln(p.w)
}

// scorecardRung renders the "current → next (ceiling X)" rung line, collapsing to
// a single rung at maximum maturity.
func scorecardRung(st *commonv1.PhasePresentation) string {
	cur := firstNonEmpty(st.GetCurrentLevel(), "?")
	if label := strings.TrimSpace(st.GetCurrentLevelLabel()); label != "" {
		cur = fmt.Sprintf("%s %s", cur, label)
	}
	if st.GetAtMaximum() {
		return cur + " · maximum maturity"
	}
	next := strings.TrimSpace(st.GetNextLevel())
	if next == "" {
		return cur
	}
	line := fmt.Sprintf("%s → %s", cur, next)
	if ceil := strings.TrimSpace(st.GetCeilingLevel()); ceil != "" && ceil != next {
		line += fmt.Sprintf(" (ceiling %s)", ceil)
	}
	return line
}

func topGaps(codes []string, limit int) string {
	if len(codes) == 0 {
		return ""
	}
	seen := map[string]bool{}
	out := make([]string, 0, limit)
	for _, c := range codes {
		c = strings.TrimSpace(c)
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)
		if len(out) >= limit {
			break
		}
	}
	joined := strings.Join(out, ", ")
	if remaining := len(codes) - len(out); remaining > 0 {
		joined += fmt.Sprintf(" (+%d more)", remaining)
	}
	return joined
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

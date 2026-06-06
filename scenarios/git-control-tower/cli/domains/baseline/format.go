package baseline

import (
	"fmt"
	"sort"
	"strings"

	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
)

// verdictMark returns a glyph for a surface/overall verdict.
func verdictMark(verdict string) string {
	switch verdict {
	case "clean":
		return "✓"
	case "regression", "new-failure":
		return "✗"
	case "preexisting":
		return "•"
	case "not-comparable":
		return "?"
	}
	return "-"
}

func gitLine(g *baselinesv1.GitState) string {
	if g == nil {
		return ""
	}
	sha := g.GetSha()
	if len(sha) > 8 {
		sha = sha[:8]
	}
	branch := g.GetBranch()
	if g.GetDetached() {
		branch = "(detached)"
	}
	line := fmt.Sprintf("branch: %s   sha: %s", branch, sha)
	if g.GetDirty() {
		line += fmt.Sprintf("   dirty: %s", g.GetDirtySummary())
	}
	return line
}

func printSnapshot(resp *baselinesv1.SnapshotForBaselineResponse) {
	b := resp.GetBaseline()
	fmt.Printf("✓ Captured baseline %q for %s\n", b.GetName(), b.GetScenario())
	fmt.Printf("  %s\n", gitLine(b.GetGit()))
	if len(b.GetSurfaces()) > 0 {
		var parts []string
		for _, id := range surfaceIDsSorted(b.GetSurfaces()) {
			p := b.GetSurfaces()[id]
			parts = append(parts, fmt.Sprintf("%s(%s)", id, summaryText(p.GetSummary())))
		}
		fmt.Printf("  surfaces: %s\n", strings.Join(parts, " "))
	}
	for _, surface := range stringKeysSorted(resp.GetSkipped()) {
		fmt.Printf("  ⚠ skipped %s: %s\n", surface, resp.GetSkipped()[surface])
	}
	if w := resp.GetDirtyWarning(); w != "" {
		fmt.Printf("⚠ %s\n", w)
	}
}

func printDiff(resp *baselinesv1.DiffBaselineResponse) {
	b := resp.GetBaseline()
	fmt.Printf("Baseline: %s   captured %s\n", b.GetName(), b.GetCreatedAt())
	if cg := resp.GetCurrentGit(); cg != nil {
		sha := cg.GetSha()
		if len(sha) > 8 {
			sha = sha[:8]
		}
		if st := resp.GetStaleness(); st != nil && (st.GetCommitsSince() > 0 || st.GetFilesChanged() > 0) {
			stale := ""
			if st.GetLikelyStale() {
				stale = " (likely stale)"
			}
			fmt.Printf("Working tree: sha=%s (+%d commits, %d files changed since baseline)%s\n", sha, st.GetCommitsSince(), st.GetFilesChanged(), stale)
		} else {
			fmt.Printf("Working tree: sha=%s\n", sha)
		}
	}
	fmt.Println()
	for _, s := range resp.GetSurfaces() {
		fmt.Printf("%-10s %s %s\n", s.GetSurfaceId(), verdictMark(s.GetVerdict()), s.GetSummary())
		printLines("regression", s.GetRegressions())
		printLines("new", s.GetNewFailures())
		printLines("preexisting", s.GetPreexisting())
		printLines("cleared", s.GetCleared())
	}
	fmt.Println()
	if w := resp.GetDirtyWarning(); w != "" {
		fmt.Printf("⚠ %s\n", w)
	}
	fmt.Printf("Overall: %s %s\n", verdictMark(resp.GetVerdict()), resp.GetVerdict())
}

func printLines(label string, lines []string) {
	for _, l := range lines {
		fmt.Printf("            %s: %s\n", label, l)
	}
}

func printList(baselines []*baselinesv1.BaselineManifest) {
	if len(baselines) == 0 {
		fmt.Println("No baselines found.")
		return
	}
	fmt.Printf("Baselines (%d):\n", len(baselines))
	for _, b := range baselines {
		skipped := ""
		if n := len(b.GetSkipped()); n > 0 {
			skipped = fmt.Sprintf(" skipped=%d", n)
		}
		fmt.Printf("  %-20s branch=%-12s surfaces=%d%s  %s\n", b.GetName(), b.GetBranch(), len(b.GetSurfaces()), skipped, b.GetCreatedAt())
	}
}

func printShow(b *baselinesv1.BaselineManifest) {
	fmt.Printf("Baseline: %s\n", b.GetName())
	fmt.Printf("  scenario:   %s\n", b.GetScenario())
	fmt.Printf("  branch:     %s\n", b.GetBranch())
	fmt.Printf("  created:    %s  by %s\n", b.GetCreatedAt(), b.GetCreatedBy())
	if g := b.GetGit(); g != nil {
		fmt.Printf("  %s\n", gitLine(g))
		if g.GetDirty() {
			fmt.Printf("  ⚠ captured against dirty tree\n")
		}
	}
	fmt.Println("  surfaces:")
	for _, id := range surfaceIDsSorted(b.GetSurfaces()) {
		p := b.GetSurfaces()[id]
		fmt.Printf("    %-10s kind=%s ref=%s  %s\n", id, p.GetKind(), p.GetRef(), summaryText(p.GetSummary()))
	}
	if skipped := b.GetSkipped(); len(skipped) > 0 {
		fmt.Println("  not captured (this baseline cannot speak to these surfaces):")
		for _, id := range stringKeysSorted(skipped) {
			fmt.Printf("    ⚠ %-10s %s\n", id, skipped[id])
		}
	}
}

// stringKeysSorted returns a map's keys in stable sorted order.
func stringKeysSorted(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// summaryText renders the compact per-surface JSON summary as a flat string.
func summaryText(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "{")
	s = strings.TrimSuffix(s, "}")
	s = strings.ReplaceAll(s, "\"", "")
	return strings.TrimSpace(s)
}

// printStart renders the engagement mode header (Target End State §6): the mode,
// the resolved target, the available next verbs, the ambient routing var, the
// human-friendly TTL, and the decision-tree reasoning.
func printStart(res startResult) {
	target := res.Scenario
	header := fmt.Sprintf("Mode: %s · Target: %s", strings.ToUpper(res.Decision.Mode), target)
	if res.Variant != "" && res.Variant != "live" {
		header = fmt.Sprintf("Mode: %s · Target: %s@%s (db vrooli_%s_%s)",
			strings.ToUpper(res.Decision.Mode), res.Scenario, res.Variant, res.Scenario, res.Variant)
	}
	header += " · Available: " + strings.Join(res.Available, ", ")
	fmt.Println(header)
	if res.AmbientVar != "" {
		fmt.Printf("  ambient routing: VROOLI_SHADOW_SCENARIOS=%s (nested CLI calls auto-target the shadow)\n", res.AmbientVar)
	}
	if res.Anchor != "" {
		fmt.Printf("  diff anchor: %s\n", res.Anchor)
	}
	for _, note := range res.DataPopulation {
		fmt.Printf("  shadow data: %s\n", note)
	}
	if res.TTL != "" {
		fmt.Printf("  auto-cleanup in ~%s; change with: baseline status (set-ttl)\n", res.TTL)
	} else {
		fmt.Println("  no idle TTL (orchestrator-heartbeat mode)")
	}
	fmt.Println("  decision:")
	for _, r := range res.Decision.Reasons {
		fmt.Printf("    - %s\n", r)
	}
}

// printCheck renders a check result: the verdict glyph plus the mode-aware guidance.
func printCheck(res checkResult) {
	fmt.Printf("%s %s/%s [%s] verdict: %s\n", verdictMark(res.Verdict), res.Scenario, res.Slug, res.Mode, res.Verdict)
	fmt.Printf("  → %s\n", res.Guidance)
}

// printPromote renders a promote outcome: the headline verdict, the ordered
// step trace, and (on rollback) the data-snapshot pointer for manual recovery.
func printPromote(res promoteResult) {
	mark := "✓"
	if !res.Promoted {
		mark = "✗"
	}
	fmt.Printf("%s %s/%s [%s] %s\n", mark, res.Scenario, res.Slug, res.Mode, res.Message)
	for _, s := range res.Steps {
		fmt.Printf("  · %s\n", s)
	}
	if res.RolledBack && res.DataSnapshot != "" {
		fmt.Printf("  data snapshot for manual restore: %s\n", res.DataSnapshot)
	}
}

// printStatus renders the active engagements (globbed from the floor manifests).
func printStatus(engagements []engagementView) {
	if len(engagements) == 0 {
		fmt.Println("no active engagements")
		return
	}
	for _, e := range engagements {
		expiry := "never (idle)"
		if e.ExpiresAt != nil {
			expiry = e.ExpiresAt.Format("2006-01-02T15:04:05Z07:00")
			if e.Expired {
				expiry += " (EXPIRED)"
			}
		}
		target := e.Scenario
		if e.Variant != "" && e.Variant != "live" {
			target = e.Scenario + "@" + e.Variant
		}
		fmt.Printf("%s/%s\t%s\ttarget=%s\tttl=%s\texpires=%s\n", e.Scenario, e.Slug, e.Mode, target, e.TTL, expiry)
	}
}

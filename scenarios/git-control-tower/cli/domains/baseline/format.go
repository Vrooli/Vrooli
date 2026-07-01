package baseline

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
)

const (
	waitTimeoutBufferPercent  = 75
	minRecommendedWaitSeconds = 120
	unknownETAWaitSeconds     = 900
)

func recommendedWaitSeconds(eta int32, etaKnown bool) int {
	if !etaKnown || eta <= 0 {
		return unknownETAWaitSeconds
	}
	withBuffer := int(eta) + (int(eta)*waitTimeoutBufferPercent)/100
	if withBuffer < minRecommendedWaitSeconds {
		return minRecommendedWaitSeconds
	}
	return withBuffer
}

func waitCommand(scenario, run string, seconds int) string {
	if seconds <= 0 {
		return "test-genie runs wait --json " + scenario + " " + run
	}
	return "test-genie runs wait --json --timeout=" + strconv.Itoa(seconds) + " " + scenario + " " + run
}

func agentWaitBlock(scenario, run string, eta int32, etaKnown bool) string {
	waitSeconds := recommendedWaitSeconds(eta, etaKnown)
	etaStr := "unknown"
	if etaKnown {
		etaStr = (time.Duration(eta) * time.Second).String()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "  Agent wait protocol:\n")
	fmt.Fprintf(&b, "    Run exactly once:\n      %s\n", waitCommand(scenario, run, waitSeconds))
	fmt.Fprintf(&b, "    Expected duration: ~%s; recommended wait timeout: %s.\n", etaStr, (time.Duration(waitSeconds) * time.Second).String())
	fmt.Fprintf(&b, "    In coding-agent tool execution, give the command at least this timeout and do not poll with short output checks.\n")
	fmt.Fprintf(&b, "    If a wait process was already started and then interrupted:\n")
	fmt.Fprintf(&b, "      pgrep -af 'test-genie runs wait --json .* %s %s'\n", scenario, run)
	fmt.Fprintf(&b, "      tail --pid=<pid> -f /dev/null\n")
	return b.String()
}

func snapshotStatusWaitBlock(scenario, name, run string, eta int32, etaKnown bool) string {
	waitSeconds := recommendedWaitSeconds(eta, etaKnown)
	etaStr := "unknown"
	if etaKnown {
		etaStr = (time.Duration(eta) * time.Second).String()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "  Agent wait protocol:\n")
	fmt.Fprintf(&b, "    Run exactly once:\n      git-control-tower baseline snapshot status --scenario %s --name %s --run %s --wait --json\n", scenario, name, run)
	fmt.Fprintf(&b, "    Expected duration: ~%s; recommended client timeout: %s.\n", etaStr, (time.Duration(waitSeconds) * time.Second).String())
	fmt.Fprintf(&b, "    Raw run diagnostics, if needed:\n      %s\n", waitCommand(scenario, run, waitSeconds))
	fmt.Fprintf(&b, "      test-genie runs follow %s %s\n", scenario, run)
	return b.String()
}

// verdictMark returns a glyph for a surface/overall verdict.
func verdictMark(verdict string) string {
	switch verdict {
	case "clean":
		return "✓"
	case "changed":
		return "≈"
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

// printSnapshot renders the snapshot START result: the run handle + ETA + the
// streaming follow command. The baseline pins server-side when the run
// completes, so the snapshot returns fast and the operator follows the durable
// run by id (anti-polling: no silent block).
func printSnapshot(resp *baselinesv1.SnapshotForBaselineResponse) {
	fmt.Print(snapshotBanner(resp))
}

// snapshotBanner builds the snapshot-start banner. Pure (no I/O) so the
// anti-polling contract — an up-front run id + ETA + a streaming follow command,
// never a silent block — is unit-testable.
func snapshotBanner(resp *baselinesv1.SnapshotForBaselineResponse) string {
	eta := "unknown"
	if resp.GetEtaKnown() {
		eta = (time.Duration(resp.GetEstimatedTotalSeconds()) * time.Second).String()
	}
	var b strings.Builder
	if resp.GetCoalesced() {
		fmt.Fprintf(&b, "▶ Baseline %q for %s — re-using in-flight comprehensive run %s for %s (no new suite)\n", resp.GetName(), resp.GetScenario(), resp.GetRunId(), resp.GetScenario())
	} else {
		fmt.Fprintf(&b, "▶ Baseline %q for %s — comprehensive run %s started\n", resp.GetName(), resp.GetScenario(), resp.GetRunId())
	}
	fmt.Fprintf(&b, "  estimated %s — the run is durable server-side; the baseline pins automatically when it completes.\n", eta)
	b.WriteString(snapshotStatusWaitBlock(resp.GetScenario(), resp.GetName(), resp.GetRunId(), resp.GetEstimatedTotalSeconds(), resp.GetEtaKnown()))
	fmt.Fprintf(&b, "  then inspect:  git-control-tower baseline snapshot status --scenario %s --name %s --run %s\n", resp.GetScenario(), resp.GetName(), resp.GetRunId())
	if w := resp.GetDirtyWarning(); w != "" {
		fmt.Fprintf(&b, "⚠ %s\n", w)
	}
	return b.String()
}

// printDiffStart renders the durable-diff START banner: the run handle + ETA +
// the streaming follow command + the resolve command, with an outcome-specific
// header (fresh / coalesced onto an in-flight run / reused a completed run). The
// diff returns fast and the verdict is computed + cached server-side — no client
// polling (the anti-polling contract, mirror of snapshotBanner).
func printDiffStart(resp *baselinesv1.StartDiffResponse) {
	fmt.Print(diffStartBanner(resp))
}

// diffStartBanner builds the start banner. Pure (no I/O) so the anti-polling
// contract is unit-testable.
func diffStartBanner(resp *baselinesv1.StartDiffResponse) string {
	scenario, name, run := resp.GetScenario(), resp.GetName(), resp.GetRunId()
	var b strings.Builder
	switch {
	case resp.GetReusedRun():
		fmt.Fprintf(&b, "▶ Diff of baseline %q for %s — re-using completed run %s at %s (no suite re-run)\n", name, scenario, run, resp.GetReusedSha())
	case resp.GetCoalesced():
		fmt.Fprintf(&b, "▶ Diff of baseline %q for %s — re-using in-flight comprehensive run %s for %s (no new suite)\n", name, scenario, run, scenario)
	default:
		eta := "unknown"
		if resp.GetEtaKnown() {
			eta = (time.Duration(resp.GetEstimatedTotalSeconds()) * time.Second).String()
		}
		fmt.Fprintf(&b, "▶ Diff of baseline %q for %s — comprehensive run %s started (estimated %s)\n", name, scenario, run, eta)
	}
	fmt.Fprintf(&b, "  the run is durable server-side; the diff verdict is computed and cached when it completes.\n")
	b.WriteString(agentWaitBlock(scenario, run, resp.GetEstimatedTotalSeconds(), resp.GetEtaKnown()))
	fmt.Fprintf(&b, "  watch live:    test-genie runs follow %s %s\n", scenario, run)
	fmt.Fprintf(&b, "  then resolve:  git-control-tower baseline diff status --scenario %s --name %s --run %s\n", scenario, name, run)
	if w := resp.GetDirtyWarning(); w != "" {
		fmt.Fprintf(&b, "⚠ %s\n", w)
	}
	return b.String()
}

// printDiffPending renders the in-flight guidance when `diff status` is called
// before the run is terminal: the follow/resolve commands + a backoff hint.
func printDiffPending(scenario, name, run string, nextCheckSeconds int) {
	fmt.Printf("⏳ Diff of baseline %q for %s is still computing on run %s.\n", name, scenario, run)
	fmt.Printf("  block on it:   git-control-tower baseline diff status --scenario %s --name %s --run %s --wait\n", scenario, name, run)
	fmt.Printf("  (watch live:   test-genie runs follow %s %s)\n", scenario, run)
	if nextCheckSeconds > 0 {
		fmt.Printf("  if you must re-check instead, wait ~%ds (do not poll faster)\n", nextCheckSeconds)
	}
}

func printSnapshotStatus(resp *baselinesv1.GetSnapshotStatusResponse) {
	fmt.Printf("Snapshot: %s/%s", resp.GetScenario(), resp.GetName())
	if resp.GetBranch() != "" {
		fmt.Printf(" branch=%s", resp.GetBranch())
	}
	if resp.GetRunId() != "" {
		fmt.Printf(" run=%s", resp.GetRunId())
	}
	fmt.Printf(" status=%s", resp.GetStatus())
	if resp.GetRunStatus() != "" {
		fmt.Printf(" run_status=%s", resp.GetRunStatus())
	}
	fmt.Println()
	if b := resp.GetBaseline(); b != nil {
		fmt.Printf("  baseline ready: git-control-tower baseline show --scenario %s --name %s --branch %s\n", b.GetScenario(), b.GetName(), b.GetBranch())
	}
	printSnapshotStatusDiagnostics(os.Stdout, resp)
	if resp.GetStatus() == "pending" {
		if n := resp.GetRecommendedNextCheckSeconds(); n > 0 {
			fmt.Printf("  still running; if you re-check manually, wait ~%ds first\n", n)
		}
		fmt.Printf("  block: git-control-tower baseline snapshot status --scenario %s --name %s --run %s --wait\n", resp.GetScenario(), resp.GetName(), resp.GetRunId())
	}
}

func printSnapshotStatusDiagnostics(w io.Writer, resp *baselinesv1.GetSnapshotStatusResponse) {
	if resp == nil {
		return
	}
	if errText := resp.GetError(); errText != "" {
		fmt.Fprintf(w, "  detail: %s\n", errText)
	}
	if names := resp.GetSimilarBaselines(); len(names) > 0 {
		fmt.Fprintf(w, "  similar baselines: %s\n", strings.Join(names, ", "))
	}
}

func printDiff(resp *baselinesv1.DiffResult) {
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
		printLines("changed — review", s.GetChanged())
	}
	if len(resp.GetPhases()) > 0 {
		fmt.Println()
		fmt.Println("Phases:")
		for _, p := range resp.GetPhases() {
			label := p.GetPhase()
			if p.GetSurfaceId() != "" {
				label += " (" + p.GetSurfaceId() + ")"
			}
			fmt.Printf("  %-24s %s %s\n", label, verdictMark(p.GetVerdict()), p.GetSummary())
			printLines("regression", p.GetRegressions())
			printLines("new", p.GetNewFailures())
			printLines("preexisting", p.GetPreexisting())
			printLines("cleared", p.GetCleared())
		}
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

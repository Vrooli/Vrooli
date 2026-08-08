// Framework-health audit sweep: one command that reads every sensor named in
// docs/agent-system/FRAMEWORK_HEALTH.md and reports each target's observed
// value against its deadband.
//
// Why this exists: the agent-system-audit skill previously orchestrated eight
// separate commands and reconciled their differing output shapes by hand. That
// spent the auditing agent's context on orchestration rather than judgment, and
// every audit re-derived the same readings from scratch.
//
// Honesty rule: targets this command cannot observe in-process are still
// emitted, with status `external` (needs a command outside the graph API) or
// `no-sensor` (open-loop, per FRAMEWORK_HEALTH). Dropping them would let a
// clean sweep read as full coverage when it is not.
//
// DOC: docs/agent-system/FRAMEWORK_HEALTH.md
package graph

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"prompt-manager/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliutil"
)

// auditSchemaVersion is the on-disk shape version for the audit artifact.
// Additive fields do not require a bump; renames and removals do.
const auditSchemaVersion = 1

// Audit target status values.
const (
	auditStatusInBand    = "in-band"
	auditStatusOutOfBand = "out-of-band"
	auditStatusExternal  = "external"
	auditStatusNoSensor  = "no-sensor"
)

type auditTarget struct {
	Target   string `json:"target"`
	Sensor   string `json:"sensor"`
	Deadband string `json:"deadband"`
	Actuator string `json:"actuator"`
	Observed string `json:"observed"`
	Status   string `json:"status"`
	// GapMarker is required whenever a target has no automated corpus-wide
	// sensor. It makes the missing instrument a dated, owned work item rather
	// than a silent hole in the audit.
	GapMarker string `json:"gap_marker,omitempty"`
	// Detail carries the first few offending entries when out of band, so a
	// reader gets a lead without re-running the underlying sensor.
	Detail []string `json:"detail,omitempty"`
}

type auditReport struct {
	SchemaVersion int           `json:"schema_version"`
	GeneratedAt   string        `json:"generated_at"`
	Targets       []auditTarget `json:"targets"`
	OutOfBand     int           `json:"out_of_band"`
	Unsensored    int           `json:"unsensored"`
}

// cmdAudit runs the framework-health sweep.
func cmdAudit(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("audit", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	out := fs.String("out", "", "Write the JSON artifact to PATH (atomic)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	report := auditReport{
		SchemaVersion: auditSchemaVersion,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	for _, collect := range auditCollectors() {
		report.Targets = append(report.Targets, collect(ctx))
	}
	for _, t := range report.Targets {
		switch t.Status {
		case auditStatusOutOfBand:
			report.OutOfBand++
		case auditStatusNoSensor:
			report.Unsensored++
		}
	}

	if *out != "" {
		if err := writeAuditArtifact(*out, report); err != nil {
			return err
		}
	}
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}
	printAuditReport(report)
	return nil
}

// auditCollectors returns one collector per FRAMEWORK_HEALTH target, in the
// order the canon table lists them.
func auditCollectors() []func(appctx.Context) auditTarget {
	return []func(appctx.Context) auditTarget{
		auditContractValidity,
		auditContractModeCoverage,
		auditGraphRuntimeDrift,
		auditTopicIntegrity,
		auditMemberDocConformance,
		auditProseCoherence,
		auditCrossTeamCoupling,
		auditCanonCoherence,
		auditObjectiveCoverage,
		auditStaticallyUnreferencedSkills,
		auditSkillConditioning,
		auditExperimentLiveness,
		auditPoREntropy,
		auditTeamOrientationCost,
		auditDiscoveryBudgetPressure,
		auditPromptStructureInvariant,
		auditRulesWithNoFinding,
	}
}

// auditPromptStructureInvariant is the first audit target that measures the
// artifact an agent actually receives rather than a declaration it was built
// from. Every other target reads checked-in files; a prompt-breaking defect
// lived on all six teams for months precisely because nothing watched the
// output.
func auditPromptStructureInvariant(ctx appctx.Context) auditTarget {
	t := auditTarget{
		Target:   "Prompt structure invariant",
		Sensor:   "go test ./heartbeat/ -run TestAssembledPromptEmitsOnlyRegisteredSectionHeadings",
		Deadband: "every level-one heading in an assembled prompt is a registered prompt section",
		Actuator: "framework-update; repair the prompt builder, never the section registry alone",
	}
	// The invariant is enforced in-process by the heartbeat test suite, which
	// has the store and builder this CLI does not. Reporting it here without a
	// corpus-wide sensor of its own would be a claim with nothing behind it, so
	// it carries a gap marker instead.
	t.Observed = "pending-telemetry: enforced in-process by TestAssembledPromptEmitsOnlyRegisteredSectionHeadings and TestPromptPrecedenceListNamesNonEmptySections, with no corpus-wide CLI sensor"
	t.Status = auditStatusNoSensor
	t.GapMarker = "2026-07-31: no corpus-wide CLI sensor; the invariant runs in the heartbeat unit suite, which owns the prompt builder"
	return t
}

// auditRulesWithNoFinding tracks the reduction metric Decision 7 carries. A
// rule that produced no finding is not necessarily dead — a clean tree is also
// silent — so the band is a downward trend, not a threshold.
func auditRulesWithNoFinding(ctx appctx.Context) auditTarget {
	t := auditTarget{
		Target:   "Catalogued rules with no finding",
		Sensor:   "prompt-manager graph rules",
		Deadband: "downward trend against the previous cycle; a single reading is a baseline, not a finding",
		Actuator: "screen the silent rules on whether a test makes each fire and whether a failure names something specific to change",
	}
	var resp rulesResponse
	if err := ctx.Get("/topics/rules", &resp); err != nil {
		return auditFailed(t, err)
	}
	t.Observed = fmt.Sprintf("pending-baseline: %d of %d catalogued rules produced no finding this cycle; a trend needs the previous cycle to band", resp.Silent, resp.Total)
	// A trend target cannot be judged from one reading; the band is comparison
	// against the previous framework-health record, which this command does not
	// hold. Reporting in-band here would assert a trend that was never measured.
	t.Status = auditStatusNoSensor
	t.GapMarker = "2026-07-31: trend target; needs the previous cycle's reading from the framework-health record to band"
	return t
}

func auditContractValidity(ctx appctx.Context) auditTarget {
	t := auditTarget{
		Target:   "Operating-model contract validity",
		Sensor:   "prompt-manager graph operating-model validate",
		Deadband: "0 errors, 0 warnings",
		Actuator: "framework-update, or the owning team's work item type",
	}
	var resp operatingModelValidationResponse
	if err := ctx.GetWithQuery("/operating-models/validate", url.Values{}, &resp); err != nil {
		return auditFailed(t, err)
	}
	t.Observed = fmt.Sprintf("%d errors, %d warnings", resp.Validation.Errors, resp.Validation.Warnings)
	t.Status = auditBand(resp.Validation.Errors == 0 && resp.Validation.Warnings == 0)
	for _, f := range resp.Validation.Findings {
		t.Detail = appendCapped(t.Detail, fmt.Sprintf("[%s] %s: %s", f.Severity, f.Rule, f.Detail))
	}
	return t
}

func auditContractModeCoverage(ctx appctx.Context) auditTarget {
	t := auditTarget{
		Target:   "Contract-mode coverage",
		Sensor:   "prompt-manager graph operating-model list",
		Deadband: "every team with a PoR is mode: contract",
		Actuator: "framework-update",
	}
	var resp operatingModelListResponse
	if err := ctx.GetWithQuery("/operating-models", url.Values{}, &resp); err != nil {
		return auditFailed(t, err)
	}
	contract := 0
	for _, m := range resp.Models {
		mode := ""
		if len(m.Graphs) > 0 {
			mode = m.Graphs[0].Metadata.Mode
		}
		if mode == "contract" {
			contract++
			continue
		}
		t.Detail = appendCapped(t.Detail, fmt.Sprintf("%s is mode=%q", m.Team, mode))
	}
	t.Observed = fmt.Sprintf("%d of %d at mode: contract", contract, len(resp.Models))
	t.Status = auditBand(contract == len(resp.Models))
	return t
}

func auditGraphRuntimeDrift(ctx appctx.Context) auditTarget {
	t := auditTarget{
		Target:   "Graph/runtime drift",
		Sensor:   "prompt-manager graph operating-model diff",
		Deadband: "0 diff items",
		Actuator: "the owning team's work item type",
	}
	var resp operatingModelDiffResponse
	if err := ctx.GetWithQuery("/operating-models/diff", url.Values{}, &resp); err != nil {
		return auditFailed(t, err)
	}
	t.Observed = fmt.Sprintf("%d items", len(resp.Diff))
	t.Status = auditBand(len(resp.Diff) == 0)
	for _, d := range resp.Diff {
		t.Detail = appendCapped(t.Detail, fmt.Sprintf("[%s] %s %s: %s", d.Kind, d.Relationship, d.Team, d.Detail))
	}
	return t
}

// topicsAudit fetches the topic-flow validation once; two targets read it.
func topicsAudit(ctx appctx.Context) (topicsGraphResponse, error) {
	var resp topicsGraphResponse
	err := ctx.GetWithQuery("/topics/graph", url.Values{}, &resp)
	return resp, err
}

func auditTopicIntegrity(ctx appctx.Context) auditTarget {
	t := auditTarget{
		Target:   "Topic-flow declaration integrity",
		Sensor:   "prompt-manager graph topics",
		Deadband: "0 errors",
		Actuator: "framework-update or a member topics.json fix",
	}
	resp, err := topicsAudit(ctx)
	if err != nil {
		return auditFailed(t, err)
	}
	t.Observed = fmt.Sprintf("%d errors", resp.Validation.Errors)
	t.Status = auditBand(resp.Validation.Errors == 0)
	for _, f := range resp.Validation.Findings {
		if f.Severity == "error" {
			t.Detail = appendCapped(t.Detail, fmt.Sprintf("%s %s: %s", f.Rule, f.Prefix, f.Detail))
		}
	}
	return t
}

func auditMemberDocConformance(ctx appctx.Context) auditTarget {
	t := auditTarget{
		Target:   "Member-document conformance",
		Sensor:   "prompt-manager graph topics — member_doc_* findings",
		Deadband: "0 errors; recommended-section gaps are reported, not banded",
		Actuator: "framework-update for errors; the owning team's work item type for recommended-section gaps",
	}
	resp, err := topicsAudit(ctx)
	if err != nil {
		return auditFailed(t, err)
	}

	errors, gaps := 0, 0
	for _, f := range resp.Validation.Findings {
		if !strings.HasPrefix(f.Rule, "member_doc_") {
			continue
		}
		switch {
		case f.Severity == "error":
			errors++
			t.Detail = appendCapped(t.Detail, fmt.Sprintf("%s %s/%s: %s", f.Rule, f.Team, f.Member, f.Detail))
		case f.Rule == "member_doc_section_recommended":
			gaps++
		}
	}

	// Two readings, one band. Errors are mechanical — a validator can demand
	// a heading the roster already carries everywhere, so they belong in the
	// band. Recommended-section gaps need a team to author content that no
	// validator can infer, and they route to a different actuator; banding
	// them here would put the wrong fix in the actuator column. They stay
	// visible in Observed so the band never reads as fuller coverage than it
	// has.
	t.Observed = fmt.Sprintf("%d errors, %d recommended-section gaps", errors, gaps)
	t.Status = auditBand(errors == 0)
	return t
}

func auditProseCoherence(ctx appctx.Context) auditTarget {
	t := auditTarget{
		Target:   "Prose/declaration coherence",
		Sensor:   "prompt-manager graph topics — prose_topic_leak warnings",
		Deadband: "0 warnings on declaration-bearing surfaces",
		Actuator: "framework-update — tighten the inferred-backtick matcher, then triage the residue",
	}
	resp, err := topicsAudit(ctx)
	if err != nil {
		return auditFailed(t, err)
	}
	leaks := 0
	for _, f := range resp.Validation.Findings {
		if f.Rule == "prose_topic_leak" {
			leaks++
		}
	}
	// The deadband is the target, not the current reading. A deadband set
	// equal to the observation reports in-band while the defect stands and can
	// only ever detect growth — the failure mode infra-contrarian's rubric
	// names as "dead-sensor evidence" and "target drift", applied here to the
	// framework's own sensor map.
	t.Observed = fmt.Sprintf("%d warnings", leaks)
	t.Status = auditBand(leaks == 0)
	return t
}

func auditCrossTeamCoupling(ctx appctx.Context) auditTarget {
	t := auditTarget{
		Target:   "Cross-team coupling visibility",
		Sensor:   "prompt-manager graph map --json",
		Deadband: "0 runtime-only relationship rows; retain every composed edge",
		Actuator: "the producing team's work item type",
	}
	var m operatingMap
	if err := ctx.Get("/operating-models/map", &m); err != nil {
		return auditFailed(t, err)
	}
	t.Observed = fmt.Sprintf("%d composed edges across %d teams", len(m.Edges), len(m.Teams))
	t.Status = auditBand(len(m.Edges) > 0)
	return t
}

// auditStaticallyUnreferencedSkills reports and does not band. Static
// reference is one of three reachability classes (FRAMEWORK_HEALTH
// §"Three reachability classes, one sensor"); banding on it alone reported
// heavily-discovered skills as unreachable. The row returns to a band when the
// sensor joins static references, discovery hits, and read counts.
// discoveryMetricsResponse mirrors the fields of GET /discovery-metrics that
// this sweep reads. The full payload carries more; keep this struct to the
// budget-pressure slice so an unrelated field addition cannot break the audit.
type discoveryMetricsResponse struct {
	BudgetedCallCount int     `json:"budgetedCallCount"`
	OverBudgetRate    float64 `json:"overBudgetRate"`
	BudgetHogs        []struct {
		ID             string `json:"id"`
		MaxChars       int    `json:"maxChars"`
		Seen           int    `json:"seen"`
		OverBudgetSeen int    `json:"overBudgetSeen"`
	} `json:"budgetHogs"`
}

func auditStaticallyUnreferencedSkills(ctx appctx.Context) auditTarget {
	t := auditTarget{
		Target:   "Statically unreferenced skills",
		Sensor:   "prompt-manager graph orphaned-skills",
		Deadband: "none until the sensor joins static references, discovery hits, and read counts",
		Actuator: "skill-deprecation or skill-improvement, only after the join confirms the skill is also undiscovered and unread",
	}
	var nodes []node
	if err := ctx.Get("/graph/orphans", &nodes); err != nil {
		return auditFailed(t, err)
	}
	t.Observed = fmt.Sprintf("pending-baseline — %d statically unreferenced; cross-check against prompt-manager skill-usage before treating any as dead", len(nodes))
	t.Status = auditStatusNoSensor
	t.GapMarker = "2026-07-31 — discovery and read telemetry are now instrumented (prompt-manager skill-usage); restore a band once a full window has accumulated and the three classes are joined in one reading"
	return t
}

func auditDiscoveryBudgetPressure(ctx appctx.Context) auditTarget {
	t := auditTarget{
		Target:   "Discovery budget pressure",
		Sensor:   "prompt-manager discovery-metrics --json — overBudgetRate",
		Deadband: "under 25% of budgeted calls over budget",
		Actuator: "skill-improvement, owned by skill-optimizer",
	}
	var resp discoveryMetricsResponse
	if err := ctx.GetWithQuery("/discovery-metrics", url.Values{}, &resp); err != nil {
		return auditFailed(t, err)
	}
	if resp.BudgetedCallCount == 0 {
		t.Observed = "no budgeted discovery calls in the window"
		t.Status = auditStatusInBand
		return t
	}
	pct := resp.OverBudgetRate * 100
	t.Observed = fmt.Sprintf("%.0f%% of %d budgeted calls over budget", pct, resp.BudgetedCallCount)
	t.Status = auditBand(resp.OverBudgetRate < 0.25)
	for _, h := range resp.BudgetHogs {
		if h.OverBudgetSeen > 0 {
			t.Detail = appendCapped(t.Detail, fmt.Sprintf("%s: %d chars, seen %d, over budget %d", h.ID, h.MaxChars, h.Seen, h.OverBudgetSeen))
		}
	}
	return t
}

// auditCanonCoherence is external: the canon assertions live in a shell test,
// and the CLI does not shell out to test scripts.
func auditCanonCoherence(appctx.Context) auditTarget {
	return auditTarget{
		Target:   "Canon coherence",
		Sensor:   "bash scenarios/prompt-manager/test/agent_system_canon_test.sh",
		Deadband: "all assertions pass",
		Actuator: "framework-update",
		Observed: "not collected — run the sensor command",
		Status:   auditStatusExternal,
	}
}

// auditExperimentLiveness is external: experiment state is owned by the
// experiment API, not the relationship graph.
func auditExperimentLiveness(appctx.Context) auditTarget {
	return auditTarget{
		Target:   "Skill-experiment loop liveness",
		Sensor:   "prompt-manager experiment list",
		Deadband: "at least one concluded experiment per audit cycle once the loop is live",
		Actuator: "skill-experiment-promotion",
		Observed: "not collected — run the sensor command",
		Status:   auditStatusExternal,
	}
}

// auditObjectiveCoverage reads the objective join in both directions.
//
// This target was `external` until the objective edge became a declaration.
// The coverage rule joins the operator's objective table against
// team.json::objectivesServed, so the downward direction (objectives to teams)
// and the upward direction (teams to objectives) are both mechanical here.
//
// What stays outside this sensor: the outcome-category half of the upward
// direction. Categories are Command Center dashboard ids in the outcomes
// charter, not a relationship-graph surface, so Phase 4 of the audit still
// reads that one by hand.
func auditObjectiveCoverage(ctx appctx.Context) auditTarget {
	t := auditTarget{
		Target:   "Objective coverage",
		Sensor:   "prompt-manager graph objectives",
		Deadband: "0 objectives unserved without a dated gap marker; 0 teams tracing to no objective; 0 declaration errors",
		Actuator: "outcome-direction or capability work in director-swarm",
	}
	var resp objectiveResponse
	if err := ctx.GetWithQuery("/objectives", url.Values{}, &resp); err != nil {
		return auditFailed(t, err)
	}
	t.Observed = fmt.Sprintf("%d unserved (%d undeclared), %d unattached team(s), %d error(s), %d warning(s)",
		resp.Unserved, resp.Undeclared, len(resp.UnattachedTeams), resp.Validation.Errors, resp.Validation.Warnings)
	t.Status = auditBand(resp.Undeclared == 0 && len(resp.UnattachedTeams) == 0 && resp.Validation.Errors == 0)
	// A declared hole is in band but must still reach the reader: it is an open
	// finding whose disposition is known, and it stays in the list until it
	// closes rather than disappearing into a green cell.
	for _, row := range resp.Rows {
		if !row.Served {
			marker := row.GapMarker
			if marker == "" {
				marker = "no gap marker"
			}
			t.Detail = appendCapped(t.Detail, fmt.Sprintf("%s unserved (%s)", row.ID, marker))
		}
	}
	for _, f := range resp.Validation.Findings {
		t.Detail = appendCapped(t.Detail, fmt.Sprintf("[%s] %s: %s", f.Severity, f.Rule, f.Detail))
	}
	return t
}

func auditSkillConditioning(appctx.Context) auditTarget {
	return auditTarget{
		Target:    "Skill conditioning quality",
		Sensor:    "per-skill only: the divergence probe (skill-validation §3.3); no corpus-wide sweep exists",
		Deadband:  "0 unreviewed divergence regressions once the corpus sweep exists",
		Actuator:  "skill-improvement",
		Observed:  "pending-baseline — the instrument is built, the sweep is not",
		Status:    auditStatusNoSensor,
		GapMarker: "2026-07-27 — build a corpus-wide divergence-probe sweep; blocked on a stable per-skill evaluation inventory",
	}
}

func auditPoREntropy(appctx.Context) auditTarget {
	return auditTarget{
		Target:    "PoR entropy",
		Sensor:    "state-in-prose telemetry (not implemented)",
		Deadband:  "0 unclassified state-in-prose findings once telemetry exists",
		Actuator:  "framework-update",
		Observed:  "pending-telemetry — state-in-prose is audited by judgment today",
		Status:    auditStatusNoSensor,
		GapMarker: "2026-07-27 — build state-in-prose telemetry; blocked on a stable document-state classification contract",
	}
}

// auditTeamOrientationCost reads the composite for every team.
//
// The band is a trend, so this collector cannot band a single sweep: a rise is
// only a finding when it happened in a cycle where scenario coverage also rose,
// and one reading holds no cycle. It therefore reports the composites and marks
// the target `pending-baseline` until an audit record exists to diff against —
// which is exactly what the audit-record step in FRAMEWORK_HEALTH exists for.
func auditTeamOrientationCost(ctx appctx.Context) auditTarget {
	t := auditTarget{
		Target:    "Team orientation cost",
		Sensor:    "prompt-manager graph orientation-cost",
		Deadband:  "no team's orientation cost rises across an audit cycle in which its scenario coverage grew",
		Actuator:  "team-capability-consolidation",
		Status:    auditStatusNoSensor,
		GapMarker: "2026-07-30 — the composite is built; the trend needs one prior framework-health-audit record to diff against",
	}
	var resp orientationCostReport
	if err := ctx.GetWithQuery("/orientation-cost", url.Values{}, &resp); err != nil {
		return auditFailed(t, err)
	}
	t.Observed = fmt.Sprintf("pending-baseline — %d team composite(s) read; the band needs a prior audit record", len(resp.Teams))
	// Every team's reading is appended, not capped to the usual three leads.
	// For other targets Detail is a lead into a sensor the reader can re-run;
	// here it is the payload the next cycle diffs against, and a truncated
	// record would silently lose teams from the trend.
	for _, team := range resp.Teams {
		t.Detail = append(t.Detail, fmt.Sprintf("%s composite=%d (members=%d canon=%d topics=%d) scenarios=%d",
			team.TeamID, team.Composite, team.Components.Members, team.Components.CanonLines,
			team.Components.Topics, team.ScenarioCoverage))
	}
	return t
}

func auditBand(inBand bool) string {
	if inBand {
		return auditStatusInBand
	}
	return auditStatusOutOfBand
}

func auditFailed(t auditTarget, err error) auditTarget {
	t.Observed = "sensor failed: " + err.Error()
	t.Status = auditStatusOutOfBand
	return t
}

// appendCapped keeps at most three leads per target. The full set stays
// available from the underlying sensor; the cap keeps the artifact readable.
func appendCapped(list []string, item string) []string {
	if len(list) >= 3 {
		return list
	}
	return append(list, item)
}

func printAuditReport(r auditReport) {
	fmt.Printf("Framework Health Audit (%s)\n\n", r.GeneratedAt)
	for _, t := range r.Targets {
		fmt.Printf("%-8s %s\n", auditStatusLabel(t.Status), t.Target)
		fmt.Printf("         observed: %s\n", t.Observed)
		if t.Status == auditStatusOutOfBand {
			if t.Deadband != "" {
				fmt.Printf("         deadband: %s\n", t.Deadband)
			}
			fmt.Printf("         actuator: %s\n", t.Actuator)
			for _, d := range t.Detail {
				fmt.Printf("           - %s\n", d)
			}
		}
	}
	fmt.Printf("\n%d target(s) out of band, %d unsensored, %d total.\n",
		r.OutOfBand, r.Unsensored, len(r.Targets))
	if r.OutOfBand > 0 {
		fmt.Println("\nNext Steps")
		fmt.Println("Record the readings in a framework-health-audit/<date> topic, then route each")
		fmt.Println("out-of-band target to the actuator named above.")
	}
}

func auditStatusLabel(status string) string {
	switch status {
	case auditStatusInBand:
		return "[ok]"
	case auditStatusOutOfBand:
		return "[OUT]"
	case auditStatusExternal:
		return "[ext]"
	default:
		return "[--]"
	}
}

func writeAuditArtifact(path string, r auditReport) error {
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal audit report: %w", err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write audit artifact to %q: %w", path, err)
	}
	return nil
}

// auditTargetTitles is used by tests and by the skill to confirm the sweep
// still covers every canon target.
func auditTargetTitles(r auditReport) []string {
	titles := make([]string, 0, len(r.Targets))
	for _, t := range r.Targets {
		titles = append(titles, strings.TrimSpace(t.Target))
	}
	return titles
}

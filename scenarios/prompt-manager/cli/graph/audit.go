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
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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

// Honesty flags for open-loop targets, sharing the vocabulary defined in
// docs/infra-health/strategy/RELIABILITY_TARGETS.md §"Honesty flags".
//
// The distinction is a routing decision, not a label: pending-telemetry means
// no instrument exists and one must be built, while pending-baseline means the
// instrument is built and only the sweep or the prior reading is missing. Those
// are different actuators and very different costs, which is why the flag is a
// field rather than a prefix a reader has to parse back out of prose.
const (
	auditHonestyTelemetry = "pending-telemetry"
	auditHonestyBaseline  = "pending-baseline"
)

// discoveryBudgetCeiling is the share of budgeted discovery calls allowed to
// exceed budget. It is declared once and rendered into the deadband text, so
// the prose and the comparison cannot drift apart.
const discoveryBudgetCeiling = 0.25

// auditTrend carries the machine-comparable reading a trend target bands on.
//
// A trend target cannot be judged from one sweep — "downward trend" and "no
// team's cost rises" are both statements about two cycles. Rather than each
// such collector hand-rolling a comparison it has no data for, it emits its
// reading here and stays `no-sensor`; applyBaseline upgrades it to a real band
// when a prior cycle's artifact is supplied via --baseline.
type auditTrend struct {
	// Metric names what Values measure. A baseline whose metric differs is
	// rejected rather than silently compared, so renaming what a target counts
	// cannot produce a meaningless delta.
	Metric string `json:"metric"`
	// Values is keyed by entity, or by "" for a single scalar reading.
	Values map[string]float64 `json:"values"`
	// Guard, when set, is a second per-entity series that must also have risen
	// for a rise in Values to count as a finding. Team orientation cost is only
	// out of band when scenario coverage grew in the same cycle.
	Guard map[string]float64 `json:"guard,omitempty"`
	// Rising reports which direction is the defect. Every trend target today
	// fails upward; the field exists so a future "should be climbing" target
	// does not have to invert its Values to fit.
	RiseIsDefect bool `json:"rise_is_defect"`
}

type auditTarget struct {
	Target   string `json:"target"`
	Sensor   string `json:"sensor"`
	Deadband string `json:"deadband"`
	Actuator string `json:"actuator"`
	Observed string `json:"observed"`
	Status   string `json:"status"`
	// HonestyFlag is set whenever Status is no-sensor, and names which kind of
	// open loop it is. The audit skill routes on this value, so it is typed
	// rather than left for a reader to grep out of Observed.
	HonestyFlag string `json:"honesty_flag,omitempty"`
	// GapMarker is required whenever a target has no automated corpus-wide
	// sensor. It makes the missing instrument a dated, owned work item rather
	// than a silent hole in the audit.
	//
	// The date is parsed out into GapOpenedOn and GapOpenDays rather than left
	// frozen inside the prose: a marker that cannot age cannot answer "how long
	// has this been open", which is the question a trend board exists to ask.
	GapMarker   string `json:"gap_marker,omitempty"`
	GapOpenedOn string `json:"gap_opened_on,omitempty"`
	GapOpenDays int    `json:"gap_open_days,omitempty"`
	// Detail carries the first few offending entries when out of band, so a
	// reader gets a lead without re-running the underlying sensor.
	Detail []string `json:"detail,omitempty"`
	// Trend is set by targets whose deadband is a comparison against the
	// previous cycle. See auditTrend.
	Trend *auditTrend `json:"trend,omitempty"`
}

type auditReport struct {
	SchemaVersion int           `json:"schema_version"`
	GeneratedAt   string        `json:"generated_at"`
	Targets       []auditTarget `json:"targets"`
	OutOfBand     int           `json:"out_of_band"`
	Unsensored    int           `json:"unsensored"`
	// External counts targets whose sensor lives outside this command. They
	// were previously omitted from the printed summary, which let a sweep
	// reporting "3 out of band" hide a target that was failing but uncollected.
	External int `json:"external"`
	// BaselineFrom records the generated_at of the artifact this sweep banded
	// its trend targets against, empty when none was supplied. A trend verdict
	// with no stated baseline is not reproducible.
	BaselineFrom string `json:"baseline_from,omitempty"`
}

// cmdAudit runs the framework-health sweep.
func cmdAudit(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("audit", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	out := fs.String("out", "", "Write the JSON artifact to PATH (atomic)")
	baseline := fs.String("baseline", "", "Band trend targets against the audit artifact at PATH (as written by --out)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	prior, err := loadAuditBaseline(*baseline)
	if err != nil {
		return err
	}

	report := auditReport{
		SchemaVersion: auditSchemaVersion,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		BaselineFrom:  prior.from,
	}
	for _, collect := range auditCollectors() {
		report.Targets = append(report.Targets, applyBaseline(collect(ctx), prior))
	}
	// The open-loop count is derived from the other targets, so it is computed
	// after they have been banded: a trend target the baseline just upgraded is
	// no longer open-loop and must not be counted as one.
	report.Targets = append(report.Targets,
		applyBaseline(auditOpenLoopTargetCount(report.Targets, prior), prior))
	for _, t := range report.Targets {
		switch t.Status {
		case auditStatusOutOfBand:
			report.OutOfBand++
		case auditStatusNoSensor:
			report.Unsensored++
		case auditStatusExternal:
			report.External++
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

// loadAuditBaseline reads a prior sweep's artifact and indexes it by target
// name. An absent path is not an error — banding a trend is an upgrade over the
// open-loop default, never a precondition for running the sweep.
//
// A malformed or unreadable path IS an error. Silently degrading to "no
// baseline" would report every trend target as pending-baseline while the
// operator believed a comparison had happened, which is the dead-sensor shape
// FRAMEWORK_HEALTH.md's deadband rule exists to prevent.
func loadAuditBaseline(path string) (auditBaseline, error) {
	if path == "" {
		return auditBaseline{}, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return auditBaseline{}, fmt.Errorf("read audit baseline %q: %w", path, err)
	}
	var prior auditReport
	if err := json.Unmarshal(raw, &prior); err != nil {
		return auditBaseline{}, fmt.Errorf("parse audit baseline %q: %w", path, err)
	}
	if prior.SchemaVersion != auditSchemaVersion {
		return auditBaseline{}, fmt.Errorf(
			"audit baseline %q is schema version %d, this command writes %d; re-run the sweep to regenerate it",
			path, prior.SchemaVersion, auditSchemaVersion)
	}
	b := auditBaseline{
		from:    prior.GeneratedAt,
		targets: make(map[string]auditTarget, len(prior.Targets)),
	}
	for _, t := range prior.Targets {
		b.targets[strings.TrimSpace(t.Target)] = t
	}
	return b, nil
}

// auditBaseline is the previous cycle's readings, indexed by target name, plus
// the timestamp every trend verdict cites. The zero value means "no baseline
// supplied" and leaves trend targets open-loop.
type auditBaseline struct {
	from    string
	targets map[string]auditTarget
}

// applyBaseline upgrades one trend target from open-loop to a real band by
// comparing it against the same target in the previous cycle.
//
// Targets without a Trend pass through untouched: their deadband is absolute
// and one reading already decides it.
func applyBaseline(t auditTarget, prior auditBaseline) auditTarget {
	if t.Trend == nil {
		return t
	}
	was, ok := prior.targets[strings.TrimSpace(t.Target)]
	if !ok || was.Trend == nil || was.Trend.Metric != t.Trend.Metric {
		return t
	}
	risen := trendRisen(t.Trend, was.Trend)
	t.Status = auditBand(len(risen) == 0)
	// The gap marker and honesty flag both named the missing baseline. It has
	// arrived, so leaving either would keep advertising work that is done.
	t.GapMarker = ""
	t.HonestyFlag = ""
	t.GapOpenedOn = ""
	t.GapOpenDays = 0
	t.Detail = nil
	for _, r := range risen {
		t.Detail = appendCapped(t.Detail, r)
	}
	summary := fmt.Sprintf("%s; %d of %d series rose against the %s baseline",
		trendSummary(t.Trend), len(risen), len(t.Trend.Values), prior.from)
	if t.Trend.Guard != nil {
		summary += " (a rise counts only where the guard series also rose)"
	}
	t.Observed = summary
	return t
}

// trendRisen returns a human-readable line per series that moved in the defect
// direction. A series absent from the baseline is not a rise — it is new, and
// calling a first reading a regression would punish honest declaration.
func trendRisen(now, was *auditTrend) []string {
	keys := make([]string, 0, len(now.Values))
	for k := range now.Values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	risen := make([]string, 0, len(keys))
	for _, k := range keys {
		prev, seen := was.Values[k]
		if !seen {
			continue
		}
		cur := now.Values[k]
		moved := cur > prev
		if !now.RiseIsDefect {
			moved = cur < prev
		}
		if !moved {
			continue
		}
		// The guard gates the finding, not the reading: a cost that rose while
		// its guard series held flat is expected growth, not decay.
		if now.Guard != nil {
			guardPrev, guardSeen := was.Guard[k]
			if !guardSeen || now.Guard[k] <= guardPrev {
				continue
			}
		}
		label := k
		if label == "" {
			label = now.Metric
		}
		risen = append(risen, fmt.Sprintf("%s: %s → %s", label, trendNum(prev), trendNum(cur)))
	}
	return risen
}

func trendSummary(tr *auditTrend) string {
	if len(tr.Values) == 1 {
		if v, ok := tr.Values[""]; ok {
			return fmt.Sprintf("%s = %s", tr.Metric, trendNum(v))
		}
	}
	return fmt.Sprintf("%s across %d series", tr.Metric, len(tr.Values))
}

// trendNum prints whole numbers without a decimal tail; every trend reading
// today is a count, and "62.0 → 63.0" reads as false precision.
func trendNum(v float64) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%d", int64(v))
	}
	return fmt.Sprintf("%.2f", v)
}

// priorTrendUsable reports whether a baseline can band the named target. The
// open-loop count needs this before counting, because whether it counts itself
// depends on whether it is about to be banded.
func priorTrendUsable(prior auditBaseline, target, metric string) bool {
	was, ok := prior.targets[target]
	return ok && was.Trend != nil && was.Trend.Metric == metric
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
		auditInstrumentCoverage,
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
	return openLoop(t, auditHonestyTelemetry,
		"enforced in-process by TestAssembledPromptEmitsOnlyRegisteredSectionHeadings and TestPromptPrecedenceListNamesNonEmptySections, with no corpus-wide CLI sensor",
		"2026-07-31: no corpus-wide CLI sensor; the invariant runs in the heartbeat unit suite, which owns the prompt builder")
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
	// A trend target cannot be judged from one reading. The reading is emitted
	// as a Trend so --baseline can band it; without one it stays open-loop,
	// because reporting in-band would assert a trend that was never measured.
	t.Trend = &auditTrend{
		Metric:       "silent_rules",
		Values:       map[string]float64{"": float64(resp.Silent)},
		RiseIsDefect: true,
	}
	return openLoop(t, auditHonestyBaseline,
		fmt.Sprintf("%d of %d catalogued rules produced no finding this cycle; a trend needs the previous cycle to band", resp.Silent, resp.Total),
		"2026-07-31: trend target; supply the previous cycle's artifact with --baseline to band")
}

// auditOpenLoopTargetCount is derived from the other targets rather than
// collected beside them: it counts how many targets in this sweep have no
// working instrument, which is a property of the sensor map itself.
// `path:docs/director-swarm/strategy/OBJECTIVES.md` cites this count as the
// measure for objective I3 (Enablement). The citation runs one way — the
// measurement is defined and owned here, and the objective borrows it.
//
// It is no-sensor for the same reason auditRulesWithNoFinding is, which means
// it counts itself. That is correct rather than cute: an open-loop count with
// no trend to read against is itself open-loop, and saying otherwise would be
// the dead-sensor shape FRAMEWORK_HEALTH.md's deadband rule names.
func auditOpenLoopTargetCount(collected []auditTarget, prior auditBaseline) auditTarget {
	t := auditTarget{
		Target:   "Open-loop target count",
		Sensor:   "prompt-manager graph audit — count of targets reported no-sensor",
		Deadband: "downward trend across audit cycles; a single reading is a baseline, not a finding. Not banded against a fixed count — a target enters this set the moment it is honestly declared, so growth can mean new honesty rather than new decay",
		Actuator: "capability-work in director-swarm, sequenced by the instrument rule in docs/director-swarm/strategy/PORTFOLIO_PHILOSOPHY.md",
	}
	names := make([]string, 0, len(collected))
	for _, c := range collected {
		if c.Status == auditStatusNoSensor {
			names = append(names, c.Target)
		}
	}
	// This target is one of the rows it counts, so whether it counts itself is
	// decided by whether a baseline is about to band it — resolved here rather
	// than after the fact, so the number it reports and the number a reader
	// gets from tallying the printed list are the same in both states.
	open := len(names)
	if !priorTrendUsable(prior, t.Target, "open_loop_targets") {
		open++
	}
	total := len(collected) + 1

	t.Trend = &auditTrend{
		Metric:       "open_loop_targets",
		Values:       map[string]float64{"": float64(open)},
		RiseIsDefect: true,
	}
	t.Detail = names
	return openLoop(t, auditHonestyBaseline,
		fmt.Sprintf("%d of %d targets have no working instrument; a trend needs the previous cycle to band", open, total),
		"2026-08-09: trend target; supply the previous cycle's artifact with --baseline to band")
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
	// The band has two clauses and the edge count only speaks to the second.
	// Checking `len(m.Edges) > 0` alone could detect nothing but total collapse
	// of the map — the dead-sensor shape the deadband rule names — so the
	// runtime-only clause is read from the coverage endpoint that computes it.
	var cov operatingModelCoverageResponse
	if err := ctx.GetWithQuery("/operating-models/coverage", url.Values{}, &cov); err != nil {
		return auditFailed(t, err)
	}
	runtimeOnly := 0
	for _, graph := range cov.Coverage {
		for _, rel := range graph.Relationships {
			if rel.RuntimeOnly == 0 {
				continue
			}
			runtimeOnly += rel.RuntimeOnly
			t.Detail = appendCapped(t.Detail, fmt.Sprintf("%s/%s: %d runtime-only row(s) not shown in the graph",
				graph.Team, rel.Relationship, rel.RuntimeOnly))
		}
	}
	t.Observed = fmt.Sprintf("%d composed edges across %d teams, %d runtime-only row(s)",
		len(m.Edges), len(m.Teams), runtimeOnly)
	t.Status = auditBand(runtimeOnly == 0 && len(m.Edges) > 0)
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
	return openLoop(t, auditHonestyBaseline,
		fmt.Sprintf("%d statically unreferenced; cross-check against prompt-manager skill-usage before treating any as dead", len(nodes)),
		"2026-07-31 — discovery and read telemetry are now instrumented (prompt-manager skill-usage); restore a band once a full window has accumulated and the three classes are joined in one reading")
}

func auditDiscoveryBudgetPressure(ctx appctx.Context) auditTarget {
	t := auditTarget{
		Target:   "Discovery budget pressure",
		Sensor:   "prompt-manager discovery-metrics --json — overBudgetRate",
		Deadband: fmt.Sprintf("under %.0f%% of budgeted calls over budget", discoveryBudgetCeiling*100),
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
	t.Status = auditBand(resp.OverBudgetRate < discoveryBudgetCeiling)
	for _, h := range resp.BudgetHogs {
		if h.OverBudgetSeen > 0 {
			t.Detail = appendCapped(t.Detail, fmt.Sprintf("%s: %d chars, seen %d, over budget %d", h.ID, h.MaxChars, h.Seen, h.OverBudgetSeen))
		}
	}
	return t
}

// auditCanonCoherence runs the canon assertion script when the repository is
// resolvable, and degrades to `external` when it is not.
//
// This is the one collector that shells out. The rule it bends — the CLI does
// not invoke test scripts — existed because the binary installs to ~/.vrooli/bin
// and may run with no checkout in reach. That is a reason to degrade, not a
// reason to stay blind: the target was failing while the sweep reported "not
// collected", and an uncollected failure reads exactly like a pass.
func auditCanonCoherence(appctx.Context) auditTarget {
	t := auditTarget{
		Target:   "Canon coherence",
		Sensor:   "bash scenarios/prompt-manager/test/agent_system_canon_test.sh",
		Deadband: "all assertions pass",
		Actuator: "framework-update",
	}
	root := cliutil.ResolveRepoRoot()
	if root == "" {
		t.Observed = "not collected — no repository root in reach; run the sensor command from a checkout"
		t.Status = auditStatusExternal
		return t
	}
	script := filepath.Join(root, "scenarios", "prompt-manager", "test", "agent_system_canon_test.sh")
	if _, err := os.Stat(script); err != nil {
		t.Observed = "not collected — " + script + " is not present; run the sensor command"
		t.Status = auditStatusExternal
		return t
	}
	raw, err := canonScriptRunner(root, script)
	passed, failed, parsed := parseCanonTally(string(raw))
	if !parsed {
		// An unparseable run is not a pass. Reporting external here keeps the
		// reader on the hook to run it rather than banding on a guess.
		t.Observed = "not collected — sensor ran but emitted no Passed/Failed tally; run the sensor command"
		t.Status = auditStatusExternal
		return t
	}
	t.Observed = fmt.Sprintf("%d pass, %d fail", passed, failed)
	t.Status = auditBand(failed == 0 && err == nil)
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.Contains(line, "FAIL") {
			t.Detail = appendCapped(t.Detail, strings.TrimSpace(stripANSI(line)))
		}
	}
	return t
}

// canonScriptRunner is a seam, not indirection for its own sake: without it the
// unit suite would shell out to the real canon script on every run, making a
// sweep test depend on repository state it does not own.
var canonScriptRunner = func(root, script string) ([]byte, error) {
	cmd := exec.Command("bash", script)
	cmd.Dir = root
	return cmd.CombinedOutput()
}

// parseCanonTally reads the script's trailing "Passed: N" / "Failed: N" lines.
// It reports parsed=false when either is absent, so a script that changed its
// output shape surfaces as uncollected instead of as zero failures.
func parseCanonTally(out string) (passed, failed int, parsed bool) {
	var sawPassed, sawFailed bool
	for _, line := range strings.Split(out, "\n") {
		clean := strings.TrimSpace(stripANSI(line))
		if v, ok := strings.CutPrefix(clean, "Passed:"); ok {
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				passed, sawPassed = n, true
			}
		}
		if v, ok := strings.CutPrefix(clean, "Failed:"); ok {
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				failed, sawFailed = n, true
			}
		}
	}
	return passed, failed, sawPassed && sawFailed
}

// stripANSI removes the colour escapes the canon script emits, so parsing and
// Detail lines do not carry terminal control bytes into the JSON artifact.
func stripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// auditExperimentLiveness reads the experiment API directly.
//
// It was `external` on the grounds that experiment state is owned by the
// experiment API rather than the relationship graph — but both are served by
// the same prompt-manager API this context already talks to, so the split cost
// a manual step and bought nothing.
func auditExperimentLiveness(ctx appctx.Context) auditTarget {
	t := auditTarget{
		Target:   "Skill-experiment loop liveness",
		Sensor:   "prompt-manager experiment list",
		Deadband: "at least one concluded experiment per audit cycle once the loop is live",
		Actuator: "skill-experiment-promotion",
	}
	var experiments []experimentLivenessRow
	if err := ctx.GetWithQuery("/experiments", url.Values{}, &experiments); err != nil {
		return auditFailed(t, err)
	}
	concluded := 0
	for _, e := range experiments {
		if e.ConcludedAt != nil && *e.ConcludedAt != "" {
			concluded++
		}
	}
	// The deadband carries a precondition the sensor must respect: "at least one
	// concluded experiment per audit cycle **once the loop is live**". With no
	// experiments at all the loop is not live, and calling that out-of-band
	// would report a failure to conclude work nobody started. It is open-loop,
	// which is what no-sensor is for.
	if len(experiments) == 0 {
		return openLoop(t, auditHonestyBaseline,
			"no experiments exist, so the per-cycle conclusion rate has nothing to band",
			"2026-08-10 — the loop is instrumented but has never run; band once the first experiment is created")
	}
	t.Observed = fmt.Sprintf("%d experiment(s), %d concluded", len(experiments), concluded)
	t.Status = auditBand(concluded > 0)
	for _, e := range experiments {
		if e.ConcludedAt == nil || *e.ConcludedAt == "" {
			t.Detail = appendCapped(t.Detail, fmt.Sprintf("%s: %s, not concluded", e.ID, e.Status))
		}
	}
	return t
}

// experimentLivenessRow is the narrow projection this target needs. The full
// shape lives in the experiments package; duplicating two fields here keeps the
// graph package from importing a sibling command package for one count.
type experimentLivenessRow struct {
	ID          string  `json:"id"`
	Status      string  `json:"status"`
	ConcludedAt *string `json:"concludedAt,omitempty"`
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
	return openLoop(auditTarget{
		Target:   "Skill conditioning quality",
		Sensor:   "per-skill only: the divergence probe (skill-validation §3.3); no corpus-wide sweep exists",
		Deadband: "0 unreviewed divergence regressions once the corpus sweep exists",
		Actuator: "skill-improvement",
	}, auditHonestyBaseline,
		"the instrument is built, the sweep is not",
		"2026-07-27 — build a corpus-wide divergence-probe sweep; blocked on a stable per-skill evaluation inventory")
}

func auditPoREntropy(appctx.Context) auditTarget {
	return openLoop(auditTarget{
		Target:   "PoR entropy",
		Sensor:   "state-in-prose telemetry (not implemented)",
		Deadband: "0 unclassified state-in-prose findings once telemetry exists",
		Actuator: "framework-update",
	}, auditHonestyTelemetry,
		"state-in-prose is audited by judgment today",
		"2026-07-27 — build state-in-prose telemetry; blocked on a stable document-state classification contract")
}

// auditInstrumentCoverage reads whether each team declares the one scenario it
// reads for the state of its domain.
//
// The band is "declared or dated", never "present". A team with no instrument
// is in band as long as it says so with a date, for the same reason an unserved
// objective with a gap marker is: a declared hole is a decision the reader can
// age and argue with, while silence is neither. Making presence the band would
// turn this sensor into a controller, which is the boundary the target model
// forbids for instruments themselves.
func auditInstrumentCoverage(ctx appctx.Context) auditTarget {
	t := auditTarget{
		Target:   "Instrument coverage",
		Sensor:   "prompt-manager graph instruments",
		Deadband: "every team declares an instrument or carries a dated gap marker; declarations are internally coherent",
		Actuator: "team-capability-consolidation",
	}
	var resp instrumentCoverageReport
	if err := ctx.GetWithQuery("/instruments", url.Values{}, &resp); err != nil {
		return auditFailed(t, err)
	}
	t.Observed = fmt.Sprintf("%d live, %d partial, %d none, %d undeclared across %d team(s); %d out of band",
		resp.Live, resp.Partial, resp.None, resp.Undeclared, len(resp.Teams), resp.OutOfBand)
	t.Status = auditBand(resp.OutOfBand == 0)
	// A declared hole is in band but still reaches the reader with its age, so
	// a deliberate deferral and an overdue one are told apart in the record.
	for _, team := range resp.Teams {
		for _, finding := range team.Findings {
			t.Detail = appendCapped(t.Detail, fmt.Sprintf("%s: %s", team.TeamID, finding))
		}
		if len(team.Findings) == 0 && team.Instrument != nil && team.Instrument.Status != "live" {
			t.Detail = appendCapped(t.Detail, fmt.Sprintf("%s declared %s since %s", team.TeamID, team.Instrument.Status, team.GapOpenedOn))
		}
	}
	return t
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
		GapMarker: "2026-07-30 — the composite is built; supply the previous cycle's artifact with --baseline to band",
	}
	var resp orientationCostReport
	if err := ctx.GetWithQuery("/orientation-cost", url.Values{}, &resp); err != nil {
		return auditFailed(t, err)
	}
	t = openLoop(t, auditHonestyBaseline,
		fmt.Sprintf("%d team composite(s) read; the band needs a prior audit record", len(resp.Teams)),
		t.GapMarker)
	// The deadband is conditional — a rise is only a finding when scenario
	// coverage grew in the same cycle — so coverage rides along as the guard
	// series rather than being re-derived from the Detail prose next cycle.
	t.Trend = &auditTrend{
		Metric:       "orientation_composite",
		Values:       make(map[string]float64, len(resp.Teams)),
		Guard:        make(map[string]float64, len(resp.Teams)),
		RiseIsDefect: true,
	}
	// Every team's reading is appended, not capped to the usual three leads.
	// For other targets Detail is a lead into a sensor the reader can re-run;
	// here it is the payload the next cycle diffs against, and a truncated
	// record would silently lose teams from the trend.
	for _, team := range resp.Teams {
		t.Trend.Values[team.TeamID] = float64(team.Composite)
		t.Trend.Guard[team.TeamID] = float64(team.ScenarioCoverage)
		t.Detail = append(t.Detail, fmt.Sprintf("%s composite=%d (members=%d canon=%d topics=%d) scenarios=%d",
			team.TeamID, team.Composite, team.Components.Members, team.Components.CanonLines,
			team.Components.Topics, team.ScenarioCoverage))
	}
	return t
}

// openLoop marks a target as having no working instrument. It renders the
// honesty prefix into Observed from the typed flag, so the field a consumer
// routes on and the prose a human reads cannot disagree — and so the delimiter
// stops varying between ": " and " — " from one collector to the next.
func openLoop(t auditTarget, flag, detail, gapMarker string) auditTarget {
	t.HonestyFlag = flag
	t.Observed = flag + " — " + detail
	t.Status = auditStatusNoSensor
	t.GapMarker = gapMarker
	if opened, ok := gapMarkerDate(gapMarker); ok {
		t.GapOpenedOn = opened.Format("2006-01-02")
		t.GapOpenDays = int(auditNow().Sub(opened).Hours() / 24)
	}
	return t
}

// gapMarkerDate lifts the leading YYYY-MM-DD out of a gap marker. Every marker
// in this file starts with one by convention; parsing it rather than trusting
// the convention means a marker that loses its date degrades to "no age known"
// instead of reporting a wrong one.
func gapMarkerDate(marker string) (time.Time, bool) {
	if len(marker) < 10 {
		return time.Time{}, false
	}
	opened, err := time.Parse("2006-01-02", marker[:10])
	if err != nil {
		return time.Time{}, false
	}
	return opened, true
}

// auditNow is a seam so gap ages are deterministic under test. Production reads
// the wall clock; a test that asserted against it would rot on its own.
var auditNow = func() time.Time { return time.Now().UTC() }

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
		// An open gap's age is the thing that turns a visible hole into an
		// overdue one; without it every marker reads as equally fresh.
		if t.GapOpenDays > 0 {
			fmt.Printf("         gap open: %d days (since %s)\n", t.GapOpenDays, t.GapOpenedOn)
		}
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
	// Externals are named in the tally rather than left implicit. A summary that
	// counted only out-of-band and unsensored let an uncollected target read as
	// a clean one — which is how a failing canon-coherence run stayed invisible.
	fmt.Printf("\n%d target(s) out of band, %d unsensored, %d not collected, %d total.\n",
		r.OutOfBand, r.Unsensored, r.External, len(r.Targets))
	if r.BaselineFrom != "" {
		fmt.Printf("Trend targets banded against the %s baseline.\n", r.BaselineFrom)
	} else {
		fmt.Println("No --baseline supplied: trend targets report their reading and stay open-loop.")
	}

	if r.OutOfBand == 0 && r.External == 0 {
		return
	}
	fmt.Println("\nNext Steps")
	if r.OutOfBand > 0 {
		fmt.Println("Record the readings in a framework-health-audit/<date> topic, then route each")
		fmt.Println("out-of-band target to the actuator named above.")
	}
	for _, t := range r.Targets {
		if t.Status == auditStatusExternal {
			fmt.Printf("Run the uncollected sensor for %q: %s\n", t.Target, t.Sensor)
		}
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

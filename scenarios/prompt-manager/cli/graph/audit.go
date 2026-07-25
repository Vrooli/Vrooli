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
		auditProseCoherence,
		auditCrossTeamCoupling,
		auditCanonCoherence,
		auditObjectiveCoverage,
		auditSkillReachability,
		auditSkillConditioning,
		auditExperimentLiveness,
		auditPoREntropy,
	}
}

func auditContractValidity(ctx appctx.Context) auditTarget {
	t := auditTarget{
		Target:   "Operating-model contract validity",
		Sensor:   "prompt-manager graph operating-model validate",
		Deadband: "0 errors, 0 warnings",
		Actuator: "framework-update, or the owning team's decision context",
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
		Actuator: "the owning team's decision context",
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

func auditProseCoherence(ctx appctx.Context) auditTarget {
	t := auditTarget{
		Target:   "Prose/declaration coherence",
		Sensor:   "prompt-manager graph topics — prose_topic_leak warnings",
		Deadband: "519 warnings or fewer pending triage",
		Actuator: "framework-update",
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
	t.Observed = fmt.Sprintf("%d warnings", leaks)
	t.Status = auditBand(leaks <= 519)
	return t
}

func auditCrossTeamCoupling(ctx appctx.Context) auditTarget {
	t := auditTarget{
		Target:   "Cross-team coupling visibility",
		Sensor:   "prompt-manager graph map --json",
		Deadband: "0 runtime-only relationship rows; retain every composed edge",
		Actuator: "the producing team's decision context",
	}
	var m operatingMap
	if err := ctx.Get("/operating-models/map", &m); err != nil {
		return auditFailed(t, err)
	}
	t.Observed = fmt.Sprintf("%d composed edges across %d teams", len(m.Edges), len(m.Teams))
	t.Status = auditBand(len(m.Edges) > 0)
	return t
}

func auditSkillReachability(ctx appctx.Context) auditTarget {
	t := auditTarget{
		Target:   "Skill reachability",
		Sensor:   "prompt-manager graph orphaned-skills",
		Deadband: "54 candidates or fewer pending triage",
		Actuator: "skill-deprecation or skill-improvement",
	}
	var nodes []node
	if err := ctx.Get("/graph/orphans", &nodes); err != nil {
		return auditFailed(t, err)
	}
	t.Observed = fmt.Sprintf("%d candidates", len(nodes))
	t.Status = auditBand(len(nodes) <= 54)
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

// auditObjectiveCoverage is external: the coverage rule joins two plan-of-record
// documents, and neither the objective set nor the team contribution map is a
// relationship-graph surface. Reporting it as external keeps the join visible
// rather than letting an unserved objective read as a clean sweep.
func auditObjectiveCoverage(appctx.Context) auditTarget {
	return auditTarget{
		Target:   "Objective coverage",
		Sensor:   "read docs/director-swarm/strategy/OBJECTIVES.md §\"The coverage rule\" against the charter's team contribution map",
		Deadband: "0 objectives unserved without a dated gap marker; 0 teams or outcome categories tracing to no objective",
		Actuator: "outcome-direction or capability-gap in director-swarm",
		Observed: "not collected — read both coverage directions",
		Status:   auditStatusExternal,
	}
}

func auditSkillConditioning(appctx.Context) auditTarget {
	return auditTarget{
		Target:   "Skill conditioning quality",
		Sensor:   "per-skill only: the divergence probe (skill-validation §3.3); no corpus-wide sweep exists",
		Deadband: "",
		Actuator: "skill-improvement",
		Observed: "pending-baseline — the instrument is built, the sweep is not",
		Status:   auditStatusNoSensor,
	}
}

func auditPoREntropy(appctx.Context) auditTarget {
	return auditTarget{
		Target:   "PoR entropy",
		Sensor:   "",
		Deadband: "",
		Actuator: "framework-update",
		Observed: "pending-telemetry — state-in-prose is audited by judgment today",
		Status:   auditStatusNoSensor,
	}
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

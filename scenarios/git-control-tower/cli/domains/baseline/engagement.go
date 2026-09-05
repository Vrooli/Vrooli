// Engagement verbs (Baseline Modes, plan phase P2) layer the stateful
// shadow/live *engagement* on top of the passive baseline *record*. They are a
// thin orchestration: every recovery-floor primitive lives in the platform
// `vrooli recovery` command group and the data substrate in `data-backup-manager
// safety` — this file only sequences shell-outs to them plus the existing
// BaselinesService client (for the anchor snapshot / diff). GCT owns no recovery
// state: the floor-owned engagement manifest is the source of truth, so a broken
// GCT can still be recovered from the floor (the trust-boundary in P2 §control-plane).
//
// Verbs implemented here: start / check / status / abandon / gc. `promote`
// (shadow→live, terminal "keep") lives in promote.go — it shells the P6
// promote-quiesce drain (`agent-manager run quiesce`) plus the floor + data
// substrate.
package baseline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/coreset"
	"github.com/vrooli/cli-core/cliapp"
	vroolicli "github.com/vrooli/vrooli-cli-go"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
)

// Execution modes (Contract Decision §8).
const (
	modeShadow = "shadow"
	modeLive   = "live"
	modeAuto   = "auto"
)

// defaultEngagementSlug names the per-scenario engagement directory
// (baseline-<slug>) when the caller does not pass --name. v1 runs one engagement
// per scenario at a time, so a stable default keeps the common path flag-free.
const defaultEngagementSlug = "wip"

// runCommand shells a trusted-base command (`vrooli …`) or a scenario CLI
// (on PATH, e.g. `scenario-dependency-analyzer`). It is the single seam every
// engagement verb routes external calls through, so tests inject a recorder.
var runCommand = func(ctx context.Context, name string, args ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		return stdout.Bytes(), fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, msg)
	}
	return stdout.Bytes(), nil
}

// cliVrooliRunner bridges the typed vrooli CLI client onto this package's single
// runCommand seam. Recovery JSON reads then decode the generated vrooli.cli.v1
// contracts (typed, snake_case) while every external call still routes through
// the one recorder that tests inject.
type cliVrooliRunner struct{}

func (cliVrooliRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return runCommand(ctx, name, args...)
}

func (cliVrooliRunner) RunCombined(ctx context.Context, name string, args ...string) ([]byte, error) {
	return runCommand(ctx, name, args...)
}

// cliClient issues typed `vrooli recovery …` reads through runCommand.
var cliClient = vroolicli.New(vroolicli.WithRunner(cliVrooliRunner{}))

// snapshotAnchor captures a baseline record to diff against later. It is a seam
// (not an inline clientFactory call) so engagement tests need not implement the
// full Connect client interface.
var snapshotAnchor = func(core *cliapp.ScenarioApp, ctx context.Context, scenario, name string) error {
	client := clientFactory(core)
	_, err := client.SnapshotForBaseline(ctx, connect.NewRequest(&baselinesv1.SnapshotForBaselineRequest{
		Scenario: scenario, Name: name, CreatedBy: "agent", Reason: "baseline engagement anchor",
	}))
	return err
}

// diffAnchor diffs a baseline record against the working tree and returns the
// overall verdict. It starts the durable diff then resolves the verdict with a
// server-side wait (no client polling) — the engagement flow needs the verdict
// synchronously. Seam, same rationale as snapshotAnchor.
var diffAnchor = func(core *cliapp.ScenarioApp, ctx context.Context, scenario, name string) (string, error) {
	client := clientFactory(core)
	start, err := client.StartDiff(ctx, connect.NewRequest(&baselinesv1.StartDiffRequest{
		Scenario: scenario, Name: name,
	}))
	if err != nil {
		return "", err
	}
	resp, err := client.GetDiffResult(ctx, connect.NewRequest(&baselinesv1.GetDiffResultRequest{
		Scenario: scenario, Name: name, RunId: start.Msg.GetRunId(), Wait: true,
	}))
	if err != nil {
		return "", err
	}
	return resp.Msg.GetDiff().GetVerdict(), nil
}

// registerEngagementVerbs returns the engagement subcommands appended to the
// baseline group by Register.
func registerEngagementVerbs(core *cliapp.ScenarioApp) []cliapp.Command {
	return []cliapp.Command{
		{Name: "start", NeedsAPI: true, Description: "Begin a shadow/live engagement: decide mode, take a restore point, capture an anchor, stand up the shadow (--scenario [--mode auto|shadow|live] [--ttl] [--name] [--operator-confirm])", Run: func(a []string) error { return runStartCmd(core, a) }},
		{Name: "check", NeedsAPI: true, Description: "Validate the engagement target and emit mode-aware guidance; renews the lease (--scenario [--name])", Run: func(a []string) error { return runCheckCmd(core, a) }},
		{Name: "promote", NeedsAPI: false, Description: "Keep the work (terminal): shadow → drain live, snapshot, re-point+restart, probe, auto-rollback on failure, tear down the shadow; live → accept in place (--scenario [--name] [--exclude-run] [--tag-prefix] [--drain-timeout] [--force] [--no-drain])", Run: func(a []string) error { return runPromoteCmd(core, a) }},
		{Name: "status", NeedsAPI: false, Description: "List active engagements (globs the floor-owned manifests) (--json)", Run: func(a []string) error { return runStatusCmd(core, a) }},
		{Name: "abandon", NeedsAPI: false, Description: "Throw the engagement away: shadow → tear down (live untouched); live → restore the working tree from the restore point (--scenario [--name])", Run: func(a []string) error { return runAbandonCmd(core, a) }},
		{Name: "gc", NeedsAPI: false, Description: "Reap expired/orphaned shadows + clean their restore points/manifests (--force) (--json)", Run: func(a []string) error { return runGCCmd(core, a) }},
	}
}

// ---- decision tree -------------------------------------------------------

// modeSignals are the change-shape inputs the decision tree reasons over.
// trusted-base membership and the namespaceability gate are auto-derived from
// the core set; the rest are declared by the planning agent (command recommends,
// agent confirms — Contract Decision §8).
type modeSignals struct {
	writesSharedStore bool // writes an un-adopted Redis/Qdrant store (hard namespaceability gate)
	modifiesLifecycle bool // touches lifecycle/registry/promote machinery (soft gate)
	singletonResource bool // needs a non-duplicable singleton resource (soft gate)
	operatorConfirm   bool // operator nod authorizing live-on-reflexive
}

// coreSet is the reflexive-set view the decision tree consults: the full core
// membership (over-inclusion is safe) and the never-shadowed trusted-base subset.
type coreSet struct {
	members     map[string]bool
	trustedBase map[string]bool
	source      string
}

func (c coreSet) isMember(s string) bool      { return c.members[s] }
func (c coreSet) isTrustedBase(s string) bool { return c.trustedBase[s] }

// modeDecision is the decision tree's verdict for a (scenario, requested-mode).
type modeDecision struct {
	Mode          string   `json:"mode"`
	Reflexive     bool     `json:"reflexive"`
	NeedsOperator bool     `json:"needsOperator"`
	Reasons       []string `json:"reasons"`
}

// decideMode runs the Baseline Modes decision tree. Hard gates (trusted base,
// namespaceability) override even an explicit --mode shadow; soft gates only
// bind in auto. live-on-reflexive flags NeedsOperator unless --operator-confirm.
func decideMode(scenario, requested string, sig modeSignals, cs coreSet) modeDecision {
	reflexive := cs.isMember(scenario)
	d := modeDecision{Reflexive: reflexive}

	// Hard stop: trusted-base scenarios run the engagement machinery — never shadow.
	if cs.isTrustedBase(scenario) {
		d.Mode = modeLive
		d.Reasons = append(d.Reasons, scenario+" ∈ trusted base (runs the engagement) — hard-routed to live, never shadowed")
		d.NeedsOperator = !sig.operatorConfirm
		return d
	}

	hardLive := sig.writesSharedStore
	var soft []string
	if sig.modifiesLifecycle {
		soft = append(soft, "change modifies lifecycle/registry/promote machinery")
	}
	if sig.singletonResource {
		soft = append(soft, "requires a non-duplicable singleton resource")
	}

	finishReflexiveLive := func() modeDecision {
		if reflexive {
			d.NeedsOperator = !sig.operatorConfirm
			d.Reasons = append(d.Reasons, scenario+" is reflexive (core set) — live edits need an operator nod")
		}
		return d
	}

	switch requested {
	case modeLive:
		d.Mode = modeLive
		d.Reasons = append(d.Reasons, "live mode requested")
		return finishReflexiveLive()
	case modeShadow:
		if hardLive {
			d.Mode = modeLive
			d.Reasons = append(d.Reasons, "namespaceability gate (hard): scenario writes an un-adopted Redis/Qdrant store — routed to live despite --mode shadow")
			return finishReflexiveLive()
		}
		d.Mode = modeShadow
		d.Reasons = append(d.Reasons, "shadow requested; no hard gate tripped")
		for _, s := range soft {
			d.Reasons = append(d.Reasons, "note (overridden by explicit --mode shadow): "+s)
		}
		return d
	default: // auto / ""
		switch {
		case hardLive:
			d.Mode = modeLive
			d.Reasons = append(d.Reasons, "namespaceability gate: scenario writes an un-adopted Redis/Qdrant store")
			return finishReflexiveLive()
		case len(soft) > 0:
			d.Mode = modeLive
			d.Reasons = append(d.Reasons, soft...)
			return finishReflexiveLive()
		default:
			d.Mode = modeShadow
			d.Reasons = append(d.Reasons, "shadow is the default safe path; no live-routing gate tripped")
			return d
		}
	}
}

// loadCoreSet seeds the reflexive set from the api-core SSOT constant (always
// available — the trusted-base hard-stop must hold even with the analyzer down)
// then best-effort augments it with the live closure from
// scenario-dependency-analyzer. Unions only add; the trusted base is never
// shrunk (over-inclusion is the safe bias).
func loadCoreSet(ctx context.Context) coreSet {
	cs := coreSet{members: map[string]bool{}, trustedBase: map[string]bool{}, source: "fallback"}
	for _, s := range coreset.DefaultFallbackCoreSet() {
		cs.members[s] = true
	}
	for _, s := range coreset.TrustedBaseScenarios() {
		cs.members[s] = true
		cs.trustedBase[s] = true
	}
	if out, err := runCommand(ctx, "scenario-dependency-analyzer", "core-set", "--json"); err == nil {
		var resp struct {
			Source      string   `json:"source"`
			CoreSet     []string `json:"core_set"`
			TrustedBase []string `json:"trusted_base"`
		}
		if json.Unmarshal(out, &resp) == nil && len(resp.CoreSet) > 0 {
			for _, s := range resp.CoreSet {
				cs.members[s] = true
			}
			for _, s := range resp.TrustedBase {
				cs.trustedBase[s] = true
			}
			cs.source = "analyzer:" + resp.Source
		}
	}
	return cs
}

// ---- engagement view (parsed from `vrooli recovery … --json`) ------------

// engagementView is the subset of the floor's EngagementView the verbs read
// back. It is an internal view-model mapped from the typed
// vrooli.cli.v1.RecoveryEngagementView contract (see engagementFromProto).
type engagementView struct {
	Scenario           string
	Slug               string
	Mode               string
	Variant            string
	ShadowInstanceKey  string
	AnchorBaselineName string
	AmbientVar         string
	TTL                string
	ExpiresAt          *time.Time
	Expired            bool
}

// engagementFromProto maps the typed recovery-floor engagement contract onto the
// local view-model. expires_at arrives as an RFC3339Nano string ("" when none);
// an unparseable value degrades to nil rather than failing the read.
func engagementFromProto(v *cliv1.RecoveryEngagementView) engagementView {
	ev := engagementView{
		Scenario:           v.GetScenario(),
		Slug:               v.GetSlug(),
		Mode:               v.GetMode(),
		Variant:            v.GetVariant(),
		ShadowInstanceKey:  v.GetShadowInstanceKey(),
		AnchorBaselineName: v.GetAnchorBaselineName(),
		AmbientVar:         v.GetAmbientVar(),
		TTL:                v.GetTtl(),
		Expired:            v.GetExpired(),
	}
	if s := strings.TrimSpace(v.GetExpiresAt()); s != "" {
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			ev.ExpiresAt = &t
		}
	}
	return ev
}

func readEngagement(ctx context.Context, scenario, slug string) (engagementView, error) {
	v, err := cliClient.RecoveryShow(ctx, scenario, slug)
	if err != nil {
		return engagementView{}, fmt.Errorf("read engagement %s/%s: %w", scenario, slug, err)
	}
	return engagementFromProto(v), nil
}

// ---- start ---------------------------------------------------------------

type startResult struct {
	Scenario   string       `json:"scenario"`
	Slug       string       `json:"slug"`
	Decision   modeDecision `json:"decision"`
	Variant    string       `json:"variant"`
	Anchor     string       `json:"anchor,omitempty"`
	AmbientVar string       `json:"ambientVar,omitempty"`
	TTL        string       `json:"ttl,omitempty"`
	// DataPopulation holds the human notes from seeding the shadow with a copy of
	// live data (shadow mode only). Empty in live mode or when the scenario has no
	// stateful targets to copy.
	DataPopulation []string `json:"dataPopulation,omitempty"`
	Available      []string `json:"available"`
}

func runStartCmd(core *cliapp.ScenarioApp, args []string) error {
	var scenario, modeFlag, slug, ttlStr, anchor string
	var operatorConfirm, writesShared, modifiesLifecycle, singleton, noAnchor, replace, jsonOut bool
	fs := newFlagSet("baseline start")
	fs.StringVar(&scenario, "scenario", "", "Scenario slug (required)")
	fs.StringVar(&modeFlag, "mode", modeAuto, "Execution mode: auto|shadow|live")
	fs.StringVar(&slug, "name", defaultEngagementSlug, "Engagement slug (the baseline-<slug> directory)")
	fs.StringVar(&ttlStr, "ttl", "", "Idle TTL for a human-owned engagement (e.g. 3h); omit for orchestrator-heartbeat mode")
	fs.StringVar(&anchor, "anchor", "", "Reuse an existing baseline record as the diff anchor (default: capture engagement-<slug>)")
	fs.BoolVar(&operatorConfirm, "operator-confirm", false, "Operator nod authorizing live mode on a reflexive scenario")
	fs.BoolVar(&writesShared, "writes-shared-store", false, "Force the namespaceability gate (→ live); auto-detected from scenario-auditor storage-namespace-v1 by default, this overrides it")
	fs.BoolVar(&modifiesLifecycle, "modifies-lifecycle", false, "Declare the change modifies lifecycle/registry/promote machinery (→ live)")
	fs.BoolVar(&singleton, "singleton-resource", false, "Declare the change needs a non-duplicable singleton resource (→ live)")
	fs.BoolVar(&noAnchor, "no-anchor", false, "Skip capturing a diff anchor (restore-point safety net only)")
	fs.BoolVar(&replace, "replace", false, "Take over a live engagement for this scenario and slug (refused without it)")
	fs.BoolVar(&jsonOut, "json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	res, err := startEngagement(core, startParams{
		scenario: scenario, mode: modeFlag, slug: slug, ttl: ttlStr, anchor: anchor,
		signals:  modeSignals{writesSharedStore: writesShared, modifiesLifecycle: modifiesLifecycle, singletonResource: singleton, operatorConfirm: operatorConfirm},
		noAnchor: noAnchor,
		replace:  replace,
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return printJSON(res)
	}
	printStart(res)
	return nil
}

type startParams struct {
	scenario string
	mode     string
	slug     string
	ttl      string
	anchor   string
	signals  modeSignals
	noAnchor bool
	replace  bool
}

// startEngagement runs the full start sequence, returning the structured result.
// Split from runStartCmd so it is unit-testable with an injected runCommand.
func startEngagement(core *cliapp.ScenarioApp, p startParams) (startResult, error) {
	scenario := strings.TrimSpace(p.scenario)
	if scenario == "" {
		return startResult{}, fmt.Errorf("--scenario is required")
	}
	slug := strings.TrimSpace(p.slug)
	if slug == "" {
		slug = defaultEngagementSlug
	}
	modeReq := strings.ToLower(strings.TrimSpace(p.mode))
	if modeReq == "" {
		modeReq = modeAuto
	}
	switch modeReq {
	case modeAuto, modeShadow, modeLive:
	default:
		return startResult{}, fmt.Errorf("--mode must be auto|shadow|live, got %q", p.mode)
	}
	var ttl time.Duration
	if s := strings.TrimSpace(p.ttl); s != "" {
		d, err := time.ParseDuration(s)
		if err != nil {
			return startResult{}, fmt.Errorf("invalid --ttl %q: %w", p.ttl, err)
		}
		ttl = d
	}

	ctx := context.Background()
	cs := loadCoreSet(ctx)

	// Namespaceability gate: auto-derive whether the scenario writes an un-adopted
	// Redis/Qdrant store from the scenario-auditor storage-namespace-v1 standard
	// (plan §1a — the mode must be chosen automatically, not hand-declared). Only
	// worth querying when a shadow could still be chosen: live mode is already the
	// gate's destination, and a trusted-base scenario is hard-routed to live
	// regardless. The declared --writes-shared-store flag remains an override.
	sig := p.signals
	var nsNote string
	if modeReq != modeLive && !cs.isTrustedBase(scenario) {
		v := resolveSharedStoreSignal(ctx, scenario, sig.writesSharedStore)
		sig.writesSharedStore = v.writesSharedStore
		nsNote = v.note
	}

	decision := decideMode(scenario, modeReq, sig, cs)
	if nsNote != "" {
		decision.Reasons = append(decision.Reasons, "namespaceability signal: "+nsNote)
	}
	if decision.NeedsOperator {
		return startResult{}, fmt.Errorf("live mode on reflexive scenario %q requires an operator nod — re-run with --operator-confirm (reasons: %s)",
			scenario, strings.Join(decision.Reasons, "; "))
	}

	variant := modeLive
	ambient := ""
	if decision.Mode == modeShadow {
		variant = modeShadow
		ambient = scenario
	}

	// 1. Restore point (the git-free undo) — always, in either mode.
	if _, err := runCommand(ctx, "vrooli", "recovery", "capture", "--scenario", scenario, "--slug", slug); err != nil {
		return startResult{}, fmt.Errorf("capture restore point: %w", err)
	}

	// 2. Anchor snapshot (subsumes `snapshot`) unless reused/skipped.
	anchorName := strings.TrimSpace(p.anchor)
	if anchorName == "" && !p.noAnchor {
		anchorName = "engagement-" + slug
		if err := snapshotAnchor(core, ctx, scenario, anchorName); err != nil {
			return startResult{}, fmt.Errorf("capture anchor baseline %q: %w", anchorName, err)
		}
	}

	// 3. Floor-owned engagement manifest.
	writeArgs := []string{"recovery", "write", "--scenario", scenario, "--slug", slug, "--mode", decision.Mode}
	if ttl > 0 {
		writeArgs = append(writeArgs, "--ttl", ttl.String())
	}
	if ambient != "" {
		writeArgs = append(writeArgs, "--ambient-var", ambient)
	}
	if anchorName != "" {
		writeArgs = append(writeArgs, "--anchor", anchorName)
	}
	if p.replace {
		writeArgs = append(writeArgs, "--replace")
	}
	// A live engagement for this scenario and slug is another session's
	// restore point; the control plane refuses to overwrite it and names the
	// holder. --replace is the explicit override.
	if _, err := runCommand(ctx, "vrooli", writeArgs...); err != nil {
		return startResult{}, fmt.Errorf("write engagement manifest: %w", err)
	}

	// 4. Shadow stand-up (a second named instance on alternate ports/namespaces).
	var dataNotes []string
	if decision.Mode == modeShadow {
		if _, err := runCommand(ctx, "vrooli", "scenario", "start", scenario, "--instance", variant); err != nil {
			return startResult{}, fmt.Errorf("stand up shadow instance %s@%s: %w", scenario, variant, err)
		}

		// 5. Shadow data population (the data half): seed the fresh shadow with a
		//    copy of live's stateful data so `check` validates against realistic
		//    state. Best-effort + NON-FATAL — a code-only scenario or an
		//    unreachable substrate leaves the shadow running with empty data.
		dataNotes = populateShadowData(ctx, scenario, variant)
	}

	res := startResult{
		Scenario: scenario, Slug: slug, Decision: decision, Variant: variant,
		Anchor: anchorName, AmbientVar: ambient, DataPopulation: dataNotes,
		Available: []string{"check", "promote", "abandon", "status"},
	}
	if ttl > 0 {
		res.TTL = ttl.String()
	}
	return res, nil
}

// ---- shadow data population (the data half of `baseline start`) ----------

// Shadow data population polling budget. A safety backup of a single scenario's
// targets is typically seconds; the cap bounds a hung substrate without blocking
// the engagement forever (exhausting it is non-fatal — the shadow runs empty).
const (
	populatePollInterval = 2 * time.Second
	populateMaxAttempts  = 60
)

// sleepFn is the poll-delay seam so tests drive the wait loop without real time.
var sleepFn = time.Sleep

// populateShadowData seeds a freshly-started shadow instance with a copy of
// live's stateful data — the data half of `baseline start` in shadow mode. The
// orchestration is: register the scenario's conventional backup targets → snapshot
// live now → wait for that backup to finish → resolve the shadow's SSOT storage
// namespaces (the trusted-base `vrooli recovery namespace` query) → restore each
// target's snapshot into its shadow namespace.
//
// Every step is best-effort and NON-FATAL: a code-only scenario (no stateful
// targets), an unreachable data substrate, or a backup that never reaches
// terminal all leave the shadow running with EMPTY data — valid for many
// validations — rather than failing the whole engagement. It returns human notes
// for the start step log so the outcome is never silent.
func populateShadowData(ctx context.Context, scenario, variant string) []string {
	targets, note := registeredSafetyTargets(ctx, scenario)
	if note != "" {
		return []string{note}
	}
	if len(targets) == 0 {
		return []string{"shadow data population skipped — no registered stateful targets (code-only)"}
	}

	// Reuse the same backup-now primitive promote uses for its pre-promote snapshot.
	runID, ok := prePromoteSnapshot(ctx, scenario)
	if !ok {
		return []string{"shadow data population skipped — safety backup unavailable (substrate unreachable?)"}
	}
	if !waitForSafetyRun(ctx, runID) {
		return []string{fmt.Sprintf("shadow data population skipped — safety run %s did not finish within the poll budget", runID)}
	}

	mappings := shadowTargetMappings(ctx, scenario, variant, targets)
	if mappings == "" {
		return []string{"shadow data population skipped — no registered target maps to a shadow namespace"}
	}
	return []string{populateShadowFromRun(ctx, scenario, runID, mappings)}
}

// registeredSafetyTargets idempotently registers the scenario's conventional
// backup targets (Postgres + data dir) and returns their stable names. A
// substrate error returns a note (not an error) so population degrades cleanly.
func registeredSafetyTargets(ctx context.Context, scenario string) ([]string, string) {
	out, err := runCommand(ctx, "data-backup-manager", "safety", "register-targets", "--scenario", scenario, "--json")
	if err != nil {
		return nil, "shadow data population skipped — register-targets failed: " + firstLine(err.Error())
	}
	var resp struct {
		Registered []struct {
			Name string `json:"name"`
		} `json:"registered"`
	}
	if json.Unmarshal(out, &resp) != nil {
		return nil, "shadow data population skipped — register-targets output unparseable"
	}
	names := make([]string, 0, len(resp.Registered))
	for _, r := range resp.Registered {
		if n := strings.TrimSpace(r.Name); n != "" {
			names = append(names, n)
		}
	}
	return names, ""
}

// waitForSafetyRun polls `runs get` until the backup run reaches a terminal state
// (its snapshots are only safe to restore from once it has finished). Returns
// true on terminal, false when the attempt budget is exhausted.
func waitForSafetyRun(ctx context.Context, runID string) bool {
	for attempt := 0; attempt < populateMaxAttempts; attempt++ {
		out, err := runCommand(ctx, "data-backup-manager", "runs", "get", runID, "--json")
		if err == nil {
			var resp struct {
				Run struct {
					Status string `json:"status"`
				} `json:"run"`
			}
			if json.Unmarshal(out, &resp) == nil && safetyRunTerminal(resp.Run.Status) {
				return true
			}
		}
		sleepFn(populatePollInterval)
	}
	return false
}

// safetyRunTerminal reports whether a data-backup-manager run status (the
// protojson RunStatus enum) is terminal — finished, with or without failures.
func safetyRunTerminal(status string) bool {
	switch status {
	case "RUN_STATUS_COMPLETED", "RUN_STATUS_PARTIAL_FAILED", "RUN_STATUS_FAILED":
		return true
	default:
		return false
	}
}

// shadowTargetMappings resolves the shadow instance's SSOT storage namespaces via
// the trusted-base `vrooli recovery namespace` query and maps each registered
// target name to its shadow location ("postgres" → the shadow DB, "data" → the
// shadow data dir). The result is the comma-separated `name=location` list
// `safety populate-shadow --mappings` consumes; unknown names or empty locations
// are dropped. GCT cannot import the platform InstanceKey SSOT, so the floor
// query is the only place these strings are derived.
func shadowTargetMappings(ctx context.Context, scenario, variant string, targetNames []string) string {
	ns, err := cliClient.RecoveryNamespace(ctx, scenario, variant)
	if err != nil {
		return ""
	}
	locByTarget := map[string]string{
		"postgres": strings.TrimSpace(ns.GetPostgresDb()),
		"data":     strings.TrimSpace(ns.GetDataDir()),
	}
	pairs := make([]string, 0, len(targetNames))
	for _, name := range targetNames {
		if loc := locByTarget[name]; loc != "" {
			pairs = append(pairs, name+"="+loc)
		}
	}
	return strings.Join(pairs, ",")
}

// populateShadowFromRun restores the resolved targets' snapshots from the given
// terminal safety run into their shadow namespaces. Each restore runs
// asynchronously inside the substrate; this call only enqueues them, matching how
// `baseline check` later validates against whatever has landed.
func populateShadowFromRun(ctx context.Context, scenario, runID, mappings string) string {
	if _, err := runCommand(ctx, "data-backup-manager", "safety", "populate-shadow",
		"--scenario", scenario, "--run-id", runID, "--mappings", mappings, "--json"); err != nil {
		return "shadow data population failed — populate-shadow: " + firstLine(err.Error())
	}
	return "shadow data populated from safety run " + runID
}

// firstLine trims a (possibly multi-line) error to its first line so step notes
// stay readable.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

// ---- check ---------------------------------------------------------------

type checkResult struct {
	Scenario string `json:"scenario"`
	Slug     string `json:"slug"`
	Mode     string `json:"mode"`
	Verdict  string `json:"verdict"`
	Guidance string `json:"guidance"`
}

func runCheckCmd(core *cliapp.ScenarioApp, args []string) error {
	var scenario, slug string
	var jsonOut bool
	fs := newFlagSet("baseline check")
	fs.StringVar(&scenario, "scenario", "", "Scenario slug (required)")
	fs.StringVar(&slug, "name", defaultEngagementSlug, "Engagement slug")
	fs.BoolVar(&jsonOut, "json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	res, err := checkEngagement(core, scenario, slug)
	if err != nil {
		return err
	}
	if jsonOut {
		if err := printJSON(res); err != nil {
			return err
		}
	} else {
		printCheck(res)
	}
	os.Exit(exitCodeForVerdict(res.Verdict))
	return nil
}

// checkEngagement validates the engagement target against its anchor, renews the
// lease, and returns mode-aware guidance. No os.Exit here so it is unit-testable.
func checkEngagement(core *cliapp.ScenarioApp, scenario, slug string) (checkResult, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return checkResult{}, fmt.Errorf("--scenario is required")
	}
	if strings.TrimSpace(slug) == "" {
		slug = defaultEngagementSlug
	}
	ctx := context.Background()
	eng, err := readEngagement(ctx, scenario, slug)
	if err != nil {
		return checkResult{}, err
	}
	if strings.TrimSpace(eng.AnchorBaselineName) == "" {
		return checkResult{}, fmt.Errorf("engagement %s/%s has no anchor baseline — `baseline check` needs one (start without --no-anchor)", scenario, slug)
	}
	verdict, err := diffAnchor(core, ctx, scenario, eng.AnchorBaselineName)
	if err != nil {
		return checkResult{}, fmt.Errorf("diff anchor %q: %w", eng.AnchorBaselineName, err)
	}
	// Renew the lease (touch-on-access) — best-effort, never fails the check.
	_, _ = runCommand(ctx, "vrooli", "recovery", "touch", "--scenario", scenario, "--slug", slug)

	return checkResult{
		Scenario: scenario, Slug: slug, Mode: eng.Mode, Verdict: verdict,
		Guidance: guidanceFor(eng.Mode, verdict),
	}, nil
}

// guidanceFor is the mode-aware steering surface (signal-and-feedback-surface-design).
func guidanceFor(mode, verdict string) string {
	switch verdict {
	case "regression":
		return "regressions detected — fix them before keeping this work; re-run `baseline check` until clean, or `baseline abandon` to roll back"
	case "not-comparable":
		return "anchor not comparable to the current tree — re-capture the engagement (or its anchor) before relying on the diff"
	default:
		if mode == modeShadow {
			return "validated clean in shadow — ready: `baseline promote` to keep it, or `baseline abandon` to tear the shadow down"
		}
		return "clean — `baseline promote` to accept the live edits, or `baseline abandon` to roll the working tree back to the restore point"
	}
}

// ---- status --------------------------------------------------------------

func runStatusCmd(core *cliapp.ScenarioApp, args []string) error {
	var jsonOut bool
	fs := newFlagSet("baseline status")
	fs.BoolVar(&jsonOut, "json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	engagements, err := listEngagements(context.Background())
	if err != nil {
		return err
	}
	if jsonOut {
		return printJSON(map[string]any{"engagements": engagements})
	}
	printStatus(engagements)
	return nil
}

func listEngagements(ctx context.Context) ([]engagementView, error) {
	resp, err := cliClient.RecoveryList(ctx)
	if err != nil {
		return nil, fmt.Errorf("list engagements: %w", err)
	}
	views := make([]engagementView, 0, len(resp.GetEngagements()))
	for _, e := range resp.GetEngagements() {
		views = append(views, engagementFromProto(e))
	}
	return views, nil
}

// ---- abandon -------------------------------------------------------------

type abandonResult struct {
	Scenario string `json:"scenario"`
	Slug     string `json:"slug"`
	Mode     string `json:"mode"`
	Action   string `json:"action"`
}

func runAbandonCmd(core *cliapp.ScenarioApp, args []string) error {
	var scenario, slug string
	var jsonOut bool
	fs := newFlagSet("baseline abandon")
	fs.StringVar(&scenario, "scenario", "", "Scenario slug (required)")
	fs.StringVar(&slug, "name", defaultEngagementSlug, "Engagement slug")
	fs.BoolVar(&jsonOut, "json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	res, err := abandonEngagement(core, scenario, slug)
	if err != nil {
		return err
	}
	if jsonOut {
		return printJSON(res)
	}
	fmt.Printf("✓ abandoned %s/%s (%s): %s\n", res.Scenario, res.Slug, res.Mode, res.Action)
	return nil
}

// abandonEngagement is the "discard the Candidate, keep the Baseline" give-up.
// Both modes overlay the Baseline (the restore point) back onto the working tree
// — the edited location — so the in-progress candidate is thrown away (git-free
// undo; post-capture untracked files are left = dirty work parked). They differ
// only in how the serving instance is handled:
//   - shadow: live has been serving the Baseline from the restore-point copy the
//     whole time and never ran the candidate, so it is left untouched (no
//     restart); its next restart resolves to the now-restored working tree. The
//     shadow instance that WAS running the candidate is torn down.
//   - live: live ran the edited working tree in place, so after the restore it is
//     restarted to pick the Baseline back up.
func abandonEngagement(core *cliapp.ScenarioApp, scenario, slug string) (abandonResult, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return abandonResult{}, fmt.Errorf("--scenario is required")
	}
	if strings.TrimSpace(slug) == "" {
		slug = defaultEngagementSlug
	}
	ctx := context.Background()
	eng, err := readEngagement(ctx, scenario, slug)
	if err != nil {
		return abandonResult{}, err
	}
	res := abandonResult{Scenario: scenario, Slug: slug, Mode: eng.Mode}
	if eng.Mode == modeShadow {
		// Stop the shadow first (it is reading the working tree as its CWD), then
		// discard the candidate by restoring the baseline over the working tree.
		variant := eng.Variant
		if variant == "" {
			variant = modeShadow
		}
		if _, err := runCommand(ctx, "vrooli", "scenario", "stop", scenario, "--instance", variant); err != nil {
			return abandonResult{}, fmt.Errorf("tear down shadow %s@%s: %w", scenario, variant, err)
		}
		if _, err := runCommand(ctx, "vrooli", "recovery", "restore", "--scenario", scenario, "--slug", slug); err != nil {
			return abandonResult{}, fmt.Errorf("restore baseline over working tree: %w", err)
		}
		res.Action = "candidate discarded (working tree restored from baseline); shadow torn down; live untouched"
	} else {
		if _, err := runCommand(ctx, "vrooli", "recovery", "restore", "--scenario", scenario, "--slug", slug); err != nil {
			return abandonResult{}, fmt.Errorf("restore baseline over working tree: %w", err)
		}
		// Rebuild from the restored tree so the running live process picks it up.
		if _, err := runCommand(ctx, "vrooli", "scenario", "restart", scenario); err != nil {
			return abandonResult{}, fmt.Errorf("restart live after restore: %w", err)
		}
		res.Action = "live edits discarded (working tree restored from baseline); live restarted"
	}
	// Drop the engagement (restore point + manifest) — idempotent.
	if _, err := runCommand(ctx, "vrooli", "recovery", "clean", "--scenario", scenario, "--slug", slug); err != nil {
		return abandonResult{}, fmt.Errorf("clean engagement: %w", err)
	}
	return res, nil
}

// ---- gc ------------------------------------------------------------------

type gcResult struct {
	Reaped  []string `json:"reaped"`
	Skipped []string `json:"skipped"`
}

func runGCCmd(core *cliapp.ScenarioApp, args []string) error {
	var force, jsonOut bool
	fs := newFlagSet("baseline gc")
	fs.BoolVar(&force, "force", false, "Reap every active engagement, not only expired/orphaned ones")
	fs.BoolVar(&jsonOut, "json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	res, err := gcEngagements(context.Background(), force)
	if err != nil {
		return err
	}
	if jsonOut {
		return printJSON(res)
	}
	fmt.Printf("reaped %d, skipped %d\n", len(res.Reaped), len(res.Skipped))
	for _, r := range res.Reaped {
		fmt.Printf("  ✓ %s\n", r)
	}
	return nil
}

// gcEngagements force-reaps stale/orphaned shadows: for each expired (or every,
// with --force) engagement, tear down any shadow instance and drop its restore
// point + manifest. The manual escape hatch over the reaper sweep.
func gcEngagements(ctx context.Context, force bool) (gcResult, error) {
	engagements, err := listEngagements(ctx)
	if err != nil {
		return gcResult{}, err
	}
	var res gcResult
	for _, e := range engagements {
		ref := e.Scenario + "/" + e.Slug
		if !force && !e.Expired {
			res.Skipped = append(res.Skipped, ref)
			continue
		}
		if e.Mode == modeShadow {
			variant := e.Variant
			if variant == "" {
				variant = modeShadow
			}
			// Best-effort: a shadow that is already gone is fine.
			_, _ = runCommand(ctx, "vrooli", "scenario", "stop", e.Scenario, "--instance", variant)
		}
		if _, err := runCommand(ctx, "vrooli", "recovery", "clean", "--scenario", e.Scenario, "--slug", e.Slug); err != nil {
			return res, fmt.Errorf("clean %s: %w", ref, err)
		}
		res.Reaped = append(res.Reaped, ref)
	}
	return res, nil
}

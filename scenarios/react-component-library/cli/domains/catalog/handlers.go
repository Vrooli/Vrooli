package catalog

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/envkit-go"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
	catalogconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog/catalog_v1connect"
)

type handlers struct {
	client catalogconnect.CatalogServiceClient
}

func (h *handlers) corpusReport(_ cliapp.RunContext) error {
	root, err := scenarioRoot()
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "run", "./cmd/corpus-report", "--root", filepath.Join(root, "..", "..")) // #nosec G204 -- fixed command and repository-owned root
	cmd.Dir = filepath.Join(root, "api")
	cmd.Env = envkit.Toolchain(envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, nil), envkit.ToolchainOptions{})
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (h *handlers) build(ctx cliapp.RunContext) error {
	if ctx.BoolFlag("list-stages") {
		for _, stage := range []string{
			"sync-exports      — synchronize package exports",
			"generate-manifests — derive component manifests",
			"release-hashes    — update immutable release hashes",
			"story-contracts   — derive story contracts",
			"dependency-locks  — resolve major-line dependency locks",
		} {
			fmt.Println(stage)
		}
		return nil
	}
	root, err := scenarioRoot()
	if err != nil {
		return err
	}
	args := []string{"tooling/catalog-build.mjs"}
	if ctx.BoolFlag("check") {
		args = append(args, "--check")
	}
	cmd := exec.Command("node", args...) // #nosec G204 -- fixed repository-owned generator
	cmd.Dir = filepath.Join(root, "..", "..", "packages", "react-component-library")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func scenarioRoot() (string, error) {
	starts := []string{}
	if cwd, err := os.Getwd(); err == nil {
		starts = append(starts, cwd)
	}
	if envRoot := os.Getenv("SCENARIO_PATH"); envRoot != "" {
		starts = append(starts, envRoot)
	}
	for _, start := range starts {
		for current := filepath.Clean(start); ; current = filepath.Dir(current) {
			if _, err := os.Stat(filepath.Join(current, "api", "cmd", "corpus-report", "main.go")); err == nil {
				return current, nil
			}
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
		}
	}
	return "", fmt.Errorf("cannot locate react-component-library scenario root")
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	// Gate matrices are cancellable server jobs, not short metadata calls. The
	// default scenario client deadline can cancel a valid matrix while it is
	// persisting its final evidence row.
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, 20*time.Minute)
	return &handlers{
		client: catalogconnect.NewCatalogServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) readiness(ctx cliapp.RunContext) error {
	floor := readinessFloor(ctx)
	resp, err := h.client.GetReadiness(context.Background(), connect.NewRequest(&catalogv1.GetReadinessRequest{Floor: floor}))
	if err != nil {
		return cliapp.WrapAPIError("get catalog readiness", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no catalog readiness")
	}
	msg := resp.Msg
	status := []string{
		fmt.Sprintf("Verdict: %s", msg.GetVerdict()),
		fmt.Sprintf("Evidence run: %s (completed=%t; completed_at=%s)", msg.GetRun().GetRunId(), msg.GetRun().GetCompleted(), msg.GetRun().GetCompletedAt()),
		fmt.Sprintf("Floor: %s; achieved: %s; gap: %d", msg.GetConfig().GetDeclaredFloor(), msg.GetConfig().GetAchievedRung(), msg.GetConfig().GetRungGap()),
	}
	if msg.GetTriageOmittedCount() > 0 {
		status = append(status, fmt.Sprintf("Triage: showing %d row(s); %d omitted", len(msg.GetTriage()), msg.GetTriageOmittedCount()))
	}
	groups := map[string][]string{}
	for _, row := range msg.GetTriage() {
		groups[row.GetGate()] = append(groups[row.GetGate()], fmt.Sprintf("%s: %s (blast=%d weight=%.2f)", row.GetAssetId(), row.GetMessage(), row.GetBlocksDownstream(), row.GetWeight()))
	}
	triage := make([]cliapp.TriageGroup, 0, len(groups))
	for gate, items := range groups {
		triage = append(triage, cliapp.TriageGroup{Heading: gate, Items: items})
	}
	sort.Slice(triage, func(i, j int) bool { return triage[i].Heading < triage[j].Heading })
	return ctx.RenderOperational(cliapp.OperationalReport{Status: status, Triage: triage, NextSteps: msg.GetNextSteps()})
}

func readinessFloor(ctx cliapp.RunContext) string {
	if ctx.FlagDeclared("floor") {
		return ctx.Flag("floor")
	}
	return ""
}

func (h *handlers) gate(ctx cliapp.RunContext) error {
	gate := ctx.Positional("gate")
	if gate == "" && !ctx.BoolFlag("all") {
		return fmt.Errorf("catalog gates requires a gate name or --all; use --help to see valid gates")
	}
	resp, err := h.client.RunGate(context.Background(), connect.NewRequest(&catalogv1.RunGateRequest{Gate: gate, All: ctx.BoolFlag("all"), AssetId: ctx.Flag("asset-id"), CalibrationOnly: ctx.BoolFlag("calibration-only"), IncludeAdvisory: ctx.BoolFlag("include-advisory")}))
	if err != nil {
		return cliapp.WrapAPIError("run catalog gate", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no catalog gate result")
	}
	// Findings render grouped by remediation, not one flat list. A gate that
	// reports 410 assets failing for one reason should print that reason once
	// and then the affected locations — repeating an identical paragraph per
	// row buries the single fact the reader needs under its own restatement.
	type group struct {
		code, severity, scope, owner, remediation, docs string
		blocking                                        bool
		locations                                       []string
	}
	order := make([]string, 0, len(resp.Msg.Findings))
	groups := map[string]*group{}
	for _, finding := range resp.Msg.Findings {
		location := finding.AssetId
		if finding.File != "" {
			location = finding.File
			if finding.Line > 0 {
				location = fmt.Sprintf("%s:%d", finding.File, finding.Line)
			}
		}
		key := finding.Code + "\x00" + finding.Remediation
		existing, seen := groups[key]
		if !seen {
			existing = &group{code: finding.Code, severity: findingSeverityLabel(finding), scope: findingScopeLabel(finding), owner: finding.Owner, blocking: finding.Blocking, remediation: finding.Remediation, docs: finding.DocsRef}
			groups[key] = existing
			order = append(order, key)
		}
		entry := location
		if finding.Message != "" {
			entry += ": " + finding.Message
		}
		existing.locations = append(existing.locations, entry)
	}

	const maxLocations = 20
	findings := make([]string, 0, len(order))
	for _, key := range order {
		g := groups[key]
		entry := fmt.Sprintf("[%s] %s — %d location(s)", g.code, g.severity, len(g.locations))
		if g.scope != "" || g.owner != "" {
			entry += fmt.Sprintf(" (%s%s)", g.scope, ownerSuffix(g.owner))
		}
		shown := g.locations
		if len(shown) > maxLocations {
			shown = shown[:maxLocations]
		}
		for _, location := range shown {
			entry += "\n    " + location
		}
		if len(g.locations) > len(shown) {
			// Never silently truncate: a hidden remainder reads as full
			// coverage. Say how many were withheld and how to see them.
			entry += fmt.Sprintf("\n    … and %d more (use --json for the complete list)", len(g.locations)-len(shown))
		}
		if g.remediation != "" {
			entry += "\n  fix: " + g.remediation
		}
		if g.docs != "" {
			entry += "\n  docs: " + g.docs
		}
		findings = append(findings, entry)
	}
	for _, calibration := range resp.Msg.Calibration {
		fixture := calibration.Fixture
		if fixture == "" {
			fixture = "(missing fixture)"
		}
		findings = append(findings, fmt.Sprintf("[calibration.%s] %s — %s (required %s, observed %s)", calibration.Status, fixture, calibration.Message, calibration.RequiredFailureCode, calibration.ObservedFailureCode))
	}
	blocking, advisory := 0, 0
	for _, finding := range resp.Msg.Findings {
		if finding.Blocking || finding.Severity == "error" {
			blocking++
		} else {
			advisory++
		}
	}
	summary := fmt.Sprintf("Gate %s: inspected %d file(s), %d finding(s)", resp.Msg.Gate, resp.Msg.InspectedFiles, len(resp.Msg.Findings))
	if len(resp.Msg.SurfaceVerdictCounts) > 0 {
		verdicts := make([]string, 0, len(resp.Msg.SurfaceVerdictCounts))
		for verdict, count := range resp.Msg.SurfaceVerdictCounts {
			verdicts = append(verdicts, fmt.Sprintf("%s=%d", verdict, count))
		}
		sort.Strings(verdicts)
		summary += "; surface verdicts " + fmt.Sprintf("%v", verdicts)
	}
	if len(resp.Msg.CompositionScores) > 0 || resp.Msg.CompositionMedian > 0 || resp.Msg.BespokeEscapeCount > 0 {
		summary += fmt.Sprintf("; composition median %.3f across %d captured asset(s), bespoke escapes %d", resp.Msg.CompositionMedian, len(resp.Msg.CompositionScores), resp.Msg.BespokeEscapeCount)
	}
	if len(resp.Msg.Findings) > 0 {
		summary += fmt.Sprintf(" (%d blocking, %d advisory) in %d distinct cause(s)", blocking, advisory, len(findings))
	}
	lines := []string{summary + "."}
	if resp.Msg.InspectedFiles == 0 {
		lines = append(lines, "WARNING: this gate inspected zero files. A gate that inspects nothing cannot pass; treat this as a runner fault, not a clean result.")
	}
	if resp.Msg.NonDiscriminating {
		lines = append(lines, "QUARANTINED: calibration did not discriminate; corpus evidence was downgraded to unmeasured.")
	}
	renderErr := cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        lines,
		ResultsHeading: "Findings",
		Results:        findings,
	})
	if renderErr != nil {
		return renderErr
	}
	if resp.Msg.NonDiscriminating {
		return fmt.Errorf("catalog gate %s is non-discriminating", resp.Msg.Gate)
	}
	return nil
}

func findingSeverityLabel(finding *catalogv1.GateFinding) string {
	if finding.Blocking || finding.SeverityClass == catalogv1.FindingSeverity_FINDING_SEVERITY_BLOCKING || finding.Severity == "error" {
		return "error"
	}
	if finding.Severity != "" {
		return finding.Severity
	}
	switch finding.SeverityClass {
	case catalogv1.FindingSeverity_FINDING_SEVERITY_WARNING:
		return "warning"
	case catalogv1.FindingSeverity_FINDING_SEVERITY_INFO:
		return "info"
	default:
		return "warning"
	}
}

func findingScopeLabel(finding *catalogv1.GateFinding) string {
	switch finding.Scope {
	case catalogv1.FindingScope_FINDING_SCOPE_CORPUS:
		return "corpus"
	case catalogv1.FindingScope_FINDING_SCOPE_ASSET:
		return "asset"
	default:
		return ""
	}
}

func ownerSuffix(owner string) string {
	if owner == "" {
		return ""
	}
	return ", owner " + owner
}

func (h *handlers) graph(ctx cliapp.RunContext) error {
	resp, err := h.client.GetAssetRelationships(context.Background(), connect.NewRequest(&catalogv1.GetAssetRelationshipsRequest{AssetId: ctx.Positional("asset-id")}))
	if err != nil {
		return cliapp.WrapAPIError("get catalog graph", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Relationships == nil {
		return fmt.Errorf("server returned no catalog graph")
	}
	r := resp.Msg.Relationships
	results := []string{fmt.Sprintf("depends-on: %d direct, %d in closure", len(r.DirectDependencies), len(r.Closure)), fmt.Sprintf("used-by: %d direct, %d transitive", len(r.DirectDependents), len(r.TransitiveDependents))}
	for _, band := range r.ClosureBands {
		results = append(results, fmt.Sprintf("rung %d (%s): %d", band.Rung, band.RungName, band.Count))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Asset graph: %s; blast radius %d.", r.Root.AssetId, len(r.TransitiveDependents))}, ResultsHeading: "Relationships", Results: results})
}

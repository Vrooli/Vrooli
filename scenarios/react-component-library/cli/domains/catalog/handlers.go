package catalog

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
	catalogconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog/catalog_v1connect"
)

type handlers struct {
	client catalogconnect.CatalogServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: catalogconnect.NewCatalogServiceClient(httpClient, baseURL)}
}

func (h *handlers) coverage(ctx cliapp.RunContext) error {
	resp, err := h.client.GetCoverage(context.Background(), connect.NewRequest(&catalogv1.GetCoverageRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get catalog coverage", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Report == nil {
		return fmt.Errorf("server returned no catalog coverage")
	}
	r := resp.Msg.Report
	summary := fmt.Sprintf("Catalog completion %.1f%% (%d/%d); mandatory gates %.1f%% (%d/%d); evidence pass %d, fail %d, unmeasured %d; weighted quality %.1f%%; production-ready %.1f%%.",
		r.Maturity.GetCatalogCompletion().GetRatio()*100, r.Maturity.GetCatalogCompletion().GetNumerator(), r.Maturity.GetCatalogCompletion().GetDenominator(),
		r.Maturity.GetMandatoryGateCoverage().GetRatio()*100, r.Maturity.GetMandatoryGateCoverage().GetNumerator(), r.Maturity.GetMandatoryGateCoverage().GetDenominator(),
		r.Maturity.GetPassEvidence(), r.Maturity.GetFailEvidence(), r.Maturity.GetUnmeasuredEvidence(),
		r.Maturity.GetWeightedQuality().GetRatio()*100, r.Maturity.GetProductionReadyCoverage().GetRatio()*100)
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{summary}, ResultsHeading: "Maturity distribution", Results: formatMaturity(r.Maturity.GetByRung())})
}

func (h *handlers) next(ctx cliapp.RunContext) error {
	resp, err := h.client.ListNextWork(context.Background(), connect.NewRequest(&catalogv1.ListNextWorkRequest{Limit: 10, Lane: ctx.Flag("lane")}))
	if err != nil {
		return cliapp.WrapAPIError("get catalog next work", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no catalog next work")
	}
	rows := make([]string, 0, len(resp.Msg.Rows))
	for _, row := range resp.Msg.Rows {
		rows = append(rows, fmt.Sprintf("%s [%s/%s] %s -> %s (blocks %d)", row.AssetId, row.Platform, row.Target, row.Achieved, row.Name, row.BlocksDownstream))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Next work: %d target(s).", len(rows))}, ResultsHeading: "Ranked next work", Results: rows})
}

func (h *handlers) scoreHistory(ctx cliapp.RunContext) error {
	resp, err := h.client.GetScoreHistory(context.Background(), connect.NewRequest(&catalogv1.GetScoreHistoryRequest{Since: ctx.Flag("since")}))
	if err != nil {
		return cliapp.WrapAPIError("get catalog score history", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no catalog score history")
	}
	results := make([]string, 0, len(resp.Msg.Points))
	for _, point := range resp.Msg.Points {
		results = append(results, fmt.Sprintf("%s score %.1f%%; at 100%% %d; below 50%% %d", point.RecordedAt, point.Score, point.AssetsAt_100, point.AssetsBelow_50))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Score history: %d day(s).", len(results))}, ResultsHeading: "Daily score", Results: results})
}

func (h *handlers) health(ctx cliapp.RunContext) error {
	resp, err := h.client.GetHealthOverview(context.Background(), connect.NewRequest(&catalogv1.GetHealthOverviewRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get catalog health", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no catalog health")
	}
	results := make([]string, 0, len(resp.Msg.Nodes))
	for _, node := range resp.Msg.Nodes {
		if node.Asset != nil {
			results = append(results, fmt.Sprintf("%s %s score %.1f%% stale %.1fd", node.Asset.AssetId, node.Health, node.Score, node.StalenessDays))
		}
	}
	for _, mismatch := range resp.Msg.KindMismatches {
		results = append(results, fmt.Sprintf("kind mismatch %s: declared %s, derived %s", mismatch.AssetId, mismatch.DeclaredKind, mismatch.DerivedKind))
	}
	for _, capability := range resp.Msg.Coverage.GetCapabilityCoverage() {
		if !capability.GetCheckable() {
			results = append(results, fmt.Sprintf("capability %s: %s; declared by %d asset(s); %s", capability.GetCapability(), capability.GetStatus(), capability.GetDeclaredAssetCount(), strings.Join(capability.GetBlockers(), "; ")))
		}
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Health nodes: %d; promote: %d; quarantined gates: %d; kind mismatches: %d; instrument-moved: %d; composition median %.3f across %d captured asset(s); bespoke escapes %d; declared capabilities %d across %d asset(s); uncheckable %d asset(s); unmeasured %d asset(s).", len(resp.Msg.Nodes), len(resp.Msg.Promote), len(resp.Msg.QuarantinedGates), resp.Msg.KindMismatchCount, resp.Msg.InstrumentMovedCount, resp.Msg.Coverage.GetCompositionMedian(), len(resp.Msg.Coverage.GetCompositionScores()), resp.Msg.Coverage.GetBespokeEscapeCount(), resp.Msg.Coverage.GetCapabilityDeclarationCount(), resp.Msg.Coverage.GetDeclaredCapabilityAssetCount(), resp.Msg.Coverage.GetDeclaredUncheckableAssetCount(), resp.Msg.Coverage.GetUnmeasuredCapabilityAssetCount())}, ResultsHeading: "Asset health", Results: results})
}

func (h *handlers) evidence(ctx cliapp.RunContext) error {
	if ctx.Positional("action") != "capture" {
		return fmt.Errorf("usage: catalog evidence capture <asset-id>")
	}
	resp, err := h.client.CaptureEvidence(context.Background(), connect.NewRequest(&catalogv1.CaptureEvidenceRequest{AssetId: ctx.Positional("asset-id"), ChangedOnly: ctx.Flag("changed-only") == "true"}))
	if err != nil {
		return cliapp.WrapAPIError("capture catalog evidence", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no catalog evidence capture")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Captured %d evidence row(s) for %s.", resp.Msg.RowsWritten, resp.Msg.AssetId)}, ResultsHeading: "Capture", Results: []string{resp.Msg.CaptureDirectory, resp.Msg.WorkbenchUrl}})
}

func (h *handlers) gate(ctx cliapp.RunContext) error {
	gate := ctx.Positional("gate")
	resp, err := h.client.RunGate(context.Background(), connect.NewRequest(&catalogv1.RunGateRequest{Gate: gate, All: ctx.BoolFlag("all"), CalibrationOnly: ctx.BoolFlag("calibration-only")}))
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
		code, severity, remediation, docs string
		locations                         []string
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
			existing = &group{code: finding.Code, severity: finding.Severity, remediation: finding.Remediation, docs: finding.DocsRef}
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
		if finding.Severity == "error" {
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

func (h *handlers) structure(ctx cliapp.RunContext) error {
	resp, err := h.client.GetCatalogStructure(context.Background(), connect.NewRequest(&catalogv1.GetCatalogStructureRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get catalog structure", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Structure == nil {
		return fmt.Errorf("server returned no catalog structure")
	}
	structure := resp.Msg.Structure
	results := make([]string, 0, len(structure.Population)+len(structure.Invariants)+len(structure.BlastRadius))
	for _, row := range structure.Population {
		results = append(results, fmt.Sprintf("rung %d (%s): %d", row.Rung, row.RungName, row.Count))
	}
	for _, invariant := range structure.Invariants {
		results = append(results, fmt.Sprintf("%s: %s", invariant.Label, invariant.Status))
	}
	for _, row := range structure.BlastRadius {
		if row.Asset != nil {
			results = append(results, fmt.Sprintf("blast radius %s: %d", row.Asset.AssetId, row.TransitiveDependentCount))
		}
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{"Catalog structure"}, ResultsHeading: "Structure", Results: results})
}

func (h *handlers) reconcile(ctx cliapp.RunContext) error {
	resp, err := h.client.ReconcileGraph(context.Background(), connect.NewRequest(&catalogv1.ReconcileGraphRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("reconcile catalog graph", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no graph reconciliation")
	}
	counts := map[string]int32{}
	if resp.Msg.Distribution != nil {
		counts = resp.Msg.Distribution.Counts
	}
	verdicts := make([]string, 0, len(counts))
	for verdict := range counts {
		verdicts = append(verdicts, verdict)
	}
	sort.Strings(verdicts)
	results := make([]string, 0, len(counts))
	for _, verdict := range verdicts {
		results = append(results, fmt.Sprintf("%s: %d", verdict, counts[verdict]))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Graph reconciliation: %d assets.", len(resp.Msg.Assets))}, ResultsHeading: "Verdict distribution", Results: results})
}

func (h *handlers) ports(ctx cliapp.RunContext) error {
	resp, err := h.client.GetAssetPortContract(context.Background(), connect.NewRequest(&catalogv1.GetAssetPortContractRequest{AssetId: ctx.Positional("asset-id")}))
	if err != nil {
		return cliapp.WrapAPIError("get catalog ports", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Contract == nil {
		return fmt.Errorf("server returned no asset port contract")
	}
	contract := resp.Msg.Contract
	results := make([]string, 0, len(contract.UnmetPorts))
	for _, port := range contract.UnmetPorts {
		results = append(results, fmt.Sprintf("%s: demanded by %d asset(s), %d candidate satisfier(s)", port.CapabilityId, len(port.DemandingAssets), len(port.CandidateSatisfiers)))
	}
	if contract.SelfContained {
		results = append(results, "self-contained: closure satisfies all host ports")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Asset port contract: %d closure assets.", contract.ClosureCount)}, ResultsHeading: "Unmet ports", Results: results})
}

func formatMaturity(values map[string]int32) []string {
	rows := make([]string, 0, len(values))
	for key, value := range values {
		rows = append(rows, fmt.Sprintf("%s: %d", key, value))
	}
	return rows
}

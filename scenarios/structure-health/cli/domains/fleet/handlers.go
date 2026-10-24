package fleet

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	repocontract "github.com/vrooli/repo-contract-go"
	fleetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/fleet"
	fleetconnect "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/fleet/fleet_v1connect"

	"structure-health/internal/packs/fleetpack"
)

type handlers struct {
	client fleetconnect.FleetServiceClient
}

// census measures the manifest fleet directly from disk. It is intentionally
// local and does not start the API or Code Facts: this command is the
// before/after evidence instrument for migrations that can themselves change
// API startup behavior. Its declared output flags keep large evidence out of
// shell redirections and make the artifact paths part of the command contract.
func (h *handlers) census(ctx cliapp.RunContext) error {
	repoRoot, err := repocontract.ResolveRepoRoot()
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	outputPath := strings.TrimSpace(ctx.Flag("output"))
	stepsOutputPath := strings.TrimSpace(ctx.Flag("steps-output"))
	includeSteps := ctx.BoolFlag("include-steps") || stepsOutputPath != ""
	report, err := fleetpack.Census(repoRoot, includeSteps)
	if err != nil {
		return fmt.Errorf("census scenario manifests: %w", err)
	}
	if outputPath != "" {
		outputReport := report
		if stepsOutputPath != "" {
			outputReport.StepsInventory = nil
		}
		if err := writeJSONArtifact(repoRoot, outputPath, outputReport); err != nil {
			return fmt.Errorf("write census output: %w", err)
		}
	}
	if stepsOutputPath != "" {
		if err := writeJSONArtifact(repoRoot, stepsOutputPath, report.StepsInventory); err != nil {
			return fmt.Errorf("write lifecycle-step inventory: %w", err)
		}
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return ctx.RenderList(cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("%d manifest(s) across %d scenario directories; %d total lifecycle steps, %d live.", report.ManifestCount, report.ScenarioDirectoryCount, report.Lifecycle.TotalSteps, report.Lifecycle.LiveSteps),
			fmt.Sprintf("%d canonical-schema violation(s) across %d manifest(s).", report.SchemaValidation.ViolationCount, report.SchemaValidation.FailingManifestCount),
			fmt.Sprintf("%d manifest(s) declare %d component(s).", report.Components.AdoptingManifestCount, report.Components.ComponentCount),
		},
		ResultsHeading: "Migration census",
		Results: []string{
			fmt.Sprintf("cohorts: template-current=%d, template-plus-extras=%d, light-drift=%d, heavy-drift=%d, pre-template=%d, no-manifest=%d", len(report.Cohorts.TemplateCurrent), len(report.Cohorts.TemplatePlusExtras), len(report.Cohorts.LightDrift), len(report.Cohorts.HeavyDrift), len(report.Cohorts.PreTemplate), len(report.Cohorts.NoManifest)),
			fmt.Sprintf("ports: %d declarations, %d convention violations, %d ranged, %d pinned", report.Ports.DeclarationCount, len(report.Ports.ConventionViolations), report.Ports.RangeAllocatedCount, report.Ports.PinnedCount),
			fmt.Sprintf("peers: %d edges from %d scenarios to %d targets", report.PeerDependencies.EdgeCount, report.PeerDependencies.DeclaringScenarioCount, report.PeerDependencies.DistinctTargetCount),
			fmt.Sprintf("shell files: scenarios=%d, resources=%d; lifecycle references=%d", report.ShellFiles.ScenarioCount, report.ShellFiles.ResourceCount, len(report.ShellFiles.LifecycleInvokedReferences)),
		},
		RetrievalHints: []string{"Use `--json` for machine-readable output; use `--output` and `--steps-output` to write stable migration evidence."},
	})
}

func writeJSONArtifact(repoRoot, requestedPath string, value any) error {
	path := requestedPath
	if !filepath.IsAbs(path) {
		path = filepath.Join(repoRoot, filepath.FromSlash(path))
	}
	path = filepath.Clean(path)
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent directory for %s: %w", path, err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: fleetconnect.NewFleetServiceClient(httpClient, baseURL)}
}

// scan grades the fleet (or a requested subset) and renders the rollup.
func (h *handlers) scan(ctx cliapp.RunContext) error {
	resp, err := h.client.ScanFleet(context.Background(), connect.NewRequest(&fleetv1.ScanFleetRequest{
		Scenarios: splitCSV(ctx.FlagValues("scenario")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("scan fleet structure conformance", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no fleet response")
	}
	msg := resp.Msg

	summary := []string{
		fmt.Sprintf("%d scenario(s): %d passing and %d with structure errors.",
			msg.GetScenarioCount(), msg.GetPassingCount(),
			msg.GetScenarioCount()-msg.GetPassingCount()),
		fmt.Sprintf("%d auto-fixable finding(s) across the fleet.", msg.GetAutofixableTotal()),
	}
	summary = append(summary, profileLines(msg.GetProfileDistribution())...)
	summary = append(summary, ruleLines(msg.GetRuleConformance())...)

	// Offenders: scenarios with errors, worst first.
	results := make([]string, 0, len(msg.GetEntries()))
	for _, e := range msg.GetEntries() {
		if e.GetErrorCount() == 0 {
			continue
		}
		flags := make([]string, 0, 2)
		if e.GetAutofixableCount() > 0 {
			flags = append(flags, fmt.Sprintf("%d auto-fixable", e.GetAutofixableCount()))
		}
		suffix := ""
		if len(flags) > 0 {
			suffix = " [" + strings.Join(flags, ", ") + "]"
		}
		results = append(results, fmt.Sprintf("%s (%s): %d error(s), %d warning(s)%s",
			e.GetScenario(), e.GetProfileId(), e.GetErrorCount(), e.GetWarningCount(), suffix))
	}
	if len(results) == 0 {
		results = append(results, "No structure offenders — every scanned scenario is conformant.")
	}

	hints := []string{
		"`structure-health validate scenario <name> --json` - drill into one scenario's findings",
		"`structure-health fix-config run <name>` - preview auto-fixable remediations",
	}
	for _, e := range msg.GetErrors() {
		hints = append(hints, fmt.Sprintf("could not grade %s: %s", e.GetScenario(), e.GetReason()))
	}

	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Offenders",
		Results:        results,
		RetrievalHints: hints,
	})
}

func profileLines(dist []*fleetv1.ProfileDistribution) []string {
	if len(dist) == 0 {
		return nil
	}
	lines := []string{"Profiles:"}
	for _, p := range dist {
		recognized := "recognized"
		if !p.GetRecognized() {
			recognized = "unrecognized"
		}
		lines = append(lines, fmt.Sprintf("  • %s [%s]: %d", p.GetProfileId(), recognized, p.GetScenarioCount()))
	}
	return lines
}

func ruleLines(rules []*fleetv1.RuleConformance) []string {
	if len(rules) == 0 {
		return nil
	}
	limit := len(rules)
	if limit > 10 {
		limit = 10
	}
	lines := []string{fmt.Sprintf("Top offending rules (%d shown of %d):", limit, len(rules))}
	for _, r := range rules[:limit] {
		fixable := ""
		if r.GetAutofixable() > 0 {
			fixable = fmt.Sprintf(", %d auto-fixable", r.GetAutofixable())
		}
		lines = append(lines, fmt.Sprintf("  • %s [%s]: %d scenario(s), %d finding(s)%s",
			r.GetCode(), r.GetWorstSeverity(), r.GetOffendingScenarios(), r.GetTotalFindings(), fixable))
	}
	return lines
}

func splitCSV(values []string) []string {
	var out []string
	for _, v := range values {
		for _, part := range strings.Split(v, ",") {
			if p := strings.TrimSpace(part); p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}

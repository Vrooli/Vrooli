package manifest

import (
	"context"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"

	manifestv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/manifest"
	manifestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/manifest/manifest_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client manifestconnect.ManifestServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: manifestconnect.NewManifestServiceClient(httpClient, baseURL),
	}
}

// validate parses + structurally checks a manifest source. When a file
// path is supplied the CLI submits its bytes; otherwise the API reads the
// scenario's default manifest path. Renders an operational report: the
// valid/invalid gate, diagnostics grouped by severity, and next steps.
func (h *handlers) validate(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	req := &manifestv1.ValidateManifestRequest{Scenario: scenario}
	if file := strings.TrimSpace(ctx.Positional("file")); file != "" {
		body, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read manifest %q: %w", file, err)
		}
		req.Source = body
		req.ContentType = contentTypeForFile(file)
	}
	resp, err := h.client.ValidateManifest(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("validate manifest for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no validate response")
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}

	status := fmt.Sprintf("Manifest for %q is VALID (no error-severity diagnostics).", scenario)
	if !resp.Msg.GetValid() {
		status = fmt.Sprintf("Manifest for %q is INVALID: error-severity diagnostics present.", scenario)
	}
	var triage []cliapp.TriageGroup
	byKind := map[string][]string{}
	order := []string{"error", "warn", "info"}
	for _, d := range resp.Msg.GetDiagnostics() {
		kind := diagnosticSeverityName(d.GetSeverity())
		line := fmt.Sprintf("%s [%s] %s", d.GetCode(), d.GetPath(), d.GetMessage())
		if d.GetLine() > 0 {
			line = fmt.Sprintf("%s:%d:%d %s", d.GetCode(), d.GetLine(), d.GetColumn(), d.GetMessage())
		}
		byKind[kind] = append(byKind[kind], line)
	}
	for _, kind := range order {
		if items := byKind[kind]; len(items) > 0 {
			triage = append(triage, cliapp.TriageGroup{Heading: kind, Items: items})
		}
	}
	return ctx.RenderOperational(cliapp.OperationalReport{
		Status: []string{status},
		Triage: triage,
		NextSteps: []string{
			fmt.Sprintf("`manifest show %s` to read the persisted definition.", scenario),
			fmt.Sprintf("`manifest list-domains %s` to see the declared domains.", scenario),
		},
	})
}

// show returns the most recently parsed manifest for a scenario.
func (h *handlers) show(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.GetManifest(context.Background(), connect.NewRequest(&manifestv1.GetManifestRequest{Scenario: scenario}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get manifest for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetManifest() == nil {
		return fmt.Errorf("server returned no manifest")
	}
	m := resp.Msg.GetManifest()
	results := []string{
		fmt.Sprintf("scenario: %s", m.GetScenario()),
		fmt.Sprintf("domains: %d", len(m.GetDomains())),
		fmt.Sprintf("shared_substrate: %s", strings.Join(m.GetSharedSubstrate(), ", ")),
		fmt.Sprintf("transitional declarations: %d", len(m.GetTransitional())),
		fmt.Sprintf("content_hash: %s", m.GetContentHash()),
	}
	for _, d := range m.GetDomains() {
		results = append(results, fmt.Sprintf("  domain %s — paths=%d allowed_deps=%s",
			d.GetName(), len(d.GetPaths()), strings.Join(d.GetAllowedDependencies(), ",")))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Persisted manifest for %q.", scenario)},
		ResultsHeading: "Definition",
		Results:        results,
		RetrievalHints: []string{fmt.Sprintf("`manifest list-domains %s` for the domain table only.", scenario)},
	})
}

// listDomains returns the declared domains without the full manifest.
func (h *handlers) listDomains(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.ListDomains(context.Background(), connect.NewRequest(&manifestv1.ListDomainsRequest{Scenario: scenario}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("list domains for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no domains response")
	}
	results := make([]string, 0, len(resp.Msg.GetDomains()))
	for _, d := range resp.Msg.GetDomains() {
		results = append(results, fmt.Sprintf("%s — paths: %s — allowed_deps: %s",
			d.GetName(), strings.Join(d.GetPaths(), ", "), strings.Join(d.GetAllowedDependencies(), ", ")))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d declared domain(s) for %q.", len(resp.Msg.GetDomains()), scenario)},
		ResultsHeading: "Domains",
		Results:        results,
	})
}

func contentTypeForFile(path string) string {
	switch {
	case strings.HasSuffix(path, ".json"):
		return "application/json"
	case strings.HasSuffix(path, ".yaml"), strings.HasSuffix(path, ".yml"):
		return "application/yaml"
	default:
		return "" // let the API detect from the first non-whitespace byte
	}
}

func diagnosticSeverityName(s manifestv1.DiagnosticSeverity) string {
	switch s {
	case manifestv1.DiagnosticSeverity_DIAGNOSTIC_SEVERITY_INFO:
		return "info"
	case manifestv1.DiagnosticSeverity_DIAGNOSTIC_SEVERITY_WARN:
		return "warn"
	case manifestv1.DiagnosticSeverity_DIAGNOSTIC_SEVERITY_ERROR:
		return "error"
	default:
		return "unspecified"
	}
}

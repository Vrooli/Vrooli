// Package codefacts is ui-health's intake seam onto the Code Facts authority.
//
// It answers two static questions ui-health needs before it decides whether to
// run the runtime/render group against a scenario: does the scenario have a UI
// surface, and what framework is it? Code Facts owns this answer statically (no
// directory probing, no port guessing) and is the canonical replacement for the
// hand-rolled CheckUIDirectory/CheckUIPort preflight smoke used to do.
//
// Code Facts is a declared-but-optional scenario dependency, so this package
// degrades gracefully: when the service is unreachable it falls back to a thin
// filesystem probe (ui/package.json presence + framework string) and records a
// DegradedReason rather than failing. ui-health never hard-fails on Code Facts
// absence.
package codefacts

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/discovery"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	factsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts/facts_v1connect"
)

const codeFactsScenarioID = "code-facts"

// Facts is ui-health's normalized view of a scenario's UI surface.
type Facts struct {
	// HasUI is true when the scenario ships a recognizable UI surface.
	HasUI bool
	// Framework is the detected UI framework (e.g. "react-vite"), best-effort.
	Framework string
	// UIRootPath is the absolute path to the ui/ surface when one exists.
	UIRootPath string
	// Degraded is true when the answer came from the filesystem fallback rather
	// than the Code Facts authority.
	Degraded bool
	// DegradedReason explains why the fallback was used (empty when not degraded).
	DegradedReason string
}

// Describer is the seam the validation handler depends on. nil is safe at the
// call site; the handler treats a nil Describer as "degrade to filesystem".
type Describer interface {
	// Describe returns the UI facts for a scenario. It never returns an error:
	// any failure degrades to the filesystem probe with a DegradedReason set.
	Describe(ctx context.Context, scenario, scenarioDir string) Facts
}

// resolver is the slice of api-core/discovery the client needs.
type resolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}

// Client calls CodeFactsService.DescribeCodeFacts and degrades to a filesystem
// probe when Code Facts is unreachable.
type Client struct {
	// Resolver resolves the code-facts scenario URL. nil → a default resolver.
	Resolver resolver
	// HTTPClient is the Connect transport. nil → http.DefaultClient.
	HTTPClient connect.HTTPClient
}

// New constructs a Client with default discovery + HTTP transport.
func New() *Client { return &Client{} }

// Describe asks Code Facts for the scenario's surfaces and reduces them to the
// UI facts ui-health needs, degrading to a filesystem probe on any failure.
func (c *Client) Describe(ctx context.Context, scenario, scenarioDir string) Facts {
	scenario = strings.TrimSpace(scenario)
	scenarioDir = strings.TrimSpace(scenarioDir)

	res := c.Resolver
	if res == nil {
		res = discovery.NewResolver(discovery.ResolverConfig{})
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	baseURL, err := res.ResolveScenarioURLDefault(ctx, codeFactsScenarioID)
	if err != nil || strings.TrimSpace(baseURL) == "" {
		return filesystemFacts(scenarioDir, "Code Facts unavailable: "+resolveErr(err))
	}

	client := factsconnect.NewCodeFactsServiceClient(httpClient, strings.TrimRight(baseURL, "/"))
	target := &factsv1.CodeTarget{
		Kind:     factsv1.TargetKind_TARGET_KIND_SCENARIO,
		Scenario: scenario,
	}
	if scenarioDir != "" {
		target.Kind = factsv1.TargetKind_TARGET_KIND_PATH
		target.Path = scenarioDir
	}
	resp, err := client.DescribeCodeFacts(ctx, connect.NewRequest(&factsv1.DescribeCodeFactsRequest{
		Target:  target,
		Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_SURFACES, factsv1.FactFamily_FACT_FAMILY_PARSE_UNITS},
	}))
	if err != nil {
		return filesystemFacts(scenarioDir, "Code Facts unavailable: "+err.Error())
	}
	return fromReport(resp.Msg, scenarioDir)
}

func resolveErr(err error) string {
	if err == nil {
		return "no base URL"
	}
	return err.Error()
}

// fromReport reduces a CodeFactsReport to UI facts. When the report carries no
// UI surface signal at all (rather than an explicit "no UI"), it falls back to
// the filesystem so a thin/cold Code Facts cache never masks a real UI.
func fromReport(report *factsv1.CodeFactsReport, scenarioDir string) Facts {
	if report == nil {
		return filesystemFacts(scenarioDir, "Code Facts returned an empty report")
	}
	rootPath := scenarioDir
	if rp := strings.TrimSpace(report.GetTarget().GetRootPath()); rp != "" {
		rootPath = rp
	}
	for _, s := range report.GetSurfaces() {
		if s.GetKind() != factsv1.SurfaceKind_SURFACE_KIND_UI {
			continue
		}
		// A surface that Code Facts marks MISSING/UNKNOWN is not a usable UI.
		switch s.GetStatus() {
		case factsv1.SurfaceStatus_SURFACE_STATUS_MISSING, factsv1.SurfaceStatus_SURFACE_STATUS_UNKNOWN:
			continue
		}
		uiRoot := s.GetPath()
		if uiRoot != "" && !filepath.IsAbs(uiRoot) && rootPath != "" {
			uiRoot = filepath.Join(rootPath, uiRoot)
		}
		if uiRoot == "" && rootPath != "" {
			uiRoot = filepath.Join(rootPath, "ui")
		}
		return Facts{
			HasUI:      true,
			Framework:  frameworkFromRoot(uiRoot),
			UIRootPath: uiRoot,
		}
	}
	// Code Facts reported surfaces but none was a UI; trust that as an explicit
	// "no UI" only when it actually returned surfaces. An empty surface list is
	// treated as a cold answer and falls back to the filesystem.
	if len(report.GetSurfaces()) > 0 {
		return Facts{HasUI: false}
	}
	return filesystemFacts(scenarioDir, "Code Facts returned no surfaces")
}

// filesystemFacts is the degraded fallback: a scenario "has UI" if it ships a
// ui/package.json (the react-vite template guarantees this).
func filesystemFacts(scenarioDir, reason string) Facts {
	f := Facts{Degraded: true, DegradedReason: reason}
	if scenarioDir == "" {
		return f
	}
	uiRoot := filepath.Join(scenarioDir, "ui")
	if st, err := os.Stat(filepath.Join(uiRoot, "package.json")); err == nil && !st.IsDir() {
		f.HasUI = true
		f.UIRootPath = uiRoot
		f.Framework = frameworkFromRoot(uiRoot)
	}
	return f
}

// frameworkFromRoot reads ui/package.json to name the framework. Best-effort;
// an empty string means "unknown", which callers treat as a generic UI.
func frameworkFromRoot(uiRoot string) string {
	if uiRoot == "" {
		return ""
	}
	raw, err := os.ReadFile(filepath.Join(uiRoot, "package.json"))
	if err != nil {
		return ""
	}
	text := string(raw)
	switch {
	case strings.Contains(text, `"vite"`) && strings.Contains(text, `"react"`):
		return "react-vite"
	case strings.Contains(text, `"react"`):
		return "react"
	case strings.Contains(text, `"vite"`):
		return "vite"
	default:
		return "node"
	}
}

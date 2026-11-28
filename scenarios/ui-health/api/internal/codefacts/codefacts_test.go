package codefacts

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

func TestFromReportDetectsUISurface(t *testing.T) {
	report := &factsv1.CodeFactsReport{
		Target: &factsv1.TargetContext{RootPath: "/abs/scenarios/demo"},
		Surfaces: []*factsv1.Surface{
			{Kind: factsv1.SurfaceKind_SURFACE_KIND_API, Path: "api", Status: factsv1.SurfaceStatus_SURFACE_STATUS_KNOWN},
			{Kind: factsv1.SurfaceKind_SURFACE_KIND_UI, Path: "ui", Status: factsv1.SurfaceStatus_SURFACE_STATUS_KNOWN},
		},
	}
	facts := fromReport(report, "/abs/scenarios/demo")
	if !facts.HasUI {
		t.Fatal("expected HasUI=true for a known UI surface")
	}
	if facts.Degraded {
		t.Fatal("authoritative answer must not be degraded")
	}
	if facts.UIRootPath != filepath.Join("/abs/scenarios/demo", "ui") {
		t.Fatalf("UIRootPath = %q", facts.UIRootPath)
	}
}

func TestFromReportSurfacesButNoUIIsExplicitNoUI(t *testing.T) {
	report := &factsv1.CodeFactsReport{
		Surfaces: []*factsv1.Surface{
			{Kind: factsv1.SurfaceKind_SURFACE_KIND_API, Status: factsv1.SurfaceStatus_SURFACE_STATUS_KNOWN},
		},
	}
	facts := fromReport(report, "/abs/scenarios/api-only")
	if facts.HasUI {
		t.Fatal("expected HasUI=false when surfaces exist but none is UI")
	}
	if facts.Degraded {
		t.Fatal("an explicit no-UI answer is authoritative, not degraded")
	}
}

func TestFromReportMissingUISurfaceIsNotUsable(t *testing.T) {
	report := &factsv1.CodeFactsReport{
		Surfaces: []*factsv1.Surface{
			{Kind: factsv1.SurfaceKind_SURFACE_KIND_UI, Path: "ui", Status: factsv1.SurfaceStatus_SURFACE_STATUS_MISSING},
		},
	}
	facts := fromReport(report, "/abs/scenarios/demo")
	if facts.HasUI {
		t.Fatal("a MISSING UI surface must not count as a usable UI")
	}
}

func TestFromReportEmptySurfacesFallsBackToFilesystem(t *testing.T) {
	dir := t.TempDir()
	writeUIPackage(t, dir, `{"dependencies":{"react":"^18","vite":"^5"}}`)

	facts := fromReport(&factsv1.CodeFactsReport{}, dir)
	if !facts.HasUI {
		t.Fatal("empty surface list must fall back to filesystem, which finds the UI")
	}
	if !facts.Degraded {
		t.Fatal("filesystem fallback must be marked degraded")
	}
	if facts.Framework != "react-vite" {
		t.Fatalf("framework = %q, want react-vite", facts.Framework)
	}
}

func TestFilesystemFactsNoUI(t *testing.T) {
	dir := t.TempDir() // no ui/package.json
	facts := filesystemFacts(dir, "test reason")
	if facts.HasUI {
		t.Fatal("expected HasUI=false with no ui/package.json")
	}
	if !facts.Degraded || facts.DegradedReason != "test reason" {
		t.Fatalf("degraded reason not propagated: %+v", facts)
	}
}

// fakeResolver lets Describe exercise the degraded path deterministically.
type fakeResolver struct {
	url string
	err error
}

func (f fakeResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return f.url, f.err
}

func TestDescribeDegradesWhenResolverFails(t *testing.T) {
	dir := t.TempDir()
	writeUIPackage(t, dir, `{"dependencies":{"react":"^18"}}`)

	c := &Client{Resolver: fakeResolver{err: errors.New("no code-facts port")}}
	facts := c.Describe(context.Background(), "demo", dir)
	if !facts.Degraded {
		t.Fatal("resolver failure must degrade")
	}
	if !facts.HasUI || facts.Framework != "react" {
		t.Fatalf("filesystem fallback did not detect react UI: %+v", facts)
	}
}

func TestDescribeDegradesWhenServiceUnreachable(t *testing.T) {
	dir := t.TempDir()
	writeUIPackage(t, dir, `{"dependencies":{}}`)

	// A resolvable but dead URL: the Connect call fails, so Describe degrades.
	c := &Client{
		Resolver:   fakeResolver{url: "http://127.0.0.1:0"},
		HTTPClient: failingHTTPClient{},
	}
	facts := c.Describe(context.Background(), "demo", dir)
	if !facts.Degraded {
		t.Fatal("unreachable service must degrade")
	}
	if !facts.HasUI {
		t.Fatal("filesystem fallback should still find the ui/ surface")
	}
}

type failingHTTPClient struct{}

func (failingHTTPClient) Do(*http.Request) (*http.Response, error) {
	return nil, errors.New("dial refused")
}

func writeUIPackage(t *testing.T, scenarioDir, contents string) {
	t.Helper()
	uiDir := filepath.Join(scenarioDir, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Ensure failingHTTPClient satisfies connect.HTTPClient at compile time.
var _ connect.HTTPClient = failingHTTPClient{}

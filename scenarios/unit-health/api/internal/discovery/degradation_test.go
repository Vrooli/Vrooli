package discovery

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeLocator struct{ root string }

func (f fakeLocator) Locate(context.Context, string, string) (string, string, string, error) {
	return "demo", "scenario", f.root, nil
}

type fakeResolver struct {
	url string
	err error
}

func (r fakeResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return r.url, r.err
}

type errHTTPClient struct{}

func (errHTTPClient) Do(*http.Request) (*http.Response, error) {
	return nil, errors.New("connection refused")
}

func writeGoModFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, "api", "go.mod"), "module demo\n")
	return root
}

func TestDiscoverDegradesWhenResolverFails(t *testing.T) {
	root := writeGoModFixture(t)
	c := CodeFactsClient{
		Locator:  fakeLocator{root: root},
		Resolver: fakeResolver{err: errors.New("no api url")},
	}
	inv, err := c.Discover(context.Background(), "demo", "scenario", "", false)
	if err != nil {
		t.Fatalf("Discover should degrade, not error: %v", err)
	}
	if !strings.Contains(inv.DegradedReason, "Code Facts unavailable") {
		t.Errorf("expected a Code-Facts-unavailable degrade reason, got %q", inv.DegradedReason)
	}
}

func TestDiscoverDegradesWhenRPCFails(t *testing.T) {
	root := writeGoModFixture(t)
	c := CodeFactsClient{
		Locator:    fakeLocator{root: root},
		Resolver:   fakeResolver{url: "http://127.0.0.1:0"},
		HTTPClient: errHTTPClient{},
	}
	inv, err := c.Discover(context.Background(), "demo", "scenario", "", false)
	if err != nil {
		t.Fatalf("Discover should degrade, not error: %v", err)
	}
	if !strings.Contains(inv.DegradedReason, "Code Facts unavailable") {
		t.Errorf("expected a Code-Facts-unavailable degrade reason, got %q", inv.DegradedReason)
	}
}

package reconcile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"flow-verifier/internal/flows/kind"
	"flow-verifier/internal/flows/kinds/navigation"
	"flow-verifier/internal/flows/schemas"
)

func loadFullGraph(t *testing.T) interface{} {
	t.Helper()
	k, _ := kind.Get(navigation.Name)
	spec, err := k.Load(schemas.NavigationFullExample, "schemas/examples/navigation-full.json")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return spec
}

func writeSrc(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}

func TestReconcileHappy(t *testing.T) {
	scenarioRoot := t.TempDir()
	// One file declaring every spec route as <Route path=...> and one
	// file using <Link to=...>. The full-example spec has 11 routes.
	var routes strings.Builder
	routes.WriteString("import { Route } from 'react-router';\n")
	for _, p := range []string{
		"/", "/login", "/forgot-password", "/tasks", "/tasks/:id",
		"/settings", "/settings/display", "/settings/notifications",
		"/settings/about", "/admin/users", "/beta",
	} {
		routes.WriteString("<Route path=\"" + p + "\" />\n")
	}
	writeSrc(t, scenarioRoot, "ui/src/routes.tsx", routes.String())
	writeSrc(t, scenarioRoot, "ui/src/nav.tsx", `
import { Link, useNavigate } from 'react-router';
function Nav() {
  const navigate = useNavigate();
  return <>
    <Link to="/">Home</Link>
    <Link to="/tasks">Tasks</Link>
    <Link to="/settings">Settings</Link>
    <button onClick={() => navigate("/login")}>Login</button>
    <Link to="/tasks/42">First task</Link>
  </>;
}
`)
	spec := loadFullGraph(t).(*navigation.Spec)
	res, err := Run(spec.Graph(), scenarioRoot)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, f := range res.Findings {
		if f.Severity == "error" {
			t.Errorf("unexpected error finding: %+v", f)
		}
	}
	if !res.Passed {
		t.Fatalf("expected reconcile pass, got %d findings", len(res.Findings))
	}
	if res.FilesScanned < 2 {
		t.Errorf("expected ≥2 files scanned, got %d", res.FilesScanned)
	}
}

func TestReconcileDetectsRouteInCodeNotInSpec(t *testing.T) {
	scenarioRoot := t.TempDir()
	writeSrc(t, scenarioRoot, "ui/src/extra.tsx", `<Route path="/wat-no-spec" />`)
	spec := loadFullGraph(t).(*navigation.Spec)
	res, err := Run(spec.Graph(), scenarioRoot)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Passed {
		t.Fatal("expected reconcile fail")
	}
	found := false
	for _, f := range res.Findings {
		if strings.HasPrefix(f.ID, "route_in_code_not_in_spec:") && f.SourceFile != "" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected route_in_code_not_in_spec finding with source_file")
	}
}

func TestReconcileDetectsNavTargetNotInSpec(t *testing.T) {
	scenarioRoot := t.TempDir()
	writeSrc(t, scenarioRoot, "ui/src/nav.tsx", `<Link to="/totally-fake" />`)
	spec := loadFullGraph(t).(*navigation.Spec)
	res, err := Run(spec.Graph(), scenarioRoot)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Passed {
		t.Fatal("expected reconcile fail")
	}
	found := false
	for _, f := range res.Findings {
		if strings.HasPrefix(f.ID, "nav_target_not_in_spec:/totally-fake") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected nav_target_not_in_spec finding")
	}
}

func TestReconcileMissingUiSrc(t *testing.T) {
	scenarioRoot := t.TempDir() // no ui/src
	spec := loadFullGraph(t).(*navigation.Spec)
	res, err := Run(spec.Graph(), scenarioRoot)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Passed {
		t.Fatal("missing ui/src should not pass")
	}
}

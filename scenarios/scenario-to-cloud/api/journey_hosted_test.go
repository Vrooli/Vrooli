package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"scenario-to-cloud/bundle"
	"scenario-to-cloud/closure"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/manifest"
	"scenario-to-cloud/vps"
)

type hostedBundleFixture struct {
	root     string
	closure  domain.Closure
	manifest domain.CloudManifest
	artifact domain.BundleArtifact
	files    map[string][]byte
}

func hostedBundle(t *testing.T) hostedBundleFixture {
	t.Helper()
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := closure.NewRepoCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	// Explicit hosted selection: the gateway code is required, its local and
	// router resource wrappers are not. The gateway resolves hosted suppliers
	// through its private execution route instead of starting local wrappers.
	deselected := []string{"ollama", "ollaya", "openrouter", "orcarouter", "abliteration"}
	c, err := closure.Resolve(context.Background(), closure.Inputs{
		ScenarioID: "landing-page-business-suite", Environment: "production", Platform: closure.Platform{OS: "linux", Arch: "amd64"},
		Scope: closure.ScopeBundle, RepoRoot: root, Catalog: catalog, HostRequirements: closure.ControlPlaneHostRequirements{},
		Overrides: closure.Overrides{DeselectOptional: deselected},
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "scenarios/landing-page-business-suite/cloud-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var declared domain.CloudManifest
	if err := json.Unmarshal(data, &declared); err != nil {
		t.Fatal(err)
	}
	// The fixture uses the committed release shape. Only the external target
	// address is replaced; no network operation is issued by these tests.
	declared.Target.VPS.Host = "203.0.113.13"
	declared.Edge.Domain = "hosted.example.invalid"
	declared.Edge.MailDNS = nil
	m, issues := manifest.ValidateAndNormalize(declared)
	if manifest.HasBlockingIssues(issues) {
		t.Fatalf("hosted manifest validation: %+v", issues)
	}
	deps, _ := closure.ManifestDependencies(c)
	for _, id := range deps.Scenarios {
		if !containsHostedID(m.Dependencies.Scenarios, id) || !containsHostedID(m.Bundle.Scenarios, id) {
			t.Fatalf("committed hosted manifest omits resolved scenario %s", id)
		}
	}
	for _, id := range deselected {
		if !containsHostedID(m.Dependencies.DeselectOptional, id) {
			t.Fatalf("committed hosted manifest does not exclude %s", id)
		}
	}
	// A bounded tarball fixture copies real source and manifests rather than
	// fake scenario stubs. Resolution above always reads the real repository.
	fixtureRoot := t.TempDir()
	for _, dir := range []string{".vrooli", "templates", "scenarios", "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(fixtureRoot, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	paths := []string{".vrooli/repo-contract.json", ".vrooli/service.json", "go.mod", "scenarios/landing-page-business-suite/.vrooli/service.json", "scenarios/landing-page-business-suite/api/main.go", "scenarios/landing-page-business-suite/api/go.mod", "scenarios/landing-page-business-suite/ui/package.json", "scenarios/ai-gateway/.vrooli/service.json", "scenarios/ai-gateway/api/main.go", "scenarios/ai-gateway/api/go.mod", "resources/postgres/resource.json", "scenarios/vrooli-autoheal/.vrooli/service.json", "scenarios/vrooli-autoheal/api/main.go", "scenarios/vrooli-autoheal/api/go.mod"}
	for _, id := range deselected {
		paths = append(paths, "resources/"+id+"/resource.json")
	}
	for _, name := range paths {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(fixtureRoot, name)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	artifact, err := bundle.BuildMiniVrooliBundle(fixtureRoot, t.TempDir(), m)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(artifact.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	files := map[string][]byte{}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		files[strings.TrimPrefix(header.Name, "./")] = data
	}
	return hostedBundleFixture{root: root, closure: c, manifest: m, artifact: artifact, files: files}
}

func TestJ13_TarballContainsGatewayCode(t *testing.T) {
	f := hostedBundle(t)
	required := false
	for _, component := range f.closure.Components {
		if component.Kind == domain.ClosureKindScenario && component.ID == "ai-gateway" {
			required = component.Required
		}
	}
	if !required {
		t.Fatal("LPBS must require gateway in the resolved production closure")
	}
	const source = "scenarios/ai-gateway/api/main.go"
	want, err := os.ReadFile(filepath.Join(f.root, source))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(f.files[source], want) {
		t.Fatal("tarball must contain the real gateway source")
	}
	if len(f.files["scenarios/ai-gateway/.vrooli/service.json"]) == 0 {
		t.Fatal("gateway lifecycle declaration missing")
	}
}

func TestJ13_OllamaAndRouterResourcesNotPulled(t *testing.T) {
	f := hostedBundle(t)
	for _, id := range []string{"ollama", "ollaya", "openrouter", "orcarouter", "abliteration"} {
		for _, component := range f.closure.Components {
			if component.Kind == domain.ClosureKindResource && component.ID == id {
				t.Errorf("excluded resource %s in closure", id)
			}
		}
		for name := range f.files {
			if strings.HasPrefix(name, "resources/"+id+"/") {
				t.Errorf("excluded resource source in tarball: %s", name)
			}
		}
	}
	if len(f.files["resources/postgres/resource.json"]) == 0 {
		t.Fatal("required database resource must still ship")
	}
}

func TestJ13_GatewayHasNoPublicListener(t *testing.T) {
	f := hostedBundle(t)
	gatewayListeners := 0
	for _, listener := range f.closure.Listeners {
		if listener.Owner == "scenario:ai-gateway" {
			gatewayListeners++
			if listener.Visibility != "private" {
				t.Errorf("gateway listener exposed: %+v", listener)
			}
		}
		if listener.Visibility == "public_via_edge" && listener.Owner != "scenario:landing-page-business-suite" {
			t.Errorf("second public surface: %+v", listener)
		}
	}
	if gatewayListeners == 0 {
		t.Fatal("gateway private listener missing")
	}
	plan, err := compileHostedFixture(t, f, f.manifest)
	if err != nil {
		t.Fatal(err)
	}
	edgeFound := false
	for _, action := range plan.Actions {
		if action.OwnerOperation == execplan.OpEdgeRouteApply {
			edgeFound = true
			spec, err := vps.EdgeSpecFor(vps.CommandContext{DeploymentID: plan.DeploymentID, ScenarioID: f.manifest.Scenario.ID, Manifest: f.manifest}, action.Inputs)
			if err != nil {
				t.Fatal(err)
			}
			for _, route := range spec.Routes {
				if !strings.HasPrefix(route.ListenerID, "landing-page-business-suite/") || route.Host != f.manifest.Edge.Domain {
					t.Errorf("public route escapes the LPBS door: %+v", route)
				}
			}
		}
	}
	if !edgeFound {
		t.Fatal("compiled public edge missing")
	}
}

func TestJ13_ValidatorAndPlanCompilerPass(t *testing.T) {
	f := hostedBundle(t)
	var packed domain.CloudManifest
	if err := json.Unmarshal(f.files[".vrooli/cloud/manifest.json"], &packed); err != nil {
		t.Fatal(err)
	}
	normalized, issues := manifest.ValidateAndNormalize(packed)
	if manifest.HasBlockingIssues(issues) {
		t.Fatalf("packed manifest invalid: %+v", issues)
	}
	plan, err := compileHostedFixture(t, f, normalized)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) == 0 || plan.MustDigest() == "" {
		t.Fatal("compiler returned no executable plan")
	}
	dependencyIndex, primaryIndex := -1, -1
	for i, action := range plan.Actions {
		if action.OwnerOperation == execplan.OpRuntimeStartDeps {
			ids := strings.Split(action.Inputs["scenarios"], ",")
			if !containsHostedID(ids, "ai-gateway") {
				t.Fatal("required gateway missing from dependency activation")
			}
			if containsHostedID(ids, "landing-page-business-suite") {
				t.Fatal("primary must be activated by release owner, not started twice")
			}
			dependencyIndex = i
		}
		if action.OwnerOperation == execplan.OpReleaseActivate {
			if !containsHostedID(strings.Split(action.Inputs["scenarios"], ","), "landing-page-business-suite") {
				t.Fatal("release activation omits primary")
			}
			if !containsHostedID(action.DependsOn, execplan.OpRuntimeStartDeps) {
				t.Fatal("primary activation does not depend on completed dependency startup")
			}
			primaryIndex = i
		}
	}
	if dependencyIndex < 0 || primaryIndex <= dependencyIndex {
		t.Fatal("gateway must start before primary release activation")
	}

}

func containsHostedID(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

// Explicitly simulated credential-presence observations allow offline plan
// compilation. They contain no credentials and authorize no execution.
func compileHostedFixture(t *testing.T, f hostedBundleFixture, m domain.CloudManifest) (*execplan.Plan, error) {
	t.Helper()
	var satisfied []string
	for _, component := range f.closure.Components {
		if component.Kind == domain.ClosureKindCredentialDescriptor {
			satisfied = append(satisfied, component.ID)
		}
	}
	if m.Secrets != nil {
		for _, secret := range m.Secrets.BundleSecrets {
			satisfied = append(satisfied, secret.ID)
		}
	}
	return vps.CompilePlan(context.Background(), vps.PlanRequest{Manifest: m, Closure: &f.closure, BundlePath: f.artifact.Path, Observations: execplan.Observations{SatisfiedInputs: satisfied}})
}

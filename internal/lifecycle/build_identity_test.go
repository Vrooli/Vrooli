package lifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func TestScenarioBuildIdentityTracksAuthoredInputsAndIgnoresRuntimeOutputs(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "service.json"), []byte(`{"service":{"name":"alpha"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "dist"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "PRD.md"), []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "dist", "bundle.js"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	item := scenario.Scenario{
		Path: root,
		Slug: "alpha",
		Manifest: scenario.ServiceManifest{Components: map[string]scenario.Component{
			"api": {
				Build: scenario.ComponentBuild{Kind: "go_module", Dir: ".", Output: "api/mock-api"},
			},
		}},
	}
	if err := os.MkdirAll(filepath.Join(root, "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "api", "mock-api"), []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	first, err := scenarioBuildIdentity(item)
	if err != nil {
		t.Fatalf("scenarioBuildIdentity() error = %v", err)
	}
	if first == "" {
		t.Fatal("scenarioBuildIdentity() returned empty identity")
	}
	runtimeReceipt := filepath.Join(root, ".vrooli", "runtime", "rehabilitation-evidence", "profile-receipt.json")
	if err := os.MkdirAll(filepath.Dir(runtimeReceipt), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runtimeReceipt, []byte("generated owner receipt"), 0o600); err != nil {
		t.Fatal(err)
	}
	runtimeReceiptIdentity, err := scenarioBuildIdentity(item)
	if err != nil {
		t.Fatalf("scenarioBuildIdentity() after runtime receipt: %v", err)
	}
	if runtimeReceiptIdentity != first {
		t.Fatalf("ignored runtime receipt changed identity: first=%q afterReceipt=%q", first, runtimeReceiptIdentity)
	}
	if err := os.WriteFile(filepath.Join(root, "dist", "bundle.js"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := scenarioBuildIdentity(item)
	if err != nil {
		t.Fatalf("scenarioBuildIdentity() after output change: %v", err)
	}
	if second != first {
		t.Fatalf("runtime output changed identity: first=%q second=%q", first, second)
	}
	if err := os.WriteFile(filepath.Join(root, "api", "mock-api"), []byte("new binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	third, err := scenarioBuildIdentity(item)
	if err != nil {
		t.Fatalf("scenarioBuildIdentity() after component output change: %v", err)
	}
	if third != second {
		t.Fatalf("declared component output changed identity: first=%q third=%q", second, third)
	}
	if err := os.WriteFile(filepath.Join(root, "PRD.md"), []byte("beta"), 0o644); err != nil {
		t.Fatal(err)
	}
	fourth, err := scenarioBuildIdentity(item)
	if err != nil {
		t.Fatalf("scenarioBuildIdentity() after source change: %v", err)
	}
	if fourth == third {
		t.Fatal("authored source change did not change build identity")
	}
}

func TestRegistryRuntimeHealthRejectsStaleBuildIdentity(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	item := scenario.Scenario{Slug: "alpha", Path: root}
	runner := &Runner{}

	if runner.isRegistryRuntimeHealthy(item, registryRuntimeView{
		Authoritative: true,
		Instance:      scenarioruntime.Instance{BuildIdentity: "sha256:older"},
	}) {
		t.Fatal("stale build identity must prevent healthy runtime reuse")
	}
}

func TestScenarioBuildIdentityUsesDeclaredRuntimeDocsAndExcludedPaths(t *testing.T) {
	root := t.TempDir()
	operationalDocs := filepath.Join(root, "docs", "internal")
	if err := os.MkdirAll(operationalDocs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"REFRACTOR_PROGRESS.md": "checkpoint one",
		"OPERATOR_FEEDBACK.md":  "feedback one",
	} {
		if err := os.WriteFile(filepath.Join(operationalDocs, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	problemsPath := filepath.Join(root, "docs", "PROBLEMS.md")
	if err := os.WriteFile(problemsPath, []byte("issue ledger one"), 0o644); err != nil {
		t.Fatal(err)
	}
	runtimeDocPath := filepath.Join(root, "docs", "guide.md")
	if err := os.WriteFile(runtimeDocPath, []byte("runtime-served guide one"), 0o644); err != nil {
		t.Fatal(err)
	}
	testHelperPath := filepath.Join(root, "api", "internal", "testutil", "testutil.go")
	if err := os.MkdirAll(filepath.Dir(testHelperPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testHelperPath, []byte("test helper one"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatal(err)
	}
	manifestBody := `{"build_identity":{"runtime_docs":["docs/guide.md"],"exclude_paths":["api/internal/testutil/testutil.go"]}}`
	if err := os.WriteFile(manifestPath, []byte(manifestBody), 0o644); err != nil {
		t.Fatal(err)
	}
	item := scenario.Scenario{Path: root, Slug: "browser-automation-studio", Manifest: scenario.ServiceManifest{
		BuildIdentity: scenario.BuildIdentityPolicy{
			RuntimeDocs:  []string{"docs/guide.md"},
			ExcludePaths: []string{"api/internal/testutil/testutil.go"},
		},
	}}

	first, err := scenarioBuildIdentity(item)
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"REFRACTOR_PROGRESS.md": "checkpoint two",
		"OPERATOR_FEEDBACK.md":  "feedback two",
	} {
		if err := os.WriteFile(filepath.Join(operationalDocs, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(problemsPath, []byte("issue ledger two"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testHelperPath, []byte("test helper two"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := scenarioBuildIdentity(item)
	if err != nil {
		t.Fatal(err)
	}
	if second != first {
		t.Fatalf("operational documentation changed candidate identity: first=%q second=%q", first, second)
	}
	if err := os.WriteFile(runtimeDocPath, []byte("runtime-served guide two"), 0o644); err != nil {
		t.Fatal(err)
	}
	third, err := scenarioBuildIdentity(item)
	if err != nil {
		t.Fatal(err)
	}
	if third == second {
		t.Fatal("runtime-served documentation did not change candidate identity")
	}
	if err := os.WriteFile(manifestPath, []byte(strings.Replace(manifestBody, "docs/guide.md", "docs/other.md", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	fourth, err := scenarioBuildIdentity(item)
	if err != nil {
		t.Fatal(err)
	}
	if fourth == third {
		t.Fatal("service manifest policy change did not change candidate identity")
	}
}

func TestBuildIdentityRuntimeDocsRequireExistingDeclaredPaths(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "service.json"), []byte(`{"service":{"name":"alpha"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	item := scenario.Scenario{Path: root, Slug: "alpha", Manifest: scenario.ServiceManifest{
		BuildIdentity: scenario.BuildIdentityPolicy{RuntimeDocs: []string{"docs/missing.md"}},
	}}
	if _, err := scenarioBuildIdentity(item); err == nil || !strings.Contains(err.Error(), "declared runtime documentation") {
		t.Fatalf("scenarioBuildIdentity() error = %v, want missing declared document error", err)
	}
}

func TestBuildIdentityExclusionsRejectUnsafePaths(t *testing.T) {
	for _, candidate := range []string{"", ".", "../outside", "/absolute", "docs/**", ".vrooli/service.json", "docs\\..\\outside"} {
		t.Run(candidate, func(t *testing.T) {
			if _, err := buildIdentityExcludedPaths([]string{candidate}); err == nil {
				t.Fatalf("buildIdentityExcludedPaths(%q) succeeded, want error", candidate)
			}
		})
	}
}

func TestBuildIdentityRuntimeDocsRejectUnsafeOrConflictingPaths(t *testing.T) {
	excluded, err := buildIdentityExcludedPaths([]string{"docs/private"})
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range []string{"", "docs", "docs/../outside.md", "docs/**", "/docs/guide.md", "docs/private/guide.md"} {
		t.Run(candidate, func(t *testing.T) {
			if _, err := buildIdentityRuntimeDocs([]string{candidate}, excluded); err == nil {
				t.Fatalf("buildIdentityRuntimeDocs(%q) succeeded, want error", candidate)
			}
		})
	}
	if _, err := buildIdentityRuntimeDocs([]string{"docs/guides", "docs/guides/setup.md"}, nil); err == nil {
		t.Fatal("overlapping runtime documentation paths were accepted")
	}
}

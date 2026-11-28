package dependencyhealth

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"connectrpc.com/connect"
	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

func TestGoSurfaceGoModsIncludesScenarioRootModuleAndSkipsRetainedModules(t *testing.T) {
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "scenarios", "demo")
	writeFile(t, filepath.Join(scenarioDir, "go.mod"), "module example.com/demo\n")
	writeFile(t, filepath.Join(scenarioDir, "cli", "go.mod"), "module example.com/demo-cli\n")
	writeFile(t, filepath.Join(scenarioDir, "ui", "package.json"), "{}\n")
	for _, retained := range []string{"docs", "docs/internal/evidence/E1/bytes/api", "api/internal/testdata/fixture", "bas/fixtures/go"} {
		writeFile(t, filepath.Join(scenarioDir, retained, "go.mod"), "module example.com/retained\n")
	}

	got := goSurfaceGoMods(scenarioDir)
	want := []string{
		filepath.Join(scenarioDir, "cli", "go.mod"),
		filepath.Join(scenarioDir, "go.mod"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("goSurfaceGoMods = %#v, want %#v", got, want)
	}
}

// fixRepo is the incident shape in a repository fixture: packages/leaf imports
// golang.org/x/mod, and scenarios/demo/api reaches it through a local replace
// without the requirement or checksums.
func fixRepo(t *testing.T, registry string, consumerFiles map[string]string) (handler *connectHandler, consumerDir string) {
	t.Helper()
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOFLAGS", "-mod=mod")
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Version}}", "golang.org/x/mod").Output()
	if err != nil {
		t.Skipf("resolve golang.org/x/mod from the SDA module: %v", err)
	}
	version := strings.TrimSpace(string(out))

	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteScenarioStub(t, "demo")
	root := fixture.Root
	writeFile(t, filepath.Join(root, ".vrooli", "dependencies", "approved-dependencies.json"), registry)
	writeFile(t, filepath.Join(root, "packages", "leaf", "go.mod"), "module example.com/leaf\n\ngo 1.25.0\n\nrequire golang.org/x/mod "+version+"\n")
	writeFile(t, filepath.Join(root, "packages", "leaf", "leaf.go"), "package leaf\n\nimport \"golang.org/x/mod/module\"\n\ntype V = module.Version\n")
	consumerDir = filepath.Join(root, "scenarios", "demo", "api")
	writeFile(t, filepath.Join(consumerDir, "go.mod"), "module demo/api\n\ngo 1.25.0\n\nrequire example.com/leaf v0.0.0\n\nreplace example.com/leaf => ../../../packages/leaf\n")
	writeFile(t, filepath.Join(consumerDir, "main.go"), "package main\n\nimport \"example.com/leaf\"\n\nfunc main() { _ = leaf.V{} }\n")
	for name, body := range consumerFiles {
		writeFile(t, filepath.Join(consumerDir, name), body)
	}
	scenariosDir := filepath.Join(root, "scenarios")
	return &connectHandler{scenariosDir: func() string { return scenariosDir }}, consumerDir
}

func fix(t *testing.T, h *connectHandler, apply bool) *scenariovalidationv1.FixResponse {
	t.Helper()
	req := connect.NewRequest(&scenariovalidationv1.FixRequest{Scenario: "demo", RuleIds: []string{goModTidyRuleID}})
	var resp *connect.Response[scenariovalidationv1.FixResponse]
	var err error
	if apply {
		resp, err = h.ApplyFix(context.Background(), req)
	} else {
		resp, err = h.PreviewFix(context.Background(), req)
	}
	if err != nil {
		t.Fatal(err)
	}
	return resp.Msg
}

func TestFixConvergesSharedModuleRequirementThroughGovernanceGate(t *testing.T) {
	h, consumerDir := fixRepo(t, `{"records": []}`, nil)
	goModBefore, _ := os.ReadFile(filepath.Join(consumerDir, "go.mod"))

	preview := fix(t, h, false)
	if preview.GetApplied() || len(preview.GetCandidates()) != 2 {
		t.Fatalf("preview = %+v, want unapplied go.mod and go.sum candidates", preview)
	}
	if now, _ := os.ReadFile(filepath.Join(consumerDir, "go.mod")); string(now) != string(goModBefore) {
		t.Fatal("PreviewFix must not write")
	}
	if !strings.Contains(preview.GetCandidates()[0].GetDescription(), "add golang.org/x/mod") {
		t.Fatalf("go.mod candidate must name the synchronized requirement: %q", preview.GetCandidates()[0].GetDescription())
	}

	applied := fix(t, h, true)
	if !applied.GetApplied() || len(applied.GetCandidates()) != 2 {
		t.Fatalf("apply = %+v", applied)
	}
	for i, candidate := range applied.GetCandidates() {
		if !candidate.GetApplied() || candidate.GetAfter() != preview.GetCandidates()[i].GetAfter() {
			t.Fatalf("applied candidate %d must match its preview and be marked applied", i)
		}
	}
	tidy := exec.Command("go", "mod", "tidy", "-diff")
	tidy.Dir = consumerDir
	if out, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("surface not tidy after ApplyFix: %v\n%s", err, out)
	}

	again := fix(t, h, false)
	if len(again.GetCandidates()) != 0 || again.GetApplied() {
		t.Fatalf("converged surface must have no candidates: %+v", again)
	}
}

func TestFixRefusesUnapprovedDirectRequirementAndWritesNothing(t *testing.T) {
	h, consumerDir := fixRepo(t, `{"records": []}`, map[string]string{
		"direct.go": "package main\n\nimport \"golang.org/x/mod/semver\"\n\nvar _ = semver.IsValid\n",
	})
	goModBefore, _ := os.ReadFile(filepath.Join(consumerDir, "go.mod"))

	resp := fix(t, h, true)
	if resp.GetApplied() || len(resp.GetCandidates()) != 0 {
		t.Fatalf("a refused surface must not be applied: %+v", resp)
	}
	refused := ""
	for _, message := range resp.GetMessages() {
		if strings.HasPrefix(message, fixMessageRefused) {
			refused = message
		}
	}
	if !strings.Contains(refused, "deps approved approve-observed go/golang.org/x/mod") {
		t.Fatalf("refusal must name the approval command, messages=%q", resp.GetMessages())
	}
	if now, _ := os.ReadFile(filepath.Join(consumerDir, "go.mod")); string(now) != string(goModBefore) {
		t.Fatal("a refused surface must be left untouched")
	}
	if _, err := os.Stat(filepath.Join(consumerDir, "go.sum")); !os.IsNotExist(err) {
		t.Fatalf("a refused surface must not gain a go.sum: %v", err)
	}
}

func TestFixWithBareModulePathConvergesOnlyThatModule(t *testing.T) {
	h, consumerDir := fixRepo(t, `{"records": []}`, nil)
	nested := filepath.Join(consumerDir, "tools", "gen")
	writeFile(t, filepath.Join(nested, "go.mod"), "module demo/gen\n\ngo 1.25.0\n\nrequire example.com/leaf v0.0.0\n")

	resp, err := h.PreviewFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{Path: consumerDir}))
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range resp.Msg.GetCandidates() {
		if strings.Contains(candidate.GetFilePath(), "tools/gen") {
			t.Fatalf("a bare module path must not converge nested modules: %s", candidate.GetFilePath())
		}
	}
	if len(resp.Msg.GetCandidates()) == 0 {
		t.Fatal("the addressed module itself must converge")
	}
}

func TestModuleOwnerNamesGovernanceScope(t *testing.T) {
	root := "/repo"
	for _, tc := range []struct{ dir, owner, surface string }{
		{"/repo", "vrooli", "control-plane"},
		{"/repo/scenarios/demo", "demo", ""},
		{"/repo/scenarios/demo/api", "demo", "api"},
		{"/repo/scenarios/demo/cli", "demo", "cli"},
		{"/repo/resources", "resources", "resource"},
		{"/repo/resources/doc-parse/cli", "doc-parse", "resource"},
		{"/repo/packages/cli-core", "cli-core", "packages"},
		{"/tmp/workspace/scenarios/new-app/api", "new-app", "api"},
	} {
		owner, surface := moduleOwner(root, tc.dir)
		if owner != tc.owner || surface != tc.surface {
			t.Errorf("moduleOwner(%s) = %q,%q want %q,%q", tc.dir, owner, surface, tc.owner, tc.surface)
		}
	}
}

func TestGoModHealthFlagsUnfencedDocsEvidenceUntilFenced(t *testing.T) {
	h, consumerDir := fixRepo(t, `{"records": []}`, nil)
	scenarioDir := filepath.Dir(consumerDir)
	writeFile(t, filepath.Join(scenarioDir, "docs", "internal", "evidence", "E1", "copy", "copy.go"), "package copy\n")
	writeFile(t, filepath.Join(scenarioDir, "docs", "README.md"), "# docs\n")

	_, findings, _ := h.evaluateGoModReplace(context.Background(), "demo", nil)
	var evidence []string
	for _, finding := range findings {
		if finding.GetRuleId() == goModEvidenceRuleID {
			evidence = append(evidence, finding.GetObserved())
			if finding.GetSeverity() != "ERROR" || !strings.Contains(finding.GetRemediation(), "scenarios/demo/docs-evidence") {
				t.Fatalf("evidence finding must gate and cite the docs-evidence convention: %+v", finding)
			}
		}
	}
	if len(evidence) != 1 || !strings.Contains(evidence[0], "docs/internal/evidence/E1/copy") {
		t.Fatalf("evidence findings = %q", evidence)
	}

	writeFile(t, filepath.Join(scenarioDir, "docs", "go.mod"), "module github.com/vrooli/vrooli/scenarios/demo/docs-evidence\n\ngo 1.26.0\n")
	_, findings, _ = h.evaluateGoModReplace(context.Background(), "demo", nil)
	for _, finding := range findings {
		if finding.GetRuleId() == goModEvidenceRuleID {
			t.Fatalf("a fenced evidence tree must not be flagged: %+v", finding)
		}
	}
}

func TestFixRefusesModuleWithUnfencedDocsEvidence(t *testing.T) {
	h, consumerDir := fixRepo(t, `{"records": []}`, nil)
	writeFile(t, filepath.Join(consumerDir, "docs", "evidence", "copy.go"), "package copy\n\nimport _ \"golang.org/x/mod/semver\"\n")
	writeFile(t, filepath.Join(consumerDir, "docs", "README.md"), "# docs\n")
	goModBefore, _ := os.ReadFile(filepath.Join(consumerDir, "go.mod"))

	resp := fix(t, h, true)
	if resp.GetApplied() || len(resp.GetCandidates()) != 0 {
		t.Fatalf("a module with unfenced evidence must not converge: %+v", resp)
	}
	refused := strings.Join(resp.GetMessages(), "\n")
	if !strings.Contains(refused, fixMessageRefused) || !strings.Contains(refused, "docs/evidence") || !strings.Contains(refused, "docs-evidence convention") {
		t.Fatalf("messages = %q", resp.GetMessages())
	}
	if now, _ := os.ReadFile(filepath.Join(consumerDir, "go.mod")); string(now) != string(goModBefore) {
		t.Fatal("refused module must be untouched")
	}
}

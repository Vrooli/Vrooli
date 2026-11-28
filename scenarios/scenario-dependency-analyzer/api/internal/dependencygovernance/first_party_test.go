package dependencygovernance

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
)

// firstPartyRepo lays out a repository with one workspace package and a
// scenario UI that links to it, and an empty registry (plus any records).
func firstPartyRepo(t *testing.T, records string) (repoRoot, uiDir string) {
	t.Helper()
	repoRoot = t.TempDir()
	writeRegistry(t, repoRoot, `{"records":[`+records+`]}`)
	uiDir = filepath.Join(repoRoot, "scenarios", "demo", "ui")
	for path, text := range map[string]string{
		"packages/widgets/package.json":       `{"name":"@vrooli/widgets","version":"1.2.3"}`,
		"scenarios/demo/ui/pnpm-lock.yaml":    "",
		"scenarios/demo/ui/package.json":      `{"name":"demo-ui","dependencies":{"@vrooli/widgets":"file:../../../packages/widgets","@vrooli/ghost":"file:../../../packages/ghost"}}`,
		"scenarios/demo/other/package.json":   `{"name":"other"}`,
		"scenarios/demo/other/pnpm-lock.yaml": "",
	} {
		full := filepath.Join(repoRoot, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return repoRoot, uiDir
}

func firstPartyRequest(pkg, version string, apply bool) *connect.Request[governancev1.InstallDependencyRequest] {
	return connect.NewRequest(&governancev1.InstallDependencyRequest{
		Scenario: "demo", Surface: "ui", Ecosystem: "npm", PackageName: pkg, Version: version, Apply: apply,
	})
}

func TestFirstPartyPackageIsAdmittedWithoutARecordAndTheApplyIsAudited(t *testing.T) {
	repoRoot, _ := firstPartyRepo(t, "")
	installer := &recordingInstaller{}
	h := withTestDiskAdmission(t, &connectHandler{scenariosDir: func() string { return filepath.Join(repoRoot, "scenarios") }, installer: installer})

	preview, err := h.InstallDependency(context.Background(), firstPartyRequest("@vrooli/widgets", "file:../../../packages/widgets", false))
	if err != nil {
		t.Fatal(err)
	}
	if preview.Msg.GetBlocked() || preview.Msg.GetVerdict() != "first_party" || installer.called {
		t.Fatalf("preview: blocked=%t verdict=%q installerCalled=%t message=%s", preview.Msg.GetBlocked(), preview.Msg.GetVerdict(), installer.called, preview.Msg.GetMessage())
	}
	if !strings.Contains(strings.Join(preview.Msg.GetSecurityNotes(), "\n"), "first-party packages/widgets admitted without a governance record") {
		t.Fatalf("security notes = %v", preview.Msg.GetSecurityNotes())
	}
	if _, err := os.Stat(filepath.Join(repoRoot, firstPartyAuditFile)); !os.IsNotExist(err) {
		t.Fatalf("a preview must not write the audit trail: %v", err)
	}

	applied, err := h.InstallDependency(context.Background(), firstPartyRequest("@vrooli/widgets", "file:../../../packages/widgets", true))
	if err != nil {
		t.Fatal(err)
	}
	if applied.Msg.GetVerdict() != preview.Msg.GetVerdict() || !applied.Msg.GetInstalled() || !installer.called {
		t.Fatalf("apply must return the preview's verdict and install: %+v", applied.Msg)
	}
	audit, err := os.ReadFile(filepath.Join(repoRoot, firstPartyAuditFile))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"scenario":"demo"`, `"surface":"ui"`, `"package":"@vrooli/widgets"`, `"path":"packages/widgets"`} {
		if !strings.Contains(string(audit), want) {
			t.Errorf("audit line lacks %s: %s", want, audit)
		}
	}
	if strings.Count(string(audit), "\n") != 1 {
		t.Errorf("one apply is one audit line, got %q", audit)
	}
}

func TestFirstPartyAdmissionNeverWidensBeyondWhatTheLayoutProves(t *testing.T) {
	repoRoot, _ := firstPartyRepo(t, "")
	for _, test := range []struct{ name, pkg, version string }{
		{"a first-party name from the registry", "@vrooli/widgets", "1.2.3"},
		{"a package that is not in the repository", "@vrooli/ghost", "file:../../../packages/ghost"},
		{"a link under another name", "@vrooli/other", "file:../../../packages/widgets"},
		{"a link outside packages", "demo-other", "file:../other"},
	} {
		t.Run(test.name, func(t *testing.T) {
			installer := &recordingInstaller{}
			h := withTestDiskAdmission(t, &connectHandler{scenariosDir: func() string { return filepath.Join(repoRoot, "scenarios") }, installer: installer})
			resp, err := h.InstallDependency(context.Background(), firstPartyRequest(test.pkg, test.version, true))
			if err != nil {
				t.Fatal(err)
			}
			if !resp.Msg.GetBlocked() || resp.Msg.GetVerdict() != "unrecorded" || installer.called {
				t.Fatalf("blocked=%t verdict=%q installerCalled=%t", resp.Msg.GetBlocked(), resp.Msg.GetVerdict(), installer.called)
			}
		})
	}
}

func TestADenialRecordedForAFirstPartyPackageStillWins(t *testing.T) {
	repoRoot, _ := firstPartyRepo(t, `{"ecosystem":"npm","package_name":"@vrooli/widgets","version_range":"*","state":"denied","rationale":"Retired.","replacement":"Use @vrooli/gadgets."}`)
	h := withTestDiskAdmission(t, &connectHandler{scenariosDir: func() string { return filepath.Join(repoRoot, "scenarios") }, installer: &recordingInstaller{}})
	resp, err := h.InstallDependency(context.Background(), firstPartyRequest("@vrooli/widgets", "file:../../../packages/widgets", false))
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Msg.GetBlocked() || resp.Msg.GetVerdict() != "denied" {
		t.Fatalf("blocked=%t verdict=%q", resp.Msg.GetBlocked(), resp.Msg.GetVerdict())
	}
}

func TestAnApprovedRecordsSurfacesAndRangeDoNotRestrictAFirstPartyPackage(t *testing.T) {
	repoRoot, _ := firstPartyRepo(t, `{"ecosystem":"npm","package_name":"@vrooli/widgets","version_range":"file:../../../packages/elsewhere","range_policy":"exact","state":"approved","rationale":"Old record.","allowed_surfaces":["api"]}`)
	h := withTestDiskAdmission(t, &connectHandler{scenariosDir: func() string { return filepath.Join(repoRoot, "scenarios") }, installer: &recordingInstaller{}})
	resp, err := h.InstallDependency(context.Background(), firstPartyRequest("@vrooli/widgets", "file:../../../packages/widgets", false))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetBlocked() || resp.Msg.GetVerdict() != "first_party" {
		t.Fatalf("blocked=%t verdict=%q", resp.Msg.GetBlocked(), resp.Msg.GetVerdict())
	}
}

func TestValidationDoesNotAskForARecordOfALinkedFirstPartyPackage(t *testing.T) {
	repoRoot, uiDir := firstPartyRepo(t, "")
	registry := NewRegistry(repoRoot)
	manifest := filepath.Join(uiDir, "package.json")
	observed := []*governancev1.ObservedDependency{
		{Ecosystem: "npm", PackageName: "@vrooli/widgets", Version: "file:../../../packages/widgets", FilePath: manifest, DependencyGroup: "dependencies"},
		{Ecosystem: "npm", PackageName: "@vrooli/ghost", Version: "file:../../../packages/ghost", FilePath: manifest, DependencyGroup: "dependencies"},
	}
	resp, err := registry.ValidateObserved("demo", observed)
	if err != nil {
		t.Fatal(err)
	}
	var unrecorded []string
	for _, finding := range resp.GetFindings() {
		if finding.GetFindingClass() == "UNRECORDED_DIRECT" {
			unrecorded = append(unrecorded, finding.GetPackageName())
		}
	}
	if len(unrecorded) != 1 || unrecorded[0] != "@vrooli/ghost" {
		t.Fatalf("only the link the layout cannot prove may need a record, got %v", unrecorded)
	}
}

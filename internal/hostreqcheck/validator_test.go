package hostreqcheck

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	testresource "github.com/vrooli/vrooli/internal/resources/resourcestest"
	"github.com/vrooli/vrooli/internal/scenario"
	testscenario "github.com/vrooli/vrooli/internal/scenario/scenariotest"
)

func TestValidateReportsRootOverreachWithoutScanningUnrelatedScenarioSources(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	testscenario.WriteProjectService(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "vrooli"},
		HostTools: []hostreqspec.Declaration{
			{Name: "git", Required: true, Reason: "root git"},
			{Name: "ffmpeg", Required: false, Reason: "should not be root"},
		},
	})
	testscenario.WriteScenarioService(t, root, "alpha", scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "alpha"},
		HostTools: []hostreqspec.Declaration{
			{Name: "x11vnc", Required: false, Reason: "desktop bridge"},
		},
	})
	testresource.WriteResourceManifest(t, root, "beta", manifestpkg.ResourceManifest{
		Name:      "beta",
		Driver:    "external-cli",
		Binary:    "beta",
		Privilege: hostreqspec.PrivilegeUser,
		Bundling:  hostreqspec.BundlingHostRequired,
	})
	report, err := Validate(root, home)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	assertFinding(t, report, FindingRootOverreach, "root", "vrooli", "ffmpeg")
}

func TestValidateReportsResourceWithoutDeploymentClassification(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectService(t, root, scenario.ServiceManifest{Service: scenario.ServiceMetadata{Name: "vrooli"}})
	testresource.WriteResourceManifest(t, root, "unclassified", manifestpkg.ResourceManifest{
		Name: "unclassified", Driver: "external-cli", Binary: "unclassified",
	})

	report, err := Validate(root, home)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	assertFinding(t, report, FindingMissingClassification, "resource", "unclassified", "")
}

func TestCurrentRepoPhase4DeclarationsPresent(t *testing.T) {
	root := testkitgo.ProjectRoot(t)

	assertManifestContainsTool(t, filepath.Join(root, ".vrooli", "service.json"), "git")
	assertManifestContainsTool(t, filepath.Join(root, ".vrooli", "service.json"), "curl")
	assertManifestContainsTool(t, filepath.Join(root, ".vrooli", "service.json"), "jq")
	assertManifestContainsTool(t, filepath.Join(root, ".vrooli", "service.json"), "java")
	assertManifestContainsTool(t, filepath.Join(root, ".vrooli", "service.json"), "quint")
	assertManifestContainsTool(t, filepath.Join(root, "resources", "codex", "resource.json"), "yq")
	assertManifestContainsTool(t, filepath.Join(root, "scenarios", "web-console", ".vrooli", "service.json"), "tmux")
	assertManifestContainsTool(t, filepath.Join(root, "scenarios", "scenario-to-desktop", ".vrooli", "service.json"), "Xvfb")
	assertManifestContainsTool(t, filepath.Join(root, "scenarios", "scenario-to-desktop", ".vrooli", "service.json"), "websockify")
	assertManifestContainsTool(t, filepath.Join(root, "resources", "whisper", "resource.json"), "ffmpeg")
}

func TestCatalogConformanceFailsForPrivilegeMismatch(t *testing.T) {
	root := t.TempDir()
	testkitgo.WriteFile(t, filepath.Join(root, "internal", "safeguards", "bad", "safeguard.json"), `{"name":"bad","privilege":"user","bundling":"prohibited","verificationCheck":{"files":["/etc/bad.conf"]}}`)
	report := validateSafeguardCatalog(root)
	assertFinding(t, Report{Findings: report}, FindingPrivilegeMismatch, "safeguard", "bad", "")
}

func TestCurrentRepoPhase4ValidatorIsClean(t *testing.T) {
	root := testkitgo.ProjectRoot(t)
	report, err := Validate(root, t.TempDir())
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("expected no current repo host requirement findings, got %+v", report.Findings)
	}
}

func TestPortableToolSourceAcceptsExplicitUnsupportedTargets(t *testing.T) {
	manifest := hostreqkit.ToolManifest{
		Bundling: "vendorable",
		Acquisition: &hostreqkit.ToolSource{
			Kind: "url",
			Targets: []hostreqkit.ToolSourceTarget{
				{When: map[string]string{"os": "linux", "arch": "amd64"}, URL: "https://example.test/tool.zip", SHA256: strings.Repeat("a", 64)},
				{When: map[string]string{"os": "linux", "arch": "arm64"}, Unsupported: "no upstream arm64 release"},
			},
		},
	}
	if !hasPortableToolSource(manifest) {
		t.Fatal("expected a valid target plus explicit unsupported target to satisfy portable source validation")
	}
}

func TestContainsCandidateReferenceIgnoresCommentsAndHyphenatedNames(t *testing.T) {
	if containsCandidateReference("# shellcheck disable=SC1091\n", "shellcheck") {
		t.Fatal("expected comment-only shellcheck reference to be ignored")
	}
	if containsCandidateReference("// tmux is discussed here\n", "tmux") {
		t.Fatal("expected comment-only tmux reference to be ignored")
	}
	if containsCandidateReference("resource-vault status\n", "vault") {
		t.Fatal("expected resource-vault to not count as a vault command reference")
	}
	if containsCandidateReference("vault::status() { :; }\n", "vault") {
		t.Fatal("expected vault shell function names to be ignored")
	}
	if containsCandidateReference("buf := bytes.Buffer{}\n", "buf") {
		t.Fatal("expected generic buf identifiers to be ignored")
	}
	if !containsCandidateReference("if command -v yq >/dev/null; then yq eval '.' file; fi\n", "yq") {
		t.Fatal("expected yq command usage to be detected")
	}
	if !containsCandidateReference("cmd := exec.Command(\"tmux\", \"new-session\")\n", "tmux") {
		t.Fatal("expected quoted tmux exec usage to be detected")
	}
	if !containsCandidateReference("#!/usr/bin/env bats\n", "bats") {
		t.Fatal("expected bats shebang to be detected")
	}
	if !containsCandidateReference("stripe listen --forward-to localhost:3000/api/v1/webhooks/stripe\n", "stripe") {
		t.Fatal("expected stripe CLI usage to be detected")
	}
}

func assertManifestContainsTool(t *testing.T, path, name string) {
	t.Helper()
	names := manifestToolNames(t, path)
	if !slices.Contains(names, name) {
		t.Fatalf("%s does not declare %q", path, name)
	}
}

func manifestToolNames(t *testing.T, path string) []string {
	t.Helper()
	names := []string{}
	switch filepath.Base(path) {
	case "service.json":
		manifest, err := scenario.ReadService(path)
		if err != nil {
			t.Fatalf("ReadService(%s): %v", path, err)
		}
		for _, tool := range manifest.HostTools {
			names = append(names, tool.Name)
		}
	case "resource.json":
		manifest, err := manifestpkg.Load(path)
		if err != nil {
			t.Fatalf("manifest.Load(%s): %v", path, err)
		}
		for _, tool := range manifest.HostTools {
			names = append(names, tool.Name)
		}
	default:
		t.Fatalf("unsupported manifest path %s", path)
	}
	return names
}

func assertFinding(t *testing.T, report Report, code FindingCode, ownerKind, ownerName, requirement string) {
	t.Helper()
	for _, finding := range report.Findings {
		if finding.Code == code &&
			finding.OwnerKind == ownerKind &&
			finding.OwnerName == ownerName &&
			finding.Requirement == requirement {
			return
		}
	}
	t.Fatalf("missing finding %s %s %s %s in %+v", code, ownerKind, ownerName, requirement, report.Findings)
}

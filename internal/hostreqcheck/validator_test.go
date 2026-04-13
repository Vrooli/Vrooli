package hostreqcheck

import (
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqspec"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
	testfixture "github.com/vrooli/vrooli/packages/testkit-go/vrooli"
)

func TestValidateReportsUndeclaredReferencesMissingHandlersAndRootOverreach(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	testfixture.WriteProjectService(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "vrooli"},
		HostTools: []hostreqspec.Declaration{
			{Name: "git", Required: true, Reason: "root git"},
			{Name: "ffmpeg", Required: false, Reason: "should not be root"},
		},
	})
	testfixture.WriteScenarioService(t, root, "alpha", scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "alpha"},
		HostTools: []hostreqspec.Declaration{
			{Name: "x11vnc", Required: false, Reason: "desktop bridge"},
		},
	})
	testkitgo.WriteFile(t, filepath.Join(root, "scenarios", "alpha", "api", "main.go"), `package main

import "os/exec"

func main() {
	_ = exec.Command("websockify", "--help")
}`)
	testfixture.WriteResourceManifest(t, root, "beta", manifestpkg.ResourceManifest{
		Name:            "beta",
		Driver:          "external-cli",
		Binary:          "beta",
		PortabilityTier: "full",
	})
	testkitgo.WriteFile(t, filepath.Join(root, "resources", "beta", "lib", "install.sh"), `#!/usr/bin/env bash
echo ffmpeg`)

	report, err := Validate(root, home)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	assertFinding(t, report, FindingRootOverreach, "root", "vrooli", "ffmpeg")
	assertFinding(t, report, FindingUndeclaredReference, "scenario", "alpha", "websockify")
	assertFinding(t, report, FindingUndeclaredReference, "resource", "beta", "ffmpeg")
}

func TestCurrentRepoPhase4DeclarationsPresent(t *testing.T) {
	root := testkitgo.ProjectRoot(t)

	assertManifestContainsTool(t, filepath.Join(root, ".vrooli", "service.json"), "git")
	assertManifestContainsTool(t, filepath.Join(root, ".vrooli", "service.json"), "curl")
	assertManifestContainsTool(t, filepath.Join(root, ".vrooli", "service.json"), "jq")
	assertManifestContainsTool(t, filepath.Join(root, "resources", "searxng", "resource.json"), "yq")
	assertManifestContainsTool(t, filepath.Join(root, "resources", "codex", "resource.json"), "yq")
	assertManifestContainsTool(t, filepath.Join(root, "scenarios", "web-console", ".vrooli", "service.json"), "tmux")
	assertManifestContainsTool(t, filepath.Join(root, "scenarios", "web-console", ".vrooli", "service.json"), "ffmpeg")
	assertManifestContainsTool(t, filepath.Join(root, "scenarios", "browser-automation-studio", ".vrooli", "service.json"), "ffmpeg")
	assertManifestContainsTool(t, filepath.Join(root, "scenarios", "landing-manager", ".vrooli", "service.json"), "stripe")
	assertManifestContainsTool(t, filepath.Join(root, "scenarios", "landing-page-business-suite", ".vrooli", "service.json"), "stripe")
	assertManifestContainsTool(t, filepath.Join(root, "scenarios", "scenario-to-desktop", ".vrooli", "service.json"), "Xvfb")
	assertManifestContainsTool(t, filepath.Join(root, "scenarios", "scenario-to-desktop", ".vrooli", "service.json"), "websockify")
	assertManifestContainsTool(t, filepath.Join(root, "scenarios", "ecosystem-manager", ".vrooli", "service.json"), "bats")
	assertManifestContainsTool(t, filepath.Join(root, "scenarios", "palette-gen", ".vrooli", "service.json"), "bats")
	assertManifestContainsTool(t, filepath.Join(root, "scenarios", "scenario-completeness-scoring", ".vrooli", "service.json"), "bats")
	assertManifestContainsTool(t, filepath.Join(root, "resources", "whisper", "resource.json"), "ffmpeg")
	assertManifestLacksTool(t, filepath.Join(root, "resources", "vault", "resource.json"), "vault")
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
	if !testkitgo.ContainsString(names, name) {
		t.Fatalf("%s does not declare %q", path, name)
	}
}

func assertManifestLacksTool(t *testing.T, path, name string) {
	t.Helper()
	names := manifestToolNames(t, path)
	if testkitgo.ContainsString(names, name) {
		t.Fatalf("%s unexpectedly declares %q", path, name)
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

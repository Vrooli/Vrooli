package pipeline

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"connectrpc.com/connect"
	att "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/attestation"
	comp "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/composition"
	conf "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/conformance"
	decl "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/declaration"
	dist "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/distribution"
	reh "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/rehearsal"
)

func fixtureRoot(t *testing.T, plugin bool) string {
	t.Helper()
	root := t.TempDir()
	scenario := filepath.Join(root, "scenarios", "fixture")
	if err := os.MkdirAll(filepath.Join(scenario, ".vrooli", "skills", "hello"), 0755); err != nil {
		t.Fatal(err)
	}
	m := map[string]any{"service": map[string]string{"name": "fixture", "version": "1.0.0"}}
	if plugin {
		m["plugin"] = map[string]any{"slug": "fixture", "skills": []any{map[string]any{"name": "hello", "source": "skills/hello/SKILL.md", "command_groups": []string{"hello"}}}, "standalone": map[string]any{"install_script": "cli/install.sh", "runtime_binaries": []string{"cli/fixture"}, "resources": []string{}}}
	}
	b, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(scenario, ".vrooli", "service.json"), b, 0644); err != nil {
		t.Fatal(err)
	}
	if plugin {
		if err := os.MkdirAll(filepath.Join(scenario, "skills", "hello", "..", "..", "cli"), 0755); err != nil {
			t.Fatal(err)
		}
		for _, rel := range []string{"skills/hello/SKILL.md", "cli/install.sh", "cli/fixture"} {
			p := filepath.Join(scenario, rel)
			if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
				t.Fatal(err)
			}
			content := "#!/bin/sh\n"
			if rel == "skills/hello/SKILL.md" {
				content = "---\nname: hello\ndescription: Fixture skill.\n---\n\nRun fixture hello.\n```bash\nfixture hello\n```\n"
			}
			if rel == "cli/install.sh" {
				content = "#!/bin/sh\nmkdir -p \"${FIXTURE_PREFIX:-$HOME/.local/bin}\"\n"
			}
			if err := os.WriteFile(p, []byte(content), 0755); err != nil {
				t.Fatal(err)
			}
		}
		manifest := []byte(`{"groups":[{"name":"hello","commands":[{"name":"hello"}]}]}`)
		if err := os.WriteFile(filepath.Join(scenario, "cli", "manifest.json"), manifest, 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestReadinessNamesMissingDeclaration(t *testing.T) {
	h := &handler{root: fixtureRoot(t, false), packages: map[string]packageRecord{}}
	_, r, _, err := h.readiness("fixture")
	if err != nil {
		t.Fatal(err)
	}
	if r.Eligible || r.BlockingPrerequisite != "PLG-DECL-SOURCE" {
		t.Fatalf("readiness = %+v", r)
	}
}

func TestComposeCopiesOwnedSkillAndRecordsDigest(t *testing.T) {
	h := &handler{root: fixtureRoot(t, true), packages: map[string]packageRecord{}}
	resp, err := h.Compose(context.Background(), connect.NewRequest(&comp.ComposeRequest{Scenario: "fixture", SourceRevision: "abc"}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.Package.Digest == "" {
		t.Fatal("compose returned no digest")
	}
	if _, err := os.Stat(filepath.Join(h.packages[resp.Msg.Package.Id].Root, "plugin.json")); err != nil {
		t.Fatal(err)
	}
}

func TestConformanceFailsClosedWhenPinnedCLISurfaceIsMissing(t *testing.T) {
	root := fixtureRoot(t, true)
	if err := os.Remove(filepath.Join(root, "scenarios", "fixture", "cli", "manifest.json")); err != nil {
		t.Fatal(err)
	}
	h := &handler{root: root, packages: map[string]packageRecord{}}
	resp, err := h.Compose(context.Background(), connect.NewRequest(&comp.ComposeRequest{Scenario: "fixture", SourceRevision: "abc"}))
	if err != nil {
		t.Fatal(err)
	}
	check, err := h.Check(context.Background(), connect.NewRequest(&conf.CheckRequest{PackageId: resp.Msg.Package.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if check.Msg.Passed {
		t.Fatal("conformance unexpectedly passed without cli/manifest.json")
	}
	if !hasFinding(check.Msg.Findings, "PLG-CONF-DRIFT") {
		t.Fatalf("findings = %+v", check.Msg.Findings)
	}
}

func TestConformanceAcceptsNFCTextAndRejectsDecomposedText(t *testing.T) {
	root := fixtureRoot(t, true)
	skill := filepath.Join(root, "scenarios", "fixture", "skills", "hello", "SKILL.md")
	composed := "---\nname: hello\ndescription: Fixture skill.\n---\n\nRun café.\n```bash\nfixture hello\n```\n"
	if err := os.WriteFile(skill, []byte(composed), 0644); err != nil {
		t.Fatal(err)
	}
	h := &handler{root: root, packages: map[string]packageRecord{}}
	resp, err := h.Compose(context.Background(), connect.NewRequest(&comp.ComposeRequest{Scenario: "fixture", SourceRevision: "nfc"}))
	if err != nil {
		t.Fatal(err)
	}
	check, err := h.Check(context.Background(), connect.NewRequest(&conf.CheckRequest{PackageId: resp.Msg.Package.Id}))
	if err != nil || !check.Msg.Passed {
		t.Fatalf("composed NFC skill was rejected: err=%v findings=%+v", err, check.Msg.Findings)
	}
	decomposed := strings.Replace(composed, "é", "e\u0301", 1)
	if err := os.WriteFile(skill, []byte(decomposed), 0644); err != nil {
		t.Fatal(err)
	}
	resp, err = h.Compose(context.Background(), connect.NewRequest(&comp.ComposeRequest{Scenario: "fixture", SourceRevision: "nfc-negative"}))
	if err != nil {
		t.Fatal(err)
	}
	check, err = h.Check(context.Background(), connect.NewRequest(&conf.CheckRequest{PackageId: resp.Msg.Package.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if check.Msg.Passed || !hasFinding(check.Msg.Findings, "PLG-CONF-UNICODE") {
		t.Fatalf("decomposed skill was not rejected: %+v", check.Msg.Findings)
	}
}

func TestCompositionEmitsCanonicalMCPConfigurationAndAuthPosture(t *testing.T) {
	root := fixtureRoot(t, true)
	manifestPath := filepath.Join(root, "scenarios", "fixture", ".vrooli", "service.json")
	manifest := map[string]any{}
	b, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &manifest); err != nil {
		t.Fatal(err)
	}
	plugin := manifest["plugin"].(map[string]any)
	plugin["mcp"] = map[string]any{
		"name":           "fixture-tools",
		"command":        "./cli/fixture",
		"args":           []string{"serve"},
		"authentication": "operator-approved-token",
	}
	b, err = json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, b, 0644); err != nil {
		t.Fatal(err)
	}

	h := &handler{root: root, packages: map[string]packageRecord{}}
	resp, err := h.Compose(context.Background(), connect.NewRequest(&comp.ComposeRequest{Scenario: "fixture", SourceRevision: "mcp"}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.Package.McpAuthentication != "operator-approved-token" {
		t.Fatalf("MCP posture = %q", resp.Msg.Package.McpAuthentication)
	}
	var mcp map[string]any
	b, err = os.ReadFile(filepath.Join(resp.Msg.Package.ArtifactRoot, "mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &mcp); err != nil {
		t.Fatal(err)
	}
	if mcp["$schema"] != "https://agent-plugins.org/schemas/1.0.0/mcp.schema.json" {
		t.Fatalf("MCP schema = %v", mcp["$schema"])
	}
	servers := mcp["mcpServers"].(map[string]any)
	server := servers["fixture-tools"].(map[string]any)
	if server["type"] != "stdio" || server["command"] != "./cli/fixture" {
		t.Fatalf("MCP server = %+v", server)
	}
}

func TestHelloPluginCompositionConformanceAndRehearsal(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../.."))
	h := &handler{root: repoRoot, packages: map[string]packageRecord{}}
	resp, err := h.Compose(context.Background(), connect.NewRequest(&comp.ComposeRequest{Scenario: "hello-plugin", SourceRevision: "test-revision"}))
	if err != nil {
		t.Fatal(err)
	}
	packageRoot := resp.Msg.Package.ArtifactRoot
	var manifest map[string]any
	b, err := os.ReadFile(filepath.Join(packageRoot, "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	if json.Unmarshal(b, &manifest) != nil {
		t.Fatal("plugin.json is not JSON")
	}
	if _, ok := manifest["skills"]; ok {
		t.Fatal("skills must be discovered from the fixed skills/ location")
	}
	if manifest["$schema"] != "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json" {
		t.Fatalf("manifest schema = %v", manifest["$schema"])
	}
	check, err := h.Check(context.Background(), connect.NewRequest(&conf.CheckRequest{PackageId: resp.Msg.Package.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if !check.Msg.Passed {
		t.Fatalf("hello-plugin conformance findings = %+v", check.Msg.Findings)
	}
	rehearsal, err := h.Run(context.Background(), connect.NewRequest(&reh.RunRequest{PackageId: resp.Msg.Package.Id, Sandbox: "workspace-sandbox"}))
	if err != nil {
		t.Fatal(err)
	}
	if !rehearsal.Msg.Passed || len(rehearsal.Msg.Commands) != 2 {
		t.Fatalf("rehearsal = %+v", rehearsal.Msg)
	}
}

func TestWorkspaceSandboxCompositionConformanceAndRehearsal(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../.."))
	h := &handler{root: repoRoot, packages: map[string]packageRecord{}}
	resp, err := h.Compose(context.Background(), connect.NewRequest(&comp.ComposeRequest{Scenario: "workspace-sandbox", SourceRevision: "test-revision"}))
	if err != nil {
		t.Fatal(err)
	}
	check, err := h.Check(context.Background(), connect.NewRequest(&conf.CheckRequest{PackageId: resp.Msg.Package.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if !check.Msg.Passed {
		t.Fatalf("workspace-sandbox conformance findings = %+v", check.Msg.Findings)
	}
	rehearsal, err := h.Run(context.Background(), connect.NewRequest(&reh.RunRequest{PackageId: resp.Msg.Package.Id, Sandbox: "workspace-sandbox"}))
	if err != nil {
		t.Fatal(err)
	}
	if !rehearsal.Msg.Passed || len(rehearsal.Msg.Commands) != 3 {
		t.Fatalf("rehearsal = %+v", rehearsal.Msg)
	}
}

func TestAttestationRequiresConformanceAndEmitsDigestBoundEvidence(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../.."))
	h := &handler{root: repoRoot, packages: map[string]packageRecord{}}
	resp, err := h.Compose(context.Background(), connect.NewRequest(&comp.ComposeRequest{Scenario: "hello-plugin", SourceRevision: "test-revision"}))
	if err != nil {
		t.Fatal(err)
	}
	attested, err := h.Attest(context.Background(), connect.NewRequest(&att.AttestRequest{PackageId: resp.Msg.Package.Id, DryRun: true}))
	if err != nil {
		t.Fatal(err)
	}
	if !attested.Msg.Passed || len(attested.Msg.Evidence) != 3 {
		t.Fatalf("attestation = %+v", attested.Msg)
	}
	for _, evidence := range attested.Msg.Evidence {
		if evidence.Digest != resp.Msg.Package.Digest || evidence.Reference == "dry-run" {
			t.Fatalf("evidence = %+v", evidence)
		}
		if _, err := os.Stat(filepath.Join(resp.Msg.Package.ArtifactRoot, filepath.Base(evidence.Reference))); err != nil {
			t.Fatal(err)
		}
	}
}

func TestManagedAttestationRejectsSecretsBeforePublication(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../.."))
	h := &handler{root: repoRoot, packages: map[string]packageRecord{}}
	resp, err := h.Compose(context.Background(), connect.NewRequest(&comp.ComposeRequest{Scenario: "hello-plugin", SourceRevision: "managed-secret"}))
	if err != nil {
		t.Fatal(err)
	}
	managed := t.TempDir()
	for _, name := range []string{"cosign.signature.json", "provenance.intoto.json", "bom.json"} {
		body := []byte(`{"subject":"sha256:artifact"}`)
		if name == "cosign.signature.json" {
			body = []byte(`{"verificationMaterial":{"certificate":"sk-leaked-secret"},"messageSignature":{}}`)
		}
		if name == "bom.json" {
			body = []byte(`{"components":[{"name":"sk-leaked-secret"}]}`)
		}
		if err := os.WriteFile(filepath.Join(managed, name), body, 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("SCENARIO_TO_PLUGIN_ATTESTATION_DIR", managed)
	attested, err := h.Attest(context.Background(), connect.NewRequest(&att.AttestRequest{PackageId: resp.Msg.Package.Id, DryRun: false}))
	if err != nil {
		t.Fatal(err)
	}
	if attested.Msg.Passed || !hasAttestationFinding(attested.Msg.Findings, "PLG-ATTEST-NO-SECRETS") {
		t.Fatalf("managed secret was not rejected: %+v", attested.Msg)
	}
}

func TestManagedAttestationRequiresDigestBoundEvidence(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../.."))
	h := &handler{root: repoRoot, packages: map[string]packageRecord{}}
	resp, err := h.Compose(context.Background(), connect.NewRequest(&comp.ComposeRequest{Scenario: "hello-plugin", SourceRevision: "managed-valid"}))
	if err != nil {
		t.Fatal(err)
	}
	managed := t.TempDir()
	digest := strings.TrimPrefix(resp.Msg.Package.Digest, "sha256:")
	evidence := map[string][]byte{
		"cosign.signature.json":  []byte(`{"verificationMaterial":{},"messageSignature":{}}`),
		"provenance.intoto.json": []byte(`{"subject":[{"name":"agent-plugin.tar.gz","digest":{"sha256":"` + digest + `"}}]}`),
		"bom.json":               []byte(`{"bomFormat":"CycloneDX","specVersion":"1.6","components":[]}`),
	}
	for name, body := range evidence {
		if err := os.WriteFile(filepath.Join(managed, name), body, 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("SCENARIO_TO_PLUGIN_ATTESTATION_DIR", managed)
	attested, err := h.Attest(context.Background(), connect.NewRequest(&att.AttestRequest{PackageId: resp.Msg.Package.Id, DryRun: false}))
	if err != nil {
		t.Fatal(err)
	}
	if !attested.Msg.Passed || len(attested.Msg.Evidence) != 3 {
		t.Fatalf("managed attestation = %+v", attested.Msg)
	}
}

func TestPublishRefusesWithoutMatchingReleaseDecision(t *testing.T) {
	root := fixtureRoot(t, true)
	h := &handler{root: root, packages: map[string]packageRecord{}}
	resp, err := h.Compose(context.Background(), connect.NewRequest(&comp.ComposeRequest{Scenario: "fixture", SourceRevision: "release-1"}))
	if err != nil {
		t.Fatal(err)
	}
	published, err := h.Publish(context.Background(), connect.NewRequest(&dist.PublishRequest{PackageId: resp.Msg.Package.Id, SourceRevision: "release-1", Channel: "oci"}))
	if err != nil {
		t.Fatal(err)
	}
	if published.Msg.Published || !strings.Contains(published.Msg.Refusal, "PLG-DIST-GATE") {
		t.Fatalf("publish refusal = %+v", published.Msg)
	}
}

func TestPermanentConformanceFixturesFailWithNamedReasons(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../.."))
	cases := []struct{ name, source, want string }{
		{"drifted-command", "SKILL.md", "PLG-CONF-DRIFT"},
		{"angle-frontmatter", "SKILL.md", "PLG-CONF-ANGLE"},
		{"hidden-unicode", "SKILL.md", "PLG-CONF-UNICODE"},
		{"unrestricted-tools", "SKILL.md", "PLG-CONF-TOOLS"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := fixtureRoot(t, true)
			body, err := os.ReadFile(filepath.Join(repoRoot, "scenarios/scenario-to-plugin/testdata/conformance", tc.name, tc.source))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "scenarios", "fixture", "skills", "hello", "SKILL.md"), body, 0644); err != nil {
				t.Fatal(err)
			}
			h := &handler{root: root, packages: map[string]packageRecord{}}
			resp, err := h.Compose(context.Background(), connect.NewRequest(&comp.ComposeRequest{Scenario: "fixture", SourceRevision: tc.name}))
			if err != nil {
				t.Fatal(err)
			}
			check, err := h.Check(context.Background(), connect.NewRequest(&conf.CheckRequest{PackageId: resp.Msg.Package.Id}))
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, finding := range check.Msg.Findings {
				if finding.Code == tc.want {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected %s, findings=%+v", tc.want, check.Msg.Findings)
			}
		})
	}

	installCases := []struct{ name, want string }{{"mutable-download", "PLG-CONF-INSTALL-PIN"}, {"missing-checksum", "PLG-CONF-INSTALL-SUM"}, {"privileged-install", "PLG-CONF-INSTALL-PRIV"}, {"outside-prefix", "PLG-CONF-INSTALL-PRIV"}}
	for _, tc := range installCases {
		t.Run(tc.name, func(t *testing.T) {
			root := fixtureRoot(t, true)
			body, err := os.ReadFile(filepath.Join(repoRoot, "scenarios/scenario-to-plugin/testdata/conformance", tc.name, "install.sh"))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "scenarios", "fixture", "cli", "install.sh"), body, 0755); err != nil {
				t.Fatal(err)
			}
			h := &handler{root: root, packages: map[string]packageRecord{}}
			resp, err := h.Compose(context.Background(), connect.NewRequest(&comp.ComposeRequest{Scenario: "fixture", SourceRevision: tc.name}))
			if err != nil {
				t.Fatal(err)
			}
			check, err := h.Check(context.Background(), connect.NewRequest(&conf.CheckRequest{PackageId: resp.Msg.Package.Id}))
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, finding := range check.Msg.Findings {
				if finding.Code == tc.want {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected %s, findings=%+v", tc.want, check.Msg.Findings)
			}
		})
	}
}

func hasFinding(findings []*conf.Finding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func hasAttestationFinding(findings []*att.Finding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

var _ = decl.Readiness{}

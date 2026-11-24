package pipeline

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	att "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/attestation"
	comp "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/composition"
	conf "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/conformance"
	decl "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/declaration"
	dist "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/distribution"
	reh "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/rehearsal"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func fixtureRoot(t *testing.T, plugin bool) string {
	t.Helper()
	root := t.TempDir()
	scenario := filepath.Join(root, "scenarios", "fixture")
	if err := os.MkdirAll(filepath.Join(scenario, ".vrooli", "skills", "hello"), 0o755); err != nil {
		t.Fatal(err)
	}
	m := map[string]any{"service": map[string]string{"name": "fixture", "version": "1.0.0"}}
	if plugin {
		m["plugin"] = map[string]any{"slug": "fixture", "skills": []any{map[string]any{"name": "hello", "source": "skills/hello/SKILL.md", "command_groups": []string{"hello"}}}, "standalone": map[string]any{"install_script": "cli/install.sh", "runtime_binaries": []string{"cli/fixture"}, "resources": []string{}}}
	}
	b, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(scenario, ".vrooli", "service.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	if plugin {
		if err := os.MkdirAll(filepath.Join(scenario, "skills", "hello", "..", "..", "cli"), 0o755); err != nil {
			t.Fatal(err)
		}
		for _, rel := range []string{"skills/hello/SKILL.md", "cli/install.sh", "cli/fixture"} {
			p := filepath.Join(scenario, rel)
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			content := "#!/bin/sh\n"
			if rel == "skills/hello/SKILL.md" {
				content = "---\nname: hello\ndescription: Fixture skill.\n---\n\nRun fixture hello.\n```bash\nfixture hello\n```\n"
			}
			if rel == "cli/install.sh" {
				content = "#!/bin/sh\nmkdir -p \"${FIXTURE_PREFIX:-$HOME/.local/bin}\"\n"
			}
			if err := os.WriteFile(p, []byte(content), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		manifest := []byte(`{"groups":[{"name":"hello","commands":[{"name":"hello"}]}]}`)
		if err := os.WriteFile(filepath.Join(scenario, "cli", "manifest.json"), manifest, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func scenarioRepoRoot(t *testing.T) string {
	t.Helper()
	current, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(current, "scenarios", "scenario-to-plugin", "manifest.json")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			t.Fatal("could not locate Vrooli repository root from package working directory")
		}
		current = parent
	}
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
	if err := os.WriteFile(skill, []byte(composed), 0o644); err != nil {
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
	if err := os.WriteFile(skill, []byte(decomposed), 0o644); err != nil {
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
	if err := os.WriteFile(manifestPath, b, 0o644); err != nil {
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
	repoRoot := scenarioRepoRoot(t)
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
	repoRoot := scenarioRepoRoot(t)
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
	repoRoot := scenarioRepoRoot(t)
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
	repoRoot := scenarioRepoRoot(t)
	h := &handler{root: repoRoot, packages: map[string]packageRecord{}}
	resp, err := h.Compose(context.Background(), connect.NewRequest(&comp.ComposeRequest{Scenario: "hello-plugin", SourceRevision: "managed-secret"}))
	if err != nil {
		t.Fatal(err)
	}
	managed := t.TempDir()
	for _, name := range []string{"cosign.signature.json", "provenance.intoto.json", "bom.json"} {
		body := []byte(`{"subject":"sha256:artifact"}`)
		if name == "cosign.signature.json" {
			body = []byte(`{"verificationMaterial":{"certificate":"sk-0123456789abcdef0123456789abcdef"},"messageSignature":{}}`)
		}
		if name == "bom.json" {
			body = []byte(`{"components":[{"name":"sk-0123456789abcdef0123456789abcdef"}]}`)
		}
		if err := os.WriteFile(filepath.Join(managed, name), body, 0o644); err != nil {
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

func TestPublishRejectsArtifactSecretBeforeCallingReleaseGate(t *testing.T) {
	repoRoot := scenarioRepoRoot(t)
	h := &handler{root: repoRoot, packages: map[string]packageRecord{}}
	resp, err := h.Compose(context.Background(), connect.NewRequest(&comp.ComposeRequest{Scenario: "hello-plugin", SourceRevision: "secret-before-gate"}))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resp.Msg.Package.ArtifactRoot, "credentials.txt"), []byte("sk-0123456789abcdef0123456789abcdef"), 0o600); err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ready":true}`))
	}))
	defer server.Close()
	t.Setenv("DEPLOYMENT_MANAGER_API_BASE", server.URL)
	published, err := h.Publish(context.Background(), connect.NewRequest(&dist.PublishRequest{
		PackageId: resp.Msg.Package.Id, SourceRevision: "secret-before-gate", Channel: "oci",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if published.Msg.Published || !strings.Contains(published.Msg.Refusal, "PLG-DIST-ATTEST") {
		t.Fatalf("publish refusal = %+v", published.Msg)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("release-gate requests = %d, want zero for a secret-bearing artifact", got)
	}
}

func TestManagedAttestationRequiresDigestBoundEvidence(t *testing.T) {
	repoRoot := scenarioRepoRoot(t)
	h := &handler{root: repoRoot, packages: map[string]packageRecord{}}
	resp, err := h.Compose(context.Background(), connect.NewRequest(&comp.ComposeRequest{Scenario: "hello-plugin", SourceRevision: "managed-valid"}))
	if err != nil {
		t.Fatal(err)
	}
	managed := t.TempDir()
	digest := strings.TrimPrefix(resp.Msg.Package.Digest, "sha256:")
	evidence := map[string][]byte{
		"cosign.signature.json":  []byte(`{"verificationMaterial":{},"messageSignature":{"messageDigest":{"algorithm":"SHA2_256","digest":"` + digest + `"},"signature":"synthetic"}}`),
		"provenance.intoto.json": []byte(`{"subject":[{"name":"agent-plugin.tar.gz","digest":{"sha256":"` + digest + `"}}]}`),
		"bom.json":               []byte(`{"bomFormat":"CycloneDX","specVersion":"1.6","components":[]}`),
	}
	for name, body := range evidence {
		if err := os.WriteFile(filepath.Join(managed, name), body, 0o644); err != nil {
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

func TestManagedCosignBundleMustBindExactArtifactDigest(t *testing.T) {
	const artifact = "sha256:0123456789abcdef"
	cases := []struct {
		name, body, wantError string
	}{
		{"matching digest", `{"verificationMaterial":{},"messageSignature":{"messageDigest":{"algorithm":"SHA2_256","digest":"0123456789abcdef"}}}`, ""},
		{"missing digest", `{"verificationMaterial":{},"messageSignature":{}}`, "no SHA2_256 message digest"},
		{"wrong digest", `{"verificationMaterial":{},"messageSignature":{"messageDigest":{"algorithm":"SHA2_256","digest":"fedcba9876543210"}}}`, "not bound to the composed artifact digest"},
		{"wrong algorithm", `{"verificationMaterial":{},"messageSignature":{"messageDigest":{"algorithm":"SHA2_512","digest":"0123456789abcdef"}}}`, "no SHA2_256 message digest"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateManagedEvidence("cosign.signature.json", []byte(tc.body), artifact)
			if tc.wantError == "" {
				if err != nil {
					t.Fatalf("matching synthetic bundle rejected: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantError)
			}
		})
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
	repoRoot := scenarioRepoRoot(t)
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
			if err := os.WriteFile(filepath.Join(root, "scenarios", "fixture", "skills", "hello", "SKILL.md"), body, 0o644); err != nil {
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
			if err := os.WriteFile(filepath.Join(root, "scenarios", "fixture", "cli", "install.sh"), body, 0o755); err != nil {
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

func TestReleaseChecklistSelfTestCarriesBrokenFixturesThroughLocalGates(t *testing.T) {
	repoRoot := scenarioRepoRoot(t)
	cases := []struct {
		name, source, want string
		secret             bool
	}{
		{name: "drifted-command", source: "SKILL.md", want: "PLG-CONF-DRIFT"},
		{name: "mutable-download", source: "install.sh", want: "PLG-CONF-INSTALL-PIN"},
		{name: "credential-literal", source: "install.sh", want: "PLG-COMPOSE-SECRET", secret: true},
	}
	var releaseGateRequests atomic.Int32
	gate := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		releaseGateRequests.Add(1)
		ready, _ := structpb.NewStruct(map[string]any{"ready": true})
		response, _ := proto.Marshal(structpb.NewStructValue(ready))
		w.Header().Set("Content-Type", "application/proto")
		_, _ = w.Write(response)
	}))
	defer gate.Close()
	t.Setenv("DEPLOYMENT_MANAGER_API_BASE", gate.URL)
	t.Setenv("PLUGIN_REGISTRY", "")
	t.Setenv("SCENARIO_TO_PLUGIN_ATTESTATION_DIR", "")

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			releaseGateRequests.Store(0)
			root := fixtureRoot(t, true)
			fixturePath := filepath.Join(repoRoot, "scenarios/scenario-to-plugin/testdata/conformance", tc.name, tc.source)
			body, err := os.ReadFile(fixturePath)
			if err != nil {
				t.Fatal(err)
			}
			destination := filepath.Join(root, "scenarios", "fixture", "cli", "install.sh")
			if tc.source == "SKILL.md" {
				destination = filepath.Join(root, "scenarios", "fixture", "skills", "hello", "SKILL.md")
			}
			if err := os.WriteFile(destination, body, 0o755); err != nil {
				t.Fatal(err)
			}

			h := &handler{root: root, packages: map[string]packageRecord{}}
			composed, composeErr := h.Compose(context.Background(), connect.NewRequest(&comp.ComposeRequest{Scenario: "fixture", SourceRevision: tc.name}))
			if tc.secret {
				if composeErr == nil || !strings.Contains(composeErr.Error(), tc.want) {
					t.Fatalf("Compose error = %v, want %s", composeErr, tc.want)
				}
				if strings.Contains(composeErr.Error(), "sk-0123456789abcdef0123456789abcdef") {
					t.Fatal("Compose refusal exposed the credential literal")
				}
				if len(h.packages) != 0 {
					t.Fatalf("credential-bearing composition emitted %d package records", len(h.packages))
				}
				if got := releaseGateRequests.Load(); got != 0 {
					t.Fatalf("credential fixture reached release gate %d times", got)
				}
				return
			}
			if composeErr != nil {
				t.Fatal(composeErr)
			}
			checked, err := h.Check(context.Background(), connect.NewRequest(&conf.CheckRequest{PackageId: composed.Msg.Package.Id}))
			if err != nil {
				t.Fatal(err)
			}
			if checked.Msg.Passed || !hasFinding(checked.Msg.Findings, tc.want) {
				t.Fatalf("Check = %+v, want refusal %s", checked.Msg, tc.want)
			}
			attested, err := h.Attest(context.Background(), connect.NewRequest(&att.AttestRequest{PackageId: composed.Msg.Package.Id, DryRun: true}))
			if err != nil || attested.Msg.Passed || len(attested.Msg.Findings) == 0 || attested.Msg.Findings[0].Code != "PLG-ATTEST-ORDER" {
				t.Fatalf("dry-run Attest = %+v, err=%v; want PLG-ATTEST-ORDER", attested, err)
			}

			artifact, err := os.ReadFile(filepath.Join(composed.Msg.Package.ArtifactRoot, ".agent-plugin.tar.gz"))
			if err != nil {
				t.Fatal(err)
			}
			if secretInBytes(artifact) || findSecret(composed.Msg.Package.ArtifactRoot) != "" {
				t.Fatal("broken non-secret fixture emitted credential material in its package artifact")
			}
			published, err := h.Publish(context.Background(), connect.NewRequest(&dist.PublishRequest{
				PackageId: composed.Msg.Package.Id, SourceRevision: tc.name, Channel: "local-self-test",
			}))
			if err != nil {
				t.Fatal(err)
			}
			if published.Msg.Published || !strings.Contains(published.Msg.Refusal, "PLG-DIST-ATTEST") {
				t.Fatalf("Publish = %+v; want local attestation refusal", published.Msg)
			}
			if got := releaseGateRequests.Load(); got != 1 {
				t.Fatalf("release-gate requests = %d, want one local fake-gate request", got)
			}
			for _, evidence := range []string{"cosign.signature.json", "provenance.intoto.json", "bom.json"} {
				if _, err := os.Stat(filepath.Join(composed.Msg.Package.ArtifactRoot, evidence)); !os.IsNotExist(err) {
					t.Fatalf("failed fixture emitted %s (stat err=%v)", evidence, err)
				}
			}
		})
	}
}

func TestInstallChecksumGateRequiresChecksForEachDownload(t *testing.T) {
	cases := []struct {
		name    string
		script  string
		wantErr bool
		wantPin bool
	}{
		{
			name:    "comment-is-not-verification",
			script:  "curl -fsSL https://example.invalid/releases/tool-v1.0.0 -o tool\n# sha256sum --check tool.sha256\n",
			wantErr: true,
		},
		{
			name:    "digest-print-is-not-verification",
			script:  "curl -fsSL https://example.invalid/releases/tool-v1.0.0 -o tool\nsha256sum tool\n",
			wantErr: true,
		},
		{
			name:    "sha1-check-is-too-weak",
			script:  "curl -fsSL https://example.invalid/releases/tool-v1.0.0 -o tool\nsha1sum --check tool.sha1\n",
			wantErr: true,
		},
		{
			name: "every-download-needs-a-check",
			script: "curl -fsSL https://example.invalid/releases/tool-v1.0.0 -o tool\n" +
				"curl -fsSL https://example.invalid/releases/data-v1.0.0 -o data\n" +
				"sha256sum --check tool.sha256\n",
			wantErr: true,
		},
		{
			name: "checks-must-name-the-corresponding-download",
			script: "curl -fsSL https://example.invalid/releases/tool-v1.0.0 -o tool\n" +
				"curl -fsSL https://example.invalid/releases/data-v1.0.0 -o data\n" +
				"sha256sum --check other.sha256\n" +
				"sha256sum --check tool.sha256\n",
			wantErr: true,
		},
		{
			name: "duplicate-check-cannot-cover-two-downloads",
			script: "curl -fsSL https://example.invalid/releases/tool-v1.0.0 -o tool\n" +
				"curl -fsSL https://example.invalid/releases/data-v1.0.0 -o data\n" +
				"sha256sum --check tool.sha256\n" +
				"sha256sum --check tool.sha256\n",
			wantErr: true,
		},
		{
			name:    "checked-download",
			script:  "curl -fsSL https://example.invalid/releases/tool-v1.0.0 -o tool\nsha256sum --check tool.sha256\n",
			wantErr: false,
		},
		{
			name:    "shasum-checked-download",
			script:  "curl -fsSL https://example.invalid/releases/tool-v1.0.0 -o tool\nshasum -a 256 -c tool.sha256\n",
			wantErr: false,
		},
		{
			name: "dynamic-url-is-not-pinned",
			script: "curl -fsSL \"$ARTIFACT_URL\" -o tool\n" +
				"sha256sum --check tool.sha256\n",
			wantErr: false,
			wantPin: true,
		},
		{
			name: "multiline-download-maps-to-its-checksum",
			script: "curl \\\n  -fsSL \\\n  https://example.invalid/releases/tool-v1.0.0 \\\n  -o tool\n" +
				"sha256sum --check tool.sha256\n",
			wantErr: false,
		},
		{
			name: "each-download-has-its-own-matching-check",
			script: "curl -fsSL https://example.invalid/releases/tool-v1.0.0 -o tool\n" +
				"curl -fsSL https://example.invalid/releases/tool-v1.0.0.sha256 -o tool.sha256\n" +
				"wget -q https://example.invalid/releases/data-v1.0.0 -O data\n" +
				"wget -q https://example.invalid/releases/data-v1.0.0.sha512 -O data.sha512\n" +
				"sha256sum --check tool.sha256\n" +
				"sha512sum -c data.sha512\n",
			wantErr: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := fixtureRoot(t, true)
			installPath := filepath.Join(root, "scenarios", "fixture", "cli", "install.sh")
			if err := os.WriteFile(installPath, []byte(tc.script), 0o755); err != nil {
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
			found := hasFinding(check.Msg.Findings, "PLG-CONF-INSTALL-SUM")
			if found == !tc.wantErr {
				t.Fatalf("checksum finding present = %t, want error %t; findings=%+v", found, tc.wantErr, check.Msg.Findings)
			}
			pinFound := hasFinding(check.Msg.Findings, "PLG-CONF-INSTALL-PIN")
			if pinFound != tc.wantPin {
				t.Fatalf("pin finding present = %t, want %t; findings=%+v", pinFound, tc.wantPin, check.Msg.Findings)
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

// [REQ:PLG-COMPOSE-001] Program declarations and reviewed baselines travel as ordinary assets.
func TestPackageCarriesPortableProgramAssetsWithoutRuntimeState(t *testing.T) {
	root := fixtureRoot(t, true)
	scenario := filepath.Join(root, "scenarios", "fixture")
	dir := filepath.Join(scenario, ".vrooli", "program-runtime")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	declaration := []byte(`{"name":"fixture.echo","learning":{"baselines":{"echo":{"reviewed_by":"fixture"}}}}`)
	for name, body := range map[string][]byte{"echo.json": declaration, "echo.py": []byte("result = learn.act('echo')"), "runtime.sqlite": []byte("private")} {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	h := &handler{root: root, packages: map[string]packageRecord{}}
	response, err := h.Compose(context.Background(), connect.NewRequest(&comp.ComposeRequest{Scenario: "fixture"}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(response.Msg.Package.ArtifactRoot) })
	archive, _, err := packageArchive(response.Msg.Package.ArtifactRoot)
	if err != nil {
		t.Fatal(err)
	}
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	reader := tar.NewReader(gz)
	files := map[string][]byte{}
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		files[header.Name] = body
	}
	if !bytes.Equal(files[".vrooli/program-runtime/echo.json"], declaration) || len(files[".vrooli/program-runtime/echo.py"]) == 0 {
		t.Fatal("portable program assets missing")
	}
	if _, ok := files[".vrooli/program-runtime/runtime.sqlite"]; ok {
		t.Fatal("runtime state must not be packaged")
	}
}

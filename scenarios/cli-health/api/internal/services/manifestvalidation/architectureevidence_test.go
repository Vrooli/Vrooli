package manifestvalidation

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// writeScenario lays out a temp scenario root with cli/manifest.json and,
// optionally, the canonical generated primitive-evidence artifact, then returns a
// ctx scoped to it via WithScenarioPath so the provider resolves under the temp
// dir. Use writeScenarioLegacy to exercise the deprecated pre-migration path.
func writeScenario(t *testing.T, manifest, evidence string) context.Context {
	t.Helper()
	return writeScenarioAt(t, manifest, evidence, false)
}

// writeScenarioLegacy writes the evidence artifact at the deprecated
// cli/primitive-evidence.json path instead of the canonical generated location,
// to exercise the provider's migration fallback.
func writeScenarioLegacy(t *testing.T, manifest, evidence string) context.Context {
	t.Helper()
	return writeScenarioAt(t, manifest, evidence, true)
}

func writeScenarioAt(t *testing.T, manifest, evidence string, legacy bool) context.Context {
	t.Helper()
	root := t.TempDir()
	cliDir := filepath.Join(root, "cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cliDir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if evidence != "" {
		artifactPath := cliapp.EvidenceArtifactPath(root)
		if legacy {
			artifactPath = filepath.Join(cliDir, cliapp.EvidenceArtifactFilename)
		}
		if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
			t.Fatalf("mkdir evidence dir: %v", err)
		}
		if err := os.WriteFile(artifactPath, []byte(evidence), 0o644); err != nil {
			t.Fatalf("write evidence: %v", err)
		}
	}
	return WithScenarioPath(context.Background(), root)
}

const provManifest = `{
  "name": "demo",
  "groups": [{"name": "g1", "commands": [{
    "name": "list",
    "binding": {"kind": "connect-rpc", "service": "Svc", "method": "List"},
    "governance": {"effect": "read"},
    "architecture": {"primitive": "proto_list"}
  }]}]
}`

// freshEvidence builds an artifact whose manifest hash matches provManifest. It
// assembles the command tree through a real cli-core primitive builder
// (ProtoList) so the observed evidence is stamped by construction — scenario/test
// code cannot forge it any other way (plan decision D3).
func freshEvidence(t *testing.T) string {
	t.Helper()
	group, err := cliapp.LoadFromManifestPrimitives([]byte(provManifest), "g1", map[string]cliapp.PrimitiveHandler{
		"Svc.List": cliapp.ProtoList(
			func(ctx cliapp.OperationContext) (*wrapperspb.StringValue, error) { return wrapperspb.String(""), nil },
			func(ctx cliapp.OperationContext, resp *wrapperspb.StringValue) cliapp.ListReport {
				return cliapp.ListReport{}
			},
		),
	})
	if err != nil {
		t.Fatalf("assemble command tree: %v", err)
	}
	artifact, err := cliapp.BuildPrimitiveEvidence(cliapp.EvidenceExportInput{
		Scenario:    "demo",
		ManifestRaw: []byte(provManifest),
		Groups:      []cliapp.SubcommandGroup{group},
	})
	if err != nil {
		t.Fatalf("build evidence: %v", err)
	}
	body, err := cliapp.MarshalPrimitiveEvidence(artifact)
	if err != nil {
		t.Fatalf("marshal evidence: %v", err)
	}
	return string(body)
}

func TestFilesystemArchitectureEvidence_PresentAndFresh(t *testing.T) {
	ctx := writeScenario(t, provManifest, freshEvidence(t))
	p := NewFilesystemArchitectureEvidence(t.TempDir())
	ev, err := p.Evidence(ctx, "demo")
	if err != nil {
		t.Fatalf("Evidence: %v", err)
	}
	if ev.Status != EvidenceArtifactOK {
		t.Fatalf("fresh artifact status = %q, want OK", ev.Status)
	}
	if ev.Primitive("g1 list") != cliapp.PrimitiveProtoList {
		t.Fatalf("fresh artifact should expose observed primitive, got %q", ev.Primitive("g1 list"))
	}
}

func TestFilesystemArchitectureEvidence_MissingIsHonestNoEvidence(t *testing.T) {
	ctx := writeScenario(t, provManifest, "") // no evidence file
	p := NewFilesystemArchitectureEvidence(t.TempDir())
	ev, err := p.Evidence(ctx, "demo")
	if err != nil {
		t.Fatalf("Evidence: %v", err)
	}
	if ev.Status != EvidenceArtifactOK {
		t.Fatalf("missing artifact must be OK-with-no-primitives, got status %q", ev.Status)
	}
	if ev.Primitive("g1 list") != "" {
		t.Fatalf("missing artifact must expose no primitives")
	}
}

func TestFilesystemArchitectureEvidence_MalformedStatus(t *testing.T) {
	ctx := writeScenario(t, provManifest, `{"schema":"other/v9","commands":[]}`)
	p := NewFilesystemArchitectureEvidence(t.TempDir())
	ev, err := p.Evidence(ctx, "demo")
	if err != nil {
		t.Fatalf("Evidence should not hard-error on malformed artifact: %v", err)
	}
	if ev.Status != EvidenceArtifactMalformed {
		t.Fatalf("malformed artifact status = %q, want malformed", ev.Status)
	}
	if ev.Primitive("g1 list") != "" {
		t.Fatalf("malformed artifact must not be trusted for evidence")
	}
}

func TestFilesystemArchitectureEvidence_MissingManifestHashIsMalformed(t *testing.T) {
	ctx := writeScenario(t, provManifest, `{
  "schema": "cli-primitive-evidence/v1",
  "scenario": "demo",
  "generator": "v1.0.0",
  "commands": [{"path":"g1 list","command":"list","observed_primitive":"proto_list"}]
}`)
	p := NewFilesystemArchitectureEvidence(t.TempDir())
	ev, err := p.Evidence(ctx, "demo")
	if err != nil {
		t.Fatalf("Evidence should not hard-error on missing manifest hash: %v", err)
	}
	if ev.Status != EvidenceArtifactMalformed {
		t.Fatalf("missing manifest hash status = %q, want malformed", ev.Status)
	}
	if ev.Primitive("g1 list") != "" {
		t.Fatalf("artifact without manifest hash must not be trusted for evidence")
	}
}

// TestFilesystemArchitectureEvidence_LegacyPathFallback proves a scenario still
// mid-migration — evidence at the deprecated cli/primitive-evidence.json path,
// nothing at the canonical location — still validates. The provider falls back to
// the legacy path and reports its location.
func TestFilesystemArchitectureEvidence_LegacyPathFallback(t *testing.T) {
	ctx := writeScenarioLegacy(t, provManifest, freshEvidence(t))
	p := NewFilesystemArchitectureEvidence(t.TempDir())
	ev, err := p.Evidence(ctx, "demo")
	if err != nil {
		t.Fatalf("Evidence: %v", err)
	}
	if ev.Status != EvidenceArtifactOK {
		t.Fatalf("legacy-path artifact status = %q, want OK", ev.Status)
	}
	if ev.Primitive("g1 list") != cliapp.PrimitiveProtoList {
		t.Fatalf("legacy-path artifact should expose observed primitive, got %q", ev.Primitive("g1 list"))
	}
	if filepath.Base(ev.ArtifactPath) != cliapp.EvidenceArtifactFilename {
		t.Fatalf("legacy fallback should report the legacy path, got %q", ev.ArtifactPath)
	}
}

// TestFilesystemArchitectureEvidence_PrefersCanonicalOverLegacy proves that when
// BOTH locations exist, the canonical generated artifact wins.
func TestFilesystemArchitectureEvidence_PrefersCanonicalOverLegacy(t *testing.T) {
	ctx := writeScenario(t, provManifest, freshEvidence(t)) // canonical
	root := scenarioPathFrom(ctx)
	// Also drop a malformed artifact at the legacy path; it must be ignored.
	legacy := filepath.Join(root, "cli", cliapp.EvidenceArtifactFilename)
	if err := os.WriteFile(legacy, []byte(`{"schema":"other/v9"}`), 0o644); err != nil {
		t.Fatalf("write legacy: %v", err)
	}
	p := NewFilesystemArchitectureEvidence(t.TempDir())
	ev, err := p.Evidence(ctx, "demo")
	if err != nil {
		t.Fatalf("Evidence: %v", err)
	}
	if ev.Status != EvidenceArtifactOK || ev.Primitive("g1 list") != cliapp.PrimitiveProtoList {
		t.Fatalf("canonical artifact must win over legacy, got status %q prim %q", ev.Status, ev.Primitive("g1 list"))
	}
	if filepath.Base(filepath.Dir(ev.ArtifactPath)) != "generated" {
		t.Fatalf("expected canonical generated path, got %q", ev.ArtifactPath)
	}
}

func TestFilesystemArchitectureEvidence_StaleWhenManifestChanged(t *testing.T) {
	// Generate fresh evidence, then mutate the manifest so its hash diverges.
	evidence := freshEvidence(t)
	changedManifest := `{
  "name": "demo",
  "groups": [{"name": "g1", "commands": [{
    "name": "list",
    "description": "changed since evidence was generated",
    "binding": {"kind": "connect-rpc", "service": "Svc", "method": "List"},
    "governance": {"effect": "read"},
    "architecture": {"primitive": "proto_list"}
  }]}]
}`
	ctx := writeScenario(t, changedManifest, evidence)
	p := NewFilesystemArchitectureEvidence(t.TempDir())
	ev, err := p.Evidence(ctx, "demo")
	if err != nil {
		t.Fatalf("Evidence: %v", err)
	}
	if ev.Status != EvidenceArtifactStale {
		t.Fatalf("changed manifest must make the artifact stale, got status %q", ev.Status)
	}
	if ev.Primitive("g1 list") != "" {
		t.Fatalf("stale artifact must not be trusted for evidence")
	}
}

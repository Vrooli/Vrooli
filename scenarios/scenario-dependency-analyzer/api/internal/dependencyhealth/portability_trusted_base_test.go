package dependencyhealth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/storage"
)

// writeTrustedBaseGrant records an operator core grant whose trusted base is
// the single scenario "trusted".
func writeTrustedBaseGrant(t *testing.T) {
	t.Helper()
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		t.Fatal(err)
	}
	paths, err := resolver.Resolve(storage.Options{ScenarioID: "vrooli-onboarding"})
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(paths.StateDir, "operator-state.json"), `{"version":"1.0.0","core":{"seed":["trusted"],"trusted_base":["trusted"]}}`)
}

// isolatedTarget stages a target scenario alone in a workspace, the shape
// template deep validation and routed runs hand to the analyzer.
func isolatedTarget(t *testing.T) string {
	t.Helper()
	workspace := t.TempDir()
	scenarioDir := filepath.Join(workspace, "scenarios", "fixture")
	writeFile(t, filepath.Join(scenarioDir, ".vrooli", "service.json"), `{"dependencies":{}}`)
	return scenarioDir
}

func TestValidatePortabilityHostVerifiesTrustedBaseInHostRepository(t *testing.T) {
	writeTrustedBaseGrant(t)
	host := t.TempDir()
	writeFile(t, filepath.Join(host, "scenarios", "trusted", ".vrooli", "service.json"), `{"dependencies":{}}`)

	h := &connectHandler{scenariosDir: func() string { return filepath.Join(host, "scenarios") }}
	if err := h.validatePortabilityHost(isolatedTarget(t), "fixture"); err != nil {
		t.Fatalf("isolated target failed on host-scoped trusted base: %v", err)
	}
}

func TestValidatePortabilityHostRejectsBrokenHostTrustedBase(t *testing.T) {
	writeTrustedBaseGrant(t)
	host := t.TempDir()
	if err := os.MkdirAll(filepath.Join(host, "scenarios"), 0o755); err != nil {
		t.Fatal(err)
	}

	h := &connectHandler{scenariosDir: func() string { return filepath.Join(host, "scenarios") }}
	err := h.validatePortabilityHost(isolatedTarget(t), "fixture")
	if err == nil || !strings.Contains(err.Error(), "trusted-base") {
		t.Fatalf("error = %v, want the host trusted-base closure failure", err)
	}
}

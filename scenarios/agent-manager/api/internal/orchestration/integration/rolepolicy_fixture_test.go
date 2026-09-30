package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/rolepolicy"
)

// testRolePolicyCatalogJSON is a minimal valid role-policy catalog declaring the
// portable role the integration fixtures use ("code.default"). It offers both
// codex and claude-code candidates; the test registries register only the
// claude-code mock runner, so candidate selection deterministically lands there.
const testRolePolicyCatalogJSON = `{
  "schemaVersion":1,
  "metadata":{"catalogId":"integration-test","updatedAt":"2026-07-13"},
  "defaultRole":"code.default",
  "roles":{"code.default":{"description":"test","intent":"test","candidates":[{"runner":"codex","resourceRole":"code.default"},{"runner":"claude-code","resourceRole":"code.default"}]}}
}`

// fakeRoleResolver stands in for the resource-owned role resolver so CreateRun's
// execution-policy resolution runs in these integration tests without shelling
// out to resource CLIs. It reports each requested runner/role as available with
// a concrete model, mirroring a healthy resource response.
type fakeRoleResolver struct{}

func (fakeRoleResolver) Resolve(_ context.Context, r domain.RunnerType, role string) (rolepolicy.ResolvedRole, error) {
	return rolepolicy.ResolvedRole{Runner: r, Role: role, Model: "mock-model"}, nil
}

// newTestRolePolicyOption wires a working role-policy state + resolver into an
// integration-test orchestrator so CreateRun's execution-policy resolution
// succeeds exactly as it does in production. Without it, any CreateRun with a
// RoleRef fails at resolveExecutionPolicy because no catalog/resolver is
// configured.
func newTestRolePolicyOption(t *testing.T) orchestration.Option {
	t.Helper()
	path := filepath.Join(t.TempDir(), "role-policy.json")
	if err := os.WriteFile(path, []byte(testRolePolicyCatalogJSON), 0o600); err != nil {
		t.Fatalf("write role catalog: %v", err)
	}
	state, err := rolepolicy.NewState(path, rolepolicy.Requirement{Required: true})
	if err != nil {
		t.Fatalf("build role policy state: %v", err)
	}
	return orchestration.WithRolePolicyState(state, fakeRoleResolver{})
}

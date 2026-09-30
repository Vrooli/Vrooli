package orchestration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/rolepolicy"
)

const currentModelPolicyFixture = `{"schemaVersion":1,"metadata":{"catalogId":"orchestration-current-policy-fixture","updatedAt":"2026-07-13"},"defaultRole":"code.default","roles":{"code.default":{"description":"test","intent":"test","candidates":[{"runner":"codex","resourceRole":"code.default"},{"runner":"claude-code","resourceRole":"code.default"},{"runner":"opencode","resourceRole":"code.default"}]}}}`

type currentModelPolicyFixtureResolver struct{}

func (currentModelPolicyFixtureResolver) Resolve(_ context.Context, runner domain.RunnerType, role string) (rolepolicy.ResolvedRole, error) {
	return rolepolicy.ResolvedRole{Runner: runner, Role: role, Model: "fixture-model"}, nil
}

func newCurrentModelPolicyFixtureOption(t *testing.T) Option {
	t.Helper()
	path := filepath.Join(t.TempDir(), "role-policy.json")
	if err := os.WriteFile(path, []byte(currentModelPolicyFixture), 0o600); err != nil {
		t.Fatalf("write current role policy fixture: %v", err)
	}
	state, err := rolepolicy.NewState(path, rolepolicy.Requirement{Required: true})
	if err != nil {
		t.Fatalf("build current role policy fixture: %v", err)
	}
	return WithRolePolicyState(state, currentModelPolicyFixtureResolver{})
}

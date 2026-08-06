package ensure

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"resource-openrouter/cli/internal/policy"
	"resource-openrouter/cli/internal/policytest"
)

var errFake = errors.New("fake catalog error")

func fixturePolicy(t *testing.T) policy.Policy {
	t.Helper()
	path := filepath.Join(t.TempDir(), "model-policy.json")
	if err := os.WriteFile(path, []byte(policytest.FixturePolicyJSON), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	p, err := policy.LoadFile(path)
	if err != nil {
		t.Fatalf("load fixture policy: %v", err)
	}
	return p
}

type fakeChecker struct {
	present map[string]bool
	err     error
}

func (f fakeChecker) Present(_ context.Context, _, models []string) (map[string]bool, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := map[string]bool{}
	for _, m := range models {
		out[m] = f.present[m]
	}
	return out, nil
}

func TestRunResolvesRoles(t *testing.T) {
	p := fixturePolicy(t)
	cfg := Config{ModelRoles: []policy.RoleRequest{{Role: "chat.default"}}}
	var out bytes.Buffer
	if err := Run(context.Background(), cfg, p, nil, &out); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "resolved role chat.default -> vendor/chat-a [chat]") {
		t.Fatalf("missing resolution line: %s", out.String())
	}
	if !strings.Contains(out.String(), "skipped") {
		t.Fatalf("expected catalog skip note: %s", out.String())
	}
}

func TestRunUnknownRoleFails(t *testing.T) {
	p := fixturePolicy(t)
	cfg := Config{ModelRoles: []policy.RoleRequest{{Role: "nope"}}}
	if err := Run(context.Background(), cfg, p, nil, &bytes.Buffer{}); err == nil {
		t.Fatal("expected unknown role failure")
	}
}

func TestRunDeprecatedFieldFails(t *testing.T) {
	p := fixturePolicy(t)
	cfg, err := ParseConfig([]byte(`{"model":"openai/gpt-4o"}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := Run(context.Background(), cfg, p, nil, &bytes.Buffer{}); err == nil {
		t.Fatal("expected deprecated-field failure")
	}
}

func TestRunCatalogWarnsOnMissing(t *testing.T) {
	p := fixturePolicy(t)
	cfg := Config{ModelRoles: []policy.RoleRequest{{Role: "chat.default"}}}
	checker := fakeChecker{present: map[string]bool{"vendor/chat-a": true}} // chat-b missing
	var out bytes.Buffer
	if err := Run(context.Background(), cfg, p, checker, &out); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), `model "vendor/chat-b" is not visible`) {
		t.Fatalf("expected missing-model warning: %s", out.String())
	}
}

func TestRunCatalogDegradesOnError(t *testing.T) {
	p := fixturePolicy(t)
	cfg := Config{ModelRoles: []policy.RoleRequest{{Role: "chat.default"}}}
	checker := fakeChecker{err: errFake}
	var out bytes.Buffer
	if err := Run(context.Background(), cfg, p, checker, &out); err != nil {
		t.Fatalf("Run should degrade, not fail: %v", err)
	}
	if !strings.Contains(out.String(), "degraded") {
		t.Fatalf("expected degraded note: %s", out.String())
	}
}

package validationbroker

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContentIdentityResolverUsesDeclaredRepositoryRelativeInputs(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-IDENTITY-P0]
	repoRoot := t.TempDir()
	scenarioRoot := filepath.Join(repoRoot, "scenarios", "demo")
	if err := os.MkdirAll(filepath.Join(scenarioRoot, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioRoot, "src", "main.go"), []byte("package demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	intent := validIntent("plan-manager", "resolver")
	resolver := NewContentIdentityResolver(repoRoot, 1<<20)
	identity, err := resolver.Resolve(context.Background(), intent)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(identity.GetIdentity(), "ci:v1:") || len(identity.GetRoots()) != 1 || identity.GetRoots()[0].GetName() != "scenario" {
		t.Fatalf("resolved identity = %#v", identity)
	}
	if err := os.WriteFile(filepath.Join(scenarioRoot, "src", "main.go"), []byte("package changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := resolver.Resolve(context.Background(), intent)
	if err != nil || changed.GetIdentity() == identity.GetIdentity() {
		t.Fatalf("changed identity = %#v err=%v", changed, err)
	}
}

func TestContentIdentityResolverRejectsEscapingRoot(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-IDENTITY-P0]
	intent := validIntent("plan-manager", "escape")
	intent.ContentInputs[0].Root = "../outside"
	_, err := NewContentIdentityResolver(t.TempDir(), 0).Resolve(context.Background(), intent)
	if err == nil || !strings.Contains(err.Error(), "repository-relative") {
		t.Fatalf("escaping root error = %v", err)
	}
}

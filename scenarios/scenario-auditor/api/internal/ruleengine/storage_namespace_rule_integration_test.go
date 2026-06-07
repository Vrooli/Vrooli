package ruleengine

import (
	"path/filepath"
	"testing"
)

// TestStorageNamespaceRuleInterprets verifies that the variant-aware storage
// namespace rule loads under yaegi (the real runtime path, distinct from the
// native ruletests build) and produces the expected verdicts. It guards against
// using a construct or import that the interpreter cannot resolve.
func TestStorageNamespaceRuleInterprets(t *testing.T) {
	moduleRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve module root: %v", err)
	}
	rulesDir, err := filepath.Abs(filepath.Join("..", "..", "rules"))
	if err != nil {
		t.Fatalf("resolve rules dir: %v", err)
	}

	loader, err := NewLoader(Options{RuleDirs: []string{rulesDir}, ModuleRoot: moduleRoot})
	if err != nil {
		t.Fatalf("new loader: %v", err)
	}

	rules, err := loader.Load()
	if err != nil {
		t.Fatalf("load rules: %v", err)
	}

	info, ok := rules["storage_namespace_helpers"]
	if !ok {
		t.Fatalf("storage_namespace_helpers rule not discovered")
	}
	if !info.Implementation.Valid {
		t.Fatalf("storage_namespace_helpers failed to compile under yaegi: %s", info.Implementation.Error)
	}

	bad := `package idea

import (
	"fmt"

	"github.com/redis/go-redis/v9"
)

func researchKey(rdb *redis.Client, id string) string {
	return fmt.Sprintf("swarm-manager:idea:%s:research", id)
}
`
	violations, err := info.Check(bad, "api/internal/idea/redis.go", "swarm-manager")
	if err != nil {
		t.Fatalf("check bad input: %v", err)
	}
	if len(violations) == 0 {
		t.Fatalf("expected a hardcoded-namespace violation for the bad input")
	}

	good := `package auth

import "github.com/vrooli/api-core/storage"

func sessionKey(id string) (string, error) {
	return storage.RedisKey("auth", "session", id)
}
`
	violations, err = info.Check(good, "api/internal/auth/redis.go", "swarm-manager")
	if err != nil {
		t.Fatalf("check good input: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations for the helper-adopted input, got %d", len(violations))
	}
}

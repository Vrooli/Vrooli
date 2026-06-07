//go:build ruletests
// +build ruletests

package api

import "testing"

func TestStorageNamespaceHelpersDocCases(t *testing.T) {
	runDocTestsViolations(t, "storage_namespace_helpers.go", "api/internal/store/redis.go", CheckStorageNamespaceHelpers)
}

func TestIsRedisKeyShape(t *testing.T) {
	keys := []string{
		`"swarm-manager:idea:%s:research"`,
		`"sandbox:run:%s"`,
		`"lpbs:auth:session:"`,
		`"search:*"`,
	}
	for _, k := range keys {
		if !isRedisKeyShape(k) {
			t.Errorf("expected %s to be recognized as a redis key shape", k)
		}
	}

	notKeys := []string{
		`"localhost:6379"`,
		`"postgres://user:pass@host:5432/db"`,
		`"15:04:05"`,
		`"application/json"`,
		`"foo"`,
		`"foo:bar"`,
	}
	for _, k := range notKeys {
		if isRedisKeyShape(k) {
			t.Errorf("expected %s NOT to be recognized as a redis key shape", k)
		}
	}
}

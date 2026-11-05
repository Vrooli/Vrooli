package lifecycle

import (
	"strings"
	"testing"
)

// A scope name has to be readable in systemctl and stable across restarts,
// so it is derived from the scenario and step rather than minted per launch.
func TestServiceScopeNameIsDerivedAndStable(t *testing.T) {
	first := serviceScopeName("test-genie", "api")
	second := serviceScopeName("test-genie", "api")
	if first != second {
		t.Fatalf("the same service must get the same scope name: %q then %q", first, second)
	}
	if !strings.HasPrefix(first, "vrooli-service-test-genie-api-") {
		t.Fatalf("scope name: got %q", first)
	}
}

// Systemd unit names take a restricted alphabet; a step name with a slash or
// a space must not produce a unit systemd refuses.
func TestServiceScopeNameIsAUsableUnitName(t *testing.T) {
	name := serviceScopeName("React Component/Library", "dev server")
	if strings.ContainsAny(name, " /_.") {
		t.Fatalf("scope name must carry no separator systemd rejects: %q", name)
	}
	if !strings.HasPrefix(name, "vrooli-service-") {
		t.Fatalf("scope name must be recognisable as a service: %q", name)
	}
	if strings.Contains(name, "--") && !strings.Contains(name, "component-library") {
		t.Fatalf("scope name lost the identity it is meant to carry: %q", name)
	}
}

func TestServiceScopeNameStaysInsideSystemdsBound(t *testing.T) {
	name := serviceScopeName(strings.Repeat("scenario", 40), strings.Repeat("step", 40))
	if len(name) > serviceScopeNameLimit {
		t.Fatalf("scope name length %d exceeds the limit %d", len(name), serviceScopeNameLimit)
	}
}

func TestServiceScopesKeepDistinctOwnership(t *testing.T) {
	pairs := [][4]string{
		{"app", "api", "app@shadow", "api"},
		{"app@shadow", "api", "app-shadow", "api"},
		{"a-b", "c", "a", "b-c"},
		{"App", "api", "app", "api"},
		{strings.Repeat("a", 300), "api", strings.Repeat("a", 300), "ui"},
		{"app", "dev/server", "app", "dev-server"},
	}
	for _, p := range pairs {
		if serviceScopeName(p[0], p[1]) == serviceScopeName(p[2], p[3]) {
			t.Fatalf("distinct owners shared scope: %q", p)
		}
	}
}

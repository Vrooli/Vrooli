package lifecycle

import (
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// A stable scope name identifies its scenario and step in systemctl.
func TestServiceScopeNameIsDerivedAndStable(t *testing.T) {
	first := scenarioruntime.ServiceScopeName("test-genie", "api")
	second := scenarioruntime.ServiceScopeName("test-genie", "api")
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
	name := scenarioruntime.ServiceScopeName("React Component/Library", "dev server")
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
	name := scenarioruntime.ServiceScopeName(strings.Repeat("scenario", 40), strings.Repeat("step", 40))
	if len(name) > scenarioruntime.ServiceScopeNameLimit {
		t.Fatalf("scope name length %d exceeds the limit %d", len(name), scenarioruntime.ServiceScopeNameLimit)
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
		if scenarioruntime.ServiceScopeName(p[0], p[1]) == scenarioruntime.ServiceScopeName(p[2], p[3]) {
			t.Fatalf("distinct owners shared scope: %q", p)
		}
	}
}

func TestServiceScopeGenerationIsLimitedToLiveAgentManagerAPI(t *testing.T) {
	const generation = "11223344556677889900aabbccddeeff"
	for _, service := range []struct{ slug, step, generation string }{
		{"agent-manager@shadow", "start-api", generation},
		{"agent-manager", "start-ui", generation},
		{"web-console", "start-api", generation},
		{"agent-manager", "start-api", "bad-generation"},
	} {
		stable := scenarioruntime.ServiceScopeName(service.slug, service.step)
		if got := scenarioruntime.ServiceLaunchScopeName(service.slug, service.step, service.generation); got != stable {
			t.Errorf("%s/%s scope = %q, want %q", service.slug, service.step, got, stable)
		}
	}
}

func TestOnlyLiveRecoveryPlaneGetsOOMPreference(t *testing.T) {
	for _, tc := range []struct {
		slug  string
		avoid bool
	}{
		{"agent-manager", true}, {"web-console", true}, {"vrooli-autoheal", true}, {"prompt-manager", true},
		{"web-console@shadow", false}, {"ordinary-app", false}, {"resource-ollama", false}, {"agent-manager@live", true},
	} {
		if got := recoveryServiceContainment(tc.slug).ManagedOOMAvoid; got != tc.avoid {
			t.Errorf("%s avoidance=%v, want %v", tc.slug, got, tc.avoid)
		}
	}
}

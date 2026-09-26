package domain

import "testing"

// The reachable path and the informative path are often different: probing
// "/health" on a scenario whose dependency report lives at "/api/v1/health"
// reports "no dependencies declared" and hides a real outage.
func TestAppHealthPathDefaultsAndNormalises(t *testing.T) {
	for name, testCase := range map[string]struct{ declared, want string }{
		"undeclared":  {"", "/health"},
		"blank":       {"   ", "/health"},
		"declared":    {"/api/v1/health", "/api/v1/health"},
		"unrooted":    {"api/v1/health", "/api/v1/health"},
		"with spaces": {" /api/v1/health ", "/api/v1/health"},
	} {
		t.Run(name, func(t *testing.T) {
			if got := (ManifestEdge{HealthPath: testCase.declared}).AppHealthPath(); got != testCase.want {
				t.Errorf("AppHealthPath() = %q, want %q", got, testCase.want)
			}
		})
	}
}

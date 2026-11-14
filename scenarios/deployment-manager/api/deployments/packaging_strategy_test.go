package deployments

import "testing"

func TestResolvePackagingStrategyOverridePrecedence(t *testing.T) {
	settings := map[string]interface{}{"packaging_strategy": "electron-sidecar"}
	if got := resolvePackagingStrategy(" pinned-by-request ", settings); got != "pinned-by-request" {
		t.Fatalf("request override = %q", got)
	}
	if got := resolvePackagingStrategy("", settings); got != "electron-sidecar" {
		t.Fatalf("profile pin = %q", got)
	}
	if got := resolvePackagingStrategy("", map[string]interface{}{"custom": "value"}); got != "" {
		t.Fatalf("unset pin = %q", got)
	}
}

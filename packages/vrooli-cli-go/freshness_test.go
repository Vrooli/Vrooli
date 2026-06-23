package vroolicli

import (
	"context"
	"strings"
	"testing"
)

func TestScenarioFreshnessRequiresName(t *testing.T) {
	client := New(WithRunner(&stubRunner{}))
	if _, err := client.ScenarioFreshness(context.Background(), "  "); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestScenarioFreshnessDecodesReport(t *testing.T) {
	runner := &stubRunner{responses: []stubResponse{{output: []byte(`{
		"success": true,
		"scenario": "web-console",
		"stale": true,
		"checks": [
			{"check_type": "ui-bundle", "target": "ui/dist/index.html", "stale": true, "cause": "content changed", "file": "packages/iframe-bridge/dist/index.js"}
		],
		"dependencies": [
			{"name": "browser-automation-studio", "policy": "restart"}
		]
	}`)}}}
	client := New(WithRunner(runner))

	resp, err := client.ScenarioFreshness(context.Background(), "web-console")
	if err != nil {
		t.Fatalf("ScenarioFreshness: %v", err)
	}
	if !resp.GetSuccess() || resp.GetScenario() != "web-console" || !resp.GetStale() {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if len(resp.GetChecks()) != 1 {
		t.Fatalf("expected 1 check, got %d", len(resp.GetChecks()))
	}
	chk := resp.GetChecks()[0]
	if chk.GetCheckType() != "ui-bundle" || !chk.GetStale() || chk.GetCause() != "content changed" {
		t.Errorf("check not decoded: %+v", chk)
	}
	if args := runner.calls[0].args; strings.Join(args, " ") != "--no-stale-check scenario freshness web-console --json" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestScenarioFreshnessWithPath(t *testing.T) {
	runner := &stubRunner{responses: []stubResponse{{output: []byte(`{"success": true, "scenario": "x"}`)}}}
	client := New(WithRunner(runner))

	if _, err := client.ScenarioFreshness(context.Background(), "x", WithFreshnessPath("/custom/path")); err != nil {
		t.Fatalf("ScenarioFreshness: %v", err)
	}
	if args := runner.calls[0].args; strings.Join(args, " ") != "--no-stale-check scenario freshness x --json --path /custom/path" {
		t.Errorf("unexpected args: %v", args)
	}
}

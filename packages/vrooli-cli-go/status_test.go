package vroolicli

import (
	"context"
	"strings"
	"testing"
)

func TestResourceStatusesDecodesFleetForm(t *testing.T) {
	runner := &stubRunner{responses: []stubResponse{{output: []byte(`{
		"success": true,
		"resources": [
			{"resource": {"name": "redis"}, "installed": true, "running": true, "status_code": "ok"}
		],
		"discovery_failures": []
	}`)}}}
	client := New(WithRunner(runner))

	resp, err := client.ResourceStatuses(context.Background())
	if err != nil {
		t.Fatalf("ResourceStatuses: %v", err)
	}
	if !resp.GetSuccess() || len(resp.GetResources()) != 1 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	got := resp.GetResources()[0]
	if got.GetResource().GetName() != "redis" || !got.GetRunning() || got.GetStatusCode() != "ok" {
		t.Errorf("status not decoded: %+v", got)
	}
	// Fleet form takes no name argument.
	if args := runner.calls[0].args; strings.Join(args, " ") != "--no-stale-check resource status --json" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestResourceStatusSingleRequiresName(t *testing.T) {
	client := New(WithRunner(&stubRunner{}))
	if _, err := client.ResourceStatus(context.Background(), "  "); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestResourceStatusSinglePassesName(t *testing.T) {
	runner := &stubRunner{responses: []stubResponse{{output: []byte(`{"success": true, "name": "redis", "running": true}`)}}}
	client := New(WithRunner(runner))

	resp, err := client.ResourceStatus(context.Background(), "redis")
	if err != nil {
		t.Fatalf("ResourceStatus: %v", err)
	}
	if resp.GetName() != "redis" || !resp.GetRunning() {
		t.Errorf("unexpected response: %+v", resp)
	}
	if args := runner.calls[0].args; strings.Join(args, " ") != "--no-stale-check resource status redis --json" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestScenarioStatusesDecodesListForm(t *testing.T) {
	runner := &stubRunner{responses: []stubResponse{{output: []byte(`{
		"success": true,
		"summary": {"total_scenarios": 2, "running": 1, "stopped": 1},
		"scenarios": [
			{"name": "swarm-manager", "status": "running", "processes": 2}
		],
		"discovery_failures": []
	}`)}}}
	client := New(WithRunner(runner))

	resp, err := client.ScenarioStatuses(context.Background())
	if err != nil {
		t.Fatalf("ScenarioStatuses: %v", err)
	}
	if resp.GetSummary().GetTotalScenarios() != 2 || resp.GetSummary().GetRunning() != 1 {
		t.Errorf("summary not decoded: %+v", resp.GetSummary())
	}
	if len(resp.GetScenarios()) != 1 || resp.GetScenarios()[0].GetName() != "swarm-manager" {
		t.Errorf("scenarios not decoded: %+v", resp.GetScenarios())
	}
}

func TestScenarioStatusSinglePassesName(t *testing.T) {
	runner := &stubRunner{responses: []stubResponse{{output: []byte(`{
		"success": true,
		"scenario": {"name": "swarm-manager", "status": "running"},
		"info": {"name": "swarm-manager"},
		"runtime": {"status": "running", "processes": 2}
	}`)}}}
	client := New(WithRunner(runner))

	resp, err := client.ScenarioStatus(context.Background(), "swarm-manager")
	if err != nil {
		t.Fatalf("ScenarioStatus: %v", err)
	}
	if resp.GetScenario().GetName() != "swarm-manager" || resp.GetRuntime().GetProcesses() != 2 {
		t.Errorf("unexpected response: %+v", resp)
	}
	if args := runner.calls[0].args; strings.Join(args, " ") != "--no-stale-check scenario status swarm-manager --json" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestScenarioStatusSingleRequiresName(t *testing.T) {
	client := New(WithRunner(&stubRunner{}))
	if _, err := client.ScenarioStatus(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestListScenariosWithPortsInjectsFlag(t *testing.T) {
	runner := &stubRunner{responses: []stubResponse{{output: []byte(`{"success": true, "scenarios": []}`)}}}
	client := New(WithRunner(runner))

	if _, err := client.ListScenarios(context.Background(), WithPorts()); err != nil {
		t.Fatalf("ListScenarios: %v", err)
	}
	if args := runner.calls[0].args; strings.Join(args, " ") != "--no-stale-check scenario list --json --include-ports" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestListScenariosDefaultOmitsPortsFlag(t *testing.T) {
	runner := &stubRunner{responses: []stubResponse{{output: []byte(`{"success": true, "scenarios": []}`)}}}
	client := New(WithRunner(runner))

	if _, err := client.ListScenarios(context.Background()); err != nil {
		t.Fatalf("ListScenarios: %v", err)
	}
	if args := runner.calls[0].args; strings.Join(args, " ") != "--no-stale-check scenario list --json" {
		t.Errorf("unexpected args: %v", args)
	}
}

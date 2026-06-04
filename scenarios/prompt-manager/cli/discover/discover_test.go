package discover

import (
	"testing"

	clitest "prompt-manager/cli/internal/testutil"
)

func TestFormatNumberAddsThousandsSeparators(t *testing.T) {
	cases := map[int]string{
		999:     "999",
		1200:    "1,200",
		1234567: "1,234,567",
	}
	for input, expected := range cases {
		if got := formatNumber(input); got != expected {
			t.Fatalf("formatNumber(%d) = %q, want %q", input, got, expected)
		}
	}
}

func TestCommandsRegistersDiscoverCommand(t *testing.T) {
	group := Commands(nil)
	if group.Title != "Discovery" || len(group.Commands) != 3 {
		t.Fatalf("unexpected command group: %+v", group)
	}
	byName := map[string]bool{}
	for _, command := range group.Commands {
		if !command.NeedsAPI {
			t.Fatalf("command %q should need the API: %+v", command.Name, command)
		}
		byName[command.Name] = true
	}
	if !byName["discover"] || !byName["discovery-gaps"] || !byName["discovery-metrics"] {
		t.Fatalf("expected discover, discovery-gaps, and discovery-metrics commands, got %+v", group.Commands)
	}
}

func TestCmdDiscoveryMetricsQueriesEndpoint(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/discovery-metrics", DiscoveryMetricsResponse{
		Since:     "168h0m0s",
		CallCount: 3,
		ReturnedCount: DistributionStats{
			Count: 3, Min: 1, P10: 1, Median: 2, P90: 4, Max: 4, Mean: 2.3,
		},
		BudgetedCallCount: 3,
		OverBudgetRate:    0.33,
		PerComplexity: map[string]ComplexityMetric{
			"moderate": {CallCount: 3, OverBudgetRate: 0.33, MedianReturned: 2},
		},
	})
	if err := cmdDiscoveryMetrics(ctx, []string{"--since=7d"}); err != nil {
		t.Fatalf("cmdDiscoveryMetrics: %v", err)
	}
	request := ctx.LastRequest()
	if request.Method != "GET" || request.Path != "/discovery-metrics" {
		t.Fatalf("unexpected request: %+v", request)
	}
}

func TestCmdDiscoveryMetricsRejectsInvalidType(t *testing.T) {
	ctx := clitest.NewContext(t)
	if err := cmdDiscoveryMetrics(ctx, []string{"--type=capability"}); err == nil {
		t.Fatal("expected invalid type to fail")
	}
	ctx.RequireNoRequests()
}

func TestCmdDiscoverDefaultsToSkillRequest(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("POST", "/discover", DiscoverResponse{})
	if err := cmdDiscover(ctx, []string{"debugging"}); err != nil {
		t.Fatalf("cmdDiscover: %v", err)
	}
	request := ctx.LastRequest()
	if request.Method != "POST" || request.Path != "/discover" {
		t.Fatalf("unexpected request: %+v", request)
	}
	req, ok := request.Payload.(DiscoverRequest)
	if !ok {
		t.Fatalf("payload type = %T, want DiscoverRequest", request.Payload)
	}
	if req.Type != "" {
		t.Fatalf("default request type = %q, want empty for legacy API compatibility", req.Type)
	}
}

func TestCmdDiscoverSendsActionType(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("POST", "/discover", DiscoverResponse{})
	if err := cmdDiscover(ctx, []string{"list decisions", "--type=action"}); err != nil {
		t.Fatalf("cmdDiscover: %v", err)
	}
	request := ctx.LastRequest()
	req, ok := request.Payload.(DiscoverRequest)
	if !ok {
		t.Fatalf("payload type = %T, want DiscoverRequest", request.Payload)
	}
	if req.Type != "action" {
		t.Fatalf("request type = %q, want action", req.Type)
	}
}

func TestCmdDiscoverRejectsInvalidType(t *testing.T) {
	ctx := clitest.NewContext(t)
	if err := cmdDiscover(ctx, []string{"debugging", "--type=capability"}); err == nil {
		t.Fatal("expected invalid type to fail")
	}
	ctx.RequireNoRequests()
}

package usage

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	measureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures/measures_v1connect"
)

func TestCalculateCapacityKeepsUnsupportedNumbersUnknown(t *testing.T) {
	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	base := capacityWindow{Provider: "openai", Pool: "primary", Window: "5_hour", HasPercent: true, UsedPercent: 25, Samples: 3, ObservedAt: now.Add(-time.Hour).Format(time.RFC3339), ResetsAt: now.Add(4 * time.Hour).Format(time.RFC3339)}
	got := calculateCapacity(base, now)
	if !got.HasPoints || got.PointsLeft != 75 || !got.HasHours || got.ResetHours != 4 {
		t.Fatalf("quota/reset math = %#v", got)
	}
	if got.HasConcurrent || len(got.Groups) != 0 {
		t.Fatalf("no burner data must not invent rate/capacity: %#v", got)
	}

	base.ObservedAt = now.Add(-25 * time.Hour).Format(time.RFC3339)
	got = calculateCapacity(base, now)
	if got.HasPoints || got.HasHours {
		t.Fatalf("stale input must remain unknown: %#v", got)
	}

	base.ObservedAt = now.Format(time.RFC3339)
	base.Samples = 0
	got = calculateCapacity(base, now)
	if got.HasPoints {
		t.Fatalf("zero samples must remain unknown: %#v", got)
	}
}

func TestCalculateCapacityPreservesBurnRatesAndUnattributed(t *testing.T) {
	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	w := capacityWindow{
		Provider: "openai", Pool: "primary", Window: "5_hour", HasPercent: true, UsedPercent: 20, Samples: 4,
		ObservedAt: now.Add(-time.Minute).Format(time.RFC3339), ResetsAt: now.Add(4 * time.Hour).Format(time.RFC3339),
		PeriodStart: now.Add(-4 * time.Hour).Format(time.RFC3339), PeriodEnd: now.Format(time.RFC3339),
		ObservedIncrease: 12, Unattributed: 3,
		RunningByModel: map[string]int{"model-a": 2, "model-b": 1}, RunningByRole: map[string]int{"research": 2},
		RunningByModelRole: map[string]map[string]int{"model-a": {"research": 1, "review": 1}, "model-b": {"research": 1}},
		Burners:            []*measurepb.UsageBurner{{Model: "model-a", Sessions: 2, Points: 8}, {Model: "model-b", Sessions: 1, Points: 4}},
	}
	got := calculateCapacity(w, now)
	if !got.HasUnattributed || got.Unattributed != 3 {
		t.Fatalf("unattributed remainder lost: %#v", got)
	}
	if len(got.Groups) != 2 || got.Groups[0].MeasuredRate != 2 || got.Groups[0].PerAgentRate != 1 {
		t.Fatalf("expected measured model rate and current-census apportionment, got %#v", got.Groups)
	}
	if got.Groups[0].RunningByRole["research"] != 1 || got.Groups[0].RunningByRole["review"] != 1 || got.Groups[0].EstimatedRateByRole["research"] != 1 || got.Groups[0].EstimatedRateByRole["review"] != 1 {
		t.Fatalf("role/model burn split = %#v, want one matched run and one apportioned point/hour per role", got.Groups[0])
	}
	if !got.HasConcurrent || got.Concurrent != 20 {
		t.Fatalf("sustainable headroom arithmetic = %#v, want floor(80/(4*1)) = 20", got)
	}
	w.RunningByModel = map[string]int{}
	got = calculateCapacity(w, now)
	if got.HasConcurrent {
		t.Fatalf("without matching live runs sustainable count must be unknown: %#v", got)
	}
	w.RunningByModel = map[string]int{"model-a": 2, "model-b": 1, "model-c": 1}
	got = calculateCapacity(w, now)
	if got.HasConcurrent {
		t.Fatalf("one unmeasured active model must keep sustainable count unknown: %#v", got)
	}
}

func TestMakeCapacityProtoUsesNullForUnsupportedCapacity(t *testing.T) {
	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	r := makeCapacityProto(&measurepb.UsageForecastResponse{GeneratedAt: now.Format(time.RFC3339), Windows: []*measurepb.UsageWindowForecast{{Provider: "openai", Pool: "primary", Window: "5_hour", HasPercent: true, UsedPercent: 20, SampleCount: 2, ObservedAt: now.Format(time.RFC3339), ResetsAt: now.Add(time.Hour).Format(time.RFC3339)}}}, &measurepb.UsageBurnersResponse{}, &measurepb.UsageBurnersResponse{}, nil)
	w := r.AsMap()["windows"].([]any)[0].(map[string]any)
	if v, ok := w["sustainable_agents"]; !ok || v != nil {
		t.Fatalf("unsupported capacity should serialize as null, got %#v", w["sustainable_agents"])
	}
	if v, ok := w["points_left"]; !ok || v.(float64) != 80 {
		t.Fatalf("points left not serialized: %#v", v)
	}
	if w["resets_at"] != time.Date(2026, 11, 12, 13, 0, 0, 0, time.UTC).Format(time.RFC3339) {
		t.Fatalf("absolute reset time missing: %#v", w["resets_at"])
	}
}

func TestCensusCountsUsesResolvedModelAndRole(t *testing.T) {
	runs := []*domainpb.Run{
		{ActualModel: "model-a", ResolvedConfig: &domainpb.RunConfig{RoleRef: "research", RunnerType: domainpb.RunnerType_RUNNER_TYPE_CODEX}},
		{ResolvedConfig: &domainpb.RunConfig{Model: "model-b", RoleRef: "review", RunnerType: domainpb.RunnerType_RUNNER_TYPE_CODEX}},
		{RequestedModel: "model-a", ResolvedConfig: &domainpb.RunConfig{RunnerType: domainpb.RunnerType_RUNNER_TYPE_CLAUDE_CODE}},
	}
	models, roles, modelRoles := censusCounts(runs, "openai", nil)
	if models["model-a"] != 1 || models["model-b"] != 1 || roles["research"] != 1 || roles["review"] != 1 {
		t.Fatalf("Codex census model/role counts = %#v %#v", models, roles)
	}
	if modelRoles["model-a"]["research"] != 1 || modelRoles["model-b"]["review"] != 1 {
		t.Fatalf("Codex model/role census = %#v", modelRoles)
	}
	models, roles, modelRoles = censusCounts(runs, "anthropic", nil)
	if models["model-a"] != 1 || roles["unknown"] != 1 || roles["research"] != 0 {
		t.Fatalf("Claude census model/role counts = %#v %#v", models, roles)
	}
	if modelRoles["model-a"]["unknown"] != 1 {
		t.Fatalf("Claude model/role census = %#v", modelRoles)
	}
	models, roles, modelRoles = censusCounts([]*domainpb.Run{{Id: "indexed", HarnessKind: "codex"}}, "openai", map[string]runUsageFact{"indexed": {Model: "model-c", Runner: "codex", ProfileID: "operator"}})
	if models["model-c"] != 1 || roles["operator"] != 1 || modelRoles["model-c"]["operator"] != 1 {
		t.Fatalf("fresh indexed session census = %#v %#v %#v", models, roles, modelRoles)
	}
}

type capacityFixture struct {
	measureconnect.UnimplementedMeasuresServiceHandler
	forecastAt string
	requests   []*measurepb.UsageBurnersRequest
}

func (f *capacityFixture) UsageForecast(context.Context, *connect.Request[measurepb.UsageForecastRequest]) (*connect.Response[measurepb.UsageForecastResponse], error) {
	now := time.Now().UTC()
	f.forecastAt = now.Format(time.RFC3339)
	return connect.NewResponse(&measurepb.UsageForecastResponse{GeneratedAt: f.forecastAt, Windows: []*measurepb.UsageWindowForecast{{Provider: "openai", Pool: "primary", Window: "10080m", HasPercent: true, UsedPercent: 20, SampleCount: 3, ObservedAt: f.forecastAt, ResetsAt: now.Add(4 * time.Hour).Format(time.RFC3339)}}}), nil
}

func (f *capacityFixture) UsageBurners(_ context.Context, req *connect.Request[measurepb.UsageBurnersRequest]) (*connect.Response[measurepb.UsageBurnersResponse], error) {
	f.requests = append(f.requests, req.Msg)
	now := time.Now().UTC()
	return connect.NewResponse(&measurepb.UsageBurnersResponse{Windows: []*measurepb.UsageWindowAttribution{{Provider: "openai", Pool: "primary", Window: "10080m", PeriodStart: now.Add(-time.Hour).Format(time.RFC3339), PeriodEnd: now.Format(time.RFC3339), ObservedIncreasePoints: 6, UnattributedPoints: 1, Burners: []*measurepb.UsageBurner{{Model: "model-a", Points: 6}}}}}), nil
}

func TestCapacityCommandUsesRecentBurnAndLiveCensusForHeadroom(t *testing.T) {
	fixture := &capacityFixture{}
	path, handler := measureconnect.NewMeasuresServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	censusCalls := 0
	cmd := capacityCommand(func(*cliapp.ScenarioApp) measureconnect.MeasuresServiceClient {
		return measureconnect.NewMeasuresServiceClient(server.Client(), server.URL)
	}, func() ([]*domainpb.Run, error) {
		censusCalls++
		return []*domainpb.Run{{ActualModel: "model-a", ResolvedConfig: &domainpb.RunConfig{RoleRef: "research", RunnerType: domainpb.RunnerType_RUNNER_TYPE_CODEX}}, {ActualModel: "model-a", ResolvedConfig: &domainpb.RunConfig{RoleRef: "research", RunnerType: domainpb.RunnerType_RUNNER_TYPE_CODEX}}}, nil
	})
	var stdout bytes.Buffer
	ctx, err := cliapp.NewTestRunContextFromArgs(cmd.Args, []string{"--json"}, nil, &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunCtx(ctx); err != nil {
		t.Fatal(err)
	}
	if censusCalls != 1 {
		t.Fatalf("live census calls = %d", censusCalls)
	}
	if len(fixture.requests) != 2 || fixture.requests[0].GetGroupBy() != "model" || fixture.requests[1].GetGroupBy() != "run" || fixture.requests[1].GetLimit() != 10000 {
		t.Fatalf("burner queries = %#v, want bounded model and run evidence", fixture.requests)
	}
	since, err := time.Parse(time.RFC3339, fixture.requests[0].GetSince())
	if err != nil || time.Since(since) < 59*time.Minute || time.Since(since) > 61*time.Minute {
		t.Fatalf("burner since = %q, want previous hour", fixture.requests[0].GetSince())
	}
	if fixture.requests[0].GetSince() != fixture.requests[1].GetSince() {
		t.Fatalf("model and run evidence used different windows: %+v", fixture.requests)
	}
	var got struct {
		Windows []struct {
			Sustainable *float64 `json:"sustainable_agents"`
			Running     int      `json:"running_agents"`
			Groups      []struct {
				Model    string  `json:"model"`
				Running  int     `json:"running_agents"`
				Measured float64 `json:"measured_points_per_hour"`
				PerRun   float64 `json:"estimated_points_per_hour_per_running_run"`
			} `json:"groups"`
		} `json:"windows"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &got); err != nil {
		t.Fatalf("capacity JSON: %v\n%s", err, stdout.String())
	}
	if len(got.Windows) != 1 || got.Windows[0].Sustainable == nil || *got.Windows[0].Sustainable != 6 {
		t.Fatalf("capacity headroom = %#v", got.Windows)
	}
	if got.Windows[0].Running != 2 || len(got.Windows[0].Groups) != 1 || got.Windows[0].Groups[0].Running != 2 || got.Windows[0].Groups[0].Measured != 6 || got.Windows[0].Groups[0].PerRun != 3 {
		t.Fatalf("model census/rate output = %#v", got.Windows[0])
	}
	window := makeCapacityProto(&measurepb.UsageForecastResponse{GeneratedAt: fixture.forecastAt, Windows: []*measurepb.UsageWindowForecast{{Provider: "openai", Pool: "primary", Window: "10080m", HasPercent: true, UsedPercent: 20, SampleCount: 3, ObservedAt: fixture.forecastAt, ResetsAt: time.Now().UTC().Add(4 * time.Hour).Format(time.RFC3339)}}},
		&measurepb.UsageBurnersResponse{Windows: []*measurepb.UsageWindowAttribution{{Provider: "openai", Pool: "primary", Window: "10080m", PeriodStart: time.Now().UTC().Add(-time.Hour).Format(time.RFC3339), PeriodEnd: time.Now().UTC().Format(time.RFC3339), ObservedIncreasePoints: 6, Burners: []*measurepb.UsageBurner{{Model: "model-a", Points: 6}}}}},
		&measurepb.UsageBurnersResponse{}, []*domainpb.Run{{ActualModel: "model-a", ResolvedConfig: &domainpb.RunConfig{RoleRef: "research", RunnerType: domainpb.RunnerType_RUNNER_TYPE_CODEX}}, {ActualModel: "model-a", ResolvedConfig: &domainpb.RunConfig{RoleRef: "review", RunnerType: domainpb.RunnerType_RUNNER_TYPE_CODEX}}}).AsMap()["windows"].([]any)[0].(map[string]any)
	groups := window["groups"].([]any)
	group := groups[0].(map[string]any)
	if group["running_by_role"].(map[string]any)["research"] != float64(1) || group["running_by_role"].(map[string]any)["review"] != float64(1) {
		t.Fatalf("role/model capacity census = %#v", group)
	}
	if window["running_by_model_role"].(map[string]any)["model-a"].(map[string]any)["review"] != float64(1) {
		t.Fatalf("role/model census missing from window = %#v", window["running_by_model_role"])
	}
}

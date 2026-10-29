package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures/measures_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// runStatsMeasures is the provider fixture: one typed response per procedure
// `run stats` reads. A procedure missing here fails the command, so the test
// pins exactly which measures the summary is composed from.
func runStatsMeasures() map[string]proto.Message {
	breakdown := func(value string) []*measurepb.RunBreakdownRow {
		return []*measurepb.RunBreakdownRow{{Value: value, RunCount: 3, SuccessCount: 2, FailedCount: 1, TotalCostUsd: 1.25, TotalTokens: 900, AverageDurationMs: 1500}}
	}
	return map[string]proto.Message{
		measures_v1connect.MeasuresServiceRunStatusDistributionProcedure: &measurepb.RunStatusDistributionResponse{Rows: []*measurepb.RunStatusCount{{Status: "complete", Count: 2}, {Status: "failed", Count: 1}, {Status: "running", Count: 4}}},
		measures_v1connect.MeasuresServiceRunSuccessRateProcedure:        &measurepb.RunSuccessRateResponse{Rate: 0.667, SuccessfulRuns: 2, TerminalRuns: 3},
		measures_v1connect.MeasuresServiceRunDurationStatisticsProcedure: &measurepb.RunDurationStatisticsResponse{Count: 3, AverageDurationMs: 1500, P50DurationMs: 1200, P95DurationMs: 2400, P99DurationMs: 2400, MinDurationMs: 900, MaxDurationMs: 2400},
		measures_v1connect.MeasuresServiceRunCostProcedure:               &measurepb.RunCostResponse{TotalCostUsd: 1.25, AverageCostUsd: 0.4167, TotalRuns: 3, TotalTokens: 900, InputTokens: 500, OutputTokens: 300, CacheReadTokens: 100},
		measures_v1connect.MeasuresServiceRunnerBreakdownProcedure:       &measurepb.RunnerBreakdownResponse{Rows: breakdown("codex")},
		measures_v1connect.MeasuresServiceProfileBreakdownProcedure:      &measurepb.ProfileBreakdownResponse{Rows: breakdown("Builder")},
		measures_v1connect.MeasuresServiceModelBreakdownProcedure:        &measurepb.ModelBreakdownResponse{Rows: breakdown("gpt-test")},
		measures_v1connect.MeasuresServiceToolUsageProcedure:             &measurepb.ToolUsageResponse{Rows: []*measurepb.ToolUsageRow{{ToolName: "read_file", CallCount: 7, SuccessCount: 6, FailedCount: 1, TotalTokens: 40}}},
	}
}

type runStatsServer struct {
	mu       sync.Mutex
	requests map[string]string
}

func (f *runStatsServer) start(t *testing.T) *App {
	t.Helper()
	f.requests = map[string]string{}
	responses := runStatsMeasures()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.requests[r.URL.Path] = string(body)
		f.mu.Unlock()
		response, ok := responses[r.URL.Path]
		if r.Method != http.MethodPost || !ok {
			http.Error(w, "not a run stats measure: "+r.Method+" "+r.URL.Path, http.StatusNotFound)
			return
		}
		data, err := protojson.Marshal(response)
		if err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}))
	t.Cleanup(server.Close)
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: server.URL} }, func() string { return "owner-fixture" })
	return &App{services: NewServices(api)}
}

func TestRunStatsSummarizesMeasures(t *testing.T) {
	fixture := &runStatsServer{}
	app := fixture.start(t)
	output := captureStdout(t, func() error {
		return app.runStats([]string{"--since", "2026-10-01T00:00:00Z", "--until", "2026-10-02T00:00:00Z", "--profile", "p-1", "--tag-prefix", "eco-", "--runner", "codex", "--model", "gpt-test"})
	})
	for _, want := range []string{
		"Run stats 2026-10-01T00:00:00Z .. 2026-10-02T00:00:00Z",
		"Runs: total=7 complete=2 failed=1 running=4",
		"Success: rate=0.667 (2/3 terminal)",
		"Duration: n=3 avg=1.5s p50=1.2s p95=2.4s p99=2.4s min=900ms max=2.4s",
		"Cost: total=$1.2500",
		"Tokens: total=900 input=500 output=300 cache_read=100",
		"Runners:", "codex", "Profiles:", "Builder", "Models:", "gpt-test",
		"Tools:", "read_file", "calls=7",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q:\n%s", want, output)
		}
	}
	if len(fixture.requests) != len(runStatsMeasures()) {
		t.Fatalf("read %d measures, want %d: %v", len(fixture.requests), len(runStatsMeasures()), fixture.requests)
	}
	wantFilter := &measurepb.InvocationFilter{ProfileId: "p-1", TagPrefix: "eco-", RunnerType: "codex", Model: "gpt-test"}
	for procedure, body := range fixture.requests {
		// Every summary measure shares the {window, filter} request shape.
		var request measurepb.RunVolumeRequest
		if err := protojson.Unmarshal([]byte(body), &request); err != nil {
			t.Fatalf("%s request %s: %v", procedure, body, err)
		}
		if !proto.Equal(request.GetFilter(), wantFilter) {
			t.Errorf("%s filter=%v want %v", procedure, request.GetFilter(), wantFilter)
		}
		custom := request.GetWindow().GetCustom()
		if custom.GetFrom().AsTime().Format(time.RFC3339) != "2026-10-01T00:00:00Z" || custom.GetTo().AsTime().Format(time.RFC3339) != "2026-10-02T00:00:00Z" {
			t.Errorf("%s window=%v", procedure, request.GetWindow())
		}
	}
}

func TestRunStatsJSONCarriesEachMeasureResponse(t *testing.T) {
	app := (&runStatsServer{}).start(t)
	output := captureStdout(t, func() error { return app.runStats([]string{"--json"}) })
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("decode --json output: %v\n%s", err, output)
	}
	for _, key := range []string{"statusDistribution", "successRate", "duration", "cost", "runners", "profiles", "models", "tools"} {
		if len(decoded[key]) == 0 {
			t.Errorf("--json missing %s: %s", key, output)
		}
	}
}

func TestRunStatsWindowDefaultsToTheLastDay(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	window, err := runStatsWindow("", "", now)
	if err != nil {
		t.Fatal(err)
	}
	if got := window.GetCustom().GetFrom().AsTime(); !got.Equal(now.Add(-24 * time.Hour)) {
		t.Fatalf("from=%v", got)
	}
	if got := window.GetCustom().GetTo().AsTime(); !got.Equal(now) {
		t.Fatalf("to=%v", got)
	}
	if _, err := runStatsWindow("2026-10-30T00:00:00Z", "", now); err == nil {
		t.Fatal("since after until must be refused")
	}
	if _, err := runStatsWindow("yesterday", "", now); err == nil {
		t.Fatal("non-RFC3339 since must be refused")
	}
}

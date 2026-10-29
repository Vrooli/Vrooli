package usage

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	measureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures/measures_v1connect"
)

// measuresFixture serves the real generated Connect handler with canned
// provider answers and records which procedures the CLI read.
type measuresFixture struct {
	measureconnect.UnimplementedMeasuresServiceHandler
	procedures []string
	providers  []string
}

func (f *measuresFixture) UsageForecast(_ context.Context, req *connect.Request[measurepb.UsageForecastRequest]) (*connect.Response[measurepb.UsageForecastResponse], error) {
	f.procedures, f.providers = append(f.procedures, "UsageForecast"), append(f.providers, req.Msg.GetProvider())
	return connect.NewResponse(&measurepb.UsageForecastResponse{GeneratedAt: "2026-10-29T06:00:00Z", Windows: []*measurepb.UsageWindowForecast{{
		Provider: "openai", Pool: "primary", Window: "10080m", WindowMinutes: 10080, Status: "will_exhaust", StatusReason: "the whole projection band ends before the reset",
		HasPercent: true, UsedPercent: 34, LastReadingPercent: 34, ResetsAt: "2026-11-04T06:57:00Z", ObservedAt: "2026-10-29T05:48:31Z", SampleCount: 2104,
		Burn:                  []*measurepb.UsageBurnRate{{Horizon: "1h", Stale: true}, {Horizon: "6h", Available: true, PercentPerHour: 2.33}, {Horizon: "24h", Available: true, PercentPerHour: 1.48}},
		CentralPercentPerHour: 1.21, CalibrationFactor: 0.93, CalibrationWindows: 10,
		ProjectedExhaustionAt: "2026-10-31T02:40:47Z", ExhaustionBandStart: "2026-10-30T11:58:41Z", ExhaustionBandEnd: "2026-10-31T16:35:23Z",
		LastingPercentPerHour: 0.46, EarlyReset: true, PreviousEndedAt: "2026-10-28T06:57:00Z", PreviousResetsAt: "2026-11-02T06:08:00Z", PreviousFinalPercent: 27,
	}, {Provider: "anthropic", Pool: "five_hour", Window: "five_hour", Status: "unknown", StatusReason: "the newest reading is older than 6 hours", Standing: "available", ObservedAt: "2026-10-28T20:00:00Z"}, {
		Provider: "anthropic", Pool: "seven_day", Window: "10080m", WindowMinutes: 10080, Status: "on_track", StatusReason: "projected exhaustion is after the reset",
		HasPercent: true, UsedPercent: 23, LastReadingPercent: 21, ResetsAt: "2026-11-04T15:00:00Z", ObservedAt: "2026-10-29T06:41:20Z", SampleCount: 7,
		Burn: []*measurepb.UsageBurnRate{{Horizon: "1h"}}, CentralPercentPerHour: 0.2, CalibrationFactor: 1, CalibrationWindows: 0,
	}, {Provider: "openai", Pool: "secondary", Window: "300m", WindowMinutes: 300, Status: "idle", StatusReason: "no work has used this provider since the last reading, so it lasts at this pace", HasPercent: true, UsedPercent: 34, ObservedAt: "2026-10-29T05:00:00Z"}}}), nil
}

func (f *measuresFixture) UsageForecastBacktest(_ context.Context, req *connect.Request[measurepb.UsageForecastBacktestRequest]) (*connect.Response[measurepb.UsageForecastBacktestResponse], error) {
	f.procedures, f.providers = append(f.procedures, "UsageForecastBacktest"), append(f.providers, req.Msg.GetProvider())
	return connect.NewResponse(&measurepb.UsageForecastBacktestResponse{
		UnobservedWindows: 2,
		Validity:          &measurepb.MeasureValidity{State: "available"},
		Summaries: []*measurepb.UsageBacktestSummary{
			{Population: "last_5_windows_per_scope", ElapsedFraction: 0.25, Windows: 9, MedianAbsErrorPoints: 5, BaselineMedianAbsErrorPoints: 5},
			{Population: "all_windows", ElapsedFraction: 0.5, Windows: 9, MedianAbsErrorPoints: 6.8, BaselineMedianAbsErrorPoints: 7, BeatsBaseline: true},
		},
		Windows: []*measurepb.UsageBacktestWindow{{Provider: "openai", Window: "10080m", WindowStart: "2026-10-26T06:08:00Z", EndedAt: "2026-10-28T06:57:00Z", EarlyReset: true, FinalPercent: 27, OutcomeObserved: true, Checkpoints: []*measurepb.UsageBacktestCheckpoint{{ElapsedFraction: 0.5, PredictedFinalPercent: 34, BaselinePredictedFinalPercent: 20}}}},
	}), nil
}

func runForecast(t *testing.T, fixture *measuresFixture, args ...string) string {
	t.Helper()
	path, handler := measureconnect.NewMeasuresServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	cmd := forecastCommand(func(*cliapp.ScenarioApp) measureconnect.MeasuresServiceClient {
		return measureconnect.NewMeasuresServiceClient(server.Client(), server.URL)
	})
	var stdout bytes.Buffer
	ctx, err := cliapp.NewTestRunContextFromArgs(cmd.Args, args, nil, &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunCtx(ctx); err != nil {
		t.Fatalf("usage forecast %v: %v", args, err)
	}
	return stdout.String()
}

func TestUsageCommandsReadMeasuresProcedures(t *testing.T) {
	fixture := &measuresFixture{}
	out := runForecast(t, fixture, "--provider", "openai")
	if strings.Join(fixture.procedures, ",") != "UsageForecast" || fixture.providers[0] != "openai" {
		t.Fatalf("read %v with providers %v", fixture.procedures, fixture.providers)
	}
	for _, want := range []string{
		"Codex · weekly  ███████░░░░░░░░░░░░░ 34%  resets Wed Nov 4 06:57 UTC",
		"WILL RUN OUT ~Sat Oct 31 02:40 UTC (band Fri Oct 30 11:58 UTC – Sat Oct 31 16:35 UTC)",
		"burn 1h stale · 6h 2.33%/h · 24h 1.48%/h   to last until reset: ≤0.46%/h",
		"projected burn to reset 1.21%/h (calibrated ×0.93 on 10 windows)",
		"last reading 34% at Thu Oct 29 05:48 UTC (2104 readings)",
		"burn 1h —",
		"projected burn to reset 0.20%/h (uncalibrated: 0 completed window(s))",
		"last reading 21% at Thu Oct 29 06:41 UTC (7 readings); a lagging reader, the window stands at 23%",
		"early reset: this window began Wed Oct 28 06:57 UTC, before the previous window's Mon Nov 2 06:08 UTC reset (it ended at 27%)",
		"Claude · 5-hour  standing: available",
		"unknown: the newest reading is older than 6 hours",
		"Codex · 5-hour",
		"idle, lasts at this pace: no work has used this provider",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}

	jsonOut := runForecast(t, fixture, "--json")
	var decoded struct {
		Windows []struct {
			Status                string  `json:"status"`
			ProjectedExhaustionAt string  `json:"projected_exhaustion_at"`
			LastReadingPercent    float64 `json:"last_reading_percent"`
		} `json:"windows"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &decoded); err != nil || len(decoded.Windows) != 4 || decoded.Windows[0].Status != "will_exhaust" || decoded.Windows[0].ProjectedExhaustionAt == "" || decoded.Windows[0].LastReadingPercent != 34 {
		t.Fatalf("--json must be the Measures wire shape: %v\n%s", err, jsonOut)
	}
}

func TestUsageForecastBacktestReportsErrorAgainstLinearBaseline(t *testing.T) {
	fixture := &measuresFixture{}
	out := runForecast(t, fixture, "--backtest")
	if strings.Join(fixture.procedures, ",") != "UsageForecastBacktest" {
		t.Fatalf("read %v", fixture.procedures)
	}
	for _, want := range []string{
		"Last 5 windows per scope at 25% of a window: median error 5.0 pts vs linear baseline 5.0 pts over 9 window(s) — forecast does not beat the baseline",
		"All windows at 50% of a window: median error 6.8 pts vs linear baseline 7.0 pts over 9 window(s) — forecast beats the baseline",
		"2 completed window(s) not scored",
		"Codex · weekly Mon Oct 26 06:08 UTC → Wed Oct 28 06:57 UTC final 27% (early reset); @50% predicted 34% (baseline 20%)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	jsonOut := runForecast(t, fixture, "--backtest", "--json")
	var decoded struct {
		Summaries []struct {
			BeatsBaseline bool `json:"beats_baseline"`
		} `json:"summaries"`
		UnobservedWindows string `json:"unobserved_windows"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &decoded); err != nil || len(decoded.Summaries) != 2 || !decoded.Summaries[1].BeatsBaseline || decoded.UnobservedWindows != "2" {
		t.Fatalf("--backtest --json must carry the summaries: %v\n%s", err, jsonOut)
	}
}

func TestRegisterDeclaresTheUsageGroup(t *testing.T) {
	group := Register()
	var names []string
	for _, cmd := range group.Subcommands {
		names = append(names, cmd.Name)
		if err := cmd.Args.Validate(); err != nil {
			t.Fatalf("%s: %v", cmd.Name, err)
		}
	}
	// SPEC §4 and P2-D1: usage forecast|burners|recommendations|friction (with
	// its report and dismiss actions)|coverage|compare.
	if group.Name != "usage" || strings.Join(names, ",") != "forecast,burners,recommendations,friction,friction-report,friction-dismiss,coverage,compare" {
		t.Fatalf("group %q declares %v", group.Name, names)
	}
}

// Window names match the UI (format.ts windowTitle): slot-only pools are
// dropped, named pools kept, and colliding titles append the pool id.
func TestWindowTitlesMatchTheUIAndDisambiguateCollisions(t *testing.T) {
	titles := windowTitles([]windowScope{
		{"anthropic", "five_hour", "300m", 300},
		{"anthropic", "seven_day", "10080m", 10080},
		{"anthropic", "seven_day_opus", "10080m", 10080},
		{"openai", "primary", "10080m", 10080},
		{"openai", "secondary", "10080m", 10080},
		{"openai", "primary", "300m", 0},
	})
	want := []string{"Claude · 5-hour", "Claude · weekly", "Claude · weekly · Opus", "Codex · weekly · primary", "Codex · weekly · secondary", "Codex · 5-hour"}
	for i := range want {
		if titles[i] != want[i] {
			t.Errorf("title %d = %q, want %q", i, titles[i], want[i])
		}
	}
}

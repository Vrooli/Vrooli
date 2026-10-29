// Package usage exposes provider subscription usage questions. Every command
// reads Agent Manager's Measures procedures; the API owns the computation.
package usage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/proto"

	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	measureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures/measures_v1connect"
)

// clientFor builds the Measures client a command reads; tests replace it.
type clientFor func(*cliapp.ScenarioApp) measureconnect.MeasuresServiceClient

func connectClient(core *cliapp.ScenarioApp) measureconnect.MeasuresServiceClient {
	h, base := cliapp.NewConnectHTTPClient(core)
	return measureconnect.NewMeasuresServiceClient(h, base)
}

// Register declares the `usage` group; run adds the per-run commands the
// CLI's run services implement (`usage run`, `usage reindex`).
func Register(run ...cliapp.Command) cliapp.SubcommandGroup {
	commands := []cliapp.Command{forecastCommand(connectClient), burnersCommand(connectClient), recommendationsCommand(connectClient), coverageCommand(connectClient), compareCommand(connectClient)}
	return cliapp.SubcommandGroup{Name: "usage", Description: "Provider subscription usage from the provider's own quota readings", NeedsAPI: true, Subcommands: append(commands, run...)}
}

func forecastCommand(newClient clientFor) cliapp.Command {
	cmd := cliapp.Command{
		Name: "forecast", NeedsAPI: true,
		Description: "Will each subscription window run out before its reset? [--backtest] [--provider NAME] [--json]",
		Usage:       "agent-manager usage forecast [--backtest] [--provider NAME] [--json]",
		HelpText:    "Projects each provider window from its used-percent readings: burn over the last 1h/6h/24h (stale when no reading lies inside the horizon), projected run-out with a band from the burn since the window began shaped by the operator's hour-of-week profile and calibrated on completed windows, the burn that lasts to the reset, the last reading, and on_track | at_risk | will_exhaust | unknown. --backtest replays the forecast in time order over completed windows against a naive linear baseline: the last 5 windows per scope (SPEC A4), then all windows.",
		Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "provider", Description: "Provider filter (e.g. openai, anthropic)"},
			{Name: "backtest", Bool: true, LocalOnly: true, Description: "Score the forecast on completed windows"},
		}},
	}
	return cmd.WithPrimitive(cliapp.ProtoList(
		func(op cliapp.OperationContext) (proto.Message, error) {
			client, provider := newClient(op.Core()), strings.TrimSpace(op.Flag("provider"))
			if op.BoolFlag("backtest") {
				response, err := client.UsageForecastBacktest(context.Background(), connect.NewRequest(&measurepb.UsageForecastBacktestRequest{Provider: provider}))
				if err != nil {
					return nil, err
				}
				return response.Msg, nil
			}
			response, err := client.UsageForecast(context.Background(), connect.NewRequest(&measurepb.UsageForecastRequest{Provider: provider}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response proto.Message) cliapp.ListReport {
			switch r := response.(type) {
			case *measurepb.UsageForecastBacktestResponse:
				return backtestReport(r)
			case *measurepb.UsageForecastResponse:
				return forecastReport(r)
			}
			return cliapp.ListReport{}
		},
	))
}

func forecastReport(r *measurepb.UsageForecastResponse) cliapp.ListReport {
	report := cliapp.ListReport{ResultsHeading: "Windows", Summary: []string{fmt.Sprintf("%d provider window(s) as of %s", len(r.GetWindows()), stamp(r.GetGeneratedAt()))}}
	if len(r.GetWindows()) == 0 {
		report.Summary = append(report.Summary, "No provider quota readings yet: the status is unknown, not zero.")
	}
	scopes := make([]windowScope, len(r.GetWindows()))
	for i, w := range r.GetWindows() {
		scopes[i] = windowScope{w.GetProvider(), w.GetPool(), w.GetWindow(), w.GetWindowMinutes()}
	}
	titles := windowTitles(scopes)
	for i, w := range r.GetWindows() {
		report.Results = append(report.Results, windowLines(w, titles[i])...)
	}
	report.RetrievalHints = []string{"agent-manager usage forecast --backtest  # how accurate the forecast has been"}
	return report
}

func windowLines(w *measurepb.UsageWindowForecast, title string) []string {
	head := title
	if w.GetHasPercent() {
		head += fmt.Sprintf("  %s %.0f%%", bar(w.GetUsedPercent(), 20), w.GetUsedPercent())
	} else {
		head += "  standing: " + w.GetStanding()
	}
	if w.GetResetsAt() != "" {
		head += "  resets " + stamp(w.GetResetsAt())
	}
	lines := []string{head, "  " + statusLine(w)}
	if w.GetHasPercent() {
		parts := make([]string, 0, len(w.GetBurn()))
		for _, b := range w.GetBurn() {
			switch {
			case b.GetAvailable():
				parts = append(parts, fmt.Sprintf("%s %.2f%%/h", b.GetHorizon(), b.GetPercentPerHour()))
			case b.GetStale():
				parts = append(parts, b.GetHorizon()+" stale")
			default:
				parts = append(parts, b.GetHorizon()+" —")
			}
		}
		burn := "  burn " + strings.Join(parts, " · ")
		if w.GetLastingPercentPerHour() > 0 {
			burn += fmt.Sprintf("   to last until reset: ≤%.2f%%/h", w.GetLastingPercentPerHour())
		}
		lines = append(lines, burn)
		if w.GetCentralPercentPerHour() > 0 {
			lines = append(lines, "  projected burn to reset "+fmt.Sprintf("%.2f%%/h", w.GetCentralPercentPerHour())+calibrationNote(w.GetCalibrationFactor(), w.GetCalibrationWindows()))
		}
	}
	if w.GetEarlyReset() {
		lines = append(lines, fmt.Sprintf("  early reset: this window began %s, before the previous window's %s reset (it ended at %.0f%%)", stamp(w.GetPreviousEndedAt()), stamp(w.GetPreviousResetsAt()), w.GetPreviousFinalPercent()))
	}
	last := "  last reading "
	if w.GetHasPercent() {
		last += fmt.Sprintf("%.0f%% ", w.GetLastReadingPercent())
	}
	last += fmt.Sprintf("at %s (%d readings)", stamp(w.GetObservedAt()), w.GetSampleCount())
	if w.GetHasPercent() && w.GetLastReadingPercent() < w.GetUsedPercent() {
		last += fmt.Sprintf("; a lagging reader, the window stands at %.0f%%", w.GetUsedPercent())
	}
	return append(lines, last)
}

func calibrationNote(factor float64, windows int64) string {
	if windows < 3 {
		return fmt.Sprintf(" (uncalibrated: %d completed window(s))", windows)
	}
	return fmt.Sprintf(" (calibrated ×%.2f on %d windows)", factor, windows)
}

func statusLine(w *measurepb.UsageWindowForecast) string {
	switch w.GetStatus() {
	case "will_exhaust", "at_risk":
		label := "WILL RUN OUT"
		if w.GetStatus() == "at_risk" {
			label = "AT RISK"
		}
		if w.GetProjectedExhaustionAt() == "" {
			return fmt.Sprintf("%s: %s", label, w.GetStatusReason())
		}
		band := ""
		if w.GetExhaustionBandStart() != "" {
			end := "never"
			if w.GetExhaustionBandEnd() != "" {
				end = stamp(w.GetExhaustionBandEnd())
			}
			band = fmt.Sprintf(" (band %s – %s)", stamp(w.GetExhaustionBandStart()), end)
		}
		return fmt.Sprintf("%s ~%s%s", label, stamp(w.GetProjectedExhaustionAt()), band)
	case "on_track":
		if w.GetProjectedExhaustionAt() != "" {
			return "on track: projected run-out " + stamp(w.GetProjectedExhaustionAt()) + " is after the reset"
		}
		return "on track: " + w.GetStatusReason()
	}
	return "unknown: " + w.GetStatusReason()
}

func backtestReport(r *measurepb.UsageForecastBacktestResponse) cliapp.ListReport {
	report := cliapp.ListReport{ResultsHeading: "Windows"}
	for _, s := range r.GetSummaries() {
		verdict := "does not beat"
		if s.GetBeatsBaseline() {
			verdict = "beats"
		}
		population := "All windows"
		if s.GetPopulation() == "last_5_windows_per_scope" {
			population = "Last 5 windows per scope"
		}
		line := fmt.Sprintf("%s at %.0f%% of a window: median error %.1f pts vs linear baseline %.1f pts over %d window(s) — forecast %s the baseline", population, s.GetElapsedFraction()*100, s.GetMedianAbsErrorPoints(), s.GetBaselineMedianAbsErrorPoints(), s.GetWindows(), verdict)
		if s.GetHourErrorWindows() > 0 {
			line += fmt.Sprintf("; run-out time error %.1fh vs %.1fh (%d exhausted)", s.GetMedianAbsErrorHours(), s.GetBaselineMedianAbsErrorHours(), s.GetHourErrorWindows())
		}
		report.Summary = append(report.Summary, line)
	}
	if v := r.GetValidity(); v.GetState() == "unreliable" {
		report.Summary = append(report.Summary, "Unreliable: "+v.GetReason())
	}
	if r.GetUnobservedWindows() > 0 {
		report.Summary = append(report.Summary, fmt.Sprintf("%d completed window(s) not scored: no reading within 6h of their end", r.GetUnobservedWindows()))
	}
	for _, w := range r.GetWindows() {
		line := fmt.Sprintf("%s %s → %s final %.0f%%", windowTitle(windowScope{w.GetProvider(), w.GetPool(), w.GetWindow(), 0}), stamp(w.GetWindowStart()), stamp(w.GetEndedAt()), w.GetFinalPercent())
		if w.GetEarlyReset() {
			line += " (early reset)"
		}
		if !w.GetOutcomeObserved() {
			line += " [not scored]"
		}
		for _, c := range w.GetCheckpoints() {
			line += fmt.Sprintf("; @%.0f%% predicted %.0f%% (baseline %.0f%%)", c.GetElapsedFraction()*100, c.GetPredictedFinalPercent(), c.GetBaselinePredictedFinalPercent())
		}
		report.Results = append(report.Results, line)
	}
	return report
}

func providerLabel(provider string) string {
	switch provider {
	case "openai":
		return "Codex"
	case "anthropic":
		return "Claude"
	}
	return provider
}

// windowLabel is "weekly", "daily" or "5-hour"; an unrecognised window is
// returned as-is.
func windowLabel(minutes int64, window string) string {
	if minutes == 0 && strings.HasSuffix(window, "m") {
		fmt.Sscanf(window, "%dm", &minutes)
	}
	if minutes == 0 {
		minutes = namedWindowMinutes[window]
	}
	switch {
	case minutes == 10080:
		return "weekly"
	case minutes == 1440:
		return "daily"
	case minutes > 0 && minutes%60 == 0:
		return fmt.Sprintf("%d-hour", minutes/60)
	}
	return window
}

var namedWindowMinutes = map[string]int64{"five_hour": 300, "seven_day": 10080}

// unnamedPools only restate the window or the provider's slot order.
var unnamedPools = map[string]bool{"primary": true, "secondary": true, "five_hour": true, "seven_day": true}

type windowScope struct {
	provider, pool, window string
	minutes                int64
}

// windowTitle names a window as the UI does (ui/src/features/usage/format.ts
// windowTitle): "Claude · weekly", "Codex · 5-hour", and "Claude · weekly ·
// Opus" when the pool has a name of its own.
func windowTitle(s windowScope) string {
	parts := []string{providerLabel(s.provider), windowLabel(s.minutes, s.window)}
	if pool := s.pool; pool != "" && !unnamedPools[pool] {
		for prefix := range namedWindowMinutes {
			pool = strings.TrimPrefix(pool, prefix+"_")
		}
		words := strings.FieldsFunc(pool, func(r rune) bool { return r == '_' || r == '-' })
		for i, word := range words {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
		parts = append(parts, strings.Join(words, " "))
	}
	return strings.Join(parts, " · ")
}

// windowTitles names every scope; scopes whose titles collide (two live Codex
// weekly windows, primary and secondary) each append their pool id:
// "Codex · weekly · primary" and "Codex · weekly · secondary".
func windowTitles(scopes []windowScope) []string {
	titles := make([]string, len(scopes))
	count := map[string]int{}
	for i, s := range scopes {
		titles[i] = windowTitle(s)
		count[titles[i]]++
	}
	for i, s := range scopes {
		if count[titles[i]] > 1 && s.pool != "" {
			titles[i] += " · " + s.pool
		}
	}
	return titles
}

func bar(percent float64, width int) string {
	filled := int(percent/100*float64(width) + 0.5)
	filled = max(0, min(width, filled))
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func stamp(value string) string {
	if value == "" {
		return "unknown"
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return t.UTC().Format("Mon Jan 2 15:04 UTC")
}

package usage

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

type RunningRunCensus func() ([]*domainpb.Run, error)

// capacityWindow is the CLI's deliberately small, pure arithmetic input. A
// burner unit is a session in the selected model group; it is not a claim about
// simultaneous sessions or a causal per-agent cost.
type capacityWindow struct {
	Provider, Pool, Window string
	UsedPercent            float64
	HasPercent             bool
	Samples                int64
	ObservedAt, ResetsAt   string
	Burners                []*measurepb.UsageBurner
	RunningByModel         map[string]int
	RunningByRole          map[string]int
	RunningByModelRole     map[string]map[string]int
	Unattributed           float64
	ObservedIncrease       float64
	PeriodStart, PeriodEnd string
}

type capacityRow struct {
	Title, State                       string
	ResetsAt                           string
	PointsLeft, ResetHours, Concurrent float64
	HasPoints, HasHours, HasConcurrent bool
	Groups                             []capacityGroup
	RunningByRole                      map[string]int
	Unattributed                       float64
	HasUnattributed                    bool
}

type capacityGroup struct {
	Model                      string
	Running                    int
	MeasuredRate, PerAgentRate float64
	HasMeasuredRate            bool
	RunningByRole              map[string]int
	EstimatedRateByRole        map[string]float64
}

// calculateCapacity uses no inferred zeroes: missing, invalid, stale, and
// zero-use observations stay unknown. Burn rates are measured across the
// recent window. Dividing by the live matching-model census is an explicit
// current-census estimate, not per-run billing or measured active hours.
func calculateCapacity(w capacityWindow, now time.Time) capacityRow {
	row := capacityRow{Title: windowTitle(windowScope{w.Provider, w.Pool, w.Window, 0})}
	if w.ObservedIncrease >= 0 {
		row.Unattributed, row.HasUnattributed = w.Unattributed, true
	}
	if !w.HasPercent || w.Samples <= 0 || w.UsedPercent < 0 || w.UsedPercent > 100 {
		row.State = "unknown: no usable quota sample"
		return row
	}
	observed, err := time.Parse(time.RFC3339, w.ObservedAt)
	if err != nil || now.Sub(observed) > 24*time.Hour || observed.After(now.Add(5*time.Minute)) {
		row.State = "unknown: quota sample is stale or has an invalid timestamp"
		return row
	}
	row.PointsLeft, row.HasPoints = math.Max(0, 100-w.UsedPercent), true
	reset, err := time.Parse(time.RFC3339, w.ResetsAt)
	if err != nil || !reset.After(now) {
		row.State = "unknown: reset time unavailable or passed"
		return row
	}
	row.ResetHours, row.HasHours = reset.Sub(now).Hours(), true
	row.ResetsAt = reset.UTC().Format(time.RFC3339)
	start, e1 := time.Parse(time.RFC3339, w.PeriodStart)
	end, e2 := time.Parse(time.RFC3339, w.PeriodEnd)
	if e1 != nil || e2 != nil || !end.After(start) || w.ObservedIncrease <= 0 {
		row.State = "unknown: burner samples do not support a rate"
		return row
	}
	periodHours := end.Sub(start).Hours()
	observedRate, matchingRuns, totalRunning := 0.0, 0, 0
	for _, count := range w.RunningByModel {
		totalRunning += count
	}
	for _, b := range w.Burners {
		if b == nil || b.GetPoints() <= 0 {
			continue
		}
		model := strings.TrimSpace(b.GetModel())
		if model == "" {
			model = "unknown model"
		}
		rate := b.GetPoints() / periodHours
		count := w.RunningByModel[model]
		roleCounts := w.RunningByModelRole[model]
		group := capacityGroup{
			Model: model, Running: count, MeasuredRate: rate, HasMeasuredRate: true,
			RunningByRole: cloneIntCounts(roleCounts), EstimatedRateByRole: map[string]float64{},
		}
		if count > 0 && rate > 0 && !math.IsInf(rate, 0) && !math.IsNaN(rate) {
			group.PerAgentRate = rate / float64(count)
			for role, roleCount := range roleCounts {
				group.EstimatedRateByRole[role] = group.PerAgentRate * float64(roleCount)
			}
			matchingRuns += count
			observedRate += rate
		}
		row.Groups = append(row.Groups, group)
	}
	seenModels := make(map[string]bool, len(row.Groups))
	for _, group := range row.Groups {
		seenModels[group.Model] = true
	}
	for model, count := range w.RunningByModel {
		if !seenModels[model] {
			row.Groups = append(row.Groups, capacityGroup{Model: model, Running: count, RunningByRole: cloneIntCounts(w.RunningByModelRole[model])})
		}
	}
	row.RunningByRole = w.RunningByRole
	// A running run whose model shows no measured burn in the period was idle
	// for it (attached operator sessions often are). It is excluded from the
	// divisor, and the numerator is the provider's whole observed increase when
	// that exceeds the attributed burn, so unattributed or off-disk usage and
	// finished runs' usage are charged to the runs that did burn. Both choices
	// can only raise the per-run rate, never invent headroom.
	idleRuns := totalRunning - matchingRuns
	if providerRate := w.ObservedIncrease / periodHours; providerRate > observedRate && !math.IsInf(providerRate, 0) && !math.IsNaN(providerRate) {
		observedRate = providerRate
	}
	if observedRate > 0 && matchingRuns > 0 && row.HasPoints && row.HasHours {
		row.Concurrent = math.Floor(row.PointsLeft / (row.ResetHours * (observedRate / float64(matchingRuns))))
		row.HasConcurrent = true
		row.State = "current-census estimate from recent measured model burn; assumes currently running matching-model runs represent the one-hour usage window"
		if idleRuns > 0 {
			row.State += fmt.Sprintf("; %d running run(s) without measured burn in the window were idle and are excluded, and the provider's whole observed increase is charged to the %d run(s) that burned", idleRuns, matchingRuns)
		}
	} else {
		row.State = "unknown: no currently running run has measured burn in the recent window"
	}
	return row
}

func cloneIntCounts(counts map[string]int) map[string]int {
	copy := make(map[string]int, len(counts))
	for key, count := range counts {
		copy[key] = count
	}
	return copy
}

func capacityCommand(newClient clientFor, census RunningRunCensus) cliapp.Command {
	cmd := cliapp.Command{Name: "capacity", NeedsAPI: true, Description: "How much provider quota remains for running agents? [--provider NAME] [--json]", Usage: "agent-manager usage capacity [--provider NAME] [--json]", HelpText: "Reads current forecast, model-grouped usage from the last hour, and the live run census. Model burn divided across currently running matching runs is a current-census estimate, not per-run billing or measured active hours. Missing or stale evidence remains unknown; off-disk usage stays unattributed.", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{providerFlag}}}
	return cmd.WithPrimitive(cliapp.ProtoList(func(op cliapp.OperationContext) (proto.Message, error) {
		client := newClient(op.Core())
		provider := strings.TrimSpace(op.Flag("provider"))
		forecast, err := client.UsageForecast(context.Background(), connect.NewRequest(&measurepb.UsageForecastRequest{Provider: provider}))
		if err != nil {
			return nil, err
		}
		cutoff := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
		burners, err := client.UsageBurners(context.Background(), connect.NewRequest(&measurepb.UsageBurnersRequest{Provider: provider, GroupBy: "model", Since: cutoff}))
		if err != nil {
			return nil, err
		}
		runBurners, err := client.UsageBurners(context.Background(), connect.NewRequest(&measurepb.UsageBurnersRequest{Provider: provider, GroupBy: "run", Since: cutoff, Limit: 10000}))
		if err != nil {
			return nil, err
		}
		var runs []*domainpb.Run
		if census != nil {
			runs, err = census()
			if err != nil {
				return nil, fmt.Errorf("read live run census: %w", err)
			}
		}
		return makeCapacityProto(forecast.Msg, burners.Msg, runBurners.Msg, runs), nil
	}, func(_ cliapp.OperationContext, response proto.Message) cliapp.ListReport {
		r := response.(*structpb.Struct).AsMap()
		windows, _ := r["windows"].([]any)
		report := cliapp.ListReport{ResultsHeading: "Provider windows", Summary: []string{"Capacity is estimated from provider readings; per-session-hour is not concurrent-agent telemetry."}}
		for _, raw := range windows {
			w, _ := raw.(map[string]any)
			title, _ := w["title"].(string)
			line := title
			if reset, ok := w["resets_at"].(string); ok && reset != "" {
				report.Results = append(report.Results, "  resets at "+reset)
			} else {
				report.Results = append(report.Results, "  reset time unknown")
			}
			for _, key := range []string{"points_left", "hours_to_reset", "sustainable_agents"} {
				v, ok := w[key].(float64)
				label := map[string]string{"points_left": "points left", "hours_to_reset": "hours to reset", "sustainable_agents": "sustainable concurrent agents"}[key]
				if ok {
					line += fmt.Sprintf(" · %.2f %s", v, label)
				} else {
					line += " · " + label + " unknown"
				}
			}
			report.Results = append(report.Results, line)
			if state, ok := w["confidence"].(string); ok {
				report.Results = append(report.Results, "  "+state)
			}
			if groups, ok := w["groups"].([]any); ok {
				for _, g := range groups {
					group, _ := g.(map[string]any)
					model, _ := group["model"].(string)
					rate, ok := group["measured_points_per_hour"].(float64)
					perAgent, agentOK := group["estimated_points_per_hour_per_running_run"].(float64)
					count, _ := group["running_agents"].(float64)
					if ok {
						line := fmt.Sprintf("  %s %.3f measured points/hour", model, rate)
						if agentOK {
							line += fmt.Sprintf("; %.3f estimated points/hour per running run across %d matching run(s)", perAgent, int(count))
						}
						report.Results = append(report.Results, line)
						if roles, ok := group["running_by_role"].(map[string]any); ok {
							for role, count := range roles {
								estimated, _ := group["estimated_points_per_hour_by_role"].(map[string]any)
								roleRate, rateOK := estimated[role].(float64)
								roleLine := fmt.Sprintf("    role %s: %.0f matching run(s)", role, count.(float64))
								if rateOK {
									roleLine += fmt.Sprintf("; estimated %.3f points/hour apportioned from measured model burn", roleRate)
								}
								report.Results = append(report.Results, roleLine)
							}
						}
					}
				}
			}
			if roles, ok := w["running_by_role"].(map[string]any); ok {
				for role, count := range roles {
					report.Results = append(report.Results, fmt.Sprintf("  role %s: %.0f running", role, count))
				}
			}
			if v, ok := w["unattributed_points"].(float64); ok {
				report.Results = append(report.Results, fmt.Sprintf("  %.1f points unattributed / off-disk", v))
			}
		}
		if len(windows) == 0 {
			report.Summary = append(report.Summary, "No provider quota samples; capacity is unknown.")
		}
		return report
	}))
}

func makeCapacityProto(forecast *measurepb.UsageForecastResponse, burners, runBurners *measurepb.UsageBurnersResponse, runs []*domainpb.Run) *structpb.Struct {
	now, _ := time.Parse(time.RFC3339, forecast.GetGeneratedAt())
	byScope := map[string]*measurepb.UsageWindowAttribution{}
	runFacts := map[string]runUsageFact{}
	for _, b := range burners.GetWindows() {
		byScope[scopeKey(b.GetProvider(), b.GetPool(), b.GetWindow())] = b
	}
	for _, window := range runBurners.GetWindows() {
		for _, burner := range window.GetBurners() {
			fact := runUsageFact{Model: strings.TrimSpace(burner.GetModel()), Runner: strings.TrimSpace(burner.GetRunner()), ProfileID: strings.TrimSpace(burner.GetProfileId())}
			for _, id := range burner.GetRunIds() {
				runFacts[id] = fact
			}
			if id := strings.TrimSpace(burner.GetKey()); id != "" {
				runFacts[id] = fact
			}
		}
	}
	rows := make([]any, 0, len(forecast.GetWindows()))
	for _, w := range forecast.GetWindows() {
		modelCounts, roleCounts, modelRoleCounts := censusCounts(runs, w.GetProvider(), runFacts)
		b := byScope[scopeKey(w.GetProvider(), w.GetPool(), w.GetWindow())]
		input := capacityWindow{Provider: w.GetProvider(), Pool: w.GetPool(), Window: w.GetWindow(), UsedPercent: w.GetUsedPercent(), HasPercent: w.GetHasPercent(), Samples: w.GetSampleCount(), ObservedAt: w.GetObservedAt(), ResetsAt: w.GetResetsAt()}
		if b != nil {
			input.Burners = b.GetBurners()
			input.Unattributed = b.GetUnattributedPoints()
			input.ObservedIncrease = b.GetObservedIncreasePoints()
			input.PeriodStart = b.GetPeriodStart()
			input.PeriodEnd = b.GetPeriodEnd()
		}
		input.RunningByModel, input.RunningByRole, input.RunningByModelRole = modelCounts, roleCounts, modelRoleCounts
		row := calculateCapacity(input, now)
		groups := make([]any, 0, len(row.Groups))
		for _, group := range row.Groups {
			entry := map[string]any{"model": group.Model, "running_agents": group.Running, "running_by_role": intCountsAny(group.RunningByRole), "estimated_points_per_hour_by_role": floatCountsAny(group.EstimatedRateByRole)}
			if group.HasMeasuredRate {
				entry["measured_points_per_hour"] = group.MeasuredRate
			} else {
				entry["measured_points_per_hour"] = nil
			}
			if group.Running > 0 && group.HasMeasuredRate {
				entry["estimated_points_per_hour_per_running_run"] = group.PerAgentRate
			} else {
				entry["estimated_points_per_hour_per_running_run"] = nil
			}
			groups = append(groups, entry)
		}
		roleCountsJSON := make(map[string]any, len(roleCounts))
		for role, count := range roleCounts {
			roleCountsJSON[role] = count
		}
		modelCountsJSON := make(map[string]any, len(modelCounts))
		totalRunning := 0
		for model, count := range modelCounts {
			modelCountsJSON[model] = count
			totalRunning += count
		}
		item := map[string]any{"title": row.Title, "provider": w.GetProvider(), "pool": w.GetPool(), "window": w.GetWindow(), "resets_at": row.ResetsAt, "confidence": row.State, "groups": groups, "running_agents": totalRunning, "running_by_model": modelCountsJSON, "running_by_role": roleCountsJSON, "running_by_model_role": nestedCountsAny(modelRoleCounts), "limitations": []any{"the run census is current; observed model spend is from the previous hour", "model usage is measured by model and apportioned across currently running matching-model runs and roles; it is not per-run billing", "active run hours are unavailable"}}
		if row.HasPoints {
			item["points_left"] = row.PointsLeft
		} else {
			item["points_left"] = nil
		}
		if row.HasHours {
			item["hours_to_reset"] = row.ResetHours
		} else {
			item["hours_to_reset"] = nil
		}
		if row.HasConcurrent {
			item["sustainable_agents"] = row.Concurrent
		} else {
			item["sustainable_agents"] = nil
		}
		if row.HasUnattributed {
			item["unattributed_points"] = row.Unattributed
		} else {
			item["unattributed_points"] = nil
		}
		rows = append(rows, item)
	}
	obj, _ := structpb.NewStruct(map[string]any{"generated_at": forecast.GetGeneratedAt(), "windows": rows, "limitations": []any{"sustainable_agents is a current-census estimate and assumes recent model usage is representative of the currently running model mix", "measured model burn is apportioned across currently running roles by matching-model run count; role-specific provider billing is not directly observed", "active run hours and per-run billing are unavailable"}})
	return obj
}

type runUsageFact struct {
	Model, Runner, ProfileID string
}

func censusCounts(runs []*domainpb.Run, provider string, facts map[string]runUsageFact) (map[string]int, map[string]int, map[string]map[string]int) {
	models, roles, modelRoles := map[string]int{}, map[string]int{}, map[string]map[string]int{}
	var expectedRunner domainpb.RunnerType
	switch provider {
	case "openai":
		expectedRunner = domainpb.RunnerType_RUNNER_TYPE_CODEX
	case "anthropic":
		expectedRunner = domainpb.RunnerType_RUNNER_TYPE_CLAUDE_CODE
	default:
		return models, roles, modelRoles
	}
	for _, run := range runs {
		if run == nil {
			continue
		}
		fact := facts[run.GetId()]
		runnerMatches := run.GetResolvedConfig().GetRunnerType() == expectedRunner
		if !runnerMatches {
			runner := strings.ToLower(strings.TrimSpace(fact.Runner + " " + run.GetHarnessKind() + " " + run.GetImportSourceHarness()))
			if provider == "openai" {
				runnerMatches = strings.Contains(runner, "codex")
			} else {
				runnerMatches = strings.Contains(runner, "claude")
			}
		}
		if !runnerMatches {
			continue
		}
		model := strings.TrimSpace(run.GetActualModel())
		if model == "" {
			model = strings.TrimSpace(run.GetResolvedConfig().GetModel())
		}
		if model == "" {
			model = strings.TrimSpace(run.GetRequestedModel())
		}
		if model == "" {
			model = fact.Model
		}
		if model == "" {
			model = "unknown"
		}
		models[model]++
		role := strings.TrimSpace(run.GetResolvedConfig().GetRoleRef())
		if role == "" {
			role = fact.ProfileID
		}
		if role == "" {
			role = "unknown"
		}
		roles[role]++
		if modelRoles[model] == nil {
			modelRoles[model] = map[string]int{}
		}
		modelRoles[model][role]++
	}
	return models, roles, modelRoles
}

func intCountsAny(values map[string]int) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func floatCountsAny(values map[string]float64) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func nestedCountsAny(values map[string]map[string]int) map[string]any {
	out := make(map[string]any, len(values))
	for model, roles := range values {
		out[model] = intCountsAny(roles)
	}
	return out
}

func scopeKey(provider, pool, window string) string {
	return provider + "\x00" + pool + "\x00" + window
}

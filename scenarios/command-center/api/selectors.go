package main

// A selector turns one upstream payload into one number. Every metric names
// its selector in source.select; an unknown selector is a registry defect and
// resolves to UNAVAILABLE with that reason, never to a guessed key.
type selector func(payload any) (float64, bool)

var selectors = map[string]selector{
	// vrooli core · GET /scenarios
	"active_scenarios":        scenarioCount(func(s scenarioRow) bool { return s.Status == "running" }),
	"scenario_health":         scenarioCount(func(s scenarioRow) bool { return s.Health == "healthy" }),
	"scenario_health_detail":  scenarioCount(func(s scenarioRow) bool { return s.Health == "degraded" || s.Health == "unhealthy" }),
	"total_scenarios":         scenarioCount(func(scenarioRow) bool { return true }),
	"scenario_completeness":   scenarioCount(func(s scenarioRow) bool { return s.Status == "running" && s.Health == "healthy" }),
	"scenario_ports":          scenarioCount(func(s scenarioRow) bool { return len(s.Ports) > 0 }),
	"composite_portfolio":     scenarioCount(func(s scenarioRow) bool { return s.Status == "running" }),
	"composite_system_health": scenarioCount(func(s scenarioRow) bool { return s.Health == "healthy" }),

	// swarm-manager · GET /api/v1/stats
	"swarm_throughput":     number("throughput", "completed_last_7_days"),
	"throughput_stats":     number("throughput", "created_last_7_days"),
	"swarm_active_agents":  number("agent", "total_executions"),
	"agent_stats":          number("agent", "success_rate"),
	"timing_stats":         number("agent", "avg_execution_minutes"),
	"blocking_stats":       number("blocking", "currently_blocked"),
	"dashboard_stats":      number("dashboard", "total_backlog_size"),
	"composite_throughput": number("dashboard", "total_completed_all_time"),
	"review_stats":         number("review", "rounds_completed"),
	"scope_stats":          scopeCount,

	// landing-page-business-suite · GET /api/v1/admin/dashboard/summary
	"visitors":           number("visitors"),
	"conversions":        number("conversions"),
	"cta_clicks":         number("cta_clicks"),
	"scroll_depth":       number("scroll_depth"),
	"variant_ab":         number("variant_ab"),
	"revenue_mrr":        number("revenue", "mrr"),
	"revenue_today":      number("revenue", "today"),
	"revenue_rollup":     number("revenue", "month"),
	"subscriber_counts":  number("subscriptions", "active"),
	"churn":              number("subscriptions", "churned_30d"),
	"credit_balances":    number("credits", "balance_total"),
	"credit_consumption": number("credits", "burned_per_day"),
	"usage_records":      number("usage", "records"),
	"composite_revenue":  number("revenue", "mrr"),
	"composite_reach":    number("visitors"),
}

type scenarioRow struct {
	Status string
	Health string
	Ports  map[string]any
}

func scenarioRows(payload any) ([]scenarioRow, bool) {
	root, ok := payload.(map[string]any)
	if !ok {
		return nil, false
	}
	items, ok := root["data"].([]any)
	if !ok {
		return nil, false
	}
	rows := make([]scenarioRow, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		row := scenarioRow{}
		row.Status, _ = m["status"].(string)
		row.Health, _ = m["health_status"].(string)
		row.Ports, _ = m["ports"].(map[string]any)
		rows = append(rows, row)
	}
	return rows, true
}

func scenarioCount(keep func(scenarioRow) bool) selector {
	return func(payload any) (float64, bool) {
		rows, ok := scenarioRows(payload)
		if !ok {
			return 0, false
		}
		n := 0
		for _, r := range rows {
			if keep(r) {
				n++
			}
		}
		return float64(n), true
	}
}

func walk(payload any, path ...string) (any, bool) {
	cur := payload
	for _, key := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = m[key]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

func number(path ...string) selector {
	return func(payload any) (float64, bool) {
		v, ok := walk(payload, path...)
		if !ok {
			return 0, false
		}
		n, ok := v.(float64)
		return n, ok
	}
}

func arrayLength(path ...string) selector {
	return func(payload any) (float64, bool) {
		v, ok := walk(payload, path...)
		if !ok {
			return 0, false
		}
		arr, ok := v.([]any)
		if !ok {
			return 0, false
		}
		return float64(len(arr)), true
	}
}

// scopeCount accepts the typed producer projection and the legacy REST
// envelope during mixed-version rollout. The typed contract owns the count;
// the legacy fallback derives it from the goals array.
func scopeCount(payload any) (float64, bool) {
	if value, ok := number("scope_stats")(payload); ok {
		return value, true
	}
	return arrayLength("scope", "goals")(payload)
}

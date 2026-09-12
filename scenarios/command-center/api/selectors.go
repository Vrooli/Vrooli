package main

import "fmt"

// A selector turns one upstream payload into one number. Every metric names
// its selector in source.select; an unknown selector is a registry defect and
// resolves to UNAVAILABLE with that reason, never to a guessed key.
type selector func(payload any) (float64, bool)
type panelSelector func(payload any) ([]PanelRow, bool)

type PanelRow struct {
	Key    string  `json:"key"`
	Label  string  `json:"label"`
	Value  float64 `json:"value"`
	Share  float64 `json:"share"`
	Detail string  `json:"detail,omitempty"`
	Ink    string  `json:"ink,omitempty"`
}

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

var panelSelectors = map[string]panelSelector{
	"traffic_countries":     trafficPanel,
	"traffic_referrers":     trafficPanel,
	"traffic_campaigns":     trafficPanel,
	"traffic_devices":       trafficPanel,
	"traffic_landing_paths": trafficPanel,
	"traffic_variants":      trafficPanel,
	"release_ladder":        releaseLadderPanel,
	"goal_progress":         goalProgressPanel,
	"deployment_readiness":  readinessPanel,
}

func trafficPanel(payload any) ([]PanelRow, bool) {
	root, ok := payload.(map[string]any)
	if !ok {
		return nil, false
	}
	raw, ok := root["rows"].([]any)
	if !ok {
		return nil, false
	}
	rows := make([]PanelRow, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		key, keyOK := m["key"].(string)
		label, labelOK := m["label"].(string)
		value, valueOK := m["value"].(float64)
		if !valueOK {
			value, valueOK = m["sessions"].(float64)
		}
		share, shareOK := m["share"].(float64)
		if !keyOK || !labelOK || !valueOK || !shareOK {
			return nil, false
		}
		rows = append(rows, PanelRow{Key: key, Label: label, Value: value, Share: share})
	}
	return rows, true
}

func releaseLadderPanel(payload any) ([]PanelRow, bool) {
	root, ok := payload.(map[string]any)
	if !ok {
		return nil, false
	}
	entries, ok := root["entries"].([]any)
	if !ok {
		return nil, false
	}
	rows := make([]PanelRow, 0, len(entries))
	share := 1 / float64(len(entries))
	for _, item := range entries {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		deliverable, ok := entry["deliverable"].(map[string]any)
		if !ok {
			return nil, false
		}
		id, idOK := deliverable["id"].(string)
		name, nameOK := deliverable["name"].(string)
		rank, rankOK := deliverable["releaseRank"].(float64)
		if !rankOK {
			rank, rankOK = deliverable["release_rank"].(float64)
		}
		status, _ := deliverable["status"].(string)
		if !idOK || !nameOK || !rankOK {
			return nil, false
		}
		rows = append(rows, PanelRow{Key: id, Label: name, Value: rank, Share: share, Detail: status})
	}
	return rows, len(rows) > 0
}

func goalProgressPanel(payload any) ([]PanelRow, bool) {
	root, ok := payload.(map[string]any)
	if !ok {
		return nil, false
	}
	items, ok := root["items"].([]any)
	if !ok {
		return nil, false
	}
	rows := make([]PanelRow, 0, len(items))
	for _, item := range items {
		wrapped, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		goal, ok := wrapped["goal"].(map[string]any)
		if !ok {
			return nil, false
		}
		scope, _ := wrapped["scope"].(map[string]any)
		name, nameOK := goal["name"].(string)
		title, titleOK := goal["title"].(string)
		completed, _ := scope["completed_count"].(float64)
		if completed == 0 {
			completed, _ = scope["completedCount"].(float64)
		}
		total, _ := scope["total"].(float64)
		if !nameOK || !titleOK || total <= 0 {
			continue
		}
		detail := ""
		if eta, ok := wrapped["eta"].(map[string]any); ok {
			if p50, ok := eta["p50_hours"].(float64); ok {
				if p80, ok := eta["p80_hours"].(float64); ok {
					detail = fmt.Sprintf("%.1fh–%.1fh ETA", p50, p80)
				}
			}
		}
		rows = append(rows, PanelRow{Key: name, Label: title, Value: completed, Share: completed / total, Detail: detail})
	}
	return rows, len(rows) > 0
}

func readinessPanel(payload any) ([]PanelRow, bool) {
	root, ok := payload.(map[string]any)
	if !ok {
		return nil, false
	}
	goal, _ := root["goal_exists"].(bool)
	if !goal {
		goal, _ = root["goalExists"].(bool)
	}
	value := float64(0)
	if goal {
		value = 1
	}
	label := "Readiness goal missing"
	if goal {
		label = "Readiness goal present"
	}
	return []PanelRow{{Key: "readiness", Label: label, Value: value, Share: 1, Detail: fmt.Sprintf("closed=%t", root["goal_closed"])}}, true
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

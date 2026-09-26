package main

import (
	"fmt"
	"strconv"
	"strings"
)

// A selector turns one upstream payload into one number. Every metric names
// its selector in source.select; an unknown selector is a registry defect and
// resolves to UNAVAILABLE with that reason, never to a guessed key.
type selector func(payload any) (float64, bool)
type panelSelector func(payload any) ([]PanelRow, bool)
type postureSelector func(payload any) (map[string]any, bool)

type PanelRow struct {
	Key         string  `json:"key"`
	Label       string  `json:"label"`
	Value       float64 `json:"value"`
	Share       float64 `json:"share"`
	Detail      string  `json:"detail,omitempty"`
	Ink         string  `json:"ink,omitempty"`
	Denominator float64 `json:"denominator,omitempty"`
	Rate        float64 `json:"rate,omitempty"`
	Probability float64 `json:"probability,omitempty"`
	Verdict     string  `json:"verdict,omitempty"`
	IsControl   bool    `json:"is_control,omitempty"`
	CTAClicks   float64 `json:"cta_clicks,omitempty"`
	CTATrials   float64 `json:"cta_trials,omitempty"`
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
	"visitors":              number("visitors"),
	"conversions":           number("conversions"),
	"cta_clicks":            number("cta_clicks"),
	"scroll_depth":          number("scroll_depth"),
	"variant_ab":            number("variant_ab"),
	"revenue_mrr":           number("revenue", "mrr"),
	"revenue_today":         number("revenue", "today"),
	"revenue_rollup":        number("revenue", "month"),
	"ai_cost_30d":           number("cost", "usd"),
	"credit_margin_30d":     creditMargin,
	"subscriber_counts":     number("subscriptions", "active"),
	"churn":                 number("subscriptions", "churned_30d"),
	"credit_balances":       number("credits", "balance_total"),
	"credit_consumption":    number("credits", "burned_per_day"),
	"usage_records":         number("usage", "records"),
	"composite_revenue":     number("revenue", "mrr"),
	"composite_reach":       number("visitors"),
	"digest_visitors":       digestFunnel("visitors"),
	"digest_paid":           digestFunnel("paid"),
	"digest_cta_clicks":     number("cta_clicks"),
	"digest_revenue":        number("funnel", "paid_revenue_minor"),
	"digest_credits":        digestNumber("credits", "credits_burned"),
	"digest_purchased":      digestNumber("credits", "credits_purchased"),
	"digest_operations":     digestNumber("credits", "operations"),
	"digest_consumers":      digestNumber("credits", "distinct_consumers"),
	"digest_signups":        digestNumber("growth", "signups"),
	"digest_waitlist":       digestNumber("growth", "waitlist_joins"),
	"digest_paid_subs":      digestNumber("growth", "new_paid_subscriptions"),
	"digest_trials":         digestNumber("growth", "trials_started"),
	"digest_retained":       digestNumber("retention", "still_active"),
	"digest_revenue_mrr":    number("revenue", "mrr"),
	"digest_revenue_today":  number("revenue", "today"),
	"digest_revenue_window": number("revenue", "month"),
	"usage_operations_30d":  digestNumber("usage", "records"),
}

var postureSelectors = map[string]postureSelector{
	"offer_posture": offerPosture,
}

var panelSelectors = map[string]panelSelector{
	"traffic_countries":     trafficPanel,
	"traffic_referrers":     trafficPanel,
	"traffic_campaigns":     trafficPanel,
	"traffic_devices":       trafficPanel,
	"traffic_landing_paths": trafficPanel,
	"traffic_variants":      trafficPanel,
	"digest_apps":           digestAppsPanel,
	"digest_app_updates":    digestAppUpdatesPanel,
	"digest_credit_app":     digestCreditRows("by_app"),
	"digest_credit_model":   digestCreditRows("by_model"),
	"revenue_by_line":       revenueLinePanel,
	"digest_experiment":     digestExperimentPanel,
	"digest_funnel":         digestFunnelPanel,
	"goal_progress":         goalProgressPanel,
	"deployment_readiness":  readinessPanel,
}

func creditMargin(payload any) (float64, bool) {
	root, ok := payload.(map[string]any)
	if !ok {
		return 0, false
	}
	lines, ok := root["revenue_by_line"].([]any)
	if !ok {
		return 0, false
	}
	var creditRevenue float64
	for _, raw := range lines {
		line, ok := raw.(map[string]any)
		if !ok {
			return 0, false
		}
		key, _ := line["key"].(string)
		if key == "credit_top_up" {
			value, ok := asFloat(line["amount_minor"])
			if !ok {
				return 0, false
			}
			creditRevenue = value / 100
		}
	}
	cost, ok := walk(root, "cost", "usd")
	if !ok {
		return 0, false
	}
	costUSD, ok := asFloat(cost)
	return creditRevenue - costUSD, ok
}

func revenueLinePanel(payload any) ([]PanelRow, bool) {
	root, ok := payload.(map[string]any)
	if !ok {
		return nil, false
	}
	items, ok := root["revenue_by_line"].([]any)
	if !ok {
		return []PanelRow{}, true
	}
	rows := make([]PanelRow, 0, len(items))
	for _, raw := range items {
		line, ok := raw.(map[string]any)
		if !ok {
			return nil, false
		}
		key, keyOK := line["key"].(string)
		label, labelOK := line["label"].(string)
		amount, amountOK := asFloat(line["amount_minor"])
		if !keyOK || !labelOK || !amountOK || key == "" {
			return nil, false
		}
		detail := ""
		if transactions, present := asFloat(line["transactions"]); present {
			detail = fmt.Sprintf("%d transactions", int64(transactions))
		}
		rows = append(rows, PanelRow{Key: key, Label: label, Value: amount / 100, Detail: detail})
	}
	return rows, true
}

func offerPosture(payload any) (map[string]any, bool) {
	root, ok := payload.(map[string]any)
	if !ok {
		return nil, false
	}
	position, ok := root["position"].(map[string]any)
	if !ok {
		return nil, false
	}
	field := func(m map[string]any, snake, camel string) (any, bool) {
		if value, exists := m[camel]; exists {
			return value, true
		}
		value, exists := m[snake]
		return value, exists
	}
	for _, keys := range [][2]string{{"cash_minor", "cashMinor"}, {"burn_minor", "burnMinor"}, {"revenue_minor", "revenueMinor"}, {"runway_months", "runwayMonths"}, {"runway_available", "runwayAvailable"}} {
		if _, exists := field(position, keys[0], keys[1]); !exists {
			return nil, false
		}
	}
	cash, _ := field(position, "cash_minor", "cashMinor")
	burn, _ := field(position, "burn_minor", "burnMinor")
	revenue, _ := field(position, "revenue_minor", "revenueMinor")
	runway, _ := field(position, "runway_months", "runwayMonths")
	available, _ := field(position, "runway_available", "runwayAvailable")
	gap, _ := field(root, "default_alive_gap", "defaultAliveGap")
	source, _ := field(root, "posture_source", "postureSource")
	age, _ := field(root, "posture_age_seconds", "postureAgeSeconds")
	return map[string]any{
		"cashMinor": cash, "burnMinor": burn, "revenueMinor": revenue,
		"runwayMonths": runway, "runwayAvailable": available,
		"gap": gap, "source": source, "ageSeconds": age, "goals": root["goals"],
	}, true
}

func digestAppsPanel(payload any) ([]PanelRow, bool) {
	root, ok := payload.(map[string]any)
	if !ok {
		return nil, false
	}
	items, ok := root["apps"].([]any)
	if !ok {
		return []PanelRow{}, true
	}
	total := 0.0
	rows := make([]PanelRow, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			return nil, false
		}
		value, _ := asFloat(item["downloads"])
		key, _ := item["app_key"].(string)
		label, _ := item["app_name"].(string)
		if key == "" {
			key = label
		}
		if label == "" {
			label = key
		}
		if key == "" {
			return nil, false
		}
		rows = append(rows, PanelRow{Key: key, Label: label, Value: value})
		total += value
	}
	for i := range rows {
		if total > 0 {
			rows[i].Share = rows[i].Value / total
		}
	}
	return rows, true
}

func digestAppUpdatesPanel(payload any) ([]PanelRow, bool) {
	root, ok := payload.(map[string]any)
	if !ok {
		return nil, false
	}
	items, ok := root["apps"].([]any)
	if !ok {
		return []PanelRow{}, true
	}
	rows := make([]PanelRow, 0, len(items))
	total := 0.0
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			return nil, false
		}
		value, _ := asFloat(item["update_checks"])
		key, _ := item["app_key"].(string)
		label, _ := item["app_name"].(string)
		if key == "" {
			key = label
		}
		if label == "" {
			label = key
		}
		if key == "" {
			return nil, false
		}
		rows = append(rows, PanelRow{Key: key, Label: label, Value: value})
		total += value
	}
	for i := range rows {
		if total > 0 {
			rows[i].Share = rows[i].Value / total
		}
	}
	return rows, true
}

func digestCreditRows(field string) panelSelector {
	return func(payload any) ([]PanelRow, bool) {
		root, ok := payload.(map[string]any)
		if !ok {
			return nil, false
		}
		credits, ok := root["credits"].(map[string]any)
		if !ok {
			return nil, false
		}
		items, ok := credits[field].([]any)
		if !ok {
			return []PanelRow{}, true
		}
		rows := make([]PanelRow, 0, len(items))
		total := 0.0
		for _, raw := range items {
			item, ok := raw.(map[string]any)
			if !ok {
				return nil, false
			}
			value, _ := asFloat(item["credits"])
			key, _ := item["key"].(string)
			label, _ := item["label"].(string)
			if key == "" {
				return nil, false
			}
			if label == "" {
				label = key
			}
			rows = append(rows, PanelRow{Key: key, Label: label, Value: value})
			total += value
		}
		for i := range rows {
			if total > 0 {
				rows[i].Share = rows[i].Value / total
			}
		}
		return rows, true
	}
}

func digestExperimentPanel(payload any) ([]PanelRow, bool) {
	root, ok := payload.(map[string]any)
	if !ok {
		return nil, false
	}
	experiment, ok := root["experiment"].(map[string]any)
	if !ok {
		return nil, false
	}
	items, ok := experiment["arms"].([]any)
	if !ok {
		return nil, false
	}
	total := 0.0
	rows := make([]PanelRow, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			return nil, false
		}
		paid, ok := item["paid"].(map[string]any)
		if !ok {
			return nil, false
		}
		value, _ := asFloat(paid["successes"])
		trials, _ := asFloat(paid["trials"])
		key, _ := item["variant_slug"].(string)
		label, _ := item["variant_name"].(string)
		if key == "" {
			return nil, false
		}
		if label == "" {
			label = key
		}
		cta, _ := item["ctaClick"].(map[string]any)
		if cta == nil {
			cta, _ = item["cta_click"].(map[string]any)
		}
		ctaSuccesses, _ := asFloat(cta["successes"])
		ctaTrials, _ := asFloat(cta["trials"])
		probability, _ := asFloat(paid["probabilityBeatsControl"])
		if probability == 0 {
			probability, _ = asFloat(paid["probability_beats_control"])
		}
		verdict, _ := paid["verdict"].(string)
		isControl := boolValue(item["isControl"])
		if !isControl {
			isControl = boolValue(item["is_control"])
		}
		rows = append(rows, PanelRow{Key: key, Label: label, Value: value, Share: value / maxFloat(1, trials), Detail: fmt.Sprintf("trials=%.0f", trials), Denominator: trials, Rate: value / maxFloat(1, trials), Probability: probability, Verdict: verdict, IsControl: isControl, CTAClicks: ctaSuccesses, CTATrials: ctaTrials})
		total += value
	}
	for i := range rows {
		if total > 0 {
			rows[i].Share = rows[i].Value / total
		}
	}
	return rows, true
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
func boolValue(value any) bool { result, _ := value.(bool); return result }

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
		value, valueOK := asFloat(m["value"])
		if !valueOK {
			value, valueOK = asFloat(m["sessions"])
		}
		if !valueOK {
			value, valueOK = asFloat(m["visitors"])
		}
		share, shareOK := asFloat(m["share"])
		if !keyOK || !labelOK || !valueOK || !shareOK {
			return nil, false
		}
		if strings.EqualFold(key, "unknown") {
			label = "Unknown"
		}
		rows = append(rows, PanelRow{Key: key, Label: label, Value: value, Share: share})
	}
	if other, ok := asFloat(root["otherVisitors"]); ok && other > 0 {
		total, _ := asFloat(root["totalVisitors"])
		rows = append(rows, PanelRow{Key: "unknown", Label: "Unknown", Value: other, Share: other / maxFloat(1, total)})
	}
	return rows, true
}

func digestFunnel(key string) selector {
	return func(payload any) (float64, bool) {
		root, ok := payload.(map[string]any)
		if !ok {
			return 0, false
		}
		funnel, ok := root["funnel"].(map[string]any)
		if !ok {
			return 0, false
		}
		steps, ok := funnel["steps"].([]any)
		if !ok {
			return 0, false
		}
		for _, raw := range steps {
			step, ok := raw.(map[string]any)
			if ok && step["key"] == key {
				value, ok := asFloat(step["value"])
				return value, ok
			}
		}
		return 0, false
	}
}

func digestNumber(path ...string) selector {
	return func(payload any) (float64, bool) {
		value, ok := walk(payload, path...)
		if ok {
			return asFloat(value)
		}
		// protojson omits zero-valued scalar fields. Once the declared parent
		// object exists, an absent leaf is a measured zero, not malformed data.
		parent, ok := payload.(map[string]any)
		if !ok || len(path) < 2 {
			return 0, false
		}
		for _, key := range path[:len(path)-1] {
			parent, ok = parent[key].(map[string]any)
			if !ok {
				return 0, false
			}
		}
		return 0, true
	}
}

func digestFunnelPanel(payload any) ([]PanelRow, bool) {
	root, ok := payload.(map[string]any)
	if !ok {
		return nil, false
	}
	funnel, ok := root["funnel"].(map[string]any)
	if !ok {
		return nil, false
	}
	items, ok := funnel["steps"].([]any)
	if !ok {
		return nil, false
	}
	visitors := 0.0
	rows := make([]PanelRow, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			return nil, false
		}
		value, _ := asFloat(item["value"])
		key, _ := item["key"].(string)
		label, _ := item["label"].(string)
		if key == "" || label == "" {
			return nil, false
		}
		if visitors == 0 {
			visitors = value
		}
		denominator, _ := asFloat(item["denominator"])
		rate := 0.0
		if denominator > 0 {
			rate = value / denominator
		}
		rows = append(rows, PanelRow{Key: key, Label: label, Value: value, Denominator: denominator, Rate: rate})
	}
	for i := range rows {
		if visitors > 0 {
			rows[i].Share = rows[i].Value / visitors
		}
	}
	return rows, true
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
		return asFloat(v)
	}
}

func asFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		n, err := strconv.ParseFloat(v, 64)
		return n, err == nil
	default:
		return 0, false
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

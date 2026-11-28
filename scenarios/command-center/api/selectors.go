package main

import (
	"fmt"
	"math"
	"sort"
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
	"total_scenarios":         scenarioInventoryTotal,
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
	"median_execution":     number("timing", "median_execution_minutes"),
	"blocking_stats":       number("blocking", "currently_blocked"),
	"dashboard_stats":      number("dashboard", "total_backlog_size"),
	"composite_throughput": number("dashboard", "total_completed_all_time"),
	"review_stats":         number("review", "rounds_completed"),
	"scope_stats":          scopeCount,

	// landing-page-business-suite · GET /api/v1/admin/dashboard/summary
	"visitors":              analyticsSummaryNumber("totalVisitors"),
	"conversions":           analyticsVariantTotal("conversions"),
	"cta_clicks":            analyticsVariantTotal("ctaClicks"),
	"scroll_depth":          number("scroll_depth"),
	"variant_ab":            number("variant_ab"),
	"revenue_mrr":           scaledNumber(0.01, "mrr_minor"),
	"revenue_today":         scaledNumber(0.01, "revenue_today_minor"),
	"revenue_rollup":        scaledNumber(0.01, "revenue_window_minor"),
	"ai_cost_30d":           scaledNumber(0.000001, "cost_micros"),
	"credit_margin_30d":     creditMargin,
	"subscriber_counts":     number("active_subscriptions"),
	"churn":                 number("subscriptions_churned_window"),
	"credit_balances":       number("credit_balance_total"),
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
	"usage_operations_30d":  number("usage_records_window"),
}

// analyticsSummaryNumber reads the lower-camel JSON shape emitted by the
// LPBS AnalyticsSummary protobuf response.
func analyticsSummaryNumber(field string) selector {
	return func(payload any) (float64, bool) {
		root, ok := payload.(map[string]any)
		if !ok {
			return 0, false
		}
		value, exists := root[field]
		if !exists {
			return 0, false
		}
		return asFloat(value)
	}
}

func analyticsVariantTotal(field string) selector {
	return func(payload any) (float64, bool) {
		root, ok := payload.(map[string]any)
		if !ok {
			return 0, false
		}
		variants, ok := root["variantStats"].([]any)
		if !ok {
			return 0, false
		}
		var total float64
		for _, raw := range variants {
			variant, ok := raw.(map[string]any)
			if !ok {
				return 0, false
			}
			value, ok := asFloat(variant[field])
			if !ok {
				return 0, false
			}
			total += value
		}
		return total, true
	}
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
	costUSD, ok := scaledNumber(0.000001, "cost_micros")(payload)
	if !ok {
		cost, exists := walk(root, "cost", "usd")
		if !exists {
			return 0, false
		}
		costUSD, ok = asFloat(cost)
	}
	return creditRevenue - costUSD, ok
}

func revenueLinePanel(payload any) ([]PanelRow, bool) {
	root, ok := payload.(map[string]any)
	if !ok {
		return nil, false
	}
	items, ok := root["revenue_by_line"].([]any)
	if !ok {
		if rows, exists := root["revenue_by_line"]; exists && rows == nil {
			return []PanelRow{}, true
		}
		return nil, false
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
	if partial, exists := position["partial"]; exists {
		value, valid := partial.(bool)
		if !valid || value {
			return nil, false
		}
	}
	for _, key := range []string{"cashMinor", "burnMinor", "revenueMinor", "runwayMonths"} {
		if _, valid := asFloat(position[key]); !valid {
			return nil, false
		}
	}
	if _, valid := position["runwayAvailable"].(bool); !valid {
		return nil, false
	}
	currency, currencyOK := position["currency"].(string)
	source, sourceOK := root["postureSource"].(string)
	if !currencyOK || strings.TrimSpace(currency) == "" || !sourceOK || strings.TrimSpace(source) == "" {
		return nil, false
	}
	gap, _ := root["defaultAliveGap"].(string)
	age := float64(0)
	if rawAge, ageExists := root["postureAgeSeconds"]; ageExists {
		ageValue, valid := asFloat(rawAge)
		if !valid || ageValue < 0 || math.Trunc(ageValue) != ageValue {
			return nil, false
		}
		age = ageValue
	}
	cash, _ := asFloat(position["cashMinor"])
	burn, _ := asFloat(position["burnMinor"])
	revenue, _ := asFloat(position["revenueMinor"])
	runway, _ := asFloat(position["runwayMonths"])
	return map[string]any{
		"cashMinor": cash, "burnMinor": burn, "revenueMinor": revenue,
		"runwayMonths": runway, "runwayAvailable": position["runwayAvailable"], "currency": currency,
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
		if arms, exists := experiment["arms"]; !exists || arms == nil {
			return []PanelRow{}, true
		}
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
	type candidate struct {
		row      PanelRow
		priority float64
	}
	rows := make([]candidate, 0, len(items))
	for _, item := range items {
		wrapped, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		goal, ok := wrapped["goal"].(map[string]any)
		if !ok {
			return nil, false
		}
		status, statusOK := goal["status"].(string)
		if !statusOK {
			return nil, false
		}
		switch status {
		case "achieved", "archived":
			continue
		case "active":
		default:
			return nil, false
		}
		scope, scopeOK := wrapped["scope"].(map[string]any)
		name, nameOK := goal["name"].(string)
		title, titleOK := goal["title"].(string)
		completed, completedOK := scope["completed_count"].(float64)
		total, totalOK := scope["total"].(float64)
		if !scopeOK || !nameOK || !titleOK || name == "" || title == "" || !completedOK || !totalOK || total < 0 || completed < 0 || completed > total || math.Trunc(total) != total || math.Trunc(completed) != completed {
			return nil, false
		}
		priority, priorityOK := goal["priority"].(float64)
		if !priorityOK || priority < 0 || priority > 10 || math.Trunc(priority) != priority {
			return nil, false
		}
		detail := ""
		share := float64(0)
		if total == 0 {
			detail = "No scoped work"
		} else {
			share = completed / total
		}
		if total > 0 {
			eta, ok := wrapped["eta"].(map[string]any)
			if !ok {
				eta = nil
			}
			if p50, ok := eta["p50_hours"].(float64); ok {
				if p80, ok := eta["p80_hours"].(float64); ok {
					detail = fmt.Sprintf("%.1fh–%.1fh ETA", p50, p80)
				}
			}
		}
		rows = append(rows, candidate{
			row:      PanelRow{Key: name, Label: title, Value: completed, Share: share, Detail: detail},
			priority: priority,
		})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].priority == rows[j].priority {
			return rows[i].row.Key < rows[j].row.Key
		}
		return rows[i].priority < rows[j].priority
	})
	if len(rows) > 6 {
		rows = rows[:6]
	}
	result := make([]PanelRow, len(rows))
	for i, candidate := range rows {
		result[i] = candidate.row
	}
	return result, true
}

func readinessPanel(payload any) ([]PanelRow, bool) {
	root, ok := payload.(map[string]any)
	if !ok {
		return nil, false
	}
	goal, goalOK := root["goal_exists"].(bool)
	closed, closedOK := root["goal_closed"].(bool)
	if !goalOK || !closedOK {
		return nil, false
	}
	value := float64(0)
	if goal {
		value = 1
	}
	label := "Readiness goal missing"
	if goal {
		label = "Readiness goal present"
	}
	return []PanelRow{{Key: "readiness", Label: label, Value: value, Share: 1, Detail: fmt.Sprintf("closed=%t", closed)}}, true
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
	if !scenarioDiscoveryComplete(payload) {
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
			return nil, false
		}
		status, statusOK := m["status"].(string)
		ports, portsOK := m["ports"].(map[string]any)
		if !statusOK || (status != "running" && status != "stopped") || !portsOK {
			return nil, false
		}
		for key, rawPort := range ports {
			port, numberOK := rawPort.(float64)
			if key == "" || !numberOK || port <= 0 || math.Trunc(port) != port {
				return nil, false
			}
		}
		healthValue, healthPresent := m["health_status"]
		if !healthPresent {
			return nil, false
		}
		health := ""
		if status == "stopped" {
			if healthValue != nil {
				return nil, false
			}
		} else {
			var healthOK bool
			health, healthOK = healthValue.(string)
			if !healthOK || (health != "running" && health != "healthy" && health != "degraded" && health != "unhealthy") {
				return nil, false
			}
		}
		row := scenarioRow{Status: status, Health: health, Ports: ports}
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

func scenarioInventoryTotal(payload any) (float64, bool) {
	rows, ok := scenarioRows(payload)
	if !ok {
		return 0, false
	}
	total, ok := integerNumber("total_scenarios")(payload)
	return total, ok && total == float64(len(rows))
}

func scenarioDiscoveryComplete(payload any) bool {
	root, ok := payload.(map[string]any)
	if !ok {
		return false
	}
	raw, exists := root["discovery_failures"]
	if !exists {
		return true
	}
	failures, ok := raw.([]any)
	return ok && len(failures) == 0
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

func integerNumber(path ...string) selector {
	base := number(path...)
	return func(payload any) (float64, bool) {
		value, ok := base(payload)
		return value, ok && math.Trunc(value) == value
	}
}

func scaledNumber(scale float64, path ...string) selector {
	base := number(path...)
	return func(payload any) (float64, bool) {
		value, ok := base(payload)
		if !ok {
			return 0, false
		}
		return value * scale, true
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

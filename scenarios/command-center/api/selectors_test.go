package main

import (
	"fmt"
	"math"
	"testing"
)

func TestLedgerFinancialSelectorsUseMeasuredProducerFields(t *testing.T) {
	payload := map[string]any{
		"cost": map[string]any{"usd": 7.5},
		"revenue_by_line": []any{
			map[string]any{"key": "credit_top_up", "label": "Credit top-up", "amount_minor": float64(2500), "transactions": float64(4)},
			map[string]any{"key": "subscription", "label": "Subscription", "amount_minor": float64(1000), "transactions": float64(2)},
		},
		"position": map[string]any{
			"cashMinor":       float64(5000),
			"burnMinor":       float64(900),
			"revenueMinor":    float64(1200),
			"runwayMonths":    5.5,
			"runwayAvailable": true,
			"currency":        "USD",
		},
		"defaultAliveGap":   "threshold met",
		"postureSource":     "money-ledger.position",
		"postureAgeSeconds": float64(12),
	}

	margin, ok := creditMargin(payload)
	if !ok || math.Abs(margin-17.5) > 1e-9 {
		t.Fatalf("credit margin = %v, ok=%v; want 17.5", margin, ok)
	}
	rows, ok := revenueLinePanel(payload)
	if !ok || len(rows) != 2 || rows[0].Value != 25 || rows[0].Detail != "4 transactions" || rows[1].Value != 10 || rows[1].Detail != "2 transactions" {
		t.Fatalf("revenue lines = %#v, ok=%v", rows, ok)
	}
	posture, ok := offerPosture(payload)
	if !ok || posture["cashMinor"] != float64(5000) || posture["gap"] != "threshold met" || posture["source"] != "money-ledger.position" {
		t.Fatalf("posture = %#v, ok=%v", posture, ok)
	}
}

func TestRevenueSummarySelectorsHonorProducerUnits(t *testing.T) {
	payload := map[string]any{
		"mrr_minor": float64(240050), "revenue_today_minor": float64(1299), "revenue_window_minor": float64(9999),
		"active_subscriptions": float64(2), "subscriptions_churned_window": float64(1),
		"credit_balance_total": float64(500), "cost_micros": float64(7500000), "usage_records_window": float64(8),
		"revenue_by_line": []any{
			map[string]any{"key": "credit_top_up", "label": "Credit top-up", "amount_minor": float64(2500), "transactions": float64(4)},
			map[string]any{"key": "subscription", "label": "Subscription", "amount_minor": float64(1000), "transactions": float64(2)},
		},
	}
	for _, tc := range []struct {
		selector string
		want     float64
	}{
		{"revenue_mrr", 2400.5}, {"revenue_today", 12.99}, {"revenue_rollup", 99.99},
		{"subscriber_counts", 2}, {"churn", 1}, {"credit_balances", 500}, {"ai_cost_30d", 7.5}, {"usage_operations_30d", 8},
	} {
		got, ok := selectors[tc.selector](payload)
		if !ok || math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("%s = %v, ok=%v; want %v", tc.selector, got, ok, tc.want)
		}
	}
	margin, ok := creditMargin(payload)
	if !ok || math.Abs(margin-17.5) > 1e-9 {
		t.Errorf("credit margin = %v, ok=%v; want 17.5", margin, ok)
	}
}

func TestExecutionTimingSelectorsKeepMeanAndMedianDistinct(t *testing.T) {
	payload := map[string]any{
		"agent":  map[string]any{"avg_execution_minutes": 5194.663306311462},
		"timing": map[string]any{"median_execution_minutes": 36.952940601899996, "execution_duration_samples": float64(51)},
	}
	mean, meanOK := selectors["timing_stats"](payload)
	median, medianOK := selectors["median_execution"](payload)
	if !meanOK || !medianOK || math.Abs(mean-5194.663306311462) > 1e-9 || math.Abs(median-36.952940601899996) > 1e-9 {
		t.Fatalf("mean/median = %v/%v (ok %v/%v); want producer values kept distinct", mean, median, meanOK, medianOK)
	}
	if _, ok := selectors["median_execution"](map[string]any{"timing": map[string]any{}}); ok {
		t.Fatal("missing producer median must remain unavailable rather than becoming zero")
	}
}

func TestTotalScenariosRequiresProducerCompleteAggregate(t *testing.T) {
	selector := selectors["total_scenarios"]
	for _, tc := range []struct {
		name    string
		payload map[string]any
		want    float64
		ok      bool
	}{
		{name: "complete inventory", payload: map[string]any{"total_scenarios": float64(2), "data": []any{map[string]any{"status": "stopped", "health_status": nil, "ports": map[string]any{}}, map[string]any{"status": "running", "health_status": "healthy", "ports": map[string]any{}}}}, want: 2, ok: true},
		{name: "complete empty inventory", payload: map[string]any{"total_scenarios": float64(0), "data": []any{}}, want: 0, ok: true},
		{name: "partial inventory cannot publish aggregate", payload: map[string]any{"total_scenarios": float64(1), "discovery_failures": []any{map[string]any{"name": "broken"}}, "data": []any{map[string]any{"name": "one"}}}, ok: false},
		{name: "fractional inventory is malformed", payload: map[string]any{"total_scenarios": float64(1.5)}, ok: false},
		{name: "aggregate disagrees with rows", payload: map[string]any{"total_scenarios": float64(2), "data": []any{map[string]any{"status": "stopped", "health_status": nil, "ports": map[string]any{}}}}, ok: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := selector(tc.payload)
			if ok != tc.ok || (ok && got != tc.want) {
				t.Fatalf("total_scenarios = %v, ok=%t; want %v, ok=%t", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestScenarioCountsRejectPartialDiscovery(t *testing.T) {
	payload := map[string]any{
		"data":               []any{map[string]any{"status": "running", "health_status": "healthy"}},
		"discovery_failures": []any{map[string]any{"name": "broken"}},
	}
	for _, id := range []string{"active_scenarios", "scenario_health", "scenario_completeness"} {
		if value, ok := selectors[id](payload); ok {
			t.Errorf("%s = %v, ok=true for partial discovery; want unusable", id, value)
		}
	}
}

func TestScenarioInventoryCountsRequireNativeProducerRowShape(t *testing.T) {
	valid := map[string]any{
		"data": []any{
			map[string]any{"status": "running", "health_status": "healthy", "ports": map[string]any{"API_PORT": float64(8092)}},
			map[string]any{"status": "running", "health_status": "running", "ports": map[string]any{}},
			map[string]any{"status": "stopped", "health_status": nil, "ports": map[string]any{}},
		},
	}
	for _, tc := range []struct {
		selector string
		want     float64
	}{
		{selector: "active_scenarios", want: 2},
		{selector: "scenario_health", want: 1},
		{selector: "scenario_health_detail", want: 0},
		{selector: "scenario_completeness", want: 1},
		{selector: "scenario_ports", want: 1},
		{selector: "total_scenarios", want: 3},
	} {
		t.Run(tc.selector, func(t *testing.T) {
			payload := map[string]any{}
			for key, value := range valid {
				payload[key] = value
			}
			payload["total_scenarios"] = tc.want
			got, ok := selectors[tc.selector](payload)
			if !ok || got != tc.want {
				t.Fatalf("%s = %v, ok=%t; want %v from native /scenarios row shape", tc.selector, got, ok, tc.want)
			}
		})
	}

	for _, tc := range []struct {
		name string
		row  map[string]any
	}{
		{name: "missing status", row: map[string]any{"health_status": "healthy", "ports": map[string]any{}}},
		{name: "unknown status", row: map[string]any{"status": "start-failed", "health_status": "healthy", "ports": map[string]any{}}},
		{name: "missing ports", row: map[string]any{"status": "running", "health_status": "healthy"}},
		{name: "malformed port", row: map[string]any{"status": "running", "health_status": "healthy", "ports": map[string]any{"API_PORT": "8092"}}},
		{name: "missing health", row: map[string]any{"status": "running", "ports": map[string]any{}}},
		{name: "stopped with health", row: map[string]any{"status": "stopped", "health_status": "healthy", "ports": map[string]any{}}},
		{name: "unknown health", row: map[string]any{"status": "running", "health_status": "warming-up", "ports": map[string]any{}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := map[string]any{"data": []any{tc.row}}
			if _, ok := selectors["active_scenarios"](payload); ok {
				t.Fatalf("malformed producer row was accepted: %#v", tc.row)
			}
		})
	}
}

func TestEmptyProducerCollectionsAreDistinctFromMissingFields(t *testing.T) {
	rows, ok := revenueLinePanel(map[string]any{"revenue_by_line": nil})
	if !ok || rows == nil || len(rows) != 0 {
		t.Fatalf("explicit empty revenue lines = %#v, ok=%v; want measured empty", rows, ok)
	}
	if _, ok := revenueLinePanel(map[string]any{}); ok {
		t.Fatal("missing revenue line field must not be read as a valid zero")
	}
	arms, ok := digestExperimentPanel(map[string]any{"experiment": map[string]any{}})
	if !ok || arms == nil || len(arms) != 0 {
		t.Fatalf("empty experiment arms = %#v, ok=%v; want measured empty", arms, ok)
	}
}

func TestGoalProgressSelectsSixMostUrgentActiveGoals(t *testing.T) {
	items := []any{
		map[string]any{
			"goal":  map[string]any{"name": "archived", "title": "Archived", "status": "archived", "priority": float64(0)},
			"scope": map[string]any{"completed_count": float64(4), "total": float64(4)},
		},
	}
	for priority := 7; priority >= 0; priority-- {
		name := fmt.Sprintf("goal-%d", priority)
		items = append(items, map[string]any{
			"goal":  map[string]any{"name": name, "title": name, "status": "active", "priority": float64(priority)},
			"scope": map[string]any{"completed_count": float64(3), "total": float64(10)},
			"eta":   map[string]any{"p50_hours": float64(4), "p80_hours": float64(9)},
		})
	}
	rows, ok := goalProgressPanel(map[string]any{"items": items})
	if !ok || len(rows) != 6 {
		t.Fatalf("goal progress rows = %d, ok=%v; want six rows", len(rows), ok)
	}
	for i, row := range rows {
		want := fmt.Sprintf("goal-%d", i)
		if row.Key != want || row.Share != .3 {
			t.Errorf("row[%d]=%+v; want %s with independent 30%% progress", i, row, want)
		}
	}
	if rows[0].Detail != "4.0h–9.0h ETA" {
		t.Errorf("row ETA=%q", rows[0].Detail)
	}
}

func TestGoalProgressEmptyCollectionIsMeasuredEmpty(t *testing.T) {
	rows, ok := goalProgressPanel(map[string]any{"items": []any{}})
	if !ok || rows == nil || len(rows) != 0 {
		t.Fatalf("empty goal list = %#v, ok=%v; want measured empty", rows, ok)
	}
	if _, ok := goalProgressPanel(map[string]any{}); ok {
		t.Fatal("missing items must not be accepted as empty goals")
	}
}

func TestGoalProgressKeepsValidEmptyScopeAndUsesProducerPriority(t *testing.T) {
	rows, ok := goalProgressPanel(map[string]any{"items": []any{
		map[string]any{
			"goal":  map[string]any{"name": "unscoped", "title": "Unscoped goal", "status": "active", "priority": float64(0)},
			"scope": map[string]any{"completed_count": float64(0), "total": float64(0)},
		},
		map[string]any{
			"goal":  map[string]any{"name": "scoped", "title": "Scoped goal", "status": "active", "priority": float64(8)},
			"scope": map[string]any{"completed_count": float64(1), "total": float64(4)},
		},
	}})
	if !ok || len(rows) != 2 {
		t.Fatalf("goal rows=%+v ok=%t, want both active goals", rows, ok)
	}
	if rows[0].Key != "unscoped" || rows[0].Share != 0 || rows[0].Detail != "No scoped work" {
		t.Fatalf("unscoped priority-zero row=%+v, want explicit empty-work state first", rows[0])
	}
	if rows[1].Key != "scoped" || rows[1].Share != .25 {
		t.Fatalf("scoped progress row=%+v, want 25%%", rows[1])
	}
}

func TestGoalProgressRejectsInvalidEmptyScopeAndSupersededFieldAlias(t *testing.T) {
	for _, tc := range []struct {
		name string
		goal map[string]any
	}{
		{
			name: "completed exceeds empty scope",
			goal: map[string]any{
				"goal":  map[string]any{"name": "bad", "title": "Bad", "status": "active", "priority": float64(1)},
				"scope": map[string]any{"completed_count": float64(1), "total": float64(0)},
			},
		},
		{
			name: "camel case alias absent from producer JSON",
			goal: map[string]any{
				"goal":  map[string]any{"name": "bad", "title": "Bad", "status": "active", "priority": float64(1)},
				"scope": map[string]any{"completedCount": float64(1), "total": float64(2)},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok := goalProgressPanel(map[string]any{"items": []any{tc.goal}}); ok {
				t.Fatalf("invalid producer item accepted: %#v", tc.goal)
			}
		})
	}
}

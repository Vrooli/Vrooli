package main

import "testing"

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
		},
		"defaultAliveGap":   "threshold met",
		"postureSource":     "money-ledger.position",
		"postureAgeSeconds": float64(12),
	}

	margin, ok := creditMargin(payload)
	if !ok || margin != 17.5 {
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

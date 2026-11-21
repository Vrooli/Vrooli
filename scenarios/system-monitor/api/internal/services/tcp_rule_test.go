package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
)

// Ported from the dashboard's former client-side connection rule
// (ui/src/features/metrics/connectionAlerts.test.ts), now server-owned.
func TestSustainedConnectionGrowth(t *testing.T) {
	cases := []struct {
		name   string
		series []float64
		want   bool
	}{
		{"sustained climb well before the level threshold", []float64{600, 900, 1400, 2100, 3000, 4200}, true},
		{"growth below the floor is ordinary", []float64{5, 10, 18, 25, 32, 40}, false},
		{"recovered spike", []float64{600, 900, 4000, 3000, 1200, 700}, false},
		{"steady plateau", []float64{800, 805, 795, 810, 800, 802}, false},
		{"too few samples", []float64{600, 1200, 2000}, false},
		{"minor jitter within a real climb", []float64{600, 880, 860, 1500, 2400, 3600}, true},
		{"flat at the floor", []float64{500, 500, 500, 500, 500, 500}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SustainedConnectionGrowth(tc.series); got != tc.want {
				t.Fatalf("SustainedConnectionGrowth(%v) = %v, want %v", tc.series, got, tc.want)
			}
		})
	}
}

func feedTCP(rule *tcpConnectionsRule, start time.Time, step time.Duration, values ...float64) RuleEvaluation {
	var eval RuleEvaluation
	for i, v := range values {
		eval = rule.Evaluate(context.Background(), AlertSnapshot{
			Now:     start.Add(time.Duration(i) * step),
			Metrics: &models.MetricsResponse{ConnectionsState: models.MetricState{Status: "measured", Value: v}},
		})
	}
	return eval
}

func TestTCPRule_GrowthWarnsBelowTheLevel(t *testing.T) {
	eval := feedTCP(&tcpConnectionsRule{}, time.Unix(0, 0), 20*time.Second, 600, 900, 1400, 2100, 3000, 4200)
	if len(eval.Conditions) != 1 || eval.Conditions[0].Severity != SeverityWarning || !strings.Contains(eval.Conditions[0].Message, "climbing") {
		t.Fatalf("evaluation = %+v, want a growth warning", eval.Conditions)
	}
}

func TestTCPRule_LeakAtCriticalLevelReportsBoth(t *testing.T) {
	eval := feedTCP(&tcpConnectionsRule{}, time.Unix(0, 0), 20*time.Second, 4000, 9000, 18_000, 30_000, 45_000, 57_000)
	c := eval.Conditions[0]
	if c.Severity != SeverityCritical || !strings.Contains(c.Message, "57000 established") || !strings.Contains(c.Message, "climbing") {
		t.Fatalf("condition = %s %q, want critical naming level and growth", c.Severity, c.Message)
	}
}

func TestTCPRule_GrowthWindowForgetsOldSamples(t *testing.T) {
	rule := &tcpConnectionsRule{}
	// A climb that finished 20 minutes ago is outside the 10-minute window.
	feedTCP(rule, time.Unix(0, 0), 20*time.Second, 600, 900, 1400, 2100, 3000)
	eval := feedTCP(rule, time.Unix(0, 0).Add(20*time.Minute), 20*time.Second, 3000, 3010, 2990, 3005, 3000)
	if len(eval.Conditions) != 0 {
		t.Fatalf("a plateau after an old climb alerted: %+v", eval.Conditions)
	}
}

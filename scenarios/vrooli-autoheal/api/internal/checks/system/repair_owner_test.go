package system

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

// Three host checks sit critical with no safe automatic action. None may offer
// a fake one; each must say who owns the repair, on the result itself.
func TestCriticalHostChecksWithNoSafeActionNameTheirRepairOwner(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	report := writeReport(t, t.TempDir(), map[string]any{"captured_at": now.Add(-time.Minute), "findings": []string{"fork-rate: 740.9 forks/s exceeds SB16 bar"}})
	clean := writeReport(t, t.TempDir(), map[string]any{"captured_at": now.Add(-time.Minute), "findings": []string{}})
	reader := &mockProcReader{memInfo: &checks.MemInfo{SwapTotal: 100, SwapFree: 10}}
	broken := healthyProbes(t)
	broken.Lingering = func(context.Context, string) (bool, error) { return false, nil }

	cases := []struct {
		name    string
		failing checks.Check
		healthy checks.Check
		owner   string
	}{
		{"swap",
			NewSwapCheck(WithSwapProcReader(reader), WithSwapRateReader(func() (float64, error) { return 7773, nil })),
			NewSwapCheck(WithSwapProcReader(reader), WithSwapRateReader(func() (float64, error) { return 0, nil })),
			"steward free-host lane"},
		{"emergency watchdog report",
			NewEmergencyWatchdogReportCheck(WithReportPath(report), WithReportClock(func() time.Time { return now })),
			NewEmergencyWatchdogReportCheck(WithReportPath(clean), WithReportClock(func() time.Time { return now })),
			"steward free-host lane"},
		{"boot recovery readiness",
			NewBootRecoveryReadinessCheck(WithBootRecoveryProbes(broken)),
			NewBootRecoveryReadinessCheck(WithBootRecoveryProbes(healthyProbes(t))),
			"operator via project setup"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			failed := tc.failing.Run(context.Background())
			if healable, ok := tc.failing.(checks.HealableCheck); ok {
				for _, action := range healable.RecoveryActions(&failed) {
					if action.Available {
						t.Fatalf("a check with no safe automatic action offered %q", action.ID)
					}
				}
			}
			if failed.Status != checks.StatusCritical {
				t.Fatalf("fixture is %s, want critical: %s", failed.Status, failed.Message)
			}
			owner, _ := failed.Details["repairOwner"].(string)
			if !strings.Contains(owner, tc.owner) || !strings.Contains(failed.Message, "no automatic repair here") {
				t.Fatalf("failing result names no owner: owner=%q message=%q", owner, failed.Message)
			}
			if ok := tc.healthy.Run(context.Background()); ok.Status != checks.StatusOK || ok.Details["repairOwner"] != nil {
				t.Fatalf("a healthy result must carry no owner note: %s %v", ok.Status, ok.Details["repairOwner"])
			}
		})
	}
}

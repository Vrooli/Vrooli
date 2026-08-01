package system

import (
	"context"
	"errors"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/integrations/cleanupmanager"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/userconfig"
)

type fakeCleanupReporter struct {
	reports []cleanupmanager.Report
	outcome cleanupmanager.Outcome
	err     error
}

func (f *fakeCleanupReporter) ReportPressure(_ context.Context, report cleanupmanager.Report) (cleanupmanager.Outcome, error) {
	f.reports = append(f.reports, report)
	if f.err != nil {
		return cleanupmanager.Outcome{}, f.err
	}
	return f.outcome, nil
}

func diskAt(usedPercent int) *checks.StatfsResult {
	// Choose Bfree and Bavail so the df formula yields usedPercent while the
	// two remain distinct, keeping the Bavail semantics under test.
	const blocks = 10000
	avail := uint64((100 - usedPercent) * 100)
	return &checks.StatfsResult{
		Blocks: blocks,
		Bfree:  avail + 500, // a reserve that Bfree includes and Bavail does not
		Bavail: avail,
		Bsize:  4096,
	}
}

// TestDiskCheck_HealActionRequestsCleanup asserts the disk check can actually
// heal. Before this existed, userconfig declared AutoHeal false with the
// comment "Can't auto-heal disk space", so a critical disk produced a red
// status and no action at all.
func TestDiskCheck_HealActionRequestsCleanup(t *testing.T) {
	reporter := &fakeCleanupReporter{outcome: cleanupmanager.Outcome{Action: "applied", ReclaimedBytes: 8192}}
	c := NewDiskCheck(
		WithPartitions([]string{"/"}),
		WithDiskThresholds(80, 90),
		WithFileSystemReader(&mockFSReader{statfsResult: diskAt(96)}),
		WithCleanupReporter(reporter),
	)

	result := c.ExecuteAction(context.Background(), requestCleanupActionID)

	if !result.Success {
		t.Fatalf("heal action failed: %s / %s", result.Message, result.Error)
	}
	if len(reporter.reports) != 1 {
		t.Fatalf("sent %d reports, want 1", len(reporter.reports))
	}
	report := reporter.reports[0]
	if report.Band != cleanupmanager.BandCritical {
		t.Errorf("band = %s, want critical for a 96%% disk", report.Band)
	}
	if report.SourceScenario != "vrooli-autoheal" {
		t.Errorf("source = %q, want vrooli-autoheal; the audit must attribute this to the independent path", report.SourceScenario)
	}
	if report.Partition != "/" {
		t.Errorf("partition = %q, want /", report.Partition)
	}
}

// TestDiskCheck_HealActionBandMatchesThresholds asserts the heal action
// escalates only as far as the observation justifies.
func TestDiskCheck_HealActionBandMatchesThresholds(t *testing.T) {
	tests := []struct {
		usedPercent int
		wantBand    cleanupmanager.Band
		wantReport  bool
	}{
		{50, "", false}, // healthy: never asks for cleanup
		{79, "", false}, // just below warning
		{85, cleanupmanager.BandHigh, true},
		{95, cleanupmanager.BandCritical, true},
	}

	for _, tc := range tests {
		reporter := &fakeCleanupReporter{}
		c := NewDiskCheck(
			WithPartitions([]string{"/"}),
			WithDiskThresholds(80, 90),
			WithFileSystemReader(&mockFSReader{statfsResult: diskAt(tc.usedPercent)}),
			WithCleanupReporter(reporter),
		)

		result := c.ExecuteAction(context.Background(), requestCleanupActionID)
		if !result.Success {
			t.Fatalf("at %d%%: heal action failed: %s", tc.usedPercent, result.Error)
		}

		if !tc.wantReport {
			if len(reporter.reports) != 0 {
				t.Errorf("at %d%%: requested cleanup for a healthy disk", tc.usedPercent)
			}
			continue
		}
		if len(reporter.reports) != 1 {
			t.Fatalf("at %d%%: sent %d reports, want 1", tc.usedPercent, len(reporter.reports))
		}
		if got := reporter.reports[0].Band; got != tc.wantBand {
			t.Errorf("at %d%%: band = %s, want %s", tc.usedPercent, got, tc.wantBand)
		}
	}
}

// TestDiskCheck_HealActionTargetsFullestPartition asserts remediation is aimed
// at the partition under the most pressure, not whichever is listed first.
func TestDiskCheck_HealActionTargetsFullestPartition(t *testing.T) {
	reporter := &fakeCleanupReporter{}
	c := NewDiskCheck(
		WithPartitions([]string{"/", "/home"}),
		WithDiskThresholds(80, 90),
		WithFileSystemReader(&multiCallFSReader{results: map[string]*checks.StatfsResult{
			"/":     diskAt(60),
			"/home": diskAt(97),
		}}),
		WithCleanupReporter(reporter),
	)

	result := c.ExecuteAction(context.Background(), requestCleanupActionID)
	if !result.Success {
		t.Fatalf("heal action failed: %s", result.Error)
	}
	if len(reporter.reports) != 1 {
		t.Fatalf("sent %d reports, want 1", len(reporter.reports))
	}
	if got := reporter.reports[0].Partition; got != "/home" {
		t.Errorf("reported partition = %q, want /home (the fullest)", got)
	}
	if got := reporter.reports[0].Band; got != cleanupmanager.BandCritical {
		t.Errorf("band = %s, want critical", got)
	}
}

// TestDiskCheck_HealActionReportsFailure asserts an unreachable cleanup-manager
// surfaces as a failed action rather than a silent success.
func TestDiskCheck_HealActionReportsFailure(t *testing.T) {
	reporter := &fakeCleanupReporter{err: errors.New("connection refused")}
	c := NewDiskCheck(
		WithPartitions([]string{"/"}),
		WithDiskThresholds(80, 90),
		WithFileSystemReader(&mockFSReader{statfsResult: diskAt(96)}),
		WithCleanupReporter(reporter),
	)

	result := c.ExecuteAction(context.Background(), requestCleanupActionID)
	if result.Success {
		t.Error("an unreachable cleanup-manager reported success")
	}
	if result.Error == "" {
		t.Error("no error recorded for a failed heal")
	}
}

// TestDiskCheck_AutoHealIsEnabled pins the configuration change. The check
// previously declared AutoHeal false, so a critical disk produced a red status
// and nothing else — which is what happened on 2026-07-31.
func TestDiskCheck_AutoHealIsEnabled(t *testing.T) {
	defaults := userconfig.GetCheckDefaults("system-disk")
	if !defaults.AutoHeal {
		t.Error("system-disk has AutoHeal disabled; disk pressure would again produce a red status and no action")
	}
	if defaults.AutoHealOn != "critical" {
		t.Errorf("AutoHealOn = %q, want critical", defaults.AutoHealOn)
	}

	// The heal action must be advertised, or auto-heal has nothing to run.
	var found bool
	for _, action := range NewDiskCheck().RecoveryActions(nil) {
		if action.ID == requestCleanupActionID {
			found = true
			if action.Dangerous {
				t.Error("the cleanup request is marked dangerous, which would exclude it from unattended healing")
			}
			if !action.Available {
				t.Error("the cleanup request is marked unavailable")
			}
		}
	}
	if !found {
		t.Error("system-disk advertises no request-cleanup recovery action")
	}
}

package destinationreadiness_test

import (
	"context"
	"testing"

	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/sysmounts"
)

type fakeProber struct {
	result destinationreadiness.WriteProbeResult
	err    error
	calls  int
}

func (p *fakeProber) Probe(context.Context, string) (destinationreadiness.WriteProbeResult, error) {
	p.calls++
	return p.result, p.err
}

func degradedInspection(id destinationreadiness.DeviceIdentity) destinationreadiness.Inspection {
	return destinationreadiness.Inspection{
		LocationExists: true, LocationIsDirectory: true, Identity: id,
		Mounted: true, FreeBytes: 100 << 30, Platform: "linux",
		FilesystemState: sysmounts.FilesystemStateUnknown,
		StateEvidence:   "mount-options",
	}
}

func TestAnalyzeDegradedWriteNeedsAPassedProbe(t *testing.T) {
	id := identity()
	prober := &fakeProber{result: destinationreadiness.WriteProbeResult{Ran: true, Passed: true, Bytes: 4096}}
	svc := destinationreadiness.NewService(&fakeInspector{inspection: degradedInspection(id)}, nil).WithWriteProber(prober)
	report, err := svc.Analyze(context.Background(), destinationreadiness.AnalyzeInput{
		DestinationID: "dest-a", ExpectedDevice: id, Location: "/media/user/USB/backup",
		SelectedTargetBytes: 1 << 30, RetentionCopies: 2,
		AllowWriteProbe: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.WriteVerdict != destinationreadiness.WriteDegraded {
		t.Fatalf("verdict = %q (%s), want degraded", report.WriteVerdict, report.WriteVerdictReason)
	}
	if report.FilesystemState != sysmounts.FilesystemStateUnknown {
		t.Fatalf("native state = %q, want the truthful unknown observation", report.FilesystemState)
	}
	if report.OverallSeverity != destinationreadiness.SeverityWarning {
		t.Fatalf("overall severity = %q, want warning for a degraded write", report.OverallSeverity)
	}
	if prober.calls != 1 {
		t.Fatalf("probe calls = %d, want one bounded probe", prober.calls)
	}
}

// A read-only readiness check does not write, so it has no probe evidence and
// the verdict stays truthfully unknown rather than claiming writability.
func TestAnalyzeWithoutAProbeDoesNotUpgradeUnknownState(t *testing.T) {
	id := identity()
	svc := destinationreadiness.NewService(&fakeInspector{inspection: degradedInspection(id)}, nil).
		WithWriteProber(&fakeProber{result: destinationreadiness.WriteProbeResult{Ran: true, Passed: true}})
	report, err := svc.Analyze(context.Background(), destinationreadiness.AnalyzeInput{
		DestinationID: "dest-a", ExpectedDevice: id, Location: "/media/user/USB/backup",
		SelectedTargetBytes: 1 << 30, RetentionCopies: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.WriteVerdict != destinationreadiness.WriteUnknown {
		t.Fatalf("verdict = %q (%s), want unknown without a probe", report.WriteVerdict, report.WriteVerdictReason)
	}
	if report.OverallSeverity != destinationreadiness.SeverityUnknown {
		t.Fatalf("overall severity = %q, want unknown", report.OverallSeverity)
	}
}

func TestAnalyzeFailedProbeBlocksEvenWithCleanNativeState(t *testing.T) {
	id := identity()
	inspection := degradedInspection(id)
	inspection.FilesystemState = sysmounts.FilesystemStateClean
	svc := destinationreadiness.NewService(&fakeInspector{inspection: inspection}, nil).
		WithWriteProber(&fakeProber{result: destinationreadiness.WriteProbeResult{Ran: true, Passed: false, Error: "EIO"}})
	report, err := svc.Analyze(context.Background(), destinationreadiness.AnalyzeInput{
		DestinationID: "dest-a", ExpectedDevice: id, Location: "/media/user/USB/backup",
		SelectedTargetBytes: 1 << 30, RetentionCopies: 2,
		AllowWriteProbe: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.WriteVerdict != destinationreadiness.WriteBlocked {
		t.Fatalf("verdict = %q (%s), want blocked", report.WriteVerdict, report.WriteVerdictReason)
	}
	if report.OverallSeverity != destinationreadiness.SeverityFail {
		t.Fatalf("overall severity = %q, want fail", report.OverallSeverity)
	}
}

func TestReadOnlyAnalyzeDoesNotProbe(t *testing.T) {
	id := identity()
	prober := &fakeProber{result: destinationreadiness.WriteProbeResult{Ran: true, Passed: true}}
	svc := destinationreadiness.NewService(&fakeInspector{inspection: degradedInspection(id)}, nil).WithWriteProber(prober)
	report, err := svc.Analyze(context.Background(), destinationreadiness.AnalyzeInput{
		DestinationID: "dest-a", ExpectedDevice: id, Location: "/media/user/USB/backup",
		SelectedTargetBytes: 1 << 30, RetentionCopies: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if prober.calls != 0 {
		t.Fatalf("probe calls = %d, want none for read-only readiness", prober.calls)
	}
	if report.WriteProbe.Ran {
		t.Fatal("read-only readiness must not report a write probe as run")
	}
}

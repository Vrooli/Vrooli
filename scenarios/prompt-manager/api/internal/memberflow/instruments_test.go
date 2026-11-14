package memberflow

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type instrumentProbeFunc func(context.Context, string) error

func (f instrumentProbeFunc) Check(ctx context.Context, scenario string) error {
	return f(ctx, scenario)
}

func TestComputeInstrumentCoverageReportsUnreachableLiveInstrument(t *testing.T) {
	root := t.TempDir()
	teamDir := filepath.Join(root, "teams", "monetization")
	if err := os.MkdirAll(teamDir, 0o755); err != nil {
		t.Fatal(err)
	}
	team := `{"id":"monetization","instrument":{"status":"live","scenario":"offer-desk"}}`
	if err := os.WriteFile(filepath.Join(teamDir, "team.json"), []byte(team), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := ComputeInstrumentCoverageWithReachability(root, instrumentProbeFunc(func(context.Context, string) error {
		return context.DeadlineExceeded
	}))
	if err != nil {
		t.Fatal(err)
	}
	if report.OutOfBand != 1 || len(report.Teams) != 1 || report.Teams[0].Reachable == nil || *report.Teams[0].Reachable {
		t.Fatalf("report = %#v, want one unreachable team", report)
	}
	if !strings.Contains(strings.Join(report.Teams[0].Findings, " "), "unavailable") {
		t.Fatalf("findings = %#v, want unavailable reason", report.Teams[0].Findings)
	}
}

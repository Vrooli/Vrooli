package interventions

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func isolatedHome(t *testing.T) {
	t.Helper()
	t.Setenv("VROOLI_HOME", t.TempDir())
	t.Setenv(SignalsSwitchEnv, "")
}

func TestSignalSwitchTurnsOffAndOnWithoutARestart(t *testing.T) {
	isolatedHome(t)
	control := SignalControl{Now: func() time.Time { return time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC) }}
	if enabled, source, _ := SignalState(); !enabled || source != SignalSourceDefault {
		t.Fatalf("default = %v %s, want on by default", enabled, source)
	}
	if err := control.Set(false, "operator@example.com"); err != nil {
		t.Fatal(err)
	}
	// The job asks before every pass, so the next ask sees the new state.
	if SignalsEnabled() {
		t.Fatal("signals must read as off immediately after `off`")
	}
	status, err := control.Status(context.Background())
	if err != nil || status.Enabled || status.Source != SignalSourceOperator || status.ChangedBy != "operator@example.com" {
		t.Fatalf("status = %+v %v", status, err)
	}
	if err := control.Set(true, "operator@example.com"); err != nil {
		t.Fatal(err)
	}
	if !SignalsEnabled() {
		t.Fatal("signals must read as on after `on`")
	}
}

func TestEnvironmentOffBeatsAnOperatorOnAndSaysSo(t *testing.T) {
	isolatedHome(t)
	control := SignalControl{}
	if err := control.Set(true, "operator@example.com"); err != nil {
		t.Fatal(err)
	}
	t.Setenv(SignalsSwitchEnv, "off")
	status, err := control.Status(context.Background())
	if err != nil || status.Enabled || status.Source != SignalSourceEnvironment || !strings.Contains(status.Note, SignalsSwitchEnv) {
		t.Fatalf("status = %+v %v", status, err)
	}
}

func TestACorruptSwitchFileFailsClosedBecauseItIsAPrivacyControl(t *testing.T) {
	isolatedHome(t)
	path := SignalSwitchPath()
	if err := os.MkdirAll(path[:strings.LastIndex(path, "/")], 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if SignalsEnabled() {
		t.Fatal("an unreadable switch must not leave message classification running")
	}
	if err := (SignalControl{}).Set(true, "operator@example.com"); err != nil || !SignalsEnabled() {
		t.Fatalf("`on` must repair the file: %v", err)
	}
}

func TestPurgeDeletesEveryStoredSignalAndStatusCountsThem(t *testing.T) {
	isolatedHome(t)
	f, _ := newSignalFixture(t)
	now := f.now
	records := []SignalRecord{
		{SignalCandidate: SignalCandidate{EventID: "e1", RunID: "r1", OccurredAt: now.Format(time.RFC3339Nano)}, Answer: SignalAnswer{Status: StatusClassified, Signal: ReportedFailure}},
		{SignalCandidate: SignalCandidate{EventID: "e2", RunID: "r1", OccurredAt: now.Format(time.RFC3339Nano)}, Answer: SignalAnswer{Status: StatusClassified, Signal: Neutral}},
	}
	if err := f.store.SaveSignals(context.Background(), SignalVersion, records, now); err != nil {
		t.Fatal(err)
	}
	control := SignalControl{Store: f.store}
	status, err := control.Status(context.Background())
	if err != nil || status.Stored != 2 {
		t.Fatalf("stored = %d %v", status.Stored, err)
	}
	purged, err := control.Purge(context.Background())
	if err != nil || purged != 2 {
		t.Fatalf("purged = %d %v", purged, err)
	}
	if after, _ := control.Status(context.Background()); after.Stored != 0 {
		t.Fatalf("stored after purge = %d", after.Stored)
	}
	if _, err := (SignalControl{}).Purge(context.Background()); err == nil {
		t.Fatal("a purge with no store must say so, not report success")
	}
}

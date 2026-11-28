package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	sessionsH "web-console/handlers/sessions"
	"web-console/internal/sessionstore"
)

// recordingDaemons is an AgentDaemonControl whose Stop notes whether the
// session's packages/ install still existed, to prove the order of operations.
type recordingDaemons struct {
	root           string
	running        map[string]sessionsH.AgentDaemonUsage
	stopped        []string
	packagesAtStop map[string]bool
}

func (d *recordingDaemons) Running(context.Context) (map[string]sessionsH.AgentDaemonUsage, error) {
	return d.running, nil
}

func (d *recordingDaemons) Stop(_ context.Context, id string) (sessionsH.AgentDaemonUsage, error) {
	usage := d.running[id]
	delete(d.running, id)
	d.stopped = append(d.stopped, id)
	if d.packagesAtStop == nil {
		d.packagesAtStop = map[string]bool{}
	}
	_, err := os.Lstat(filepath.Join(d.root, "codex", id, "packages"))
	d.packagesAtStop[id] = err == nil
	return usage, nil
}

func newRecordingDaemons(root string, ids ...string) *recordingDaemons {
	d := &recordingDaemons{root: root, running: map[string]sessionsH.AgentDaemonUsage{}}
	for _, id := range ids {
		d.running[id] = sessionsH.AgentDaemonUsage{Processes: 3, MemoryBytes: 1000}
	}
	return d
}

func TestSessionHomeRetentionStopsLeakedDaemonsOfRetiredSessionsOnly(t *testing.T) {
	root := t.TempDir()
	f := homeFixture{t: t, root: root}
	for _, id := range []string{"archived", "dismissed", "active", "live-archived", "orphan"} {
		// Fresh homes: the idle-strip rules protect them, but a daemon with
		// no session using it is stopped regardless of home age.
		f.write("codex/"+id+"/sessions/rollout.jsonl", 10)
	}
	rows := []sessionstore.Metadata{
		{ID: "archived", AgentType: sessionstore.AgentCodex, Status: sessionstore.StatusLive, ArchivedAt: time.Now()},
		{ID: "dismissed", AgentType: sessionstore.AgentCodex, Status: sessionstore.StatusDismissed},
		{ID: "active", AgentType: sessionstore.AgentCodex, Status: sessionstore.StatusLive},
		{ID: "live-archived", AgentType: sessionstore.AgentCodex, Status: sessionstore.StatusLive, ArchivedAt: time.Now()},
		{ID: "orphan", AgentType: sessionstore.AgentCodex, Status: sessionstore.StatusAwaitingRecovery},
	}
	daemons := newRecordingDaemons(root, "archived", "dismissed", "active", "live-archived", "orphan", "no-row")
	owner := newHomeRetention(root, rows, "live-archived")
	owner.daemons = daemons

	result, err := owner.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := len(daemons.stopped); got != 2 || result.StoppedDaemons != 2 || result.DaemonBytesFreed != 2000 {
		t.Fatalf("stopped %v; result stopped=%d bytes=%d", daemons.stopped, result.StoppedDaemons, result.DaemonBytesFreed)
	}
	for _, id := range []string{"active", "live-archived", "orphan", "no-row"} {
		if _, ok := daemons.running[id]; !ok {
			t.Errorf("daemon of %s was stopped", id)
		}
	}
	if result.Failures != 0 {
		t.Fatalf("failures = %d (%s)", result.Failures, result.FirstFailure)
	}
}

func TestSessionHomeRetentionStopsDaemonBeforeStrippingItsInstall(t *testing.T) {
	root := t.TempDir()
	f := homeFixture{t: t, root: root}
	f.write("codex/idle/sessions/rollout.jsonl", 10)
	f.write("codex/idle/packages/app-server-daemon/releases/1/bin/codex", 4000)
	f.age("codex/idle", 40*24*time.Hour)
	rows := []sessionstore.Metadata{{ID: "idle", AgentType: sessionstore.AgentCodex, Status: sessionstore.StatusAwaitingRecovery}}
	daemons := newRecordingDaemons(root, "idle")
	owner := newHomeRetention(root, rows)
	owner.daemons = daemons

	if _, err := owner.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(daemons.stopped) != 1 || !daemons.packagesAtStop["idle"] {
		t.Fatalf("stopped=%v, install present at stop=%v: the daemon must stop before its install is removed", daemons.stopped, daemons.packagesAtStop)
	}
	if exists(t, filepath.Join(root, "codex", "idle", "packages")) {
		t.Fatal("install was not stripped")
	}
}

func TestSessionDaemonsReportsNothingForRootWithoutDaemons(t *testing.T) {
	control := newSessionDaemons(func() string { return t.TempDir() })
	running, err := control.Running(context.Background())
	if err != nil || len(running) != 0 {
		t.Fatalf("running = %v, %v", running, err)
	}
	if usage, err := control.Stop(context.Background(), "absent"); err != nil || usage.Processes != 0 {
		t.Fatalf("stop = %+v, %v", usage, err)
	}
}

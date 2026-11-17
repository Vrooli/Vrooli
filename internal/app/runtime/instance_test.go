package runtimeapp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/supervision"
)

func TestInspectRuntimeInstanceReadOnlyReturnsExactStopEvidence(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.CreateLease(ctx, scenarioruntime.Instance{InstanceID: "inst-owner-proof", Scenario: "web-console", Variant: "live"}, time.Minute)
	if err != nil {
		store.Close()
		t.Fatal(err)
	}
	stopped, err := store.StopLease(ctx, created.InstanceID, created.Generation, "fixture stop")
	if err != nil {
		store.Close()
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := (&Service{}).InspectRuntimeInstance(ctx, home, created.InstanceID, &output, true); err != nil {
		t.Fatal(err)
	}
	var got RuntimeInstanceInspection
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.InstanceID != created.InstanceID || got.Scenario != "web-console" || got.Variant != "live" || got.Generation != created.Generation || got.Status != scenarioruntime.StatusStopped || got.StoppedAt == "" || stopped.StoppedAt == nil {
		t.Fatalf("inspection = %+v, want exact stopped instance record", got)
	}

	output.Reset()
	if err := (&Service{}).InspectRuntimeInstance(ctx, home, created.InstanceID, &output, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Status: stopped") || !strings.Contains(output.String(), "Stopped: "+got.StoppedAt) {
		t.Fatalf("human inspection = %q", output.String())
	}
}

func TestRuntimeProcessRefStateRequiresBootAndBirthMatch(t *testing.T) {
	pid := 4242
	started := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	ref := scenarioruntime.ProcessRef{PID: &pid, HostBootID: "boot-current", StartedAt: started}
	processes := map[int]supervision.ProcessInfo{pid: {PID: pid, StartedAt: started.Add(time.Second)}}
	if got := runtimeProcessRefState(ref, "boot-current", processes, nil); got != "present" {
		t.Fatalf("matching process state = %q", got)
	}
	processes[pid] = supervision.ProcessInfo{PID: pid, StartedAt: started.Add(3 * time.Second)}
	if got := runtimeProcessRefState(ref, "boot-current", processes, nil); got != "absent" {
		t.Fatalf("reused PID state = %q", got)
	}
	if got := runtimeProcessRefState(ref, "boot-current", nil, context.DeadlineExceeded); got != "unknown" {
		t.Fatalf("failed inventory state = %q", got)
	}
}

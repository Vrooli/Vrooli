package hostinventory

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/shell/shelltest"
	"github.com/vrooli/vrooli/internal/testenv"
)

func TestNvidiaProcessObservationReplacesPreviousRowsAndQualifiesAbsence(t *testing.T) {
	const command = "nvidia-smi pmon -s m -c 1"
	for _, test := range []struct {
		name, output, wantStatus string
		err                      error
		wantRows                 int
	}{
		{name: "new-process", output: "# gpu pid type fb ccpm command\n0 42 C 214 - python\n", wantStatus: "ok", wantRows: 1},
		{name: "complete-empty", output: "# gpu pid type fb ccpm command\n", wantStatus: "ok"},
		{name: "empty-nonzero-tool-convention", output: "No running processes found", err: errors.New("exit status 1"), wantStatus: "ok"},
		{name: "failure", err: errors.New("permission denied"), wantStatus: "failed"},
		{name: "partial", output: "# gpu pid type fb ccpm command\n42 C 214 - python\n0 bad C 1 - broken\n", wantStatus: "incomplete"},
	} {
		t.Run(test.name, func(t *testing.T) {
			now := time.Date(2026, 10, 16, 20, 0, 0, 0, time.UTC)
			commands := &shelltest.Fake{Outputs: map[string][]byte{command: []byte(test.output)}, Errors: map[string]error{command: test.err}}
			collector := Collector{Commands: commands, GOOS: "linux"}
			snapshot := Snapshot{GPUs: []GPU{{Index: 0, UUID: "GPU-test"}}, GPUProcesses: []GPUProcess{{PID: 10, UsedBytes: 100}}, ProbeStatuses: map[string]string{"nvidia_gpu_processes": "ok"}, FieldProvenance: map[string]Provenance{}}
			collector.collectNvidiaGPUProcesses(context.Background(), &snapshot, now)
			if len(snapshot.GPUProcesses) != test.wantRows || snapshot.ProbeStatuses["nvidia_gpu_processes"] != test.wantStatus {
				t.Fatalf("stale rows or unqualified absence: %+v", snapshot)
			}
			if test.wantRows > 0 && snapshot.GPUProcesses[0].PID != 42 {
				t.Fatalf("old process retained: %+v", snapshot.GPUProcesses)
			}
			if test.wantStatus == "ok" && snapshot.FieldProvenance["gpu_processes"].ObservedAt != now {
				t.Fatalf("missing current successful observation witness: %+v", snapshot.FieldProvenance)
			}
			if !reflect.DeepEqual(commands.Calls(), []string{command}) {
				t.Fatalf("dynamic observation recollected unrelated facts: %v", commands.Calls())
			}
		})
	}
}

func TestNvidiaProcessObservationUsesProbeCompletionTime(t *testing.T) {
	const command = "nvidia-smi pmon -s m -c 1"
	started := time.Date(2026, 10, 16, 20, 0, 0, 0, time.UTC)
	completed := started.Add(2 * time.Second)
	commands := &shelltest.Fake{Outputs: map[string][]byte{command: []byte("# gpu pid type fb ccpm command\n0 42 C 214 - python\n")}}
	collector := Collector{Commands: commands, Clock: testenv.NewClock(completed), GOOS: "linux"}
	snapshot := Snapshot{GPUs: []GPU{{Index: 0, UUID: "GPU-test"}}, ProbeStatuses: map[string]string{}, FieldProvenance: map[string]Provenance{}}
	collector.collectNvidiaGPUProcesses(context.Background(), &snapshot, started)
	if got := snapshot.FieldProvenance["gpu_processes"].ObservedAt; !got.Equal(completed) {
		t.Fatalf("GPU process observation time = %s, want probe completion %s", got, completed)
	}
}

func TestRefreshNvidiaGPUProcessesReusesStableFactsWithoutMutatingCachedSnapshot(t *testing.T) {
	const query = "nvidia-smi pmon -s m -c 1"
	now := time.Date(2026, 10, 16, 20, 0, 0, 0, time.UTC)
	cached := Snapshot{GPUs: []GPU{{Index: 3, UUID: "GPU-current", Name: "stable identity"}}, RuntimeTools: map[string]Tool{ToolNvidiaSMI: {Present: true}}, GPUProcesses: []GPUProcess{{PID: 10, UsedBytes: 100}}, ProbeStatuses: map[string]string{"nvidia_gpu": "ok", "nvidia_gpu_processes": "ok"}, FieldProvenance: map[string]Provenance{"gpu_processes": {ObservedAt: now.Add(-time.Minute)}}, Warnings: []string{"retained static warning"}}
	before, _ := json.Marshal(cached)
	commands := &shelltest.Fake{Outputs: map[string][]byte{query: []byte("# gpu pid type fb ccpm command\n3 42 C 214 - python\n")}}
	got, err := (Collector{Commands: commands, Clock: testenv.NewClock(now), GOOS: "linux"}).RefreshNvidiaGPUProcesses(context.Background(), cached)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(cached)
	if string(before) != string(after) {
		t.Fatalf("refresh mutated reusable cached facts: %s", after)
	}
	if len(got.GPUProcesses) != 1 || got.GPUProcesses[0].PID != 42 || got.GPUProcesses[0].GPUIndex != 3 {
		t.Fatalf("current process was not joined to stable identity: %+v", got.GPUProcesses)
	}
	if !reflect.DeepEqual(got.GPUs, cached.GPUs) || got.ProbeStatuses["nvidia_gpu"] != "ok" || got.FieldProvenance["gpu_processes"].ObservedAt != now {
		t.Fatalf("static facts changed or observation is stale: %+v", got)
	}
	if !reflect.DeepEqual(commands.Calls(), []string{query}) {
		t.Fatalf("unrelated inventory work: %v", commands.Calls())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (Collector{Commands: commands}).RefreshNvidiaGPUProcesses(ctx, cached); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation ignored: %v", err)
	}
	if len(commands.Calls()) != 1 {
		t.Fatal("canceled refresh still invoked the tool")
	}
}

package accel

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// nvidia-smi reports a process basename, not the artifact path, so a daemon
// whose model runner is a separate child (Ollaya's cuda runner, Ollama's
// llama-server) can only be attributed by ancestry. Without this a healthy GPU
// daemon is reported as CPU drift.
func TestMatchComputeProcessAttributesALiveDescendant(t *testing.T) {
	previous := descendantPIDs
	descendantPIDs = func(root int) map[int]struct{} {
		if root != 1210690 {
			return nil
		}
		return map[int]struct{}{1219552: {}}
	}
	t.Cleanup(func() { descendantPIDs = previous })

	snapshot := hostinventory.Snapshot{GPUProcesses: []hostinventory.GPUProcess{
		{GPUIndex: 0, PID: 1219552, ProcessName: "ollaya", UsedBytes: 1264 << 20},
	}}
	row, ok := matchComputeProcess(snapshot, HostProcess{PID: 1210690, Name: "ollaya"})
	if !ok || row.PID != 1219552 {
		t.Fatalf("matchComputeProcess = %+v ok=%t, want the descendant runner", row, ok)
	}

	// A process outside the supervised tree is still not this resource.
	descendantPIDs = func(int) map[int]struct{} { return nil }
	if _, ok := matchComputeProcess(snapshot, HostProcess{PID: 1210690, Name: "ollaya"}); ok {
		t.Fatal("matched a GPU process that is not the supervised process or a descendant")
	}
}

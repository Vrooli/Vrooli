package hostapp

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// TestWriteHostSnapshotJSONContract pins the `host inventory --json` wire shape.
func TestWriteHostSnapshotJSONContract(t *testing.T) {
	temp := 61.5
	snapshot := hostinventory.Snapshot{
		OS:     "linux",
		Arch:   "amd64",
		CPU:    hostinventory.CPU{Cores: 8},
		Load:   hostinventory.Load{Load1: 1.5, RunningProcs: 3, LastPID: 42, NormalizedLoad1: 0.1875},
		Memory: hostinventory.Memory{TotalBytes: 17179869184, AvailableBytes: 8589934592},
		Swap:   hostinventory.Swap{TotalBytes: 2147483648},
		GPUs: []hostinventory.GPU{{
			Index:        0,
			Name:         "NVIDIA",
			VRAMBytes:    8589934592,
			TemperatureC: &temp,
			Source:       "nvidia-smi",
		}},
		GPUProcesses: []hostinventory.GPUProcess{{GPUIndex: 0, PID: 100, ProcessName: "x", UsedBytes: 1024}},
		RuntimeTools: map[string]hostinventory.Tool{"docker": {Present: true, Path: "/usr/bin/docker"}},
		DockerGPU:    hostinventory.DockerGPU{NvidiaRuntime: true},
		Warnings:     []string{"w1"},
		ProbeStatuses: map[string]string{
			"cpu": "ok",
		},
		FieldProvenance: map[string]hostinventory.Provenance{
			"os": {
				SourceKind: hostinventory.SourceKindRuntime,
				Source:     "runtime.GOOS",
				ObservedAt: time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC),
				Confidence: "high",
			},
		},
	}

	var buf bytes.Buffer
	if err := writeHostSnapshotJSON(&buf, snapshot); err != nil {
		t.Fatalf("writeHostSnapshotJSON: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}

	if got["os"] != "linux" || got["arch"] != "amd64" {
		t.Errorf("os/arch mismatch: %v %v", got["os"], got["arch"])
	}

	// Byte totals are int64 (physical memory exceeds int32); protojson
	// serializes int64 as a JSON string, which is acceptable here because the
	// value is large and the field is documented as int64.
	mem := got["memory"].(map[string]any)
	if mem["total_bytes"] != "17179869184" {
		t.Errorf("memory.total_bytes: want \"17179869184\", got %T %v", mem["total_bytes"], mem["total_bytes"])
	}

	cpu := got["cpu"].(map[string]any)
	if cpu["cores"].(float64) != 8 {
		t.Errorf("cpu.cores: %v", cpu["cores"])
	}

	gpus, ok := got["gpus"].([]any)
	if !ok || len(gpus) != 1 {
		t.Fatalf("gpus: want 1, got %v", got["gpus"])
	}
	gpu := gpus[0].(map[string]any)
	if gpu["temperature_c"].(float64) != 61.5 {
		t.Errorf("gpu.temperature_c: %v", gpu["temperature_c"])
	}

	tools := got["runtime_tools"].(map[string]any)
	docker := tools["docker"].(map[string]any)
	if docker["present"] != true || docker["path"] != "/usr/bin/docker" {
		t.Errorf("runtime_tools.docker mismatch: %v", docker)
	}

	// FieldProvenance has no json tag in Go; ensure it is exposed as snake_case.
	fp, ok := got["field_provenance"].(map[string]any)
	if !ok {
		t.Fatalf("field_provenance missing/wrong (snake_case?): %v", got["field_provenance"])
	}
	osProv := fp["os"].(map[string]any)
	if osProv["source_kind"] != "runtime" || osProv["observed_at"] != "2026-06-11T12:00:00Z" {
		t.Errorf("field_provenance.os mismatch: %v", osProv)
	}
}

package capacity

import (
	"context"
	"testing"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// fakeAttributor maps PIDs to fixed attributions for deterministic tests.
type fakeAttributor map[int]Attribution

func (f fakeAttributor) Attribute(_ context.Context, pid int) Attribution {
	if a, ok := f[pid]; ok {
		a.PID = pid
		return a
	}
	return Attribution{PID: pid, OwnerID: OwnerUnknown}
}

func snapshotWithProcs(procs ...hostinventory.GPUProcess) hostinventory.Snapshot {
	return hostinventory.Snapshot{
		GPUs:         []hostinventory.GPU{{Index: 0, Name: "Test", Source: "nvidia-smi", VRAMBytes: 16 * uint64(gib)}},
		GPUProcesses: procs,
	}
}

func TestReconcileFlagsUnclaimedConsumer(t *testing.T) {
	ctx := context.Background()
	snap := snapshotWithProcs(hostinventory.GPUProcess{GPUIndex: 0, PID: 1000, ProcessName: "python", UsedBytes: 7 * uint64(gib)})
	attr := fakeAttributor{1000: {ContainerName: "/vrooli-whisper-1", OwnerID: "whisper"}}

	findings := Reconcile(ctx, snap, nil, attr, DefaultPolicy())
	if len(findings) != 1 {
		t.Fatalf("len(findings) = %d, want 1", len(findings))
	}
	f := findings[0]
	if f.Class != FindingUnclaimed || f.Severity != "warn" {
		t.Errorf("class/severity = %s/%s, want unclaimed/warn", f.Class, f.Severity)
	}
	if f.OwnerID != "whisper" {
		t.Errorf("owner = %q, want whisper (normalized)", f.OwnerID)
	}
}

func TestReconcileClassifiesClaimed(t *testing.T) {
	ctx := context.Background()
	snap := snapshotWithProcs(hostinventory.GPUProcess{GPUIndex: 0, PID: 1000, ProcessName: "whisper", UsedBytes: 7 * uint64(gib)})
	attr := fakeAttributor{1000: {ContainerName: "/vrooli-whisper-1", OwnerID: "whisper"}}
	ledger := []CapacityClaim{{
		ClaimID: "clm-w", OwnerID: "whisper", OwnerKind: OwnerKindResource, ResourceKind: ResourceKindVRAM,
		GPUIndex: gpu(0), AmountBytes: 8 * gib, Status: StatusGranted,
	}}

	findings := Reconcile(ctx, snap, ledger, attr, DefaultPolicy())
	if len(findings) != 1 || findings[0].Class != FindingClaimed {
		t.Fatalf("findings = %+v, want one claimed", findings)
	}
	if findings[0].Severity != "info" || findings[0].ClaimID != "clm-w" {
		t.Errorf("claimed finding = %+v", findings[0])
	}
}

func TestReconcileFlagsOverClaim(t *testing.T) {
	ctx := context.Background()
	// Uses 8 GiB but only claimed 2 GiB; drift well over the 512 MiB default.
	snap := snapshotWithProcs(hostinventory.GPUProcess{GPUIndex: 0, PID: 1000, ProcessName: "sd", UsedBytes: 8 * uint64(gib)})
	attr := fakeAttributor{1000: {ContainerName: "/image-tools", OwnerID: "image-tools"}}
	ledger := []CapacityClaim{{
		ClaimID: "clm-it", OwnerID: "image-tools:job-1", OwnerKind: OwnerKindOp, ResourceKind: ResourceKindVRAM,
		GPUIndex: gpu(0), AmountBytes: 2 * gib, Status: StatusGranted,
	}}

	findings := Reconcile(ctx, snap, ledger, attr, DefaultPolicy())
	if len(findings) != 1 || findings[0].Class != FindingOverClaim {
		t.Fatalf("findings = %+v, want over_claim (op-owner should match scenario prefix)", findings)
	}
	if findings[0].Severity != "warn" {
		t.Errorf("over_claim severity = %q, want warn", findings[0].Severity)
	}
}

func TestReconcileIgnoresBelowThreshold(t *testing.T) {
	ctx := context.Background()
	snap := snapshotWithProcs(hostinventory.GPUProcess{GPUIndex: 0, PID: 1000, UsedBytes: 64 * 1024 * 1024}) // 64 MiB < 256 MiB
	if findings := Reconcile(ctx, snap, nil, fakeAttributor{}, DefaultPolicy()); len(findings) != 0 {
		t.Errorf("findings = %+v, want none below tracking threshold", findings)
	}
}

func TestReconcileNilAttributorDegradesToUnknown(t *testing.T) {
	ctx := context.Background()
	snap := snapshotWithProcs(hostinventory.GPUProcess{GPUIndex: 0, PID: 1000, ProcessName: "ollama", UsedBytes: 3 * uint64(gib)})
	findings := Reconcile(ctx, snap, nil, nil, DefaultPolicy())
	if len(findings) != 1 || findings[0].Class != FindingUnclaimed {
		t.Fatalf("findings = %+v, want one unclaimed", findings)
	}
	if findings[0].OwnerID != "ollama" { // falls back to process name
		t.Errorf("owner = %q, want process-name fallback ollama", findings[0].OwnerID)
	}
}

func TestNormalizeOwnerName(t *testing.T) {
	cases := map[string]string{
		"/vrooli-whisper-1":  "whisper",
		"vrooli_kyutai-stt":  "kyutai-stt",
		"/resource-ollama-2": "ollama",
		"image-tools":        "image-tools",
		"":                   OwnerUnknown,
	}
	for in, want := range cases {
		if got := NormalizeOwnerName(in); got != want {
			t.Errorf("NormalizeOwnerName(%q) = %q, want %q", in, got, want)
		}
	}
}

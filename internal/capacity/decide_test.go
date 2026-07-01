package capacity

import (
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

const gib = int64(1) << 30

// snapshotWith builds a single-GPU snapshot with the given total/used VRAM.
func snapshotWith(totalGiB, usedGiB int64) hostinventory.Snapshot {
	return hostinventory.Snapshot{
		GPUs: []hostinventory.GPU{{
			Index:         0,
			Name:          "Test GPU",
			Source:        "nvidia-smi",
			VRAMBytes:     uint64(totalGiB * gib),
			VRAMUsedBytes: uint64(usedGiB * gib),
		}},
	}
}

func vramReq(preferredGiB, floorGiB int64, priority int) CapacityRequest {
	return CapacityRequest{
		OwnerKind:      OwnerKindScenario,
		OwnerID:        "image-tools",
		ResourceKind:   ResourceKindVRAM,
		GPUIndex:       gpu(0),
		PreferredBytes: preferredGiB * gib,
		FloorBytes:     floorGiB * gib,
		Priority:       priority,
	}
}

func TestDecideGrantWhenFree(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	v := Decide(vramReq(4, 1, PriorityBatch), snapshotWith(16, 4), nil, DefaultPolicy(), now)
	if v.Kind != VerdictGrant {
		t.Fatalf("kind = %q, want grant (%s)", v.Kind, v.Reason)
	}
	if v.GrantedBytes != 4*gib {
		t.Errorf("granted = %d, want %d", v.GrantedBytes, 4*gib)
	}
	if len(v.ReclaimTargets) != 0 {
		t.Errorf("unexpected reclaim targets: %v", v.ReclaimTargets)
	}
}

func TestDecideDegradeToFittingStep(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	req := vramReq(8, 1, PriorityBatch)
	req.Profile = &DegradeProfile{Steps: []DegradeStep{
		{Label: "fp16", AmountBytes: 8 * gib},
		{Label: "fp16-tiled", AmountBytes: 3 * gib},
		{Label: "cpu", AmountBytes: 0},
	}}
	// 16 total, 14 used -> 2 GiB free. fp16(8) and fp16-tiled(3) don't fit; cpu(0) does.
	v := Decide(req, snapshotWith(16, 14), nil, DefaultPolicy(), now)
	if v.Kind != VerdictDegrade {
		t.Fatalf("kind = %q, want degrade (%s)", v.Kind, v.Reason)
	}
	if v.Step != "cpu" || v.GrantedBytes != 0 {
		t.Errorf("step = %q amount = %d, want cpu/0", v.Step, v.GrantedBytes)
	}
}

func TestDecideDegradePrefersHighestFittingStep(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	req := vramReq(8, 1, PriorityBatch)
	req.Profile = &DegradeProfile{Steps: []DegradeStep{
		{Label: "large", AmountBytes: 8 * gib},
		{Label: "medium", AmountBytes: 3 * gib},
		{Label: "small", AmountBytes: 1 * gib},
	}}
	// 5 GiB free -> medium(3) is the highest fitting step.
	v := Decide(req, snapshotWith(16, 11), nil, DefaultPolicy(), now)
	if v.Kind != VerdictDegrade || v.Step != "medium" {
		t.Fatalf("got %q/%q, want degrade/medium (%s)", v.Kind, v.Step, v.Reason)
	}
}

func TestDecideEffectiveUsedCountsLedgerCommitments(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	// Observed used is 0 but a granted claim commits 14 GiB that hasn't loaded
	// yet — the new request must see only 2 GiB free, not 16.
	ledger := []CapacityClaim{{
		ClaimID: "clm-resident", OwnerID: "whisper", ResourceKind: ResourceKindVRAM,
		GPUIndex: gpu(0), AmountBytes: 14 * gib, Status: StatusGranted, Priority: PriorityService,
	}}
	// floor==preferred==4 GiB doesn't fit the 2 GiB the commitment leaves, and
	// the resident claim is higher priority (service > batch) so unreclaimable.
	v := Decide(vramReq(4, 4, PriorityBatch), snapshotWith(16, 0), ledger, DefaultPolicy(), now)
	if v.Kind != VerdictDeny {
		t.Fatalf("kind = %q, want deny (committed should be counted) (%s)", v.Kind, v.Reason)
	}
}

func TestDecideReclaimsIdleLowerPriority(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	lastActive := now.Add(-10 * time.Minute)
	created := now.Add(-20 * time.Minute)
	// A lower-priority idle batch claim holds 8 GiB; the host is full of it.
	ledger := []CapacityClaim{{
		ClaimID: "clm-batch", OwnerID: "image-tools", ResourceKind: ResourceKindVRAM,
		GPUIndex: gpu(0), AmountBytes: 8 * gib, Status: StatusGranted, Priority: PriorityBatch,
		ActivityState: ActivityIdle, LastActiveAt: &lastActive, CreatedAt: created,
	}}
	// Interactive request needs 6 GiB; only 2 free observed, but reclaiming the
	// idle batch claim frees 8 more.
	req := vramReq(6, 1, PriorityInteractive)
	req.OwnerID = "whisper"
	v := Decide(req, snapshotWith(16, 14), ledger, DefaultPolicy(), now)
	if v.Kind != VerdictGrant {
		t.Fatalf("kind = %q, want grant via reclaim (%s)", v.Kind, v.Reason)
	}
	if len(v.ReclaimTargets) != 1 || v.ReclaimTargets[0] != "clm-batch" {
		t.Errorf("reclaim targets = %v, want [clm-batch]", v.ReclaimTargets)
	}
}

func TestDecideNeverReclaimsProtectedOrActive(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	lastActive := now.Add(-1 * time.Minute)
	// Lower priority but ACTIVE+protected -> must not be reclaimed.
	ledger := []CapacityClaim{{
		ClaimID: "clm-active", OwnerID: "image-tools", ResourceKind: ResourceKindVRAM,
		GPUIndex: gpu(0), AmountBytes: 8 * gib, Status: StatusGranted, Priority: PriorityBatch,
		ActivityState: ActivityActive, Protected: true, LastActiveAt: &lastActive,
	}}
	req := vramReq(6, 6, PriorityInteractive)
	v := Decide(req, snapshotWith(16, 14), ledger, DefaultPolicy(), now)
	// 2 free, can't reclaim the active claim, but it COULD be reclaimed if it
	// went idle -> queue, never grant.
	if v.Kind == VerdictGrant {
		t.Fatalf("must not grant by reclaiming a protected/active claim; got %+v", v)
	}
}

func TestDecideAgeAloneDoesNotReclaim(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	lastActive := now.Add(-3 * time.Hour) // very old, but currently ACTIVE
	ledger := []CapacityClaim{{
		ClaimID: "clm-old-active", OwnerID: "ollama", ResourceKind: ResourceKindVRAM,
		GPUIndex: gpu(0), AmountBytes: 8 * gib, Status: StatusGranted, Priority: PriorityBatch,
		ActivityState: ActivityActive, LastActiveAt: &lastActive,
	}}
	req := vramReq(6, 6, PriorityInteractive)
	v := Decide(req, snapshotWith(16, 14), ledger, DefaultPolicy(), now)
	if len(v.ReclaimTargets) != 0 {
		t.Errorf("age alone must not make an active claim reclaimable; targets = %v", v.ReclaimTargets)
	}
}

func TestDecideIdleGraceMustElapse(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	// Idle, but only just (5s < 60s default grace).
	lastActive := now.Add(-5 * time.Second)
	ledger := []CapacityClaim{{
		ClaimID: "clm-just-idle", OwnerID: "image-tools", ResourceKind: ResourceKindVRAM,
		GPUIndex: gpu(0), AmountBytes: 8 * gib, Status: StatusGranted, Priority: PriorityBatch,
		ActivityState: ActivityIdle, LastActiveAt: &lastActive,
	}}
	req := vramReq(6, 6, PriorityInteractive)
	v := Decide(req, snapshotWith(16, 14), ledger, DefaultPolicy(), now)
	if len(v.ReclaimTargets) != 0 {
		t.Errorf("claim within idle grace must not be reclaimable; targets = %v", v.ReclaimTargets)
	}
	if v.Kind != VerdictQueue {
		t.Errorf("kind = %q, want queue (will be reclaimable after grace) (%s)", v.Kind, v.Reason)
	}
}

func TestDecideClaimSpecificIdleGraceOverridesGlobalGrace(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	lastActive := now.Add(-2 * time.Minute)
	ledger := []CapacityClaim{{
		ClaimID: "clm-warm-stt", OwnerID: "kyutai-stt", ResourceKind: ResourceKindVRAM,
		GPUIndex: gpu(0), AmountBytes: 3 * gib, Status: StatusGranted, Priority: PriorityService,
		ActivityState: ActivityIdle, LastActiveAt: &lastActive, CreatedAt: now.Add(-30 * time.Minute),
		YieldWhenIdle: true, IdleGrace: 15 * time.Minute,
	}}
	req := vramReq(6, 6, PriorityService)
	v := Decide(req, snapshotWith(16, 13), ledger, DefaultPolicy(), now)
	if len(v.ReclaimTargets) != 0 {
		t.Fatalf("warm idle claim should not be reclaimable yet; targets = %v", v.ReclaimTargets)
	}
	if v.Kind != VerdictQueue {
		t.Fatalf("kind = %q, want queue until claim-specific idle grace elapses (%s)", v.Kind, v.Reason)
	}

	cold := ledger
	cold[0].LastActiveAt = ptrTime(now.Add(-16 * time.Minute))
	v = Decide(req, snapshotWith(16, 13), cold, DefaultPolicy(), now)
	if v.Kind != VerdictGrant {
		t.Fatalf("kind = %q, want grant after claim-specific idle grace (%s)", v.Kind, v.Reason)
	}
	if len(v.ReclaimTargets) != 1 || v.ReclaimTargets[0] != "clm-warm-stt" {
		t.Fatalf("reclaim targets = %v, want [clm-warm-stt]", v.ReclaimTargets)
	}
}

func TestDecideQueueVsDeny(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	// Host full with a SAME-priority claim (can never be reclaimed) -> deny.
	ledger := []CapacityClaim{{
		ClaimID: "clm-peer", OwnerID: "kyutai-stt", ResourceKind: ResourceKindVRAM,
		GPUIndex: gpu(0), AmountBytes: 15 * gib, Status: StatusGranted, Priority: PriorityInteractive,
		ActivityState: ActivityActive,
	}}
	req := vramReq(6, 6, PriorityInteractive)
	v := Decide(req, snapshotWith(16, 15), ledger, DefaultPolicy(), now)
	if v.Kind != VerdictDeny {
		t.Fatalf("kind = %q, want deny (peer priority, nothing reclaimable) (%s)", v.Kind, v.Reason)
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestDecideNoGPUDenies(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	v := Decide(vramReq(4, 1, PriorityBatch), hostinventory.Snapshot{}, nil, DefaultPolicy(), now)
	if v.Kind != VerdictDeny {
		t.Errorf("kind = %q, want deny when no GPU present", v.Kind)
	}
}

func TestDecideCPUIsAdvisoryGrant(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	req := CapacityRequest{OwnerID: "x", ResourceKind: ResourceKindCPU, PreferredBytes: 0}
	v := Decide(req, hostinventory.Snapshot{}, nil, DefaultPolicy(), now)
	if v.Kind != VerdictGrant {
		t.Errorf("cpu kind = %q, want grant (advisory)", v.Kind)
	}
	if len(v.Warnings) == 0 {
		t.Error("cpu grant should warn it is not enforced in V1")
	}
}

func TestDecideRejectsFloorAbovePreferred(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	req := vramReq(2, 8, PriorityBatch) // floor 8 > preferred 2
	v := Decide(req, snapshotWith(16, 0), nil, DefaultPolicy(), now)
	if v.Kind != VerdictDeny {
		t.Errorf("kind = %q, want deny for floor > preferred", v.Kind)
	}
}

func TestDecideGrantedHelper(t *testing.T) {
	if !(Verdict{Kind: VerdictGrant}).Granted() || !(Verdict{Kind: VerdictDegrade}).Granted() {
		t.Error("grant/degrade should report Granted()")
	}
	if (Verdict{Kind: VerdictQueue}).Granted() || (Verdict{Kind: VerdictDeny}).Granted() {
		t.Error("queue/deny should not report Granted()")
	}
}

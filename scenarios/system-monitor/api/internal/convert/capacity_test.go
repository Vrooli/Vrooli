package convert

import (
	"reflect"
	"testing"

	engine "github.com/vrooli/vrooli/internal/capacity"
	capacitypb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/capacity"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestCapacityFindingToProtoPreservesAggregateWorkerPIDsOnWire(t *testing.T) {
	gpu := 1
	finding := engine.Finding{
		Class: "over_claim", OwnerID: "music", OwnerKind: "resource",
		ResourceKind: "vram", GPUIndex: &gpu, PID: 0, PIDs: []int{231, 587},
		ProcessName: "workers", ObservedBytes: 900, ClaimedBytes: 800,
		ClaimID: "claim-1", Severity: "warn", Message: "aggregate workers exceed claim",
	}
	got := CapacityFindingToProto(finding)
	wire, err := protojson.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var decoded capacitypb.CapacityFinding
	if err := protojson.Unmarshal(wire, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded.GetPids(), []int32{231, 587}) {
		t.Fatalf("wire worker PIDs = %v, want both workers; JSON=%s", decoded.GetPids(), wire)
	}
	if decoded.GetPid() != 0 || decoded.GpuIndex == nil || decoded.GetGpuIndex() != 1 {
		t.Fatalf("aggregate identity lost: pid=%d gpu=%v", decoded.GetPid(), decoded.GpuIndex)
	}
	if decoded.GetClaimId() != finding.ClaimID || decoded.GetObservedBytes() != 900 || decoded.GetClaimedBytes() != 800 {
		t.Fatalf("aggregate claim/amounts lost: %v", &decoded)
	}
	finding.PIDs[0] = 999
	if got.GetPids()[0] != 231 {
		t.Fatal("converted worker list must remain an independent snapshot")
	}
}

func TestCapacityFindingToProtoRetainsSingleAndLegacyProcessIdentity(t *testing.T) {
	for _, tc := range []struct {
		name string
		pids []int
	}{
		{name: "single worker", pids: []int{991}},
		{name: "legacy singular only"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := CapacityFindingToProto(engine.Finding{PID: 991, PIDs: tc.pids, ProcessName: "worker"})
			if got.GetPid() != 991 || got.GetProcessName() != "worker" || got.GpuIndex != nil {
				t.Fatalf("single/legacy identity changed: %v", got)
			}
			if len(got.GetPids()) != len(tc.pids) || len(tc.pids) > 0 && got.GetPids()[0] != 991 {
				t.Fatalf("worker PIDs = %v, want %v", got.GetPids(), tc.pids)
			}
		})
	}
}

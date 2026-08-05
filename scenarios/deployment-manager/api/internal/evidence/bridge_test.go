package evidence

import (
	"testing"

	"deployment-manager/crossosgate"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestVerdictFromBridgeMapsDispositionAndTarget(t *testing.T) {
	result, err := VerdictFromBridge(crossosgate.OSResult{OS: "linux", RunID: "bridge-run", NodeID: "node-1", Disposition: "success", Detail: "smoke passed"}, "desktop", "linux", "")
	if err != nil {
		t.Fatal(err)
	}
	if result.RunId != "bridge-run" || result.Disposition != commonv1.Disposition_DISPOSITION_PASSED || result.Target.GetDeviceKind() != commonv1.DeviceKind_DEVICE_KIND_HOST {
		t.Fatalf("mapped verdict = %+v", result)
	}
	if result.Target.GetBridgeNodeId() != "node-1" || result.Target.GetBridgeJobId() != "bridge-run" {
		t.Fatalf("bridge references = %+v", result.Target)
	}
}

func TestVerdictFromBridgeRejectsIncompleteAndUnknownResults(t *testing.T) {
	if _, err := VerdictFromBridge(crossosgate.OSResult{Disposition: "passed"}, "desktop", "linux", "run"); err == nil {
		t.Fatal("missing OS was accepted")
	}
	if _, err := VerdictFromBridge(crossosgate.OSResult{OS: "linux", Disposition: "mystery"}, "desktop", "linux", "run"); err == nil {
		t.Fatal("unknown disposition was accepted")
	}
	if _, err := VerdictFromBridge(crossosgate.OSResult{OS: "linux", Disposition: "passed"}, "desktop", "linux", ""); err == nil {
		t.Fatal("missing run ID was accepted")
	}
	for _, disposition := range []string{"pending", "failed", "skipped"} {
		result, err := VerdictFromBridge(crossosgate.OSResult{OS: "linux", RunID: "run", Disposition: disposition}, "desktop", "linux", "explicit")
		if err != nil || result.RunId != "explicit" {
			t.Fatalf("disposition %s = %+v, %v", disposition, result, err)
		}
	}
}

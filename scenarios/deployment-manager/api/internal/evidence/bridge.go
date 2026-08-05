package evidence

import (
	"fmt"
	"strings"

	"deployment-manager/crossosgate"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// VerdictFromBridge maps bridge's reach-plane OS result into the shared
// evidence contract. The bridge is a host-side producer, so its device kind is
// always DEVICE_KIND_HOST; emulator and physical producers report directly.
func VerdictFromBridge(result crossosgate.OSResult, ramp, platform, runID string) (*commonv1.TargetVerdict, error) {
	if strings.TrimSpace(result.OS) == "" || strings.TrimSpace(ramp) == "" || strings.TrimSpace(platform) == "" {
		return nil, fmt.Errorf("bridge result requires os, ramp, and platform")
	}
	if strings.TrimSpace(runID) == "" {
		runID = result.RunID
	}
	if strings.TrimSpace(runID) == "" {
		return nil, fmt.Errorf("bridge result requires run_id")
	}
	var disposition commonv1.Disposition
	switch strings.ToLower(result.Disposition) {
	case "pending":
		disposition = commonv1.Disposition_DISPOSITION_PENDING
	case "passed", "pass", "success":
		disposition = commonv1.Disposition_DISPOSITION_PASSED
	case "failed", "fail", "error":
		disposition = commonv1.Disposition_DISPOSITION_FAILED
	case "skipped":
		disposition = commonv1.Disposition_DISPOSITION_SKIPPED
	default:
		return nil, fmt.Errorf("unsupported bridge disposition %q", result.Disposition)
	}
	target := &commonv1.EvidenceTarget{
		Ramp: ramp, Platform: platform, Os: result.OS,
		DeviceKind: commonv1.DeviceKind_DEVICE_KIND_HOST,
	}
	if result.NodeID != "" {
		target.BridgeNodeId = &result.NodeID
	}
	if result.RunID != "" {
		target.BridgeJobId = &result.RunID
	}
	return &commonv1.TargetVerdict{Target: target, Disposition: disposition, RunId: runID, Detail: result.Detail}, nil
}

package hostbroker

import "testing"

func TestAdmissionRequestIsPinnedToBridgePolicy(t *testing.T) {
	req := AdmissionRequest("bridge.ufw.allow", "r-1", "192.168.1.176")
	if req.Version != "v1" || req.Subject.Scenario != "vrooli-bridge" || req.Subject.Port != 18767 || req.Subject.CandidateIP != "192.168.1.176" {
		t.Fatalf("request=%+v", req)
	}
}

package control

import (
	"testing"
	"time"

	"device-control/internal/evidence"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestAndroidSelfTestPassRequiresEvidenceProvenance(t *testing.T) {
	result := AndroidCapabilitySelfTestResult{Disposition: "passed", EvidenceClass: "release-grade", RunID: "run-1"}
	finalizeAndroidSelfTest(&result, nil)
	if result.Disposition != "failed" || result.Reason == "" {
		t.Fatalf("pass without evidence provenance was retained: %+v", result)
	}
	verdict := conformanceVerdict(result, nil)
	if verdict.GetDisposition() != commonv1.Disposition_DISPOSITION_FAILED {
		t.Fatalf("missing evidence provenance must produce a failed verdict: %+v", verdict)
	}

	result = AndroidCapabilitySelfTestResult{Disposition: "passed", EvidenceClass: "release-grade", RunID: "run-1"}
	refs := []evidence.Reference{{ID: "artifact", Producer: "device-control", Kind: "log", Checksum: "sha256:test", CreatedAt: time.Now().UTC()}}
	finalizeAndroidSelfTest(&result, refs)
	if result.Disposition != "passed" {
		t.Fatalf("complete evidence provenance was downgraded: %+v", result)
	}
	verdict = conformanceVerdict(result, refs)
	if verdict.GetDisposition() != commonv1.Disposition_DISPOSITION_PASSED {
		t.Fatalf("complete evidence provenance must preserve a pass: %+v", verdict)
	}
}

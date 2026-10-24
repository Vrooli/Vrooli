package deliveryramp

import (
	"strings"
	"testing"
	"time"
)

func TestDistributionResultValidateRequiresReceiptForPass(t *testing.T) {
	result := DistributionResult{Disposition: DispositionPass}
	if err := result.Validate(); err == nil || !strings.Contains(err.Error(), "effect receipt") {
		t.Fatalf("pass without receipt error = %v", err)
	}
}

func TestDistributionResultValidateAllowsCapabilityOnlyDegradedResult(t *testing.T) {
	result := DistributionResult{
		Disposition:     DispositionDegraded,
		CapabilityReady: true,
		Reason:          "toolchain is ready; external publication was not performed",
		Targets:         []DistributionTarget{{ID: "testflight", Kind: "store", Available: true}},
	}
	if err := result.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestDistributionResultValidateBindsReceiptToDeclaredTarget(t *testing.T) {
	result := DistributionResult{
		Disposition: DispositionDegraded,
		Reason:      "catalog registration completed; payload verification is pending",
		Targets:     []DistributionTarget{{ID: "catalog", Kind: "catalog", Available: true}},
		EffectReceipt: &DistributionEffectReceipt{
			TargetID:        "other-target",
			ArtifactRef:     "artifact:1",
			ExternalReceipt: "owner:1",
			Outcome:         "registered",
			ObservedAt:      time.Now().UTC(),
		},
	}
	if err := result.Validate(); err == nil || !strings.Contains(err.Error(), "not declared") {
		t.Fatalf("receipt target error = %v", err)
	}
}

func TestDistributionResultValidateRejectsPartialPass(t *testing.T) {
	result := DistributionResult{
		Disposition: DispositionPass,
		Targets: []DistributionTarget{
			{ID: "linux", Kind: "desktop", Available: true},
			{ID: "windows", Kind: "desktop", Available: false, Reason: "Windows signing authority is unavailable"},
		},
		EffectReceipt: &DistributionEffectReceipt{
			TargetID:        "linux",
			ArtifactRef:     "artifact:1",
			ExternalReceipt: "owner:1",
			Outcome:         "published",
			ObservedAt:      time.Now().UTC(),
		},
	}
	if err := result.Validate(); err == nil || !strings.Contains(err.Error(), "unavailable target") {
		t.Fatalf("partial pass error = %v", err)
	}
}

func TestDistributionRequestValidateResultBindsArtifactIdentity(t *testing.T) {
	request := DistributionRequest{Artifact: Artifact{ImmutableRef: "artifact:approved"}}
	result := DistributionResult{
		Disposition: DispositionPass,
		EffectReceipt: &DistributionEffectReceipt{
			TargetID:        "catalog",
			ArtifactRef:     "artifact:other",
			ExternalReceipt: "owner:1",
			Outcome:         "published",
			ObservedAt:      time.Now().UTC(),
		},
	}
	if err := request.ValidateResult(result); err == nil || !strings.Contains(err.Error(), "does not match requested artifact") {
		t.Fatalf("artifact binding error = %v", err)
	}
}

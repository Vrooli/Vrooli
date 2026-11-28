package assessment

import (
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/proto"
)

func TestPhasePresentationIsDeterministicAndRollsUpRepeatedCodes(t *testing.T) {
	spec := validMultiCapabilitySpec()
	forward, err := BuildProtoAssessment(BuildInput{
		Scenario: "demo",
		Spec:     spec,
		Findings: []Finding{
			{Code: "interop.proxy_base_missing", Location: "ui/a.ts", Title: "Missing proxy", Message: "Configure a proxy."},
			{Code: "pwa.service_worker_missing", Location: "ui/sw.ts", Title: "No service worker", Message: "Register one.", AutofixAvailable: true},
			{Code: "interop.proxy_base_missing", Location: "ui/b.ts", Title: "Missing proxy", Message: "Configure a proxy."},
		},
	})
	if err != nil {
		t.Fatalf("BuildProtoAssessment forward: %v", err)
	}
	reversed, err := BuildProtoAssessment(BuildInput{
		Scenario: "demo",
		Spec:     spec,
		Findings: []Finding{
			{Code: "interop.proxy_base_missing", Location: "ui/b.ts", Title: "Missing proxy", Message: "Configure a proxy."},
			{Code: "pwa.service_worker_missing", Location: "ui/sw.ts", Title: "No service worker", Message: "Register one.", AutofixAvailable: true},
			{Code: "interop.proxy_base_missing", Location: "ui/a.ts", Title: "Missing proxy", Message: "Configure a proxy."},
		},
	})
	if err != nil {
		t.Fatalf("BuildProtoAssessment reversed: %v", err)
	}
	if !proto.Equal(forward.GetPresentation(), reversed.GetPresentation()) {
		t.Fatalf("presentation depends on raw finding order:\nforward=%+v\nreversed=%+v", forward.GetPresentation(), reversed.GetPresentation())
	}
	presentation := forward.GetPresentation()
	if presentation.GetContractVersion() != PhasePresentationContractVersion {
		t.Fatalf("contract version = %q", presentation.GetContractVersion())
	}
	if len(presentation.GetCapabilities()) != 2 || presentation.GetCapabilities()[0].GetId() != "pwa_native_readiness" {
		t.Fatalf("capability priority order = %+v", presentation.GetCapabilities())
	}
	interop := presentation.GetCapabilities()[1]
	if len(interop.GetFindings()) != 1 || interop.GetFindings()[0].GetCount() != 2 {
		t.Fatalf("repeated code rollup = %+v", interop.GetFindings())
	}
	pwa := presentation.GetCapabilities()[0]
	if pwa.GetFindings()[0].GetFixAffordance().String() != "FIX_AFFORDANCE_PREVIEW_AVAILABLE" {
		t.Fatalf("autofix affordance = %s", pwa.GetFindings()[0].GetFixAffordance())
	}
	if err := ValidatePhasePresentation(forward); err != nil {
		t.Fatalf("ValidatePhasePresentation: %v", err)
	}
}

func TestValidatePhasePresentationRejectsMissingOrModifiedProjection(t *testing.T) {
	assessment, err := BuildProtoAssessment(BuildInput{Scenario: "demo", Spec: validSpec()})
	if err != nil {
		t.Fatalf("BuildProtoAssessment: %v", err)
	}
	assessment.Presentation = nil
	if err := ValidatePhasePresentation(assessment); err == nil {
		t.Fatal("missing presentation unexpectedly validated")
	}
	assessment.Presentation = BuildPhasePresentation(assessment)
	assessment.Presentation.NextAction = "invented by consumer"
	if err := ValidatePhasePresentation(assessment); err == nil {
		t.Fatal("modified presentation unexpectedly validated")
	}
}

func TestRequireProviderContractEnforcesIdentityAndPresentation(t *testing.T) {
	assessment, err := BuildProtoAssessment(BuildInput{Scenario: "demo", Spec: validSpec()})
	if err != nil {
		t.Fatalf("BuildProtoAssessment: %v", err)
	}
	provider, phase := assessment.GetProvider(), assessment.GetPhase()
	if err := RequireProviderContract(provider, phase, assessment); err != nil {
		t.Fatalf("conforming assessment rejected: %v", err)
	}
	if err := RequireProviderContract("other-provider", phase, assessment); err == nil {
		t.Fatal("identity mismatch unexpectedly validated")
	}
	assessment.Presentation = nil
	if err := RequireProviderContract(provider, phase, assessment); err == nil {
		t.Fatal("missing presentation unexpectedly validated")
	}
}

func TestPhasePresentationFixAffordancesAreTruthful(t *testing.T) {
	spec := validSpec()
	mapping := spec.Findings["measures.uncovered-domain"]
	mapping.FixClass = FixClassManual
	mapping.FixReason = "requires product judgment"
	spec.Findings["measures.uncovered-domain"] = mapping
	manual, err := BuildProtoAssessment(BuildInput{Scenario: "demo", Spec: spec, Findings: []Finding{{Code: "measures.uncovered-domain"}}})
	if err != nil {
		t.Fatalf("BuildProtoAssessment manual: %v", err)
	}
	if got := manual.GetPresentation().GetCapabilities()[0].GetFindings()[0].GetFixAffordance(); got != commonv1.FixAffordance_FIX_AFFORDANCE_MANUAL {
		t.Fatalf("manual affordance = %s", got)
	}

	detectionSpec := validSpec()
	detectionMapping := detectionSpec.Findings["measures.uncovered-domain"]
	detectionMapping.FixClass = FixClassAuto
	detectionMapping.FixerStatus = FixerStatusPending
	detectionSpec.Findings["measures.uncovered-domain"] = detectionMapping
	detection, err := BuildProtoAssessment(BuildInput{Scenario: "demo", Spec: detectionSpec, Findings: []Finding{{Code: "measures.uncovered-domain"}}})
	if err != nil {
		t.Fatalf("BuildProtoAssessment detection-only: %v", err)
	}
	if got := detection.GetPresentation().GetCapabilities()[0].GetFindings()[0].GetFixAffordance(); got != commonv1.FixAffordance_FIX_AFFORDANCE_DETECTION_ONLY {
		t.Fatalf("detection-only affordance = %s", got)
	}
}

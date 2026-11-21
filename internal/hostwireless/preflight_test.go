package hostwireless

import (
	"context"
	"os"
	"testing"
)

func TestPreflightRejectsMissingIdentityBeforeObservation(t *testing.T) {
	for _, tc := range []struct{ bundle, dir, cert string }{{}, {bundle: "reviewed", dir: "signed"}} {
		if _, err := PreflightActivation(context.Background(), NobleProfile, tc.bundle, "", tc.dir, tc.cert); err == nil {
			t.Fatal("accepted incomplete identity")
		}
	}
}

func TestPreflightDoesNotPromoteSignedBytesOrEnrollmentToLiveReadiness(t *testing.T) {
	r := ActivationPreflight{Plan: ActivationPlan{KernelRelease: nobleABI, RecoveryRelease: "7.0.0-29-generic", Preserved: []FileIdentity{{Path: "/boot/grub/grub.cfg", Issue: "permission denied"}}},
		SignedCopy: &SignedCopyEvidence{SignaturesVerified: true}, SigningTrust: &SigningTrust{Enrollment: "enrolled", KernelKeyring: "subject_key_identifier_found"},
		Firmware: []FirmwarePreflight{{CandidateVerified: true}}}
	steps := activationSteps(r)
	want := map[string]string{"stock_recovery": "protected_identity_unavailable", "sign": "authenticated_copies_observed", "kernel_trust": "independent_native_proof_required", "firmware": "verified_candidates_require_transaction", "stage_and_rollback": "privileged_adapter_and_live_qualification_required", "activate": "coordinated_device_effect_required"}
	for _, step := range steps {
		if expected, ok := want[step.Name]; ok && step.State != expected {
			t.Errorf("%s=%s, want %s", step.Name, step.State, expected)
		}
	}
	if r.Plan.ActivationEligible {
		t.Fatal("preflight conferred live authority")
	}
}

func TestNativeReadOnlySigningTrust(t *testing.T) {
	certificate := os.Getenv("VROOLI_WIRELESS_PUBLIC_CERT_TEST")
	if certificate == "" {
		t.Skip("explicit public certificate required")
	}
	r, err := InspectSigningTrust(context.Background(), certificate)
	if err != nil {
		t.Fatal(err)
	}
	if !validSHA(r.CertificateSHA256) {
		t.Fatal("missing public certificate identity")
	}
	t.Logf("public trust observation: %+v", r)
}

package releases

import "testing"

func TestRecoveryExpectedBundleSHAUsesLatestOwnerEffect(t *testing.T) {
	release := &Release{
		DeploymentID:   "dep-1",
		ArtifactDigest: "sha256:reviewed-manifest",
		PublicationReceipts: []PublicationReceipt{
			{TargetID: "cloud:demo", Producer: "scenario-to-cloud", ArtifactDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		},
		RecoveryReceipts: []RecoveryReceipt{
			{DeploymentID: "dep-1", BundleSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		},
	}
	if got := recoveryExpectedBundleSHA(release); got != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Fatalf("latest recovery bundle = %q", got)
	}

	release.RecoveryReceipts = nil
	if got := recoveryExpectedBundleSHA(release); got != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("publication bundle = %q", got)
	}

	release.PublicationReceipts = nil
	if got := recoveryExpectedBundleSHA(release); got != "sha256:reviewed-manifest" {
		t.Fatalf("legacy fallback bundle = %q", got)
	}
}

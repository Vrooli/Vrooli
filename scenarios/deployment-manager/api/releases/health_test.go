package releases

import (
	"context"
	"testing"
	"time"
)

func TestAssessHealthRequiresDurablePublicationEvidence(t *testing.T) {
	created := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	release := &Release{
		ID: "release-1", Status: StatusPublished, CreatedAt: created,
		PublishedAt: timePtr(created.Add(90 * time.Second)),
		Platforms:   []ReleasePlatform{{Platform: "linux", Status: PlatformStatusPublished}},
	}
	withoutReceipt := AssessHealth(release, created.Add(2*time.Minute))
	if withoutReceipt.PublicationVerified || withoutReceipt.Status == "healthy" {
		t.Fatalf("published release without receipt was healthy: %+v", withoutReceipt)
	}
	release.PublicationReceipts = []PublicationReceipt{{CandidateID: "candidate-1", DestinationRevisionID: "destination-1", TargetID: "linux", ArtifactDigest: "sha512:artifact", DestinationObject: "lpbs://stable/linux/1", Producer: "lpbs", ExternalReceipt: "publish-1", Outcome: ReceiptVerified, ObservedAt: created.Add(90 * time.Second)}}
	withReceipt := AssessHealth(release, created.Add(2*time.Minute))
	if !withReceipt.PublicationVerified || withReceipt.Status != "attention" {
		t.Fatalf("verified release standing = %+v", withReceipt)
	}
	release.VerificationEvidence = []VerificationItem{{Platform: "linux", Match: true, SHA512Match: true, CheckedAt: created.Add(2 * time.Minute)}}
	withClientReceipt := AssessHealth(release, created.Add(2*time.Minute))
	if withClientReceipt.Status != "healthy" || len(withClientReceipt.Alerts) != 0 {
		t.Fatalf("release with fresh client evidence = %+v", withClientReceipt)
	}
}

func TestAssessHealthRequiresTrustedClientUpdateReceiptPerTarget(t *testing.T) {
	created := time.Now().UTC().Add(-4 * time.Minute)
	release := &Release{
		ID: "release-client-receipts", CandidateID: "candidate-1", Status: StatusPublished, CreatedAt: created,
		PublishedAt: timePtr(created.Add(time.Minute)),
		Platforms:   []ReleasePlatform{{Platform: "linux-x64", Status: PlatformStatusPublished}, {Platform: "windows-x64", Status: PlatformStatusPublished}},
		PublicationReceipts: []PublicationReceipt{
			{CandidateID: "candidate-1", DestinationRevisionID: "destination-1", TargetID: "linux-x64", ArtifactDigest: "sha256:a", DestinationObject: "lpbs://linux", Producer: "lpbs", ExternalReceipt: "pub-linux", Outcome: ReceiptVerified, ObservedAt: created.Add(90 * time.Second)},
			{CandidateID: "candidate-1", DestinationRevisionID: "destination-1", TargetID: "windows-x64", ArtifactDigest: "sha256:b", DestinationObject: "lpbs://windows", Producer: "lpbs", ExternalReceipt: "pub-windows", Outcome: ReceiptVerified, ObservedAt: created.Add(90 * time.Second)},
		},
		ClientUpdateReceipts: []ClientUpdateReceipt{{CandidateID: "candidate-1", PredecessorRef: "desktop-0.9", SuccessorDigest: "sha256:a", TargetID: "linux-x64", VerifiedVersion: "1.0.0", Outcome: ReceiptVerified, Producer: "scenario-to-desktop", ExternalReceipt: "update-linux", ObservedAt: created.Add(3 * time.Minute)}},
	}
	health := AssessHealth(release, created.Add(4*time.Minute))
	if health.ClientUpdatesHealthy {
		t.Fatalf("client updates unexpectedly healthy: %+v", health)
	}
	if !hasReleaseAlert(health.Alerts, "missing_client_update_receipt", "windows-x64") {
		t.Fatalf("missing target alert not found: %+v", health.Alerts)
	}
	release.ClientUpdateReceipts = append(release.ClientUpdateReceipts, ClientUpdateReceipt{CandidateID: "candidate-1", PredecessorRef: "desktop-0.9", SuccessorDigest: "sha256:b", TargetID: "windows-x64", VerifiedVersion: "1.0.0", Outcome: ReceiptVerified, Producer: "scenario-to-desktop", ExternalReceipt: "update-windows", ObservedAt: created.Add(3 * time.Minute)})
	health = AssessHealth(release, created.Add(4*time.Minute))
	if !health.ClientUpdatesHealthy || health.Status != "healthy" {
		t.Fatalf("client updates remained unhealthy: %+v", health)
	}
	release.ClientUpdateReceipts[1].SuccessorDigest = "sha256:wrong"
	health = AssessHealth(release, created.Add(4*time.Minute))
	if health.ClientUpdatesHealthy || !hasReleaseAlert(health.Alerts, "missing_client_update_receipt", "windows-x64") {
		t.Fatalf("mismatched successor digest was trusted: %+v", health)
	}
}

func TestAssessHealthRequiresLegacyVerificationEvidencePerTarget(t *testing.T) {
	created := time.Now().UTC().Add(-4 * time.Minute)
	release := &Release{
		ID: "release-legacy-evidence", Status: StatusPublished, CreatedAt: created,
		PublishedAt: timePtr(created.Add(time.Minute)),
		Platforms:   []ReleasePlatform{{Platform: "linux-x64", Status: PlatformStatusPublished}, {Platform: "windows-x64", Status: PlatformStatusPublished}},
		PublicationReceipts: []PublicationReceipt{
			{TargetID: "linux-x64", ArtifactDigest: "sha256:a", DestinationObject: "lpbs://linux", Producer: "lpbs", ExternalReceipt: "pub-linux", Outcome: ReceiptVerified, ObservedAt: created.Add(90 * time.Second)},
			{TargetID: "windows-x64", ArtifactDigest: "sha256:b", DestinationObject: "lpbs://windows", Producer: "lpbs", ExternalReceipt: "pub-windows", Outcome: ReceiptVerified, ObservedAt: created.Add(90 * time.Second)},
		},
		VerificationEvidence: []VerificationItem{{Platform: "linux-x64", Match: true, SHA512Match: true, CheckedAt: created.Add(3 * time.Minute)}},
	}
	health := AssessHealth(release, created.Add(4*time.Minute))
	if health.ClientUpdatesHealthy {
		t.Fatalf("legacy verification evidence for one target was trusted for all targets: %+v", health)
	}
	if !hasReleaseAlert(health.Alerts, "missing_client_update_receipt", "windows-x64") {
		t.Fatalf("missing target alert not found: %+v", health.Alerts)
	}
}

func hasReleaseAlert(alerts []ReleaseAlert, code, target string) bool {
	for _, alert := range alerts {
		if alert.Code == code && alert.Target == target {
			return true
		}
	}
	return false
}

func TestAssessHealthMarksAmbiguousAndUnsupportedControls(t *testing.T) {
	health := AssessHealth(&Release{ID: "release-ambiguous", Status: StatusAmbiguous}, time.Now())
	if health.Status != "attention" || len(health.Alerts) == 0 {
		t.Fatalf("ambiguous health = %+v", health)
	}
	if len(health.UnsupportedControls) != 1 || len(health.SupportedControls) != 1 {
		t.Fatalf("unsupported controls = %+v", health.UnsupportedControls)
	}
}

func TestHealthDurationMeasurementsUseReceiptTimestamps(t *testing.T) {
	created := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	release := &Release{
		ID: "release-timing", Status: StatusPublished, CreatedAt: created,
		PublishedAt: timePtr(created.Add(90 * time.Second)),
		Platforms:   []ReleasePlatform{{Platform: "linux", Status: PlatformStatusPublished}},
		PublicationReceipts: []PublicationReceipt{{
			CandidateID: "candidate-1", DestinationRevisionID: "destination-1", TargetID: "linux", ArtifactDigest: "sha512:artifact", DestinationObject: "lpbs://stable/linux/1",
			Producer: "lpbs", ExternalReceipt: "publish-1", Outcome: ReceiptVerified,
			ObservedAt: created.Add(120 * time.Second),
		}},
		VerificationEvidence: []VerificationItem{{
			Platform: "linux", Match: true, SHA512Match: true, CheckedAt: created.Add(180 * time.Second),
		}},
	}

	health := AssessHealth(release, created.Add(4*time.Minute))
	if !health.PublicationVerified || !health.ClientUpdatesHealthy {
		t.Fatalf("health = %+v", health)
	}
	(&Handler{}).addHealthDurationMeasurements(context.Background(), release, &health)
	if got := health.KnownDurationsMillis["promote_to_publicly_verified"]; got != 30000 {
		t.Fatalf("promote_to_publicly_verified = %d", got)
	}
	if got := health.KnownDurationsMillis["update_to_healthy"]; got != 90000 {
		t.Fatalf("update_to_healthy = %d", got)
	}
}

func timePtr(value time.Time) *time.Time { return &value }

package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"landing-page-business-suite-api/internal/delivery"
)

func TestChannelPromotionUsesPredecessorCASAndRetainsHistory(t *testing.T) {
	db := setupTestDB(t)
	const bundle, app, variant = "channel_revision_bundle", "desktop", "stable"
	_, err := db.Exec(`
		INSERT INTO download_apps (bundle_key, app_key, name) VALUES ($1,$2,'Desktop')
	`, bundle, app)
	if err != nil {
		t.Fatalf("insert app: %v", err)
	}

	artifactID := func(platform, version string) int64 {
		t.Helper()
		var id int64
		err := db.QueryRow(`
			INSERT INTO download_artifacts
				(bundle_key, app_key, provider, bucket, object_key, etag, size_bytes, sha512, platform, release_version)
			VALUES ($1,$2,'s3','bucket',$3,$4,10,$5,$6,$7)
			RETURNING id
		`, bundle, app, "releases/"+version+"/"+platform, version+"-etag", version+"-digest", platform, version).Scan(&id)
		if err != nil {
			t.Fatalf("insert artifact: %v", err)
		}
		return id
	}
	first := artifactID("windows", "1.0.0")
	second := artifactID("windows", "2.0.0")
	_, err = db.Exec(`
		INSERT INTO download_assets
			(bundle_key, app_key, platform, variant_key, artifact_source, artifact_id, release_version, release_notes, checksum)
		VALUES ($1,$2,'windows',$3,'managed',$4,'1.0.0','', '')
	`, bundle, app, variant, first)
	if err != nil {
		t.Fatalf("insert asset: %v", err)
	}

	catalog := delivery.NewCatalogService(db)
	firstRevision, err := catalog.PromoteChannel(context.Background(), delivery.ChannelPromotionRequest{
		BundleKey: bundle, AppKey: app, VariantKey: variant, ExpectedRevision: 0,
		ArtifactIDs: map[string]int64{"windows": first},
	})
	if err != nil {
		t.Fatalf("first promotion: %v", err)
	}
	if firstRevision.Revision != 1 || firstRevision.PredecessorRevision != 0 {
		t.Fatalf("unexpected first revision: %+v", firstRevision)
	}

	if _, err := catalog.PromoteChannel(context.Background(), delivery.ChannelPromotionRequest{
		BundleKey: bundle, AppKey: app, VariantKey: variant, ExpectedRevision: 0,
		ArtifactIDs: map[string]int64{"windows": second},
	}); !errors.Is(err, delivery.ErrChannelRevisionConflict) {
		t.Fatalf("expected stale promotion conflict, got %v", err)
	}

	secondRevision, err := catalog.PromoteChannel(context.Background(), delivery.ChannelPromotionRequest{
		BundleKey: bundle, AppKey: app, VariantKey: variant, ExpectedRevision: 1,
		ArtifactIDs: map[string]int64{"windows": second},
	})
	if err != nil {
		t.Fatalf("second promotion: %v", err)
	}
	if secondRevision.Revision != 2 || secondRevision.PredecessorRevision != 1 {
		t.Fatalf("unexpected second revision: %+v", secondRevision)
	}

	asset, err := catalog.GetAssetByVariant(bundle, app, "windows", variant)
	if err != nil {
		t.Fatalf("get promoted asset: %v", err)
	}
	if asset.ArtifactID == nil || *asset.ArtifactID != second {
		t.Fatalf("lookup did not consume channel head: %+v", asset.ArtifactID)
	}

	var revisionCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM download_channel_revisions WHERE bundle_key = $1`, bundle).Scan(&revisionCount); err != nil {
		t.Fatalf("count revisions: %v", err)
	}
	if revisionCount != 2 {
		t.Fatalf("expected immutable revision history, got %d rows", revisionCount)
	}

	var current int64
	if err := db.QueryRow(`SELECT current_revision FROM download_channel_heads WHERE bundle_key = $1`, bundle).Scan(&current); err != nil {
		t.Fatalf("read channel head: %v", err)
	}
	if current != 2 {
		t.Fatalf("current revision=%d, want 2", current)
	}
}

func TestChannelPromotionRejectsArtifactWithoutSHA512Identity(t *testing.T) {
	db := setupTestDB(t)
	const bundle, app, variant = "channel_missing_digest_bundle", "desktop", "stable"
	if _, err := db.Exec(`INSERT INTO download_apps (bundle_key, app_key, name) VALUES ($1,$2,'Desktop')`, bundle, app); err != nil {
		t.Fatalf("insert app: %v", err)
	}
	var artifactID int64
	if err := db.QueryRow(`
		INSERT INTO download_artifacts (bundle_key, app_key, provider, bucket, object_key, platform, release_version)
		VALUES ($1,$2,'s3','bucket','missing-digest.zip','windows','1.0.0') RETURNING id
	`, bundle, app).Scan(&artifactID); err != nil {
		t.Fatalf("insert artifact: %v", err)
	}

	_, err := delivery.NewCatalogService(db).PromoteChannel(context.Background(), delivery.ChannelPromotionRequest{
		BundleKey: bundle, AppKey: app, VariantKey: variant, ArtifactIDs: map[string]int64{"windows": artifactID},
	})
	if err == nil || !strings.Contains(err.Error(), "sha512 identity") {
		t.Fatalf("expected missing SHA512 refusal, got %v", err)
	}
}

func TestReleaseBoundChannelPromotionRequiresMatchingArtifactIdentity(t *testing.T) {
	db := setupTestDB(t)
	const bundle, app, variant = "channel_release_identity_bundle", "desktop", "stable"
	if _, err := db.Exec(`INSERT INTO download_apps (bundle_key, app_key, name) VALUES ($1,$2,'Desktop')`, bundle, app); err != nil {
		t.Fatalf("insert app: %v", err)
	}
	var artifactID int64
	if err := db.QueryRow(`
		INSERT INTO download_artifacts
			(bundle_key, app_key, provider, bucket, object_key, release_id, sha512, platform, release_version, metadata)
		VALUES ($1,$2,'s3','bucket','release.zip','release-1','digest','windows','1.0.0',$3::jsonb)
		RETURNING id
	`, bundle, app, `{"bundle_key":"channel_release_identity_bundle","app_key":"desktop","release_id":"release-1","artifact_manifest_digest":"sha256:manifest","candidate_id":"candidate-1","destination_revision_id":"destination-1","authorization_epoch":7,"readiness_review_key":"review-1"}`).Scan(&artifactID); err != nil {
		t.Fatalf("insert artifact: %v", err)
	}
	catalog := delivery.NewCatalogService(db)
	baseRequest := delivery.ChannelPromotionRequest{
		BundleKey: bundle, AppKey: app, VariantKey: variant, ReleaseID: "release-1",
		ArtifactManifestDigest: "sha256:manifest", DestinationRevisionID: "destination-1",
		AuthorizationEpoch: 7, ReadinessReviewKey: "review-1",
		ArtifactIDs: map[string]int64{"windows": artifactID},
	}
	baseRequest.CandidateID = "candidate-2"
	if _, err := catalog.PromoteChannel(context.Background(), baseRequest); err == nil || !strings.Contains(err.Error(), "candidate_id mismatch") {
		t.Fatalf("expected candidate identity refusal, got %v", err)
	}
	baseRequest.CandidateID = "candidate-1"
	if _, err := db.Exec(`UPDATE download_artifacts SET release_id = 'release-2' WHERE id = $1`, artifactID); err != nil {
		t.Fatalf("mutate stored release identity: %v", err)
	}
	if _, err := catalog.PromoteChannel(context.Background(), baseRequest); err == nil || !strings.Contains(err.Error(), "release_id") {
		t.Fatalf("expected release column identity refusal, got %v", err)
	}
	if _, err := db.Exec(`UPDATE download_artifacts SET release_id = 'release-1' WHERE id = $1`, artifactID); err != nil {
		t.Fatalf("restore stored release identity: %v", err)
	}
	if revision, err := catalog.PromoteChannel(context.Background(), baseRequest); err != nil || revision.Revision != 1 || revision.PredecessorRevision != 0 || revision.ReleaseID != "release-1" || revision.CandidateID != "candidate-1" || revision.DestinationRevisionID != "destination-1" || revision.AuthorizationEpoch != 7 || revision.ReadinessReviewKey != "review-1" {
		t.Fatalf("matching release identity promotion = %#v, err=%v", revision, err)
	}
}

func TestChannelHaltBindsToCurrentRevisionAndStopsOffers(t *testing.T) {
	db := setupTestDB(t)
	const bundle, app, variant = "channel_halt_bundle", "desktop", "stable"
	if _, err := db.Exec(`INSERT INTO download_apps (bundle_key, app_key, name) VALUES ($1,$2,'Desktop')`, bundle, app); err != nil {
		t.Fatalf("insert app: %v", err)
	}
	artifactID := int64(0)
	if err := db.QueryRow(`
		INSERT INTO download_artifacts (bundle_key, app_key, provider, bucket, object_key, sha512, platform, release_version)
		VALUES ($1,$2,'s3','bucket','halt.zip','digest','windows','1.0.0') RETURNING id
	`, bundle, app).Scan(&artifactID); err != nil {
		t.Fatalf("insert artifact: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO download_assets (bundle_key, app_key, platform, variant_key, artifact_source, artifact_id, release_version)
		VALUES ($1,$2,'windows',$3,'managed',$4,'1.0.0')
	`, bundle, app, variant, artifactID); err != nil {
		t.Fatalf("insert asset: %v", err)
	}
	catalog := delivery.NewCatalogService(db)
	if _, err := catalog.PromoteChannel(context.Background(), delivery.ChannelPromotionRequest{BundleKey: bundle, AppKey: app, VariantKey: variant, ArtifactIDs: map[string]int64{"windows": artifactID}}); err != nil {
		t.Fatalf("promote channel: %v", err)
	}
	halt, err := catalog.SetChannelHalt(context.Background(), delivery.ChannelHaltRequest{BundleKey: bundle, AppKey: app, VariantKey: variant, ExpectedRevision: 1, Halted: true, Reason: "verified regression"})
	if err != nil {
		t.Fatalf("halt channel: %v", err)
	}
	if !halt.Halted || halt.Revision != 1 {
		t.Fatalf("unexpected halt: %+v", halt)
	}
	halted, err := catalog.IsChannelHalted(bundle, app, variant)
	if err != nil || !halted {
		t.Fatalf("halt state=%t err=%v", halted, err)
	}
	if _, err := catalog.SetChannelHalt(context.Background(), delivery.ChannelHaltRequest{BundleKey: bundle, AppKey: app, VariantKey: variant, ExpectedRevision: 0, Halted: false}); !errors.Is(err, delivery.ErrChannelRevisionConflict) {
		t.Fatalf("expected stale resume conflict, got %v", err)
	}
	if channels, err := catalog.ListChannels(bundle, app); err != nil {
		t.Fatalf("list channels: %v", err)
	} else if len(channels) != 0 {
		t.Fatalf("halted channel remained discoverable: %+v", channels)
	}
}

func TestChannelRecoveryBindsPredecessorAndRefusesUnsafeRollback(t *testing.T) {
	db := setupTestDB(t)
	const bundle, app, variant = "channel_recovery_bundle", "desktop", "stable"
	if _, err := db.Exec(`INSERT INTO download_apps (bundle_key, app_key, name) VALUES ($1,$2,'Desktop')`, bundle, app); err != nil {
		t.Fatal(err)
	}
	artifact := func(name, version string) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRow(`INSERT INTO download_artifacts (bundle_key, app_key, provider, bucket, object_key, sha512, platform, release_version) VALUES ($1,$2,'s3','bucket',$3,$4,'windows',$5) RETURNING id`, bundle, app, name, name+"-digest", version).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	first, second, repair := artifact("first.zip", "1.0.0"), artifact("second.zip", "2.0.0"), artifact("repair.zip", "2.0.1")
	if _, err := db.Exec(`INSERT INTO download_assets (bundle_key, app_key, platform, variant_key, artifact_source, artifact_id, release_version) VALUES ($1,$2,'windows',$3,'managed',$4,'1.0.0')`, bundle, app, variant, first); err != nil {
		t.Fatal(err)
	}
	catalog := delivery.NewCatalogService(db)
	if _, err := catalog.PromoteChannel(context.Background(), delivery.ChannelPromotionRequest{BundleKey: bundle, AppKey: app, VariantKey: variant, ArtifactIDs: map[string]int64{"windows": first}}); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.PromoteChannel(context.Background(), delivery.ChannelPromotionRequest{BundleKey: bundle, AppKey: app, VariantKey: variant, ExpectedRevision: 1, ArtifactIDs: map[string]int64{"windows": second}}); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.RecoverChannel(context.Background(), delivery.ChannelRecoveryRequest{BundleKey: bundle, AppKey: app, VariantKey: variant, ExpectedRevision: 2, ExpectedPredecessor: 1, Action: "rollback", DataCompatibility: "incompatible", Reason: "migration pending"}); err == nil {
		t.Fatal("unsafe rollback was accepted")
	}
	recovery, err := catalog.RecoverChannel(context.Background(), delivery.ChannelRecoveryRequest{BundleKey: bundle, AppKey: app, VariantKey: variant, ExpectedRevision: 2, ExpectedPredecessor: 1, Action: "rollback", DataCompatibility: "compatible", Reason: "verified compatible schema"})
	if err != nil || recovery.Outcome != "rolled_back" || recovery.PredecessorRevision != 2 {
		t.Fatalf("rollback = %#v err=%v", recovery, err)
	}
	if _, err := catalog.RecoverChannel(context.Background(), delivery.ChannelRecoveryRequest{BundleKey: bundle, AppKey: app, VariantKey: variant, ExpectedRevision: recovery.Revision, Action: "forward_repair", DataCompatibility: "compatible", ArtifactIDs: map[string]int64{"windows": repair}, Reason: "forward migration repair"}); err != nil {
		t.Fatalf("forward repair: %v", err)
	}
}

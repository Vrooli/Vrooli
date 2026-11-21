package deployments

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"deployment-manager/releases"
)

func governedCloudReceipt(release, target string) *CloudDeploymentReceipt {
	return &CloudDeploymentReceipt{
		SchemaVersion: 1, DeploymentID: "dep-gov", ScenarioID: "demo", TargetKind: "vps",
		DestinationID: "sha256:target", DestinationHost: "vps.example", DestinationWorkdir: "/srv/demo", DestinationDomain: "demo.example",
		BundleSHA256: strings.Repeat("a", 64), ReleaseDigest: release, TargetKey: target, Outcome: "deployed", Health: "healthy",
		ExternalReceipt: "scenario-to-cloud:dep-gov", ObservedAt: time.Now().UTC(), ProducerRef: cloudReceiptProducerRef,
	}
}

func governedDeployState() *deployState {
	ds := newDeployState("p1", "rel-gov", "stable", "1.0.0", []string{"linux-x64"})
	ds.req.CandidateID = "candidate-1"
	ds.req.DestinationRevisionID = "destination-1"
	ds.req.GitCommitHash = "abc123"
	ds.req.ReadinessReviewKey = "rr-gov"
	ds.req.AuthorizationCheck = func(context.Context) error { return nil }
	ds.req.CloudManifest = json.RawMessage(`{"scenario":{"id":"demo"}}`)
	return ds
}

// TestDeployCloudRefusesPromotionWithoutCompleteEvidence [REQ:STC-P0-035]
// proves GOV-01 (P17-A01): a receipt whose owner evidence is absent or has
// any required cell not passed is refused before any release identity or
// publication receipt is persisted. A healthy observation never substitutes.
func TestDeployCloudRefusesPromotionWithoutCompleteEvidence(t *testing.T) {
	release := strings.Repeat("a", 64)
	cases := map[string]*CloudEvidenceSummary{
		"absent": nil,
		"missing cell": {
			SchemaVersion: "1", ProfileID: "cloud-launch-v1", ReleaseDigest: release, TargetKey: "host:vps.example", RequiredCells: 2, Passed: false, BlockingCells: []string{"GOV-05/api"}, ProducerRef: cloudReceiptProducerRef,
			Cells: []CloudEvidenceCell{{CaseID: "GOV-01", Lane: "api", Disposition: "passed", Required: true, RecordID: "r1", ReceiptRefs: []string{"x"}}, {CaseID: "GOV-05", Lane: "api", Disposition: "missing", Required: true}},
		},
		"unavailable cell": {
			SchemaVersion: "1", ProfileID: "cloud-launch-v1", ReleaseDigest: release, TargetKey: "host:vps.example", RequiredCells: 1, Passed: false, BlockingCells: []string{"GOV-07/api"}, ProducerRef: cloudReceiptProducerRef,
			Cells: []CloudEvidenceCell{{CaseID: "GOV-07", Lane: "api", Disposition: "unavailable", Required: true, Reason: "evidence fetch lacks bytes"}},
		},
		"passed flag without cells": {SchemaVersion: "1", ProfileID: "cloud-launch-v1", ReleaseDigest: release, TargetKey: "host:vps.example", RequiredCells: 0, Passed: true, ProducerRef: cloudReceiptProducerRef},
		"foreign producer": {
			SchemaVersion: "1", ProfileID: "cloud-launch-v1", ReleaseDigest: release, TargetKey: "host:vps.example", RequiredCells: 1, Passed: true, ProducerRef: "caller",
			Cells: []CloudEvidenceCell{{CaseID: "GOV-01", Lane: "api", Disposition: "passed", Required: true, RecordID: "r1"}},
		},
	}
	for name, summary := range cases {
		t.Run(name, func(t *testing.T) {
			receipt := governedCloudReceipt(release, "host:vps.example")
			receipt.Evidence = summary
			cloud := &fakeCloudDeploymentClient{fakeCloudClient: fakeCloudClient{healthy: true}, receipt: receipt}
			relRepo := newFakeReleasesRepo()
			o := newOrch(cloud, nil, nil, relRepo)
			ds := governedDeployState()
			if code := o.deployCloud(ds); code != http.StatusPreconditionFailed {
				t.Fatalf("deployCloud() status = %d, want refusal", code)
			}
			if ds.response.Status != "failed" || !strings.Contains(ds.response.Steps[0].Error, "cloud evidence refuses promotion") {
				t.Fatalf("response = %#v", ds.response)
			}
			if len(relRepo.receipts) != 0 || relRepo.deployments["rel-gov"] != "" {
				t.Fatalf("refused promotion must persist nothing: receipts=%d deployments=%v", len(relRepo.receipts), relRepo.deployments)
			}
			if relRepo.statuses["rel-gov"] != releases.StatusFailed {
				t.Fatalf("release status = %q", relRepo.statuses["rel-gov"])
			}
		})
	}
}

// TestDeployCloudRefusesEvidenceForAnotherReleaseOrTarget [REQ:STC-P0-035]
// proves GOV-02: complete evidence bound to a different release digest or
// target key is incompatible with this receipt.
func TestDeployCloudRefusesEvidenceForAnotherReleaseOrTarget(t *testing.T) {
	release := strings.Repeat("a", 64)
	for name, summary := range map[string]*CloudEvidenceSummary{
		"other release": passingCloudEvidence(strings.Repeat("b", 64), "host:vps.example"),
		"other target":  passingCloudEvidence(release, "machine:other"),
	} {
		t.Run(name, func(t *testing.T) {
			receipt := governedCloudReceipt(release, "host:vps.example")
			receipt.Evidence = summary
			o := newOrch(&fakeCloudDeploymentClient{fakeCloudClient: fakeCloudClient{healthy: true}, receipt: receipt}, nil, nil, newFakeReleasesRepo())
			ds := governedDeployState()
			if code := o.deployCloud(ds); code != http.StatusPreconditionFailed || !strings.Contains(ds.response.Steps[0].Error, "incompatible") {
				t.Fatalf("status = %d response = %#v", code, ds.response)
			}
		})
	}
}

// TestDeployCloudRecordsTheActivatedReleaseAndPredecessor [REQ:STC-P0-035]
// proves GOV-05 / P17-A03 / P17-A04 on the DM side: the persisted
// publication receipt carries the release the target actually activated,
// the predecessor the owner resolved, the review key and the evidence
// drill-down reference; the review identity is passed to the owner.
func TestDeployCloudRecordsTheActivatedReleaseAndPredecessor(t *testing.T) {
	release := strings.Repeat("a", 64)
	receipt := governedCloudReceipt(release, "host:vps.example")
	receipt.Evidence = passingCloudEvidence(release, "host:vps.example")
	receipt.Publication = &CloudPublication{ID: "pub-1", RequestKey: "req-1", ReviewRef: "rr-gov", ReleaseDigest: release, State: "published", ActivatedReleaseDigest: release, PredecessorReleaseDigest: strings.Repeat("9", 64), TargetKey: "host:vps.example", TargetReceipt: &CloudTargetReceipt{ActiveRelease: release, PreviousRelease: strings.Repeat("9", 64)}}
	cloud := &recordingCloudDeploymentClient{fakeCloudDeploymentClient: fakeCloudDeploymentClient{fakeCloudClient: fakeCloudClient{healthy: true}, receipt: receipt}}
	relRepo := newFakeReleasesRepo()
	o := newOrch(cloud, nil, nil, relRepo)
	ds := governedDeployState()
	if code := o.deployCloud(ds); code != 0 {
		t.Fatalf("deployCloud() status = %d steps=%#v", code, ds.response.Steps)
	}
	if cloud.request == nil || cloud.request.Review == nil || cloud.request.Review.ReviewKey != "rr-gov" || cloud.request.Review.CandidateCommit != "abc123" || cloud.request.Review.PolicyVersion <= 0 || cloud.request.Review.CandidateID != "candidate-1" {
		t.Fatalf("review identity passed to the owner = %#v", cloud.request)
	}
	if len(relRepo.receipts) != 1 {
		t.Fatalf("receipts = %#v", relRepo.receipts)
	}
	stored := relRepo.receipts[0]
	if stored.ArtifactDigest != release || stored.PredecessorArtifactDigest != strings.Repeat("9", 64) || stored.ReviewKey != "rr-gov" || !strings.Contains(stored.EvidenceRef, "/deployments/dep-gov/evidence") || stored.Producer != cloudReceiptProducerRef {
		t.Fatalf("stored receipt = %#v", stored)
	}
}

type recordingCloudDeploymentClient struct {
	fakeCloudDeploymentClient
	request *CloudDeploymentRequest
}

func (c *recordingCloudDeploymentClient) DeployCloud(ctx context.Context, request *CloudDeploymentRequest) (*CloudDeploymentReceipt, error) {
	c.request = request
	return c.fakeCloudDeploymentClient.DeployCloud(ctx, request)
}

// TestValidateCloudReceiptRejectsForgedStaleAndMismatchedReceipts
// [REQ:STC-P0-035] proves P17 step 12 on the consumer: a receipt from
// another producer, without a target, for another release, or observed
// outside the freshness policy is refused.
func TestValidateCloudReceiptRejectsForgedStaleAndMismatchedReceipts(t *testing.T) {
	release := strings.Repeat("a", 64)
	request := &CloudDeploymentRequest{Manifest: json.RawMessage(`{"scenario":{"id":"demo"}}`), ExpectedReleaseDigest: release}
	if err := validateCloudReceipt(*governedCloudReceipt(release, "host:vps.example"), "dep-gov", request); err != nil {
		t.Fatalf("exact receipt must validate: %v", err)
	}
	forged := governedCloudReceipt(release, "host:vps.example")
	forged.ProducerRef = "caller"
	if err := validateCloudReceipt(*forged, "dep-gov", request); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("forged producer error = %v", err)
	}
	untargeted := governedCloudReceipt(release, "")
	if err := validateCloudReceipt(*untargeted, "dep-gov", request); err == nil || !strings.Contains(err.Error(), "target") {
		t.Fatalf("missing target error = %v", err)
	}
	other := governedCloudReceipt(strings.Repeat("b", 64), "host:vps.example")
	if err := validateCloudReceipt(*other, "dep-gov", request); err == nil || !strings.Contains(err.Error(), "release") {
		t.Fatalf("wrong release error = %v", err)
	}
	wrongDeployment := governedCloudReceipt(release, "host:vps.example")
	if err := validateCloudReceipt(*wrongDeployment, "dep-other", request); err == nil {
		t.Fatal("wrong deployment must be refused")
	}
	stale := governedCloudReceipt(release, "host:vps.example")
	stale.ObservedAt = time.Now().UTC().Add(-cloudReceiptMaxAge - time.Minute)
	if err := validateCloudReceipt(*stale, "dep-gov", request); err == nil || !strings.Contains(err.Error(), "older than") {
		t.Fatalf("stale receipt error = %v", err)
	}
	previous := cloudReceiptNow
	cloudReceiptNow = func() time.Time { return stale.ObservedAt.Add(time.Minute) }
	defer func() { cloudReceiptNow = previous }()
	if err := validateCloudReceipt(*stale, "dep-gov", request); err != nil {
		t.Fatalf("freshness is measured on the DM clock: %v", err)
	}
}

// TestRecoverBindsReviewAndPreviewToTheCloudOwner [REQ:STC-P0-035] proves
// GOV-08 / P17-A05 on the DM side: the exact review key and the preview
// reference reach the cloud owner, and the owner's preview reference is
// returned to the caller.
func TestRecoverBindsReviewAndPreviewToTheCloudOwner(t *testing.T) {
	cloud := &fakeCloudRecoveryClient{receipt: &CloudRecoveryReceipt{SchemaVersion: 1, DeploymentID: "dep-gov", Action: "rollback", Outcome: "preview", Health: "unknown", DryRun: true, BundleSHA256: strings.Repeat("9", 64), PreviewRef: "preview-1", ReviewKey: "rr-gov", RouteKind: "bundle_pipeline"}}
	o := newOrch(cloud, nil, nil, newFakeReleasesRepo())
	receipt, err := o.Recover(context.Background(), &releases.RecoveryRequest{ReleaseID: "rel-gov", ReviewKey: "rr-gov", CandidateID: "candidate-1", DestinationRevisionID: "destination-1", Action: "rollback", DryRun: true, DataCompatibility: "compatible", RepairBundleSHA256: strings.Repeat("9", 64), PreviewRef: ""}, "dep-gov", strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	if cloud.request == nil || cloud.request.ReviewKey != "rr-gov" || cloud.request.DeploymentID != "dep-gov" {
		t.Fatalf("owner request = %#v", cloud.request)
	}
	if receipt.PreviewRef != "preview-1" || !receipt.DryRun {
		t.Fatalf("receipt = %#v", receipt)
	}
}

type fakeCloudRecoveryClient struct {
	fakeCloudClient
	receipt *CloudRecoveryReceipt
	request *CloudRecoveryRequest
}

func (f *fakeCloudRecoveryClient) RecoverCloud(_ context.Context, request *CloudRecoveryRequest) (*CloudRecoveryReceipt, error) {
	f.request = request
	return f.receipt, nil
}

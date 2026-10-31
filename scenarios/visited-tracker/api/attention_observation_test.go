package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	attentionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/visited-tracker/v1/attention"
)

// [REQ:VT-REQ-024] Guarded rejections and expired claims are not overlap defects.
func TestClaimIntegrityObservesCurrentRevisionOwnership(t *testing.T) {
	now := time.Now()
	claim := ReviewClaim{ID: "one", RequestID: "request-one", Worker: "worker", FileID: uuid.New(), Revision: "rev-one", CreatedAt: now, ExpiresAt: now.Add(time.Minute)}
	differentRevision := claim
	differentRevision.ID, differentRevision.RequestID, differentRevision.Revision = "two", "request-two", "rev-two"
	expired := claim
	expired.ExpiresAt = now
	c := &Campaign{Claims: []ReviewClaim{claim, differentRevision, expired}}
	snapshot := observeClaimIntegrity(c, now)
	if snapshot.ExaminedClaims != 2 || snapshot.Violations != 0 {
		t.Fatalf("valid ownership=%+v", snapshot)
	}
	duplicate := claim
	duplicate.ID, duplicate.RequestID = "three", "request-three"
	c.Claims = append(c.Claims, duplicate)
	snapshot = observeClaimIntegrity(c, now)
	if snapshot.Violations != 1 {
		t.Fatalf("overlap escaped sensor: %+v", snapshot)
	}
	malformed := claim
	malformed.ID = ""
	malformed.FileID = uuid.Nil
	c.Claims = []ReviewClaim{malformed}
	if snapshot := observeClaimIntegrity(c, now); snapshot.Violations != 1 {
		t.Fatalf("malformed active claim=%+v", snapshot)
	}
}

// [REQ:VT-REQ-026] Storage observations are taken from real replacement outcomes.
func TestPreviewReportsWriteFailureAndRecoveryWithoutWriting(t *testing.T) {
	c, _ := attentionFixture(t, 1)
	before := campaignWriteObserver.snapshot(time.Now())
	c.Metadata = map[string]interface{}{"invalid": func() {}}
	if err := writeCampaign(c); err == nil {
		t.Fatal("expected marshal failure")
	}
	snapshot := campaignWriteObserver.snapshot(time.Now())
	if snapshot.CompletedTotal != before.CompletedTotal+1 || snapshot.FailedTotal != before.FailedTotal+1 || snapshot.ConsecutiveFailures < 1 {
		t.Fatalf("failed write not observed: %+v", snapshot)
	}
	c.Metadata = nil
	if err := writeCampaign(c); err != nil {
		t.Fatal(err)
	}
	recovered := campaignWriteObserver.snapshot(time.Now())
	if recovered.ConsecutiveFailures != 0 || recovered.FailedTotal != snapshot.FailedTotal || recovered.Epoch != snapshot.Epoch {
		t.Fatalf("recovery lost history: %+v", recovered)
	}
	response, err := (&attentionRPC{}).Preview(context.Background(), connect.NewRequest(&attentionv1.PreviewRequest{CampaignId: c.ID.String(), Limit: 1}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.Integrity == nil || response.Msg.Integrity.CheckSchemaVersion != 1 || response.Msg.StorageWrites == nil || response.Msg.StorageWrites.CompletedTotal != recovered.CompletedTotal {
		t.Fatalf("preview fabricated write or omitted evidence: %+v", response.Msg)
	}
	stored, err := loadCampaign(c.ID)
	if err != nil || stored.Revision != c.Revision {
		t.Fatalf("preview mutated revision: %+v %v", stored, err)
	}
}

func TestStorageObservationConcurrentSnapshotsAndFreshness(t *testing.T) {
	o := &storageWriteObserver{epoch: "test-process"}
	now := time.Now()
	if o.snapshot(now).CompletedTotal != 0 || o.snapshot(now).LastCompletedAt != "" {
		t.Fatal("new epoch fabricated samples")
	}
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() { defer wg.Done(); o.record(now, errors.New("write failed")); _ = o.snapshot(now) }()
	}
	wg.Wait()
	sample := o.snapshot(now.Add(301 * time.Second))
	if sample.CompletedTotal != 100 || sample.FailedTotal != 100 || sample.ConsecutiveFailures != 100 || sample.LastCompletedAgeSeconds != 301 {
		t.Fatalf("snapshot=%+v", sample)
	}
	o.record(now.Add(302*time.Second), nil)
	sample = o.snapshot(now.Add(303 * time.Second))
	if sample.CompletedTotal != 101 || sample.FailedTotal != 100 || sample.ConsecutiveFailures != 0 || sample.LastCompletedAgeSeconds != 1 {
		t.Fatalf("recovered=%+v", sample)
	}
}

package providers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"storage-manager/internal/cleanup"
	cleanupfakes "storage-manager/internal/testutil/cleanup"
)

type orphanLiveness struct{ open bool }

func (l orphanLiveness) IsRunning(context.Context, string) (bool, error) { return l.open, nil }

type orphanMutableClock struct{ now time.Time }

func (c *orphanMutableClock) Now() time.Time { return c.now }

func orphanFixture(open bool, legacyAge, liveAge time.Duration) (*OrphanedDatabaseProvider, *cleanupfakes.FileSystem) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	fs := &cleanupfakes.FileSystem{Root: "/tmp/db", Files: map[string]cleanup.FileInfo{
		"/tmp/db/old.sqlite":  {Path: "/tmp/db/old.sqlite", Size: 100, ModTime: now.Add(-legacyAge)},
		"/tmp/db/live.sqlite": {Path: "/tmp/db/live.sqlite", Size: 200, ModTime: now.Add(-liveAge)},
	}}
	p := NewOrphanedDatabaseProvider(fs, orphanLiveness{open: open}, cleanupfakes.Clock{Time: now}, "/tmp/db/old.sqlite", "/tmp/db/live.sqlite", 30*24*time.Hour).(*OrphanedDatabaseProvider)
	return p, fs
}

func TestOrphanedDatabaseRequiresAgeReplacementAndNoOpenHandle(t *testing.T) {
	cases := []struct {
		name               string
		open               bool
		legacyAge, liveAge time.Duration
		want               bool
	}{
		{"eligible", false, 31 * 24 * time.Hour, time.Hour, true},
		{"open handle", true, 31 * 24 * time.Hour, time.Hour, false},
		{"legacy too young", false, 29 * 24 * time.Hour, time.Hour, false},
		{"replacement stale", false, 31 * 24 * time.Hour, 25 * time.Hour, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, _ := orphanFixture(tc.open, tc.legacyAge, tc.liveAge)
			estimate, err := p.Estimate(context.Background(), cleanup.EstimateRequest{Scope: cleanup.ObservationScope{Now: time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)}})
			if err != nil {
				t.Fatalf("Estimate: %v", err)
			}
			if (estimate.EstimatedBytes > 0) != tc.want {
				t.Fatalf("estimate = %#v, eligible=%t want %t", estimate, estimate.EstimatedBytes > 0, tc.want)
			}
		})
	}
}

func TestOrphanedDatabaseApplyRemovesOnlyLegacyFamily(t *testing.T) {
	p, fs := orphanFixture(false, 31*24*time.Hour, time.Hour)
	fs.Files["/tmp/db/old.sqlite-wal"] = cleanup.FileInfo{Path: "/tmp/db/old.sqlite-wal", Size: 10}
	fs.Files["/tmp/db/old.sqlite-shm"] = cleanup.FileInfo{Path: "/tmp/db/old.sqlite-shm", Size: 10}
	fs.AllowRemove = true
	result, err := p.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", IdempotencyKey: "run|orphan"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(fs.Removed) != 3 || len(result.AppliedItems) != 3 {
		t.Fatalf("removed=%v result=%#v", fs.Removed, result)
	}
	if result.ReclaimedBytes != 120 {
		t.Fatalf("reclaimed bytes = %d, want 120", result.ReclaimedBytes)
	}
	if _, ok := fs.Files["/tmp/db/live.sqlite"]; !ok {
		t.Fatal("live replacement was removed")
	}
}

func TestOrphanedDatabaseQuarantinesThenExpiresAfter24Hours(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	clock := &orphanMutableClock{now: now}
	fs := &cleanupfakes.FileSystem{Root: "/tmp/db", AllowRemove: true, Files: map[string]cleanup.FileInfo{
		"/tmp/db/old.sqlite":     {Path: "/tmp/db/old.sqlite", Size: 100, ModTime: now.Add(-31 * 24 * time.Hour)},
		"/tmp/db/old.sqlite-wal": {Path: "/tmp/db/old.sqlite-wal", Size: 10, ModTime: now.Add(-31 * 24 * time.Hour)},
		"/tmp/db/live.sqlite":    {Path: "/tmp/db/live.sqlite", Size: 200, ModTime: now.Add(-time.Hour)},
	}}
	p := NewOrphanedDatabaseProvider(fs, orphanLiveness{open: false}, clock, "/tmp/db/old.sqlite", "/tmp/db/live.sqlite", 30*24*time.Hour, "/tmp/db/quarantine").(*OrphanedDatabaseProvider)

	first, err := p.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", IdempotencyKey: "run|orphan|1"})
	if err != nil {
		t.Fatalf("first Apply: %v", err)
	}
	if first.ReclaimedBytes != 0 || len(fs.Renamed) != 2 {
		t.Fatalf("first result = %#v, renames = %#v; want quarantine without reclaim", first, fs.Renamed)
	}
	if _, ok := fs.Files["/tmp/db/live.sqlite"]; !ok {
		t.Fatal("live replacement was removed during quarantine")
	}

	clock.now = now.Add(23 * time.Hour)
	beforeExpiry, err := p.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", IdempotencyKey: "run|orphan|2"})
	if err != nil {
		t.Fatalf("before-expiry Apply: %v", err)
	}
	if beforeExpiry.ReclaimedBytes != 0 || len(fs.Removed) != 0 {
		t.Fatalf("before-expiry result = %#v removed = %#v; want no removal", beforeExpiry, fs.Removed)
	}

	clock.now = now.Add(25 * time.Hour)
	afterExpiry, err := p.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", IdempotencyKey: "run|orphan|3"})
	if err != nil {
		t.Fatalf("after-expiry Apply: %v", err)
	}
	if afterExpiry.ReclaimedBytes != 110 || len(fs.Removed) != 2 {
		t.Fatalf("after-expiry result = %#v removed = %#v; want 110 bytes and two sidecars/family files", afterExpiry, fs.Removed)
	}
	if _, ok := fs.Files["/tmp/db/live.sqlite"]; !ok {
		t.Fatal("live replacement was removed after quarantine expiry")
	}
}

func TestOrphanedDatabaseVerifyRequiresHealthyReplacement(t *testing.T) {
	p, fs := orphanFixture(false, 31*24*time.Hour, time.Hour)
	provider := NewOrphanedDatabaseProviderWithVerifier(fs, p.liveness, p.clock, p.legacy, p.live, p.minAge, func(context.Context, string) error {
		return nil
	}).(*OrphanedDatabaseProvider)
	verified, err := provider.Verify(context.Background(), cleanup.VerifyRequest{})
	if err != nil || !verified.Verified {
		t.Fatalf("healthy replacement verification = %#v, err=%v", verified, err)
	}

	provider = NewOrphanedDatabaseProviderWithVerifier(fs, p.liveness, p.clock, p.legacy, p.live, p.minAge, func(context.Context, string) error {
		return fmt.Errorf("database malformed")
	}).(*OrphanedDatabaseProvider)
	verified, err = provider.Verify(context.Background(), cleanup.VerifyRequest{})
	if err == nil || verified.Verified {
		t.Fatalf("malformed replacement verification = %#v, err=%v", verified, err)
	}
}

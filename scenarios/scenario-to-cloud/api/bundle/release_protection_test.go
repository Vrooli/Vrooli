package bundle

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"scenario-to-cloud/domain"

	"github.com/vrooli/vrooli/packages/artifactlease"
	"github.com/vrooli/vrooli/packages/cloudrelease"
)

func init() {
	// The GC runner tests in this package exercise retention only; the
	// production default reads the operator's bundle store, which a unit test
	// must never depend on.
	releaseProtectionFn = func(time.Time) ([]string, error) { return nil, nil }
}

func writeLeasedRelease(t *testing.T, releasesDir, bundleSHA string, leased bool, expires time.Time) string {
	t.Helper()
	manifest := cloudrelease.Manifest{
		SchemaVersion:       cloudrelease.ManifestSchemaVersion,
		BundleSHA256:        bundleSHA,
		NativeCLI:           cloudrelease.NativeCLI{SHA256: strings.Repeat("b", 64), GOOS: "linux", GOARCH: "amd64"},
		ClosureDigest:       "sha256:" + strings.Repeat("c", 64),
		ConfigurationDigest: strings.Repeat("d", 64),
	}
	digest, err := cloudrelease.ComputeReleaseDigest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifest.ReleaseDigest = digest
	dir := filepath.Join(releasesDir, digest)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(dir, cloudrelease.ManifestFileName), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if leased {
		lease := artifactlease.Lease{Schema: "vrooli.artifact-lease/1", Artifact: dir, Generation: 1, Owner: artifactlease.Owner{Node: "test"}, AcquiredAt: time.Now().UTC().Format(time.RFC3339Nano), ExpiresAt: expires.UTC().Format(time.RFC3339Nano)}
		if err := artifactlease.Save(lease); err != nil {
			t.Fatal(err)
		}
	}
	return digest
}

// TestReleaseProtectedBundleSHA256sHonoursUnexpiredLeases [REQ:STC-P0-026]
// proves the retention planner reads ownership from leases, not from age.
func TestReleaseProtectedBundleSHA256sHonoursUnexpiredLeases(t *testing.T) {
	releasesDir := filepath.Join(t.TempDir(), ReleasesDirName)
	now := time.Now().UTC()
	active := strings.Repeat("1", 64)
	predecessor := strings.Repeat("2", 64)
	expired := strings.Repeat("3", 64)
	unleased := strings.Repeat("4", 64)
	writeLeasedRelease(t, releasesDir, active, true, now.Add(time.Hour))
	writeLeasedRelease(t, releasesDir, predecessor, true, now.Add(time.Hour))
	writeLeasedRelease(t, releasesDir, expired, true, now.Add(-time.Hour))
	writeLeasedRelease(t, releasesDir, unleased, false, now)
	if err := os.MkdirAll(filepath.Join(releasesDir, "not-a-digest"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := ReleaseProtectedBundleSHA256s(releasesDir, now)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != active+","+predecessor {
		t.Fatalf("protected = %v, want active and predecessor only", got)
	}
	missing, err := ReleaseProtectedBundleSHA256s(filepath.Join(t.TempDir(), "absent"), now)
	if err != nil || missing != nil {
		t.Fatalf("absent releases dir: %v %v", missing, err)
	}
}

// TestGCTargetReleasesKeepsLeasedReleasesBeyondRetention [REQ:STC-P0-026]
// proves GC never asks the owner to prune a leased release's bundle, even
// past keep_latest.
func TestGCTargetReleasesKeepsLeasedReleasesBeyondRetention(t *testing.T) {
	owner := ownerWith(4)
	leasedSHA := owner.releases[digestN(1)].BundleSHA256
	original := releaseProtectionFn
	defer func() { releaseProtectionFn = original }()
	releaseProtectionFn = func(time.Time) ([]string, error) { return []string{leasedSHA}, nil }

	resp := GCTargetReleases(context.Background(), owner, gcTarget(), "dep", "app", domain.VPSBundleGCRequest{ScenarioID: "app", KeepLatest: 2})
	if !resp.OK {
		t.Fatalf("gc failed: %s", resp.Error)
	}
	if resp.DeletedCount != 1 || resp.Deleted[0].Filename != digestN(0) {
		t.Fatalf("deleted = %+v, want only the unleased oldest release", resp.Deleted)
	}
	if _, ok := owner.releases[digestN(1)]; !ok {
		t.Fatal("leased release was pruned")
	}
}

func TestGCTargetReleasesRefusesWhenLeasesUnreadable(t *testing.T) {
	owner := ownerWith(1)
	original := releaseProtectionFn
	defer func() { releaseProtectionFn = original }()
	releaseProtectionFn = func(time.Time) ([]string, error) { return nil, os.ErrPermission }
	resp := GCTargetReleases(context.Background(), owner, gcTarget(), "dep", "app", domain.VPSBundleGCRequest{})
	if resp.OK || !strings.Contains(resp.Error, "release leases") || owner.prunes != 0 {
		t.Fatalf("expected refusal on unreadable leases with no prune, got %+v prunes=%d", resp, owner.prunes)
	}
}

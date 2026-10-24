package cloudtarget

import (
	"context"
	"os"
	"strings"
	"testing"
)

// [REQ:STC-P0-013] Release pruning is an owner decision: the cloud may name
// any digest, but the active release, its predecessor and an in-flight
// activation are refused whatever the request says, a staged release that
// is neither is removed with its bytes reported, and an absent digest is
// idempotently "deleted". The listing also carries the facts retention needs
// (size, mod time, bundle digest) so no shell inventory is required.
func TestPruneRefusesActiveAndPreviousAndRemovesOnlyRetiredReleases(t *testing.T) {
	ctx := context.Background()
	first := newFixture(t, defaultBundleEntries(), nil)
	second := newFixture(t, append(defaultBundleEntries(), bundleEntry{name: "scenarios/" + testScenario + "/CHANGELOG", body: "v2"}), nil)
	third := newFixture(t, append(defaultBundleEntries(), bundleEntry{name: "scenarios/" + testScenario + "/CHANGELOG", body: "v3"}), nil)
	second.store, third.store = first.store, first.store
	for i, f := range []fixture{first, second, third} {
		if _, err := f.store.Stage(ctx, StageRequest{Effect: f.effect("op-stage", "stage-"+string(rune('a'+i)), uint64(i+1)), Verify: f.verify()}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := first.store.Activate(ctx, ActivateRequest{Effect: first.effect("op-a1", "activate", 4), Release: first.manifest.ReleaseDigest, Activator: &recordingActivator{}}); err != nil {
		t.Fatal(err)
	}
	if _, err := second.store.Activate(ctx, ActivateRequest{Effect: second.effect("op-a2", "activate", 5), Release: second.manifest.ReleaseDigest, Activator: &recordingActivator{}}); err != nil {
		t.Fatal(err)
	}

	listing, err := first.store.ListReleases(testDeployment)
	if err != nil || len(listing.Releases) != 3 {
		t.Fatalf("listing = %+v err=%v", listing, err)
	}
	for _, row := range listing.Releases {
		if row.SizeBytes <= 0 || row.ModTime == "" || row.BundleSHA256 == "" {
			t.Fatalf("listing row lacks retention facts: %+v", row)
		}
	}

	absent := strings.Repeat("e", 64)
	report, err := first.store.PruneReleases(PruneRequest{DeploymentID: testDeployment, Releases: []string{
		first.manifest.ReleaseDigest, second.manifest.ReleaseDigest, third.manifest.ReleaseDigest, absent,
	}})
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if report.Refused[second.manifest.ReleaseDigest] != "active" || report.Refused[first.manifest.ReleaseDigest] != "previous" {
		t.Fatalf("active/previous must be refused: %+v", report.Refused)
	}
	if len(report.Deleted) != 2 || report.ReclaimedBytes <= 0 {
		t.Fatalf("expected the retired release and the absent digest deleted: %+v", report)
	}
	dir, _ := first.store.ReleaseDir(testDeployment, third.manifest.ReleaseDigest)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("retired release still on disk: %v", err)
	}
	for _, keep := range []string{first.manifest.ReleaseDigest, second.manifest.ReleaseDigest} {
		dir, _ := first.store.ReleaseDir(testDeployment, keep)
		if _, err := os.Stat(dir); err != nil {
			t.Fatalf("protected release %s removed: %v", keep, err)
		}
	}
	if _, err := first.store.PruneReleases(PruneRequest{DeploymentID: testDeployment}); mustCode(t, err) != CodeInvalidArgument {
		t.Fatalf("empty prune must be refused, got %v", err)
	}
}

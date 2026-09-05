package baselinefloor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func sampleManifest(scenario, slug string, created time.Time) Manifest {
	return Manifest{
		Scenario:           scenario,
		Slug:               slug,
		Variant:            "shadow",
		Mode:               ModeShadow,
		RestorePointPath:   "/tmp/rp/" + scenario,
		AnchorBaselineName: "baseline-modes-" + slug,
		AmbientVar:         scenario,
		ShadowInstanceKey:  scenario + "@shadow",
		CreatedAt:          created,
		LastTouchedAt:      created,
		TTL:                Duration(3 * time.Hour),
	}
}

func TestManifest_WriteReadRoundTrip(t *testing.T) {
	store := NewStore(t.TempDir())
	created := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	m := sampleManifest("swarm-manager", "p2-abc", created)

	if err := store.WriteManifest(m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}
	// The manifest lands at the documented path.
	want := filepath.Join(store.Root(), "swarm-manager", "baseline-p2-abc", "engagement.json")
	if got := store.ManifestPath("swarm-manager", "p2-abc"); got != want {
		t.Errorf("ManifestPath = %q, want %q", got, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("manifest file not written: %v", err)
	}
	// TTL is human-friendly in the file.
	raw := readFile(t, want)
	if !strings.Contains(raw, `"ttl": "3h0m0s"`) {
		t.Errorf("manifest TTL not human-friendly: %s", raw)
	}

	got, err := store.ReadManifest("swarm-manager", "p2-abc")
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if got.Scenario != "swarm-manager" || got.Mode != ModeShadow || got.ShadowInstanceKey != "swarm-manager@shadow" {
		t.Errorf("round-trip mismatch: %+v", got)
	}
	if got.TTL.AsDuration() != 3*time.Hour {
		t.Errorf("TTL = %v, want 3h", got.TTL.AsDuration())
	}
	if !got.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, created)
	}
}

func TestManifest_ValidateRejectsIncomplete(t *testing.T) {
	store := NewStore(t.TempDir())
	cases := []Manifest{
		{Slug: "s", Mode: ModeShadow, RestorePointPath: "/x"},     // no scenario
		{Scenario: "a", Mode: ModeShadow, RestorePointPath: "/x"}, // no slug
		{Scenario: "a", Slug: "s", RestorePointPath: "/x"},        // bad mode
		{Scenario: "a", Slug: "s", Mode: ModeLive},                // no restore point
	}
	for i, m := range cases {
		if err := store.WriteManifest(m); err == nil {
			t.Errorf("case %d: expected validation error, got nil", i)
		}
	}
}

func TestStore_ListManifestsGlobsSortsAndSkipsCorrupt(t *testing.T) {
	store := NewStore(t.TempDir())
	older := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC)

	if err := store.WriteManifest(sampleManifest("agent-manager", "x", newer)); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteManifest(sampleManifest("test-genie", "y", older)); err != nil {
		t.Fatal(err)
	}
	// A corrupt sibling must not blind the listing.
	corruptDir := store.EngagementDir("bad-scenario", "z")
	if err := os.MkdirAll(corruptDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corruptDir, manifestFile), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	list, err := store.ListManifests()
	if err != nil {
		t.Fatalf("ListManifests: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("listed %d manifests, want 2 (corrupt skipped)", len(list))
	}
	// Sorted oldest-first.
	if list[0].Scenario != "test-genie" || list[1].Scenario != "agent-manager" {
		t.Errorf("sort order wrong: %s then %s", list[0].Scenario, list[1].Scenario)
	}
}

func TestStore_TouchRenewsLease(t *testing.T) {
	store := NewStore(t.TempDir())
	created := time.Date(2026, 6, 5, 9, 0, 0, 0, time.UTC)
	if err := store.WriteManifest(sampleManifest("git-control-tower", "p2", created)); err != nil {
		t.Fatal(err)
	}
	later := created.Add(90 * time.Minute)
	got, err := store.Touch("git-control-tower", "p2", later)
	if err != nil {
		t.Fatalf("Touch: %v", err)
	}
	if !got.LastTouchedAt.Equal(later) {
		t.Errorf("LastTouchedAt = %v, want %v", got.LastTouchedAt, later)
	}
	// CreatedAt is unchanged by a touch.
	if !got.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt changed on touch: %v", got.CreatedAt)
	}
	// Persisted.
	reread, _ := store.ReadManifest("git-control-tower", "p2")
	if !reread.LastTouchedAt.Equal(later) {
		t.Errorf("touch not persisted: %v", reread.LastTouchedAt)
	}
}

func TestStore_SetTTL(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.WriteManifest(sampleManifest("swarm-manager", "p6", time.Now().UTC())); err != nil {
		t.Fatal(err)
	}
	got, err := store.SetTTL("swarm-manager", "p6", 6*time.Hour)
	if err != nil {
		t.Fatalf("SetTTL: %v", err)
	}
	if got.TTL.AsDuration() != 6*time.Hour {
		t.Errorf("TTL = %v, want 6h", got.TTL.AsDuration())
	}
	// Negative clears the TTL (orchestrator heartbeat mode).
	cleared, err := store.SetTTL("swarm-manager", "p6", -1)
	if err != nil {
		t.Fatal(err)
	}
	if cleared.TTL != 0 {
		t.Errorf("negative SetTTL should clear, got %v", cleared.TTL)
	}
}

func TestManifest_ExpiryLogic(t *testing.T) {
	base := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)

	withTTL := Manifest{LastTouchedAt: base, TTL: Duration(time.Hour)}
	expiry, ok := withTTL.ExpiresAt()
	if !ok || !expiry.Equal(base.Add(time.Hour)) {
		t.Errorf("ExpiresAt = %v ok=%v, want %v true", expiry, ok, base.Add(time.Hour))
	}
	if withTTL.Expired(base.Add(30 * time.Minute)) {
		t.Error("should not be expired before TTL elapses")
	}
	if !withTTL.Expired(base.Add(2 * time.Hour)) {
		t.Error("should be expired after TTL elapses")
	}

	// No TTL ⇒ never idle-expires (orchestrator-heartbeated).
	noTTL := Manifest{LastTouchedAt: base}
	if _, ok := noTTL.ExpiresAt(); ok {
		t.Error("zero TTL should report no expiry")
	}
	if noTTL.Expired(base.Add(1000 * time.Hour)) {
		t.Error("zero TTL should never expire")
	}
}

func TestStore_CleanRemovesEngagement(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.WriteManifest(sampleManifest("data-backup-manager", "p4", time.Now().UTC())); err != nil {
		t.Fatal(err)
	}
	dir := store.EngagementDir("data-backup-manager", "p4")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("engagement dir missing pre-clean: %v", err)
	}
	if err := store.Clean("data-backup-manager", "p4"); err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("engagement dir survived clean: %v", err)
	}
	// Clean is idempotent.
	if err := store.Clean("data-backup-manager", "p4"); err != nil {
		t.Errorf("second Clean should be a no-op, got %v", err)
	}
}

func TestMode_Valid(t *testing.T) {
	if !ModeShadow.Valid() || !ModeLive.Valid() {
		t.Error("shadow and live must be valid")
	}
	if Mode("bogus").Valid() {
		t.Error("bogus mode must be invalid")
	}
}

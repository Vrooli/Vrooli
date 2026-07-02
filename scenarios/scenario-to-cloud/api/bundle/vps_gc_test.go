package bundle

import (
	"testing"

	"scenario-to-cloud/domain"
)

func TestPlanVPSBundleGC_DefaultKeepAndProtect(t *testing.T) {
	bundles := []domain.VPSBundleInfo{
		{Filename: "mini-vrooli_app_a3.tar.gz", ScenarioID: "app", Sha256: "a3", SizeBytes: 30, ModTime: "2026-02-10T03:00:00Z"},
		{Filename: "mini-vrooli_app_a2.tar.gz", ScenarioID: "app", Sha256: "a2", SizeBytes: 20, ModTime: "2026-02-09T03:00:00Z"},
		{Filename: "mini-vrooli_app_a1.tar.gz", ScenarioID: "app", Sha256: "a1", SizeBytes: 10, ModTime: "2026-02-08T03:00:00Z"},
	}

	kept, deleted, deletedBytes := PlanVPSBundleGC(bundles, "app", 0, []string{"a1"})
	if len(kept) != 3 {
		t.Fatalf("expected 3 kept (2 newest + protected), got %d", len(kept))
	}
	if len(deleted) != 0 {
		t.Fatalf("expected 0 deleted, got %d", len(deleted))
	}
	if deletedBytes != 0 {
		t.Fatalf("expected 0 deleted bytes, got %d", deletedBytes)
	}
}

func TestPlanVPSBundleGC_DeletesOldBeyondKeep(t *testing.T) {
	bundles := []domain.VPSBundleInfo{
		{Filename: "mini-vrooli_app_a4.tar.gz", ScenarioID: "app", Sha256: "a4", SizeBytes: 40, ModTime: "2026-02-11T03:00:00Z"},
		{Filename: "mini-vrooli_app_a3.tar.gz", ScenarioID: "app", Sha256: "a3", SizeBytes: 30, ModTime: "2026-02-10T03:00:00Z"},
		{Filename: "mini-vrooli_app_a2.tar.gz", ScenarioID: "app", Sha256: "a2", SizeBytes: 20, ModTime: "2026-02-09T03:00:00Z"},
		{Filename: "mini-vrooli_app_a1.tar.gz", ScenarioID: "app", Sha256: "a1", SizeBytes: 10, ModTime: "2026-02-08T03:00:00Z"},
	}

	_, deleted, deletedBytes := PlanVPSBundleGC(bundles, "app", 2, nil)
	if len(deleted) != 2 {
		t.Fatalf("expected 2 deleted, got %d", len(deleted))
	}
	if deletedBytes != 30 {
		t.Fatalf("expected 30 deleted bytes, got %d", deletedBytes)
	}
}

func TestPlanVPSBundleGC_ScopesByScenario(t *testing.T) {
	bundles := []domain.VPSBundleInfo{
		{Filename: "mini-vrooli_a_x1.tar.gz", ScenarioID: "a", Sha256: "x1", SizeBytes: 10, ModTime: "2026-02-08T03:00:00Z"},
		{Filename: "mini-vrooli_b_y1.tar.gz", ScenarioID: "b", Sha256: "y1", SizeBytes: 10, ModTime: "2026-02-08T03:00:00Z"},
	}

	_, deleted, _ := PlanVPSBundleGC(bundles, "a", 1, nil)
	if len(deleted) != 0 {
		t.Fatalf("expected 0 deleted for scenario a (keep=1), got %d", len(deleted))
	}
}

func TestIsSafeBundleFilename(t *testing.T) {
	cases := []struct {
		name string
		ok   bool
	}{
		{"mini-vrooli_app_abcdef.tar.gz", true},
		{"../mini-vrooli_app_abcdef.tar.gz", false},
		{"mini-vrooli_app/evil.tar.gz", false},
		{"not-a-bundle.tar.gz", false},
	}
	for _, tc := range cases {
		if got := isSafeBundleFilename(tc.name); got != tc.ok {
			t.Fatalf("isSafeBundleFilename(%q)=%v, want %v", tc.name, got, tc.ok)
		}
	}
}

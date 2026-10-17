package monetization

import "testing"

func TestReconcileDeclarationFindsBundleAppAndTierLimitDrift(t *testing.T) {
	decl := declaration{BundleKey: "business_suite", AppKey: "web-console", Meters: []surface{{LimitKey: "voice_minutes"}}}
	snapshot := CatalogSnapshot{Bundles: map[string]BundleSnapshot{"business_suite": {
		Apps: map[string]bool{}, Tiers: map[string]map[string]bool{"free": {}, "pro": {"voice_minutes": true}},
	}}}
	findings := ReconcileDeclaration(decl, snapshot, "fixture")
	if len(findings) != 2 {
		t.Fatalf("findings = %v, want app plus free limit drift", findings)
	}
	if len(ReconcileDeclaration(decl, CatalogSnapshot{Bundles: map[string]BundleSnapshot{}}, "fixture")) != 1 {
		t.Fatal("unknown bundle should produce one blocking finding")
	}
}

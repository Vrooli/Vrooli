package safety

import "testing"

func TestAllowImportServe(t *testing.T) {
	cases := []struct {
		name      string
		tier      Tier
		facts     ImportServeFacts
		wantAllow bool
	}{
		{"local always allowed", TierLocal, ImportServeFacts{Provenance: "user-imported", CommercialUse: "conditional"}, true},
		{"public seed entry allowed", TierPublic, ImportServeFacts{Provenance: "", CommercialUse: ""}, true},
		{"public user-import unattested refused", TierPublic, ImportServeFacts{Provenance: "user-imported", CommercialUse: "conditional"}, false},
		{"public user-import attested allowed", TierPublic, ImportServeFacts{Provenance: "user-imported", CommercialUse: "yes"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			allow, reason := AllowImportServe(tc.tier, tc.facts)
			if allow != tc.wantAllow {
				t.Fatalf("allow = %v, want %v", allow, tc.wantAllow)
			}
			if !allow && reason == "" {
				t.Error("a refusal must carry an actionable reason")
			}
		})
	}
}

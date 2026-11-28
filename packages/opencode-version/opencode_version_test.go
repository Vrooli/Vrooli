package opencodeversion

import "testing"

func TestParseMajor(t *testing.T) {
	cases := []struct {
		raw       string
		wantMajor int
		wantOK    bool
	}{
		{"1.19.5", 1, true},
		{"v1.19.6", 1, true},
		{"2.3.2", 2, true},
		{"opencode v2.3.2", 2, true},
		{"opencode 2.0.0-beta.1", 2, true},
		{"no version here", 0, false},
		{"", 0, false},
		{"v0.1.2", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			major, ok := ParseMajor(tc.raw)
			if ok != tc.wantOK || major != tc.wantMajor {
				t.Fatalf("ParseMajor(%q) = (%d, %v), want (%d, %v)", tc.raw, major, ok, tc.wantMajor, tc.wantOK)
			}
		})
	}
}

func TestFromMajor(t *testing.T) {
	v1 := FromMajor(1)
	if v1.IsV2() || v1.Directory != DirectoryFlag || v1.Variant != VariantFlag || v1.SessionTable != "session" || v1.Catalog != CatalogModelsJSON || v1.PrivateServer || v1.HookSurface != HookSurfaceV1 {
		t.Fatalf("unexpected V1 capabilities: %+v", v1)
	}
	v2 := FromMajor(2)
	if !v2.IsV2() || v2.Directory != DirectoryProcess || v2.Variant != VariantModelSuffix || v2.SessionTable != "session_v2" || v2.Catalog != CatalogNone || !v2.PrivateServer || v2.HookSurface != HookSurfaceV2 {
		t.Fatalf("unexpected V2 capabilities: %+v", v2)
	}
	if unknown := FromMajor(0); unknown.Major != 1 || unknown.IsV2() {
		t.Fatalf("unknown major should fall back to V1: %+v", unknown)
	}
}

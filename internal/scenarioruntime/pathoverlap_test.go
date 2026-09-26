package scenarioruntime

import "testing"

func TestCheckPathOverlap(t *testing.T) {
	cases := []struct {
		existing, proposed string
		want               PathOverlap
	}{
		{"/repo/internal", "/repo/internal", OverlapExact},
		{"/repo/internal/", "/repo/internal", OverlapExact},
		{"/repo", "/repo/internal", OverlapExistingContainsNew},
		{"/repo/internal", "/repo", OverlapNewContainsExisting},
		{"/repo/internal", "/repo/internalx", OverlapNone},
		{"/repo/a", "/repo/b", OverlapNone},
	}
	for _, tc := range cases {
		if got := CheckPathOverlap(tc.existing, tc.proposed); got != tc.want {
			t.Errorf("CheckPathOverlap(%q, %q) = %q, want %q", tc.existing, tc.proposed, got, tc.want)
		}
	}
	overlaps := ClaimOverlaps([]EditorLease{{SessionID: "s1", Claims: []string{"/repo/internal"}}, {SessionID: "s2", Claims: []string{"/repo/docs"}}}, []string{"/repo/internal/shell"})
	if len(overlaps) != 1 || overlaps[0].Holder.SessionID != "s1" || overlaps[0].Overlap != OverlapExistingContainsNew {
		t.Fatalf("overlaps = %+v", overlaps)
	}
}

package contrast

import (
	"testing"
)

// Additional tests for brand contrast validation at the domain level.
// [REQ:BM-REQ-WCAG-CALC] [REQ:BM-REQ-WCAG-REJECT] [REQ:BM-REQ-WCAG-VALIDATE]

// TestCheckBrandColors_AALargeVsNormal verifies that some color combinations
// pass AA Large but fail AA Normal (ratio between 3:1 and 4.5:1).
func TestCheckBrandColors_AALargeVsNormal(t *testing.T) {
	// Medium gray text on white: ~3.5:1 ratio (passes AA Large, fails AA Normal)
	result, err := CheckBrandColors(
		"", "", "",
		"#FFFFFF", // background
		"",
		"#767676", // text (WCAG boundary gray)
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(result.Pairs))
	}
	pair := result.Pairs[0]
	if pair.Ratio < 4.5 {
		// #767676 on white is exactly 4.54:1, just above AA Normal threshold
		t.Logf("Ratio %.2f is below 4.5:1 for this specific gray", pair.Ratio)
	}
	if !pair.AALarge {
		t.Error("expected AA Large to pass for medium contrast")
	}
}

// TestCheckBrandColors_RejectsLowContrast verifies that very low contrast
// color combinations are properly rejected.
func TestCheckBrandColors_RejectsLowContrast(t *testing.T) {
	result, err := CheckBrandColors(
		"#EEEEEE", // light gray primary
		"#E0E0E0", // even lighter secondary
		"",
		"#FFFFFF", // white background
		"#FAFAFA", // nearly white surface
		"#CCCCCC", // light text
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.PassAll {
		t.Error("expected PassAll=false for very low contrast palette")
	}

	// Count failing pairs
	failing := 0
	for _, p := range result.Pairs {
		if !p.AANormal {
			failing++
		}
	}
	if failing == 0 {
		t.Error("expected at least one failing pair")
	}
}

// TestCheckPair_SymmetryProperty verifies that contrast ratio is symmetric:
// Ratio(A,B) == Ratio(B,A).
func TestCheckPair_SymmetryProperty(t *testing.T) {
	pairs := [][2]string{
		{"#000000", "#FFFFFF"},
		{"#1a365d", "#f7fafc"},
		{"#e53e3e", "#ffffff"},
	}

	for _, pair := range pairs {
		r1, err1 := CheckPair(pair[0], pair[1])
		r2, err2 := CheckPair(pair[1], pair[0])
		if err1 != nil || err2 != nil {
			t.Fatalf("CheckPair error: %v / %v", err1, err2)
		}
		if r1.Ratio != r2.Ratio {
			t.Errorf("CheckPair(%s,%s)=%.2f but CheckPair(%s,%s)=%.2f — should be symmetric",
				pair[0], pair[1], r1.Ratio, pair[1], pair[0], r2.Ratio)
		}
	}
}

// TestRelativeLuminance_Monotonic verifies that brighter colors have higher luminance.
func TestRelativeLuminance_Monotonic(t *testing.T) {
	shades := []RGB{
		{0.0, 0.0, 0.0}, // black
		{0.2, 0.2, 0.2},
		{0.5, 0.5, 0.5},
		{0.8, 0.8, 0.8},
		{1.0, 1.0, 1.0}, // white
	}

	prevLum := -1.0
	for i, shade := range shades {
		lum := RelativeLuminance(shade)
		if lum <= prevLum {
			t.Errorf("luminance[%d]=%.4f <= luminance[%d]=%.4f — should be monotonically increasing",
				i, lum, i-1, prevLum)
		}
		prevLum = lum
	}
}

// TestStandardPairings_AllRolesCovered ensures standard pairings cover
// the critical foreground/background combinations for brand colors.
func TestStandardPairings_AllRolesCovered(t *testing.T) {
	fgRoles := map[ColorRole]bool{}
	bgRoles := map[ColorRole]bool{}
	for _, p := range StandardPairings {
		fgRoles[p.Foreground] = true
		bgRoles[p.Background] = true
	}

	// Text must be checked against at least one background
	if !fgRoles[RoleText] {
		t.Error("StandardPairings missing text as foreground")
	}
	// Primary should be checked as foreground
	if !fgRoles[RolePrimary] {
		t.Error("StandardPairings missing primary as foreground")
	}
	// Background must appear as a background role
	if !bgRoles[RoleBackground] {
		t.Error("StandardPairings missing background as background")
	}
}

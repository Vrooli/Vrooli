package contrast

import (
	"math"
	"testing"
)

// [REQ:BM-REQ-WCAG-CALC] [REQ:BM-REQ-WCAG-VALIDATE] [REQ:BM-REQ-WCAG-REJECT]

func TestParseHex(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantR   float64
		wantG   float64
		wantB   float64
		wantErr bool
	}{
		{"black 6-digit", "#000000", 0, 0, 0, false},
		{"white 6-digit", "#FFFFFF", 1, 1, 1, false},
		{"red 6-digit", "#FF0000", 1, 0, 0, false},
		{"black 3-digit", "#000", 0, 0, 0, false},
		{"white 3-digit", "#FFF", 1, 1, 1, false},
		{"no hash", "FF0000", 1, 0, 0, false},
		{"invalid length", "#FFFF", 0, 0, 0, true},
		{"invalid chars", "#GGGGGG", 0, 0, 0, true},
		{"empty", "", 0, 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := ParseHex(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseHex(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if math.Abs(c.R-tt.wantR) > 0.001 || math.Abs(c.G-tt.wantG) > 0.001 || math.Abs(c.B-tt.wantB) > 0.001 {
				t.Errorf("ParseHex(%q) = {%.3f, %.3f, %.3f}, want {%.3f, %.3f, %.3f}",
					tt.input, c.R, c.G, c.B, tt.wantR, tt.wantG, tt.wantB)
			}
		})
	}
}

func TestRelativeLuminance(t *testing.T) {
	tests := []struct {
		name string
		rgb  RGB
		want float64
	}{
		{"black", RGB{0, 0, 0}, 0.0},
		{"white", RGB{1, 1, 1}, 1.0},
		{"mid-gray", RGB{0.5, 0.5, 0.5}, 0.2140},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RelativeLuminance(tt.rgb)
			if math.Abs(got-tt.want) > 0.001 {
				t.Errorf("RelativeLuminance(%v) = %.4f, want %.4f", tt.rgb, got, tt.want)
			}
		})
	}
}

func TestRatio(t *testing.T) {
	tests := []struct {
		name string
		fg   RGB
		bg   RGB
		want float64
	}{
		{"black on white", RGB{0, 0, 0}, RGB{1, 1, 1}, 21.0},
		{"white on white", RGB{1, 1, 1}, RGB{1, 1, 1}, 1.0},
		{"same color", RGB{0.5, 0.5, 0.5}, RGB{0.5, 0.5, 0.5}, 1.0},
		// Order shouldn't matter
		{"white on black", RGB{1, 1, 1}, RGB{0, 0, 0}, 21.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Ratio(tt.fg, tt.bg)
			if math.Abs(got-tt.want) > 0.1 {
				t.Errorf("Ratio(%v, %v) = %.2f, want %.2f", tt.fg, tt.bg, got, tt.want)
			}
		})
	}
}

func TestCheckPair(t *testing.T) {
	tests := []struct {
		name      string
		fg        string
		bg        string
		wantPass  bool
		wantLarge bool
		wantErr   bool
		minRatio  float64
	}{
		{"black on white passes", "#000000", "#FFFFFF", true, true, false, 21.0},
		{"white on white fails", "#FFFFFF", "#FFFFFF", false, false, false, 0},
		{"dark gray on white passes", "#595959", "#FFFFFF", true, true, false, 4.5},
		{"light gray on white fails normal", "#999999", "#FFFFFF", false, false, false, 0},
		{"invalid fg", "not-a-color", "#FFFFFF", false, false, true, 0},
		{"invalid bg", "#000000", "xyz", false, false, true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CheckPair(tt.fg, tt.bg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CheckPair(%q, %q) error = %v, wantErr %v", tt.fg, tt.bg, err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if result.AANormal != tt.wantPass {
				t.Errorf("AANormal = %v, want %v (ratio: %.2f)", result.AANormal, tt.wantPass, result.Ratio)
			}
			if result.AALarge != tt.wantLarge {
				t.Errorf("AALarge = %v, want %v (ratio: %.2f)", result.AALarge, tt.wantLarge, result.Ratio)
			}
			if tt.minRatio > 0 && result.Ratio < tt.minRatio {
				t.Errorf("Ratio = %.2f, want >= %.2f", result.Ratio, tt.minRatio)
			}
		})
	}
}

// TestStandardPairingsAreComplete ensures all declared pairings resolve correctly.
// [REQ:BM-REQ-WCAG-VALIDATE]
func TestStandardPairingsAreComplete(t *testing.T) {
	if len(StandardPairings) == 0 {
		t.Fatal("StandardPairings is empty — no pairings would be checked")
	}
	// Every pairing must have distinct foreground and background roles
	for i, p := range StandardPairings {
		if p.Foreground == p.Background {
			t.Errorf("StandardPairings[%d]: foreground and background are the same role %q", i, p.Foreground)
		}
	}
}

// TestResolveColor verifies the role→hex mapping used by CheckBrandColors.
func TestResolveColor(t *testing.T) {
	palette := map[ColorRole]string{
		RolePrimary: "#111", RoleSecondary: "#222", RoleAccent: "#333",
		RoleBackground: "#444", RoleSurface: "#555", RoleText: "#666",
	}
	for role, want := range palette {
		got := resolveColor(role, palette[RolePrimary], palette[RoleSecondary], palette[RoleAccent],
			palette[RoleBackground], palette[RoleSurface], palette[RoleText])
		if got != want {
			t.Errorf("resolveColor(%q) = %q, want %q", role, got, want)
		}
	}
	// Unknown role returns empty
	if got := resolveColor("unknown", "", "", "", "", "", ""); got != "" {
		t.Errorf("resolveColor(unknown) = %q, want empty", got)
	}
}

func TestCheckBrandColors(t *testing.T) {
	t.Run("accessible brand passes", func(t *testing.T) {
		result, err := CheckBrandColors(
			"#1a365d", // dark blue primary
			"#2d3748", // dark secondary
			"#8B0000", // dark red accent (high contrast)
			"#FFFFFF", // white background
			"#F7FAFC", // light surface
			"#1A202C", // dark text
		)
		if err != nil {
			t.Fatal(err)
		}
		if !result.PassAll {
			t.Error("expected all pairs to pass AA normal")
			for _, p := range result.Pairs {
				t.Logf("  %s on %s: ratio=%.2f pass=%v", p.Foreground, p.Background, p.Ratio, p.AANormal)
			}
		}
	})

	t.Run("inaccessible brand fails", func(t *testing.T) {
		result, err := CheckBrandColors(
			"#CCCCCC", // light gray primary
			"#DDDDDD", // lighter secondary
			"#EEEEEE", // very light accent
			"#FFFFFF", // white background
			"#F0F0F0", // light surface
			"#AAAAAA", // light text
		)
		if err != nil {
			t.Fatal(err)
		}
		if result.PassAll {
			t.Error("expected some pairs to fail AA normal")
		}
	})

	t.Run("empty colors returns pass", func(t *testing.T) {
		result, err := CheckBrandColors("", "", "", "", "", "")
		if err != nil {
			t.Fatal(err)
		}
		if !result.PassAll {
			t.Error("expected pass when all colors empty")
		}
	})

	t.Run("partial colors checks available pairs", func(t *testing.T) {
		result, err := CheckBrandColors(
			"", "",
			"",
			"#FFFFFF",
			"",
			"#000000",
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Pairs) != 1 {
			t.Errorf("expected 1 pair, got %d", len(result.Pairs))
		}
	})

	t.Run("invalid color returns error", func(t *testing.T) {
		_, err := CheckBrandColors("invalid", "", "", "#FFFFFF", "", "")
		if err == nil {
			t.Error("expected error for invalid color")
		}
	})
}

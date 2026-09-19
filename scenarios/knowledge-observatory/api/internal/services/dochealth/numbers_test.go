package dochealth

import "testing"

func TestScanNumbersLine_Detection(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool // expect an unmarked_number finding
	}{
		{"digit + plural noun", "We run four teams across the org.", true},
		{"digit-plus + plural noun", "The system composes 30+ resources locally.", true},
		{"one intervening adjective", "The auditor applies seven audit lenses.", true},
		{"two intervening modifiers", "Six themed dashboard pages ship in v1.", true},
		{"three members", "scenario-qa runs three members.", true},
		{"large count plus", "vite build processes 4444+ modules.", true},

		{"singular noun not flagged", "There is one team here.", false},
		{"time unit not flagged", "It completes in 5 minutes.", false},
		{"standalone year", "In 2026 we shipped the kernel.", false},
		{"year before plural noun", "The 2024 releases were stable.", false},
		{"version with dot", "Go 1.25 modules compile fine.", false},
		{"preceding version word", "REST version 2 endpoints remain.", false},
		{"preceding phase word", "Phase 3 tasks are pending.", false},
		{"ordinal suffix", "She took 1st place overall.", false},
		{"percentage", "Coverage rose to 80% overall.", false},
		{"date token", "Shipped on 2026-06-09 cleanly.", false},
		{"no noun nearby", "We counted to four.", false},

		// Evidence-driven excludes (from a real-docs scan).
		{"identifier before verb", "This is how criterion 2 shows up in the portfolio.", false},
		{"parenthesized citation", "meta-optimization (3). Also feeds back into themes.", false},
		{"cross-sentence boundary", "Propose adding one. Theme boundaries are a tool.", false},
		{"verb not noun", "The selector runs and three members remain.", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := scanNumbersLine("x.md", tt.line, 1)
			got := hasFinding(findings, findingUnmarkedNumber)
			if got != tt.want {
				t.Fatalf("scanNumbersLine(%q) unmarked_number=%v, want %v (findings=%#v)", tt.line, got, tt.want, findings)
			}
		})
	}
}

func TestScanNumbers_MarkerSuppressesAndValidates(t *testing.T) {
	tests := []struct {
		name         string
		line         string
		wantUnmarked bool
		wantNoReason bool
	}{
		{"valid category suppresses", "The free tier allows `num[threshold]:100` requests.", false, false},
		{"target category clean", "We target `num[target]:1000` paying users.", false, false},
		{"bare num marker flagged", "We keep `num:5` services running.", false, true},
		{"unknown category flagged", "We keep `num[vibes]:5` services running.", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := scanNumbersLine("x.md", tt.line, 1)
			if got := hasFinding(findings, findingUnmarkedNumber); got != tt.wantUnmarked {
				t.Fatalf("unmarked_number=%v want %v (%#v)", got, tt.wantUnmarked, findings)
			}
			if got := hasFinding(findings, findingNumberMarkerNoReason); got != tt.wantNoReason {
				t.Fatalf("number_marker_without_reason=%v want %v (%#v)", got, tt.wantNoReason, findings)
			}
		})
	}
}

func TestScanNumbersContent_SkipsFencesAndFrontMatter(t *testing.T) {
	content := "---\nweight: 4 items\n---\n" +
		"# Intro\n\n" +
		"We run four teams.\n\n" +
		"```\nplain 5 widgets in code\n```\n\n" +
		"`inline 6 spans` are skipped.\n"
	findings, count := scanNumbersContent("x.md", content)
	if count != 1 {
		t.Fatalf("expected exactly 1 finding, got %d: %#v", count, findings)
	}
	if !hasFinding(findings, findingUnmarkedNumber) {
		t.Fatalf("expected the prose 'four teams' to be flagged: %#v", findings)
	}
	if findings[0].Line != 6 {
		t.Fatalf("expected finding on line 6, got %d", findings[0].Line)
	}
}

func TestScanNumbersLine_ListMarkerNotCounted(t *testing.T) {
	// The "1." is a list marker; only "four teams" should flag (one finding).
	findings := scanNumbersLine("x.md", "1. four teams collaborate", 1)
	if len(findings) != 1 || findings[0].Code != findingUnmarkedNumber {
		t.Fatalf("expected exactly one unmarked_number finding, got %#v", findings)
	}
}

func TestScanNumbersFindingsAreWarnings(t *testing.T) {
	findings := scanNumbersLine("x.md", "We run four teams and keep `num:9` daemons.", 1)
	if len(findings) == 0 {
		t.Fatal("expected findings")
	}
	for _, f := range findings {
		if f.Severity != SeverityWarning {
			t.Fatalf("finding %q severity=%v, want Warning (non-fatal by design)", f.Code, f.Severity)
		}
		if f.Message == "" || f.Code == "" {
			t.Fatalf("finding must have non-empty code/message: %#v", f)
		}
	}
}

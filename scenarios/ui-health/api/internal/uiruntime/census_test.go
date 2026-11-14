package uiruntime

import (
	"strings"
	"testing"
)

func TestPageCensusRetainsTokenizedSourcePropertyCounts(t *testing.T) {
	const pattern = "footer:nav|fixed|block|nowrap|color(srgb 1 1 1 / 0.96)"
	layout := `{"patternCensus":{"footer:nav|fixed|block|nowrap|color(srgb 1 1 1 / 0.96)":1},"tokenizedPatternProperties":{"footer:nav|fixed|block|nowrap|color(srgb 1 1 1 / 0.96)":{"background-color":1}}}`
	got := pageCensus(layout, "http://sample/", "mobile", "http://bas/screenshot.png")
	if got.TokenizedPatternProperties[pattern]["background-color"] != 1 {
		t.Fatalf("token-backed property evidence not captured: %+v", got)
	}
}

func TestCompareScenarioCensusFlagsOnlyDominantStyleMismatch(t *testing.T) {
	got := CompareScenarioCensus([]PagePatternCensus{
		{Scenario: "sample", Route: "/", Viewport: "desktop", Patterns: map[string]int{"button:44|8|14|600|blue|white": 3}},
		{Scenario: "sample", Route: "/settings", Viewport: "desktop", Patterns: map[string]int{"button:44|8|14|600|blue|white": 2}},
		{Scenario: "sample", Route: "/reports", Viewport: "desktop", Patterns: map[string]int{"button:36|4|12|400|gray|black": 1}, ScreenshotRef: "http://bas.local/screenshot.png"},
	}, 0.5)
	if len(got) != 1 {
		t.Fatalf("got %d outliers, want one dominant-style mismatch: %+v", len(got), got)
	}
	if got[0].Route != "/reports" || got[0].Score < 0.666 || got[0].Score > 0.667 || got[0].Family != "button" {
		t.Fatalf("unexpected outlier: %+v", got[0])
	}
	if got[0].Pattern != "button:36|4|12|400|gray|black" || got[0].DominantPattern != "button:44|8|14|600|blue|white" {
		t.Fatalf("finding should carry observed and dominant patterns: %+v", got[0])
	}
	if got[0].ScreenshotRef != "http://bas.local/screenshot.png" {
		t.Fatalf("screenshot evidence ref = %q", got[0].ScreenshotRef)
	}
}

func TestCompareScenarioCensusIgnoresPagesWithoutElementFamily(t *testing.T) {
	got := CompareScenarioCensus([]PagePatternCensus{
		{Scenario: "sample", Route: "/", Viewport: "desktop", Patterns: map[string]int{"footer:common": 1}},
		{Scenario: "sample", Route: "/settings", Viewport: "desktop", Patterns: map[string]int{"footer:common": 1}},
		{Scenario: "sample", Route: "/reports", Viewport: "desktop", Patterns: map[string]int{"button:common": 1}},
	}, 0.5)
	if len(got) != 0 {
		t.Fatalf("page without a footer should not be flagged for lacking one: %+v", got)
	}
}

func TestCompareScenarioCensusKeepsHeadingLevelsSeparate(t *testing.T) {
	got := CompareScenarioCensus([]PagePatternCensus{
		{Scenario: "sample", Route: "/", Viewport: "desktop", Patterns: map[string]int{"heading:h1|18px|700|24px|black": 1}},
		{Scenario: "sample", Route: "/settings", Viewport: "desktop", Patterns: map[string]int{"heading:h3|14px|600|20px|black": 1}},
	}, 0.5)
	if len(got) != 0 {
		t.Fatalf("different semantic heading levels are not style mismatches: %+v", got)
	}
}

func TestCompareScenarioCensusIgnoresTokenizedStyleVariant(t *testing.T) {
	got := CompareScenarioCensus([]PagePatternCensus{
		{Scenario: "sample", Route: "/a", Viewport: "desktop", Patterns: map[string]int{"button:44|8|14|600|blue|white": 2}, Tokens: []string{"44px", "8px", "14px", "600", "blue", "white", "36px", "4px", "12px", "400", "gray", "black"}},
		{Scenario: "sample", Route: "/b", Viewport: "desktop", Patterns: map[string]int{"button:44|8|14|600|blue|white": 1}, Tokens: []string{"44px", "8px", "14px", "600", "blue", "white", "36px", "4px", "12px", "400", "gray", "black"}},
		{Scenario: "sample", Route: "/c", Viewport: "desktop", Patterns: map[string]int{"button:36|4|12|400|gray|black": 1}, Tokens: []string{"44px", "8px", "14px", "600", "blue", "white", "36px", "4px", "12px", "400", "gray", "black"}},
	}, 0.5)
	if len(got) != 0 {
		t.Fatalf("intentional style variant composed of declared tokens should not be an outlier: %+v", got)
	}
}

func TestCompareScenarioCensusChecksButtonRadiusComponentsAgainstTokens(t *testing.T) {
	tokens := []string{"44px", "6px", "14px", "600", "#2563eb", "#ffffff"}
	got := CompareScenarioCensus([]PagePatternCensus{
		{Scenario: "sample", Route: "/a", Viewport: "desktop", Patterns: map[string]int{"button:44px|6px|14px|600|#2563eb|#ffffff": 2}, Tokens: tokens},
		{Scenario: "sample", Route: "/b", Viewport: "desktop", Patterns: map[string]int{"button:44px|6px 6px 0px 0px|14px|600|#2563eb|#ffffff": 1}, Tokens: tokens},
	}, 0.5)
	if len(got) != 0 {
		t.Fatalf("border-radius shorthand composed of declared token values should not be an outlier: %+v", got)
	}
}

func TestCompareScenarioCensusIgnoresComputedUnitlessHeadingLineHeight(t *testing.T) {
	tokens := []string{"18px", "24px", "600", "#142943"}
	got := CompareScenarioCensus([]PagePatternCensus{
		{Scenario: "sample", Route: "/a", Viewport: "desktop", Patterns: map[string]int{"heading:h2|24px|600|31.2px|rgb(20, 41, 67)": 1}, Tokens: tokens},
		{Scenario: "sample", Route: "/b", Viewport: "desktop", Patterns: map[string]int{"heading:h2|18px|600|23.4px|rgb(20, 41, 67)": 1}, Tokens: tokens},
	}, 0.5)
	if len(got) != 0 {
		t.Fatalf("tokenized heading sizes with different computed unitless line heights should not be outliers: %+v", got)
	}
}

func TestNormalizeDesignValueMatchesComputedRGBToHexToken(t *testing.T) {
	got, ok := normalizeDesignValue("rgb(37, 99, 235)")
	if !ok || got != "#2563eb" {
		t.Fatalf("normalized color = %q, %v", got, ok)
	}
}

func TestNormalizeDesignValuePreservesAlphaSemantics(t *testing.T) {
	if got, ok := normalizeDesignValue("rgba(0, 0, 0, 0)"); ok || got != "" {
		t.Fatalf("fully transparent color = %q, %v; want ignored paint", got, ok)
	}
	if got, ok := normalizeDesignValue("rgb(0 0 0 / 0%)"); ok || got != "" {
		t.Fatalf("modern fully transparent color = %q, %v; want ignored paint", got, ok)
	}
	if got, ok := normalizeDesignValue("rgba(0, 0, 0, 50%)"); !ok || got != "#000000@0.500" {
		t.Fatalf("half-transparent color = %q, %v; want alpha-preserving value", got, ok)
	}
	if got, ok := normalizeDesignValue("rgba(0, 0, 0, 1)"); !ok || got != "#000000" {
		t.Fatalf("opaque color = %q, %v; want opaque token value", got, ok)
	}
}

func TestCompareScenarioCensusIgnoresFullyTransparentPaintAsTokenOutlier(t *testing.T) {
	got := CompareScenarioCensus([]PagePatternCensus{
		{Scenario: "sample", Route: "/a", Viewport: "desktop", Patterns: map[string]int{"heading:h1|18px|700|24px|black": 1}, Tokens: []string{"18px", "700", "24px", "black"}},
		{Scenario: "sample", Route: "/b", Viewport: "desktop", Patterns: map[string]int{"heading:h1|18px|700|24px|rgba(0, 0, 0, 0)": 1}, Tokens: []string{"18px", "700", "24px", "black"}},
	}, 0.5)
	for _, finding := range got {
		if finding.Route == "/b" && strings.Contains(finding.Pattern, "rgba(0, 0, 0, 0)") {
			t.Fatalf("fully transparent paint was treated as a design-token outlier: %+v", finding)
		}
	}
}

func TestCompareScenarioCensusNeedsMultiplePages(t *testing.T) {
	got := CompareScenarioCensus([]PagePatternCensus{{Scenario: "one", Route: "/", Patterns: map[string]int{"button:one": 1}}}, 0.5)
	if len(got) != 0 {
		t.Fatalf("single page cannot establish a scenario majority: %+v", got)
	}
}

func TestCompareScenarioCensusFlagsUnmatchedDesignTokenValues(t *testing.T) {
	got := CompareScenarioCensus([]PagePatternCensus{
		{Scenario: "sample", Route: "/a", Patterns: map[string]int{"spacing:16|16|16": 1}, Tokens: []string{"1rem", "16px"}},
		{Scenario: "sample", Route: "/b", Patterns: map[string]int{"spacing:12|12|12": 1}, Tokens: []string{"1rem", "16px"}},
	}, 0.5)
	tokenFindings := 0
	for _, finding := range got {
		if strings.Contains(finding.Reason, "declared design tokens") {
			tokenFindings++
			if finding.Route != "/b" {
				t.Fatalf("token outlier route = %s, want /b", finding.Route)
			}
			if finding.ElementFamily != "spacing" || finding.Property != "gap" || finding.ElementPattern != "spacing:12|12|12" || finding.ObservedCount != 1 {
				t.Fatalf("token outlier should retain its measured property and source pattern: %+v", finding)
			}
		}
	}
	if tokenFindings != 1 {
		t.Fatalf("want one non-token value finding, got %+v", got)
	}
}

func TestCompareScenarioCensusAcceptsComputedValueFromTokenizedSource(t *testing.T) {
	const tokenizedFooter = "footer:nav|fixed|block|nowrap|color(srgb 1 1 1 / 0.96)"
	got := CompareScenarioCensus([]PagePatternCensus{
		{Scenario: "sample", Route: "/a", Viewport: "mobile", Patterns: map[string]int{"footer:nav|fixed|block|nowrap|#ffffff": 1}, Tokens: []string{"#ffffff"}},
		{Scenario: "sample", Route: "/b", Viewport: "mobile", Patterns: map[string]int{"footer:nav|fixed|block|nowrap|#ffffff": 1}, Tokens: []string{"#ffffff"}},
		{Scenario: "sample", Route: "/c", Viewport: "mobile", Patterns: map[string]int{tokenizedFooter: 1}, Tokens: []string{"#ffffff"}, TokenizedPatternProperties: map[string]map[string]int{tokenizedFooter: {"background-color": 1}}},
	}, 0.5)
	if len(got) != 0 {
		t.Fatalf("computed color derived from a token-backed CSS declaration should not be reported as an untokenized mismatch: %+v", got)
	}
}

func TestCompareScenarioCensusKeepsViewportsSeparate(t *testing.T) {
	got := CompareScenarioCensus([]PagePatternCensus{
		{Scenario: "sample", Route: "/a", Viewport: "desktop", Patterns: map[string]int{"button:desktop": 1}},
		{Scenario: "sample", Route: "/b", Viewport: "desktop", Patterns: map[string]int{"button:desktop": 1}},
		{Scenario: "sample", Route: "/a", Viewport: "mobile", Patterns: map[string]int{"button:mobile": 1}},
		{Scenario: "sample", Route: "/b", Viewport: "mobile", Patterns: map[string]int{"button:mobile": 1}},
	}, 0.5)
	if len(got) != 0 {
		t.Fatalf("intentional viewport-specific patterns should not be cross-compared: %+v", got)
	}
}

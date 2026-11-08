package tokenparity

import (
	"testing"

	"react-component-library/internal/themes"
)

func TestCheckVersionFlagsOnlyForeignDisagreeingFallbacks(t *testing.T) {
	canonical := []themes.DesignToken{{Name: "--space-md", Value: "24px"}, {Name: "--radius-panel", Value: "0.5rem"}, {Name: "--font-mono", Value: `"JetBrains Mono", monospace`}}
	sources := map[string][]byte{
		"Panel.tsx": []byte("padding: var(--space-md, 1rem); border-radius: var(--radius-panel,  0.5rem); font: var(--font-mono, monospace); color: var(--local, red); --local: blue; gap: var(--rcl-gap, 3px);"),
		"story.tsx": []byte("margin: var(--space-md, 24px);"),
	}
	got := CheckVersion(canonical, sources)
	if len(got) != 2 || got[0].Property != "--space-md" || got[0].Canonical != "24px" || got[1].Property != "--font-mono" {
		t.Fatalf("mismatches = %+v, want --space-md and --font-mono from Panel.tsx", got)
	}
}

func TestParseFallbacksHonoursNestedParentheses(t *testing.T) {
	got := ParseFallbacks("a: var(--x, calc(1px + var(--y))); b: var(--z);")
	if len(got) != 1 || got[0].Property != "--x" || got[0].Value != "calc(1px + var(--y))" {
		t.Fatalf("fallbacks = %+v", got)
	}
}

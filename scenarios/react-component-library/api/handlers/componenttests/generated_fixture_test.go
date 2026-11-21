package componenttests

import (
	"strings"
	"testing"
)

func TestFixtureNameIsStable(t *testing.T) {
	first := fixtureName("positive")
	second := fixtureName("positive")
	if first != second {
		t.Fatalf("fixtureName is not stable: %q != %q", first, second)
	}
	if !strings.HasPrefix(first, generatedFixtureNamePrefix) {
		t.Fatalf("fixtureName = %q, want prefix %q", first, generatedFixtureNamePrefix)
	}
	if strings.ContainsAny(first, "0123456789") {
		t.Fatalf("fixtureName = %q contains a timestamp-like numeric suffix", first)
	}
}

func TestRegionStyleSamplePreservesStyledControlProof(t *testing.T) {
	raw := `{"schema":"bas-region-style-sample/v1","computed_style":{"color":"rgb(1, 2, 3)","background_color":"rgb(240, 240, 240)","border_color":"rgb(20, 20, 20)","border_style":"solid","border_width":"1px","visibility":"visible","display":"inline-flex","opacity":"1"}}`
	if !regionStyleHasStyledControl(raw) {
		t.Fatal("expected visible styled control to pass")
	}
	for _, invalid := range []string{
		`{"schema":"bas-region-style-sample/v1","computed_style":{"color":"black","background_color":"transparent","border_color":"black","border_style":"none","border_width":"0px","visibility":"visible","display":"block","opacity":"1"}}`,
		`{"schema":"bas-region-style-sample/v1","text":"must not be accepted","computed_style":{}}`,
	} {
		if regionStyleHasStyledControl(invalid) {
			t.Fatalf("invalid region style sample passed: %s", invalid)
		}
	}
}

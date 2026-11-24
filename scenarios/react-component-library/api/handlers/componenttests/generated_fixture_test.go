package componenttests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveRequiredSurfaceTokenUsesAtomicWriteAndPreservesMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "design-tokens.css")
	if err := os.WriteFile(path, []byte(":root {\n  --color-surface: #fff;\n  --color-primary: #123;\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatal(err)
	}

	if err := removeRequiredSurfaceToken(path); err != nil {
		t.Fatalf("removeRequiredSurfaceToken: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "--color-surface:") {
		t.Fatalf("surface token remains in %q", got)
	}
	if !strings.Contains(string(got), "--color-primary: #123;") {
		t.Fatalf("unrelated token was removed: %q", got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Fatalf("mode = %04o, want 0640", got)
	}
}

func TestRemoveRequiredSurfaceTokenFailsWhenFixtureIsMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.css")
	if err := removeRequiredSurfaceToken(path); !os.IsNotExist(err) {
		t.Fatalf("removeRequiredSurfaceToken error = %v, want not-exist", err)
	}
}

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

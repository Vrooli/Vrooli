package looks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	looksv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/looks"
)

func TestParseKind(t *testing.T) {
	cases := map[string]looksv1.LookKind{
		"film":     looksv1.LookKind_LOOK_KIND_FILM,
		"STYLE":    looksv1.LookKind_LOOK_KIND_STYLE,
		"camera":   looksv1.LookKind_LOOK_KIND_CAMERA,
		"enhance":  looksv1.LookKind_LOOK_KIND_ENHANCE,
		"custom":   looksv1.LookKind_LOOK_KIND_CUSTOM,
		"":         looksv1.LookKind_LOOK_KIND_UNSPECIFIED,
		"nonsense": looksv1.LookKind_LOOK_KIND_UNSPECIFIED,
	}
	for in, want := range cases {
		if got := parseKind(in); got != want {
			t.Errorf("parseKind(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestFormatLook(t *testing.T) {
	l := &looksv1.Look{Id: "noir", Name: "Noir", Kind: looksv1.LookKind_LOOK_KIND_FILM, Builtin: true}
	line := formatLook(l)
	if !strings.Contains(line, "noir") || !strings.Contains(line, "built-in") || !strings.Contains(line, "film") {
		t.Errorf("formatLook missing fields: %q", line)
	}
}

func TestFormatParamsSorted(t *testing.T) {
	got := formatParams(map[string]string{"saturation": "30", "contrast": "12"})
	if got != "contrast=12 saturation=30" {
		t.Errorf("formatParams not sorted/stable: %q", got)
	}
}

func TestReadLookFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "look.json")
	if err := os.WriteFile(path, []byte(`{"name":"Sepia","steps":[{"operation":"filter","kind":"STEP_KIND_DETERMINISTIC","params":{"filter":"sepia"}}]}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	look, err := readLookFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if look.GetName() != "Sepia" || len(look.GetSteps()) != 1 {
		t.Errorf("unexpected look: %+v", look)
	}

	if _, err := readLookFile(""); err == nil {
		t.Error("empty path should error")
	}
	if _, err := readLookFile(filepath.Join(dir, "missing.json")); err == nil {
		t.Error("missing file should error")
	}
}

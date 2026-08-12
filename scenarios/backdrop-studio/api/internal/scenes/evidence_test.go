package scenes

import (
	"os"
	"path/filepath"
	"testing"
)

// The generator sheet.
//
// It is a test rather than a command because the alternative — a throwaway
// probe run by hand — is the exact practice PROBLEMS.md records as the defect:
// twelve style previews were once produced that way and had to be deleted
// because nobody could reproduce them. A test is a command with a name, a
// checked-in expectation of where its output goes, and no way to drift from the
// code it illustrates.
//
// It writes nothing unless BACKDROP_STUDIO_WRITE_EVIDENCE is set, so the normal
// suite stays read-only and fast.
const evidenceEnv = "BACKDROP_STUDIO_WRITE_EVIDENCE"

// evidenceVariants are the parameter sets worth showing per generator: enough
// to see the range each one covers, not a sweep. A generator with palettes
// shows each palette, because a palette is the decision a style author makes
// first.
var evidenceVariants = map[string][]string{
	"mesh":      {`{"palette":0}`, `{"palette":1}`, `{"palette":2,"smear":0.9,"angle":118}`},
	"contour":   {`{"palette":0}`, `{"palette":1,"relief":0.85,"bands":22}`},
	"truchet":   {`{"palette":0}`, `{"palette":1,"cells":4,"line_width":0.3}`},
	"attractor": {`{"palette":0}`, `{"palette":1}`},
	"nebula":    {`{"palette":0}`, `{"palette":1,"dust":0.85}`},
}

func TestGeneratorSheetEvidence(t *testing.T) {
	if os.Getenv(evidenceEnv) == "" {
		t.Skipf("set %s=1 to regenerate docs/evidence/scenes/", evidenceEnv)
	}
	dir := filepath.Join("..", "..", "..", "docs", "evidence", "scenes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create evidence directory: %v", err)
	}
	// The hero surface, because a generator judged at thumbnail size is not
	// judged: the failures this scenario has actually shipped — sub-pixel
	// lines, moire, speckle — are all invisible below delivery resolution.
	const w, h = 1440, 720
	for _, preset := range Presets {
		variants := evidenceVariants[preset]
		if len(variants) == 0 {
			variants = []string{""}
		}
		for i, params := range variants {
			res, err := Render(Request{Preset: preset, Width: w, Height: h, Seed: 7, ParamsJSON: params})
			if err != nil {
				t.Fatalf("render %s variant %d: %v", preset, i, err)
			}
			name := preset + ".png"
			if len(variants) > 1 {
				name = preset + "-" + string(rune('a'+i)) + ".png"
			}
			if err := os.WriteFile(filepath.Join(dir, name), res.PNG, 0o644); err != nil {
				t.Fatalf("write %s: %v", name, err)
			}
		}
	}
}

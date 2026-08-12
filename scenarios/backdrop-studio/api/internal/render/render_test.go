package render

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/imageengine"
	"backdrop-studio/internal/scaffold"
	"backdrop-studio/internal/scenes"

	"github.com/stretchr/testify/require"
)

type fakeExecutor struct{ calls int }

func (f *fakeExecutor) Apply(_ context.Context, input []byte, treatments []string, _ map[string]string) ([]byte, error) {
	f.calls++
	if len(treatments) == 0 {
		return nil, fmt.Errorf("expected treatment chain")
	}
	return append(append([]byte(nil), input...), 0x42), nil
}

type fakeGenerator struct {
	calls int
	last  imageengine.GenerationRequest
}

func (f *fakeGenerator) Generate(_ context.Context, req imageengine.GenerationRequest) ([]byte, error) {
	f.calls++
	f.last = req
	return append([]byte("generated:"), req.Conditioning...), nil
}

func TestSubmitIsReproducibleAndRequiresSelection(t *testing.T) {
	store := NewStore(&fakeExecutor{})
	style := catalog.Style{ID: "horizon", Strategy: "procedural-treated", Subject: "horizon", Placements: []string{"full_bleed"}, Treatments: []string{"duotone"}}
	a, err := store.Submit(style, "full_bleed", 7, 1)
	require.NoError(t, err)
	b, err := store.Submit(style, "full_bleed", 7, 1)
	require.NoError(t, err)
	require.Equal(t, a.Candidates[0].PNG, b.Candidates[0].PNG)
	require.Empty(t, a.SelectedCandidateID)
	_, err = store.Select(a.ID, a.Candidates[0].ID, "operator")
	require.NoError(t, err)
}

func TestModelBackedLanesSubmitConditioningAndAlwaysTreat(t *testing.T) {
	gen := &fakeGenerator{}
	store := NewStoreWithGenerator(&fakeExecutor{}, gen)
	style := catalog.Style{ID: "guided", Strategy: "guided", Subject: "horizon", Placements: []string{"full_bleed"}, Treatments: []string{"duotone"}, Scaffold: &catalog.ScaffoldBinding{Preset: "horizon", Conditioner: "edge"}, Generation: &catalog.GenerationBlock{Role: "image.generate.default", Profile: "PROFILE_QUALITY_FIRST", PromptTemplate: "quiet horizon"}}
	job, err := store.Submit(style, "full_bleed", 11, 1)
	require.NoError(t, err)
	require.Equal(t, 1, gen.calls)
	require.Equal(t, "quiet horizon", gen.last.Prompt)
	require.NotEmpty(t, gen.last.Conditioning)
	require.True(t, job.Candidates[0].ConditioningSubmitted)
	require.True(t, job.Candidates[0].DisclosureRequired)
	require.True(t, job.Candidates[0].TreatmentApplied)

	synth := style
	synth.ID, synth.Strategy, synth.Scaffold = "synth", "synthesized", nil
	job, err = store.Submit(synth, "full_bleed", 12, 1)
	require.NoError(t, err)
	require.Empty(t, gen.last.Conditioning)
	require.False(t, job.Candidates[0].ConditioningSubmitted)
}

func TestModelBackedLaneRefusesWithoutInferenceCapability(t *testing.T) {
	style := catalog.Style{ID: "guided", Strategy: "guided", Subject: "horizon", Placements: []string{"full_bleed"}, Treatments: []string{"duotone"}, Scaffold: &catalog.ScaffoldBinding{Preset: "horizon", Conditioner: "depth"}, Generation: &catalog.GenerationBlock{Role: "image.generate.default", Profile: "PROFILE_QUALITY_FIRST", PromptTemplate: "quiet horizon"}}
	_, err := NewStore(&fakeExecutor{}).Submit(style, "full_bleed", 1, 1)
	require.ErrorContains(t, err, "inference capability")
}

// scenePNG renders a real scene for evidence. Evidence rendered from a 16x10
// synthetic gradient (as pngFixture provides) cannot show whether a contact
// sheet or a placement preview actually works — that is how a colour test chart
// came to be filed as a hero placement preview.
func scenePNG(t *testing.T, preset string, seed int64) []byte {
	t.Helper()
	r, err := scenes.Render(scenes.Request{Preset: preset, Width: deliveryWidth, Height: deliveryHeight, Seed: seed})
	require.NoError(t, err)
	return r.PNG
}

func pngFixture(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 16, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 12), uint8(y * 20), 80, 255})
		}
	}
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, img))
	return b.Bytes()
}

func TestContactSheetAndPlacementPreviewAreLabeledAndPerViewport(t *testing.T) {
	pngBytes := pngFixture(t)
	sheet, err := ContactSheet([]SheetCell{{RowLabel: "HORIZON", ColumnLabel: "HALFTONE", PNG: pngBytes}, {RowLabel: "HORIZON", ColumnLabel: "GRAIN", PNG: pngBytes}}, 2)
	require.NoError(t, err)
	require.NotEmpty(t, sheet)
	previews, err := PreviewPlacements(pngBytes, []string{"full_bleed", "split_panel"}, func(_ []byte, placement string) (float64, bool) { return 4.5, placement == "full_bleed" })
	require.NoError(t, err)
	require.Len(t, previews, 4)
	require.Equal(t, "desktop", previews[0].Viewport)
	require.Equal(t, "mobile", previews[1].Viewport)
	require.True(t, previews[0].Passes)
	require.False(t, previews[2].Passes)
	if dir := os.Getenv("EVIDENCE_DIR"); dir != "" {
		require.NoError(t, os.MkdirAll(dir, 0o755))
		realSheet, err := ContactSheet([]SheetCell{
			{RowLabel: "SUBJECT", ColumnLabel: "HORIZON", PNG: scenePNG(t, "horizon", 7)},
			{RowLabel: "SUBJECT", ColumnLabel: "ARCADE", PNG: scenePNG(t, "arcade", 7)},
			{RowLabel: "SUBJECT", ColumnLabel: "TERRAIN", PNG: scenePNG(t, "terrain", 7)},
			{RowLabel: "SUBJECT", ColumnLabel: "FIELD", PNG: scenePNG(t, "field", 7)},
		}, 4)
		require.NoError(t, err)
		realPreviews, err := PreviewPlacements(scenePNG(t, "horizon", 7), []string{"full_bleed", "split_panel"},
			func(_ []byte, placement string) (float64, bool) { return 4.5, placement == "full_bleed" })
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "contact-sheet.png"), realSheet, 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "placement-preview-desktop.png"), realPreviews[0].PNG, 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "placement-preview-mobile.png"), realPreviews[1].PNG, 0o644))
	}
}

func TestWritePhaseEvidenceWhenRequested(t *testing.T) {
	dir := os.Getenv("EVIDENCE_DIR")
	if dir == "" {
		t.Skip("evidence output is opt-in")
	}
	require.NoError(t, os.MkdirAll(dir, 0o755))

	// Procedural output is the one lane that can be evidenced end to end
	// without a model, so it is evidenced honestly and at delivery size.
	for _, preset := range scenes.Presets {
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, "scene-"+preset+".png"), scenePNG(t, preset, 7), 0o644))
	}

	// Conditioning scaffolds are written as what they are: model INPUT. The
	// previous version of this test wrote the same scaffold bytes twice, once
	// as "guided-conditioning-scaffold.png" and once as
	// "guided-lane-reference.png", which filed the lane's input as its output.
	// A guided or synthesized lane reference cannot be produced without a real
	// generator, so none is emitted rather than fabricating one.
	for _, preset := range []string{"field", "horizon"} {
		sc, err := scaffold.Render(scaffold.Request{
			Preset: preset, Conditioner: "edge",
			Width: scaffoldWidth, Height: scaffoldHeight, Seed: 42,
			Regions: []scaffold.Region{{X: .08, Y: .1, Width: .5, Height: .3}},
		})
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, "conditioning-input-"+preset+".png"), sc.PNG, 0o644))
	}
}

// TestCandidateNeverEqualsItsInput is the regression gate for the defect the
// 2026-08-11 audit found in the shipped evidence: guided-lane-reference.png was
// byte-identical to guided-conditioning-scaffold.png, so the lane's recorded
// output was literally its own input and nothing downstream had run.
//
// It asserts the property directly for every strategy: whatever a lane produces
// must differ from the bytes that lane was handed.
func TestCandidateNeverEqualsItsInput(t *testing.T) {
	base := catalog.Style{Subject: "horizon", Placements: []string{"full_bleed"}, Treatments: []string{"duotone"}}

	for _, tc := range []struct {
		name     string
		mutate   func(catalog.Style) catalog.Style
		withGen  bool
		scaffold bool
	}{
		{name: "procedural-treated", mutate: func(s catalog.Style) catalog.Style {
			s.ID, s.Strategy = "proc", "procedural-treated"
			return s
		}},
		{name: "guided", withGen: true, scaffold: true, mutate: func(s catalog.Style) catalog.Style {
			s.ID, s.Strategy = "guided", "guided"
			s.Scaffold = &catalog.ScaffoldBinding{Preset: "horizon", Conditioner: "depth"}
			s.Generation = &catalog.GenerationBlock{Role: "image.generate.default", Profile: "PROFILE_QUALITY_FIRST", PromptTemplate: "quiet horizon"}
			return s
		}},
		{name: "synthesized", withGen: true, mutate: func(s catalog.Style) catalog.Style {
			s.ID, s.Strategy = "synth", "synthesized"
			s.Generation = &catalog.GenerationBlock{Role: "image.generate.default", Profile: "PROFILE_QUALITY_FIRST", PromptTemplate: "quiet horizon"}
			return s
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			style := tc.mutate(base)
			var store *Store
			gen := &fakeGenerator{}
			if tc.withGen {
				store = NewStoreWithGenerator(&fakeExecutor{}, gen)
			} else {
				store = NewStore(&fakeExecutor{})
			}
			job, err := store.Submit(style, "full_bleed", 21, 1)
			require.NoError(t, err)
			got := job.Candidates[0].PNG
			require.NotEmpty(t, got)

			// The conditioning scaffold is the input the model-backed lanes are
			// handed; a candidate must never be that same artifact.
			if tc.scaffold {
				sc, err := scaffold.Render(scaffold.Request{
					Preset: "horizon", Conditioner: "depth",
					Width: scaffoldWidth, Height: scaffoldHeight, Seed: 21,
				})
				require.NoError(t, err)
				require.NotEqual(t, sc.PNG, got,
					"candidate is byte-identical to its conditioning scaffold — the lane recorded its input as its output")
				require.NotEmpty(t, gen.last.Conditioning)
				require.NotEqual(t, gen.last.Conditioning, got,
					"candidate is byte-identical to the conditioning that was submitted")
			}
		})
	}
}

// TestProceduralLaneRendersFinishedScenesAtDeliverySize pins the scene/scaffold
// split. The procedural lane must render through internal/scenes at a size a
// page can use — not the 320x180 conditioning geometry it used to ship — and
// the result must carry a full tonal range for the ink treatments to map into.
func TestProceduralLaneRendersFinishedScenesAtDeliverySize(t *testing.T) {
	store := NewStore(&fakeExecutor{})
	style := catalog.Style{ID: "proc", Strategy: "procedural-treated", Subject: "horizon", Placements: []string{"full_bleed"}, Treatments: []string{"duotone"}}
	job, err := store.Submit(style, "full_bleed", 7, 1)
	require.NoError(t, err)
	c := job.Candidates[0]
	require.Equal(t, deliveryWidth, c.Width, "procedural candidates must record delivery width")
	require.Equal(t, deliveryHeight, c.Height, "procedural candidates must record delivery height")
	require.Greater(t, deliveryWidth, 1200, "delivery width must be large enough to judge a screen")

	// The candidate PNG is the fake executor's passthrough (input + one byte),
	// so the scene bytes are decodable from its prefix.
	img, err := png.Decode(bytes.NewReader(c.PNG[:len(c.PNG)-1]))
	require.NoError(t, err)
	require.Equal(t, deliveryWidth, img.Bounds().Dx())

	var lo, hi float64 = 1, 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y += 3 {
		for x := b.Min.X; x < b.Max.X; x += 3 {
			r, g, bl, _ := img.At(x, y).RGBA()
			l := (0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(bl>>8)) / 255
			if l < lo {
				lo = l
			}
			if l > hi {
				hi = l
			}
		}
	}
	require.Less(t, lo, 0.15, "scene has no dark values for ink to map into")
	require.Greater(t, hi, 0.85, "scene has no highlight for paper to map into")
}

// TestPlacementsProduceDistinctLayouts pins that the placement argument does
// something. It used to be ignored entirely: PreviewPlacements resized the
// candidate to the viewport and returned the same pixels for every placement,
// so a "placement preview" showed no placement.
func TestPlacementsProduceDistinctLayouts(t *testing.T) {
	src := scenePNG(t, "horizon", 7)
	places := []string{"full_bleed", "split_panel", "framed_inset", "corner_bleed"}
	previews, err := PreviewPlacements(src, places, nil)
	require.NoError(t, err)
	require.Len(t, previews, len(places)*2)

	seen := map[string]string{}
	for _, p := range previews {
		key := p.Placement + "/" + p.Viewport
		sum := fmt.Sprintf("%x", sha256.Sum256(p.PNG))
		for otherKey, otherSum := range seen {
			if otherSum == sum {
				t.Fatalf("%s and %s produced byte-identical previews; the placement is not being applied", key, otherKey)
			}
		}
		seen[key] = sum
	}

	// Every preview must carry copy chrome, so it reads as a layout rather than
	// a bare image: assert a run of identical pixels wide enough to be a bar.
	for _, p := range previews {
		img, err := png.Decode(bytes.NewReader(p.PNG))
		require.NoError(t, err)
		b := img.Bounds()
		widest := 0
		for y := b.Min.Y; y < b.Max.Y; y += 4 {
			run, prev := 0, uint32(0xffffffff)
			for x := b.Min.X; x < b.Max.X; x++ {
				r, g, bl, _ := img.At(x, y).RGBA()
				cur := r>>8<<16 | g>>8<<8 | bl>>8
				if cur == prev {
					run++
					if run > widest {
						widest = run
					}
				} else {
					run, prev = 1, cur
				}
			}
		}
		require.Greater(t, widest, b.Dx()/8,
			"%s/%s has no flat copy bar; the preview is a bare image, not a layout", p.Placement, p.Viewport)
	}
}

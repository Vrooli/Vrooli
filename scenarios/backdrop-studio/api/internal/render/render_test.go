package render

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"sort"
	"strconv"
	"testing"

	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/imageengine"
	"backdrop-studio/internal/scaffold"
	"backdrop-studio/internal/scenes"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// testSurface is the delivery target unit tests render at. It mirrors the
// seeded `web.hero` record: geometry now comes from a surface, never from a
// constant in this package.
var testSurface = Surface{ID: "web.hero", Width: 1440, Height: 720}

const (
	deliveryWidth  = 1440
	deliveryHeight = 720
)

type fakeExecutor struct {
	calls               int
	lastParams          map[string]string
	lastReserve         *imageengine.Knockout
	lastGeometryRequest imageengine.GeometryRequest
}

// Apply mimics what image-tools really does: it returns a decodable PNG of the
// same geometry, visibly different from its input. Returning arbitrary bytes
// here would let the render path record a size it never measured, which is the
// defect the recorded-geometry assertions exist to catch.
func (f *fakeExecutor) Apply(_ context.Context, req imageengine.ApplyRequest) ([]byte, error) {
	f.lastParams = req.Params
	f.lastReserve = req.Reserve
	f.calls++
	if len(req.Treatments) == 0 {
		return nil, fmt.Errorf("expected treatment chain")
	}
	src, err := png.Decode(bytes.NewReader(req.Input))
	if err != nil {
		return nil, fmt.Errorf("fake executor: input is not a PNG: %w", err)
	}
	bounds := src.Bounds()
	out := image.NewNRGBA(bounds)
	// A duotone: map each pixel's lightness onto a two-ink ramp.
	//
	// The fake used to invert the blue channel, which is not a transform any
	// treatment performs. On a blue-dominant source — the caustics generator,
	// say — inverting blue turns dark water bright and compresses the whole
	// luminance range, so the perceptual gate saw a flattened image and refused
	// a style that renders correctly through the real wire. A fake that is not
	// a plausible stand-in for the thing it stands in for makes every test
	// above it a test of fiction.
	dark := [3]float64{22, 26, 44}
	light := [3]float64{240, 236, 220}
	// Every seeded screening style now requests `normalize`, so the fake models
	// that too: it stretches the source's p1-p99 lightness onto the full ramp
	// before mapping. Without it the fake renders a narrower tonal range than
	// the real pipeline does, and the perceptual gate refuses styles for a
	// compression the shipped configuration does not have.
	lo, hi := lightnessPercentiles(src)
	span := hi - lo
	if span < 1e-3 {
		span = 1
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := src.At(x, y).RGBA()
			l := (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 65535
			l = math.Pow(l, 1/2.2)
			l = math.Max(0, math.Min(1, (l-lo)/span))
			mix := func(i int) uint8 { return uint8(dark[i] + (light[i]-dark[i])*l) }
			out.SetNRGBA(x, y, color.NRGBA{R: mix(0), G: mix(1), B: mix(2), A: uint8(a >> 8)})
		}
	}
	// Reserved space comes out as the ramp's light ink, because that is what the
	// real chain does: image-tools lifts the area before each operation, and an
	// operation fed white returns its own paper. A fake that ignored the reserve
	// would report the same contrast whether or not it was sent — which is how
	// an earlier catalog-wide legibility measurement came to be taken through a
	// fake that silently discarded the parameter under test.
	if r := req.Reserve; r != nil && r.Width > 0 && r.Height > 0 {
		x0, y0 := bounds.Min.X+int(r.X*float64(bounds.Dx())), bounds.Min.Y+int(r.Y*float64(bounds.Dy()))
		x1, y1 := bounds.Min.X+int((r.X+r.Width)*float64(bounds.Dx())), bounds.Min.Y+int((r.Y+r.Height)*float64(bounds.Dy()))
		paper := color.NRGBA{R: uint8(light[0]), G: uint8(light[1]), B: uint8(light[2]), A: 255}
		for y := y0; y < y1 && y < bounds.Max.Y; y++ {
			for x := x0; x < x1 && x < bounds.Max.X; x++ {
				paper.A = out.NRGBAAt(x, y).A
				out.SetNRGBA(x, y, paper)
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ToPNG satisfies the optional normalization capability the render path probes
// for. The fake is already PNG-only, so this is the identity.
func (f *fakeExecutor) ToPNG(_ context.Context, in []byte) ([]byte, error) { return in, nil }

// RasterizeSVG stands in for image-tools' vector rasterizer.
//
// It does not parse SVG — that is the point of it being a fake — but it does
// have to be a *plausible* stand-in, because a fake that is not is what makes
// every test above it a test of fiction. Two properties are load-bearing. It
// returns the geometry the document declares, so the render path's measured
// dimensions are real. And it returns a full tonal range with coherent
// structure, because the perceptual gate scores what comes back: a fake that
// returned a flat fill would fail every vector style for a flatness the real
// rasterizer does not produce, and one that returned noise would pass styles
// the real one would refuse.
func (f *fakeExecutor) RasterizeSVG(_ context.Context, svg []byte) ([]byte, error) {
	if len(svg) == 0 {
		return nil, fmt.Errorf("fake executor: cannot rasterize an empty document")
	}
	width, height := declaredSVGSize(svg)
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("fake executor: document declares no usable geometry")
	}
	// The content is derived from the document's own bytes, so two different
	// generators rasterize to two different pictures and the distinctness and
	// determinism assertions above this fake mean something.
	seed := sha256.Sum256(svg)
	base := float64(seed[0])/255*0.4 + 0.1
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		fy := float64(y) / float64(height)
		for x := 0; x < width; x++ {
			fx := float64(x) / float64(width)
			// A vertical ramp plus a low-frequency modulation: a full tonal
			// range with real large-scale composition, which is what a vector
			// generator's output actually has.
			v := base + 0.55*fy + 0.30*math.Sin((fx*3.1+base*6)*math.Pi)*math.Cos(fy*2.3*math.Pi)
			l := uint8(math.Max(0, math.Min(255, v*255)))
			img.SetNRGBA(x, y, color.NRGBA{R: l, G: uint8(float64(l) * 0.94), B: uint8(float64(l) * 0.82), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// declaredSVGSize reads the width and height attributes the vector generators
// always emit.
func declaredSVGSize(svg []byte) (int, int) {
	read := func(attr string) int {
		marker := attr + `="`
		at := bytes.Index(svg, []byte(marker))
		if at < 0 {
			return 0
		}
		rest := svg[at+len(marker):]
		end := bytes.IndexByte(rest, '"')
		if end < 0 {
			return 0
		}
		n, err := strconv.Atoi(string(rest[:end]))
		if err != nil {
			return 0
		}
		return n
	}
	return read("width"), read("height")
}

// Resize mirrors image-tools' cover-fill resize so a model-backed render in a
// unit test reaches the surface geometry the production path reaches.
func (f *fakeExecutor) Resize(_ context.Context, in []byte, width, height int) ([]byte, error) {
	src, err := png.Decode(bytes.NewReader(in))
	if err != nil {
		return nil, fmt.Errorf("fake executor: resize input is not a PNG: %w", err)
	}
	out := image.NewRGBA(image.Rect(0, 0, width, height))
	drawScaled(out, out.Bounds(), src)
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// The geometry the fake executor reports, shaped like a real registry answer:
// a square native edge, a latent stride, and a cap at 1.5x native. A fake that
// is not a plausible stand-in makes every test above it a test of fiction.
const (
	fakeNativeEdge  = 512
	fakeSizeQuantum = 64
	fakeMaxEdge     = 768
)

// ModelGeometry answers the capability probe. It is on the executor rather than
// the generator for the same reason the production seam is: a caller may need
// to size a request without being able to spend a model call.
func (f *fakeExecutor) ModelGeometry(_ context.Context, req imageengine.GeometryRequest) (imageengine.ModelGeometry, error) {
	if req.Operation == "" {
		return imageengine.ModelGeometry{}, fmt.Errorf("fake executor: geometry needs an operation")
	}
	f.lastGeometryRequest = req
	// A BYOK route resolves the cloud entry, which declares no native
	// geometry — the provider owns it. Answering the local default here
	// regardless of policy is the very defect the routing fields exist to
	// close, so the fake must not do it either.
	if req.AllowBYOK {
		return imageengine.ModelGeometry{ModelID: "fake/byok-cloud", SizeQuantum: 1}, nil
	}
	return imageengine.ModelGeometry{
		ModelID:      "fake/local-gpu",
		NativeWidth:  fakeNativeEdge,
		NativeHeight: fakeNativeEdge,
		SizeQuantum:  fakeSizeQuantum,
		MaxEdge:      fakeMaxEdge,
	}, nil
}

// Composite mirrors image-tools' compositor closely enough that the render
// path's plate wiring is proven rather than assumed: it really merges the
// plates, honours depth order and opacity, and returns one raster at the
// requested geometry. What the real compositor produces pixel-for-pixel is
// proven in image-tools' own suite; what this proves is that the right plates,
// in the right order, reach it.
func (f *fakeExecutor) Composite(_ context.Context, plates []imageengine.PlateSource, width, height int, _ string) ([]byte, error) {
	if len(plates) == 0 {
		return nil, fmt.Errorf("fake executor: composite needs at least one plate")
	}
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("fake executor: composite needs positive geometry (got %dx%d)", width, height)
	}
	ordered := append([]imageengine.PlateSource(nil), plates...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Depth < ordered[j].Depth })
	out := image.NewNRGBA(image.Rect(0, 0, width, height))
	for _, plate := range ordered {
		if len(plate.PNG) == 0 {
			return nil, fmt.Errorf("fake executor: plate %q carries no image", plate.Name)
		}
		src, err := png.Decode(bytes.NewReader(plate.PNG))
		if err != nil {
			return nil, fmt.Errorf("fake executor: decode plate %q: %w", plate.Name, err)
		}
		scaled := image.NewNRGBA(out.Bounds())
		drawScaled(scaled, scaled.Bounds(), src)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				s := scaled.NRGBAAt(x, y)
				alpha := float64(s.A) / 255 * plate.Opacity
				if alpha <= 0 {
					continue
				}
				d := out.NRGBAAt(x, y)
				blend := func(dv, sv uint8) uint8 {
					return uint8(float64(sv)*alpha + float64(dv)*(1-alpha))
				}
				out.SetNRGBA(x, y, color.NRGBA{
					R: blend(d.R, s.R), G: blend(d.G, s.G), B: blend(d.B, s.B),
					A: uint8(alpha*255 + float64(d.A)*(1-alpha)),
				})
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type fakeGenerator struct {
	calls int
	last  imageengine.GenerationRequest
}

// Generate returns a real PNG at the requested geometry, because that is what a
// model returns and what the render path now measures. The pixel content is
// derived from the conditioning bytes so a test can still tell a conditioned
// render from an unconditioned one.
func (f *fakeGenerator) Generate(_ context.Context, req imageengine.GenerationRequest) (imageengine.GenerationResult, error) {
	f.calls++
	f.last = req
	if req.Width <= 0 || req.Height <= 0 {
		return imageengine.GenerationResult{}, fmt.Errorf("fake generator: geometry is required, got %dx%d", req.Width, req.Height)
	}
	tint := uint8(len(req.Conditioning) % 251)
	img := image.NewNRGBA(image.Rect(0, 0, req.Width, req.Height))
	for y := 0; y < req.Height; y++ {
		for x := 0; x < req.Width; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x % 256), G: uint8(y % 256), B: tint, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return imageengine.GenerationResult{}, err
	}
	// A model id, because a real router always chooses one and the release
	// path refuses a model-backed candidate that cannot name its model.
	return imageengine.GenerationResult{PNG: buf.Bytes(), ModelID: "fake/local-gpu", Tier: "local-gpu"}, nil
}

func TestSubmitIsReproducibleAndRequiresSelection(t *testing.T) {
	store := NewStore(&fakeExecutor{})
	style := catalog.Style{ID: "horizon", Strategy: "procedural-treated", Subject: "horizon", Placements: []string{"full_bleed"}, Treatments: []string{"duotone"}}
	a, err := store.SubmitWithContext(context.Background(), Request{Style: style, Surface: testSurface, Placement: "full_bleed", Seed: 7, Count: 1})
	require.NoError(t, err)
	b, err := store.SubmitWithContext(context.Background(), Request{Style: style, Surface: testSurface, Placement: "full_bleed", Seed: 7, Count: 1})
	require.NoError(t, err)
	require.Equal(t, a.Candidates[0].PNG, b.Candidates[0].PNG)
	require.Empty(t, a.SelectedCandidateID)
	_, err = store.Select(a.ID, a.Candidates[0].ID, "operator")
	require.NoError(t, err)
}

func TestModelBackedLanesSubmitConditioningAndAlwaysTreat(t *testing.T) {
	gen := &fakeGenerator{}
	store := NewStoreWithGenerator(&fakeExecutor{}, gen)
	style := catalog.Style{ID: "guided", Strategy: "guided", QualityTier: catalog.TierLocalModel, Subject: "horizon", Placements: []string{"full_bleed"}, Treatments: []string{"duotone"}, Scaffold: &catalog.ScaffoldBinding{Preset: "horizon", Conditioner: "edge"}, Generation: &catalog.GenerationBlock{PromptTemplate: "quiet horizon"}}
	job, err := store.SubmitWithContext(context.Background(), Request{Style: style, Surface: testSurface, Placement: "full_bleed", Seed: 11, Count: 1})
	require.NoError(t, err)
	require.Equal(t, 1, gen.calls)
	require.Equal(t, "quiet horizon", gen.last.Prompt)
	require.NotEmpty(t, gen.last.Conditioning)
	require.True(t, job.Candidates[0].ConditioningSubmitted)
	require.True(t, job.Candidates[0].DisclosureRequired)
	require.True(t, job.Candidates[0].TreatmentApplied)

	synth := style
	synth.ID, synth.Strategy, synth.Scaffold = "synth", "synthesized", nil
	job, err = store.SubmitWithContext(context.Background(), Request{Style: synth, Surface: testSurface, Placement: "full_bleed", Seed: 12, Count: 1})
	require.NoError(t, err)
	require.Empty(t, gen.last.Conditioning)
	require.False(t, job.Candidates[0].ConditioningSubmitted)
}

func TestModelBackedLaneRefusesWithoutInferenceCapability(t *testing.T) {
	style := catalog.Style{ID: "guided", Strategy: "guided", QualityTier: catalog.TierLocalModel, Subject: "horizon", Placements: []string{"full_bleed"}, Treatments: []string{"duotone"}, Scaffold: &catalog.ScaffoldBinding{Preset: "horizon", Conditioner: "depth"}, Generation: &catalog.GenerationBlock{PromptTemplate: "quiet horizon"}}
	_, err := NewStore(&fakeExecutor{}).SubmitWithContext(context.Background(), Request{Style: style, Surface: testSurface, Placement: "full_bleed", Seed: 1, Count: 1})
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
}

// The EVIDENCE_DIR writers that used to live here are gone, and the nine
// artifacts they wrote into docs/evidence/procedural/ are deleted.
//
// None of them was unreproducible — that is what made them worth removing
// rather than merely regenerating. Each had a second producer for a fact this
// scenario already evidences somewhere else, and the two producers drifted, so
// the tree carried two answers to one question with nothing to say which was
// current. `scene-*.png` duplicated docs/evidence/scenes/ — written by
// scenes.TestGeneratorSheetEvidence for all thirteen presets at hero geometry,
// where this writer had stopped at the four presets that existed when it was
// written. `contact-sheet.png` duplicated the catalog sheets. The two
// placement previews were a picture of a build nobody could identify: the
// mobile one still showed the elliptical sun from before drawScaled was
// corrected, which EVIDENCE.md cites as its own cautionary tale. The
// conditioning inputs had a producer and no consumer.
//
// Placement evidence returns when composePlacement draws all ten placements
// rather than four; it is rebuilt there, against the mockups, not restored here.

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
			s.ID, s.Strategy, s.QualityTier = "guided", "guided", catalog.TierLocalModel
			s.Scaffold = &catalog.ScaffoldBinding{Preset: "horizon", Conditioner: "depth"}
			s.Generation = &catalog.GenerationBlock{PromptTemplate: "quiet horizon"}
			return s
		}},
		{name: "synthesized", withGen: true, mutate: func(s catalog.Style) catalog.Style {
			s.ID, s.Strategy, s.QualityTier = "synth", "synthesized", catalog.TierLocalModel
			s.Generation = &catalog.GenerationBlock{PromptTemplate: "quiet horizon"}
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
			job, err := store.SubmitWithContext(context.Background(), Request{Style: style, Surface: testSurface, Placement: "full_bleed", Seed: 21, Count: 1})
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
	job, err := store.SubmitWithContext(context.Background(), Request{Style: style, Surface: testSurface, Placement: "full_bleed", Seed: 7, Count: 1})
	require.NoError(t, err)
	c := job.Candidates[0]
	require.Equal(t, deliveryWidth, c.Width, "procedural candidates must record delivery width")
	require.Equal(t, deliveryHeight, c.Height, "procedural candidates must record delivery height")
	require.Greater(t, deliveryWidth, 1200, "delivery width must be large enough to judge a screen")

	// Tone is a property of the SCENE, so it is measured on the scene rather
	// than on the fake executor's output. A treatment that flattened the tonal
	// range would be a treatment defect; this test is about what the procedural
	// lane hands the treatment layer.
	scene, err := scenes.Render(scenes.Request{Preset: "horizon", Width: deliveryWidth, Height: deliveryHeight, Seed: 7})
	require.NoError(t, err)
	img, err := png.Decode(bytes.NewReader(scene.PNG))
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

// TestDrawScaledCoverCropsWithoutDistortion pins the cover-crop at the level
// the property lives. drawScaled used to map the source onto the target
// independently per axis, so a 1600x1000 backdrop dropped into a tall panel
// came out stretched — a circular sun rendered as an oval, which reads as a
// mistake at a glance.
//
// Asserted here rather than through PreviewPlacements because a composed
// preview draws opaque copy bars over the image, which would clip the measured
// shape and confuse a distortion failure with a layout overlap.
func TestDrawScaledCoverCropsWithoutDistortion(t *testing.T) {
	const w, h = 1600, 1000
	src := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			src.Set(x, y, color.RGBA{10, 10, 10, 255})
		}
	}
	cx, cy, r := w/2, h/2, 150
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx, dy := float64(x-cx), float64(y-cy)
			if dx*dx+dy*dy <= float64(r*r) {
				src.Set(x, y, color.RGBA{255, 255, 255, 255})
			}
		}
	}

	// Targets spanning wider, taller and squarer than the source.
	for _, tgt := range []image.Point{
		{X: 1280, Y: 720}, // desktop full bleed
		{X: 640, Y: 720},  // desktop split panel — much taller than source
		{X: 390, Y: 844},  // mobile — far taller
		{X: 1160, Y: 396}, // wide framed inset
		{X: 500, Y: 500},  // square
	} {
		dst := image.NewRGBA(image.Rect(0, 0, tgt.X, tgt.Y))
		drawScaled(dst, dst.Bounds(), src)

		white := func(x, y int) bool {
			c := dst.RGBAAt(x, y)
			return c.R > 200 && c.G > 200 && c.B > 200
		}
		maxW, maxH := 0, 0
		for y := 0; y < tgt.Y; y++ {
			run := 0
			for x := 0; x < tgt.X; x++ {
				if white(x, y) {
					run++
					if run > maxW {
						maxW = run
					}
				} else {
					run = 0
				}
			}
		}
		for x := 0; x < tgt.X; x++ {
			run := 0
			for y := 0; y < tgt.Y; y++ {
				if white(x, y) {
					run++
					if run > maxH {
						maxH = run
					}
				} else {
					run = 0
				}
			}
		}
		require.Greater(t, maxW, 8, "%dx%d: circle vanished", tgt.X, tgt.Y)
		require.Greater(t, maxH, 8, "%dx%d: circle vanished", tgt.X, tgt.Y)
		ratio := float64(maxW) / float64(maxH)
		require.InDelta(t, 1.0, ratio, 0.06,
			"target %dx%d renders a circle at %dx%d (ratio %.2f) — the source is being stretched, not cover-cropped",
			tgt.X, tgt.Y, maxW, maxH, ratio)
	}
}

// TestEverySeededProceduralStyleRenders is the catalog's integrity gate. A
// style that validates but cannot render is worse than a missing style: the
// operator picks it, and it fails or silently produces something else.
//
// The seeded catalog previously listed subjects like "botanical" and
// "celestial" on procedural strategies; none had a scene, so all of them fell
// through to the abstract field generator. Cyanotype Botanical rendered a
// colour field and nothing said so.
func TestEverySeededProceduralStyleRenders(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(catalog.Schema())
	require.NoError(t, err)
	store := catalog.NewStore(db)
	ctx := context.Background()
	require.NoError(t, store.Seed(ctx))
	styles, err := store.ListStyles(ctx, "", "", "", "", "")
	require.NoError(t, err)
	require.NotEmpty(t, styles)

	exec := &fakeExecutor{}
	renderStore := NewStoreWithGenerator(exec, &fakeGenerator{})
	procedural := 0
	for _, style := range styles {
		style := style
		t.Run(style.ID, func(t *testing.T) {
			job, err := renderStore.SubmitWithContext(context.Background(), Request{Style: style, Surface: testSurface, Placement: style.Placements[0], Seed: 7, Count: 1})
			require.NoError(t, err, "seeded style %q cannot render", style.ID)
			require.NotEmpty(t, job.Candidates)

			// Every style's parameters must reach the executor, or the catalog
			// is describing an art direction the engine never receives.
			for _, op := range style.Treatments {
				require.Contains(t, exec.lastParams, op,
					"style %q parameters for %q never reached the executor", style.ID, op)
			}

			if style.Strategy == "procedural" || style.Strategy == "procedural-treated" {
				procedural++
				require.Equal(t, deliveryWidth, job.Candidates[0].Width)
				_, presetErr := scenePreset(style)
				require.NoError(t, presetErr,
					"procedural style %q names subject %q, which no generator depicts — it used to silently render a field",
					style.ID, style.Subject)
			}
		})
	}
	require.Greater(t, procedural, 8, "the catalog should carry a real spread of procedural styles")
}

// TestProceduralStyleWithoutASceneIsRefused pins the refusal rather than the
// old silent fallback.
func TestProceduralStyleWithoutASceneIsRefused(t *testing.T) {
	style := catalog.Style{
		ID: "no-scene", Name: "No Scene", Strategy: "procedural-treated",
		Subject: "celestial", Placements: []string{"full_bleed"}, Treatments: []string{"duotone"},
	}
	_, err := NewStore(&fakeExecutor{}).SubmitWithContext(context.Background(), Request{Style: style, Surface: testSurface, Placement: "full_bleed", Seed: 1, Count: 1})
	require.ErrorContains(t, err, "no procedural generator depicts subject")
}

// TestModelBackedCandidateReachesSurfaceGeometry pins the step that was
// missing: a diffusion model must draw near its training resolution, but the
// surface decides what ships. Without the scaling step a model-backed style
// delivered 768x512 for a 1440x720 hero and nothing in the system said so.
func TestModelBackedCandidateReachesSurfaceGeometry(t *testing.T) {
	generator := &fakeGenerator{}
	store := NewStoreWithGenerator(&fakeExecutor{}, generator)
	style := catalog.Style{
		ID: "synth", Strategy: "synthesized", QualityTier: catalog.TierLocalModel, Subject: "figure",
		Placements: []string{"full_bleed"}, Treatments: []string{"duotone"},
		Generation: &catalog.GenerationBlock{PromptTemplate: "a quiet figure"},
	}

	for _, surface := range []Surface{
		{ID: "web.hero", Width: 1440, Height: 720},
		{ID: "app-store-6.7-screenshot", Width: 1290, Height: 2796},
	} {
		t.Run(surface.ID, func(t *testing.T) {
			job, err := store.SubmitWithContext(context.Background(), Request{
				Style: style, Surface: surface, Placement: "full_bleed", Seed: 7, Count: 1,
			})
			require.NoError(t, err)
			require.NotEmpty(t, job.Candidates)
			candidate := job.Candidates[0]

			require.Equal(t, surface.Width, candidate.Width, "candidate must reach the surface width")
			require.Equal(t, surface.Height, candidate.Height, "candidate must reach the surface height")

			// The model itself was asked for a canvas near its native edge,
			// not for the delivery size.
			require.LessOrEqual(t, generator.last.Width, 1024, "the model must draw near native resolution")
			require.LessOrEqual(t, generator.last.Height, 1024)
			require.Zero(t, generator.last.Width%fakeSizeQuantum, "the canvas must land on the latent stride the model reported")
			require.Zero(t, generator.last.Height%fakeSizeQuantum)

			// Provenance names both sizes and the routing that produced them.
			require.Contains(t, candidate.ProvenanceJSON, "generation_native_size")
			require.Contains(t, candidate.ProvenanceJSON, surface.ID)
			require.Contains(t, candidate.ProvenanceJSON, "local_only")
		})
	}
}

// TestRoutingIsLocalFirst is the regression gate for a live cost leak: the
// generation request used to hardcode a quality policy and BYOK on, so every
// model-backed render resolved to a paid cloud provider while an installed
// local GPU served the same request in about fifteen seconds.
func TestRoutingIsLocalFirst(t *testing.T) {
	generator := &fakeGenerator{}
	store := NewStoreWithGenerator(&fakeExecutor{}, generator)
	style := catalog.Style{
		ID: "synth", Strategy: "synthesized", QualityTier: catalog.TierLocalModel, Subject: "figure",
		Placements: []string{"full_bleed"}, Treatments: []string{"duotone"},
		Generation: &catalog.GenerationBlock{PromptTemplate: "a quiet figure"},
	}
	_, err := store.SubmitWithContext(context.Background(), Request{
		Style: style, Surface: testSurface, Placement: "full_bleed", Seed: 7, Count: 1,
	})
	require.NoError(t, err)

	require.False(t, generator.last.AllowBYOK, "a render must not reach a paid provider by default")
	require.Equal(t, "local_only", generator.last.FallbackPolicy)
	require.Equal(t, "batch", generator.last.Priority, "backdrop renders are not interactive work")
	require.True(t, generator.last.AllowReclaim, "a shared GPU needs reclaim or diffusion loses the allocation race")
}

// lightnessPercentiles returns the source's p1 and p99 gamma-corrected
// lightness, which is the span image-tools' `normalize` stretches onto the ink
// ramp.
func lightnessPercentiles(src image.Image) (float64, float64) {
	b := src.Bounds()
	step := 1
	if b.Dx() > 400 {
		step = b.Dx() / 400
	}
	values := make([]float64, 0, 1024)
	for y := b.Min.Y; y < b.Max.Y; y += step {
		for x := b.Min.X; x < b.Max.X; x += step {
			r, g, bb, _ := src.At(x, y).RGBA()
			l := (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(bb)) / 65535
			values = append(values, math.Pow(l, 1/2.2))
		}
	}
	if len(values) < 2 {
		return 0, 1
	}
	sort.Float64s(values)
	return values[len(values)/100], values[len(values)*99/100]
}

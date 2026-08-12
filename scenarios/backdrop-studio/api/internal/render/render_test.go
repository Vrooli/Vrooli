package render

import (
	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/imageengine"
	"backdrop-studio/internal/scaffold"
	"bytes"
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
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
		require.NoError(t, os.WriteFile(filepath.Join(dir, "contact-sheet.png"), sheet, 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "placement-preview-desktop.png"), previews[0].PNG, 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "placement-preview-mobile.png"), previews[1].PNG, 0o644))
	}
}

func TestWritePhaseEvidenceWhenRequested(t *testing.T) {
	dir := os.Getenv("EVIDENCE_DIR")
	if dir == "" {
		t.Skip("evidence output is opt-in")
	}
	require.NoError(t, os.MkdirAll(dir, 0o755))
	guided, err := scaffold.Render(scaffold.Request{Preset: "field", Conditioner: "edge", Width: 320, Height: 180, Seed: 42, Regions: []scaffold.Region{{X: .08, Y: .1, Width: .5, Height: .3}}})
	require.NoError(t, err)
	synthesized, err := scaffold.Render(scaffold.Request{Preset: "horizon", Width: 320, Height: 180, Seed: 42})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "guided-conditioning-scaffold.png"), guided.PNG, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "guided-lane-reference.png"), guided.PNG, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "synthesized-lane-reference.png"), synthesized.PNG, 0o644))
}

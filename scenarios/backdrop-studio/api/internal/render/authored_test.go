package render

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/vector"
	"backdrop-studio/internal/vector/authoring"
)

// generatorMap is the smallest thing that satisfies GeneratorStore.
type generatorMap map[string]authoring.Generator

func (m generatorMap) AuthoredGenerator(_ context.Context, id string) (authoring.Generator, error) {
	g, ok := m[id]
	if !ok {
		return authoring.Generator{}, &catalog.UnknownGeneratorError{ID: id}
	}
	return g, nil
}

func authoredStyle(preset string) catalog.Style {
	return catalog.Style{
		ID: "authored-style", Strategy: "vector", Subject: "non_representational",
		Placements: []string{"full_bleed"},
		Scaffold:   &catalog.ScaffoldBinding{Preset: preset, ParamsJSON: `{"marks": 180}`},
		Inks: map[string]string{
			vector.InkPaper: "#f1e7d2", vector.InkInk: "#123a6b", vector.InkAccent: "#b1442c",
		},
	}
}

func authoredFixture(id string) authoring.Generator {
	g := authoring.Generator{
		ID:   id,
		Name: "Authored Scatter",
		Template: `<rect width="{{f .W}}" height="{{f .H}}" fill="{{paper}}"/>
{{- $frame := .}}
{{range $i := seq ($frame.Param "marks")}}
<circle cx="{{f (mul $frame.W ($frame.Rand $i))}}" cy="{{f (mul $frame.H (add 0.15 (mul 0.7 ($frame.Rand (add $i 31)))))}}" r="{{f (mul $frame.H 0.01)}}" fill="{{ink}}"/>
{{end}}`,
		Params:  []authoring.ParamSpec{{Name: "marks", Min: 20, Max: 600, Default: 200}},
		Inks:    []string{vector.InkPaper, vector.InkInk},
		Prompt:  "a scatter",
		ModelID: "test/model",
	}
	g.Validation = authoring.Validate(g, vector.Presets)
	return g
}

// A style may bind to a generator a model wrote, and it renders like any other.
func TestAStyleRendersFromAnAuthoredGenerator(t *testing.T) {
	fixture := authoredFixture("authored-scatter")
	require.Truef(t, fixture.Validation.Passed, "fixture invalid: %v", fixture.Validation.Refusals)

	store := NewStore(&fakeExecutor{}).WithGeneratorStore(generatorMap{fixture.ID: fixture})
	job, err := store.SubmitWithContext(context.Background(), Request{
		Style: authoredStyle(fixture.ID), Surface: testSurface, Placement: "full_bleed", Seed: 7, Count: 1,
	})
	require.NoError(t, err)
	require.NotEmpty(t, job.Candidates)
	require.NotEmpty(t, job.Candidates[0].SVG, "an authored generator ships its source like a hand-written one")
	require.NotContains(t, string(job.Candidates[0].SVG), "$brand.", "inks resolve before the document leaves")
}

// The rule the plan names: a catalog row naming a generator that is not stored
// and validated is refused, by name.
func TestAStyleBoundToAnUnstoredGeneratorIsRefusedByName(t *testing.T) {
	store := NewStore(&fakeExecutor{}).WithGeneratorStore(generatorMap{})
	_, err := store.SubmitWithContext(context.Background(), Request{
		Style: authoredStyle("never-authored"), Surface: testSurface, Placement: "full_bleed", Seed: 7, Count: 1,
	})
	require.Error(t, err)
	var unknown *catalog.UnknownGeneratorError
	require.ErrorAs(t, err, &unknown, "the refusal must stay typed through the render path")
	require.Equal(t, "never-authored", unknown.ID)
}

// A built-in preset always wins, whatever a stored generator calls itself.
// Validation refuses the collision at storage time; this proves the render path
// honours the same order even if one got in another way.
func TestABuiltInPresetIsNeverShadowedByAnAuthoredGenerator(t *testing.T) {
	shadow := authoredFixture("colonnade")
	store := NewStore(&fakeExecutor{}).WithGeneratorStore(generatorMap{shadow.ID: shadow})
	style := authoredStyle("colonnade")
	style.Subject = "statuary_architecture"

	job, err := store.SubmitWithContext(context.Background(), Request{
		Style: style, Surface: testSurface, Placement: "full_bleed", Seed: 7, Count: 1,
	})
	require.NoError(t, err)
	// The hand-written colonnade declares depth planes and an ink filter; the
	// authored envelope declares exactly one plane named "authored".
	document := string(job.Candidates[0].SVG)
	require.NotContains(t, document, `data-plane="authored"`, "the hand-written generator must have drawn this")
	require.Contains(t, document, "<filter", "the hand-written colonnade declares its ink filter")
}

// With no generator store configured, a built-in vector style still renders.
// A host that never authored anything must not lose the shipping lane.
func TestTheVectorLaneWorksWithNoGeneratorStore(t *testing.T) {
	style := authoredStyle("")
	style.Subject = "cartographic"
	store := NewStore(&fakeExecutor{})
	job, err := store.SubmitWithContext(context.Background(), Request{
		Style: style, Surface: testSurface, Placement: "full_bleed", Seed: 7, Count: 1,
	})
	require.NoError(t, err)
	require.NotEmpty(t, job.Candidates)
}

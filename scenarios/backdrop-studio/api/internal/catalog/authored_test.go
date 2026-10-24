package catalog

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"backdrop-studio/internal/vector"
	"backdrop-studio/internal/vector/authoring"
)

func validGenerator() authoring.Generator {
	g := authoring.Generator{
		ID:   "authored-band",
		Name: "Authored Band",
		Template: `<rect width="{{f .W}}" height="{{f .H}}" fill="{{paper}}"/>
{{- $frame := .}}{{- $n := int ($frame.Param "marks")}}
{{range $i := seq $n}}
<circle cx="{{f (mul $frame.W ($frame.Rand $i))}}" cy="{{f (mul $frame.H (add 0.2 (mul 0.6 ($frame.Rand (add $i 7)))))}}" r="{{f (mul $frame.H 0.008)}}" fill="{{ink}}"/>
{{end}}`,
		Params:  []authoring.ParamSpec{{Name: "marks", Min: 20, Max: 600, Default: 220}},
		Inks:    []string{vector.InkPaper, vector.InkInk},
		Prompt:  "a scattered band",
		ModelID: "test/model",
	}
	g.Validation = authoring.Validate(g, vector.Presets)
	return g
}

func TestAValidatedGeneratorRoundTripsThroughTheStore(t *testing.T) {
	store := NewStore(freshDB(t))
	ctx := context.Background()
	require.NoError(t, store.Seed(ctx))

	g := validGenerator()
	require.Truef(t, g.Validation.Passed, "fixture must be valid; refusals: %v", g.Validation.Refusals)
	require.NoError(t, store.PutAuthoredGenerator(ctx, g))

	read, err := store.AuthoredGenerator(ctx, g.ID)
	require.NoError(t, err)
	require.Equal(t, g.Template, read.Template)
	require.Equal(t, g.Params, read.Params)
	require.Equal(t, g.Inks, read.Inks)
	require.Equal(t, g.ModelID, read.ModelID, "the authoring model must survive; it is the disclosure record")
	require.Equal(t, g.Prompt, read.Prompt, "the prompt must survive; without it nobody can re-author or review it")
	require.True(t, read.Validation.Passed)

	listed, err := store.ListAuthoredGenerators(ctx)
	require.NoError(t, err)
	require.Len(t, listed, 1)
}

// The rule that makes a stored generator trustworthy: only a passing report
// gets in. A failed one is refused rather than stored with a flag, because a
// later code path may reasonably assume a stored generator was checked.
func TestTheStoreRefusesAGeneratorThatFailedValidation(t *testing.T) {
	store := NewStore(freshDB(t))
	ctx := context.Background()
	require.NoError(t, store.Seed(ctx))

	g := validGenerator()
	g.Params = nil // the template still reads "marks"
	g.Validation = authoring.Validate(g, vector.Presets)
	require.False(t, g.Validation.Passed)

	err := store.PutAuthoredGenerator(ctx, g)
	require.ErrorContains(t, err, "did not pass validation")

	_, readErr := store.AuthoredGenerator(ctx, g.ID)
	require.Error(t, readErr, "nothing may be readable after a refused store")
}

// A generator with no authoring model or no prompt cannot be disclosed or
// re-authored, so it is refused even when its template is fine.
func TestTheStoreRefusesAGeneratorWithNoProvenance(t *testing.T) {
	ctx := context.Background()
	for name, mutate := range map[string]func(*authoring.Generator){
		"no model":  func(g *authoring.Generator) { g.ModelID = "" },
		"no prompt": func(g *authoring.Generator) { g.Prompt = "" },
	} {
		t.Run(name, func(t *testing.T) {
			store := NewStore(freshDB(t))
			require.NoError(t, store.Seed(ctx))
			g := validGenerator()
			mutate(&g)
			require.Error(t, store.PutAuthoredGenerator(ctx, g))
		})
	}
}

// An unknown id is a typed refusal, because the handler edge maps it to a
// precondition failure and the fix — author it, or change the binding — is
// never a retry.
func TestAnUnknownGeneratorIsATypedRefusal(t *testing.T) {
	store := NewStore(freshDB(t))
	ctx := context.Background()
	require.NoError(t, store.Seed(ctx))

	_, err := store.AuthoredGenerator(ctx, "never-authored")
	var unknown *UnknownGeneratorError
	require.ErrorAs(t, err, &unknown)
	require.Equal(t, "never-authored", unknown.ID)
	require.Contains(t, err.Error(), "author and store it")
}

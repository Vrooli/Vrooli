package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"

	"backdrop-studio/internal/scenes"

	"github.com/stretchr/testify/require"
)

// The art-direction distinctness lint.
//
// The audit's central finding was that sixteen named styles rendered four
// pictures: `aquatic`, `atmospheric` and `horizon` all drew the horizon scene,
// and Ukiyo Tide, Riso Horizon, City Pop Horizon and Solar Bloom Horizon were
// the same image under different filters. Nothing anywhere said so. Phase 7
// fixed the substitution; this test is what stops it coming back, and it is
// deliberately a build-time check rather than a review checklist, because "are
// any two of these forty styles the same picture?" is exactly the question a
// human reviewer stops asking around style twelve.
//
// Two styles are the same picture when everything that decides the pixels
// agrees: the source they draw, the chain they run over it, and the parameters
// that chain runs with. Anything else — a different name, lineage, or set of
// placements — is metadata, and metadata does not make a second style.

// pictureKey is everything that decides what a style's render looks like.
func pictureKey(t *testing.T, v Style) string {
	t.Helper()
	source := v.Subject
	if v.Strategy == "procedural" || v.Strategy == "procedural-treated" {
		// The generator, not the subject, is what the procedural lane draws:
		// four generators share `non_representational`, so keying on the
		// subject would report four distinct pictures as one.
		preset, err := scenes.ResolvePreset(v.Subject, declaredPreset(v))
		require.NoErrorf(t, err, "style %q names a subject the procedural lane cannot draw", v.ID)
		source = "scene:" + preset + ":" + canonicalJSON(t, scaffoldParamsJSON(v))
	} else if v.Generation != nil {
		// For a model-backed style the prompt is the picture.
		source = "model:" + strings.TrimSpace(v.Generation.PromptTemplate)
	}
	params := make([]string, 0, len(v.TreatmentParams))
	for _, treatment := range v.Treatments {
		params = append(params, treatment+"="+canonicalJSON(t, v.TreatmentParams[treatment]))
	}
	return source + "|" + strings.Join(params, ";")
}

func declaredPreset(v Style) string {
	if v.Scaffold == nil {
		return ""
	}
	return v.Scaffold.Preset
}

func scaffoldParamsJSON(v Style) string {
	if v.Scaffold == nil {
		return ""
	}
	return v.Scaffold.ParamsJSON
}

// canonicalJSON re-encodes an object so that key order and whitespace cannot
// make two identical parameter sets look different. Without it a style could
// evade the lint by reordering its own JSON, which is not a distinction anyone
// can see in the rendered image.
func canonicalJSON(t *testing.T, raw string) string {
	t.Helper()
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		// Not an object: compare it as written. A malformed parameter block is
		// the wire-contract test's finding to report, not this one's.
		return raw
	}
	keys := make([]string, 0, len(decoded))
	for k := range decoded {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s:%v", k, decoded[k]))
	}
	return strings.Join(parts, ",")
}

// TestNoTwoSettledStylesRenderTheSamePicture checks the catalog an install
// actually ends up with, not each version on the way there. A style corrected
// by a later seed version was never wrong from the install's point of view, and
// judging the versions separately would fail on exactly the corrections that
// fixed this defect in the first place.
func TestNoTwoSettledStylesRenderTheSamePicture(t *testing.T) {
	store := NewStore(freshDB(t))
	ctx := context.Background()
	require.NoError(t, store.Seed(ctx))
	styles, err := store.ListStyles(ctx, "", "", "", "", "")
	require.NoError(t, err)
	require.NotEmpty(t, styles)

	seen := map[string]string{}
	for _, style := range styles {
		key := pictureKey(t, style)
		if prior, dup := seen[key]; dup {
			t.Errorf("styles %q and %q render the same picture: same source, same treatment chain, same parameters.\n"+
				"Two styles that render identically are one style with two names — change the generator, the chain, or the parameters, or drop one.\n"+
				"  key: %s", prior, style.ID, key)
			continue
		}
		seen[key] = style.ID
	}
}

// TestEveryGeneratorIsReachedByAStyle is the other half. A generator no style
// names is code that ships, is tested, and is invisible to every operator —
// which is the same failure as a declared taxonomy value with nothing behind
// it, just pointed the other way.
func TestEveryGeneratorIsReachedByAStyle(t *testing.T) {
	store := NewStore(freshDB(t))
	ctx := context.Background()
	require.NoError(t, store.Seed(ctx))
	styles, err := store.ListStyles(ctx, "", "", "", "", "")
	require.NoError(t, err)

	reached := map[string]bool{}
	for _, style := range styles {
		if style.Strategy != "procedural" && style.Strategy != "procedural-treated" {
			continue
		}
		preset, err := scenes.ResolvePreset(style.Subject, declaredPreset(style))
		require.NoError(t, err)
		reached[preset] = true
	}
	for _, preset := range scenes.Presets {
		require.Truef(t, reached[preset],
			"generator %q is reached by no seeded style: it is shipped, tested, and invisible. Author a style that uses it or remove it.", preset)
	}
}

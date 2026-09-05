package motion

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func layers() []Layer {
	return []Layer{
		{Name: "sky", Depth: 0, Blend: "normal", File: "sky.png", Opacity: 1, Motion: Profile{Parallax: 0.05}},
		{
			Name: "headlands", Depth: 1, Blend: "multiply", File: "headlands.png", Opacity: 0.9,
			Motion: Profile{Parallax: 0.18, Ambient: AmbientDrift, AmbientSeconds: 24, AmbientAmplitude: 0.01},
		},
		{Name: "bank", Depth: 2, Blend: "normal", File: "bank.png", Opacity: 1, Motion: Profile{Parallax: 0.42}},
	}
}

// Every plate in the manifest gets exactly one layer rule. A plate with no rule
// is a file a consumer downloads and never draws; a plate with two is a rule
// whose winner depends on source order.
func TestTheStylesheetReferencesEveryPlateExactlyOnce(t *testing.T) {
	manifest, css, err := Describe("engraved-colonnade-vector", 1440, 720, "composite.png", layers())
	require.NoError(t, err)
	require.Len(t, manifest.Layers, 3)
	require.NotEmpty(t, css)

	for _, layer := range manifest.Layers {
		selector := ".engraved-colonnade-vector__layer--" + layer.Name
		require.Equalf(t, 2, strings.Count(css, selector+" {"),
			"plate %q needs exactly one layer rule and one motion rule; found %d occurrences of %q",
			layer.Name, strings.Count(css, selector+" {"), selector)
		require.Containsf(t, css, `url("`+layer.File+`")`, "plate %q's file is never referenced", layer.Name)
	}
	// And no rule references a file the manifest does not list.
	for _, file := range []string{"sky.png", "headlands.png", "bank.png", "composite.png"} {
		require.Contains(t, css, file)
	}
	require.NotContains(t, css, "url(\"\")", "an empty file reference would download nothing and draw nothing")
}

// Reduced motion is the DEFAULT, not an override. A consumer who forgets the
// media query still gets a still picture rather than an unwanted one.
func TestReducedMotionResolvesToTheCompositeAndNeverToAPlateSet(t *testing.T) {
	manifest, css, err := Describe("layered", 1440, 720, "composite.png", layers())
	require.NoError(t, err)

	require.Equal(t, "composite.png", manifest.ReducedMotion)
	require.Equal(t, manifest.Composite, manifest.ReducedMotion,
		"the reduced-motion target is the flat composite; naming it separately is for the contract, not for a different file")

	// Every transform and animation sits inside the no-preference block.
	block := strings.Index(css, "@media (prefers-reduced-motion: no-preference)")
	require.Positive(t, block, "the motion block must exist and must be media-scoped")
	before, inside := css[:block], css[block:]
	require.NotContains(t, before, "transform:", "a transform outside the media query moves a reduced-motion viewer's picture")
	require.NotContains(t, before, "animation:", "an animation outside the media query runs for a reduced-motion viewer")
	require.Contains(t, inside, "transform:")
	require.Contains(t, inside, "animation:")

	// The composite is painted by the container itself, so it is what remains
	// when every layer above it is inert.
	require.Contains(t, css, `background-image: url("composite.png")`)
}

// A stack whose plates all move together is a flat image. Emitting a manifest
// for it would tell a consumer there is parallax to render when there is not.
func TestAStackWithNoDepthEmitsNoMotionDescriptor(t *testing.T) {
	flat := layers()
	for i := range flat {
		flat[i].Motion.Parallax = 0.2
	}
	manifest, css, err := Describe("flat", 1440, 720, "composite.png", flat)
	require.Error(t, err)
	var stackErr *FlatStackError
	require.ErrorAs(t, err, &stackErr)
	require.Equal(t, 3, stackErr.Layers)
	require.Empty(t, css, "no stylesheet for a stack with no depth")

	// The manifest still comes back: a consumer needs to know what files exist,
	// and the refusal is about motion rather than about delivery.
	require.Equal(t, "composite.png", manifest.Composite)
	require.Equal(t, "composite.png", manifest.ReducedMotion)
	require.Empty(t, manifest.Layers)
	require.Empty(t, manifest.CSS)
}

// A single-plate candidate is a picture, not a degenerate stack.
func TestASinglePlateCandidateGetsAManifestAndNoMotion(t *testing.T) {
	manifest, css, err := Describe("flat", 1440, 720, "composite.png", layers()[:1])
	require.NoError(t, err, "one plate is not an error; it is the common case")
	require.Empty(t, css)
	require.Empty(t, manifest.Layers)
	require.Equal(t, "composite.png", manifest.ReducedMotion)
}

// A keyframe's transform REPLACES the rule's rather than composing with it, so
// an animation that omitted the parallax translate would snap its layer back to
// zero the moment the loop started.
func TestAnAmbientKeyframeRestatesItsParallax(t *testing.T) {
	_, css, err := Describe("layered", 1440, 720, "composite.png", layers())
	require.NoError(t, err)

	start := strings.Index(css, "@keyframes")
	require.Positive(t, start, "the drifting plate must emit keyframes")
	frames := css[start:]
	require.Contains(t, frames, "from {")
	require.Contains(t, frames, "to   {")
	require.Equal(t, 2, strings.Count(frames, "var(--scroll)"),
		"both endpoints must restate the parallax translate or the layer jumps when its loop begins")
	require.Contains(t, frames, "translateX(", "drift moves along an axis")
}

// Only the plate that declares an ambient loop gets one.
func TestOnlyADeclaredAmbientLoopIsEmitted(t *testing.T) {
	_, css, err := Describe("layered", 1440, 720, "composite.png", layers())
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(css, "@keyframes"), "exactly one plate declares an ambient loop")
	require.Contains(t, css, "@keyframes layered-headlands")
	require.NotContains(t, css, "@keyframes layered-sky")
}

// Blend maps to CSS only where it changes something. `normal` is the initial
// value, and emitting it adds a line that implies a decision nobody made.
func TestOnlyANonNormalBlendReachesTheStylesheet(t *testing.T) {
	_, css, err := Describe("layered", 1440, 720, "composite.png", layers())
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(css, "mix-blend-mode"), "one plate blends multiply; the other two are normal")
	require.Contains(t, css, "mix-blend-mode: multiply")
}

// Rules are scoped to the style, so a page carrying two backdrops does not have
// one stylesheet silently reposition the other's layers.
func TestRulesAreScopedToTheStyle(t *testing.T) {
	_, first, err := Describe("engraved-colonnade-vector", 1440, 720, "a.png", layers())
	require.NoError(t, err)
	_, second, err := Describe("tidal-halftone", 1440, 720, "b.png", layers())
	require.NoError(t, err)

	require.Contains(t, first, ".engraved-colonnade-vector__layer--sky")
	require.NotContains(t, first, ".tidal-halftone")
	require.Contains(t, second, ".tidal-halftone__layer--sky")
	require.NotContains(t, second, ".engraved-colonnade-vector")
}

// No video, no GIF, no runtime library. This is the boundary the plan records
// as a non-goal, asserted rather than trusted.
func TestTheDescriptorIsCSSAndNothingElse(t *testing.T) {
	_, css, err := Describe("layered", 1440, 720, "composite.png", layers())
	require.NoError(t, err)
	for _, forbidden := range []string{"<video", ".mp4", ".webm", ".gif", "<script", "requestAnimationFrame"} {
		require.NotContainsf(t, css, forbidden,
			"the motion descriptor must be CSS over plate images; %q means it became something else", forbidden)
	}
}

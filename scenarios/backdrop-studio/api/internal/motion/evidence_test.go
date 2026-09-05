package motion

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// The delivery-set example.
//
// It is a test rather than a hand-run probe for the reason EVIDENCE.md states:
// an artifact nobody can reproduce is a claim about a build nobody can
// identify. What it shows is the exact manifest and stylesheet a consumer
// receives, so the delivery contract can be read against something real rather
// than against prose describing it.
//
// The plate images are not written here — this package produces no pixels, and
// the stacks themselves are shown in docs/evidence/plates/.
func TestDeliverySetExampleEvidence(t *testing.T) {
	manifest, css, err := Describe("engraved-colonnade-vector", 1440, 720, "composite.png", []Layer{
		{
			Name: "distance", Depth: 0, Blend: "normal", File: "distance.png", Opacity: 1,
			Motion: Profile{Parallax: 0.04},
		},
		{
			Name: "arcade", Depth: 1, Blend: "normal", File: "arcade.png", Opacity: 1,
			Motion: Profile{Parallax: 0.22},
		},
		{
			Name: "canopy", Depth: 2, Blend: "normal", File: "canopy.png", Opacity: 1,
			Motion: Profile{Parallax: 0.46, Ambient: AmbientSway, AmbientSeconds: 26, AmbientAmplitude: 0.006},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, css)

	encoded, err := json.MarshalIndent(manifest, "", "  ")
	require.NoError(t, err)

	if os.Getenv("BACKDROP_STUDIO_WRITE_EVIDENCE") == "" {
		t.Skip("set BACKDROP_STUDIO_WRITE_EVIDENCE=1 to write docs/evidence/delivery-set/")
	}
	dir := filepath.Join("..", "..", "..", "docs", "evidence", "delivery-set")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), append(encoded, '\n'), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "motion.css"), []byte(css), 0o644))

	readme := `# A delivery set

What a consumer of ` + "`engraved-colonnade-vector`" + ` receives, at ` + "`web.hero`" + ` geometry.

**Reproduce:** ` + "`make integration-evidence`" + ` from ` + "`scenarios/backdrop-studio`" + `.

` + "`manifest.json`" + ` names every file in the set. ` + "`motion.css`" + ` is the
transform descriptor over the plates. The plate images and the composite are not
duplicated here — the stacks themselves are in
[` + "`../plates/`" + `](../plates/) — because this artifact exists to show the
CONTRACT, and two copies of the same PNG would only drift.

The contract itself is written up in
[` + "`../../reference/delivery-contract.md`" + `](../../reference/delivery-contract.md).

Two properties worth reading the CSS for:

- Every transform and keyframe sits inside
  ` + "`@media (prefers-reduced-motion: no-preference)`" + `, so a still picture is
  what a consumer gets by default and motion is what has to be opted into.
- Each ambient keyframe restates its layer's parallax translate. A CSS
  ` + "`transform`" + ` in a keyframe REPLACES the one on the rule rather than
  composing with it, so an animation that omitted it would snap its layer back to
  zero parallax the moment the loop started.
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0o644))
}

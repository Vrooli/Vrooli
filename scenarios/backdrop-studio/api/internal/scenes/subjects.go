package scenes

import (
	"fmt"
	"sort"
)

// Subject coherence.
//
// The procedural lane used to answer every subject by picking the nearest scene
// it happened to have: `aquatic`, `atmospheric` and `horizon` all rendered the
// horizon; `interior` rendered the arcade; `cartographic` rendered the terrain;
// and `textile_material` and `object_metaphor` rendered the abstract field.
// Sixteen named art directions drew four pictures, and nothing anywhere said
// so — Ukiyo Tide, Riso Horizon, City Pop Horizon and Solar Bloom Horizon were
// the same image under different filters.
//
// The rule now: a generator declares what it depicts, and a procedural style
// may only use a generator that depicts its subject. A subject with no
// generator is refused rather than substituted. Refusing is the honest answer —
// it says "this needs the model lane" instead of shipping a different picture
// under the requested name.

// subjectOfPreset is what each generator actually depicts.
//
// `caustics` claims `aquatic` rather than `non_representational`: it simulates
// light refracted through a water surface, which is a depiction of water, and
// pretending otherwise would repeat the mistake this file exists to correct.
var subjectOfPreset = map[string]string{
	"horizon":   "horizon",
	"arcade":    "statuary_architecture",
	"terrain":   "geological",
	"field":     "non_representational",
	"flow":      "non_representational",
	"voronoi":   "non_representational",
	"reaction":  "non_representational",
	"caustics":  "aquatic",
	"mesh":      "non_representational",
	"truchet":   "non_representational",
	"attractor": "non_representational",
	// `contour` draws level sets of a height field, which is a map. Filing it
	// under the abstract subject would be the same substitution this file
	// exists to prevent — and it would leave `cartographic` with no generator
	// while a generator that draws exactly that sat one line above.
	"contour": "cartographic",
	// `nebula` depicts a sky. It is the only representational subject the
	// procedural lane gained here, and it earns the claim: the emission field,
	// the dust occlusion and the star magnitudes are all modelled, not implied.
	"nebula": "atmospheric",
}

// defaultPresetForSubject is the generator a style gets when it names a subject
// but not a generator. Where several generators share a subject the default is
// the one whose look is least specific, so a style that did not choose has not
// been given a strong opinion by accident.
var defaultPresetForSubject = map[string]string{
	"non_representational":  "field",
	"horizon":               "horizon",
	"statuary_architecture": "arcade",
	"geological":            "terrain",
	"aquatic":               "caustics",
	"cartographic":          "contour",
	"atmospheric":           "nebula",
}

// SubjectOf reports what a generator depicts.
func SubjectOf(preset string) (string, bool) {
	subject, ok := subjectOfPreset[preset]
	return subject, ok
}

// ProceduralSubjects lists every subject the procedural lane can actually draw,
// in stable order. Everything else needs a model.
func ProceduralSubjects() []string {
	seen := map[string]bool{}
	for _, subject := range subjectOfPreset {
		seen[subject] = true
	}
	out := make([]string, 0, len(seen))
	for subject := range seen {
		out = append(out, subject)
	}
	sort.Strings(out)
	return out
}

// PresetsForSubject lists the generators that depict a subject, in stable
// order, so a caller can offer the choice rather than guess.
func PresetsForSubject(subject string) []string {
	out := make([]string, 0, 4)
	for preset, s := range subjectOfPreset {
		if s == subject {
			out = append(out, preset)
		}
	}
	sort.Strings(out)
	return out
}

// ResolvePreset picks the generator for one procedural render.
//
// A style may name a generator explicitly; if it does, that generator must
// depict the style's subject, because a style claiming `geological` while
// rendering `caustics` is the exact substitution this is here to stop. A style
// that names none gets its subject's default. A subject with no generator at
// all is an error naming the subject and what the lane can draw — never a
// silent fallback.
func ResolvePreset(subject, declared string) (string, error) {
	if declared != "" {
		depicts, known := subjectOfPreset[declared]
		if !known {
			return "", fmt.Errorf("scenes: unknown generator %q (known: %v)", declared, Presets)
		}
		if depicts != subject {
			return "", fmt.Errorf("scenes: generator %q depicts %q, but the style declares subject %q; use %v or change the subject",
				declared, depicts, subject, PresetsForSubject(subject))
		}
		return declared, nil
	}
	if preset, ok := defaultPresetForSubject[subject]; ok {
		return preset, nil
	}
	return "", fmt.Errorf("scenes: no procedural generator depicts subject %q; the procedural lane draws %v, and everything else needs a model-backed strategy",
		subject, ProceduralSubjects())
}

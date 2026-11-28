package vector

import (
	"fmt"
	"sort"
	"strings"
)

// Subject coherence for the vector family.
//
// The rule is the raster lane's, and it is the same rule for the same reason: a
// generator declares what it depicts, and a style may only use a generator that
// depicts its subject. A subject no generator draws is refused rather than
// answered with the nearest available picture, because silently substituting is
// how sixteen named art directions came to draw four.
//
// Keeping a separate table here rather than extending the raster one is
// deliberate. The two families do not draw the same subjects — nothing in the
// raster lane draws `object_metaphor`, and nothing here draws a nebula — and a
// merged table would let a style resolve to a generator in the other family,
// which is exactly the substitution both tables exist to prevent.
var subjectOfPreset = map[string]string{
	"colonnade":        "statuary_architecture",
	"contour-relief":   "cartographic",
	"halftone-horizon": "horizon",
	// `radiant-orb` draws a disc over a horizon line with a figure for scale.
	// It claims `celestial` because that is what it depicts; filing it under
	// the abstract subject would be a substitution, and `celestial` had no
	// procedural generator at all before it.
	"radiant-orb": "celestial",
}

// defaultPresetForSubject is the generator a style gets when it names a subject
// and no generator. One entry per subject the family covers, so the mapping can
// never be ambiguous.
var defaultPresetForSubject = map[string]string{
	"statuary_architecture": "colonnade",
	"cartographic":          "contour-relief",
	"horizon":               "halftone-horizon",
	"celestial":             "radiant-orb",
}

// SubjectOf reports what a vector generator depicts.
func SubjectOf(preset string) (string, bool) {
	subject, ok := subjectOfPreset[preset]
	return subject, ok
}

// ResolvePreset picks the vector generator for a style, and refuses rather than
// substituting.
//
// A declared preset wins, and it must depict the style's subject: a style that
// says it depicts a map and names the colonnade generator is a contradiction,
// and shipping the colonnade under the map's name is the defect. With no
// declared preset, the subject's default is used.
func ResolvePreset(subject, declared string) (string, error) {
	declared = strings.TrimSpace(declared)
	if declared != "" {
		depicts, known := subjectOfPreset[declared]
		if !known {
			return "", fmt.Errorf("vector: unknown generator %q (known: %s)", declared, knownPresets())
		}
		if depicts != subject {
			return "", fmt.Errorf(
				"vector: generator %q depicts %q, but the style declares subject %q; "+
					"a generator may only be used for the subject it draws",
				declared, depicts, subject)
		}
		return declared, nil
	}
	preset, ok := defaultPresetForSubject[subject]
	if !ok {
		return "", fmt.Errorf(
			"vector: no vector generator draws subject %q (drawn: %s); "+
				"use the raster or model lane for it rather than substituting a different picture",
			subject, knownSubjects())
	}
	return preset, nil
}

func knownPresets() string {
	out := append([]string(nil), Presets...)
	sort.Strings(out)
	return strings.Join(out, ", ")
}

func knownSubjects() string {
	out := make([]string, 0, len(defaultPresetForSubject))
	for subject := range defaultPresetForSubject {
		out = append(out, subject)
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

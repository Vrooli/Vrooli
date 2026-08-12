package catalog

// Quality is the perceptual bar one style's output must clear. It lives on the
// style record because the bar is art direction, not a global constant: a
// deliberately extreme style states its own numbers and says so in the catalog,
// rather than forcing the gate to be loose enough for the worst case.
//
// A zero field means "use the family default for this style's treatments".
// Setting a field to a negative value opts that one metric out entirely — which
// a style should only do with a reason recorded beside it, because a style that
// opts out of every metric has no gate at all.
type Quality struct {
	MinSubjectSurvival     float64 `json:"min_subject_survival,omitempty"`
	MinTonalOccupancy      float64 `json:"min_tonal_occupancy,omitempty"`
	MinFrequencyModulation float64 `json:"min_frequency_modulation,omitempty"`
	MaxReservedQuiet       float64 `json:"max_reserved_quiet,omitempty"`
}

// Treatment families. A family is a group of operations that fail the same way,
// which is why the thresholds are grouped by it rather than set per operation:
// nine screens that all fail by erasing their subject need one number, not nine.
const (
	// familyScreen rebuilds tone out of discrete marks. It legitimately
	// destroys fine detail, so its survival bar is the lowest — but it is the
	// family that produced the moire failure, so its modulation bar is the one
	// that matters most.
	familyScreen = "screen"
	// familyTonal remaps tone without adding structure. It should preserve the
	// composition almost perfectly, so its survival bar is the highest.
	familyTonal = "tonal"
	// familyOptical simulates a lens or a motion: it blurs and displaces, which
	// costs some structure and much of the high-frequency detail.
	familyOptical = "optical"
)

var treatmentFamilies = map[string]string{
	"halftone":         familyScreen,
	"line_screen":      familyScreen,
	"stipple":          familyScreen,
	"engraving":        familyScreen,
	"ascii_mosaic":     familyScreen,
	"dither_ordered":   familyScreen,
	"dither_diffusion": familyScreen,

	"duotone":   familyTonal,
	"posterize": familyTonal,
	"curve":     familyTonal,
	"grain":     familyTonal,
	"scrim":     familyTonal,

	"bloom":        familyOptical,
	"defocus":      familyOptical,
	"motion_blur":  familyOptical,
	"aberration":   familyOptical,
	"displacement": familyOptical,
	"pixel_sort":   familyOptical,
}

// familyDefaults are the documented per-family bars.
//
// Every number is set below the worst value measured across the seeded catalog
// with headroom, not chosen a priori — the measurements are recorded in
// docs/evidence/perceptual/corpus.json and the derivation is in
// docs/reference/configuration.md. A gate calibrated by taste alone either
// rejects working styles or catches nothing.
var familyDefaults = map[string]Quality{
	familyScreen: {
		MinSubjectSurvival:     0.35,
		MinTonalOccupancy:      0.30,
		MinFrequencyModulation: 0.04,
		MaxReservedQuiet:       1.60,
	},
	familyTonal: {
		MinSubjectSurvival:     0.70,
		MinTonalOccupancy:      0.25,
		MinFrequencyModulation: 0.04,
		MaxReservedQuiet:       1.60,
	},
	familyOptical: {
		MinSubjectSurvival:     0.55,
		MinTonalOccupancy:      0.25,
		MinFrequencyModulation: 0.04,
		MaxReservedQuiet:       1.60,
	},
}

// EffectiveQuality resolves the bar for one style: the family default for its
// treatments, with any value the style states itself taking precedence.
//
// When a chain spans families the strictest bar wins on every metric. A chain
// is judged on its output, and its output has to be good enough for the most
// demanding thing in it — a duotone followed by a bloom is still expected to
// read as the picture the duotone made.
func (v Style) EffectiveQuality() Quality {
	base, seen := Quality{}, false
	for _, treatment := range v.Treatments {
		family, known := treatmentFamilies[treatment]
		if !known {
			continue
		}
		bar := familyDefaults[family]
		if !seen {
			base, seen = bar, true
			continue
		}
		base = strictest(base, bar)
	}
	if !seen {
		// An unrecognised chain gets the screen family's bar: the loosest of
		// the three on structure, so a new treatment cannot be rejected merely
		// for being unclassified, while still having to prove it drew
		// something. A treatment reaching here belongs in treatmentFamilies.
		base = familyDefaults[familyScreen]
	}
	if v.Quality == nil {
		return base
	}
	return overlay(base, *v.Quality)
}

func strictest(a, b Quality) Quality {
	return Quality{
		MinSubjectSurvival:     maxFloat(a.MinSubjectSurvival, b.MinSubjectSurvival),
		MinTonalOccupancy:      maxFloat(a.MinTonalOccupancy, b.MinTonalOccupancy),
		MinFrequencyModulation: maxFloat(a.MinFrequencyModulation, b.MinFrequencyModulation),
		MaxReservedQuiet:       minFloat(a.MaxReservedQuiet, b.MaxReservedQuiet),
	}
}

// overlay applies a style's own declarations over the family default. A
// negative declaration disables that metric, which is what lets a style opt out
// of one bar without opting out of all of them.
func overlay(base, own Quality) Quality {
	set := func(dst, src float64) float64 {
		switch {
		case src < 0:
			return 0
		case src > 0:
			return src
		default:
			return dst
		}
	}
	return Quality{
		MinSubjectSurvival:     set(base.MinSubjectSurvival, own.MinSubjectSurvival),
		MinTonalOccupancy:      set(base.MinTonalOccupancy, own.MinTonalOccupancy),
		MinFrequencyModulation: set(base.MinFrequencyModulation, own.MinFrequencyModulation),
		MaxReservedQuiet:       set(base.MaxReservedQuiet, own.MaxReservedQuiet),
	}
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

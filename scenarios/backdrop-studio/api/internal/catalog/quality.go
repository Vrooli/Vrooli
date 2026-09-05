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
// Every number is derived from measurement, not chosen a priori: the whole
// seeded catalog was rendered and scored, and each bar sits below the worst
// observed value with headroom. A gate calibrated by taste alone either rejects
// working styles or catches nothing.
//
// Observed floors across the fourteen renderable seeded styles (seed 7, the
// full run is in docs/evidence/perceptual/corpus.json):
//
//	subject_survival     0.8585  (ascii-field)
//	tonal_occupancy      0.6142  (cyanotype-arcade)
//	frequency_modulation 0.0517  (ascii-field)
//	reserved_quiet       1.2735  (riso-horizon, a ceiling not a floor)
//
// The headroom is deliberately wide. These metrics separate *working* from
// *destroyed* by a wide margin — a treatment that erases its subject scores
// below 0.2 on survival and near 0 on modulation against the synthetic cases in
// internal/perceptual — so a bar placed just under the observed floor would
// reject art direction rather than catch defects. The bar's job is to make
// "unusable" impossible to ship, not to enforce a house style.
var familyDefaults = map[string]Quality{
	familyScreen: {
		// A screen rebuilds tone out of discrete marks and legitimately
		// discards fine detail, so it gets the loosest structural bar.
		MinSubjectSurvival:     0.60,
		MinTonalOccupancy:      0.40,
		MinFrequencyModulation: 0.030,
		MaxReservedQuiet:       1.60,
	},
	familyTonal: {
		// A tonal remap adds no structure of its own, so it should return the
		// composition nearly intact. Every tonal-family style measured scores
		// above 0.99; anything under 0.80 means something went wrong.
		MinSubjectSurvival:     0.80,
		MinTonalOccupancy:      0.40,
		MinFrequencyModulation: 0.030,
		MaxReservedQuiet:       1.60,
	},
	familyOptical: {
		// A lens or motion simulation blurs and displaces, which costs some
		// structure — but the measured optical styles still clear 0.96.
		MinSubjectSurvival:     0.70,
		MinTonalOccupancy:      0.35,
		MinFrequencyModulation: 0.030,
		MaxReservedQuiet:       1.60,
	},
}

// EffectiveQuality resolves the bar for one style: the family default for its
// treatments, with any value the style states itself taking precedence.
//
// When a chain spans families, the two kinds of metric combine differently, and
// the difference is the whole point of having families at all.
//
// **Structural licence takes the loosest bar.** `subject_survival` asks how much
// of the composition survived. A chain is entitled to the licence of its most
// destructive member: `posterize` then `halftone` ends in a screen, and a screen
// legitimately discards fine detail, so holding the chain to the tonal family's
// high bar because a posterize appeared earlier judges it for something it never
// claimed to be. The first version of this took the strictest bar on every
// metric and refused `ukiyo-tide` at 0.772 against a 0.800 floor — an image that
// reads correctly, rejected by arithmetic rather than by a fault.
//
// **Usability takes the strictest bar.** Occupancy, modulation and
// reserved-region quiet are not about licence; they are about whether the result
// is usable at all. Every member of a chain has to leave a usable image behind
// it, so the most demanding requirement governs.
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
		base = combine(base, bar)
	}
	if !seen {
		// Two cases land here and they mean different things.
		//
		// An *empty* chain is a `procedural` style shipping the scene as drawn.
		// Its output is its input, so subject survival is 1 by construction and
		// the metric proves nothing; what still matters is that the generator
		// produced a usable picture, which the other three measure. The
		// structural bar is dropped rather than set low, because a bar that
		// cannot fail is worse than no bar: it reads as coverage.
		//
		// An *unrecognised* chain gets the screen family's numbers — the
		// loosest of the three on structure — so a new treatment is not
		// rejected merely for being unclassified while still having to prove it
		// drew something. A treatment reaching that case belongs in
		// treatmentFamilies.
		base = familyDefaults[familyScreen]
		if len(v.Treatments) == 0 {
			base.MinSubjectSurvival = 0
		}
	}
	if v.Quality == nil {
		return base
	}
	return overlay(base, *v.Quality)
}

// combine merges two family bars: loosest on structural licence, strictest on
// everything that decides whether the result is usable. See EffectiveQuality.
func combine(a, b Quality) Quality {
	return Quality{
		MinSubjectSurvival:     minFloat(a.MinSubjectSurvival, b.MinSubjectSurvival),
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

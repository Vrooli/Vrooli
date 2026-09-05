package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/png"
	"math"
	"sort"
	"strings"

	"backdrop-studio/internal/scenes"
)

// Perceived family resemblance, as distinct from identity.
//
// `TestNoTwoSettledStylesRenderTheSamePicture` compares declared configuration
// and reports two styles as one only when the generator, the chain and every
// parameter agree exactly. That test passes today while three mesh styles read
// as one picture at a glance, because changing a single parameter is enough to
// make two configurations distinct and nowhere near enough to make two pictures
// distinct. Identity is a property of the declaration; resemblance is a property
// of the render, and it needs its own measure.
//
// Two decisions shape what is measured here.
//
// Structure is measured and colour is not. The operator's repair direction for
// a resemblance cluster is "give each member a different source, not a different
// colour ramp" — so a measure that counted the ramp would report a cluster as
// diverged the moment someone recoloured it, which is the exact evasion the
// direction forbids. The structural descriptor is contrast-normalised for the
// same reason: it answers "is this the same arrangement of masses", not "is this
// the same brightness".
//
// The chain counts, because it genuinely changes the picture. `cyanotype-arcade`
// and `engraved-colonnade` draw the same arcade and are not the same style: one
// is a dot screen and the other is a burin. A measure that looked only at the
// source would report them as a cluster and send a later phase to diverge two
// styles that are already different.

// Resemblance dimensions.
//
// The grid is deliberately coarse. It is a descriptor of composition — where the
// masses sit — and a fine grid would start measuring generator noise, which two
// seeds of one generator differ in and two art directions do not.
const (
	resembleWidth   = 240
	resembleHeight  = 150
	resembleCols    = 16
	resembleRows    = 10
	resembleCells   = resembleCols * resembleRows
	resembleSeed    = 7 // the seed every evidence artifact in this scenario uses
	sourceKindScene = "scene"
	sourceKindPromp = "prompt"
)

// FamilyResemblanceThreshold is the resemblance at or above which two styles
// read as one picture.
//
// It is set from measurement rather than taste: see
// docs/evidence/catalog/resemblance.md for the nearest-neighbour distribution
// the value was read off, and docs/evidence/catalog/verdicts.md for the styles
// a reader independently judged to be the same picture. The number sits in the
// gap between the highest-scoring pair a reader called distinct and the
// lowest-scoring pair a reader called a duplicate.
const FamilyResemblanceThreshold = 0.90

// Signature is what a style's picture is compared by.
type Signature struct {
	StyleID string
	// Kind is "scene" for a style the procedural lane draws in-process, and
	// "prompt" for a model-backed style whose picture this process cannot
	// produce. The two are never compared: a luminance grid and a bag of words
	// have no common scale, and inventing one would produce a number that looks
	// like a measurement and is not.
	Kind string
	// Structure is a contrast-normalised luminance grid. Scene kind only.
	Structure []float64
	// Tokens is the prompt's word set. Prompt kind only.
	Tokens map[string]struct{}
	// Chain and ChainParams are the treatment chain and its non-colour
	// parameters, which decide how much of the source survives to be seen.
	Chain       []string
	ChainParams map[string]map[string]float64
}

// Pair is one style's nearest neighbour and the resemblance measured to it.
type Pair struct {
	StyleID   string  `json:"style_id"`
	NearestID string  `json:"nearest_id"`
	Kind      string  `json:"kind"`
	Source    float64 `json:"source_similarity"`
	ChainSim  float64 `json:"chain_similarity"`
	// Resemblance is the product of the two. 1.0 means "the same picture":
	// the same arrangement of masses, seen through the same treatment.
	Resemblance float64 `json:"resemblance"`
}

// Report is the whole catalog's nearest-neighbour table, ordered by descending
// resemblance so the clusters are the first thing a reader sees.
type Report struct {
	Pairs []Pair `json:"pairs"`
	// Unmeasured names styles whose picture this process could not produce, so
	// a reader can tell "no cluster found" from "not looked at".
	Unmeasured []string `json:"unmeasured,omitempty"`
}

// Cluster is a set of styles whose mutual resemblance is at or above the
// threshold. It is what Phase 12's divergence work consumes.
type Cluster struct {
	StyleIDs    []string `json:"style_ids"`
	Resemblance float64  `json:"resemblance"`
}

// Signature computes one style's picture descriptor.
func ComputeSignature(v Style) (Signature, error) {
	sig := Signature{StyleID: v.ID, Chain: append([]string(nil), v.Treatments...), ChainParams: numericParams(v.TreatmentParams)}
	switch v.Strategy {
	case "procedural", "procedural-treated":
		preset, err := scenes.ResolvePreset(v.Subject, declaredScenePreset(v))
		if err != nil {
			return Signature{}, fmt.Errorf("catalog: resemblance: style %q: %w", v.ID, err)
		}
		res, err := scenes.Render(scenes.Request{
			Preset:     preset,
			ParamsJSON: scenePresetParams(v),
			Width:      resembleWidth,
			Height:     resembleHeight,
			Seed:       resembleSeed,
		})
		if err != nil {
			return Signature{}, fmt.Errorf("catalog: resemblance: render %q: %w", v.ID, err)
		}
		grid, err := structureGrid(res.PNG)
		if err != nil {
			return Signature{}, fmt.Errorf("catalog: resemblance: %q: %w", v.ID, err)
		}
		sig.Kind, sig.Structure = sourceKindScene, grid
	default:
		prompt := ""
		if v.Generation != nil {
			prompt = v.Generation.PromptTemplate
		}
		sig.Kind, sig.Tokens = sourceKindPromp, promptTokens(prompt)
	}
	return sig, nil
}

func declaredScenePreset(v Style) string {
	if v.Scaffold == nil {
		return ""
	}
	return v.Scaffold.Preset
}

func scenePresetParams(v Style) string {
	if v.Scaffold == nil {
		return ""
	}
	return v.Scaffold.ParamsJSON
}

// structureGrid reduces a scene to a contrast-normalised luminance grid.
//
// Normalising to zero mean and unit deviation is what makes this a measure of
// arrangement rather than of exposure: a style that darkened its ramp has not
// become a different picture, and this must not report that it has.
func structureGrid(encoded []byte) ([]float64, error) {
	img, err := png.Decode(bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("decode scene: %w", err)
	}
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return nil, fmt.Errorf("scene has empty bounds")
	}
	sums := make([]float64, resembleCells)
	counts := make([]float64, resembleCells)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		row := (y - b.Min.Y) * resembleRows / b.Dy()
		if row >= resembleRows {
			row = resembleRows - 1
		}
		for x := b.Min.X; x < b.Max.X; x++ {
			col := (x - b.Min.X) * resembleCols / b.Dx()
			if col >= resembleCols {
				col = resembleCols - 1
			}
			idx := row*resembleCols + col
			r, g, b16, _ := img.At(x, y).RGBA()
			// Rec.709 luma over the 8-bit values. Perceptual exactness is not
			// what this needs — it needs one consistent reduction of colour to
			// tone so two arrangements can be correlated.
			sums[idx] += (0.2126*float64(r>>8) + 0.7152*float64(g>>8) + 0.0722*float64(b16>>8)) / 255
			counts[idx]++
		}
	}
	grid := make([]float64, resembleCells)
	for i := range grid {
		if counts[i] > 0 {
			grid[i] = sums[i] / counts[i]
		}
	}
	return normalise(grid), nil
}

func normalise(grid []float64) []float64 {
	mean := 0.0
	for _, v := range grid {
		mean += v
	}
	mean /= float64(len(grid))
	variance := 0.0
	for _, v := range grid {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(grid))
	deviation := math.Sqrt(variance)
	out := make([]float64, len(grid))
	if deviation < 1e-9 {
		// A flat field has no structure to compare. Every cell reads zero,
		// which correlates with nothing and reports resemblance 0 against a
		// structured neighbour — correct, and visible as such in the report.
		return out
	}
	for i, v := range grid {
		out[i] = (v - mean) / deviation
	}
	return out
}

// promptTokens reduces a prompt to its meaningful word set. Stop words are
// dropped because two prompts sharing "a" and "with" share nothing.
func promptTokens(prompt string) map[string]struct{} {
	stop := map[string]bool{
		"a": true, "an": true, "the": true, "of": true, "in": true, "on": true,
		"with": true, "and": true, "at": true, "to": true, "for": true, "by": true,
	}
	out := map[string]struct{}{}
	for _, raw := range strings.FieldsFunc(strings.ToLower(prompt), func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	}) {
		if len(raw) < 2 || stop[raw] {
			continue
		}
		out[raw] = struct{}{}
	}
	return out
}

// colourFields are the parameter names that carry ink rather than geometry.
// They are excluded from the chain comparison because a recolour is explicitly
// not a divergence.
var colourFields = map[string]bool{
	"dark": true, "light": true, "color": true, "colour": true, "tint": true,
}

func numericParams(params map[string]string) map[string]map[string]float64 {
	out := map[string]map[string]float64{}
	for op, raw := range params {
		var fields map[string]any
		if json.Unmarshal([]byte(raw), &fields) != nil {
			continue
		}
		numeric := map[string]float64{}
		for field, value := range fields {
			if colourFields[field] {
				continue
			}
			if n, ok := value.(float64); ok {
				numeric[field] = n
			}
		}
		if len(numeric) > 0 {
			out[op] = numeric
		}
	}
	return out
}

// SourceSimilarity measures how alike two styles' sources are, in [0,1].
func SourceSimilarity(a, b Signature) float64 {
	if a.Kind != b.Kind {
		return 0
	}
	if a.Kind == sourceKindScene {
		return math.Max(0, pearson(a.Structure, b.Structure))
	}
	return jaccard(a.Tokens, b.Tokens)
}

// ChainSimilarity measures how alike two treatment chains are, in [0,1].
func ChainSimilarity(a, b Signature) float64 {
	ops := jaccardStrings(a.Chain, b.Chain)
	shared := 0
	agreement := 0.0
	for op, aFields := range a.ChainParams {
		bFields, ok := b.ChainParams[op]
		if !ok {
			continue
		}
		for field, aValue := range aFields {
			bValue, ok := bFields[field]
			if !ok {
				continue
			}
			scale := math.Max(math.Abs(aValue), math.Abs(bValue))
			if scale < 1e-9 {
				agreement++
				shared++
				continue
			}
			agreement += math.Max(0, 1-math.Abs(aValue-bValue)/scale)
			shared++
		}
	}
	if shared == 0 {
		return ops
	}
	return ops * (agreement / float64(shared))
}

// Resemble is the whole measure: two styles read as one picture when they share
// both an arrangement and a treatment.
func Resemble(a, b Signature) float64 { return SourceSimilarity(a, b) * ChainSimilarity(a, b) }

// BuildReport measures every style against every other and returns each one's
// nearest neighbour.
func BuildReport(styles []Style) (Report, error) {
	signatures := make([]Signature, 0, len(styles))
	report := Report{}
	for _, style := range styles {
		sig, err := ComputeSignature(style)
		if err != nil {
			return Report{}, err
		}
		signatures = append(signatures, sig)
	}
	for i, sig := range signatures {
		best, bestScore := "", -1.0
		var bestSource, bestChain float64
		for j, other := range signatures {
			if i == j || sig.Kind != other.Kind {
				continue
			}
			source := SourceSimilarity(sig, other)
			chain := ChainSimilarity(sig, other)
			score := source * chain
			// Tie-break on the source. A style whose chain shares no operation
			// with anything scores zero against the whole catalog, and reporting
			// whichever style happened to be compared first as its "nearest"
			// would be a fact about iteration order rather than about pictures.
			if score > bestScore || (score == bestScore && source > bestSource) {
				best, bestScore, bestSource, bestChain = other.StyleID, score, source, chain
			}
		}
		if best == "" {
			report.Unmeasured = append(report.Unmeasured, sig.StyleID)
			continue
		}
		report.Pairs = append(report.Pairs, Pair{
			StyleID: sig.StyleID, NearestID: best, Kind: sig.Kind,
			Source: bestSource, ChainSim: bestChain, Resemblance: bestScore,
		})
	}
	sort.SliceStable(report.Pairs, func(i, j int) bool {
		if report.Pairs[i].Resemblance != report.Pairs[j].Resemblance {
			return report.Pairs[i].Resemblance > report.Pairs[j].Resemblance
		}
		return report.Pairs[i].StyleID < report.Pairs[j].StyleID
	})
	return report, nil
}

// Clusters groups styles whose mutual resemblance is at or above threshold.
//
// Grouping is transitive by construction: if A reads as B and B reads as C, a
// reader looking at the catalog sees one family of three, not two pairs. That is
// the shape the divergence work needs.
func Clusters(styles []Style, threshold float64) ([]Cluster, error) {
	signatures := make([]Signature, 0, len(styles))
	for _, style := range styles {
		sig, err := ComputeSignature(style)
		if err != nil {
			return nil, err
		}
		signatures = append(signatures, sig)
	}
	parent := make([]int, len(signatures))
	for i := range parent {
		parent[i] = i
	}
	// Path-halving find. Written as a plain loop rather than a recursive
	// closure because the recursion needed a forward declaration that reads as
	// a mistake and lints as one.
	find := func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	strongest := map[int]float64{}
	for i := range signatures {
		for j := i + 1; j < len(signatures); j++ {
			score := Resemble(signatures[i], signatures[j])
			if score < threshold {
				continue
			}
			ri, rj := find(i), find(j)
			if ri != rj {
				parent[ri] = rj
			}
			root := find(i)
			if score > strongest[root] {
				strongest[root] = score
			}
		}
	}
	grouped := map[int][]string{}
	for i, sig := range signatures {
		root := find(i)
		grouped[root] = append(grouped[root], sig.StyleID)
	}
	var out []Cluster
	for root, ids := range grouped {
		if len(ids) < 2 {
			continue
		}
		sort.Strings(ids)
		out = append(out, Cluster{StyleIDs: ids, Resemblance: strongest[root]})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Resemblance != out[j].Resemblance {
			return out[i].Resemblance > out[j].Resemblance
		}
		return out[i].StyleIDs[0] < out[j].StyleIDs[0]
	})
	return out, nil
}

func pearson(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var sum, aa, bb float64
	for i := range a {
		sum += a[i] * b[i]
		aa += a[i] * a[i]
		bb += b[i] * b[i]
	}
	if aa < 1e-12 || bb < 1e-12 {
		return 0
	}
	return sum / math.Sqrt(aa*bb)
}

func jaccard(a, b map[string]struct{}) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1
	}
	shared := 0
	for k := range a {
		if _, ok := b[k]; ok {
			shared++
		}
	}
	union := len(a) + len(b) - shared
	if union == 0 {
		return 1
	}
	return float64(shared) / float64(union)
}

func jaccardStrings(a, b []string) float64 {
	setA, setB := map[string]struct{}{}, map[string]struct{}{}
	for _, v := range a {
		setA[v] = struct{}{}
	}
	for _, v := range b {
		setB[v] = struct{}{}
	}
	return jaccard(setA, setB)
}

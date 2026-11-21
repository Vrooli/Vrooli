// Package textmetrics provides deterministic, dependency-free measurements for
// generated prose. It deliberately has no import path back into prose-studio
// so other scenarios can use the same measurement contract.
package textmetrics

import (
	"bytes"
	"compress/gzip"
	"errors"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// Span identifies a lexicon hit using byte offsets in the original UTF-8 text.
type Span struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Text  string `json:"text"`
	Term  string `json:"term"`
}

// Readability contains the three deterministic grade estimates. A value is
// zero for empty input; callers should use WordCount to distinguish that case.
type Readability struct {
	FleschKincaid float64 `json:"flesch_kincaid"`
	DaleChall     float64 `json:"dale_chall"`
	GunningFog    float64 `json:"gunning_fog"`
}

// Metrics is the complete deterministic per-text measurement vector.
type Metrics struct {
	ByteLength       int                `json:"byte_length"`
	WordCount        int                `json:"word_count"`
	SentenceCount    int                `json:"sentence_count"`
	SentenceLengths  []int              `json:"sentence_lengths"`
	CompressionRatio float64            `json:"compression_ratio"`
	SelfRepetition   float64            `json:"self_repetition"`
	RougeLHomogenize float64            `json:"rouge_l_homogenize"`
	DistinctN        map[string]float64 `json:"distinct_n"`
	Burstiness       float64            `json:"burstiness"`
	Readability      Readability        `json:"readability"`
	TypeTokenRatio   float64            `json:"type_token_ratio"`
	MATTR            float64            `json:"mattr"`
	LexiconFlags     []Span             `json:"lexicon_flags"`
}

// SetMetrics describes diversity for a candidate set. PairwiseSimilarity is
// symmetric, has a zero diagonal, and is retained so selection policies can
// make the same deterministic decision without recomputing it.
type SetMetrics struct {
	CandidateCount     int             `json:"candidate_count"`
	PairwiseSimilarity [][]float64     `json:"pairwise_similarity"`
	MeanSimilarity     float64         `json:"mean_similarity"`
	Diversity          float64         `json:"diversity"`
	Coverage           float64         `json:"coverage"`
	Basis              string          `json:"basis"`
	CellCoverage       map[string]bool `json:"cell_coverage,omitempty"`
	MissedCells        []string        `json:"missed_cells,omitempty"`
}

// SamplingKey is the effective generation identity used by comparability.
// Source is retained because a role default and a profile-derived value are
// not equivalent even when their numeric values happen to match.
//
// The sampling strategy is deliberately NOT part of this key, and neither is
// any parameter the strategy owns. A strategy is the thing an experiment varies
// — the whole reason to measure two candidate sets is usually to learn whether
// one strategy produced a more diverse set than another under otherwise
// identical conditions. Including the strategy made every such comparison
// incomparable by construction, which left the measurement layer unable to
// answer the one question it exists to answer.
//
// Excluding the strategy while keeping its parameters would not have fixed
// that: a tail threshold that only one strategy defines is a proxy for the
// strategy, so a set drawn without one never matches a set drawn with one. The
// strategy and its parameters therefore leave together.
//
// The cost is real and deliberate: this key no longer distinguishes two runs of
// one strategy at different parameter values, so it cannot refuse a comparison
// between two verbalized rounds drawn at different thresholds. Those parameters
// stay fully recorded on the round; they are simply not what decides
// comparability. Refusing that case again needs comparability to be stated
// relative to a declared experiment rather than by flat key equality.
//
// What remains is the set of conditions defined independently of any strategy,
// which must hold still for a diversity number to mean the same thing on both
// sides: how many candidates were drawn, what sampling stance produced them,
// and what output ceiling bounded them.
type SamplingKey struct {
	K                    int    `json:"k"`
	TemperatureStance    string `json:"temperature_stance"`
	MaxOutputTokens      int    `json:"max_output_tokens_effective"`
	MaxOutputTokenSource string `json:"max_output_tokens_source"`
	MeasurementTier      string `json:"measurement_tier"`
}

// Comparable reports whether two measurements were produced under the same
// effective sampling inputs. It refuses comparison rather than producing a
// misleading number.
func Comparable(a, b SamplingKey) error {
	if a != b {
		return errors.New("candidate sets are not comparable: effective sampling key or max-output-token provenance differs")
	}
	return nil
}

// LexiconSpans locates terms with the same deterministic span semantics used
// by Analyze. It is exported so conformance can report anti-patterns and
// preferred vocabulary separately without duplicating byte-offset logic.
func LexiconSpans(text string, lexicon []string) []Span { return lexiconSpans(text, lexicon) }

// ArtifactsPresent returns the distinct declared artifacts that occur in text,
// in declaration order. An artifact is a literal a writer may quote -- a
// command, a path, an identifier, a figure -- so matching is the same
// case-insensitive substring match the lexicon uses rather than a word-boundary
// match: "test-genie runs wait --json" is one artifact containing spaces and
// flags, and a word matcher would never find it.
//
// Distinctness is what makes the count meaningful. Text that repeats one
// command eight times is not eight artifacts concrete, and counting spans
// rather than terms would say that it is.
func ArtifactsPresent(text string, artifacts []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, span := range lexiconSpans(text, artifacts) {
		if !seen[span.Term] {
			seen[span.Term] = true
			out = append(out, span.Term)
		}
	}
	// lexiconSpans sorts by position; declaration order is restored so callers
	// reporting "which artifacts landed" get a stable list across runs.
	ordered := make([]string, 0, len(out))
	for _, artifact := range artifacts {
		if seen[strings.TrimSpace(artifact)] {
			ordered = append(ordered, strings.TrimSpace(artifact))
		}
	}
	return ordered
}

// Analyze computes every deterministic metric unconditionally.
func Analyze(text string, lexicon []string) Metrics {
	spans := wordsWithSpans(text)
	ws := make([]string, len(spans))
	for i, span := range spans {
		ws[i] = strings.ToLower(span.Text)
	}
	sentences := sentenceWordLengths(text)
	m := Metrics{
		ByteLength:       len([]byte(text)),
		WordCount:        len(ws),
		SentenceCount:    len(sentences),
		SentenceLengths:  append([]int(nil), sentences...),
		CompressionRatio: compressionRatio(text),
		DistinctN:        distinctN(ws),
		Burstiness:       variance(sentences),
		Readability:      readability(spans, sentences),
		TypeTokenRatio:   ratio(float64(uniqueStrings(ws)), float64(len(ws))),
		MATTR:            mattr(ws, 50),
		LexiconFlags:     lexiconSpans(text, lexicon),
	}
	m.SelfRepetition = selfRepetition(ws)
	return m
}

// AnalyzeSet computes per-candidate vectors and set-level lexical diversity.
func AnalyzeSet(texts, lexicon []string) ([]Metrics, SetMetrics) {
	items := make([]Metrics, len(texts))
	for i, text := range texts {
		items[i] = Analyze(text, lexicon)
	}
	set := SetMetrics{CandidateCount: len(texts), Basis: "lexical token 1-3 gram cosine/Jaccard blend; no embeddings available"}
	set.PairwiseSimilarity = make([][]float64, len(texts))
	for i := range set.PairwiseSimilarity {
		set.PairwiseSimilarity[i] = make([]float64, len(texts))
	}
	var total float64
	var pairs int
	perCandidateRouge := make([]float64, len(texts))
	perCandidatePairs := make([]int, len(texts))
	for i := range texts {
		for j := i + 1; j < len(texts); j++ {
			sim := lexicalSimilarity(words(texts[i]), words(texts[j]))
			set.PairwiseSimilarity[i][j], set.PairwiseSimilarity[j][i] = sim, sim
			rouge := rougeL(words(texts[i]), words(texts[j]))
			perCandidateRouge[i] += rouge
			perCandidateRouge[j] += rouge
			perCandidatePairs[i]++
			perCandidatePairs[j]++
			total += sim
			pairs++
		}
	}
	for i := range items {
		items[i].RougeLHomogenize = ratio(perCandidateRouge[i], float64(perCandidatePairs[i]))
	}
	set.MeanSimilarity = ratio(total, float64(pairs))
	set.Diversity = 1 - set.MeanSimilarity
	set.Coverage = coverage(set.PairwiseSimilarity)
	return items, set
}

// AnalyzeSetSemantic retains the deterministic per-text metrics and replaces
// only the set distance tier with cosine distance over gateway-provided
// embeddings. No candidate quality ordering is introduced.
func AnalyzeSetSemantic(texts []string, vectors [][]float64) ([]Metrics, SetMetrics, error) {
	if len(texts) != len(vectors) {
		return nil, SetMetrics{}, errors.New("semantic_measurement_vector_count_mismatch")
	}
	items := make([]Metrics, len(texts))
	for i, text := range texts {
		items[i] = Analyze(text, nil)
	}
	dimension := 0
	if len(vectors) > 0 {
		dimension = len(vectors[0])
	}
	set := SetMetrics{CandidateCount: len(texts), Basis: "semantic cosine similarity over gateway embeddings (dimension " + itoa(dimension) + ")"}
	set.PairwiseSimilarity = make([][]float64, len(texts))
	for i := range set.PairwiseSimilarity {
		set.PairwiseSimilarity[i] = make([]float64, len(texts))
	}
	var total float64
	pairs := 0
	for i := range vectors {
		if len(vectors[i]) == 0 {
			return nil, SetMetrics{}, errors.New("semantic_measurement_empty_vector")
		}
		for j := i + 1; j < len(vectors); j++ {
			if len(vectors[i]) != len(vectors[j]) {
				return nil, SetMetrics{}, errors.New("semantic_measurement_dimension_mismatch")
			}
			sim := cosine(vectors[i], vectors[j])
			set.PairwiseSimilarity[i][j], set.PairwiseSimilarity[j][i] = sim, sim
			total += sim
			pairs++
		}
	}
	set.MeanSimilarity = ratio(total, float64(pairs))
	set.Diversity = 1 - set.MeanSimilarity
	set.Coverage = coverage(set.PairwiseSimilarity)
	return items, set, nil
}

func cosine(a, b []float64) float64 {
	var dot, aa, bb float64
	for i := range a {
		dot += a[i] * b[i]
		aa += a[i] * a[i]
		bb += b[i] * b[i]
	}
	if aa == 0 || bb == 0 {
		if aa == 0 && bb == 0 {
			return 1
		}
		return 0
	}
	return dot / math.Sqrt(aa*bb)
}

// CrossSectionRepetition returns the mean lexical similarity of distinct
// sections. It is deliberately a deterministic text signal, not a semantic
// claim: callers can name the basis alongside the number they publish.
func CrossSectionRepetition(sections []string) float64 {
	if len(sections) < 2 {
		return 0
	}
	var total float64
	var pairs int
	for i := 0; i < len(sections); i++ {
		for j := i + 1; j < len(sections); j++ {
			total += lexicalSimilarity(words(sections[i]), words(sections[j]))
			pairs++
		}
	}
	return ratio(total, float64(pairs))
}

// contentStopWords are excluded from novelty because they recur in any English
// prose and would swamp the signal. The list is deliberately short and closed:
// a longer list starts encoding a topic, and this measure has to work on text
// whose topic it knows nothing about.
var contentStopWords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true, "of": true, "to": true,
	"in": true, "on": true, "for": true, "is": true, "are": true, "was": true, "were": true,
	"that": true, "this": true, "it": true, "its": true, "as": true, "with": true,
	"after": true, "before": true, "once": true, "every": true, "not": true, "we": true,
	"our": true, "you": true, "your": true, "they": true, "their": true, "be": true,
	"been": true, "being": true, "by": true, "from": true, "at": true, "can": true,
	"will": true, "would": true, "could": true, "has": true, "have": true, "had": true,
	"than": true, "then": true, "so": true, "such": true, "those": true, "these": true,
	"there": true, "here": true, "what": true, "which": true, "who": true, "when": true,
	"where": true, "how": true, "all": true, "any": true, "both": true, "each": true,
	"more": true, "most": true, "other": true, "some": true, "only": true, "own": true,
	"same": true, "too": true, "very": true, "just": true,
}

// contentTerms returns the distinct content words of a text.
func contentTerms(text string) map[string]bool {
	out := map[string]bool{}
	for _, w := range words(text) {
		if len(w) > 2 && !contentStopWords[w] {
			out[w] = true
		}
	}
	return out
}

// SectionNovelty returns the share of a section's content terms that did not
// appear in the section immediately before it. It answers whether a passage
// brought new material, which is a different question from whether it reused
// wording: prose that re-argues a settled point in fresh words scores low here
// and low on CrossSectionRepetition at the same time, because it repeats the
// substance while sharing few tokens.
//
// The comparison is against the preceding section alone rather than against
// every prior section, and that choice is what makes a fixed threshold usable.
// Measured against all prior text the value falls purely because there is more
// prior text to have already said something: across 62 committed sections the
// median ran 0.721, 0.614, then 0.472 by the third position, so any single
// floor would pass an opening section unconditionally and fail a closing one
// unconditionally. Against the preceding section alone the same corpus gives
// 0.721, 0.758, 0.735, 0.719 by position -- flat, and therefore gateable.
//
// The measure rewards novelty, so a section that wanders off the document's
// subject scores near 1.0. That is its blind spot by construction, and it is
// why this belongs beside CrossSectionRepetition and the coherence verdict
// rather than in place of either. A passage has to be both new and on-subject,
// and neither number establishes the other.
func SectionNovelty(section, previous string) float64 {
	terms := contentTerms(section)
	if len(terms) == 0 {
		return 0
	}
	prior := contentTerms(previous)
	var fresh int
	for term := range terms {
		if !prior[term] {
			fresh++
		}
	}
	return float64(fresh) / float64(len(terms))
}

// MinSectionNovelty returns the lowest adjacent-section novelty in a document,
// which is the transition where the document came closest to adding nothing.
// Fewer than two sections have no transition and report 1.
func MinSectionNovelty(sections []string) float64 {
	if len(sections) < 2 {
		return 1
	}
	lowest := 1.0
	for i := 1; i < len(sections); i++ {
		if value := SectionNovelty(sections[i], sections[i-1]); value < lowest {
			lowest = value
		}
	}
	return lowest
}

// StyleDrift returns the mean absolute distance from the document's average
// section conformance. A document with one section has no measurable drift.
func StyleDrift(sectionConformance []float64) float64 {
	if len(sectionConformance) < 2 {
		return 0
	}
	var mean float64
	for _, score := range sectionConformance {
		mean += score
	}
	mean /= float64(len(sectionConformance))
	var total float64
	for _, score := range sectionConformance {
		total += math.Abs(score - mean)
	}
	return total / float64(len(sectionConformance))
}

func wordsWithSpans(text string) []Span {
	var out []Span
	start := -1
	for i, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'' {
			if start < 0 {
				start = i
			}
			continue
		}
		if start >= 0 {
			out = append(out, Span{Start: start, End: i, Text: text[start:i]})
			start = -1
		}
	}
	if start >= 0 {
		out = append(out, Span{Start: start, End: len(text), Text: text[start:]})
	}
	return out
}

// rougeL returns token-level ROUGE-L F. It is a deterministic lexical
// homogenization signal, not a semantic similarity claim.
func rougeL(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1
	}
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	dp := make([][]int, len(a)+1)
	for i := range dp {
		dp[i] = make([]int, len(b)+1)
	}
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] > dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}
	lcs := float64(dp[len(a)][len(b)])
	precision := lcs / float64(len(b))
	recall := lcs / float64(len(a))
	return ratio(2*precision*recall, precision+recall)
}

func words(text string) []string {
	spans := wordsWithSpans(text)
	out := make([]string, len(spans))
	for i, s := range spans {
		out[i] = strings.ToLower(s.Text)
	}
	return out
}

func sentenceWordLengths(text string) []int {
	lengths := []int{}
	count := 0
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			count++
		}
		if r == '.' || r == '!' || r == '?' || r == '。' || r == '！' || r == '？' {
			if count > 0 {
				lengths = append(lengths, count)
				count = 0
			}
		}
	}
	if count > 0 {
		lengths = append(lengths, count)
	}
	return lengths
}

func compressionRatio(text string) float64 {
	if text == "" {
		return 0
	}
	var b bytes.Buffer
	z := gzip.NewWriter(&b)
	_, _ = z.Write([]byte(text))
	_ = z.Close()
	return ratio(float64(b.Len()), float64(len([]byte(text))))
}

func distinctN(ws []string) map[string]float64 {
	out := map[string]float64{}
	for n := 1; n <= 3; n++ {
		if len(ws) < n {
			out["n"+itoa(n)] = 0
			continue
		}
		seen := map[string]struct{}{}
		for i := 0; i+n <= len(ws); i++ {
			seen[strings.Join(ws[i:i+n], " ")] = struct{}{}
		}
		out["n"+itoa(n)] = ratio(float64(len(seen)), float64(len(ws)-n+1))
	}
	return out
}

func selfRepetition(ws []string) float64 {
	if len(ws) < 2 {
		return 0
	}
	seen := map[string]struct{}{}
	repeated := 0
	for i := 0; i+2 <= len(ws); i++ {
		key := strings.Join(ws[i:i+2], " ")
		if _, ok := seen[key]; ok {
			repeated++
		} else {
			seen[key] = struct{}{}
		}
	}
	return ratio(float64(repeated), float64(len(ws)-1))
}

func readability(ws []Span, sentences []int) Readability {
	if len(ws) == 0 || len(sentences) == 0 {
		return Readability{}
	}
	syllables := 0
	easy := 0
	for _, w := range ws {
		syllables += syllableCount(w.Text)
		if isDaleChallEasy(w.Text) {
			easy++
		}
	}
	wordsN, sentN := float64(len(ws)), float64(len(sentences))
	return Readability{
		FleschKincaid: 0.39*(wordsN/sentN) + 11.8*(float64(syllables)/wordsN) - 15.59,
		DaleChall:     0.1579*((wordsN-float64(easy))/wordsN*100) + 0.0496*(wordsN/sentN),
		GunningFog:    0.4 * ((wordsN / sentN) + 100*float64(complexWords(ws))/wordsN),
	}
}

func mattr(ws []string, window int) float64 {
	if len(ws) == 0 {
		return 0
	}
	if len(ws) < window {
		return ratio(float64(uniqueStrings(ws)), float64(len(ws)))
	}
	var total float64
	for i := 0; i+window <= len(ws); i++ {
		total += ratio(float64(uniqueStrings(ws[i:i+window])), float64(window))
	}
	return total / float64(len(ws)-window+1)
}

func lexiconSpans(text string, lexicon []string) []Span {
	terms := append([]string(nil), lexicon...)
	sort.SliceStable(terms, func(i, j int) bool { return len(terms[i]) > len(terms[j]) })
	var out []Span
	lower := strings.ToLower(text)
	for _, term := range terms {
		term = strings.TrimSpace(term)
		if term == "" {
			continue
		}
		needle := strings.ToLower(term)
		for at := 0; at < len(lower); {
			i := strings.Index(lower[at:], needle)
			if i < 0 {
				break
			}
			i += at
			out = append(out, Span{Start: i, End: i + len(term), Text: text[i : i+len(term)], Term: term})
			at = i + len(needle)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Start == out[j].Start {
			return out[i].End < out[j].End
		}
		return out[i].Start < out[j].Start
	})
	return out
}

func lexicalSimilarity(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1
	}
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	left, right := map[string]float64{}, map[string]float64{}
	for _, w := range a {
		left[w]++
	}
	for _, w := range b {
		right[w]++
	}
	var dot, al, bl float64
	for w, n := range left {
		dot += n * right[w]
		al += n * n
	}
	for _, n := range right {
		bl += n * n
	}
	cos := dot / math.Sqrt(al*bl)
	setA, setB := map[string]struct{}{}, map[string]struct{}{}
	for _, w := range a {
		setA[w] = struct{}{}
	}
	for _, w := range b {
		setB[w] = struct{}{}
	}
	intersection := 0
	for w := range setA {
		if _, ok := setB[w]; ok {
			intersection++
		}
	}
	jaccard := ratio(float64(intersection), float64(len(setA)+len(setB)-intersection))
	return (cos + jaccard) / 2
}

func coverage(matrix [][]float64) float64 {
	if len(matrix) < 2 {
		return 0
	}
	// A simple deterministic coverage proxy: fraction of candidates whose
	// nearest neighbour is not above the set's mean similarity.
	var total float64
	for i, row := range matrix {
		nearest := 1.0
		for j, v := range row {
			if i != j && v < nearest {
				nearest = v
			}
		}
		total += nearest
	}
	return total / float64(len(matrix))
}

func variance(xs []int) float64 {
	if len(xs) == 0 {
		return 0
	}
	var mean float64
	for _, x := range xs {
		mean += float64(x)
	}
	mean /= float64(len(xs))
	var sum float64
	for _, x := range xs {
		d := float64(x) - mean
		sum += d * d
	}
	return sum / float64(len(xs))
}

func uniqueWords(ws []Span) int {
	seen := map[string]struct{}{}
	for _, w := range ws {
		seen[strings.ToLower(w.Text)] = struct{}{}
	}
	return len(seen)
}

func uniqueStrings(ws []string) int {
	seen := map[string]struct{}{}
	for _, w := range ws {
		seen[w] = struct{}{}
	}
	return len(seen)
}

func ratio(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

func complexWords(ws []Span) int {
	count := 0
	for _, w := range ws {
		if syllableCount(w.Text) >= 3 {
			count++
		}
	}
	return count
}

func syllableCount(word string) int {
	w := strings.ToLower(word)
	count := 0
	prevVowel := false
	for _, r := range w {
		vowel := strings.ContainsRune("aeiouy", r)
		if vowel && !prevVowel {
			count++
		}
		prevVowel = vowel
	}
	if strings.HasSuffix(w, "e") && count > 1 {
		count--
	}
	if count == 0 {
		count = 1
	}
	return count
}

func isDaleChallEasy(word string) bool {
	w := strings.ToLower(word)
	return len([]rune(w)) <= 4 || strings.Contains(" the of and to in is you that it he was for on are as with his they i at be this have from or one had by word but not what all were we when your can said there use an each which she do how their if will up other about out many then them these so some her would make like him into time has look two more write go see number no way could people my than first water been call who oil its now find long down day did get come made may part", " "+w+" ")
}

// itoa formats the value. It previously returned "1", "2", or "3" for
// everything else, which meant every semantic measurement published a basis
// claiming three dimensions regardless of the embedding it actually used: a
// 768-dimension nomic-embed-text measurement described itself as
// three-dimensional, and the basis string is precisely the field a reader
// consults to decide whether a similarity number can be trusted.
func itoa(n int) string { return strconv.Itoa(n) }

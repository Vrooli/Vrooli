// Package corpusgen grows a provider's golden eval corpus from its own live
// index. It samples the index (stratified across the facets the search contract
// exposes), inverts each sampled item to a natural-language query that should
// retrieve it (a positive case), optionally proposes hard negatives (plausible
// queries with no good answer), and de-dupes candidates against the existing
// corpus and each other.
//
// The package is transport-free and store-free: it depends only on three seams —
// a Sampler (reach the index), an Inverter (the LLM), and a Deduper — so the
// orchestration is unit-testable with deterministic fakes and the same core runs
// against the live gateway in production. Persistence (append-to-suite on
// --apply) and adequacy scoring are the caller's job; Generate only returns the
// de-duped proposals + the sampling stats.
//
// Every proposed case is marked tags:["generated", <stratum>]. That marker is
// load-bearing: the sweep ALWAYS holds generated cases out of the tuning fold
// (overfit guard #2), so a tuning can never be selected on cases a machine wrote
// for it. Generation AUGMENTS the curated golden core; it never replaces it.
package corpusgen

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"strings"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

// Sampler draws a sample of index items for the provider it was constructed
// for (the descriptor is bound at construction, keeping this seam transport- and
// provider-agnostic). It should spread the sample across the strata the contract
// exposes; target is the desired item count (a hint, not a hard cap).
type Sampler interface {
	Sample(ctx context.Context, target int) ([]Item, error)
}

// Deps are the three seams the Generator composes.
type Deps struct {
	Sampler   Sampler
	Inverter  Inverter
	Deduper   Deduper
	Validator ReferentialValidator
}

// ReferentialValidator re-confirms that a generated positive query can still
// retrieve the sampled item it points at. It is optional for pure offline tests,
// but production wires it so generation cannot emit phantom labels.
type ReferentialValidator interface {
	ValidPositive(ctx context.Context, it Item, query string, topK int) (bool, error)
}

// Options tune one generation run. Zero values fall back to the Default* below.
type Options struct {
	// Count is the target number of positive cases to propose.
	Count int
	// Negatives requests hard-negative cases in addition to the positives.
	Negatives bool
	// NegativeRatio is the negatives-to-positives ratio (e.g. 0.25 = one negative
	// per four positives), floored at one negative when Negatives is set.
	NegativeRatio float64
	// TopK is stamped as expect_within_top_k on every generated positive case
	// (the rank band the expected id should land within).
	TopK int
	// GibberishCeiling is stamped as expect_max_score on every generated negative
	// (the junk-rejection ceiling a good engine must stay under).
	GibberishCeiling float64
}

// Defaults. DefaultCount matches eval.MinPositiveCases so a no-argument generate
// brings a thin corpus up to the adequacy floor in one pass.
const (
	DefaultCount            = 12
	DefaultNegativeRatio    = 0.25
	DefaultTopK             = 5
	DefaultGibberishCeiling = 0.5
)

func (o Options) withDefaults() Options {
	if o.Count <= 0 {
		o.Count = DefaultCount
	}
	if o.NegativeRatio <= 0 {
		o.NegativeRatio = DefaultNegativeRatio
	}
	if o.TopK <= 0 {
		o.TopK = DefaultTopK
	}
	if o.GibberishCeiling <= 0 {
		o.GibberishCeiling = DefaultGibberishCeiling
	}
	return o
}

// Proposal is one generated case + the provenance that produced it.
type Proposal struct {
	Case     *evalv1.EvalCase
	SourceID string // the inverted item's id (empty for a negative)
	Stratum  string
}

// Result is what Generate returns: the de-duped proposals (positives first, then
// negatives) plus the sampling/inversion stats the caller renders and feeds into
// adequacy (Strata is the set of strata the live sample covered).
type Result struct {
	Proposed []Proposal
	Sampled  int
	Inverted int // candidate queries the inverter produced (pre-dedup)
	Deduped  int // candidates dropped as near-duplicates
	Rejected int // candidates dropped by quality/referential filters
	Strata   []string
}

// Generator orchestrates one corpus-generation pass over the three seams.
type Generator struct {
	deps Deps
	opts Options
}

// New constructs a Generator. Sampler, Inverter, and Deduper must be non-nil at
// Generate time (New does not validate so a caller can wire them lazily).
func New(d Deps, o Options) *Generator {
	return &Generator{deps: d, opts: o.withDefaults()}
}

// Generate samples the provider's index, inverts items into candidate cases,
// de-dupes them against the suite's existing queries (and each other), and
// returns the proposals. It mutates nothing — the suite is read-only input.
func (g *Generator) Generate(ctx context.Context, suite *evalv1.EvalSuite) (*Result, error) {
	if g.deps.Sampler == nil || g.deps.Inverter == nil || g.deps.Deduper == nil {
		return nil, fmt.Errorf("corpusgen: Sampler, Inverter and Deduper are all required")
	}
	items, err := g.deps.Sampler.Sample(ctx, g.opts.Count)
	if err != nil {
		return nil, fmt.Errorf("sample index: %w", err)
	}

	res := &Result{Sampled: len(items), Strata: distinctStrata(items)}

	// `seen` accumulates every query we must not paraphrase: the suite's existing
	// case queries plus every candidate accepted so far (so two sampled items that
	// invert to the same question don't both get proposed).
	seen := existingQueries(suite)
	usedIDs := existingCaseIDs(suite)

	// --- positive cases: invert each sampled item ---------------------------
	for _, it := range items {
		if len(positives(res)) >= g.opts.Count {
			break
		}
		q, err := g.deps.Inverter.InvertPositive(ctx, it)
		if err != nil || strings.TrimSpace(q) == "" {
			continue // a failed inversion is skipped, never fatal
		}
		res.Inverted++
		if isEchoQuery(q, it) {
			res.Rejected++
			continue
		}
		if g.deps.Validator != nil {
			ok, err := g.deps.Validator.ValidPositive(ctx, it, q, g.opts.TopK)
			if err != nil || !ok {
				res.Rejected++
				continue
			}
		}
		if g.deps.Deduper.IsDuplicate(q, seen) {
			res.Deduped++
			continue
		}
		id := positiveCaseID(q)
		if usedIDs[id] {
			res.Deduped++
			continue
		}
		seen = append(seen, q)
		usedIDs[id] = true
		res.Proposed = append(res.Proposed, Proposal{
			Case:     positiveCase(id, q, it, g.opts.TopK),
			SourceID: it.ID,
			Stratum:  it.Stratum(),
		})
	}

	// --- hard negatives: invert a slice of the sample into no-answer queries -
	if g.opts.Negatives {
		want := negativeCount(len(positives(res)), g.opts.NegativeRatio)
		for _, it := range items {
			if negativeProposals(res) >= want {
				break
			}
			q, err := g.deps.Inverter.InvertNegative(ctx, it)
			if err != nil || strings.TrimSpace(q) == "" {
				continue
			}
			res.Inverted++
			if g.deps.Deduper.IsDuplicate(q, seen) {
				res.Deduped++
				continue
			}
			id := negativeCaseID(q)
			if usedIDs[id] {
				res.Deduped++
				continue
			}
			seen = append(seen, q)
			usedIDs[id] = true
			res.Proposed = append(res.Proposed, Proposal{
				Case:    negativeCase(id, q, it, g.opts.GibberishCeiling),
				Stratum: it.Stratum(),
			})
		}
	}

	return res, nil
}

// Summary renders the one-line stats string the response carries.
func (r *Result) Summary() string {
	pos, neg := 0, 0
	for _, p := range r.Proposed {
		if p.Case.GetExpectNoStrongHit() {
			neg++
		} else {
			pos++
		}
	}
	return fmt.Sprintf("sampled %d item(s) across %d strata → %d inverted → %d rejected → %d deduped → %d proposed (%d positive, %d negative)",
		r.Sampled, len(r.Strata), r.Inverted, r.Rejected, r.Deduped, len(r.Proposed), pos, neg)
}

// --- case construction ------------------------------------------------------

func positiveCase(id, query string, it Item, topK int) *evalv1.EvalCase {
	return &evalv1.EvalCase{
		CaseId:           id,
		Query:            query,
		Status:           "candidate",
		Tags:             []string{"generated", it.Stratum()},
		ExpectIds:        []string{it.ID},
		ExpectWithinTopK: int32(topK),
		Note:             "machine-generated by query inversion (review before trusting)",
	}
}

func negativeCase(id, query string, it Item, ceiling float64) *evalv1.EvalCase {
	return &evalv1.EvalCase{
		CaseId:            id,
		Query:             query,
		Status:            "candidate",
		Tags:              []string{"generated", "gibberish", it.Stratum()},
		ExpectNoStrongHit: true,
		ExpectMaxScore:    ceiling,
		Note:              "machine-generated hard negative (review before trusting)",
	}
}

// --- helpers ----------------------------------------------------------------

func positiveCaseID(query string) string { return "gen-" + hash8(query) }
func negativeCaseID(query string) string { return "gen-neg-" + hash8(query) }

// hash8 is a stable 8-hex FNV-1a of the normalized query, so re-running generate
// produces the same case_id for the same query (apply stays idempotent).
func hash8(query string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.ToLower(strings.Join(strings.Fields(query), " "))))
	return fmt.Sprintf("%08x", h.Sum32())
}

func negativeCount(positives int, ratio float64) int {
	n := int(math.Ceil(float64(positives) * ratio))
	if n < 1 {
		n = 1
	}
	return n
}

func positives(r *Result) []Proposal {
	var out []Proposal
	for _, p := range r.Proposed {
		if !p.Case.GetExpectNoStrongHit() {
			out = append(out, p)
		}
	}
	return out
}

func negativeProposals(r *Result) int {
	n := 0
	for _, p := range r.Proposed {
		if p.Case.GetExpectNoStrongHit() {
			n++
		}
	}
	return n
}

func existingQueries(suite *evalv1.EvalSuite) []string {
	out := make([]string, 0, len(suite.GetCases()))
	for _, c := range suite.GetCases() {
		if q := strings.TrimSpace(c.GetQuery()); q != "" {
			out = append(out, q)
		}
	}
	return out
}

func existingCaseIDs(suite *evalv1.EvalSuite) map[string]bool {
	out := map[string]bool{}
	for _, c := range suite.GetCases() {
		out[c.GetCaseId()] = true
	}
	return out
}

func distinctStrata(items []Item) []string {
	set := map[string]struct{}{}
	for _, it := range items {
		set[it.Stratum()] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func isEchoQuery(query string, it Item) bool {
	titleTokens := contentTokenSet(it.Title)
	if len(titleTokens) == 0 {
		return false
	}
	queryTokens := contentTokenSet(query)
	if len(queryTokens) == 0 {
		return true
	}
	querySubsetTitle, titleSubsetQuery := true, true
	for tok := range queryTokens {
		if _, ok := titleTokens[tok]; !ok {
			querySubsetTitle = false
			break
		}
	}
	for tok := range titleTokens {
		if _, ok := queryTokens[tok]; !ok {
			titleSubsetQuery = false
			break
		}
	}
	if !querySubsetTitle && !titleSubsetQuery {
		return false
	}
	return !hasAddedIntentToken(query, titleTokens)
}

func hasAddedIntentToken(query string, titleTokens map[string]struct{}) bool {
	for _, tok := range normalizedTokens(query) {
		if _, inTitle := titleTokens[tok]; inTitle {
			continue
		}
		if _, stop := echoStopWords[tok]; !stop {
			return true
		}
	}
	return false
}

func contentTokenSet(s string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, tok := range normalizedTokens(s) {
		if _, stop := echoStopWords[tok]; stop {
			continue
		}
		out[tok] = struct{}{}
	}
	return out
}

func normalizedTokens(s string) []string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		b.WriteByte(' ')
	}
	return strings.Fields(b.String())
}

var echoStopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "as": {}, "can": {}, "do": {}, "for": {}, "how": {},
	"i": {}, "in": {}, "is": {}, "me": {}, "of": {}, "on": {}, "please": {}, "run": {}, "the": {},
	"to": {}, "use": {}, "what": {}, "with": {},
}

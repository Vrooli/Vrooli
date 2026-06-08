package measures

import (
	"context"
	"sort"
	"strings"
	"time"
)

// Match is one semantic match of a question against a measure declaration, with
// the retrieval similarity that produced it.
type Match struct {
	Decl  MeasureDeclaration
	Score float64
}

// Matcher is the semantic-match seam: given a natural-language question it
// returns candidate measures ranked best-first. The production implementation
// is the central measures index (Phase 4: aisearch-go hybrid search over the
// measure questions, embedded via MeasureComposer); tests and the search-hub
// reference provider inject a deterministic matcher. Keeping it a seam is what
// lets the engine — the reusable brain — live here while the index (which needs
// qdrant + an embedder) lives in the hosting scenario.
type Matcher interface {
	Match(ctx context.Context, question string, limit int) ([]Match, error)
}

// Executor is the execution-proxy seam: it runs a matched measure against its
// owning scenario and returns the computed result with provenance. The
// production implementation POSTs to the scenario's measures serve endpoint
// (HTTPExecutor); tests inject a fake. The engine only calls it AFTER the gate
// authorizes execution, so a write/destructive or low-confidence measure is
// never handed to the executor.
type Executor interface {
	Execute(ctx context.Context, decl MeasureDeclaration, params map[string]string) (MeasureResult, error)
}

// MeasureHit is the engine's structured answer to one analytical question — the
// Go mirror of the proto routing.MeasureHit the measures provider serializes for
// search-hub. Which fields are populated encodes the auto-execution contract
// (see Gate / the proto comment): Answer+ExecutedQuery set ⇒ executed; Needs
// non-empty ⇒ asked-not-executed; Answer empty + Needs empty ⇒ resolved but
// withheld for confirmation.
type MeasureHit struct {
	MeasureID     string            `json:"measure_id"`
	Scenario      string            `json:"scenario"`
	Params        map[string]string `json:"params,omitempty"`
	Answer        string            `json:"answer,omitempty"`
	Needs         []string          `json:"needs,omitempty"`
	Effect        string            `json:"effect"`
	ExecutedQuery string            `json:"executed_query,omitempty"`
	Confidence    float64           `json:"confidence"`
	// Score is the retrieval similarity of the match (the SearchHit.score the
	// provider sets on the outer hit). Carried for the provider's convenience;
	// distinct from Confidence (param-resolution certainty).
	Score float64 `json:"score"`
	// GateReason is the human-readable gate verdict (provenance / --explain).
	GateReason string `json:"gate_reason,omitempty"`
}

// EngineOption configures an Engine.
type EngineOption func(*Engine)

// WithThreshold overrides the auto-execute confidence threshold θ.
func WithThreshold(t float64) EngineOption {
	return func(e *Engine) {
		if t > 0 {
			e.threshold = t
		}
	}
}

// WithExtractor sets the constrained param extractor (tiers 2/3). Defaults to
// NoopExtractor (abstain) so the engine is safe before an LLM is wired.
func WithExtractor(x ParamExtractor) EngineOption {
	return func(e *Engine) {
		if x != nil {
			e.extractor = x
		}
	}
}

// WithValues sets the dynamic-enum value provider.
func WithValues(v ValuesProvider) EngineOption {
	return func(e *Engine) { e.values = v }
}

// WithExecutor sets the execution-proxy. Without it the engine resolves + gates
// but never executes (every read-only hit returns resolved-but-unexecuted).
func WithExecutor(x Executor) EngineOption {
	return func(e *Engine) { e.executor = x }
}

// WithEngineClock injects the time source used to anchor relative time-window
// resolution (a seam for deterministic tests). Defaults to time.Now. (Named
// distinctly from the Registry's WithClock since both live in this package.)
func WithEngineClock(now func() time.Time) EngineOption {
	return func(e *Engine) {
		if now != nil {
			e.now = now
		}
	}
}

// WithLocation sets the timezone for time-window resolution (defaults to UTC).
func WithLocation(loc *time.Location) EngineOption {
	return func(e *Engine) { e.loc = loc }
}

// Engine answers analytical questions by matching them to declared measures,
// resolving parameters, applying the auto-execution gate, and (for safe
// read-only measures at high confidence) executing and returning the answer. It
// is the reusable provider brain shared by the search-hub reference provider
// (Phase 3) and the measures-health central provider (Phase 4); only the
// Matcher / Executor / Completer seams differ between them.
type Engine struct {
	matcher   Matcher
	extractor ParamExtractor
	values    ValuesProvider
	executor  Executor
	threshold float64
	now       func() time.Time
	loc       *time.Location
}

// NewEngine constructs an Engine over a Matcher. Options wire the extractor,
// executor, dynamic-enum values, threshold, and clock.
func NewEngine(matcher Matcher, opts ...EngineOption) *Engine {
	e := &Engine{
		matcher:   matcher,
		extractor: NoopExtractor{},
		threshold: DefaultConfidenceThreshold,
		now:       time.Now,
	}
	for _, o := range opts {
		o(e)
	}
	return e
}

// Answer resolves a single analytical question to a MeasureHit, or returns
// (nil, nil) when no measure matches (the caller reports an honest empty
// result). It never auto-executes a write/destructive measure and never guesses
// a missing required param (the Gate enforces both); a resolution/execution
// error is returned so the caller can degrade gracefully.
func (e *Engine) Answer(ctx context.Context, question string) (*MeasureHit, error) {
	matches, err := e.matcher.Match(ctx, question, 1)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, nil
	}
	best := matches[0]
	return e.answerFor(ctx, question, best.Decl, best.Score)
}

// answerFor runs the resolve→gate→execute pipeline for an already-matched
// declaration. It is split out so a provider that does its own matching (or a
// test) can drive the gate/exec path directly.
func (e *Engine) answerFor(ctx context.Context, question string, decl MeasureDeclaration, score float64) (*MeasureHit, error) {
	res, err := ResolveParams(ctx, question, decl, ResolveOptions{
		Now:       e.now(),
		Loc:       e.loc,
		Extractor: e.extractor,
		Values:    e.values,
	})
	if err != nil {
		return nil, err
	}

	hit := &MeasureHit{
		MeasureID:  decl.Name,
		Scenario:   decl.Scenario,
		Params:     res.Params,
		Needs:      res.Needs,
		Effect:     string(decl.Effect),
		Confidence: res.Confidence,
		Score:      score,
	}

	decision := Gate(decl, res, e.threshold)
	hit.GateReason = decision.Reason
	if !decision.Execute() || e.executor == nil {
		// Resolved but not executed: needs[], confirmation, or no executor wired
		// (resolve-first). The contract fields above already encode the state.
		return hit, nil
	}

	result, err := e.executor.Execute(ctx, decl, res.Params)
	if err != nil {
		return nil, err
	}
	hit.Answer = RenderSummary(decl, result, res.Params)
	hit.ExecutedQuery = result.Provenance.ExecutedQuery
	return hit, nil
}

// RenderSummary renders a measure's one-line answer from its result + resolved
// params using the declared summary_template ("{count} items ({window})"),
// filling {value_field} from the scalar value and {param} from resolved params.
// With no template it falls back to "<value> <unit>". Table/series measures
// (no scalar value) fall back to a row count.
func RenderSummary(decl MeasureDeclaration, result MeasureResult, params map[string]string) string {
	repl := make(map[string]string, len(params)+1)
	for k, v := range params {
		repl[k] = v
	}
	if decl.Result.ValueField != "" {
		repl[decl.Result.ValueField] = result.Value
	}

	tmpl := strings.TrimSpace(decl.Result.SummaryTemplate)
	if tmpl != "" {
		return fillTemplate(tmpl, repl)
	}

	// No template: best-effort scalar rendering.
	val := strings.TrimSpace(result.Value)
	if val == "" {
		// table/series: report the row count.
		return pluralize(len(result.Fields), "row")
	}
	if u := strings.TrimSpace(decl.Result.Unit); u != "" {
		return val + " " + u
	}
	return val
}

// fillTemplate replaces {key} occurrences in tmpl from repl. Unknown
// placeholders are left intact (so a missing field is visible, not silently
// blanked).
func fillTemplate(tmpl string, repl map[string]string) string {
	// Deterministic order so overlapping keys (none expected) are stable.
	keys := make([]string, 0, len(repl))
	for k := range repl {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := tmpl
	for _, k := range keys {
		out = strings.ReplaceAll(out, "{"+k+"}", repl[k])
	}
	return out
}

// pluralize renders "<n> <noun>" with a naive plural 's'.
func pluralize(n int, noun string) string {
	s := noun
	if n != 1 {
		s += "s"
	}
	return itoa(n) + " " + s
}

// itoa avoids importing strconv just for one int render.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

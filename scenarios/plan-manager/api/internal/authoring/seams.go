package authoring

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	planmodel "plan-manager/internal/planmodel"
	"plan-manager/internal/probes"

	"github.com/google/uuid"
)

// PlanWriter is the write seam onto the plans SSOT. Finalize maps the session's
// sections into a plans.Plan and persists it through this seam. Production wraps
// the plans domain Service (a CreatePlan adapter in the handler module, mirroring
// validation's planAdapter); tests inject a fake. authoring never owns the plan
// record — it only composes and writes it.
type PlanWriter interface {
	CreatePlan(ctx context.Context, p planmodel.Plan) (planmodel.Plan, error)
}

// PlanReader is the read seam back into the plans SSOT. Finalize uses it as a
// read-after-write guard: a successful finalize response must refer to a plan
// that resolves through the same plans-domain paths callers use.
type PlanReader interface {
	GetPlan(ctx context.Context, idOrSlug string) (planmodel.Plan, error)
	RenderPlan(ctx context.Context, idOrSlug string) (string, error)
}

// PlanRenderer renders an in-progress plan to its markdown review artifact so the
// wizard can offer a render-preview before finalize. Production wires
// plans.RenderMarkdown; a nil renderer disables preview (the service returns a
// typed "preview unavailable" error). The renderer is pure — no persistence.
type PlanRenderer interface {
	Render(p planmodel.Plan) string
}

// AnchorIntentDeriver derives the typed regression-anchor INTENT for a
// plan-in-progress — the "before" the executor will snapshot. It is pure and
// deterministic (title/slug → typed intent fields), needs no git-control-tower,
// and never goes stale: the actual baseline snapshot is captured fresh at
// execution start (see the execution InputFreshener seam), however many days
// later. Production wires the default deriver; tests inject a fake.
type AnchorIntentDeriver interface {
	// DeriveAnchorIntent returns the boundary-native regression-anchor intent
	// block for the given plan title/slug and change boundary. Affected scenarios
	// and the tiered baseline/diff commands are derived from the boundary; it
	// always succeeds — intent is cheap, deterministic, and never stale (no
	// snapshot, no dependency).
	DeriveAnchorIntent(ctx context.Context, title, slug string, boundary planmodel.ChangeBoundary) string
}

// ReferenceSuggester discovers reviewable code/doc/req reference candidates from
// search-hub's Answer projection. Production shells `search-hub query --json`
// through the CommandRunner seam and routes the hits by locator shape; a nil seam
// or an error degrades honestly to no candidates (never a fabricated reference).
type ReferenceSuggester interface {
	// Suggest returns reference candidates discovered for the given query (the
	// session's title + scope + technical approach). An error degrades to no
	// candidates honestly.
	Suggest(ctx context.Context, query string) ([]ReferenceCandidate, error)
}

// ContextDiscoverer proposes relevant-context setup items from decomposed
// concepts. It discovers candidates only; acceptance remains an authoring
// decision in the Service.
type ContextDiscoverer interface {
	DiscoverContext(ctx context.Context, title string, concepts []string, complexity string) (ContextDiscoveryResult, error)
}

// CommandRunner is the exec seam shared by the production discovery sources (the
// live dispatch path to search-hub for reference suggestions and to
// prompt-manager / cli-health for context discovery). Production wires execRunner
// (LookPath-guarded, timeout-bounded); tests inject a fake. A nil runner means
// "no live dispatch" — the source degrades honestly.
type CommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

// DefaultRunner returns the production CommandRunner (LookPath-guarded,
// timeout-bounded). Wired in the handler module; tests inject a fake instead.
func DefaultRunner() CommandRunner { return execRunner }

// autofillTimeout bounds a single context-discovery dispatch. Generous (a
// discovery probe can shell out) but finite so a hung command cannot block the
// wizard forever.
const autofillTimeout = 2 * time.Minute

// execRunner is the production CommandRunner. It guards against running an
// arbitrary binary by requiring the command to be on PATH, bounds the call with
// a timeout, and returns combined output. Never fabricates results: a failure is
// a real error the caller turns into a degraded autofill honestly.
func execRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	if _, err := exec.LookPath(name); err != nil {
		return nil, fmt.Errorf("command %q not found on PATH: %w", name, err)
	}
	ctx, cancel := context.WithTimeout(ctx, autofillTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("run %s %s: %w", name, strings.Join(args, " "), err)
	}
	return out, nil
}

// --- production discovery sources ---

// defaultAnchorIntentDeriver is the production AnchorIntentDeriver: a pure,
// deterministic derivation of the typed regression-anchor intent block (no
// git-control-tower, no snapshot). It promotes the structured intent template to
// the primary authoring output.
type defaultAnchorIntentDeriver struct{}

// DefaultAnchorIntentDeriver returns the production AnchorIntentDeriver. Wired in
// the handler module; tests inject a fake instead.
func DefaultAnchorIntentDeriver() AnchorIntentDeriver { return defaultAnchorIntentDeriver{} }

func (defaultAnchorIntentDeriver) DeriveAnchorIntent(_ context.Context, title, slug string, boundary planmodel.ChangeBoundary) string {
	return RegressionAnchorIntentTemplate(title, slug, boundary)
}

// cmdReferenceSuggester discovers reviewable reference candidates from
// search-hub's Answer projection through the CommandRunner seam. It sends the
// rich query broad (let search-hub federate/rank), parses the typed
// QueryResponse, and routes the hits by locator shape — keeping only hits that
// resolve to a [CODE:]/[DOC:]/[REQ:] locator. A nil runner / error / empty
// result degrades honestly to no candidates.
type cmdReferenceSuggester struct{ run CommandRunner }

// NewCommandReferenceSuggester wires the production ReferenceSuggester over the
// given CommandRunner (search-hub). A nil runner always degrades to no
// candidates.
func NewCommandReferenceSuggester(run CommandRunner) ReferenceSuggester {
	return cmdReferenceSuggester{run: run}
}

// referenceSuggestTimeout bounds the author-initiated search-hub query. Tighter
// than the generic autofill timeout because it is on the wizard's interactive
// path; a slow/hung search-hub degrades to no candidates rather than hanging.
const referenceSuggestTimeout = 15 * time.Second

func (e cmdReferenceSuggester) Suggest(ctx context.Context, query string) ([]ReferenceCandidate, error) {
	if e.run == nil {
		return nil, fmt.Errorf("search-hub unavailable")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("no query text for reference suggestion")
	}
	ctx, cancel := context.WithTimeout(ctx, referenceSuggestTimeout)
	defer cancel()
	out, err := e.run(ctx, "search-hub", "query", query, "--json")
	if err != nil {
		return nil, err
	}
	return parseReferenceSuggestions(out), nil
}

// searchHubQueryResponse is the subset of search-hub's QueryResponse (protojson)
// the suggester reads. protojson emits camelCase field names; the json tags match
// the wire keys (providerId, rerankScore, …).
type searchHubQueryResponse struct {
	Ranked []searchHubHit `json:"ranked"`
	Groups []struct {
		Hits []searchHubHit `json:"hits"`
	} `json:"groups"`
}

type searchHubHit struct {
	ProviderID  string  `json:"providerId"`
	Type        string  `json:"type"`
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Path        string  `json:"path"`
	Score       float64 `json:"score"`
	RerankScore float64 `json:"rerankScore"`
}

// parseReferenceSuggestions parses a search-hub QueryResponse and routes the hits
// by locator shape into reference candidates. The unified ranked list is
// preferred when present (post-rerank); otherwise the per-provider groups are
// flattened. A hit that does not resolve to a [CODE:]/[DOC:]/[REQ:] locator is
// dropped — the output locator shape IS the Answer-projection filter. Duplicate
// targets are kept once (highest-scored first). A parse failure yields no
// candidates (honest degradation, never a fabricated reference).
func parseReferenceSuggestions(out []byte) []ReferenceCandidate {
	var resp searchHubQueryResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil
	}
	hits := resp.Ranked
	if len(hits) == 0 {
		for _, g := range resp.Groups {
			hits = append(hits, g.Hits...)
		}
	}
	var out2 []ReferenceCandidate
	seen := map[string]bool{}
	for _, hit := range hits {
		kind, target, ok := referenceLocatorFromHit(hit)
		if !ok {
			continue
		}
		key := string(kind) + "\x00" + target
		if seen[key] {
			continue
		}
		seen[key] = true
		score := hit.RerankScore
		if score == 0 {
			score = hit.Score
		}
		out2 = append(out2, ReferenceCandidate{
			ID:         uuid.NewString(),
			Reference:  planmodel.Reference{ID: uuid.NewString(), Kind: kind, Target: target},
			Source:     strings.TrimSpace(hit.ProviderID),
			Confidence: score,
			Status:     ReferenceCandidatePending,
		})
	}
	return out2
}

// referenceLocatorFromHit routes one search hit to a [CODE:]/[DOC:]/[REQ:]
// locator by shape, or reports ok=false when the hit is not a code-location
// answer (it is dropped, or could be offered to relevant_context elsewhere).
// Order matters: an explicit req type is honored first, then doc/code path
// routing (so a docs path like PLAN-MODEL.md is never mis-detected as a
// requirement id by the uppercase-dash token it happens to contain), and only a
// bare id-shaped value with no usable path falls back to [REQ:].
func referenceLocatorFromHit(hit searchHubHit) (planmodel.ReferenceKind, string, bool) {
	path := strings.TrimSpace(hit.Path)
	typ := strings.ToLower(strings.TrimSpace(hit.Type))
	if typ == "req" || typ == "requirement" {
		if id := firstRequirementID(hit.ID, path); id != "" {
			return planmodel.ReferenceReq, id, true
		}
	}
	if path != "" {
		if isDocReferencePath(path) && !isCodeReferencePath(path) {
			return planmodel.ReferenceDoc, path, true
		}
		if isCodeReferencePath(path) {
			return planmodel.ReferenceCode, path, true
		}
	}
	// A bare requirement id with no usable code/doc path (e.g. id "PM-AUTHOR-002",
	// path empty). The whole value must BE a requirement id — not merely contain
	// an uppercase-dash token inside a longer path.
	if path == "" && isRequirementID(hit.ID) {
		return planmodel.ReferenceReq, strings.TrimSpace(hit.ID), true
	}
	return "", "", false
}

// requirementIDPattern matches a requirement id like OT-P0-001 or PM-AUTHOR-002:
// an uppercase prefix followed by one or more dash-separated alphanumeric groups.
var requirementIDPattern = regexp.MustCompile(`\b[A-Z]{2,}(?:-[A-Z0-9]+)+\b`)

// fullRequirementIDPattern anchors the same shape so a whole value can be tested
// as "is exactly a requirement id" (vs. merely containing one).
var fullRequirementIDPattern = regexp.MustCompile(`^[A-Z]{2,}(?:-[A-Z0-9]+)+$`)

func isRequirementID(v string) bool {
	return fullRequirementIDPattern.MatchString(strings.TrimSpace(v))
}

func firstRequirementID(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if m := requirementIDPattern.FindString(v); m != "" {
			return m
		}
	}
	return ""
}

// cmdContextDiscoverer is the production ContextDiscoverer: it EXECUTES the
// discovery probes server-side through the probes package (concurrent,
// per-probe timeout, per-probe typed degradation) and converts the outcomes
// into reviewable candidates. The agent supplies concepts and judgment; the
// code runs the probes (contract decision D5).
type cmdContextDiscoverer struct {
	run           CommandRunner
	timeout       time.Duration
	maxConcurrent int
}

// NewCommandContextDiscoverer wires the production ContextDiscoverer over the
// given CommandRunner with the default per-probe timeout. A nil runner
// degrades every probe honestly.
func NewCommandContextDiscoverer(run CommandRunner) ContextDiscoverer {
	return NewCommandContextDiscovererWithTimeout(run, probes.DefaultProbeTimeout)
}

// NewCommandContextDiscovererWithTimeout wires the production ContextDiscoverer
// with an explicit per-probe timeout (D5: default 20s, configurable).
func NewCommandContextDiscovererWithTimeout(run CommandRunner, timeout time.Duration) ContextDiscoverer {
	return NewCommandContextDiscovererWithOptions(run, timeout, 0)
}

func NewCommandContextDiscovererWithOptions(run CommandRunner, timeout time.Duration, maxConcurrent int) ContextDiscoverer {
	return cmdContextDiscoverer{run: run, timeout: timeout, maxConcurrent: maxConcurrent}
}

func (d cmdContextDiscoverer) DiscoverContext(ctx context.Context, title string, concepts []string, complexity string) (ContextDiscoveryResult, error) {
	cleaned := make([]string, 0, len(concepts))
	for _, concept := range concepts {
		if concept = strings.TrimSpace(concept); concept != "" {
			cleaned = append(cleaned, concept)
		}
	}
	if len(cleaned) == 0 {
		if title = strings.TrimSpace(title); title == "" {
			return ContextDiscoveryResult{}, fmt.Errorf("no non-empty context concepts supplied")
		}
		cleaned = []string{title}
	}
	outcomes := probes.Discover(ctx, probes.Runner(d.run), cleaned, complexity, probes.Options{Timeout: d.timeout, MaxConcurrent: d.maxConcurrent})
	return candidatesFromProbeOutcomes(outcomes), nil
}

// candidatesFromProbeOutcomes converts probe outcomes to reviewable candidates
// plus batch-level probe notes: deterministic merge, deduplicated by (kind,
// target-or-command) keeping the best-scored occurrence, provenance preserved,
// and degraded probes surfaced as metadata instead of work items.
func candidatesFromProbeOutcomes(outcomes []probes.Outcome) ContextDiscoveryResult {
	var out []ContextCandidate
	var notes []ProbeNote
	bestByKey := map[string]int{}
	for _, outcome := range outcomes {
		if outcome.Degraded {
			notes = append(notes, ProbeNote{
				Probe:    strings.TrimSpace(outcome.Probe),
				Concept:  strings.TrimSpace(outcome.Concept),
				Degraded: true,
				Detail:   strings.TrimSpace(outcome.Detail),
			})
			continue
		}
		for _, discovered := range outcome.Items {
			key := contextDedupeKey(discovered.Item)
			candidate := ContextCandidate{
				ID:        uuid.NewString(),
				Item:      discovered.Item,
				Concept:   outcome.Concept,
				Source:    outcome.Probe,
				Score:     discovered.Score,
				Origin:    discovered.Origin,
				SizeChars: discovered.SizeChars,
				Tags:      append([]string(nil), discovered.Tags...),
				Title:     discovered.Title,
				Snippet:   discovered.Snippet,
				Corroboration: []ProbeHit{{
					Probe:   outcome.Probe,
					Concept: outcome.Concept,
					Score:   discovered.Score,
				}},
				Detail: fmt.Sprintf("%s score %.3f", outcome.Probe, discovered.Score),
				Status: ContextCandidatePending,
			}
			candidate.Item.ID = uuid.NewString()
			if pos, seen := bestByKey[key]; seen {
				out[pos] = mergeContextCandidate(out[pos], candidate)
				continue
			}
			bestByKey[key] = len(out)
			out = append(out, candidate)
		}
	}
	return ContextDiscoveryResult{Candidates: out, ProbeNotes: notes}
}

// contextDedupeKey identifies one discovered setup target across probes.
func contextDedupeKey(item planmodel.RelevantContextItem) string {
	target := strings.ToLower(strings.TrimSpace(item.Target))
	if target == "" {
		target = strings.ToLower(strings.TrimSpace(item.Command))
	}
	return string(item.Kind) + "\x00" + target
}

func mergeContextCandidate(existing ContextCandidate, incoming ContextCandidate) ContextCandidate {
	existing.Corroboration = append(existing.Corroboration, incoming.Corroboration...)
	if incoming.Score > existing.Score {
		incoming.Corroboration = existing.Corroboration
		incoming.Detail = incoming.Detail + "; also: " + existing.Detail
		return incoming
	}
	existing.Detail += "; also: " + incoming.Detail
	return existing
}

// SkillSuggestion is one steered skill candidate from the applicability
// resolver: a bare skill slug plus the concrete reason it applies.
type SkillSuggestion struct {
	Slug   string
	Reason string
}

// seam: SkillApplicabilityResolver proposes skills that APPLY to a plan's
// change boundary (its blast radius) rather than merely matching its text —
// e.g. "this plan touches a React UI, load react-coherence". It is invoked at
// the global skill checkpoint alongside probe discovery; suggestions flow
// through the same accept/reject disposition as any discovered candidate.
//
// Production currently wires NoopSkillApplicabilityResolver: the boundary→skill
// applicability sources this seam is designed for are still landing elsewhere.
// Future wiring (contract decision D7):
//   - health scenarios' self-declared applicability rules (each health scenario
//     declares which surfaces its steer skill applies to), and
//   - the maturity system's phase→skill links.
//
// Whatever the source, it must stay DATA-DRIVEN — a hard-coded path→skill table
// in this repo is prohibited; that is why the seam ships no-op instead of a
// heuristic.
type SkillApplicabilityResolver interface {
	// SuggestSkills returns steered skill suggestions for the plan's change
	// boundary. Failures degrade to no suggestions; never block the wizard.
	SuggestSkills(ctx context.Context, boundary planmodel.ChangeBoundary) []SkillSuggestion
}

// NoopSkillApplicabilityResolver is the production placeholder for the D7 seam.
type NoopSkillApplicabilityResolver struct{}

func (NoopSkillApplicabilityResolver) SuggestSkills(context.Context, planmodel.ChangeBoundary) []SkillSuggestion {
	return nil
}

// resolverOrNoop keeps the service's resolver non-nil so call sites never
// nil-check the seam.
func resolverOrNoop(r SkillApplicabilityResolver) SkillApplicabilityResolver {
	if r == nil {
		return NoopSkillApplicabilityResolver{}
	}
	return r
}

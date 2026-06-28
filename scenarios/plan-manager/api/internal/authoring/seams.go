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
	// DeriveAnchorIntent returns the structured regression-anchor intent block
	// for the given plan title/slug. It always succeeds — intent is cheap,
	// deterministic, and never stale (no snapshot, no dependency).
	DeriveAnchorIntent(ctx context.Context, title, slug string) string
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
	DiscoverContext(ctx context.Context, title string, concepts []string, complexity string) ([]ContextCandidate, error)
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

func (defaultAnchorIntentDeriver) DeriveAnchorIntent(_ context.Context, title, slug string) string {
	return RegressionAnchorIntentTemplate(title, slug)
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

type cmdContextDiscoverer struct{ run CommandRunner }

func NewCommandContextDiscoverer(run CommandRunner) ContextDiscoverer {
	return cmdContextDiscoverer{run: run}
}

func (d cmdContextDiscoverer) DiscoverContext(ctx context.Context, title string, concepts []string, complexity string) ([]ContextCandidate, error) {
	if len(concepts) == 0 {
		concepts = []string{title}
	}
	var out []ContextCandidate
	for _, concept := range concepts {
		concept = strings.TrimSpace(concept)
		if concept == "" {
			continue
		}
		out = append(out,
			d.commandCandidate(ctx, concept, "prompt-manager-skill-discovery", planmodel.RelevantContextSkill, promptManagerSkillDiscoveryArgv(concept, complexity), "Discover and load relevant prompt-manager skills before implementation."),
			d.commandCandidate(ctx, concept, "prompt-manager-actions", planmodel.RelevantContextCommand, []string{"prompt-manager", "discover", concept, "--type", "all"}, "Find executable actions and operational tools before hand-rolling steps."),
			d.commandCandidate(ctx, concept, "search-hub-recall", planmodel.RelevantContextSearch, []string{"search-hub", "query", concept, "--type", "record,skill,doc"}, "Recall prior records, skills, and docs connected to this concept."),
			d.commandCandidate(ctx, concept, "cli-health-search", planmodel.RelevantContextSearch, []string{"cli-health", "search", concept}, "Find current CLI commands related to this concept."),
		)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no non-empty context concepts supplied")
	}
	return out, nil
}

func (d cmdContextDiscoverer) commandCandidate(ctx context.Context, concept, source string, kind planmodel.RelevantContextKind, argv []string, reason string) ContextCandidate {
	item := planmodel.RelevantContextItem{
		ID:           uuid.NewString(),
		Kind:         kind,
		Scope:        planmodel.RelevantContextScopeGlobal,
		Label:        source + ": " + concept,
		Reason:       reason,
		Instruction:  "Run this during context setup; inspect the output and keep only relevant findings.",
		Command:      shellJoin(argv),
		Argv:         append([]string(nil), argv...),
		Required:     true,
		RepeatPolicy: planmodel.RelevantContextOncePerExecution,
		Source:       planmodel.RelevantContextSourceDiscovered,
		Status:       planmodel.RelevantContextStatusReady,
	}
	candidate := ContextCandidate{
		ID:      uuid.NewString(),
		Item:    item,
		Concept: concept,
		Source:  source,
		Status:  ContextCandidatePending,
	}
	if d.run == nil {
		candidate.Degraded = true
		candidate.Detail = source + " unavailable"
		candidate.Item.Status = planmodel.RelevantContextStatusDegraded
		return candidate
	}
	if _, err := d.run(ctx, argv[0], argv[1:]...); err != nil {
		candidate.Degraded = true
		candidate.Detail = err.Error()
		candidate.Item.Status = planmodel.RelevantContextStatusDegraded
	}
	return candidate
}

func promptManagerSkillDiscoveryArgv(concept, complexity string) []string {
	argv := []string{"prompt-manager", "discover", concept}
	if strings.TrimSpace(complexity) != "" {
		argv = append(argv, "--complexity", strings.TrimSpace(complexity))
	}
	return argv
}

func shellJoin(argv []string) string {
	parts := make([]string, 0, len(argv))
	for _, arg := range argv {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func shellQuote(arg string) string {
	if arg == "" {
		return "''"
	}
	if strings.IndexFunc(arg, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\'' || r == '"' || r == '<' || r == '>' || r == '[' || r == ']' || r == ':' || r == ';' || r == '|' || r == '&' || r == ','
	}) < 0 {
		return arg
	}
	return "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
}

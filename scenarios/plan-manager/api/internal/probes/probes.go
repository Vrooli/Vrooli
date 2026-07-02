// Package probes executes the external context-discovery probes (prompt-manager
// skill/action discovery, search-hub recall) server-side: concurrently, with a
// per-probe timeout, degrading per probe with a typed detail — the step always
// returns (contract decision D5). Synthesis stays deterministic: parse, type,
// and score-passthrough only; no model call and no fabricated results.
//
// The package is consumed by the authoring domain's ContextDiscoverer seam;
// tests substitute the Runner with fakes (fixture JSON, hanging, exit-1,
// garbage output) so no live dependency is exercised in unit tests.
package probes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	planmodel "plan-manager/internal/planmodel"
)

// Runner is the exec seam the probe runner dispatches through. It matches the
// authoring CommandRunner contract (LookPath-guarded, error on failure).
type Runner func(ctx context.Context, name string, args ...string) ([]byte, error)

// DefaultProbeTimeout bounds one probe invocation. search-hub has been observed
// systemically degraded (p95 ~20s), so the default matches that envelope; a
// slower probe degrades honestly instead of blocking the wizard.
const DefaultProbeTimeout = 20 * time.Second

// Probe identifiers (provenance tags on every discovered item).
const (
	ProbePromptManagerSkills  = "prompt-manager-skills"
	ProbePromptManagerActions = "prompt-manager-actions"
	ProbeSearchHubRecall      = "search-hub-recall"
)

// Options configures a Discover run.
type Options struct {
	// Timeout bounds each probe individually; zero means DefaultProbeTimeout.
	Timeout time.Duration
}

// Item is one typed setup candidate discovered by a probe, with its source
// score passed through untouched.
type Item struct {
	Item  planmodel.RelevantContextItem
	Score float64
}

// Outcome is the result of one (probe, concept) execution. Degraded outcomes
// carry the honest reason and no items — never fabricated candidates.
type Outcome struct {
	Probe    string
	Concept  string
	Items    []Item
	Degraded bool
	Detail   string
}

// probeSpec is one executable probe: the argv to run and the parser for its
// JSON output.
type probeSpec struct {
	name  string
	argv  []string
	parse func([]byte) ([]Item, error)
}

func probeSpecsForConcept(concept, complexity string) []probeSpec {
	skillArgv := []string{"prompt-manager", "discover", concept, "--type", "skill", "--json"}
	if strings.TrimSpace(complexity) != "" {
		skillArgv = append(skillArgv, "--complexity", strings.TrimSpace(complexity))
	}
	return []probeSpec{
		{name: ProbePromptManagerSkills, argv: skillArgv, parse: parsePromptManagerSkills},
		{name: ProbePromptManagerActions, argv: []string{"prompt-manager", "discover", concept, "--type", "all", "--json"}, parse: parsePromptManagerActions},
		{name: ProbeSearchHubRecall, argv: []string{"search-hub", "query", concept, "--type", "record,skill,doc", "--json"}, parse: parseSearchHubRecall},
	}
}

// Discover runs every (concept, probe) pair concurrently, each bounded by the
// per-probe timeout and cancellable through ctx. It always returns one Outcome
// per pair, in deterministic (concept-major, probe-minor) order.
func Discover(ctx context.Context, run Runner, concepts []string, complexity string, opts Options) []Outcome {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultProbeTimeout
	}
	type slot struct {
		index   int
		concept string
		spec    probeSpec
	}
	var slots []slot
	for _, concept := range concepts {
		concept = strings.TrimSpace(concept)
		if concept == "" {
			continue
		}
		for _, spec := range probeSpecsForConcept(concept, complexity) {
			slots = append(slots, slot{index: len(slots), concept: concept, spec: spec})
		}
	}
	out := make([]Outcome, len(slots))
	var wg sync.WaitGroup
	for _, s := range slots {
		wg.Add(1)
		go func(s slot) {
			defer wg.Done()
			out[s.index] = runProbe(ctx, run, s.concept, s.spec, timeout)
		}(s)
	}
	wg.Wait()
	return out
}

func runProbe(ctx context.Context, run Runner, concept string, spec probeSpec, timeout time.Duration) Outcome {
	outcome := Outcome{Probe: spec.name, Concept: concept}
	if run == nil {
		outcome.Degraded = true
		outcome.Detail = "no command runner configured"
		return outcome
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	raw, err := run(probeCtx, spec.argv[0], spec.argv[1:]...)
	if err != nil {
		outcome.Degraded = true
		outcome.Detail = degradationDetail(probeCtx, err, timeout)
		return outcome
	}
	items, err := spec.parse(raw)
	if err != nil {
		outcome.Degraded = true
		outcome.Detail = "unparseable output: " + err.Error()
		return outcome
	}
	outcome.Items = items
	return outcome
}

func degradationDetail(ctx context.Context, err error, timeout time.Duration) string {
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Sprintf("probe timed out after %s", timeout)
	}
	return err.Error()
}

// --- prompt-manager parsers ---

// promptManagerDiscoverResponse is the subset of `prompt-manager discover
// --json` this package reads. The shape is an external contract pinned by the
// testdata contract fixtures.
type promptManagerDiscoverResponse struct {
	Results []promptManagerResult `json:"results"`
}

type promptManagerResult struct {
	Type        string  `json:"type"`
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Score       float64 `json:"score"`
	// ShowCommand is prompt-manager's own runnable inspect command for an
	// action result; carried verbatim (never assembled here).
	ShowCommand string `json:"showCommand"`
}

func parsePromptManagerSkills(raw []byte) ([]Item, error) {
	var resp promptManagerDiscoverResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	var out []Item
	for _, r := range resp.Results {
		if r.Type != "" && r.Type != "skill" {
			continue
		}
		if item, ok := skillItem(r.ID, r.Description); ok {
			out = append(out, Item{Item: item, Score: r.Score})
		}
	}
	return out, nil
}

func parsePromptManagerActions(raw []byte) ([]Item, error) {
	var resp promptManagerDiscoverResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	var out []Item
	for _, r := range resp.Results {
		if r.Type != "action" || strings.TrimSpace(r.ShowCommand) == "" || strings.TrimSpace(r.ID) == "" {
			continue
		}
		command := strings.TrimSpace(r.ShowCommand)
		out = append(out, Item{
			Item: planmodel.RelevantContextItem{
				Kind:        planmodel.RelevantContextCommand,
				Scope:       planmodel.RelevantContextScopeGlobal,
				Label:       strings.TrimSpace(firstNonEmpty(r.Name, r.ID)),
				Reason:      strings.TrimSpace(r.Description),
				Instruction: "Inspect this executable action before hand-rolling the step.",
				Command:     command,
				Argv:        strings.Fields(command),
				Source:      planmodel.RelevantContextSourceDiscovered,
				Status:      planmodel.RelevantContextStatusReady,
			},
			Score: r.Score,
		})
	}
	return out, nil
}

// skillItem builds the typed skill candidate: Target carries ONLY the bare
// slug; the runnable command is derived by the single idempotent renderer
// function (contract decision D6) — never assembled here.
func skillItem(slug, description string) (planmodel.RelevantContextItem, bool) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return planmodel.RelevantContextItem{}, false
	}
	return planmodel.RelevantContextItem{
		Kind:        planmodel.RelevantContextSkill,
		Scope:       planmodel.RelevantContextScopeGlobal,
		Label:       slug,
		Reason:      strings.TrimSpace(description),
		Instruction: "Load this internal skill before implementation.",
		Target:      slug,
		Source:      planmodel.RelevantContextSourceDiscovered,
		Status:      planmodel.RelevantContextStatusReady,
	}, true
}

// --- search-hub parser ---

// searchHubResponse is the subset of `search-hub query --json` this package
// reads. The CLI emits snake_case field names; camelCase aliases are accepted
// for tolerance against protojson-encoded variants.
type searchHubResponse struct {
	Ranked []searchHubHit `json:"ranked"`
	Groups []struct {
		Hits []searchHubHit `json:"hits"`
	} `json:"groups"`
}

type searchHubHit struct {
	Type             string  `json:"type"`
	ID               string  `json:"id"`
	Title            string  `json:"title"`
	Snippet          string  `json:"snippet"`
	Path             string  `json:"path"`
	Score            float64 `json:"score"`
	RerankScore      float64 `json:"rerank_score"`
	RerankScoreCamel float64 `json:"rerankScore"`
}

func (h searchHubHit) bestScore() float64 {
	for _, s := range []float64{h.RerankScore, h.RerankScoreCamel, h.Score} {
		if s != 0 {
			return s
		}
	}
	return 0
}

func parseSearchHubRecall(raw []byte) ([]Item, error) {
	var resp searchHubResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	hits := resp.Ranked
	if len(hits) == 0 {
		for _, g := range resp.Groups {
			hits = append(hits, g.Hits...)
		}
	}
	var out []Item
	for _, hit := range hits {
		item, ok := searchHubItem(hit)
		if !ok {
			continue
		}
		out = append(out, Item{Item: item, Score: hit.bestScore()})
	}
	return out, nil
}

func searchHubItem(hit searchHubHit) (planmodel.RelevantContextItem, bool) {
	switch strings.ToLower(strings.TrimSpace(hit.Type)) {
	case "skill":
		return skillItem(firstNonEmpty(hit.ID, hit.Path), hit.Snippet)
	case "doc":
		target := strings.TrimSpace(firstNonEmpty(hit.Path, hit.ID))
		if target == "" {
			return planmodel.RelevantContextItem{}, false
		}
		return planmodel.RelevantContextItem{
			Kind:        planmodel.RelevantContextDoc,
			Scope:       planmodel.RelevantContextScopeGlobal,
			Label:       target,
			Reason:      strings.TrimSpace(firstNonEmpty(hit.Snippet, hit.Title)),
			Instruction: "Read this document before implementation.",
			Target:      target,
			Source:      planmodel.RelevantContextSourceDiscovered,
			Status:      planmodel.RelevantContextStatusReady,
		}, true
	case "record":
		id := strings.TrimSpace(hit.ID)
		if id == "" {
			return planmodel.RelevantContextItem{}, false
		}
		command := "swarm-manager records get --id " + id
		return planmodel.RelevantContextItem{
			Kind:        planmodel.RelevantContextCommand,
			Scope:       planmodel.RelevantContextScopeGlobal,
			Label:       strings.TrimSpace(firstNonEmpty(hit.Title, id)),
			Reason:      strings.TrimSpace(firstNonEmpty(hit.Snippet, "Prior completed-work record connected to this concept.")),
			Instruction: "Recall this prior-work record before implementation.",
			Command:     command,
			Argv:        strings.Fields(command),
			Source:      planmodel.RelevantContextSourceDiscovered,
			Status:      planmodel.RelevantContextStatusReady,
		}, true
	default:
		return planmodel.RelevantContextItem{}, false
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

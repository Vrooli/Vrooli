package eval

import (
	"context"
	"strings"

	"search-hub/internal/corpusgen"
	internaleval "search-hub/internal/eval"

	aisearch "github.com/vrooli/ai-go/search"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

// CorpusGenerator is the generation seam the connect handler depends on: given a
// suite + its provider descriptor it returns de-duped proposed cases. The
// production impl (liveCorpusGenerator) builds a per-request index sampler over
// the provider's own search endpoint; handler tests inject a fake.
type CorpusGenerator interface {
	Generate(ctx context.Context, suite *evalv1.EvalSuite, desc *registryv1.ProviderDescriptor, opts corpusgen.Options) (*corpusgen.Result, error)
}

// defaultProbeLimit caps how many hits each probe query pulls when sampling the
// index. A dozen probes × this is plenty of distinct items to invert without
// hammering the provider.
const defaultProbeLimit = 10

// liveCorpusGenerator is the production CorpusGenerator. It composes the shared
// corpusgen core with: a sampler that probes the provider's live search endpoint
// (the only enumeration the search contract affords — there is no list-all RPC),
// the Ollama-backed query inverter, and the lexical de-duper.
type liveCorpusGenerator struct {
	client   internaleval.ProviderClient
	inverter corpusgen.Inverter
	deduper  corpusgen.Deduper
	perProbe int32
}

func newLiveCorpusGenerator(client internaleval.ProviderClient) liveCorpusGenerator {
	return liveCorpusGenerator{
		client:   client,
		inverter: corpusgen.NewOllamaInverter(),
		deduper:  corpusgen.JaccardDeduper{},
		perProbe: defaultProbeLimit,
	}
}

func (g liveCorpusGenerator) Generate(ctx context.Context, suite *evalv1.EvalSuite, desc *registryv1.ProviderDescriptor, opts corpusgen.Options) (*corpusgen.Result, error) {
	sampler := &corpusSampler{
		client:   g.client,
		desc:     desc,
		probes:   probeQueries(suite, desc),
		perProbe: g.perProbe,
	}
	return corpusgen.New(corpusgen.Deps{
		Sampler:   sampler,
		Inverter:  g.inverter,
		Deduper:   g.deduper,
		Validator: liveReferentialValidator{client: g.client, desc: desc},
	}, opts).Generate(ctx, suite)
}

type liveReferentialValidator struct {
	client internaleval.ProviderClient
	desc   *registryv1.ProviderDescriptor
}

func (v liveReferentialValidator) ValidPositive(ctx context.Context, it corpusgen.Item, query string, topK int) (bool, error) {
	policy := aisearch.DefaultScoringPolicy
	if topK > 0 {
		policy.GateK = topK
	}
	if policy.DeepK < policy.GateK {
		policy.DeepK = policy.GateK
	}
	hits, err := v.client.Search(ctx, v.desc, query, int32(policy.DeepK), internaleval.SearchCallOptions{})
	if err != nil {
		return false, err
	}
	tc := aisearch.TestCase{
		ID:               positiveCaseIDForQuery(query),
		Query:            query,
		ExpectIDs:        []string{it.ID},
		ExpectWithinTopK: topK,
	}
	ref := aisearch.ClassifyExpectID(searchHitsToResults(hits), tc, policy)
	return ref == aisearch.ReferentialLive || ref == aisearch.ReferentialHard, nil
}

func positiveCaseIDForQuery(query string) string {
	return "gen-validation:" + strings.ToLower(strings.Join(strings.Fields(query), " "))
}

func searchHitsToResults(hits []*routingv1.SearchHit) []aisearch.SearchResult {
	out := make([]aisearch.SearchResult, 0, len(hits))
	for _, h := range hits {
		out = append(out, aisearch.SearchResult{ID: h.GetId(), Score: h.GetScore()})
	}
	return out
}

// corpusSampler discovers index items by issuing PROBE queries against the
// provider's registered search endpoint and collecting the distinct hits. This
// is the only enumeration the unified search contract affords — there is no
// dump/list-all RPC — so the sample is exactly what the probes surface. It
// therefore cannot see items no probe reaches; that is a documented live
// limitation (Phase 8 widens the probe set), not a silent cap — Result.Sampled
// reports the true count.
type corpusSampler struct {
	client   internaleval.ProviderClient
	desc     *registryv1.ProviderDescriptor
	probes   []string
	perProbe int32
}

func (s *corpusSampler) Sample(ctx context.Context, target int) ([]corpusgen.Item, error) {
	seen := map[string]bool{}
	var out []corpusgen.Item
	for _, p := range s.probes {
		if len(out) >= target {
			break
		}
		hits, err := s.client.Search(ctx, s.desc, p, s.perProbe, internaleval.SearchCallOptions{})
		if err != nil {
			continue // a probe that errors is skipped — sampling is best-effort
		}
		for _, h := range hits {
			id := strings.TrimSpace(h.GetId())
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			out = append(out, corpusgen.Item{
				ID:      id,
				Title:   h.GetTitle(),
				Snippet: h.GetSnippet(),
				Type:    s.desc.GetType(),
				Group:   s.desc.GetProviderGroup(),
			})
			if len(out) >= target {
				break
			}
		}
	}
	return out, nil
}

// probeQueries builds the probe set for sampling: the suite's existing case
// queries (representative of the corpus the curator already cares about) plus
// content words harvested from the descriptor's natural-language description,
// falling back to the type/group when a suite has no cases yet. De-duped,
// case-insensitively, preserving first-seen order.
func probeQueries(suite *evalv1.EvalSuite, desc *registryv1.ProviderDescriptor) []string {
	seen := map[string]bool{}
	var out []string
	add := func(q string) {
		q = strings.TrimSpace(q)
		key := strings.ToLower(q)
		if q != "" && !seen[key] {
			seen[key] = true
			out = append(out, q)
		}
	}
	for _, c := range suite.GetCases() {
		add(c.GetQuery())
	}
	for _, w := range descWords(desc.GetDescription()) {
		add(w)
	}
	if len(out) == 0 {
		add(desc.GetType())
		add(desc.GetProviderGroup())
	}
	return out
}

// descWords returns the distinct content words (length ≥ 4, lowercased) from a
// description — broad single-term probes that surface items a curated query set
// might miss. Short/stop-ish words are dropped by the length floor.
func descWords(desc string) []string {
	seen := map[string]bool{}
	var out []string
	for _, raw := range strings.FieldsFunc(desc, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9')
	}) {
		w := strings.ToLower(raw)
		if len(w) >= 4 && !seen[w] {
			seen[w] = true
			out = append(out, w)
		}
	}
	return out
}

package corpusgen

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

// --- fakes ------------------------------------------------------------------

type fakeSampler struct {
	items []Item
	err   error
}

func (f fakeSampler) Sample(ctx context.Context, target int) ([]Item, error) {
	return f.items, f.err
}

// fakeInverter maps each item id to a deterministic query so tests assert exact
// shapes. A missing id yields an empty query (a "failed" inversion).
type fakeInverter struct {
	pos map[string]string
	neg map[string]string
}

func (f fakeInverter) InvertPositive(ctx context.Context, it Item) (string, error) {
	return f.pos[it.ID], nil
}

func (f fakeInverter) InvertNegative(ctx context.Context, it Item) (string, error) {
	if f.neg == nil {
		return "no such thing as " + it.ID, nil
	}
	return f.neg[it.ID], nil
}

func items(n int, typ string) []Item {
	out := make([]Item, n)
	for i := 0; i < n; i++ {
		out[i] = Item{ID: fmt.Sprintf("cmd-%d", i), Title: fmt.Sprintf("Title %d", i), Type: typ}
	}
	return out
}

func invMap(n int) map[string]string {
	m := map[string]string{}
	for i := 0; i < n; i++ {
		m[fmt.Sprintf("cmd-%d", i)] = fmt.Sprintf("how do i do thing %d", i)
	}
	return m
}

func suiteWith(queries ...string) *evalv1.EvalSuite {
	s := &evalv1.EvalSuite{SuiteId: "s", ProviderId: "p"}
	for i, q := range queries {
		s.Cases = append(s.Cases, &evalv1.EvalCase{CaseId: fmt.Sprintf("c%d", i), Query: q, ExpectIds: []string{"x"}})
	}
	return s
}

// --- tests ------------------------------------------------------------------

func TestGenerate_PositivesAreMarkedAndAnchored(t *testing.T) {
	g := New(Deps{
		Sampler:  fakeSampler{items: items(5, "command")},
		Inverter: fakeInverter{pos: invMap(5)},
		Deduper:  JaccardDeduper{},
	}, Options{Count: 5})

	res, err := g.Generate(context.Background(), suiteWith())
	require.NoError(t, err)
	require.Len(t, res.Proposed, 5)
	for i, p := range res.Proposed {
		c := p.Case
		require.Contains(t, c.GetTags(), "generated", "every proposal carries the generated marker")
		require.Contains(t, c.GetTags(), "type:command", "the stratum is tagged for coverage")
		require.Equal(t, []string{fmt.Sprintf("cmd-%d", i)}, c.GetExpectIds(), "positive case anchors to its source item")
		require.Equal(t, int32(DefaultTopK), c.GetExpectWithinTopK())
		require.Equal(t, fmt.Sprintf("cmd-%d", i), p.SourceID)
		require.True(t, strings.HasPrefix(c.GetCaseId(), "gen-"))
	}
}

func TestGenerate_RespectsCount(t *testing.T) {
	g := New(Deps{
		Sampler:  fakeSampler{items: items(20, "command")},
		Inverter: fakeInverter{pos: invMap(20)},
		Deduper:  JaccardDeduper{},
	}, Options{Count: 6})
	res, err := g.Generate(context.Background(), suiteWith())
	require.NoError(t, err)
	require.Len(t, res.Proposed, 6, "stops at the target count even with more items available")
}

func TestGenerate_DedupesAgainstExistingCorpusAndItself(t *testing.T) {
	// Two items invert to the same query; and one matches an existing case.
	inv := fakeInverter{pos: map[string]string{
		"cmd-0": "restart the api service",
		"cmd-1": "restart the api service", // dup of cmd-0
		"cmd-2": "list all running scenarios",
	}}
	g := New(Deps{
		Sampler:  fakeSampler{items: items(3, "command")},
		Inverter: inv,
		Deduper:  JaccardDeduper{},
	}, Options{Count: 5})

	res, err := g.Generate(context.Background(), suiteWith("list all running scenarios"))
	require.NoError(t, err)
	// cmd-0 accepted; cmd-1 dropped (self-dup); cmd-2 dropped (existing corpus).
	require.Len(t, res.Proposed, 1)
	require.Equal(t, "restart the api service", res.Proposed[0].Case.GetQuery())
	require.Equal(t, 2, res.Deduped)
	require.Equal(t, 3, res.Inverted)
}

func TestGenerate_FailedInversionIsSkippedNotFatal(t *testing.T) {
	inv := fakeInverter{pos: map[string]string{
		"cmd-0": "", // failed/empty inversion
		"cmd-1": "how to do thing one",
	}}
	g := New(Deps{
		Sampler:  fakeSampler{items: items(2, "command")},
		Inverter: inv,
		Deduper:  JaccardDeduper{},
	}, Options{Count: 5})
	res, err := g.Generate(context.Background(), suiteWith())
	require.NoError(t, err)
	require.Len(t, res.Proposed, 1)
	require.Equal(t, "how to do thing one", res.Proposed[0].Case.GetQuery())
}

func TestGenerate_NegativesWhenRequested(t *testing.T) {
	g := New(Deps{
		Sampler: fakeSampler{items: items(8, "command")},
		Inverter: fakeInverter{
			pos: invMap(8),
			neg: map[string]string{"cmd-0": "unsupported quantum teleport command", "cmd-1": "make me a sandwich please now"},
		},
		Deduper: JaccardDeduper{},
	}, Options{Count: 8, Negatives: true, NegativeRatio: 0.25})

	res, err := g.Generate(context.Background(), suiteWith())
	require.NoError(t, err)

	var pos, neg []Proposal
	for _, p := range res.Proposed {
		if p.Case.GetExpectNoStrongHit() {
			neg = append(neg, p)
		} else {
			pos = append(pos, p)
		}
	}
	require.Len(t, pos, 8)
	// ceil(8 * 0.25) = 2 negatives requested; both invert cleanly.
	require.Len(t, neg, 2)
	for _, p := range neg {
		require.True(t, p.Case.GetExpectNoStrongHit())
		require.Equal(t, DefaultGibberishCeiling, p.Case.GetExpectMaxScore())
		require.Contains(t, p.Case.GetTags(), "gibberish")
		require.Contains(t, p.Case.GetTags(), "generated")
		require.Empty(t, p.SourceID, "a negative is not anchored to one item")
	}
}

func TestGenerate_NegativesFlooredAtOne(t *testing.T) {
	g := New(Deps{
		Sampler:  fakeSampler{items: items(2, "command")},
		Inverter: fakeInverter{pos: invMap(2), neg: map[string]string{"cmd-0": "totally unrelated nonsense query"}},
		Deduper:  JaccardDeduper{},
	}, Options{Count: 2, Negatives: true, NegativeRatio: 0.01})
	res, err := g.Generate(context.Background(), suiteWith())
	require.NoError(t, err)
	negs := 0
	for _, p := range res.Proposed {
		if p.Case.GetExpectNoStrongHit() {
			negs++
		}
	}
	require.Equal(t, 1, negs, "at least one negative even at a tiny ratio")
}

func TestGenerate_StableCaseIDsAreIdempotent(t *testing.T) {
	mk := func() *Result {
		g := New(Deps{
			Sampler:  fakeSampler{items: items(3, "command")},
			Inverter: fakeInverter{pos: invMap(3)},
			Deduper:  JaccardDeduper{},
		}, Options{Count: 3})
		res, err := g.Generate(context.Background(), suiteWith())
		require.NoError(t, err)
		return res
	}
	a, b := mk(), mk()
	require.Equal(t, len(a.Proposed), len(b.Proposed))
	for i := range a.Proposed {
		require.Equal(t, a.Proposed[i].Case.GetCaseId(), b.Proposed[i].Case.GetCaseId(),
			"same query → same case_id across runs (apply is idempotent)")
	}
}

func TestGenerate_StrataReported(t *testing.T) {
	mixed := append(items(2, "command"), Item{ID: "doc-0", Title: "A doc", Type: "doc"})
	g := New(Deps{
		Sampler:  fakeSampler{items: mixed},
		Inverter: fakeInverter{pos: map[string]string{"cmd-0": "q0", "cmd-1": "q1", "doc-0": "q2"}},
		Deduper:  JaccardDeduper{},
	}, Options{Count: 5})
	res, err := g.Generate(context.Background(), suiteWith())
	require.NoError(t, err)
	require.Equal(t, []string{"type:command", "type:doc"}, res.Strata)
}

func TestGenerate_RequiresAllSeams(t *testing.T) {
	g := New(Deps{Sampler: fakeSampler{}, Inverter: fakeInverter{}}, Options{})
	_, err := g.Generate(context.Background(), suiteWith())
	require.Error(t, err, "a missing Deduper is a wiring error, surfaced not ignored")
}

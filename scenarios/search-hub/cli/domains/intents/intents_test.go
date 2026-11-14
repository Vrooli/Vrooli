package intents

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

func hit(typ, id, title string) *routingv1.SearchHit {
	return &routingv1.SearchHit{Type: typ, Id: id, Title: title}
}

// Feature: an intent passes when the owning scenario's leaf is within the top-k.
func TestGradeSeparatesQualifyingLeafFromAnyLeaf(t *testing.T) {
	c := loadedCase{intentCase: intentCase{ID: "x", Query: "q", ExpectTypes: []string{"skill"}}, Scenario: "device-control"}
	hits := []*routingv1.SearchHit{
		hit("library", "browser-automation-studio.capture-surface", "browser-automation-studio.capture-surface"),
		hit("library", "device-control.volume", "device-control.volume"),
		hit("skill", "device-control", "device-control"),
	}
	got := grade(c, hits, 5)
	require.True(t, got.Pass)
	require.Equal(t, 3, got.Rank)
	require.Equal(t, 2, got.AnyLeafRank)

	got = grade(c, hits[:2], 5)
	require.False(t, got.Pass, "a program of the scenario is not the skill the case requires")
	require.Equal(t, 2, got.AnyLeafRank, "but the scenario was visible, which the report must show")

	require.False(t, grade(c, hits, 2).Pass, "ranks beyond top-k do not count")
}

func TestOwnershipShapes(t *testing.T) {
	for name, tc := range map[string]struct {
		hit  *routingv1.SearchHit
		want bool
	}{
		"skill id":         {hit("skill", "device-control", "device-control"), true},
		"skill companion":  {hit("skill", "device-control-improve", "device-control-improve"), true},
		"program":          {hit("library", "device-control.volume", "x"), true},
		"command title":    {hit("command", "capture", "device-control device capture"), true},
		"path":             {&routingv1.SearchHit{Path: "/r/scenarios/device-control/x.json"}, true},
		"other scenario":   {hit("command", "capture", "browser-automation-studio capture"), false},
		"prefix collision": {hit("command", "x", "device-controller run"), false},
		"command dash":     {hit("command", "device-control-x", "other"), false},
	} {
		require.Equal(t, tc.want, ownedBy(tc.hit, "device-control"), name)
	}
}

func TestOrderedHitsMergesUnrerankedGroupsByScore(t *testing.T) {
	resp := &routingv1.QueryResponse{Groups: []*routingv1.ProviderResultGroup{
		{Hits: []*routingv1.SearchHit{{Id: "a", Score: 0.2}}},
		{Hits: []*routingv1.SearchHit{{Id: "b", Score: 0.9}}},
	}}
	hits := orderedHits(resp)
	require.Equal(t, "b", hits[0].GetId())
}

func writeCorpus(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "c.json")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	return path
}

func TestLoadCorpusRejectsBrokenInputInsteadOfSkipping(t *testing.T) {
	for name, body := range map[string]string{
		"unknown field":    `{"scenario":"s","cases":[{"id":"a","query":"q","bogus":1}]}`,
		"no owner":         `{"scenario":"*","cases":[{"id":"a","query":"q"}]}`,
		"no query":         `{"scenario":"s","cases":[{"id":"a"}]}`,
		"duplicate id":     `{"scenario":"s","cases":[{"id":"a","query":"q"},{"id":"a","query":"r"}]}`,
		"missing scenario": `{"cases":[{"id":"a","query":"q"}]}`,
	} {
		_, err := loadCorpus([]string{writeCorpus(t, body)})
		require.Error(t, err, name)
	}
}

func TestLoadCorpusAppliesFileDefaultsAndPerCaseOwner(t *testing.T) {
	cases, err := loadCorpus([]string{writeCorpus(t, `{"scenario":"*","expect_types":["skill"],"cases":[{"id":"a","owner":"o","query":"q"}]}`)})
	require.NoError(t, err)
	require.Equal(t, "o", cases[0].Scenario)
	require.Equal(t, []string{"skill"}, cases[0].ExpectTypes)
}

// The shipped corpora must always load: a syntax slip would otherwise silently
// remove the measurement.
func TestShippedCorporaLoad(t *testing.T) {
	root, err := resolveRoot("")
	require.NoError(t, err)
	sources, err := corpusSources(root)
	require.NoError(t, err)
	require.NotEmpty(t, sources)
	cases, err := loadCorpus(sources)
	require.NoError(t, err)
	require.NotEmpty(t, cases)
}

func TestRunCasesCountsUnavailableSeparatelyFromMisses(t *testing.T) {
	cases := []loadedCase{
		{intentCase: intentCase{ID: "ok", Query: "ok"}, Scenario: "s"},
		{intentCase: intentCase{ID: "down", Query: "down"}, Scenario: "s"},
		{intentCase: intentCase{ID: "miss", Query: "miss"}, Scenario: "s"},
	}
	query := func(_ context.Context, req *routingv1.QueryRequest) (*routingv1.QueryResponse, error) {
		switch req.GetQuery() {
		case "ok":
			return &routingv1.QueryResponse{Ranked: []*routingv1.SearchHit{hit("skill", "s", "s")}}, nil
		case "down":
			return nil, errors.New("boom")
		}
		return &routingv1.QueryResponse{}, nil
	}
	out := summarize("/r", DefaultTypes, 5, runCases(cases, DefaultTypes, 5, query))
	require.Equal(t, 2, out.Cases)
	require.Equal(t, 1, out.Passed)
	require.Equal(t, 1, out.Unavailable)
	require.InDelta(t, 0.5, out.Recall, 1e-9)
}

func TestCoverageListsScenariosWithLeavesButNoIntents(t *testing.T) {
	root := t.TempDir()
	for _, p := range []string{"scenarios/a/skills/a/SKILL.md", "scenarios/b/.vrooli/program-runtime/p.json", "scenarios/c/README.md"} {
		full := filepath.Join(root, p)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte("{}"), 0o600))
	}
	leaves, err := inventoryLeaves(root)
	require.NoError(t, err)
	require.Len(t, leaves, 2, "a scenario with no discoverable leaf is not a gap")
	attachIntents(leaves, []loadedCase{{Scenario: "a"}})
	gap := uncovered(leaves)
	require.Len(t, gap, 1)
	require.Equal(t, "b", gap[0].Scenario)
}

func TestDegradedMissIsUnavailableButPassStands(t *testing.T) {
	c := loadedCase{intentCase: intentCase{ID: "x", Query: "q"}, Scenario: "s"}
	degraded := []*routingv1.ProviderResultGroup{{ProviderId: "cli-health.commands", Degraded: true}}
	miss := func(_ context.Context, _ *routingv1.QueryRequest) (*routingv1.QueryResponse, error) {
		return &routingv1.QueryResponse{Groups: degraded}, nil
	}
	got := runCase(c, DefaultTypes, 5, miss)
	require.Contains(t, got.Unavailable, "cli-health.commands")
	require.False(t, got.Pass)

	pass := func(_ context.Context, _ *routingv1.QueryRequest) (*routingv1.QueryResponse, error) {
		return &routingv1.QueryResponse{Groups: degraded, Ranked: []*routingv1.SearchHit{hit("skill", "s", "s")}}, nil
	}
	got = runCase(c, DefaultTypes, 5, pass)
	require.Empty(t, got.Unavailable)
	require.True(t, got.Pass)
}

func TestRunStopsAfterConsecutiveDegradedAnswers(t *testing.T) {
	var cases []loadedCase
	for i := 0; i < 6; i++ {
		cases = append(cases, loadedCase{intentCase: intentCase{ID: string(rune('a' + i)), Query: "q"}, Scenario: "s"})
	}
	calls := 0
	query := func(_ context.Context, _ *routingv1.QueryRequest) (*routingv1.QueryResponse, error) {
		calls++
		return &routingv1.QueryResponse{Groups: []*routingv1.ProviderResultGroup{{ProviderId: "p", Degraded: true}}}, nil
	}
	results := runCases(cases, DefaultTypes, 5, query)
	require.Equal(t, maxConsecutiveDegraded, calls, "a failing federation must not be hammered")
	require.Contains(t, results[5].Unavailable, "skipped")
}

func TestExactLeafIDQualifiesEvenWhenNameDoesNotCarryTheScenario(t *testing.T) {
	c := loadedCase{intentCase: intentCase{ID: "x", Query: "q", ExpectIDs: []string{"sandbox"}}, Scenario: "workspace-sandbox"}
	got := grade(c, []*routingv1.SearchHit{hit("skill", "other", "other"), hit("skill", "sandbox", "sandbox")}, 5)
	require.True(t, got.Pass)
	require.Equal(t, 2, got.Rank)
}

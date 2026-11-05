package aisearch

import (
	"context"
	"testing"
	"time"

	"offer-desk/internal/catalog"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	storeNow   = time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	readNow    = time.Date(2026, time.January, 2, 12, 0, 0, 0, time.UTC)
	projectNow = func() time.Time { return readNow }
)

func testSource(t *testing.T) (*catalog.Store, *StoreSource, context.Context) {
	t.Helper()
	sqlDB := databasetest.NewSQLite(t)
	db := database.NewFromPrimary(sqlDB)
	ctx := context.Background()
	require.NoError(t, database.EnsureSchemas(ctx, sqlDB, database.SchemaProviderFunc(func() string { return (&catalog.Store{}).Schema() })))
	store := catalog.NewStore(db, func() time.Time { return storeNow })
	return store, NewStoreSource(store, projectNow), ctx
}

func TestLoadProjectsEveryCatalogRecordClass(t *testing.T) {
	store, source, ctx := testSource(t)

	kinds := []offerspb.NodeKind{
		offerspb.NodeKind_OFFER, offerspb.NodeKind_VARIANT, offerspb.NodeKind_CHANNEL,
		offerspb.NodeKind_REVENUE_LINE, offerspb.NodeKind_DELIVERABLE, offerspb.NodeKind_RAMP,
		offerspb.NodeKind_STREAM, offerspb.NodeKind_AUDIENCE,
	}
	nodes := make(map[offerspb.NodeKind]*offerspb.Node, len(kinds))
	for _, kind := range kinds {
		n, err := store.CreateNode(ctx, kind, "seed "+kind.String(), offerspb.Status_IDEA, "", "")
		require.NoError(t, err)
		nodes[kind] = n
	}
	offer, variant := nodes[offerspb.NodeKind_OFFER], nodes[offerspb.NodeKind_VARIANT]
	_, err := store.CreateEdge(ctx, &offerspb.Edge{FromId: offer.Id, ToId: variant.Id, Kind: "sells_at", IntendedPriceMinor: 4900, Currency: "USD", IntendedPriceDeclared: true})
	require.NoError(t, err)
	_, err = store.AddTrigger(ctx, &offerspb.Trigger{NodeId: offer.Id, FactName: "activation_rate", Operator: ">=", Threshold: 0.5})
	require.NoError(t, err)
	_, err = store.AddFact(ctx, &offerspb.Fact{Name: "activation_rate", Value: 0.75, Dimension: "activation", ObservedAt: timestamppb.New(storeNow), StaleAfterDays: 30})
	require.NoError(t, err)
	_, err = store.Transition(ctx, offer.Id, offerspb.Status_CANDIDATE, "operator")
	require.NoError(t, err)
	_, err = store.Evaluate(ctx, false)
	require.NoError(t, err)
	_, err = store.Proposal(ctx, offer.Id, "agent", offerspb.Status_ACTIVE, "activation met")
	require.NoError(t, err)

	snapshot, err := source.Load(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, snapshot.Generation)

	got := make(map[string]int)
	for _, r := range snapshot.Records {
		got[r.Kind]++
		require.NotEmpty(t, r.ID)
		require.NotEmpty(t, r.Revision)
		require.Equal(t, VisibilityOperator, r.Visibility)
		require.NotEmpty(t, r.FollowUp)
	}
	for _, kind := range kinds {
		require.Equalf(t, 1, got["node-"+nodeKindLabel(kind)], "node kind %s missing", kind)
	}
	require.Equal(t, 1, got[KindEdge], "edge missing")
	require.Equal(t, 1, got[KindTrigger], "trigger missing")
	require.Equal(t, 1, got[KindFact], "fact missing")
	require.GreaterOrEqual(t, got[KindEvaluation], 1, "evaluation missing")
	require.Equal(t, 1, got[KindProposal], "proposal missing")
	require.GreaterOrEqual(t, got[KindAudit], 1, "audit missing")
	require.GreaterOrEqual(t, got[KindDecisionContext], 1, "decision context missing")

	again, err := source.Load(ctx)
	require.NoError(t, err)
	require.Equal(t, snapshot.Generation, again.Generation, "unchanged corpus must keep its generation")
	require.Equal(t, snapshot.MaterializedAt, again.MaterializedAt)
	require.Equal(t, storeNow, snapshot.MaterializedAt, "materialized time is the newest catalog mutation, not the read time")
}

func TestDecisionContextAggregatesOwnerRecords(t *testing.T) {
	store, source, ctx := testSource(t)
	offer, err := store.CreateNode(ctx, offerspb.NodeKind_OFFER, "Candidate offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	_, err = store.AddTrigger(ctx, &offerspb.Trigger{NodeId: offer.Id, FactName: "activation_rate", Operator: ">=", Threshold: 0.5})
	require.NoError(t, err)
	_, err = store.AddFact(ctx, &offerspb.Fact{Name: "activation_rate", Value: 0.75, Dimension: "activation", ObservedAt: timestamppb.New(readNow.Add(-24 * time.Hour)), StaleAfterDays: 30})
	require.NoError(t, err)
	_, err = store.Transition(ctx, offer.Id, offerspb.Status_CANDIDATE, "operator")
	require.NoError(t, err)
	_, err = store.Evaluate(ctx, false)
	require.NoError(t, err)

	snapshot, err := source.Load(ctx)
	require.NoError(t, err)
	var decision, node *Record
	for i := range snapshot.Records {
		r := &snapshot.Records[i]
		switch r.ID {
		case "decision:" + offer.Id:
			decision = r
		case "node:" + offer.Id:
			node = r
		}
	}
	require.NotNil(t, decision, "a node with recorded trigger/evaluation state must project decision context")
	require.Contains(t, decision.Snippet, "Offer decision context")
	require.Contains(t, decision.Snippet, "catalog status TRIGGER_MET")
	require.Contains(t, decision.Snippet, "recorded IDEA, CANDIDATE, TRIGGER_MET")
	require.Contains(t, decision.Snippet, "fires when activation_rate >= 0.5")
	require.Contains(t, decision.Snippet, "is fresh")
	require.Contains(t, decision.Snippet, "Latest evaluation")
	require.Contains(t, decision.Snippet, "No promotion proposal")
	require.NotNil(t, node)
	require.Contains(t, node.Snippet, "Offer Desk catalog offer")
	require.Contains(t, node.Snippet, "Candidate offer")

	// The aggregate is derived from owner rows, so an unchanged catalog must
	// not move its revision or the corpus generation.
	again, err := source.Load(ctx)
	require.NoError(t, err)
	require.Equal(t, snapshot.Generation, again.Generation)
}

func TestRetiredNodeHasNoCurrentDecisionContext(t *testing.T) {
	store, source, ctx := testSource(t)
	node, err := store.CreateNode(ctx, offerspb.NodeKind_OFFER, "Retired probe", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	_, err = store.AddTrigger(ctx, &offerspb.Trigger{NodeId: node.Id, FactName: "activation_rate", Operator: ">=", Threshold: 0.5})
	require.NoError(t, err)
	_, err = store.AddFact(ctx, &offerspb.Fact{Name: "activation_rate", Value: 0.75, ObservedAt: timestamppb.New(storeNow), StaleAfterDays: 30})
	require.NoError(t, err)
	_, err = store.Transition(ctx, node.Id, offerspb.Status_RETIRED, "operator")
	require.NoError(t, err)

	snapshot, err := source.Load(ctx)
	require.NoError(t, err)
	for _, r := range snapshot.Records {
		require.NotEqual(t, "decision:"+node.Id, r.ID, "retired nodes must not project a current decision aggregate")
	}
	// The node itself stays visible.
	found := false
	for _, r := range snapshot.Records {
		if r.ID == "node:"+node.Id {
			found = true
		}
	}
	require.True(t, found)
}

func TestFactFreshnessIsExplicitAndReadTimeDoesNotBleedIn(t *testing.T) {
	store, source, ctx := testSource(t)
	_, err := store.AddFact(ctx, &offerspb.Fact{Name: "fresh_metric", Value: 1, Dimension: "activation", ObservedAt: timestamppb.New(readNow.Add(-24 * time.Hour)), StaleAfterDays: 30})
	require.NoError(t, err)
	_, err = store.AddFact(ctx, &offerspb.Fact{Name: "stale_metric", Value: 2, Dimension: "pricing", ObservedAt: timestamppb.New(readNow.Add(-365 * 24 * time.Hour)), StaleAfterDays: 30})
	require.NoError(t, err)

	snapshot, err := source.Load(ctx)
	require.NoError(t, err)
	freshness := make(map[string]string)
	for _, r := range snapshot.Records {
		if r.Kind == KindFact {
			freshness[r.Title] = r.Freshness
		}
	}
	require.Equal(t, FreshnessFresh, freshness["fresh_metric"])
	require.Equal(t, FreshnessStale, freshness["stale_metric"])
}

func TestHistoricalEvaluationsAreLabelled(t *testing.T) {
	store, source, ctx := testSource(t)
	offer, err := store.CreateNode(ctx, offerspb.NodeKind_OFFER, "Repeated eval offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	_, err = store.AddTrigger(ctx, &offerspb.Trigger{NodeId: offer.Id, FactName: "activation_rate", Operator: ">=", Threshold: 0.5})
	require.NoError(t, err)
	_, err = store.AddFact(ctx, &offerspb.Fact{Name: "activation_rate", Value: 0.75, ObservedAt: timestamppb.New(storeNow), StaleAfterDays: 30})
	require.NoError(t, err)
	for i := 0; i < 2; i++ {
		_, err = store.Transition(ctx, offer.Id, offerspb.Status_CANDIDATE, "operator")
		require.NoError(t, err)
		_, err = store.Evaluate(ctx, false)
		require.NoError(t, err)
	}

	snapshot, err := source.Load(ctx)
	require.NoError(t, err)
	current, historical := 0, 0
	for _, r := range snapshot.Records {
		if r.Kind != KindEvaluation {
			continue
		}
		if r.Historical {
			historical++
		} else {
			current++
		}
	}
	require.Equal(t, 1, current, "exactly one current evaluation per node")
	require.Equal(t, 1, historical, "the superseded evaluation is history")
}

func TestRetiredRecordsRemainVisible(t *testing.T) {
	store, source, ctx := testSource(t)
	node, err := store.CreateNode(ctx, offerspb.NodeKind_OFFER, "Retired offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	_, err = store.Transition(ctx, node.Id, offerspb.Status_RETIRED, "operator")
	require.NoError(t, err)

	snapshot, err := source.Load(ctx)
	require.NoError(t, err)
	found := false
	for _, r := range snapshot.Records {
		if r.ID == "node:"+node.Id {
			found = true
			require.Equal(t, offerspb.Status_RETIRED.String(), r.Metadata["status"])
		}
	}
	require.True(t, found, "retired records stay projected, not silently dropped")
}

func TestEmptySourceIsStableAndTruthful(t *testing.T) {
	_, source, ctx := testSource(t)
	snapshot, err := source.Load(ctx)
	require.NoError(t, err)
	require.Empty(t, snapshot.Records)
	require.Zero(t, snapshot.IndexedCount())
	require.True(t, snapshot.MaterializedAt.IsZero(), "no catalog data means no claimed materialization")

	again, err := source.Load(ctx)
	require.NoError(t, err)
	require.Equal(t, snapshot.Generation, again.Generation)
}

func TestServiceSearchIsBoundedAndRejectsGibberish(t *testing.T) {
	store, source, ctx := testSource(t)
	for _, name := range []string{"Aquila launch offer", "Aquila upsell variant", "Unrelated channel"} {
		_, err := store.CreateNode(ctx, offerspb.NodeKind_OFFER, name, offerspb.Status_IDEA, "", "")
		require.NoError(t, err)
	}
	service := NewService(source)

	response, err := service.Search(ctx, "aquila", 10)
	require.NoError(t, err)
	require.Len(t, response.Hits, 2)
	require.NotEmpty(t, response.Generation)

	bounded, err := service.Search(ctx, "aquila", 1)
	require.NoError(t, err)
	require.Len(t, bounded.Hits, 1)

	gibberish, err := service.Search(ctx, "zzzxq", 10)
	require.NoError(t, err)
	require.Empty(t, gibberish.Hits)

	status, err := service.Status(ctx)
	require.NoError(t, err)
	require.True(t, status.Available)
	require.Equal(t, 3, status.IndexedCount)
}

func TestStatusDoesNotAdvanceOnUnchangedRead(t *testing.T) {
	store, source, ctx := testSource(t)
	_, err := store.CreateNode(ctx, offerspb.NodeKind_OFFER, "Status offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	service := NewService(source)

	first, err := service.Status(ctx)
	require.NoError(t, err)
	second, err := service.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, first.Generation, second.Generation)
	require.Equal(t, first.LastIndexedAt, second.LastIndexedAt, "status reads must not fabricate freshness")
	require.False(t, first.LastIndexedAt.After(storeNow), "reported materialization cannot be later than the data")
}

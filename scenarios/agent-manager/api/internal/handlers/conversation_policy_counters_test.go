package handlers

import (
	"context"
	"testing"

	"agent-manager/internal/conversationpolicy"
	"agent-manager/internal/conversationsearch"
	"agent-manager/internal/piimask"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	coredb "github.com/vrooli/api-core/database"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	_ "modernc.org/sqlite"
)

// newStatusFixtureAdapter builds an adapter backed by a minimal in-memory
// conversation-search service so GetConversationIndexStatus runs end-to-end
// (the policy_counters block is appended on the real response path).
func newStatusFixtureAdapter(t *testing.T) *ConversationSearchAdapter {
	t.Helper()
	db, err := sqlx.Connect("sqlite", "file:handler-policy-counters?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, coredb.EnsureSchemas(context.Background(), db, coredb.SchemaProviderFunc(conversationsearch.Schema)))
	repository := conversationsearch.NewSQLiteRepository(db)
	service, err := conversationsearch.NewService(repository, repository, repository, []byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)
	return NewConversationSearchAdapter(service)
}

// Named sample values keep the fixtures' intent explicit and a rename a single
// edit rather than a scatter of bare literals.
const (
	policyHarnessExcluded = "ollama"
	policyHarnessRedacted = "grok"
	policyModelHidden     = "secret-model"
	failClosedSentinel    = piimask.UnavailableSentinel
)

// policyFixtureCounts drives the two real Policy instances and the real Masker
// through their public record paths so the resulting snapshot carries counts a
// test can assert on, rather than hand-forging the Counts/Counters structs.
// Returning the live snapshot exercises Policy.Snapshot and Masker.Snapshot the
// same way the production provider does.
type policyFixtureCounts struct {
	importExcluded int
	searchHidden   int
	redactMarked   int
	failClosed     int
}

func buildPolicySnapshot(t *testing.T, want policyFixtureCounts) PolicyCountersSnapshot {
	t.Helper()
	importPolicy, err := conversationpolicy.New([]conversationpolicy.Rule{
		{Action: conversationpolicy.ActionExclude, EnforceAt: conversationpolicy.EnforceImport, Harness: policyHarnessExcluded},
		{Action: conversationpolicy.ActionRedact, EnforceAt: conversationpolicy.EnforceImport, Harness: policyHarnessRedacted},
	})
	require.NoError(t, err)
	searchPolicy, err := conversationpolicy.New([]conversationpolicy.Rule{
		{Action: conversationpolicy.ActionExclude, EnforceAt: conversationpolicy.EnforceSearch, Model: policyModelHidden},
	})
	require.NoError(t, err)

	for i := 0; i < want.importExcluded; i++ {
		importPolicy.RecordImport(conversationpolicy.Fields{Harness: policyHarnessExcluded})
	}
	for i := 0; i < want.redactMarked; i++ {
		importPolicy.RecordImport(conversationpolicy.Fields{Harness: policyHarnessRedacted})
	}
	for i := 0; i < want.searchHidden; i++ {
		searchPolicy.RecordSearchHide(conversationpolicy.Fields{Model: policyModelHidden})
	}

	// A nil detector fails closed on every scan, so this produces real
	// fail-closed counts (content withheld, never indexed in the clear).
	masker := piimask.New(nil)
	for i := 0; i < want.failClosed; i++ {
		masked, failed := masker.MaskText(context.Background(), "contact me at a@b.com")
		require.True(t, failed)
		require.Equal(t, failClosedSentinel, masked)
	}

	return PolicyCountersSnapshot{
		ImportCounts: importPolicy.Snapshot(),
		SearchCounts: searchPolicy.Snapshot(),
		PIICounts:    masker.Snapshot(),
	}
}

func TestPolicyCountersToProtoSurfacesEveryCounter(t *testing.T) {
	t.Parallel()
	snapshot := buildPolicySnapshot(t, policyFixtureCounts{
		importExcluded: 3,
		searchHidden:   2,
		redactMarked:   4,
		failClosed:     5,
	})

	out := policyCountersToProto(snapshot)
	require.NotNil(t, out)
	require.True(t, out.GetProcessLocal(), "process-local must be labelled so a low count is not read as durable zero")
	require.Equal(t, uint64(3), out.GetExcludeAtImport())
	require.Equal(t, uint64(2), out.GetHideAtSearch())
	require.Equal(t, uint64(4), out.GetRedactMarked())
	require.Equal(t, uint64(5), out.GetPiiFailClosed(), "fail-closed count must surface (masking could not run)")
	// A nil detector fails closed before a scan is recorded, so events_scanned
	// stays zero while fail-closed counts every withheld event. (A configured
	// detector would increment events_scanned per scan.)
	require.Equal(t, uint64(0), out.GetPiiEventsScanned())
	require.Equal(t, uint64(0), out.GetPiiSpansMasked())

	byRule := out.GetByRule()
	require.NotEmpty(t, byRule, "per-rule enforcement counts must surface")
	var total uint64
	for _, count := range byRule {
		total += count
	}
	require.Equal(t, uint64(3+2+4), total, "by-rule totals must reconcile with the headline counts across both seams")
}

func TestPolicyCountersToProtoEmitsZeroBlockWhenHealthy(t *testing.T) {
	t.Parallel()
	// An empty snapshot still emits a block: an operator must be able to tell "no
	// policy activity" from "no policy observability", and pii_fail_closed=0 is a
	// reassuring signal worth showing.
	out := policyCountersToProto(PolicyCountersSnapshot{})
	require.NotNil(t, out)
	require.True(t, out.GetProcessLocal())
	require.Equal(t, uint64(0), out.GetExcludeAtImport())
	require.Equal(t, uint64(0), out.GetHideAtSearch())
	require.Equal(t, uint64(0), out.GetRedactMarked())
	require.Equal(t, uint64(0), out.GetPiiFailClosed())
	require.Empty(t, out.GetByRule())
}

func TestAdapterStatusOmitsPolicyCountersWithoutProvider(t *testing.T) {
	t.Parallel()
	// No provider wired → no block, so an absent provider is distinguishable from
	// an all-zero (but observed) policy.
	adapter := NewConversationSearchAdapter(nil)
	require.Nil(t, adapter.policyCounters)
}

func TestAdapterStatusCarriesPolicyCountersThroughHandler(t *testing.T) {
	t.Parallel()
	adapter := newStatusFixtureAdapter(t)
	adapter.WithPolicyCounters(func() PolicyCountersSnapshot {
		return buildPolicySnapshot(t, policyFixtureCounts{importExcluded: 1, searchHidden: 1, failClosed: 2})
	})

	response, err := adapter.GetConversationIndexStatus(context.Background(), &domainpb.GetConversationIndexStatusRequest{})
	require.NoError(t, err)

	counters := response.GetPolicyCounters()
	require.NotNil(t, counters, "status response must carry the policy_counters block when a provider is wired")
	require.Equal(t, uint64(1), counters.GetExcludeAtImport())
	require.Equal(t, uint64(1), counters.GetHideAtSearch())
	require.Equal(t, uint64(2), counters.GetPiiFailClosed())
	require.True(t, counters.GetProcessLocal())
}

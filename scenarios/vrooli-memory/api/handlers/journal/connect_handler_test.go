package journal

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"

	"github.com/vrooli/api-core/provenance"
	journalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/journal"
	localdb "vrooli-memory/internal/database"
	internaljournal "vrooli-memory/internal/journal"
	"vrooli-memory/internal/testutil/mocks"
)

func newHandler(t *testing.T) *connectHandler {
	t.Helper()
	db, err := apidb.Open(context.Background(), apidb.Config{Driver: apidb.DriverSQLite, DSN: "file:journal-handler?mode=memory&cache=shared"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db.Primary(), apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(internaljournal.Schema)))
	return NewConnectHandler(internaljournal.NewService(internaljournal.NewSQLiteRepository(db.Primary()), &mocks.FakeInference{ClassifyOut: "decision", EmbedOut: []float64{0.1}}), nil)
}

func TestAppendDerivesCorrelationOnlyFromVerifiedProvenance(t *testing.T) { // [REQ:VMEM-P1-002]
	h := newHandler(t)
	ctx := provenance.NewContext(context.Background(), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run-verified", WorkflowExecutionID: "workflow-verified"})
	resp, err := h.AppendEntry(ctx, connect.NewRequest(&journalv1.AppendEntryRequest{Body: "correlated", Correlation: &journalv1.Correlation{RunId: "forged"}}))
	require.NoError(t, err)
	require.Equal(t, "run-verified", resp.Msg.GetEntry().GetCorrelation().GetRunId())
	require.Equal(t, "workflow-verified", resp.Msg.GetEntry().GetCorrelation().GetWorkflowExecutionId())
	require.Equal(t, provenance.ActorAgent, resp.Msg.GetEntry().GetCorrelation().GetActorKind())

	uncorrelated, err := h.AppendEntry(context.Background(), connect.NewRequest(&journalv1.AppendEntryRequest{Body: "outside a run", Correlation: &journalv1.Correlation{RunId: "forged"}}))
	require.NoError(t, err)
	require.Empty(t, uncorrelated.Msg.GetEntry().GetCorrelation().GetRunId())
}

func TestConnectHandlerAppendGetAndList(t *testing.T) { // [REQ:VMEM-P0-002]
	h := newHandler(t)
	ctx := context.Background()
	appended, err := h.AppendEntry(ctx, connect.NewRequest(&journalv1.AppendEntryRequest{Body: "Keep immutable evidence", Kind: "observation"}))
	require.NoError(t, err)
	require.NotEmpty(t, appended.Msg.GetEntry().GetId())
	require.Equal(t, "decision", appended.Msg.GetEntry().GetFacetId())
	require.Len(t, appended.Msg.GetEntry().GetFacetTexts(), 3)

	got, err := h.GetEntry(ctx, connect.NewRequest(&journalv1.GetEntryRequest{Id: appended.Msg.GetEntry().GetId()}))
	require.NoError(t, err)
	require.Equal(t, appended.Msg.GetEntry().GetId(), got.Msg.GetEntry().GetId())

	listed, err := h.ListEntries(ctx, connect.NewRequest(&journalv1.ListEntriesRequest{Limit: 10, FacetId: "decision"}))
	require.NoError(t, err)
	require.Len(t, listed.Msg.GetEntries(), 1)
}

func TestConnectHandlerValidatesAndMapsMissingEntry(t *testing.T) {
	h := newHandler(t)
	_, err := h.AppendEntry(context.Background(), connect.NewRequest(&journalv1.AppendEntryRequest{}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	_, err = h.GetEntry(context.Background(), connect.NewRequest(&journalv1.GetEntryRequest{Id: "missing"}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestWorkRecordRequiresEveryNarrativeField(t *testing.T) { // [REQ:VMEM-P1-001]
	h := newHandler(t)
	_, err := h.AppendEntry(context.Background(), connect.NewRequest(&journalv1.AppendEntryRequest{Body: "record", Kind: "work-record", Trigger: "request", Approach: "implemented", Evidence: "tests"}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	resp, err := h.AppendEntry(context.Background(), connect.NewRequest(&journalv1.AppendEntryRequest{Body: "record", Kind: "work-record", Trigger: "request", Approach: "implemented", Evidence: "tests", Outcome: "passed"}))
	require.NoError(t, err)
	require.Contains(t, resp.Msg.GetEntry().GetBody(), "Outcome: passed")
}

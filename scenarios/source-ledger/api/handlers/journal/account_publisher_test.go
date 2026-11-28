package journal

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	heartbeat "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat"
	journal "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	"google.golang.org/protobuf/types/known/structpb"
	internaljournal "source-ledger/internal/journal"
)

type investigatorHeartbeat struct {
	t     *testing.T
	runID string
}

func (f investigatorHeartbeat) GetHeartbeat(_ context.Context, req *connect.Request[heartbeat.MemberRequest]) (*connect.Response[heartbeat.JsonResponse], error) {
	require.Equal(f.t, "scenario-qa", req.Msg.TeamId)
	require.Equal(f.t, "bug-investigator", req.Msg.AgentId)
	data, err := structpb.NewValue(map[string]any{"lastExecution": map[string]any{"runId": f.runID}})
	require.NoError(f.t, err)
	return connect.NewResponse(&heartbeat.JsonResponse{Data: data}), nil
}

func TestInvestigatorAccountPublicationRequiresOwnerRun(t *testing.T) {
	h := NewConnectHandler(nil, nil)
	_, err := h.AppendEntry(t.Context(), connect.NewRequest(&journal.AppendEntryRequest{
		Kind: "observation", Body: "Claimed fix", RoutingHint: "defect-account",
		RequestKey: "remember/scenario-qa/defect-123/2/0",
	}))
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	reader := investigatorHeartbeat{t: t, runID: "owner-run"}
	require.NoError(t, verifyInvestigatorRun(t.Context(), reader, "owner-run"))
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(verifyInvestigatorRun(t.Context(), reader, "other-run")))
}

func TestRetiredHistoryImportIsReadOnly(t *testing.T) {
	h := newHandler(t)
	ctx := t.Context()
	report := &journal.AppendEntryRequest{Kind: "legacy-bug-report", Body: "Historical observation",
		RequestKey: legacyImportPrefix + "historical-observation"}
	first, err := h.service.Append(ctx, internaljournal.Entry{Kind: report.Kind, Body: report.Body, RequestKey: report.RequestKey})
	require.NoError(t, err)
	got, err := h.GetEntry(ctx, connect.NewRequest(&journal.GetEntryRequest{RequestKey: report.RequestKey}))
	require.NoError(t, err)
	require.Equal(t, first.ID, got.Msg.GetEntry().GetId())

	// A stale deployment environment cannot reopen historical writes.
	t.Setenv("VROOLI_SOURCE_LEDGER_LEGACY_BUG_IMPORT", "true")
	for _, key := range []string{report.RequestKey, legacyImportPrefix + "new-observation"} {
		_, err := h.AppendEntry(ctx, connect.NewRequest(&journal.AppendEntryRequest{Kind: report.Kind, Body: report.Body, RequestKey: key}))
		require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	}
}

func TestImportProvenanceCannotForgeReservedRequestKey(t *testing.T) {
	h := NewConnectHandler(nil, nil)
	for _, runtime := range []string{"request", "request:legacy-scenario-qa-bugs/v1"} {
		_, err := h.AppendEntry(t.Context(), connect.NewRequest(&journal.AppendEntryRequest{
			Kind: "observation", Body: "Claimed fix", RoutingHint: "defect-account",
			ImportProvenance: &journal.ImportProvenance{Runtime: runtime, SourceLocator: "remember/scenario-qa/defect-forged", ContentHash: "/1/0"},
		}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	}
}

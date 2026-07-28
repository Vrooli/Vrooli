package triage

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	triagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/triage"
	"google.golang.org/protobuf/types/known/timestamppb"
	"signal-inbox/internal/clock"
	localdb "signal-inbox/internal/database"
	"signal-inbox/internal/signals"
	"signal-inbox/internal/testutil/db"
	internal "signal-inbox/internal/triage"
)

func newHandler(t *testing.T) *connectHandler {
	t.Helper()
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(signals.Schema),
		apidb.SchemaProviderFunc(internal.Schema),
	))
	_, err := database.ExecContext(context.Background(), "INSERT INTO signal(id,source_kind,source_identity,source_url,raw_payload_ref,extracted_content,content_hash,needs_attention,captured_at) VALUES('signal','text','signal','','','body','hash',0,?)", time.Now().UTC().Format(time.RFC3339Nano))
	require.NoError(t, err)
	return NewConnectHandler(internal.NewService(internal.NewSQLiteRepository(database), clock.System{}))
}

func TestTriageTransportPreservesDispositionAndAppendsAnnotations(t *testing.T) {
	handler := newHandler(t)
	set, err := handler.SetDisposition(context.Background(), connect.NewRequest(&triagev1.SetDispositionRequest{SignalId: "signal", State: triagev1.DispositionState_DISPOSITION_STATE_TRIAGED}))
	require.NoError(t, err)
	require.Equal(t, triagev1.DispositionState_DISPOSITION_STATE_TRIAGED, set.Msg.Disposition.State)

	annotation, err := handler.AddAnnotation(context.Background(), connect.NewRequest(&triagev1.AddAnnotationRequest{SignalId: "signal", Author: triagev1.AnnotationAuthor_ANNOTATION_AUTHOR_OPERATOR, Body: "sent to planning", Outcome: &triagev1.OutcomeLink{Kind: triagev1.OutcomeKind_OUTCOME_KIND_SCENARIO, TargetId: "signal-inbox"}}))
	require.NoError(t, err)
	require.Equal(t, "signal-inbox", annotation.Msg.Annotation.Outcome.TargetId)

	got, err := handler.GetTriage(context.Background(), connect.NewRequest(&triagev1.GetTriageRequest{SignalId: "signal"}))
	require.NoError(t, err)
	require.Len(t, got.Msg.Triage.Annotations, 1)

	_, err = handler.AddAnnotation(context.Background(), connect.NewRequest(&triagev1.AddAnnotationRequest{SignalId: "", Author: triagev1.AnnotationAuthor_ANNOTATION_AUTHOR_OPERATOR, Body: "note"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = handler.SetDisposition(context.Background(), connect.NewRequest(&triagev1.SetDispositionRequest{SignalId: "signal", State: triagev1.DispositionState_DISPOSITION_STATE_UNSPECIFIED, RevisitAt: timestamppb.New(time.Now())}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

package enrichment

import (
	"context"
	"database/sql"
	"testing"

	"document-manager/internal/gatewayreq"

	"github.com/stretchr/testify/require"
	gatewaypb "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	documentpb "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
	_ "modernc.org/sqlite"
)

type fakeBuilder struct{}

func (fakeBuilder) For(context.Context, gatewayreq.DocumentClass, gatewayreq.Options) (*gatewaypb.GatewayRequest, error) {
	return new(gatewaypb.GatewayRequest), nil
}

type fakeGateway struct{ unavailable bool }

func (g fakeGateway) Invoke(context.Context, *gatewaypb.GatewayRequest) (GatewayResult, error) {
	if g.unavailable {
		return GatewayResult{}, context.Canceled
	}
	return GatewayResult{Summary: "summary", SuggestedPrivacy: documentpb.PrivacyClass_PRIVACY_CLASS_INTERNAL, Model: "test-model", Dimension: 3, Vector: []float32{0.1, 0.2, 0.3}}, nil
}

func TestEmbeddingMetadataIsCompleteAndGatewayOutageLeavesUnenriched(t *testing.T) { // [REQ:DOC-P0-011]
	db, err := sql.Open("sqlite", "file:enrichment-test?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.ExecContext(context.Background(), Schema())
	require.NoError(t, err)
	repo := NewSQLiteRepository(db)
	service := NewService(repo, fakeBuilder{}, fakeGateway{})
	ctx := context.Background()

	record, err := service.Enrich(ctx, "doc", "text", documentpb.PrivacyClass_PRIVACY_CLASS_INTERNAL)
	require.NoError(t, err)
	require.Equal(t, "enriched", record.Status)
	embedding, err := service.Embed(ctx, "doc", "unit", "text", documentpb.PrivacyClass_PRIVACY_CLASS_INTERNAL, "semantic-query")
	require.NoError(t, err)
	require.NoError(t, embedding.Validate())
	rows, err := repo.ListEmbeddings(ctx, "doc")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, 3, rows[0].Dimension)

	outage := NewService(repo, fakeBuilder{}, fakeGateway{unavailable: true})
	degraded, err := outage.Enrich(ctx, "doc-2", "text", documentpb.PrivacyClass_PRIVACY_CLASS_INTERNAL)
	require.NoError(t, err)
	require.Equal(t, "unenriched", degraded.Status)
	_, err = outage.Embed(ctx, "doc-2", "unit", "text", documentpb.PrivacyClass_PRIVACY_CLASS_INTERNAL, "semantic-query")
	require.Error(t, err)
}

func TestEmbeddingMetadataRejectsIncompleteRows(t *testing.T) {
	require.Error(t, (Embedding{Role: "", Model: "model", Dimension: 1, ContentVersion: 1, RetargetStrategy: "reembed", Vector: []float32{1}}).Validate())
	require.Error(t, (Embedding{Role: "semantic", Model: "model", Dimension: 2, ContentVersion: 1, RetargetStrategy: "reembed", Vector: []float32{1}}).Validate())
}

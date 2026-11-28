package sketch

import (
	"context"
	"log"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	inference "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"
	assistconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspaceassist/workspaceassist_v1connect"
	"react-component-library/internal/designinference"
	internal "react-component-library/internal/sketch"
)

func TestMountedWorkspaceAssistGetSketchInferenceRecoversByIdempotencyKey(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(designinference.Schema)))
	repo := designinference.NewSQLiteRepository(db)
	op, err := repo.Create(ctx, "mounted-recovery", designinference.Request{
		Role: "extract.structured", MaxOutputTokens: 4096, Source: "design source",
		Schema: `{"type":"object"}`, Instruction: "Extract requirements", Context: []byte(`{}`),
	})
	require.NoError(t, err)

	root := t.TempDir()
	got := moduleFromDeps(root, log.Default(), nil, Deps{
		Store:        internal.NewStore(root),
		InferenceFor: func(context.Context) (designinference.Repository, error) { return repo, nil },
	})
	router := mux.NewRouter()
	got.Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()

	client := assistconnect.NewWorkspaceAssistServiceClient(server.Client(), server.URL)
	recovered, err := client.GetSketchInference(ctx, connect.NewRequest(&inference.GetSketchInferenceRequest{IdempotencyKey: "mounted-recovery"}))
	require.NoError(t, err)
	require.Equal(t, op.ID, recovered.Msg.GetId())
	require.Equal(t, "prepared", recovered.Msg.GetState())
	require.NotEmpty(t, recovered.Msg.GetRequestHash())

	_, err = client.GetSketchInference(ctx, connect.NewRequest(&inference.GetSketchInferenceRequest{Id: op.ID, IdempotencyKey: "mounted-recovery"}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

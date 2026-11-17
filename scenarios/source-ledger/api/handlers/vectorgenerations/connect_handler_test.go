package vectorgenerations

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/identity"
	vectorv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/vector"
	vectorstore "source-ledger/internal/vector"
)

type lifecycleFake struct {
	stageCalls []string
	operation  vectorstore.GenerationOperation
	stageErr   error
}

func (f *lifecycleFake) Stage(_ context.Context, operationID string) (vectorstore.SQLiteGeneration, error) {
	f.stageCalls = append(f.stageCalls, operationID)
	return vectorstore.SQLiteGeneration{ID: "generation-a", RecipeID: "recipe-a", Dimensions: 3, State: "candidate"}, f.stageErr
}
func (*lifecycleFake) CatchUp(context.Context, string, string) (vectorstore.SQLiteGeneration, error) {
	return vectorstore.SQLiteGeneration{}, nil
}
func (*lifecycleFake) Validate(context.Context, string) (vectorstore.GenerationValidation, error) {
	return vectorstore.GenerationValidation{}, nil
}
func (*lifecycleFake) Promote(context.Context, string, string) (vectorstore.SQLiteGeneration, error) {
	return vectorstore.SQLiteGeneration{}, nil
}
func (*lifecycleFake) Rollback(context.Context, string) (vectorstore.SQLiteGeneration, error) {
	return vectorstore.SQLiteGeneration{}, nil
}
func (f *lifecycleFake) Operation(context.Context, string) (vectorstore.GenerationOperation, error) {
	return f.operation, nil
}
func (*lifecycleFake) ResumeOperation(context.Context, string) (vectorstore.GenerationOperation, vectorstore.SQLiteGeneration, error) {
	return vectorstore.GenerationOperation{}, vectorstore.SQLiteGeneration{}, nil
}
func (*lifecycleFake) Active(context.Context) (vectorstore.SQLiteGeneration, bool, error) {
	return vectorstore.SQLiteGeneration{}, false, nil
}
func (*lifecycleFake) Load(context.Context, string) (vectorstore.SQLiteGeneration, error) {
	return vectorstore.SQLiteGeneration{}, nil
}

func TestStageRequiresVerifiedWriteAuthorityAndStableOperationID(t *testing.T) {
	fake := &lifecycleFake{operation: vectorstore.GenerationOperation{OperationID: "op-1", Operation: "stage", State: "complete"}}
	handler := NewConnectHandler(fake, nil)
	request := connect.NewRequest(&vectorv1.StageGenerationRequest{OperationId: "op-1"})

	_, err := handler.StageGeneration(context.Background(), request)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.Empty(t, fake.stageCalls)

	unsignedScope := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true})
	_, err = handler.StageGeneration(unsignedScope, request)
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	require.Empty(t, fake.stageCalls)

	principal := identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Scopes: []string{writeCapability}}
	authorized := identity.WithPrincipal(context.Background(), principal)
	_, err = handler.StageGeneration(authorized, connect.NewRequest(&vectorv1.StageGenerationRequest{OperationId: "bad/operation"}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.Empty(t, fake.stageCalls)

	response, err := handler.StageGeneration(authorized, request)
	require.NoError(t, err)
	require.Equal(t, []string{"op-1"}, fake.stageCalls)
	require.Equal(t, "generation-a", response.Msg.GetGeneration().GetGenerationId())
	require.Equal(t, "complete", response.Msg.GetOperation().GetState())
}

func TestStageMapsOwnerFailureWithoutLosingCause(t *testing.T) {
	fake := &lifecycleFake{stageErr: vectorstore.ErrGenerationDiskAdmissionUnavailable}
	handler := NewConnectHandler(fake, nil)
	principal := identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Scopes: []string{writeCapability}}
	_, err := handler.StageGeneration(identity.WithPrincipal(context.Background(), principal), connect.NewRequest(&vectorv1.StageGenerationRequest{OperationId: "op-1"}))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
	require.True(t, errors.Is(err, vectorstore.ErrGenerationDiskAdmissionUnavailable))
}

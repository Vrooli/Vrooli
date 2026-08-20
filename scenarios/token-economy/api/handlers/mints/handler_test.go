package mints

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	mintsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/mints"

	domain "token-economy/internal/mints"
)

type fakeService struct {
	get func(context.Context, string) (domain.TokenType, error)
}

func (f fakeService) Create(context.Context, domain.CreateInput) (domain.TokenType, error) {
	return domain.TokenType{}, nil
}

func (f fakeService) Get(ctx context.Context, id string) (domain.TokenType, error) {
	return f.get(ctx, id)
}
func (f fakeService) List(context.Context, bool) ([]domain.TokenType, error) { return nil, nil }
func (f fakeService) Retire(context.Context, string) (domain.TokenType, error) {
	return domain.TokenType{}, nil
}

func (f fakeService) Mint(context.Context, string, int64) (domain.TokenType, error) {
	return domain.TokenType{}, nil
}

// [REQ:TKE-P0-001] The Connect edge preserves the typed not-found result for an undeclared type.
func TestGetTokenTypeUnknownReturnsConnectNotFound(t *testing.T) {
	handler := NewConnectHandler(fakeService{get: func(context.Context, string) (domain.TokenType, error) {
		return domain.TokenType{}, domain.ErrTokenTypeNotFound
	}}, nil)

	_, err := handler.GetTokenType(context.Background(), connect.NewRequest(&mintsv1.GetTokenTypeRequest{Id: "missing"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

package mints_test

import (
	"context"
	"net/http/httptest"
	"reflect"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	accessH "token-economy/handlers/access"
	internalaccess "token-economy/internal/access"

	accessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access"
	accessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access/accessv1connect"
)

type holderValidator struct{}

func (holderValidator) Validate(context.Context, string) (internalaccess.Identity, error) {
	return internalaccess.Identity{Subject: "auth:holder", Scopes: []string{internalaccess.ScopeHolder}}, nil
}

// [REQ:TKE-P0-005] HolderService is fixed to capability-minimal methods and
// exposes no mint, grant, catalog-write, or rule-write operation.
func TestHolderDescriptorContainsNoAuthorityMethod(t *testing.T) {
	service := accessv1.File_token_economy_v1_access_access_proto.Services().ByName("HolderService")
	require.NotNil(t, service)
	got := make([]string, 0, service.Methods().Len())
	for index := 0; index < service.Methods().Len(); index++ {
		got = append(got, string(service.Methods().Get(index).Name()))
	}
	want := []string{"ViewEconomy", "BrowseCatalog", "RequestRedemption", "SubmitRequest"}
	require.True(t, reflect.DeepEqual(want, got), "methods = %v", got)
}

// [REQ:TKE-P0-005] A valid holder credential is refused at the transport
// boundary before the absent minter delegate could execute.
func TestHolderCredentialCannotInvokeMinterService(t *testing.T) {
	module := accessH.Module(nil, nil, nil, nil, nil, nil, nil, holderValidator{})
	router := mux.NewRouter()
	module.Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()

	client := accessconnect.NewMinterServiceClient(server.Client(), server.URL)
	request := connect.NewRequest(&accessv1.GetTokenTypeRequest{Id: "allowance"})
	request.Header().Set("Authorization", "Bearer valid-holder")
	_, err := client.GetTokenType(context.Background(), request)
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}

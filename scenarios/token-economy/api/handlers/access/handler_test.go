package access

import (
	"context"
	"net/http/httptest"
	"reflect"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"

	internalaccess "token-economy/internal/access"

	accessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access"
	accessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access/accessv1connect"
	grantsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/grants"
	mintsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/mints"
)

// HolderService is a generated, capability-minimal surface:
// descriptor evolution cannot silently add authority-bearing RPCs.
func TestHolderDescriptorContainsOnlyHolderCapabilities(t *testing.T) {
	service := accessv1.File_token_economy_v1_access_access_proto.Services().ByName("HolderService")
	if service == nil {
		t.Fatal("HolderService descriptor missing")
	}
	got := make([]string, 0, service.Methods().Len())
	for index := 0; index < service.Methods().Len(); index++ {
		got = append(got, string(service.Methods().Get(index).Name()))
	}
	want := []string{"ViewEconomy", "BrowseCatalog", "RequestRedemption", "SubmitRequest"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("HolderService methods = %v, want %v", got, want)
	}
}

// A valid holder credential cannot reach a minter RPC. The
// denial happens in the Connect interceptor before the domain delegate runs.
func TestHolderTokenCannotInvokeMinterService(t *testing.T) {
	mints := &mintSpy{}
	module := Module(mints, grantStub{}, holderStub{}, nil, nil, nil, nil, staticValidator{})
	router := mux.NewRouter()
	module.Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()

	minter := accessconnect.NewMinterServiceClient(server.Client(), server.URL)
	request := connect.NewRequest(&accessv1.GetTokenTypeRequest{Id: "allowance"})
	request.Header().Set("Authorization", "Bearer holder-token")
	_, err := minter.GetTokenType(context.Background(), request)
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("holder calling minter code = %v, want permission_denied (err=%v)", connect.CodeOf(err), err)
	}
	if mints.calls != 0 {
		t.Fatalf("minter delegate called %d times after transport denial", mints.calls)
	}

	holder := accessconnect.NewHolderServiceClient(server.Client(), server.URL)
	holderRequest := connect.NewRequest(&accessv1.ViewEconomyRequest{})
	holderRequest.Header().Set("Authorization", "Bearer holder-token")
	_, err = holder.ViewEconomy(context.Background(), holderRequest)
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("valid holder token code = %v, want unimplemented from phase-owned stub (err=%v)", connect.CodeOf(err), err)
	}
}

func TestMinterIdentityComesFromValidatedCredential(t *testing.T) {
	mints := &mintSpy{}
	module := Module(mints, grantStub{}, holderStub{}, nil, nil, nil, nil, staticValidator{})
	router := mux.NewRouter()
	module.Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()

	client := accessconnect.NewMinterServiceClient(server.Client(), server.URL)
	request := connect.NewRequest(&accessv1.GetTokenTypeRequest{Id: "allowance"})
	request.Header().Set("Authorization", "Bearer minter-token")
	if _, err := client.GetTokenType(context.Background(), request); err != nil {
		t.Fatalf("GetTokenType: %v", err)
	}
	if mints.subject != "authenticated-parent" {
		t.Fatalf("delegate identity subject = %q, want authenticated-parent", mints.subject)
	}
}

// Authenticated surfaces fail closed before a holder handler
// runs when scenario-authenticator validation is unavailable.
func TestHolderSurfaceFailsClosedWhenAuthenticatorUnavailable(t *testing.T) {
	holder := &holderSpy{}
	module := Module(&mintSpy{}, grantStub{}, holder, nil, nil, nil, nil, nil)
	router := mux.NewRouter()
	module.Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()

	client := accessconnect.NewHolderServiceClient(server.Client(), server.URL)
	request := connect.NewRequest(&accessv1.ViewEconomyRequest{})
	request.Header().Set("Authorization", "Bearer otherwise-valid")
	_, err := client.ViewEconomy(context.Background(), request)
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("authenticator outage code = %v, want unavailable (err=%v)", connect.CodeOf(err), err)
	}
	if holder.calls != 0 {
		t.Fatalf("holder delegate called %d times after fail-closed denial", holder.calls)
	}
}

type staticValidator struct{}

func (staticValidator) Validate(_ context.Context, token string) (internalaccess.Identity, error) {
	switch token {
	case "holder-token":
		return internalaccess.Identity{Subject: "authenticated-child", Roles: []string{"holder"}, Scopes: []string{internalaccess.ScopeHolder}}, nil
	case "minter-token":
		return internalaccess.Identity{Subject: "authenticated-parent", Roles: []string{"minter"}, Scopes: []string{internalaccess.ScopeMinter}}, nil
	default:
		return internalaccess.Identity{}, internalaccess.ErrUnauthenticated
	}
}

type mintSpy struct {
	calls   int
	subject string
}

func (*mintSpy) CreateTokenType(context.Context, *connect.Request[mintsv1.CreateTokenTypeRequest]) (*connect.Response[mintsv1.CreateTokenTypeResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (s *mintSpy) GetTokenType(ctx context.Context, _ *connect.Request[mintsv1.GetTokenTypeRequest]) (*connect.Response[mintsv1.GetTokenTypeResponse], error) {
	s.calls++
	identity, _ := internalaccess.IdentityFromContext(ctx)
	s.subject = identity.Subject
	return connect.NewResponse(&mintsv1.GetTokenTypeResponse{}), nil
}

func (*mintSpy) ListTokenTypes(context.Context, *connect.Request[mintsv1.ListTokenTypesRequest]) (*connect.Response[mintsv1.ListTokenTypesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (*mintSpy) RetireTokenType(context.Context, *connect.Request[mintsv1.RetireTokenTypeRequest]) (*connect.Response[mintsv1.RetireTokenTypeResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (*mintSpy) MintSupply(context.Context, *connect.Request[mintsv1.MintSupplyRequest]) (*connect.Response[mintsv1.MintSupplyResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

type grantStub struct{}

func (grantStub) CreateGrant(context.Context, *connect.Request[grantsv1.CreateGrantRequest]) (*connect.Response[grantsv1.CreateGrantResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (grantStub) GetGrant(context.Context, *connect.Request[grantsv1.GetGrantRequest]) (*connect.Response[grantsv1.GetGrantResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (grantStub) ListGrants(context.Context, *connect.Request[accessv1.ListGrantsRequest]) (*connect.Response[accessv1.ListGrantsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (grantStub) RevokeGrant(context.Context, *connect.Request[accessv1.RevokeGrantRequest]) (*connect.Response[accessv1.RevokeGrantResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

type holderStub struct{}

func (holderStub) CreateHolder(context.Context, *connect.Request[accessv1.CreateHolderRequest]) (*connect.Response[accessv1.CreateHolderResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (holderStub) GetHolder(context.Context, *connect.Request[accessv1.GetHolderRequest]) (*connect.Response[accessv1.GetHolderResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (holderStub) ListHolders(context.Context, *connect.Request[accessv1.ListHoldersRequest]) (*connect.Response[accessv1.ListHoldersResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (holderStub) ViewEconomy(context.Context, string, *connect.Request[accessv1.ViewEconomyRequest]) (*connect.Response[accessv1.ViewEconomyResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

type holderSpy struct{ calls int }

func (*holderSpy) CreateHolder(context.Context, *connect.Request[accessv1.CreateHolderRequest]) (*connect.Response[accessv1.CreateHolderResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (*holderSpy) GetHolder(context.Context, *connect.Request[accessv1.GetHolderRequest]) (*connect.Response[accessv1.GetHolderResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (*holderSpy) ListHolders(context.Context, *connect.Request[accessv1.ListHoldersRequest]) (*connect.Response[accessv1.ListHoldersResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (s *holderSpy) ViewEconomy(context.Context, string, *connect.Request[accessv1.ViewEconomyRequest]) (*connect.Response[accessv1.ViewEconomyResponse], error) {
	s.calls++
	return connect.NewResponse(&accessv1.ViewEconomyResponse{}), nil
}

var (
	_ GrantsDelegate = grantStub{}
	_ HolderDelegate = holderStub{}
	_ HolderDelegate = (*holderSpy)(nil)
	_ MintsDelegate  = (*mintSpy)(nil)
)

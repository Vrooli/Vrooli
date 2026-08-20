package earning_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"

	accessH "token-economy/handlers/access"
	earningH "token-economy/handlers/earning"
	internalaccess "token-economy/internal/access"
	domain "token-economy/internal/earning"

	earningv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/earning"
	earningconnect "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/earning/earningv1connect"
)

type validator struct{}

func (validator) Validate(_ context.Context, token string) (internalaccess.Identity, error) {
	switch token {
	case "operator":
		return internalaccess.Identity{Subject: "operator:alex", Scopes: []string{internalaccess.ScopeEarning}}, nil
	case "programmatic":
		return internalaccess.Identity{Subject: "scenario:chore-tracker", Scopes: []string{internalaccess.ScopeEarning}}, nil
	case "holder":
		return internalaccess.Identity{Subject: "child:sam", Scopes: []string{internalaccess.ScopeHolder}}, nil
	default:
		return internalaccess.Identity{}, internalaccess.ErrUnauthenticated
	}
}

type delegateSpy struct {
	identities []string
}

func (s *delegateSpy) Submit(_ context.Context, adapterIdentity string, input domain.Input) (domain.Submission, error) {
	s.identities = append(s.identities, adapterIdentity)
	return domain.Submission{
		ID: "submission-" + input.DedupKey, HolderID: input.HolderID, TokenTypeID: input.TokenTypeID,
		AmountMinor: input.AmountMinor, Reason: input.Reason, DedupKey: input.DedupKey,
		AdapterIdentity: adapterIdentity, ActorIdentity: adapterIdentity,
	}, nil
}

func (*delegateSpy) List(context.Context) ([]domain.Submission, error) {
	return nil, nil
}

// [REQ:TKE-P0-007] Operator and programmatic credentials reach the same
// adapter-only RPC, and the authenticated subject supplies adapter identity.
func TestAuthenticatedAdaptersUseOneTransportMethod(t *testing.T) {
	spy := &delegateSpy{}
	module := accessH.Module(nil, nil, nil, earningH.NewConnectHandler(spy, nil), nil, nil, nil, validator{})
	router := mux.NewRouter()
	module.Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := earningconnect.NewEarningServiceClient(server.Client(), server.URL)

	for _, test := range []struct {
		token   string
		dedup   string
		subject string
	}{
		{token: "operator", dedup: "manual-1", subject: "operator:alex"},
		{token: "programmatic", dedup: "event-1", subject: "scenario:chore-tracker"},
	} {
		request := connect.NewRequest(&earningv1.SubmitEarningRequest{
			HolderId: "child:sam", TokenTypeId: "chores", AmountMinor: 5,
			Reason: "work completed", DedupKey: test.dedup,
		})
		request.Header().Set("Authorization", "Bearer "+test.token)
		response, err := client.SubmitEarning(context.Background(), request)
		if err != nil {
			t.Fatalf("SubmitEarning(%s): %v", test.token, err)
		}
		if response.Msg.Submission.AdapterIdentity != test.subject {
			t.Fatalf("adapter identity = %q, want %q", response.Msg.Submission.AdapterIdentity, test.subject)
		}
	}
	if len(spy.identities) != 2 {
		t.Fatalf("shared delegate calls = %d, want 2", len(spy.identities))
	}

	holderRequest := connect.NewRequest(&earningv1.SubmitEarningRequest{
		HolderId: "child:sam", TokenTypeId: "chores", AmountMinor: 5,
		Reason: "attempted holder submission", DedupKey: "holder-1",
	})
	holderRequest.Header().Set("Authorization", "Bearer holder")
	_, err := client.SubmitEarning(context.Background(), holderRequest)
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("holder earning call code = %v, want permission_denied (err=%v)", connect.CodeOf(err), err)
	}
	if len(spy.identities) != 2 {
		t.Fatalf("delegate called after holder denial; calls=%d", len(spy.identities))
	}
}

package peer

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"

	operatorv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/operator"
	operatorconnect "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/operator/operator_v1connect"
)

// The access is said, nothing is sent without a desk credential, and a
// refusal is remembered until a call succeeds.
func TestAccessIsShownAndARefusalIsRemembered(t *testing.T) {
	refuse := true
	var authorization []string
	mux := http.NewServeMux()
	mux.Handle(operatorconnect.NewOperatorServiceHandler(operatorconnect.UnimplementedOperatorServiceHandler{}, connect.WithInterceptors(connect.UnaryInterceptorFunc(func(connect.UnaryFunc) connect.UnaryFunc {
		return func(_ context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			authorization = append(authorization, req.Header().Get("Authorization"))
			if refuse {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.New("missing web-console:write"))
			}
			return connect.NewResponse(&operatorv1.ListReviewsResponse{}), nil
		}
	}))))
	server := httptest.NewServer(mux)
	defer server.Close()

	p := &Peer{Name: "web-console", BaseURL: func(context.Context) (string, error) { return server.URL, nil }, Credential: func(context.Context) (string, error) { return "", nil }}
	ctx := context.Background()
	if access, err := p.Access(ctx); err != nil || access != AccessAvailability {
		t.Fatalf("access = %q, %v", access, err)
	}
	call := func() error {
		client, base, opt, err := p.Dial(ctx)
		if err != nil {
			return err
		}
		_, err = operatorconnect.NewOperatorServiceClient(client, base, opt).ListReviews(ctx, connect.NewRequest(&operatorv1.ListReviewsRequest{}))
		return err
	}
	if call() == nil || p.Refused() == nil || !strings.Contains(p.Refused().Error(), "web-console refused") {
		t.Fatalf("the refusal is remembered: %v", p.Refused())
	}
	refuse = false
	if err := call(); err != nil || p.Refused() != nil {
		t.Fatalf("a later success clears it: %v %v", err, p.Refused())
	}
	if authorization[0] != "" {
		t.Fatalf("no desk credential, no Authorization header: %q", authorization[0])
	}

	p.Credential = func(context.Context) (string, error) { return "Bearer desk", nil }
	if access, _ := p.Access(ctx); access != AccessCredential {
		t.Fatalf("access = %q", access)
	}
	_ = call()
	if authorization[len(authorization)-1] != "Bearer desk" {
		t.Fatalf("the desk's own credential is sent: %q", authorization)
	}
}

package credentials

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	sharedsession "github.com/vrooli/api-core/operatorsession"
	grantv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/credentialgrant"
	grantconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/credentialgrant/credentialgrant_v1connect"
)

// GrantClient is the owner-facing, metadata-only bridge client. It deliberately
// has no method that accepts or returns a credential value.
type GrantClient struct {
	client grantconnect.CredentialGrantServiceClient
}

func NewGrantClient() (*GrantClient, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("VROOLI_BRIDGE_URL")), "/")
	if baseURL == "" {
		resolved, err := discovery.ResolveScenarioURLDefault(context.Background(), "vrooli-bridge")
		if err != nil {
			return nil, fmt.Errorf("resolve vrooli-bridge endpoint: %w", err)
		}
		baseURL = strings.TrimRight(strings.TrimSpace(resolved), "/")
	}
	authorizationScheme, token := resolveGrantAuthorization()
	if token == "" {
		return nil, fmt.Errorf("Bridge owner session is unavailable; enroll this Vrooli CLI before using credential grant commands")
	}
	transport := authorizationTransport{base: http.DefaultTransport, scheme: authorizationScheme, token: token}
	httpClient := &http.Client{Transport: transport}
	return &GrantClient{client: grantconnect.NewCredentialGrantServiceClient(httpClient, baseURL)}, nil
}

func resolveGrantAuthorization() (string, string) {
	if store, err := sharedsession.DefaultFileStore(); err == nil {
		if resolution, err := (sharedsession.LocalResolver{Store: store}).Resolve(); err == nil && strings.TrimSpace(resolution.Token) != "" {
			return sharedsession.LocalSessionScheme, resolution.Token
		}
	}
	token := strings.TrimSpace(os.Getenv("VROOLI_BRIDGE_API_TOKEN"))
	if token == "" {
		token = strings.TrimSpace(os.Getenv("VROOLI_BRIDGE_TOKEN"))
	}
	return "Bearer", token
}

type authorizationTransport struct {
	base   http.RoundTripper
	scheme string
	token  string
}

func (t authorizationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.token == "" {
		return t.base.RoundTrip(req)
	}
	copyReq := req.Clone(req.Context())
	copyReq.Header.Set("Authorization", t.scheme+" "+t.token)
	return t.base.RoundTrip(copyReq)
}

func (c *GrantClient) Create(ctx context.Context, req *grantv1.CreateGrantRequest) (*grantv1.CredentialGrant, error) {
	response, err := c.client.CreateGrant(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (c *GrantClient) List(ctx context.Context, nodeID string) ([]*grantv1.CredentialGrant, error) {
	response, err := c.client.ListGrants(ctx, connect.NewRequest(&grantv1.ListGrantsRequest{NodeId: nodeID}))
	if err != nil {
		return nil, err
	}
	return response.Msg.GetGrants(), nil
}

func (c *GrantClient) Revoke(ctx context.Context, id string) (*grantv1.CredentialGrant, error) {
	response, err := c.client.RevokeGrant(ctx, connect.NewRequest(&grantv1.RevokeGrantRequest{Id: id}))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (c *GrantClient) Rotate(ctx context.Context, logicalID, field string) (*grantv1.RotationResponse, error) {
	response, err := c.client.RotateAddress(ctx, connect.NewRequest(&grantv1.RotateAddressRequest{LogicalId: logicalID, Field: field}))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

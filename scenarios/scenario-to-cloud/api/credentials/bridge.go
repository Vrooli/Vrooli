package credentials

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	credentialgrantv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/credentialgrant"
	"github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/credentialgrant/credentialgrant_v1connect"
	dispatchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch"
	"github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch/dispatch_v1connect"
)

// BridgeConfig binds the Connect clients to one Bridge. The owner credential
// is supplied by a provider so a short-lived session token is fetched per
// call rather than pinned in configuration.
type BridgeConfig struct {
	HTTPClient    *http.Client
	ResolveURL    func(ctx context.Context) (string, error)
	TokenProvider func(ctx context.Context) (string, error)
}

// BridgeClient implements GrantClient and Dispatcher over the generated
// Connect clients.
type BridgeClient struct {
	cfg BridgeConfig
}

// NewBridgeClient validates the configuration.
func NewBridgeClient(cfg BridgeConfig) (*BridgeClient, error) {
	if cfg.ResolveURL == nil {
		return nil, errors.New("bridge client: ResolveURL is required")
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{}
	}
	return &BridgeClient{cfg: cfg}, nil
}

type bearerTransport struct {
	base     *http.Client
	provider func(ctx context.Context) (string, error)
}

func (t bearerTransport) Do(req *http.Request) (*http.Response, error) {
	if t.provider != nil {
		token, err := t.provider(req.Context())
		if err != nil {
			return nil, err
		}
		token = strings.TrimSpace(token)
		if token != "" {
			if !strings.Contains(token, " ") {
				token = "Bearer " + token
			}
			req.Header.Set("Authorization", token)
		}
	}
	return t.base.Do(req) // #nosec G704 -- request URL comes from the validated Bridge service resolver.
}

func (b *BridgeClient) grants(ctx context.Context) (credentialgrant_v1connect.CredentialGrantServiceClient, error) {
	url, err := b.cfg.ResolveURL(ctx)
	if err != nil {
		return nil, err
	}
	return credentialgrant_v1connect.NewCredentialGrantServiceClient(bearerTransport{base: b.cfg.HTTPClient, provider: b.cfg.TokenProvider}, strings.TrimRight(url, "/")), nil
}

func (b *BridgeClient) dispatch(ctx context.Context) (dispatch_v1connect.DispatchServiceClient, error) {
	url, err := b.cfg.ResolveURL(ctx)
	if err != nil {
		return nil, err
	}
	return dispatch_v1connect.NewDispatchServiceClient(bearerTransport{base: b.cfg.HTTPClient, provider: b.cfg.TokenProvider}, strings.TrimRight(url, "/")), nil
}

func grantFromProto(g *credentialgrantv1.CredentialGrant) Grant {
	if g == nil {
		return Grant{}
	}
	return Grant{ID: g.GetId(), NodeID: g.GetNodeId(), LogicalID: g.GetLogicalId(), Field: g.GetField(), Generation: g.GetGeneration(), AckedGeneration: g.GetAckedGeneration(), Revoked: g.GetRevokedAt() != nil, ReceiptAccepted: g.GetReceiptAccepted(), PurgeState: g.GetPurgeState()}
}

func (b *BridgeClient) ListGrants(ctx context.Context, nodeID string) ([]Grant, error) {
	client, err := b.grants(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.ListGrants(ctx, connect.NewRequest(&credentialgrantv1.ListGrantsRequest{NodeId: nodeID}))
	if err != nil {
		return nil, err
	}
	out := make([]Grant, 0, len(resp.Msg.GetGrants()))
	for _, g := range resp.Msg.GetGrants() {
		out = append(out, grantFromProto(g))
	}
	return out, nil
}

func (b *BridgeClient) CreateGrant(ctx context.Context, spec GrantSpec) (Grant, error) {
	client, err := b.grants(ctx)
	if err != nil {
		return Grant{}, err
	}
	resp, err := client.CreateGrant(ctx, connect.NewRequest(&credentialgrantv1.CreateGrantRequest{NodeId: spec.NodeID, LogicalId: spec.LogicalID, Field: spec.Field, Class: spec.Class, Retention: spec.Retention}))
	if err != nil {
		return Grant{}, err
	}
	return grantFromProto(resp.Msg), nil
}

func (b *BridgeClient) AnswerSecret(ctx context.Context, spec GrantSpec, value string) (Grant, error) {
	client, err := b.grants(ctx)
	if err != nil {
		return Grant{}, err
	}
	resp, err := client.AnswerSecret(ctx, connect.NewRequest(&credentialgrantv1.AnswerSecretRequest{NodeId: spec.NodeID, LogicalId: spec.LogicalID, Field: spec.Field, Class: spec.Class, Retention: spec.Retention, Value: value}))
	if err != nil {
		return Grant{}, err
	}
	return grantFromProto(resp.Msg), nil
}

func (b *BridgeClient) RevokeGrant(ctx context.Context, id string) (Grant, error) {
	client, err := b.grants(ctx)
	if err != nil {
		return Grant{}, err
	}
	resp, err := client.RevokeGrant(ctx, connect.NewRequest(&credentialgrantv1.RevokeGrantRequest{Id: id}))
	if err != nil {
		return Grant{}, err
	}
	return grantFromProto(resp.Msg), nil
}

func (b *BridgeClient) Dispatch(ctx context.Context, job DispatchJob) (string, error) {
	client, err := b.dispatch(ctx)
	if err != nil {
		return "", err
	}
	req := &dispatchv1.DispatchJobRequest{NodeId: job.NodeID, Scenario: job.Scenario, Verb: job.Verb, Args: append([]string(nil), job.Args...)}
	for _, injection := range job.Injections {
		req.CredentialInjections = append(req.CredentialInjections, &dispatchv1.CredentialInjection{LogicalId: injection.LogicalID, Field: injection.Field, EnvName: injection.EnvName})
	}
	resp, err := client.DispatchJob(ctx, connect.NewRequest(req))
	if err != nil {
		return "", err
	}
	return resp.Msg.GetRunId(), nil
}

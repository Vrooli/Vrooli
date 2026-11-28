package standinggrantpermissions

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/scopecatalog"
	permissionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/permissions"
	permissionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/permissions/permissionsv1connect"
)

type proofKey struct{}
type incomingProof struct {
	authorization, accessJWT string
	invalid                  bool
}

// WithIncomingOwnerProof forwards only original auth proof so Agent Manager
// independently authorizes the same principal. It never creates authority.
func WithIncomingOwnerProof(ctx context.Context, headers http.Header) context.Context {
	p := incomingProof{}
	for _, name := range []string{"Authorization", "Cf-Access-Jwt-Assertion"} {
		values := headers.Values(name)
		if len(values) > 1 {
			p.invalid = true
			continue
		}
		if len(values) == 1 {
			if values[0] == "" || len(values[0]) > 16384 || strings.ContainsAny(values[0], "\r\n") {
				p.invalid = true
				continue
			}
			if name == "Authorization" {
				p.authorization = values[0]
			} else {
				p.accessJWT = values[0]
			}
		}
	}
	return context.WithValue(ctx, proofKey{}, p)
}

type Client struct {
	BaseURL func(context.Context) (string, error)
	HTTP    *http.Client
}

func NewDiscoveredClient() *Client {
	return &Client{BaseURL: func(ctx context.Context) (string, error) {
		return discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
	}}
}

func (c *Client) List(ctx context.Context, limit int, cursor string) (Page, error) {
	proof, client, err := c.request(ctx)
	if err != nil {
		return Page{}, err
	}
	req := connect.NewRequest(&permissionsv1.ListPermissionProjectionRequest{Limit: uint32(limit), Cursor: cursor})
	forwardProof(proof, req)
	response, err := client.ListPermissionProjection(ctx, req)
	if err != nil {
		return Page{}, mapConnectError(err)
	}
	out := Page{Owner: response.Msg.GetOwner(), SourceRevision: response.Msg.GetSourceRevision(), NextCursor: response.Msg.GetNextCursor(), Scopes: make([]scopecatalog.Scope, 0, len(response.Msg.GetScopeCatalog())), Grants: make([]Grant, 0, len(response.Msg.GetGrants()))}
	for _, s := range response.Msg.GetScopeCatalog() {
		if s == nil {
			return Page{}, connect.NewError(connect.CodeDataLoss, errors.New("Agent Manager returned an empty scope declaration"))
		}
		out.Scopes = append(out.Scopes, scopecatalog.Scope{Scenario: s.GetScenario(), Value: s.GetScope(), Effect: scopecatalog.Effect(s.GetEffect()), RunEligible: s.GetRunEligible(), Permissions: append([]string(nil), s.GetPermissions()...), Command: s.GetCommand(), Service: s.GetService(), Method: s.GetMethod()})
	}
	for _, g := range response.Msg.GetGrants() {
		grant, err := grantFromProto(g)
		if err != nil {
			return Page{}, err
		}
		out.Grants = append(out.Grants, grant)
	}
	return out, nil
}

func (c *Client) Read(ctx context.Context, kind, reference, revision string) (OwnerValue, error) {
	proof, client, err := c.request(ctx)
	if err != nil {
		return OwnerValue{}, err
	}
	req := connect.NewRequest(&permissionsv1.ReadPermissionProjectionRequest{Kind: kind, Reference: reference, ExpectedSourceRevision: revision})
	forwardProof(proof, req)
	response, err := client.ReadPermissionProjection(ctx, req)
	if err != nil {
		return OwnerValue{}, mapConnectError(err)
	}
	out := OwnerValue{Owner: response.Msg.GetOwner(), SourceRevision: response.Msg.GetSourceRevision()}
	switch value := response.Msg.GetValue().(type) {
	case *permissionsv1.ReadPermissionProjectionResponse_Scope:
		s := value.Scope
		if s == nil {
			return OwnerValue{}, connect.NewError(connect.CodeDataLoss, errors.New("Agent Manager returned an empty scope declaration"))
		}
		out.Scope = &scopecatalog.Scope{Scenario: s.GetScenario(), Value: s.GetScope(), Effect: scopecatalog.Effect(s.GetEffect()), RunEligible: s.GetRunEligible(), Permissions: append([]string(nil), s.GetPermissions()...), Command: s.GetCommand(), Service: s.GetService(), Method: s.GetMethod()}
	case *permissionsv1.ReadPermissionProjectionResponse_Grant:
		grant, err := grantFromProto(value.Grant)
		if err != nil {
			return OwnerValue{}, err
		}
		out.Grant = &grant
	default:
		return OwnerValue{}, connect.NewError(connect.CodeDataLoss, errors.New("Agent Manager returned an empty permission value"))
	}
	return out, nil
}

func (c *Client) request(ctx context.Context) (incomingProof, permissionsconnect.PermissionProjectionServiceClient, error) {
	p, ok := ctx.Value(proofKey{}).(incomingProof)
	if !ok || p.invalid || (p.authorization == "" && p.accessJWT == "") {
		return incomingProof{}, nil, connect.NewError(connect.CodeUnauthenticated, errors.New("forwarded Agent Manager owner proof is unavailable"))
	}
	baseURL := c.BaseURL
	if baseURL == nil {
		baseURL = func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
		}
	}
	base, err := baseURL(ctx)
	if err != nil {
		return incomingProof{}, nil, connect.NewError(connect.CodeUnavailable, errors.New("Agent Manager is unavailable"))
	}
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return p, permissionsconnect.NewPermissionProjectionServiceClient(httpClient, strings.TrimRight(base, "/")), nil
}

func forwardProof[T any](proof incomingProof, request *connect.Request[T]) {
	if proof.authorization != "" {
		request.Header().Set("Authorization", proof.authorization)
	}
	if proof.accessJWT != "" {
		request.Header().Set("Cf-Access-Jwt-Assertion", proof.accessJWT)
	}
}
func grantFromProto(g *permissionsv1.StandingGrantMetadata) (Grant, error) {
	if g == nil || g.GetApprovedAt() == nil || g.GetExpiresAt() == nil {
		return Grant{}, connect.NewError(connect.CodeDataLoss, errors.New("Agent Manager returned incomplete standing grant metadata"))
	}
	out := Grant{ID: g.GetGrantRef(), Subject: g.GetSubject(), Approver: g.GetApprover(), Operations: append([]string(nil), g.GetOperations()...), Targets: append([]string(nil), g.GetTargets()...), State: g.GetState(), MaxConcurrentRuns: int(g.GetMaxConcurrentRuns()), MaxRunBudgetUSD: g.GetMaxRunBudgetUsd(), ApprovedAt: g.GetApprovedAt().AsTime(), ExpiresAt: g.GetExpiresAt().AsTime()}
	if g.GetRevokedAt() != nil {
		value := g.GetRevokedAt().AsTime()
		out.RevokedAt = &value
	}
	return out, nil
}
func mapConnectError(err error) error {
	switch connect.CodeOf(err) {
	case connect.CodeUnauthenticated, connect.CodePermissionDenied, connect.CodeInvalidArgument, connect.CodeNotFound, connect.CodeFailedPrecondition, connect.CodeResourceExhausted, connect.CodeDataLoss:
		return err
	default:
		return connect.NewError(connect.CodeUnavailable, errors.New("Agent Manager permission projection read failed"))
	}
}

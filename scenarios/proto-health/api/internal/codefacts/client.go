package codefacts

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	factsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts/facts_v1connect"
)

const scenarioID = "code-facts"

type URLResolver interface {
	ResolveScenarioURLDefault(ctx context.Context, scenarioSlug string) (string, error)
}

type Client struct {
	resolver   URLResolver
	httpClient connect.HTTPClient
}

func NewClient(resolver URLResolver, httpClient connect.HTTPClient) *Client {
	if resolver == nil {
		resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{resolver: resolver, httpClient: httpClient}
}

func (c *Client) CheckProtoAdoption(ctx context.Context, scenario string) (*factsv1.ProofReport, error) {
	client, err := c.client(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.CheckProtoAdoption(ctx, connect.NewRequest(&factsv1.CheckProtoAdoptionRequest{
		Target:   scenarioTarget(scenario),
		UseCache: true,
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func (c *Client) CheckEndpointProof(ctx context.Context, scenario string, endpointIDs []string) (*factsv1.ProofReport, error) {
	client, err := c.client(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.CheckEndpointProof(ctx, connect.NewRequest(&factsv1.CheckEndpointProofRequest{
		Target:      scenarioTarget(scenario),
		EndpointIds: endpointIDs,
		UseCache:    true,
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func (c *Client) client(ctx context.Context) (factsconnect.CodeFactsServiceClient, error) {
	baseURL, err := c.resolver.ResolveScenarioURLDefault(ctx, scenarioID)
	if err != nil {
		return nil, err
	}
	return factsconnect.NewCodeFactsServiceClient(c.httpClient, baseURL), nil
}

func scenarioTarget(scenario string) *factsv1.CodeTarget {
	return &factsv1.CodeTarget{
		Kind:     factsv1.TargetKind_TARGET_KIND_SCENARIO,
		Scenario: scenario,
	}
}

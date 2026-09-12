package baselines

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"

	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
	baselinesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines/baselines_v1connect"
)

const scenarioID = "git-control-tower"

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

func (c *Client) ListBaselines(ctx context.Context, scenario, branch string) ([]*baselinesv1.BaselineManifest, error) {
	baseURL, err := c.resolver.ResolveScenarioURLDefault(ctx, scenarioID)
	if err != nil {
		return nil, err
	}
	client := baselinesconnect.NewBaselinesServiceClient(c.httpClient, baseURL)
	resp, err := client.ListBaselines(ctx, connect.NewRequest(&baselinesv1.ListBaselinesRequest{
		Scenario: scenario,
		Branch:   branch,
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetBaselines(), nil
}

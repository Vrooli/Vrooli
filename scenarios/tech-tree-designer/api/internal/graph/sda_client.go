package graph

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"

	sdagraphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/graph"
	sdagraphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/graph/graph_v1connect"
)

const scenarioDependencyAnalyzerScenario = "scenario-dependency-analyzer"

type URLResolver interface {
	ResolveScenarioURLDefault(ctx context.Context, scenarioSlug string) (string, error)
}

type SDAInterfaceGraphClient interface {
	DescribeInterfaceGraph(ctx context.Context, req SDAInterfaceGraphRequest) (*SDAInterfaceGraphResponse, error)
}

type SDAInterfaceGraphRequest struct {
	Scenarios       []string
	Limit           int32
	StabilityFilter string
	MaxScenarioHops int32
}

type SDAInterfaceGraphResponse struct {
	Graph *sdagraphv1.InterfaceGraph
}

type SDAClient struct {
	resolver   URLResolver
	httpClient connect.HTTPClient
}

func NewSDAClient(resolver URLResolver, httpClient connect.HTTPClient) *SDAClient {
	if resolver == nil {
		resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &SDAClient{resolver: resolver, httpClient: httpClient}
}

func (c *SDAClient) DescribeInterfaceGraph(ctx context.Context, req SDAInterfaceGraphRequest) (*SDAInterfaceGraphResponse, error) {
	baseURL, err := c.resolver.ResolveScenarioURLDefault(ctx, scenarioDependencyAnalyzerScenario)
	if err != nil {
		return nil, err
	}
	client := sdagraphconnect.NewInterfaceGraphServiceClient(c.httpClient, baseURL)
	resp, err := client.DescribeInterfaceGraph(ctx, connect.NewRequest(&sdagraphv1.DescribeInterfaceGraphRequest{
		Scenarios:       req.Scenarios,
		Limit:           req.Limit,
		StabilityFilter: req.StabilityFilter,
		MaxScenarioHops: req.MaxScenarioHops,
	}))
	if err != nil {
		return nil, err
	}
	return &SDAInterfaceGraphResponse{Graph: resp.Msg.GetGraph()}, nil
}

package graph

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/shared"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/validation/validation_v1connect"
)

const protoHealthScenario = "proto-health"

type URLResolver interface {
	ResolveScenarioURLDefault(ctx context.Context, scenarioSlug string) (string, error)
}

type ProtoSurfaceClient interface {
	DescribeScenariosProtos(ctx context.Context, req ProtoSurfaceRequest) (*ProtoSurfaceResponse, error)
}

type ProtoSurfaceRequest struct {
	Scenarios       []string
	Limit           int32
	StabilityFilter string
}

type ProtoSurfaceResponse struct {
	Results []ProtoSurfaceResult
}

type ProtoSurfaceResult struct {
	Scenario string
	Surface  ProtoSurface
	Error    string
}

type ProtoSurface struct {
	Scenario             string
	Files                []ProtoFile
	CrossScenarioImports []ProtoImport
	TransportWorld       string
}

type ProtoFile struct {
	Path      string
	Stability string
}

type ProtoImport struct {
	FromFile     string
	ToFile       string
	FromScenario string
	ToScenario   string
	FromPackage  string
	ToPackage    string
}

type ProtoHealthClient struct {
	resolver   URLResolver
	httpClient connect.HTTPClient
}

func NewProtoHealthClient(resolver URLResolver, httpClient connect.HTTPClient) *ProtoHealthClient {
	if resolver == nil {
		resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &ProtoHealthClient{resolver: resolver, httpClient: httpClient}
}

func (c *ProtoHealthClient) DescribeScenariosProtos(ctx context.Context, req ProtoSurfaceRequest) (*ProtoSurfaceResponse, error) {
	baseURL, err := c.resolver.ResolveScenarioURLDefault(ctx, protoHealthScenario)
	if err != nil {
		return nil, err
	}
	client := validationconnect.NewProtoHealthServiceClient(c.httpClient, baseURL)
	resp, err := client.DescribeScenariosProtos(ctx, connect.NewRequest(&validationv1.DescribeScenariosProtosRequest{
		Scenarios:       req.Scenarios,
		Limit:           req.Limit,
		StabilityFilter: req.StabilityFilter,
	}))
	if err != nil {
		return nil, err
	}
	return protoSurfaceResponseFromProto(resp.Msg), nil
}

func protoSurfaceResponseFromProto(resp *validationv1.DescribeScenariosProtosResponse) *ProtoSurfaceResponse {
	if resp == nil {
		return &ProtoSurfaceResponse{}
	}
	out := &ProtoSurfaceResponse{Results: make([]ProtoSurfaceResult, 0, len(resp.GetResults()))}
	for _, result := range resp.GetResults() {
		item := ProtoSurfaceResult{
			Scenario: result.GetScenario(),
			Error:    result.GetError(),
		}
		if surface := result.GetSurface(); surface != nil {
			item.Surface = protoSurfaceFromProto(surface)
		}
		out.Results = append(out.Results, item)
	}
	return out
}

func protoSurfaceFromProto(surface *sharedv1.ProtoSurface) ProtoSurface {
	out := ProtoSurface{
		Scenario:       surface.GetScenario(),
		TransportWorld: normalizeTransportWorld(surface.GetTransportWorld().String()),
		Files:          make([]ProtoFile, 0, len(surface.GetFiles())),
	}
	for _, file := range surface.GetFiles() {
		out.Files = append(out.Files, ProtoFile{
			Path:      file.GetPath(),
			Stability: file.GetStability(),
		})
	}
	out.CrossScenarioImports = make([]ProtoImport, 0, len(surface.GetCrossScenarioImports()))
	for _, imp := range surface.GetCrossScenarioImports() {
		out.CrossScenarioImports = append(out.CrossScenarioImports, ProtoImport{
			FromFile:     imp.GetFromFile(),
			ToFile:       imp.GetToFile(),
			FromScenario: imp.GetFromScenario(),
			ToScenario:   imp.GetToScenario(),
			FromPackage:  imp.GetFromPackage(),
			ToPackage:    imp.GetToPackage(),
		})
	}
	return out
}

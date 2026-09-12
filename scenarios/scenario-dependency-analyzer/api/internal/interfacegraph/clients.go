package interfacegraph

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	factsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts/facts_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/shared"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/validation/validation_v1connect"
)

const (
	protoHealthScenario = "proto-health"
	codeFactsScenario   = "code-facts"
)

type URLResolver interface {
	ResolveScenarioURLDefault(ctx context.Context, scenarioSlug string) (string, error)
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

type CodeFactsClient struct {
	resolver   URLResolver
	httpClient connect.HTTPClient
}

func NewCodeFactsClient(resolver URLResolver, httpClient connect.HTTPClient) *CodeFactsClient {
	if resolver == nil {
		resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &CodeFactsClient{resolver: resolver, httpClient: httpClient}
}

func (c *CodeFactsClient) DescribeFleetImports(ctx context.Context, req ImportFactsRequest) (*ImportFactsResponse, error) {
	baseURL, err := c.resolver.ResolveScenarioURLDefault(ctx, codeFactsScenario)
	if err != nil {
		return nil, err
	}
	client := factsconnect.NewCodeFactsServiceClient(c.httpClient, baseURL)
	resp, err := client.DescribeFleetImports(ctx, connect.NewRequest(&factsv1.DescribeFleetImportsRequest{
		Scenarios:      req.Scenarios,
		Limit:          req.Limit,
		UseCache:       req.UseCache,
		RepoRoot:       req.RepoRoot,
		LanguageFilter: req.LanguageFilter,
	}))
	if err != nil {
		return nil, err
	}
	return importFactsResponseFromProto(resp.Msg), nil
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
			item.TransportWorld = item.Surface.TransportWorld
		}
		out.Results = append(out.Results, item)
	}
	return out
}

func protoSurfaceFromProto(surface *sharedv1.ProtoSurface) ProtoSurface {
	out := ProtoSurface{
		Scenario:       surface.GetScenario(),
		TransportWorld: surface.GetTransportWorld().String(),
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

func importFactsResponseFromProto(resp *factsv1.DescribeFleetImportsResponse) *ImportFactsResponse {
	if resp == nil {
		return &ImportFactsResponse{}
	}
	out := &ImportFactsResponse{Results: make([]ImportFactsResult, 0, len(resp.GetResults()))}
	for _, result := range resp.GetResults() {
		item := ImportFactsResult{
			Scenario: result.GetScenario(),
			Error:    result.GetError(),
		}
		if report := result.GetReport(); report != nil {
			item.Facts = importFactsFromProto(report.GetFacts())
		}
		out.Results = append(out.Results, item)
	}
	return out
}

func importFactsFromProto(facts []*factsv1.GenericFact) []ImportFact {
	out := make([]ImportFact, 0, len(facts))
	for _, fact := range facts {
		if fact.GetFamily() != factsv1.FactFamily_FACT_FAMILY_IMPORTS {
			continue
		}
		attrs := fact.GetAttributes()
		item := ImportFact{
			ImportPath: firstNonEmpty(attrs["import_path"], attrs["source_module"], fact.GetSubject()),
			Path:       attrs["path"],
			Language:   attrs["language"],
			Analyzer:   attrs["analyzer"],
		}
		if item.Path == "" {
			for _, ev := range fact.GetEvidence() {
				if ev.GetRange().GetFile() != "" {
					item.Path = ev.GetRange().GetFile()
					break
				}
			}
		}
		out = append(out, item)
	}
	return out
}

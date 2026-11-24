package facts

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/gograph"
	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	tsgraphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph"
	tsgraphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph/graph_v1connect"
)

const (
	goProviderScenario = "go-code-graph"
	tsProviderScenario = "typescript-code-graph"
)

type URLResolver interface {
	ResolveScenarioURLDefault(ctx context.Context, scenarioSlug string) (string, error)
}

type GraphProvider interface {
	Language() string
	AnalyzerName() string
	ExtractForFamilies(ctx context.Context, unit *factsv1.ParseUnit, families []factsv1.FactFamily) (*factsv1.LanguageGraph, error)
}

type Broker struct {
	providers map[string]GraphProvider
}

func NewBroker(providers ...GraphProvider) *Broker {
	b := &Broker{providers: map[string]GraphProvider{}}
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		b.providers[provider.Language()] = provider
	}
	return b
}

func (b *Broker) Provider(language string) GraphProvider {
	if b == nil {
		return nil
	}
	return b.providers[language]
}

type ProviderUnavailableError struct {
	Analyzer string
	Err      error
}

func (e ProviderUnavailableError) Error() string {
	if e.Err == nil {
		return e.Analyzer + " unavailable"
	}
	return e.Analyzer + " unavailable: " + e.Err.Error()
}

func (e ProviderUnavailableError) Unwrap() error { return e.Err }

type ProviderUnsupportedError struct {
	Analyzer string
	Err      error
}

func (e ProviderUnsupportedError) Error() string {
	if e.Err == nil {
		return e.Analyzer + " unsupported"
	}
	return e.Analyzer + " unsupported: " + e.Err.Error()
}

func (e ProviderUnsupportedError) Unwrap() error { return e.Err }

func errNoProvider(language string) ProviderUnavailableError {
	return ProviderUnavailableError{Analyzer: "code-facts." + language, Err: fmt.Errorf("no graph provider for language %q", language)}
}

type goGraphProvider struct {
	service *gograph.Service
}

func NewGoGraphProvider() GraphProvider {
	cache := gograph.NewMemoryExtractionCache()
	service := gograph.NewServiceWithCache(gograph.NewPackagesLoader(), gograph.NewPathMutex(), gograph.DefaultMaxConcurrentExtracts, cache)
	return &goGraphProvider{service: service}
}

func (p *goGraphProvider) Language() string     { return "go" }
func (p *goGraphProvider) AnalyzerName() string { return goProviderScenario }

func (p *goGraphProvider) ExtractForFamilies(ctx context.Context, unit *factsv1.ParseUnit, families []factsv1.FactFamily) (*factsv1.LanguageGraph, error) {
	return p.extract(ctx, unit, goExtractionProfile(families))
}

func (p *goGraphProvider) extract(ctx context.Context, unit *factsv1.ParseUnit, profile gograph.ExtractionProfile) (*factsv1.LanguageGraph, error) {
	if p.service == nil {
		return nil, ProviderUnavailableError{Analyzer: p.AnalyzerName(), Err: errors.New("missing shared Go graph service")}
	}
	started := time.Now()
	graph, warnings, err := p.service.Extract(ctx, gograph.ExtractInput{
		ModulePath: unit.GetRootPath(), Profile: profile, PackagePatterns: unit.GetPackagePatterns(),
	})
	if err != nil {
		var extractErr gograph.ExtractError
		if errors.As(err, &extractErr) && extractErr.Kind == gograph.ExtractErrorWorkspaceUnsupported {
			return nil, ProviderUnsupportedError{Analyzer: p.AnalyzerName(), Err: err}
		}
		return nil, err
	}
	result := &factsv1.LanguageGraph{
		Graph: gograph.ToProtoGraph(graph), Warnings: gograph.WarningsToProto(warnings),
		GraphHash: gograph.GraphHash(graph), ExtractionMs: time.Since(started).Milliseconds(),
		ExtractionProfile: string(profile),
	}
	for _, omission := range profile.OmittedInformation() {
		result.OmittedInformation = append(result.OmittedInformation, &factsv1.GraphInformationOmission{
			Capability: omission.Capability, Reason: omission.Reason,
		})
	}
	return result, nil
}

// goExtractionProfile returns the cheapest profile that can produce every
// requested Code Facts family. Imports, declarations, and proto-adoption need
// syntax only. References, calls, and endpoint proof synthesis require
// resolved semantic usage facts. Code Facts has no test-only fact family, so
// it never needs the full compatibility profile.
func goExtractionProfile(families []factsv1.FactFamily) gograph.ExtractionProfile {
	for _, family := range families {
		switch family {
		case factsv1.FactFamily_FACT_FAMILY_REFERENCES,
			factsv1.FactFamily_FACT_FAMILY_CALLS,
			factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS:
			return gograph.ExtractionProfileSemantic
		}
	}
	return gograph.ExtractionProfileStructural
}

func providerExtractionProfile(provider GraphProvider, families []factsv1.FactFamily) string {
	if provider == nil {
		return "default"
	}
	switch provider.Language() {
	case "go":
		return string(goExtractionProfile(families))
	case "typescript", "javascript":
		if importsOnlyFamilies(families) {
			return "typescript-imports-only-v1"
		}
		// The provider now emits normalized body identity, includes owned JS
		// sources in TS projects, and discovers standalone JavaScript packages.
		// Version this contract so cached pre-identity graphs cannot mask the
		// new detector evidence or source boundary.
		return "typescript-owned-source-body-identity-js-v1"
	}
	return "default"
}

type tsGraphProvider struct {
	resolver   URLResolver
	httpClient connect.HTTPClient
	language   string
}

func NewTypeScriptGraphProvider(resolver URLResolver, httpClient connect.HTTPClient) GraphProvider {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &tsGraphProvider{resolver: resolver, httpClient: httpClient, language: "typescript"}
}

func NewJavaScriptGraphProvider(resolver URLResolver, httpClient connect.HTTPClient) GraphProvider {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &tsGraphProvider{resolver: resolver, httpClient: httpClient, language: "javascript"}
}

func (p *tsGraphProvider) Language() string     { return p.language }
func (p *tsGraphProvider) AnalyzerName() string { return tsProviderScenario }

func (p *tsGraphProvider) ExtractForFamilies(ctx context.Context, unit *factsv1.ParseUnit, families []factsv1.FactFamily) (*factsv1.LanguageGraph, error) {
	profile := "full"
	if importsOnlyFamilies(families) {
		profile = "imports"
	}
	return p.extract(ctx, unit, profile)
}

func importsOnlyFamilies(families []factsv1.FactFamily) bool {
	if len(families) == 0 {
		return false
	}
	for _, family := range families {
		if family != factsv1.FactFamily_FACT_FAMILY_IMPORTS {
			return false
		}
	}
	return true
}

func (p *tsGraphProvider) extract(ctx context.Context, unit *factsv1.ParseUnit, profile string) (*factsv1.LanguageGraph, error) {
	if p.resolver == nil {
		return nil, ProviderUnavailableError{Analyzer: p.AnalyzerName(), Err: errors.New("missing URL resolver")}
	}
	baseURL, err := p.resolver.ResolveScenarioURLDefault(ctx, tsProviderScenario)
	if err != nil {
		return nil, ProviderUnavailableError{Analyzer: p.AnalyzerName(), Err: err}
	}
	resp, err := tsgraphconnect.NewTypeScriptCodeGraphServiceClient(p.httpClient, baseURL).Extract(ctx, connect.NewRequest(&tsgraphv1.ExtractRequest{
		ProjectPath: graphProjectPath(p.language, unit),
		Profile:     profile,
	}))
	if err != nil {
		return nil, classifyProviderError(p.AnalyzerName(), err)
	}
	return &factsv1.LanguageGraph{
		Graph:              resp.Msg.GetGraph(),
		Warnings:           resp.Msg.GetWarnings(),
		GraphHash:          resp.Msg.GetGraphHash(),
		ExtractionMs:       resp.Msg.GetExtractionMs(),
		ExtractionProfile:  resp.Msg.GetExtractionProfile(),
		OmittedInformation: extractionProfileOmissions(profile),
	}, nil
}

func extractionProfileOmissions(profile string) []*factsv1.GraphInformationOmission {
	if profile != "imports" {
		return nil
	}
	return []*factsv1.GraphInformationOmission{
		{Capability: "declarations", Reason: "imports-only extraction profile"},
		{Capability: "usage_facts", Reason: "imports-only extraction profile"},
		{Capability: "type_diagnostics", Reason: "imports-only extraction profile"},
	}
}

func graphProjectPath(language string, unit *factsv1.ParseUnit) string {
	if language == "javascript" {
		return unit.GetRootPath()
	}
	return unit.GetConfigPath()
}

func classifyProviderError(analyzer string, err error) error {
	var ce *connect.Error
	if errors.As(err, &ce) {
		switch ce.Code() {
		case connect.CodeUnavailable, connect.CodeDeadlineExceeded:
			return ProviderUnavailableError{Analyzer: analyzer, Err: err}
		case connect.CodeUnimplemented:
			return ProviderUnsupportedError{Analyzer: analyzer, Err: err}
		}
	}
	if strings.Contains(strings.ToLower(err.Error()), "connection refused") {
		return ProviderUnavailableError{Analyzer: analyzer, Err: err}
	}
	if strings.Contains(strings.ToLower(err.Error()), "workspace_unsupported") {
		return ProviderUnsupportedError{Analyzer: analyzer, Err: err}
	}
	return err
}

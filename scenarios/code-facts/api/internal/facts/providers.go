package facts

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	gographv1 "github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph"
	gographconnect "github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph/graph_v1connect"
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
	Extract(ctx context.Context, unit *factsv1.ParseUnit) (*factsv1.LanguageGraph, error)
}

// familyAwareGraphProvider is an optional provider extension used when the
// upstream graph API supports cheaper extraction profiles. Keeping it
// optional preserves the provider seam for TypeScript and test doubles.
type familyAwareGraphProvider interface {
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

func (b *Broker) Analyze(ctx context.Context, unit *factsv1.ParseUnit) (*factsv1.LanguageGraph, GraphProvider, error) {
	if b == nil {
		return nil, nil, errNoProvider(unit.GetLanguage())
	}
	provider := b.Provider(unit.GetLanguage())
	if provider == nil {
		return nil, nil, errNoProvider(unit.GetLanguage())
	}
	result, err := provider.Extract(ctx, unit)
	return result, provider, err
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
	resolver   URLResolver
	httpClient connect.HTTPClient
}

func NewGoGraphProvider(resolver URLResolver, httpClient connect.HTTPClient) GraphProvider {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &goGraphProvider{resolver: resolver, httpClient: httpClient}
}

func (p *goGraphProvider) Language() string     { return "go" }
func (p *goGraphProvider) AnalyzerName() string { return goProviderScenario }

func (p *goGraphProvider) Extract(ctx context.Context, unit *factsv1.ParseUnit) (*factsv1.LanguageGraph, error) {
	return p.extract(ctx, unit, gographv1.ExtractionProfile_EXTRACTION_PROFILE_FULL)
}

func (p *goGraphProvider) ExtractForFamilies(ctx context.Context, unit *factsv1.ParseUnit, families []factsv1.FactFamily) (*factsv1.LanguageGraph, error) {
	return p.extract(ctx, unit, goExtractionProfile(families))
}

func (p *goGraphProvider) extract(ctx context.Context, unit *factsv1.ParseUnit, profile gographv1.ExtractionProfile) (*factsv1.LanguageGraph, error) {
	if p.resolver == nil {
		return nil, ProviderUnavailableError{Analyzer: p.AnalyzerName(), Err: errors.New("missing URL resolver")}
	}
	baseURL, err := p.resolver.ResolveScenarioURLDefault(ctx, goProviderScenario)
	if err != nil {
		return nil, ProviderUnavailableError{Analyzer: p.AnalyzerName(), Err: err}
	}
	resp, err := gographconnect.NewGoCodeGraphServiceClient(p.httpClient, baseURL).Extract(ctx, connect.NewRequest(&gographv1.ExtractRequest{
		ModulePath:      unit.GetRootPath(),
		Profile:         profile,
		PackagePatterns: unit.GetPackagePatterns(),
	}))
	if err != nil {
		return nil, classifyProviderError(p.AnalyzerName(), err)
	}
	result := &factsv1.LanguageGraph{
		Graph: resp.Msg.GetGraph(), Warnings: resp.Msg.GetWarnings(),
		GraphHash: resp.Msg.GetGraphHash(), ExtractionMs: resp.Msg.GetExtractionMs(),
		ExtractionProfile: strings.ToLower(strings.TrimPrefix(resp.Msg.GetProfile().String(), "EXTRACTION_PROFILE_")),
	}
	for _, omission := range resp.Msg.GetOmittedInformation() {
		result.OmittedInformation = append(result.OmittedInformation, &factsv1.GraphInformationOmission{
			Capability: omission.GetCapability(), Reason: omission.GetReason(),
		})
	}
	return result, nil
}

// goExtractionProfile returns the cheapest profile that can produce every
// requested Code Facts family. Imports, declarations, and proto-adoption need
// syntax only. References, calls, and endpoint proof synthesis require
// resolved semantic usage facts. Code Facts has no test-only fact family, so
// it never needs the full compatibility profile.
func goExtractionProfile(families []factsv1.FactFamily) gographv1.ExtractionProfile {
	for _, family := range families {
		switch family {
		case factsv1.FactFamily_FACT_FAMILY_REFERENCES,
			factsv1.FactFamily_FACT_FAMILY_CALLS,
			factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS:
			return gographv1.ExtractionProfile_EXTRACTION_PROFILE_SEMANTIC
		}
	}
	return gographv1.ExtractionProfile_EXTRACTION_PROFILE_STRUCTURAL
}

func extractForFamilies(ctx context.Context, provider GraphProvider, unit *factsv1.ParseUnit, families []factsv1.FactFamily) (*factsv1.LanguageGraph, error) {
	if aware, ok := provider.(familyAwareGraphProvider); ok {
		return aware.ExtractForFamilies(ctx, unit, families)
	}
	return provider.Extract(ctx, unit)
}

func providerExtractionProfile(provider GraphProvider, families []factsv1.FactFamily) string {
	if provider == nil {
		return "default"
	}
	switch provider.Language() {
	case "go":
		return goExtractionProfile(families).String()
	case "typescript", "javascript":
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

func (p *tsGraphProvider) Extract(ctx context.Context, unit *factsv1.ParseUnit) (*factsv1.LanguageGraph, error) {
	if p.resolver == nil {
		return nil, ProviderUnavailableError{Analyzer: p.AnalyzerName(), Err: errors.New("missing URL resolver")}
	}
	baseURL, err := p.resolver.ResolveScenarioURLDefault(ctx, tsProviderScenario)
	if err != nil {
		return nil, ProviderUnavailableError{Analyzer: p.AnalyzerName(), Err: err}
	}
	resp, err := tsgraphconnect.NewTypeScriptCodeGraphServiceClient(p.httpClient, baseURL).Extract(ctx, connect.NewRequest(&tsgraphv1.ExtractRequest{
		ProjectPath: graphProjectPath(p.language, unit),
	}))
	if err != nil {
		return nil, classifyProviderError(p.AnalyzerName(), err)
	}
	return &factsv1.LanguageGraph{
		Graph:        resp.Msg.GetGraph(),
		Warnings:     resp.Msg.GetWarnings(),
		GraphHash:    resp.Msg.GetGraphHash(),
		ExtractionMs: resp.Msg.GetExtractionMs(),
	}, nil
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

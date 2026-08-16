// Package focus is the API handler for the FocusService — the gaps registry +
// prioritization domain. It is the proto translation edge over internal/focus;
// all business logic lives in internal/focus behind seams.
package focus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	internalcoverage "meta-optimization-manager/internal/coverage"
	internalfocus "meta-optimization-manager/internal/focus"
	"meta-optimization-manager/internal/module"
	internaltrials "meta-optimization-manager/internal/trials"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/spacedoc"

	focusconnect "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/focus/focus_v1connect"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	metricsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/metrics"
	metricsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/metrics/metrics_v1connect"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry/registry_v1connect"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing/routing_v1connect"
)

// Module returns the focus domain's contribution to the API: the generated
// FocusService Connect-RPC handler, backed by the SQLite gaps registry and a
// live GapSource derived from the owner space docs. The space reader is shared
// with the coverage domain (the same cross-scenario read seam) — wired here at
// the production edge, never imported into internal/focus.
func Module(db *database.RoutedDB, clk schedule.Clock, logger *log.Logger) module.Module {
	trialsRepo := internaltrials.NewSQLiteRepository(db, clk)
	spaceReader := internalcoverage.NewSpaceReader()
	actJoiner := internalcoverage.NewNumeratorJoiner()
	coverageService := internalcoverage.NewService(internalcoverage.Deps{Reader: spaceReader, Joiner: actJoiner, Clock: clk})
	insights := searchHubInsightsReader{
		resolver: discovery.NewResolver(discovery.ResolverConfig{}),
		http:     &http.Client{Timeout: 3 * time.Second},
	}
	maturityHTTP := &http.Client{Timeout: 30 * time.Second}
	conditionPopulation := func(ctx context.Context) (map[string]struct{}, error) {
		population := make(map[string]struct{})
		for _, projection := range []internalcoverage.Projection{internalcoverage.ProjectionAct} {
			definition, err := spaceReader.Read(ctx, projection)
			if err != nil {
				return nil, err
			}
			join := actJoiner.Join(ctx, projection, definition.Cells)
			if !join.Available {
				return nil, fmt.Errorf("%s numerator unavailable: %s", projection, join.Reason)
			}
			for _, cell := range definition.Cells {
				if join.Statuses[cell.ID] != spacedoc.StatusNow {
					continue
				}
				for _, provider := range strings.FieldsFunc(cell.Owner, func(r rune) bool { return r == '+' || r == ',' }) {
					provider = strings.TrimSpace(strings.SplitN(provider, "(", 2)[0])
					if provider != "" {
						population[strings.ToLower(provider)] = struct{}{}
					}
				}
			}
		}
		return population, nil
	}
	programConditionReader := internalfocus.NewProgramRuntimeConditionReader(conditionPopulation)
	substrateSource := internalfocus.NewSubstrateGapSource(searchHubSubstrateReader(insights).Read)
	conditionSource := internalfocus.NewMultiGapSource([]internalfocus.NamedGapSource{
		{Name: "search-hub", Source: internalfocus.NewConditionGapSourceWithPopulation(insights, conditionPopulation)},
		{Name: "search-hub-incubating", Source: internalfocus.NewIncubatingGapSource(func(ctx context.Context) ([]internalfocus.IncubatingProvider, error) {
			base, err := insights.resolver.ResolveScenarioURLDefault(ctx, "search-hub")
			if err != nil {
				return nil, err
			}
			resp, err := routingconnect.NewRoutingServiceClient(insights.http, base).Status(ctx, connect.NewRequest(&routingv1.StatusRequest{}))
			if err != nil {
				return nil, err
			}
			out := make([]internalfocus.IncubatingProvider, 0, len(resp.Msg.GetIncubating()))
			for _, provider := range resp.Msg.GetIncubating() {
				if provider == nil {
					continue
				}
				out = append(out, internalfocus.IncubatingProvider{ProviderID: provider.GetProviderId(), DeclaredAt: provider.GetDeclaredAt(), NextAction: provider.GetNextAction()})
			}
			return out, nil
		})},
		{Name: "search-hub-federation", Source: internalfocus.NewFederationHealthGapSource(insights)},
		{Name: "condition/program-runtime", Source: internalfocus.NewProgramRuntimeConditionGapSource(programConditionReader)},
		{Name: "maturity", Source: internalfocus.NewMaturityGapSource(searchHubMaturityReader{resolver: insights.resolver, http: maturityHTTP})},
		{Name: "search-hub-router-quality", Source: internalfocus.NewRouterQualityGapSource(func(ctx context.Context) ([]internalfocus.RouterQualityFinding, error) {
			report, err := coverageService.ValidateBaseDocs(ctx, internalcoverage.ProjectionAnswer)
			if err != nil {
				return nil, err
			}
			definition, err := spaceReader.Read(ctx, internalcoverage.ProjectionAnswer)
			if err != nil {
				return nil, err
			}
			owners := make(map[string]string, len(definition.Cells))
			for _, cell := range definition.Cells {
				owners[cell.ID] = cell.Owner
			}
			findings := make([]internalfocus.RouterQualityFinding, 0)
			for _, issue := range report.Issues {
				if issue.Code != "router_quality_debt" {
					continue
				}
				cellID := strings.TrimSpace(issue.Location)
				findings = append(findings, internalfocus.RouterQualityFinding{
					Projection: internalfocus.ProjectionAnswer,
					CellID:     cellID,
					Owner:      owners[cellID], Message: issue.Message,
					Locator: "coverage://validate-docs/answer/" + cellID,
				})
			}
			return findings, nil
		})},
		{Name: "search-hub-substrate", Source: substrateSource},
	})
	svc := internalfocus.NewService(internalfocus.Deps{
		Source: internalfocus.NewMultiGapSource([]internalfocus.NamedGapSource{
			{Name: "coverage", Source: internalfocus.NewSpaceGapSourceWithLiveJoin(spaceReader, func(ctx context.Context, p internalcoverage.Projection, def *spacedoc.SpaceDefinition) (map[string]spacedoc.CellStatus, error) {
				join := actJoiner.Join(ctx, p, def.Cells)
				if !join.Available {
					return nil, fmt.Errorf("%s numerator unavailable: %s", p, join.Reason)
				}
				return join.Statuses, nil
			})},
			{Name: "trials", Source: internalfocus.NewEmpiricalGapSource(trialsRepo)},
			{Name: "agent-manager", Source: internalfocus.NewAgentManagerGapSource(internalfocus.NewAgentManagerFindingReader())},
			{Name: "durability", Source: internalfocus.NewDurabilityGapSource(internalfocus.NewAgentManagerDurabilityReader())},
			{Name: "program-runtime", Source: internalfocus.NewProgramRuntimeGapSource(internalfocus.NewProgramRuntimeFrictionReader())},
			{Name: "test-genie", Source: internalfocus.NewTestGenieGapSource()},
			{Name: "condition", Source: conditionSource},
		}),
		Repo:            internalfocus.NewSQLiteRepository(db, clk),
		Insights:        insights,
		ConditionSource: conditionSource,
		ConditionReader: programConditionReader,
	})
	connectPath, connectHandler := focusconnect.NewFocusServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "focus",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

type searchHubSubstrateReader struct {
	resolver interface {
		ResolveScenarioURLDefault(context.Context, string) (string, error)
	}
	http connect.HTTPClient
}

type healthDependency struct {
	Connected bool   `json:"connected"`
	Error     string `json:"error"`
}

type searchHubHealth struct {
	Dependencies map[string]healthDependency `json:"dependencies"`
}

func (r searchHubSubstrateReader) Read(ctx context.Context) ([]internalfocus.SubstrateObservation, error) {
	if r.resolver == nil || r.http == nil {
		return nil, fmt.Errorf("search-hub substrate reader is not configured")
	}
	base, err := r.resolver.ResolveScenarioURLDefault(ctx, "search-hub")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/health", nil)
	if err != nil {
		return nil, err
	}
	resp, err := r.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("search-hub health returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var health searchHubHealth
	if err := json.Unmarshal(body, &health); err != nil {
		return nil, fmt.Errorf("decode search-hub health: %w", err)
	}
	out := make([]internalfocus.SubstrateObservation, 0, 3)
	for _, name := range []string{"ollama", "qdrant"} {
		dependency, ok := health.Dependencies[name]
		reason := strings.TrimSpace(dependency.Error)
		if !ok {
			reason = "dependency was not reported by search-hub health"
		}
		out = append(out, internalfocus.SubstrateObservation{Name: name, Healthy: ok && dependency.Connected, Reason: reason, Locator: "search-hub://health/" + name})
	}
	routingStatus, err := routingconnect.NewRoutingServiceClient(r.http, base).Status(ctx, connect.NewRequest(&routingv1.StatusRequest{}))
	if err != nil {
		out = append(out, internalfocus.SubstrateObservation{Name: "reranker", Reason: "routing status unavailable: " + err.Error(), Locator: "search-hub://routing/status"})
		return out, nil
	}
	rerankerReason := ""
	if !routingStatus.Msg.GetRerankerAvailable() {
		rerankerReason = "reranker leg unavailable"
	}
	out = append(out, internalfocus.SubstrateObservation{Name: "reranker", Healthy: routingStatus.Msg.GetRerankerAvailable(), Reason: rerankerReason, Locator: "search-hub://routing/status"})
	return out, nil
}

type searchHubMaturityReader struct {
	resolver interface {
		ResolveScenarioURLDefault(context.Context, string) (string, error)
	}
	http connect.HTTPClient
}

func (r searchHubMaturityReader) Maturity(ctx context.Context) ([]internalfocus.MaturityObservation, error) {
	if r.resolver == nil || r.http == nil {
		return nil, fmt.Errorf("search-hub registry reader is not configured")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	base, err := r.resolver.ResolveScenarioURLDefault(ctx, "search-hub")
	if err != nil {
		return nil, err
	}
	registryResp, err := registryconnect.NewRegistryServiceClient(r.http, base).ListMaturityTargets(ctx, connect.NewRequest(&registryv1.ListMaturityTargetsRequest{}))
	if err != nil {
		return nil, err
	}
	providers := make(map[string]struct{}, len(registryResp.Msg.GetTargets()))
	for _, target := range registryResp.Msg.GetTargets() {
		if target == nil {
			continue
		}
		if scenario := strings.TrimSpace(target.GetScenario()); scenario != "" {
			providers[scenario] = struct{}{}
		}
	}
	scenarios := make([]string, 0, len(providers))
	for scenario := range providers {
		scenarios = append(scenarios, scenario)
	}
	sort.Strings(scenarios)
	if len(scenarios) == 0 {
		return nil, nil
	}
	client := scenariovalidationconnect.NewScenarioValidationServiceClient(r.http, base)
	observations := make([]internalfocus.MaturityObservation, len(scenarios))
	valid := make([]bool, len(scenarios))
	errs := make([]error, len(scenarios))
	jobs := make(chan int)
	var wg sync.WaitGroup
	workers := 6
	if len(scenarios) < workers {
		workers = len(scenarios)
	}
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				scenario := scenarios[index]
				resp, callErr := client.ValidateScenario(ctx, connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
					Scenario:         scenario,
					IncludeExecution: true,
				}))
				if callErr != nil {
					errs[index] = callErr
					continue
				}
				assessment := resp.Msg.GetAssessment()
				if assessment == nil || assessment.GetLocal() == nil || len(assessment.GetLocal().GetBlockingFindingCodes()) == 0 {
					valid[index] = true
					continue
				}
				blocking := make(map[string]struct{}, len(assessment.GetLocal().GetBlockingFindingCodes()))
				for _, code := range assessment.GetLocal().GetBlockingFindingCodes() {
					blocking[code] = struct{}{}
				}
				findings := make([]internalfocus.MaturityFinding, 0, len(blocking))
				for _, finding := range assessment.GetFindings() {
					if finding == nil {
						continue
					}
					if _, ok := blocking[finding.GetCode()]; !ok {
						continue
					}
					fixClass := strings.TrimSpace(finding.GetFixClass())
					if fixClass == "" {
						// Older Search Hub assessments omit the optional field for
						// execution failures; the actionable fallback is manual.
						fixClass = "manual"
					}
					findings = append(findings, internalfocus.MaturityFinding{
						Code:          finding.GetCode(),
						Message:       finding.GetMessage(),
						Location:      finding.GetLocation(),
						Remediation:   finding.GetRemediation(),
						FixClass:      fixClass,
						RepairCommand: "search-hub maturity fix " + scenario + " --apply",
					})
				}
				seenFindings := make(map[string]struct{}, len(findings))
				for _, finding := range findings {
					seenFindings[finding.Code] = struct{}{}
				}
				for _, code := range assessment.GetLocal().GetBlockingFindingCodes() {
					if _, ok := seenFindings[code]; ok {
						continue
					}
					// Preserve the repair path even if an older Search Hub response
					// exposes only the blocking code and omits its finding detail.
					findings = append(findings, internalfocus.MaturityFinding{
						Code:          code,
						FixClass:      "manual",
						RepairCommand: "search-hub maturity fix " + scenario + " --apply",
					})
				}
				observations[index] = internalfocus.MaturityObservation{
					Scenario:      scenario,
					BlockingCodes: append([]string(nil), assessment.GetLocal().GetBlockingFindingCodes()...),
					Findings:      findings,
				}
				valid[index] = true
			}
		}()
	}
	for index := range scenarios {
		jobs <- index
	}
	close(jobs)
	wg.Wait()
	out := make([]internalfocus.MaturityObservation, 0, len(scenarios))
	var firstErr error
	for index := range scenarios {
		if errs[index] != nil && firstErr == nil {
			firstErr = fmt.Errorf("validate maturity target %q: %w", scenarios[index], errs[index])
		}
		if valid[index] && len(observations[index].BlockingCodes) > 0 {
			out = append(out, observations[index])
		}
	}
	return out, firstErr
}

// searchHubInsightsReader is the production transport seam for the focus
// domain's third ranking factor. The domain receives only its small internal
// ProviderInsight shape and remains agnostic about Connect-RPC and proto types.
type searchHubInsightsReader struct {
	resolver interface {
		ResolveScenarioURLDefault(context.Context, string) (string, error)
	}
	http connect.HTTPClient
}

func (r searchHubInsightsReader) Insights(ctx context.Context) ([]internalfocus.ProviderInsight, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	base, err := r.resolver.ResolveScenarioURLDefault(ctx, "search-hub")
	if err != nil {
		return nil, err
	}
	resp, err := metricsconnect.NewMetricsServiceClient(r.http, base).Insights(ctx, connect.NewRequest(&metricsv1.InsightsRequest{}))
	if err != nil {
		return nil, err
	}
	out := make([]internalfocus.ProviderInsight, 0, len(resp.Msg.GetProviders()))
	for _, provider := range resp.Msg.GetProviders() {
		if provider == nil {
			continue
		}
		out = append(out, internalfocus.ProviderInsight{
			ProviderID:      provider.GetProviderId(),
			ProviderGroup:   provider.GetProviderGroup(),
			TimesRouted:     provider.GetTimesRouted(),
			DegradationRate: provider.GetDegradationRate(),
		})
	}
	return out, nil
}

func (r searchHubInsightsReader) FederationHealth(ctx context.Context) ([]internalfocus.FederationHealthFinding, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	base, err := r.resolver.ResolveScenarioURLDefault(ctx, "search-hub")
	if err != nil {
		return nil, err
	}
	status, err := routingconnect.NewRoutingServiceClient(r.http, base).Status(ctx, connect.NewRequest(&routingv1.StatusRequest{}))
	if err != nil {
		return nil, err
	}
	insights, err := metricsconnect.NewMetricsServiceClient(r.http, base).Insights(ctx, connect.NewRequest(&metricsv1.InsightsRequest{WindowDays: 7}))
	if err != nil {
		return nil, err
	}
	out := make([]internalfocus.FederationHealthFinding, 0)
	for _, provider := range status.Msg.GetProviders() {
		if provider == nil {
			continue
		}
		if provider.GetStuck() {
			out = append(out, internalfocus.FederationHealthFinding{ID: provider.GetProviderId(), Kind: "stuck_provider", Value: "true", Evidence: provider.GetRecoveryState()})
		}
		if provider.GetTimesRouted() >= 5 && provider.GetTotalHits() == 0 {
			out = append(out, internalfocus.FederationHealthFinding{ID: provider.GetProviderId(), Kind: "zero_yield", Value: fmt.Sprintf("routes=%d hits=0", provider.GetTimesRouted()), Evidence: "provider has routed repeatedly without a hit"})
		}
	}
	if insights.Msg.GetLatencyP95Ms() > 2000 {
		out = append(out, internalfocus.FederationHealthFinding{ID: "fleet-latency", Kind: "p95_latency_ms", Value: fmt.Sprintf("%d", insights.Msg.GetLatencyP95Ms()), Evidence: "7-day federation p95 exceeds 2000ms budget"})
	}
	for _, provider := range insights.Msg.GetProviders() {
		if provider != nil && provider.GetDegradationRate() > 0.25 {
			out = append(out, internalfocus.FederationHealthFinding{ID: provider.GetProviderId(), Kind: "degradation_rate", Value: fmt.Sprintf("%.3f", provider.GetDegradationRate()), Evidence: "7-day provider degradation exceeds 25% budget"})
		}
	}
	return out, nil
}

// Schema re-exports internalfocus.Schema so the modules registry collects
// endpoints and schema from one symbol per handler package.
func Schema() string { return internalfocus.Schema() }

// Endpoints is the machine-readable description of the focus module's public
// surface. The Connect-RPC method paths reference the generated *Procedure
// constants, so renaming an RPC in focus.proto breaks this at compile time; the
// global parity test (registry_test.go) asserts every RPC has exactly one entry.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "focus_get",
		Path:        focusconnect.FocusServiceGetFocusProcedure,
		Method:      "POST",
		Summary:     "Ranked next-best gaps",
		Description: "Returns the ranked next-best gaps (impact × importance) with qualitative context (OT-P0-002). Optionally filtered by projection and capped by limit.",
		Category:    "focus",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"items": "array<FocusItem>"},
		},
	},
	{
		ID:          "focus_list_gaps",
		Path:        focusconnect.FocusServiceListGapsProcedure,
		Method:      "POST",
		Summary:     "List the gaps registry",
		Description: "Returns the full gaps registry (live-derived non-NOW cells overlaid with the owned registry), optionally filtered by projection/cell/status (OT-P0-003).",
		Category:    "focus",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"gaps": "array<Gap>"},
		},
	},
	{
		ID:          "focus_get_gap",
		Path:        focusconnect.FocusServiceGetGapProcedure,
		Method:      "POST",
		Summary:     "Get one gap",
		Description: "Returns one gap by id with its full qualitative context.",
		Category:    "focus",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"gap": "Gap"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No gap with that id"},
		},
	},
	{
		ID:          "focus_add_gap_note",
		Path:        focusconnect.FocusServiceAddGapNoteProcedure,
		Method:      "POST",
		Summary:     "Append an explored approach to a gap",
		Description: "Appends an explored-but-unbuilt approach to a gap — the one focus write verb (the 'store our thinking' primitive). Materializes a registry row for a derived gap.",
		Category:    "focus",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"gap": "Gap"},
		},
	},
	{
		ID: "focus_list_condition", Path: focusconnect.FocusServiceListConditionProcedure, Method: "POST",
		Summary: "Observed condition findings", Description: "Lists condition findings derived from serving telemetry.", Category: "focus",
		Response: &module.Schema{Type: "object", Properties: map[string]string{"gaps": "array<Gap>"}},
	},
	{
		ID: "focus_explain_condition", Path: focusconnect.FocusServiceExplainConditionProcedure, Method: "POST",
		Summary: "Explain condition leg", Description: "Explains one provider leg's observed condition evidence.", Category: "focus",
		Response: &module.Schema{Type: "object", Properties: map[string]string{"gap": "Gap"}},
	},
}

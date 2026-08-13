// Package focus is the API handler for the FocusService — the gaps registry +
// prioritization domain. It is the proto translation edge over internal/focus;
// all business logic lives in internal/focus behind seams.
package focus

import (
	"context"
	"fmt"
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
	conditionSource := internalfocus.NewMultiGapSource([]internalfocus.NamedGapSource{
		{Name: "search-hub", Source: internalfocus.NewConditionGapSourceWithPopulation(insights, conditionPopulation)},
		{Name: "condition/program-runtime", Source: internalfocus.NewProgramRuntimeConditionGapSource(programConditionReader)},
		{Name: "maturity", Source: internalfocus.NewMaturityGapSource(searchHubMaturityReader{resolver: insights.resolver, http: maturityHTTP})},
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

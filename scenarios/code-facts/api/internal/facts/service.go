package facts

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

type Service struct {
	broker       *Broker
	cache        CacheRepository
	fileDomains  FileDomainProvider
	search       SearchBackend
	admission    *WeightedAdmission
	graphFlights *graphFlightGroup
}

type ServiceOption func(*Service)

// SearchBackend is the production query boundary. Implementations must read
// derived index state only; source discovery and analysis belong to index
// construction or the explicit Describe APIs.
type SearchBackend interface {
	Search(context.Context, *factsv1.SearchRequest) (*factsv1.SearchResponse, error)
}

func NewService(opts ...ServiceOption) *Service {
	s := &Service{broker: NewBroker(), cache: NewMemoryCacheRepository(), admission: NewWeightedAdmission(16, 64, 2*time.Second), graphFlights: newGraphFlightGroup()}
	for _, opt := range opts {
		opt(s)
	}
	if s.cache == nil {
		s.cache = NewMemoryCacheRepository()
	}
	return s
}

func WithBroker(broker *Broker) ServiceOption {
	return func(s *Service) {
		s.broker = broker
	}
}

func WithCacheRepository(cache CacheRepository) ServiceOption {
	return func(s *Service) {
		s.cache = cache
	}
}

func WithFileDomainProvider(provider FileDomainProvider) ServiceOption {
	return func(s *Service) {
		s.fileDomains = provider
	}
}

func WithAdmission(admission *WeightedAdmission) ServiceOption {
	return func(s *Service) {
		if admission != nil {
			s.admission = admission
		}
	}
}

func WithSearchBackend(search SearchBackend) ServiceOption {
	return func(s *Service) { s.search = search }
}

func (s *Service) Describe(ctx context.Context, req *factsv1.DescribeCodeFactsRequest) (*factsv1.CodeFactsReport, error) {
	if err := validateTarget(req.GetTarget()); err != nil {
		return nil, err
	}
	target, err := resolveTarget(req.GetTarget())
	if err != nil {
		return nil, err
	}
	include := normalizeFamilies(req.GetInclude())
	parseUnits := filterParseUnits(discoverParseUnits(target), req.GetTarget().GetLanguageFilter())
	sourceHash, configHash := sourceFingerprint(target, parseUnits)
	reportPlan := reportCachePlan(req.GetTarget(), target, parseUnits, include, sourceHash, configHash, req.GetMaxDepth())
	if req.GetUseCache() {
		if paged, ok := s.cache.(PagedReportCache); ok && req.GetPageSize() > 0 {
			offset, err := pageOffset(req.GetPageToken())
			if err != nil {
				return nil, err
			}
			report, entry, hit, err := paged.GetReportPage(ctx, reportPlan.Key, offset, int(req.GetPageSize()))
			if err != nil {
				return nil, err
			}
			if hit {
				total := int(report.GetTotalFacts())
				if offset > total {
					return nil, fmt.Errorf("page_token offset %d exceeds total facts %d", offset, total)
				}
				report.Cache = entry.metadata("hit", "report cache reused for identical target, options, source/config hashes, and analyzer versions")
				end := offset + len(report.GetFacts())
				if end < total {
					report.NextPageToken = strconv.Itoa(end)
				}
				return report, nil
			}
		}
		report, entry, ok, err := s.cache.GetReport(ctx, reportPlan.Key)
		if err != nil {
			return nil, err
		}
		if ok {
			report.Cache = entry.metadata("hit", "report cache reused for identical target, options, source/config hashes, and analyzer versions")
			if report.TotalFacts == 0 {
				report.TotalFacts = int32(len(report.GetFacts()))
			}
			return pageReport(report, req.GetPageSize(), req.GetPageToken())
		}
	}

	report := &factsv1.CodeFactsReport{
		Target: target,
		Cache:  reportPlan.metadata(cacheState(req.GetUseCache()), cacheReason(req.GetUseCache())),
	}
	if hasFamily(include, factsv1.FactFamily_FACT_FAMILY_SURFACES) {
		report.Surfaces = discoverSurfaces(target)
	}
	if hasFamily(include, factsv1.FactFamily_FACT_FAMILY_PARSE_UNITS) {
		report.ParseUnits = parseUnits
	}
	analysisInclude := expandAnalyzerFamilies(include)
	analysisUnits := parseUnitsForDescribeAnalysis(parseUnits, include, req.GetTarget().GetLanguageFilter())
	facts, warnings, evidence, graphHash, err := s.analyze(ctx, target, analysisUnits, analysisInclude, sourceHash, configHash, req.GetUseCache())
	if err != nil {
		return nil, err
	}
	reportPlan.GraphHash = graphHash
	report.Cache.GraphHash = graphHash
	report.Facts = append(report.Facts, filterFactsForFamilies(facts, include)...)
	report.Warnings = append(report.Warnings, warnings...)
	report.Evidence = append(report.Evidence, evidence...)
	proofInput := s.proofInput(target, facts, warnings, evidence, report.Cache)
	if hasFamily(include, factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION) {
		proofFacts, proofEvidence, proofWarnings := synthesizeProtoAdoption(proofInput, nil)
		report.Facts = append(report.Facts, proofFacts...)
		report.Evidence = append(report.Evidence, proofEvidence...)
		report.Warnings = append(report.Warnings, proofWarnings...)
	}
	if hasFamily(include, factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS) {
		proofFacts, proofEvidence, proofWarnings := synthesizeEndpointProofs(proofInput, req.GetEndpointIds())
		report.Facts = append(report.Facts, proofFacts...)
		report.Evidence = append(report.Evidence, proofEvidence...)
		report.Warnings = append(report.Warnings, proofWarnings...)
	}
	if hasFamily(include, factsv1.FactFamily_FACT_FAMILY_FILE_DOMAIN) {
		fileDomainFacts, fileDomainEvidence, fileDomainWarnings, fileDomainErr := s.describeFileDomains(ctx, target)
		if fileDomainErr != nil {
			return nil, fileDomainErr
		}
		report.Facts = append(report.Facts, fileDomainFacts...)
		report.Evidence = append(report.Evidence, fileDomainEvidence...)
		report.Warnings = append(report.Warnings, fileDomainWarnings...)
	}
	for _, family := range include {
		if isImplementedFamily(family) {
			continue
		}
		report.Facts = append(report.Facts, unsupportedFact(family))
		report.Warnings = append(report.Warnings, unsupportedWarning(family))
	}
	report.TotalFacts = int32(len(report.GetFacts()))
	if err := s.cache.PutReport(ctx, reportPlan, report); err != nil {
		return nil, err
	}
	return pageReport(report, req.GetPageSize(), req.GetPageToken())
}

// pageReport keeps the cache entry whole while allowing callers to consume a
// large report in bounded responses. Tokens are deliberately opaque to the
// caller but encode only a validated decimal offset, so pagination remains
// deterministic and stateless across requests.
func pageReport(report *factsv1.CodeFactsReport, pageSize int32, pageToken string) (*factsv1.CodeFactsReport, error) {
	if report == nil || pageSize <= 0 {
		return report, nil
	}
	offset, err := pageOffset(pageToken)
	if err != nil {
		return nil, err
	}
	facts := report.GetFacts()
	if offset > len(facts) {
		return nil, fmt.Errorf("page_token offset %d exceeds total facts %d", offset, len(facts))
	}
	end := offset + int(pageSize)
	if end > len(facts) {
		end = len(facts)
	}
	report.Facts = facts[offset:end]
	if end < len(facts) {
		report.NextPageToken = strconv.Itoa(end)
	} else {
		report.NextPageToken = ""
	}
	return report, nil
}

func pageOffset(token string) (int, error) {
	if strings.TrimSpace(token) == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(token)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("invalid page_token %q", token)
	}
	return parsed, nil
}

// Search is the public query boundary. Production composition must provide the
// generation-fenced persistent backend; an unconfigured service fails closed
// instead of scanning source or constructing a private resident index.
func (s *Service) Search(ctx context.Context, req *factsv1.SearchRequest) (*factsv1.SearchResponse, error) {
	if s.search == nil {
		return nil, fmt.Errorf("indexed search backend is not configured")
	}
	return s.search.Search(ctx, req)
}

func (s *Service) DescribeFleetImports(ctx context.Context, req *factsv1.DescribeFleetImportsRequest) (*factsv1.DescribeFleetImportsResponse, error) {
	if req.GetLimit() < 0 || req.GetLimit() > 500 {
		return nil, fmt.Errorf("limit must be between 0 and 500")
	}
	repoRoot, err := resolveRepoRoot(req.GetRepoRoot())
	if err != nil {
		return nil, err
	}
	scenarios := normalizeScenarioList(req.GetScenarios())
	if len(scenarios) == 0 {
		release, admissionErr := s.admission.Acquire(ctx, "fleet", 0)
		if admissionErr != nil {
			return nil, admissionErr
		}
		scenarios, err = listScenarioSlugs(repoRoot)
		release()
		if err != nil {
			return nil, err
		}
	}
	if req.GetLimit() > 0 && len(scenarios) > int(req.GetLimit()) {
		scenarios = scenarios[:int(req.GetLimit())]
	}

	results := make([]*factsv1.CodeFactsResult, len(scenarios))
	workerLimit := min(16, max(1, runtime.NumCPU()))
	sem := make(chan struct{}, workerLimit)
	var wg sync.WaitGroup
	for i, scenario := range scenarios {
		i, scenario := i, scenario
		results[i] = &factsv1.CodeFactsResult{Scenario: scenario}
		select {
		case <-ctx.Done():
			results[i].Error = ctx.Err().Error()
			continue
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			report, err := s.Describe(ctx, &factsv1.DescribeCodeFactsRequest{
				Target: &factsv1.CodeTarget{
					Kind:           factsv1.TargetKind_TARGET_KIND_SCENARIO,
					Scenario:       scenario,
					RepoRoot:       repoRoot,
					LanguageFilter: req.GetLanguageFilter(),
				},
				Include:  []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS},
				UseCache: req.GetUseCache(),
			})
			if err != nil {
				results[i].Error = err.Error()
				return
			}
			results[i].Report = report
		}()
	}
	wg.Wait()
	resp := &factsv1.DescribeFleetImportsResponse{Results: results}
	return resp, nil
}

func (s *Service) Surfaces(_ context.Context, req *factsv1.ListSurfacesRequest) (*factsv1.ListSurfacesResponse, error) {
	if err := validateTarget(req.GetTarget()); err != nil {
		return nil, err
	}
	target, err := resolveTarget(req.GetTarget())
	if err != nil {
		return nil, err
	}
	return &factsv1.ListSurfacesResponse{
		Target:   target,
		Surfaces: discoverSurfaces(target),
		Cache:    surfaceCacheMetadata(req.GetTarget(), target),
	}, nil
}

func normalizeScenarioList(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, scenario := range in {
		scenario = strings.TrimSpace(scenario)
		if scenario == "" || seen[scenario] {
			continue
		}
		seen[scenario] = true
		out = append(out, scenario)
	}
	return out
}

func listScenarioSlugs(repoRoot string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(repoRoot, "scenarios"))
	if err != nil {
		return nil, fmt.Errorf("list scenarios: %w", err)
	}
	var out []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		slug := entry.Name()
		if hasServiceManifest(filepath.Join(repoRoot, "scenarios", slug)) {
			out = append(out, slug)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (s *Service) ProtoAdoption(ctx context.Context, req *factsv1.CheckProtoAdoptionRequest) (*factsv1.ProofReport, error) {
	if err := validateTarget(req.GetTarget()); err != nil {
		return nil, err
	}
	target, err := resolveTarget(req.GetTarget())
	if err != nil {
		return nil, err
	}
	input, err := s.analyzeForProof(ctx, req.GetTarget(), target, []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS}, req.GetUseCache())
	if err != nil {
		return nil, err
	}
	facts, evidence, warnings := synthesizeProtoAdoption(input, req.GetSurfaces())
	return &factsv1.ProofReport{
		Target:   target,
		Family:   factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION,
		Facts:    facts,
		Evidence: append(input.evidence, evidence...),
		Warnings: append(input.warnings, warnings...),
		Cache:    input.cache,
	}, nil
}

func (s *Service) EndpointProof(ctx context.Context, req *factsv1.CheckEndpointProofRequest) (*factsv1.ProofReport, error) {
	if err := validateTarget(req.GetTarget()); err != nil {
		return nil, err
	}
	target, err := resolveTarget(req.GetTarget())
	if err != nil {
		return nil, err
	}
	input, err := s.analyzeForProof(ctx, req.GetTarget(), target, []factsv1.FactFamily{
		factsv1.FactFamily_FACT_FAMILY_IMPORTS,
		factsv1.FactFamily_FACT_FAMILY_REFERENCES,
		factsv1.FactFamily_FACT_FAMILY_CALLS,
	}, req.GetUseCache(), endpointProofLanguages(req.GetTarget().GetLanguageFilter()))
	if err != nil {
		return nil, err
	}
	facts, evidence, warnings := synthesizeEndpointProofs(input, req.GetEndpointIds())
	return &factsv1.ProofReport{
		Target:   target,
		Family:   factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS,
		Facts:    facts,
		Evidence: append(input.evidence, evidence...),
		Warnings: append(input.warnings, warnings...),
		Cache:    input.cache,
	}, nil
}

func (s *Service) CacheStatus(_ context.Context, req *factsv1.GetCacheStatusRequest) (*factsv1.CacheStatus, error) {
	return s.cacheStatus(context.Background(), req.GetTarget(), "")
}

func (s *Service) InspectCache(_ context.Context, req *factsv1.InspectCacheRequest) (*factsv1.CacheStatus, error) {
	return s.cacheStatus(context.Background(), req.GetTarget(), strings.TrimSpace(req.GetCacheKey()))
}

func (s *Service) cacheStatus(ctx context.Context, targetReq *factsv1.CodeTarget, key string) (*factsv1.CacheStatus, error) {
	if err := validateTarget(targetReq); err != nil {
		return nil, err
	}
	target, err := resolveTarget(targetReq)
	if err != nil {
		return nil, err
	}
	entries, err := s.cache.Status(ctx, target.GetRootPath(), key)
	if err != nil {
		return nil, err
	}
	metadata := make([]*factsv1.CacheMetadata, 0, len(entries))
	for _, entry := range entries {
		metadata = append(metadata, entry.metadata("stored", "cache entry is reusable while key evidence remains unchanged"))
	}
	stats, err := s.cache.Stats(ctx)
	if err != nil {
		return nil, err
	}
	cacheKey := key
	if cacheKey == "" {
		cacheKey = cacheKeyForTarget(targetReq, target)
	}
	return cacheStatusResponse(targetReq, cacheKey, metadata, stats), nil
}

func (s *Service) ClearCache(ctx context.Context, req *factsv1.ClearCacheRequest) (*factsv1.ClearCacheResponse, error) {
	var targetRoot string
	cacheKey := "all"
	if !req.GetAll() {
		if err := validateTarget(req.GetTarget()); err != nil {
			return nil, err
		}
		target, err := resolveTarget(req.GetTarget())
		if err != nil {
			return nil, err
		}
		targetRoot = target.GetRootPath()
		cacheKey = cacheKeyForTarget(req.GetTarget(), target)
	}
	matched, cleared, err := s.cache.Clear(ctx, targetRoot, req.GetDryRun())
	if err != nil {
		return nil, err
	}
	return &factsv1.ClearCacheResponse{
		CacheKey:       cacheKey,
		MatchedEntries: matched,
		ClearedEntries: cleared,
		DryRun:         req.GetDryRun(),
	}, nil
}

func cacheStatusResponse(target *factsv1.CodeTarget, cacheKey string, metadata []*factsv1.CacheMetadata, stats CacheStats) *factsv1.CacheStatus {
	var utilization float64
	if stats.BudgetBytes > 0 {
		utilization = float64(stats.TotalPayloadBytes) / float64(stats.BudgetBytes)
	}
	scopes := make([]*factsv1.CacheScopeSummary, 0, len(stats.Scopes))
	for _, scope := range stats.Scopes {
		scopes = append(scopes, &factsv1.CacheScopeSummary{
			Scope:        scope.Scope,
			RowCount:     scope.Rows,
			PayloadBytes: scope.PayloadBytes,
		})
	}
	return &factsv1.CacheStatus{
		Target:            target,
		CacheKey:          cacheKey,
		Entries:           int64(len(metadata)),
		EntriesMetadata:   metadata,
		TotalRows:         stats.TotalRows,
		TotalPayloadBytes: stats.TotalPayloadBytes,
		BudgetBytes:       stats.BudgetBytes,
		Utilization:       utilization,
		Scopes:            scopes,
		LastSweepAtUnix:   stats.LastSweepAtUnix,
	}
}

func validateTarget(target *factsv1.CodeTarget) error {
	if target == nil {
		return fmt.Errorf("target is required")
	}
	switch target.GetKind() {
	case factsv1.TargetKind_TARGET_KIND_PATH:
		if strings.TrimSpace(target.GetPath()) == "" {
			return fmt.Errorf("target.path is required for %s", target.GetKind())
		}
	case factsv1.TargetKind_TARGET_KIND_SCENARIO:
		if strings.TrimSpace(target.GetScenario()) == "" {
			return fmt.Errorf("target.scenario is required for scenario targets")
		}
	case factsv1.TargetKind_TARGET_KIND_PROJECT, factsv1.TargetKind_TARGET_KIND_REPO, factsv1.TargetKind_TARGET_KIND_CONTROL_PLANE:
		// These kinds resolve from repo_root (or the governed process root).
	case factsv1.TargetKind_TARGET_KIND_PACKAGE:
		if strings.TrimSpace(target.GetPackageName()) == "" {
			return fmt.Errorf("target.package_name is required for package targets")
		}
	case factsv1.TargetKind_TARGET_KIND_UNSPECIFIED:
		return fmt.Errorf("target.kind is required")
	case factsv1.TargetKind_TARGET_KIND_MODULE, factsv1.TargetKind_TARGET_KIND_RESOURCE, factsv1.TargetKind_TARGET_KIND_TOOL, factsv1.TargetKind_TARGET_KIND_SAFEGUARD, factsv1.TargetKind_TARGET_KIND_DOCS, factsv1.TargetKind_TARGET_KIND_TEAM:
		return fmt.Errorf("target kind %s is unsupported", target.GetKind())
	default:
		return fmt.Errorf("target kind %s is unsupported", target.GetKind())
	}
	return nil
}

func (s *Service) analyze(ctx context.Context, target *factsv1.TargetContext, units []*factsv1.ParseUnit, include []factsv1.FactFamily, sourceHash string, configHash string, useCache bool) ([]*factsv1.GenericFact, []*factsv1.Warning, []*factsv1.Evidence, string, error) {
	if !needsAnalyzer(include) {
		return nil, nil, nil, "", nil
	}
	var facts []*factsv1.GenericFact
	var warnings []*factsv1.Warning
	var evidence []*factsv1.Evidence
	var graphHashes []string
	for _, unit := range units {
		if unit.GetStatus() != factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN {
			warnings = append(warnings, providerWarning("code-facts.analyzer-broker", unit.GetLanguage()+"_unsupported", firstUnitMessage(unit), factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED))
			continue
		}
		provider := s.broker.Provider(unit.GetLanguage())
		if provider == nil {
			unavailable := errNoProvider(unit.GetLanguage())
			if !target.GetRequested().GetStrict() {
				warnings = append(warnings, providerWarning(unavailable.Analyzer, "unavailable", unavailable.Error(), factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN))
				evidence = append(evidence, &factsv1.Evidence{
					Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN,
					Confidence: 0,
					Analyzer:   unavailable.Analyzer,
					Message:    unavailable.Error(),
				})
				continue
			}
			return nil, nil, nil, "", unavailable
		}
		// Graph entries are scoped to one parse unit. A fleet/report-wide source
		// hash would invalidate every language surface when one file changes,
		// defeating the cache's purpose for incremental edits.
		unitSourceHash, unitConfigHash := sourceFingerprintForUnit(unit)
		extractionProfile := providerExtractionProfile(provider, include)
		graphPlan := graphCachePlan(target, unit, provider, extractionProfile, unitSourceHash, unitConfigHash)
		var result *GraphResult
		if useCache {
			cached, entry, ok, err := s.cache.GetGraph(ctx, graphPlan.Key)
			if err != nil {
				return nil, nil, nil, "", err
			}
			if ok {
				result = cached
				evidence = append(evidence, &factsv1.Evidence{
					Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN,
					Confidence: 1,
					Analyzer:   "code-facts.cache",
					Message:    "Reused graph cache entry " + entry.Key + ".",
				})
			}
		}
		var err error
		if result == nil {
			result, err = s.graphFlights.Do(ctx, graphPlan.Key, func(workCtx context.Context) (*GraphResult, error) {
				if useCache {
					cached, _, ok, cacheErr := s.cache.GetGraph(workCtx, graphPlan.Key)
					if cacheErr != nil {
						return nil, cacheErr
					}
					if ok {
						return cached, nil
					}
				}
				release, admissionErr := s.admission.Acquire(workCtx, "graph", 0)
				if admissionErr != nil {
					return nil, admissionErr
				}
				defer release()
				extracted, extractionErr := extractForFamilies(workCtx, provider, unit, include)
				if extractionErr != nil {
					return nil, extractionErr
				}
				graphPlan.GraphHash = extracted.GraphHash
				if cacheErr := s.cache.PutGraph(workCtx, graphPlan, extracted); cacheErr != nil {
					return nil, cacheErr
				}
				return extracted, nil
			})
		}
		if err != nil {
			var unsupported ProviderUnsupportedError
			if errors.As(err, &unsupported) && !target.GetRequested().GetStrict() {
				warnings = append(warnings, providerWarning(unsupported.Analyzer, "unsupported", unsupported.Error(), factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED))
				evidence = append(evidence, &factsv1.Evidence{
					Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED,
					Confidence: 0,
					Analyzer:   unsupported.Analyzer,
					Message:    unsupported.Error(),
				})
				continue
			}
			var unavailable ProviderUnavailableError
			if errors.As(err, &unavailable) && !target.GetRequested().GetStrict() {
				warnings = append(warnings, providerWarning(unavailable.Analyzer, "unavailable", unavailable.Error(), factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN))
				evidence = append(evidence, &factsv1.Evidence{
					Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN,
					Confidence: 0,
					Analyzer:   unavailable.Analyzer,
					Message:    unavailable.Error(),
				})
				continue
			}
			return nil, nil, nil, "", err
		}
		graphHashes = append(graphHashes, result.GraphHash)
		unitFacts, unitWarnings, unitEvidence := normalizeGraphFacts(unit, provider, result, include)
		qualifyFleetFacts(target, unit, unitFacts)
		facts = append(facts, unitFacts...)
		warnings = append(warnings, unitWarnings...)
		evidence = append(evidence, unitEvidence...)
	}
	return facts, warnings, evidence, strings.Join(graphHashes, ","), nil
}

func qualifyFleetFacts(target *factsv1.TargetContext, unit *factsv1.ParseUnit, facts []*factsv1.GenericFact) {
	requested := target.GetRequested().GetKind()
	if requested != factsv1.TargetKind_TARGET_KIND_PROJECT && requested != factsv1.TargetKind_TARGET_KIND_REPO && requested != factsv1.TargetKind_TARGET_KIND_CONTROL_PLANE {
		return
	}
	root := filepath.Clean(unit.GetRootPath())
	label := scenarioFromPath(root, target.GetRootPath())
	if label == "" {
		label = filepath.ToSlash(root)
	}
	for _, fact := range facts {
		if fact == nil {
			continue
		}
		fact.Id = label + ":" + fact.GetId()
		if fact.Attributes == nil {
			fact.Attributes = map[string]string{}
		}
		fact.Attributes["target_root"] = root
		fact.Attributes["target_qualified"] = "true"
		fact.Attributes["target_label"] = label
	}
}

func (s *Service) analyzeForProof(ctx context.Context, targetReq *factsv1.CodeTarget, target *factsv1.TargetContext, include []factsv1.FactFamily, useCache bool, languageFilter ...[]string) (proofInput, error) {
	filter := targetReq.GetLanguageFilter()
	if len(languageFilter) > 0 {
		filter = languageFilter[0]
	}
	parseUnits := filterParseUnits(discoverParseUnits(target), filter)
	sourceHash, configHash := sourceFingerprint(target, parseUnits)
	cachePlan := reportCachePlan(targetReq, target, parseUnits, include, sourceHash, configHash, 0)
	cacheMeta := cachePlan.metadata(cacheState(useCache), cacheReason(useCache))
	facts, warnings, evidence, graphHash, err := s.analyze(ctx, target, parseUnits, include, sourceHash, configHash, useCache)
	if err != nil {
		return proofInput{}, err
	}
	cacheMeta.GraphHash = graphHash
	return s.proofInput(target, facts, warnings, evidence, cacheMeta), nil
}

func (s *Service) describeFileDomains(ctx context.Context, target *factsv1.TargetContext) ([]*factsv1.GenericFact, []*factsv1.Evidence, []*factsv1.Warning, error) {
	if s.fileDomains == nil {
		return []*factsv1.GenericFact{unsupportedFact(factsv1.FactFamily_FACT_FAMILY_FILE_DOMAIN)}, nil, []*factsv1.Warning{unsupportedWarning(factsv1.FactFamily_FACT_FAMILY_FILE_DOMAIN)}, nil
	}
	return s.fileDomains.DescribeFileDomains(ctx, target)
}

func needsAnalyzer(families []factsv1.FactFamily) bool {
	for _, family := range families {
		switch family {
		case factsv1.FactFamily_FACT_FAMILY_IMPORTS,
			factsv1.FactFamily_FACT_FAMILY_SYMBOLS,
			factsv1.FactFamily_FACT_FAMILY_REFERENCES,
			factsv1.FactFamily_FACT_FAMILY_CALLS,
			factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION,
			factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS:
			return true
		}
	}
	return false
}

func isImplementedFamily(family factsv1.FactFamily) bool {
	switch family {
	case factsv1.FactFamily_FACT_FAMILY_SURFACES,
		factsv1.FactFamily_FACT_FAMILY_PARSE_UNITS,
		factsv1.FactFamily_FACT_FAMILY_IMPORTS,
		factsv1.FactFamily_FACT_FAMILY_SYMBOLS,
		factsv1.FactFamily_FACT_FAMILY_REFERENCES,
		factsv1.FactFamily_FACT_FAMILY_CALLS,
		factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION,
		factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS,
		factsv1.FactFamily_FACT_FAMILY_FILE_DOMAIN:
		return true
	default:
		return false
	}
}

func firstUnitMessage(unit *factsv1.ParseUnit) string {
	for _, ev := range unit.GetEvidence() {
		if strings.TrimSpace(ev.GetMessage()) != "" {
			return ev.GetMessage()
		}
	}
	return "Parse unit is not supported by an analyzer."
}

func filterParseUnits(units []*factsv1.ParseUnit, languages []string) []*factsv1.ParseUnit {
	allowed := languageSet(languages)
	if len(allowed) == 0 {
		return units
	}
	out := make([]*factsv1.ParseUnit, 0, len(units))
	for _, unit := range units {
		if allowed[strings.ToLower(strings.TrimSpace(unit.GetLanguage()))] {
			out = append(out, unit)
		}
	}
	return out
}

func parseUnitsForDescribeAnalysis(units []*factsv1.ParseUnit, include []factsv1.FactFamily, languageFilter []string) []*factsv1.ParseUnit {
	if len(languageFilter) > 0 {
		return units
	}
	if hasFamily(include, factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS) &&
		!hasFamily(include, factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION) &&
		!hasDirectAnalyzerFamily(include) {
		return filterParseUnits(units, []string{"go", "typescript"})
	}
	return units
}

func hasDirectAnalyzerFamily(families []factsv1.FactFamily) bool {
	for _, family := range families {
		switch family {
		case factsv1.FactFamily_FACT_FAMILY_IMPORTS,
			factsv1.FactFamily_FACT_FAMILY_SYMBOLS,
			factsv1.FactFamily_FACT_FAMILY_REFERENCES,
			factsv1.FactFamily_FACT_FAMILY_CALLS:
			return true
		}
	}
	return false
}

func endpointProofLanguages(requested []string) []string {
	if len(requested) > 0 {
		return requested
	}
	return []string{"go", "typescript"}
}

func languageSet(languages []string) map[string]bool {
	out := map[string]bool{}
	for _, language := range languages {
		language = strings.ToLower(strings.TrimSpace(language))
		if language != "" {
			out[language] = true
		}
	}
	return out
}

func normalizeFamilies(in []factsv1.FactFamily) []factsv1.FactFamily {
	if len(in) == 0 {
		return []factsv1.FactFamily{
			factsv1.FactFamily_FACT_FAMILY_SURFACES,
			factsv1.FactFamily_FACT_FAMILY_PARSE_UNITS,
		}
	}
	all := false
	seen := map[factsv1.FactFamily]bool{}
	for _, f := range in {
		if f == factsv1.FactFamily_FACT_FAMILY_ALL {
			all = true
			break
		}
		if f != factsv1.FactFamily_FACT_FAMILY_UNSPECIFIED {
			seen[f] = true
		}
	}
	if all {
		return []factsv1.FactFamily{
			factsv1.FactFamily_FACT_FAMILY_SURFACES,
			factsv1.FactFamily_FACT_FAMILY_PARSE_UNITS,
			factsv1.FactFamily_FACT_FAMILY_IMPORTS,
			factsv1.FactFamily_FACT_FAMILY_SYMBOLS,
			factsv1.FactFamily_FACT_FAMILY_REFERENCES,
			factsv1.FactFamily_FACT_FAMILY_CALLS,
			factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION,
			factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS,
			factsv1.FactFamily_FACT_FAMILY_CLI_PROOFS,
			factsv1.FactFamily_FACT_FAMILY_UI_WIDGET_PROOFS,
			factsv1.FactFamily_FACT_FAMILY_FILE_DOMAIN,
		}
	}
	out := make([]factsv1.FactFamily, 0, len(seen))
	order := []factsv1.FactFamily{
		factsv1.FactFamily_FACT_FAMILY_SURFACES,
		factsv1.FactFamily_FACT_FAMILY_PARSE_UNITS,
		factsv1.FactFamily_FACT_FAMILY_IMPORTS,
		factsv1.FactFamily_FACT_FAMILY_SYMBOLS,
		factsv1.FactFamily_FACT_FAMILY_REFERENCES,
		factsv1.FactFamily_FACT_FAMILY_CALLS,
		factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION,
		factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS,
		factsv1.FactFamily_FACT_FAMILY_CLI_PROOFS,
		factsv1.FactFamily_FACT_FAMILY_UI_WIDGET_PROOFS,
		factsv1.FactFamily_FACT_FAMILY_FILE_DOMAIN,
	}
	for _, f := range order {
		if seen[f] {
			out = append(out, f)
		}
	}
	return out
}

func hasFamily(families []factsv1.FactFamily, family factsv1.FactFamily) bool {
	for _, f := range families {
		if f == family {
			return true
		}
	}
	return false
}

func expandAnalyzerFamilies(families []factsv1.FactFamily) []factsv1.FactFamily {
	seen := map[factsv1.FactFamily]bool{}
	for _, family := range families {
		seen[family] = true
	}
	if seen[factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION] {
		seen[factsv1.FactFamily_FACT_FAMILY_IMPORTS] = true
	}
	if seen[factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS] {
		seen[factsv1.FactFamily_FACT_FAMILY_IMPORTS] = true
		seen[factsv1.FactFamily_FACT_FAMILY_REFERENCES] = true
		seen[factsv1.FactFamily_FACT_FAMILY_CALLS] = true
	}
	out := make([]factsv1.FactFamily, 0, len(seen))
	for _, family := range []factsv1.FactFamily{
		factsv1.FactFamily_FACT_FAMILY_IMPORTS,
		factsv1.FactFamily_FACT_FAMILY_SYMBOLS,
		factsv1.FactFamily_FACT_FAMILY_REFERENCES,
		factsv1.FactFamily_FACT_FAMILY_CALLS,
		factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION,
		factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS,
	} {
		if seen[family] {
			out = append(out, family)
		}
	}
	return out
}

func filterFactsForFamilies(facts []*factsv1.GenericFact, families []factsv1.FactFamily) []*factsv1.GenericFact {
	seen := map[factsv1.FactFamily]bool{}
	for _, family := range families {
		seen[family] = true
	}
	out := make([]*factsv1.GenericFact, 0, len(facts))
	for _, fact := range facts {
		if seen[fact.GetFamily()] {
			out = append(out, fact)
		}
	}
	return out
}

func unsupportedFact(family factsv1.FactFamily) *factsv1.GenericFact {
	return &factsv1.GenericFact{
		Id:      strings.ToLower(family.String()) + ".pending",
		Family:  family,
		Kind:    "phase_6_contract_placeholder",
		Subject: family.String(),
		Evidence: []*factsv1.Evidence{
			unsupportedEvidence(unsupportedMessage(family)),
		},
	}
}

func unsupportedMessage(family factsv1.FactFamily) string {
	if family == factsv1.FactFamily_FACT_FAMILY_FILE_DOMAIN {
		return "FILE_DOMAIN is produced by architecture-cartographer; Code Facts exposes the contract but does not infer domain ownership."
	}
	return "This fact family is intentionally exposed but not yet analyzer-backed."
}

func unsupportedEvidence(message string) *factsv1.Evidence {
	return &factsv1.Evidence{
		Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED,
		Confidence: 0,
		Analyzer:   "code-facts.phase6",
		Message:    message,
	}
}

func unsupportedWarning(family factsv1.FactFamily) *factsv1.Warning {
	message := family.String() + " is exposed by the API contract but not implemented until later plan phases."
	if family == factsv1.FactFamily_FACT_FAMILY_FILE_DOMAIN {
		message = "FILE_DOMAIN requires architecture-cartographer verdicts; Code Facts will not synthesize domain ownership locally."
	}
	return &factsv1.Warning{
		Code:    "phase_6_contract_only",
		Message: message,
		Status:  factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED,
	}
}

func reportCachePlan(targetReq *factsv1.CodeTarget, target *factsv1.TargetContext, units []*factsv1.ParseUnit, families []factsv1.FactFamily, sourceHash string, configHash string, maxDepth int32) cacheEntry {
	familiesKey := familyKey(families)
	providers := providerVersions(units)
	key := cacheKey(cacheScopeReport, cacheAnalyzerVersion, cacheSchemaVersion, target.GetRootPath(), targetReq.GetKind().String(), targetReq.GetScenario(), familiesKey, fmt.Sprint(maxDepth), sourceHash, configHash, providers)
	return cacheEntry{
		Key:           key,
		LogicalKey:    cacheKey(cacheScopeReport, target.GetRootPath(), cacheAnalyzerVersion, familiesKey, targetReq.GetKind().String(), targetReq.GetScenario(), targetReq.GetPath(), targetReq.GetRepoRoot(), fmt.Sprint(maxDepth), providers),
		Scope:         cacheScopeReport,
		TargetRoot:    target.GetRootPath(),
		Analyzer:      cacheAnalyzerVersion,
		Provider:      "code-facts",
		ProviderVer:   providers,
		SchemaVersion: cacheSchemaVersion,
		SourceHash:    sourceHash,
		ConfigHash:    configHash,
		FamilyKey:     familiesKey,
		Identity:      cacheKey(targetReq.GetKind().String(), targetReq.GetScenario(), targetReq.GetPath(), targetReq.GetRepoRoot(), fmt.Sprint(maxDepth), providers),
	}
}

func graphCachePlan(target *factsv1.TargetContext, unit *factsv1.ParseUnit, provider GraphProvider, extractionProfile string, sourceHash string, configHash string) cacheEntry {
	providerVer := provider.AnalyzerName() + ":phase8"
	key := cacheKey(cacheScopeGraph, cacheAnalyzerVersion, cacheSchemaVersion, providerVer, extractionProfile, target.GetRootPath(), unit.GetId(), unit.GetRootPath(), unit.GetConfigPath(), sourceHash, configHash)
	return cacheEntry{
		Key:           key,
		LogicalKey:    cacheKey(cacheScopeGraph, target.GetRootPath(), cacheAnalyzerVersion, unit.GetLanguage(), provider.AnalyzerName(), extractionProfile, unit.GetId(), unit.GetRootPath(), unit.GetConfigPath()),
		Scope:         cacheScopeGraph,
		TargetRoot:    target.GetRootPath(),
		Analyzer:      cacheAnalyzerVersion,
		Provider:      provider.AnalyzerName(),
		ProviderVer:   providerVer,
		SchemaVersion: cacheSchemaVersion,
		SourceHash:    sourceHash,
		ConfigHash:    configHash,
		FamilyKey:     unit.GetLanguage() + ":" + extractionProfile,
		Identity:      cacheKey(provider.AnalyzerName(), extractionProfile, unit.GetId(), unit.GetRootPath(), unit.GetConfigPath()),
	}
}

func surfaceCacheMetadata(req *factsv1.CodeTarget, target *factsv1.TargetContext) *factsv1.CacheMetadata {
	entry := cacheEntry{
		Key:           cacheKeyForTarget(req, target),
		Scope:         cacheScopeReport,
		TargetRoot:    target.GetRootPath(),
		Analyzer:      cacheAnalyzerVersion,
		Provider:      "code-facts",
		ProviderVer:   cacheAnalyzerVersion,
		SchemaVersion: cacheSchemaVersion,
	}
	return entry.metadata("miss", "surface inventory is recomputed from target metadata")
}

func cacheKeyForTarget(req *factsv1.CodeTarget, target *factsv1.TargetContext) string {
	return cacheKey(cacheAnalyzerVersion, cacheSchemaVersion, target.GetRootPath(), req.GetKind().String(), req.GetScenario(), req.GetPath(), req.GetRepoRoot())
}

func cacheState(useCache bool) string {
	if useCache {
		return "miss"
	}
	return "bypassed"
}

func cacheReason(useCache bool) string {
	if useCache {
		return "no reusable report cache entry matched target, options, source/config hashes, and analyzer versions"
	}
	return "use_cache=false forced fresh extraction before refreshing the cache"
}

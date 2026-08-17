package facts

import (
	"context"
	"errors"
	"fmt"
	"math"
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
	broker      *Broker
	cache       CacheRepository
	fileDomains FileDomainProvider
	projectIdx  *lexicalProjectIndex
}

type ServiceOption func(*Service)

func NewService(opts ...ServiceOption) *Service {
	s := &Service{broker: NewBroker(), cache: NewMemoryCacheRepository()}
	for _, opt := range opts {
		opt(s)
	}
	if s.cache == nil {
		s.cache = NewMemoryCacheRepository()
	}
	if root, err := resolveRepoRoot(""); err == nil {
		s.projectIdx = newLexicalProjectIndex(root)
		go s.projectIdx.build()
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
	offset := 0
	if strings.TrimSpace(pageToken) != "" {
		parsed, err := strconv.Atoi(pageToken)
		if err != nil || parsed < 0 {
			return nil, fmt.Errorf("invalid page_token %q", pageToken)
		}
		offset = parsed
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

// Search performs a bounded lexical search over node-oriented evidence. It is
// the deliberately small query surface used by Search Hub: callers receive
// identifiers and provenance, while DescribeCodeFacts remains the authoritative
// detail endpoint. No graph edge family is embedded in this response.
func (s *Service) Search(ctx context.Context, req *factsv1.SearchRequest) (*factsv1.SearchResponse, error) {
	query := strings.TrimSpace(req.GetQuery())
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	limit := int(req.GetLimit())
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	target := req.GetTarget()
	if target == nil {
		target = &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_PROJECT}
	}
	families := req.GetFamilies()
	if len(families) == 0 {
		families = []factsv1.FactFamily{
			factsv1.FactFamily_FACT_FAMILY_SYMBOLS,
			factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION,
		}
	}
	report, err := s.searchReport(ctx, target, families, query)
	if err != nil {
		return nil, err
	}
	terms := searchQueryTokens(query)
	hits := make([]*factsv1.SearchHit, 0, len(report.GetFacts()))
	var indexedScores map[*factsv1.GenericFact]float64
	if target.GetKind() == factsv1.TargetKind_TARGET_KIND_PROJECT || target.GetKind() == factsv1.TargetKind_TARGET_KIND_REPO {
		if s.projectIdx != nil {
			indexedScores = s.projectIdx.scoreFacts(query, terms)
		}
	}
	for _, fact := range report.GetFacts() {
		if fact == nil || len(terms) == 0 {
			continue
		}
		var score float64
		if indexedScores != nil {
			var indexed bool
			score, indexed = indexedScores[fact]
			if !indexed {
				continue
			}
		} else {
			score = scoreSearchFact(fact, query, terms)
		}
		if score == 0 {
			continue
		}
		attrs := fact.GetAttributes()
		path := attrs["path"]
		status := factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN
		message := fact.GetSubject()
		if evidence := fact.GetEvidence(); len(evidence) > 0 {
			status = evidence[0].GetStatus()
			if evidence[0].GetRange() != nil && path == "" {
				path = evidence[0].GetRange().GetFile()
			}
			if strings.TrimSpace(evidence[0].GetMessage()) != "" {
				message = evidence[0].GetMessage()
			}
		}
		hits = append(hits, &factsv1.SearchHit{
			Id:             fact.GetId(),
			Title:          fact.GetSubject(),
			Text:           message,
			Score:          score,
			Path:           path,
			Analyzer:       attrs["analyzer"],
			EvidenceStatus: status,
			FactKind:       fact.GetKind(),
		})
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].GetScore() != hits[j].GetScore() {
			return hits[i].GetScore() > hits[j].GetScore()
		}
		return hits[i].GetId() < hits[j].GetId()
	})
	if len(hits) > limit {
		hits = hits[:limit]
	}
	if req.GetExpandEdges() {
		// Expand only the top-ranked node. This is the useful explanation for
		// an answer and keeps one cold graph lookup from multiplying by the
		// provider's result limit.
		if len(hits) > 0 {
			hits[0].EdgeExpansions = s.expandEdges(ctx, target, hits[0], req.GetQuery())
		}
	}
	return &factsv1.SearchResponse{Results: hits}, nil
}

// expandEdges resolves graph relationships only after a node hit has been
// selected. This keeps the Search Hub corpus small while preserving the
// authoritative graph evidence for callers that need callers or references.
func (s *Service) expandEdges(ctx context.Context, target *factsv1.CodeTarget, hit *factsv1.SearchHit, query string) []*factsv1.SearchExpansion {
	edgeTarget := target
	if target.GetKind() == factsv1.TargetKind_TARGET_KIND_PROJECT || target.GetKind() == factsv1.TargetKind_TARGET_KIND_REPO {
		// Resolve only the hit's source file for a fleet search. A whole-scenario
		// graph expansion defeats the provider deadline and is unnecessary for
		// returning the local callers/references that explain this hit.
		if root, err := resolveRepoRoot(target.GetRepoRoot()); err == nil && !filepath.IsAbs(filepath.FromSlash(hit.GetPath())) {
			edgeTarget = &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_PATH, Path: filepath.Join(root, filepath.FromSlash(hit.GetPath()))}
		}
	}
	edgeCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	report, err := s.Describe(edgeCtx, &factsv1.DescribeCodeFactsRequest{
		Target: edgeTarget,
		Include: []factsv1.FactFamily{
			factsv1.FactFamily_FACT_FAMILY_REFERENCES,
			factsv1.FactFamily_FACT_FAMILY_CALLS,
		},
		UseCache: true,
	})
	if err != nil {
		return s.lexicalEdgeExpansions(hit, query)
	}
	terms := strings.Fields(strings.ToLower(query + " " + hit.GetTitle()))
	expansions := make([]*factsv1.SearchExpansion, 0, 5)
	seen := make(map[string]struct{})
	for _, fact := range report.GetFacts() {
		if fact == nil || (fact.GetFamily() != factsv1.FactFamily_FACT_FAMILY_REFERENCES && fact.GetFamily() != factsv1.FactFamily_FACT_FAMILY_CALLS) {
			continue
		}
		corpus := strings.ToLower(strings.Join([]string{
			fact.GetId(), fact.GetSubject(), fact.GetKind(), fact.GetAttributes()["name"], fact.GetAttributes()["path"], fact.GetAttributes()["qualified_name"],
		}, " "))
		matched := false
		for _, term := range terms {
			if term != "" && strings.Contains(corpus, term) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		if _, ok := seen[fact.GetId()]; ok {
			continue
		}
		seen[fact.GetId()] = struct{}{}
		path := fact.GetAttributes()["path"]
		status := factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN
		analyzer := fact.GetAttributes()["analyzer"]
		if evidence := fact.GetEvidence(); len(evidence) > 0 {
			status = evidence[0].GetStatus()
			if analyzer == "" {
				analyzer = evidence[0].GetAnalyzer()
			}
			if path == "" && evidence[0].GetRange() != nil {
				path = evidence[0].GetRange().GetFile()
			}
		}
		text := fact.GetSubject()
		if len(fact.GetEvidence()) > 0 && fact.GetEvidence()[0].GetMessage() != "" {
			text = fact.GetEvidence()[0].GetMessage()
		}
		expansions = append(expansions, &factsv1.SearchExpansion{
			Id: fact.GetId(), Title: fact.GetSubject(), Text: text, Path: path,
			Analyzer: analyzer, EvidenceStatus: status, FactKind: fact.GetKind(), Family: fact.GetFamily(),
		})
		if len(expansions) == 5 {
			break
		}
	}
	return expansions
}

// lexicalEdgeExpansions is an explicit, lower-confidence fallback for a cold
// analyzer graph. It keeps Search Hub useful inside its provider deadline and
// never presents lexical relationships as analyzer-backed graph evidence.
func (s *Service) lexicalEdgeExpansions(hit *factsv1.SearchHit, query string) []*factsv1.SearchExpansion {
	if s.projectIdx == nil {
		return nil
	}
	terms := strings.Fields(strings.ToLower(query + " " + hit.GetTitle()))
	s.projectIdx.mu.RLock()
	facts := append([]*factsv1.GenericFact(nil), s.projectIdx.facts...)
	s.projectIdx.mu.RUnlock()
	result := make([]*factsv1.SearchExpansion, 0, 5)
	seen := make(map[string]struct{})
	for _, fact := range facts {
		path := fact.GetAttributes()["path"]
		if path == "" || path == hit.GetPath() {
			continue
		}
		corpus := strings.ToLower(fact.GetSubject() + " " + path)
		matched := false
		for _, term := range terms {
			if term != "" && strings.Contains(corpus, term) {
				matched = true
				break
			}
		}
		if !matched || fact.GetFamily() != factsv1.FactFamily_FACT_FAMILY_SYMBOLS {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		result = append(result, &factsv1.SearchExpansion{
			Id: fact.GetId(), Title: fact.GetSubject(), Text: "Lexical relationship candidate; confirm with Describe for graph evidence.",
			Path: path, Analyzer: "code-facts.lexical", EvidenceStatus: factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN,
			FactKind: "lexical_reference", Family: factsv1.FactFamily_FACT_FAMILY_REFERENCES,
		})
		if len(result) == 5 {
			break
		}
	}
	return result
}

func (s *Service) searchReport(ctx context.Context, target *factsv1.CodeTarget, families []factsv1.FactFamily, query string) (*factsv1.CodeFactsReport, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if target.GetKind() != factsv1.TargetKind_TARGET_KIND_PROJECT && target.GetKind() != factsv1.TargetKind_TARGET_KIND_REPO {
		return s.Describe(ctx, &factsv1.DescribeCodeFactsRequest{Target: target, Include: families, UseCache: true})
	}
	repoRoot, err := resolveRepoRoot(target.GetRepoRoot())
	if err != nil {
		return nil, err
	}
	if s.projectIdx != nil && filepath.Clean(s.projectIdx.root) == filepath.Clean(repoRoot) {
		if !s.projectIdx.isReady() {
			return lexicalProjectReport(ctx, repoRoot, target, families, query)
		}
		return s.projectIdx.report(ctx, target, families, query)
	}
	return lexicalProjectReport(ctx, repoRoot, target, families, query)
}

type lexicalProjectIndex struct {
	root     string
	mu       sync.RWMutex
	facts    []*factsv1.GenericFact
	postings map[string][]lexicalPosting
	docFreq  map[string]int
	docCount int
	ready    chan struct{}
}

type lexicalPosting struct {
	fact   *factsv1.GenericFact
	weight float64
}

func newLexicalProjectIndex(root string) *lexicalProjectIndex {
	return &lexicalProjectIndex{root: root, postings: make(map[string][]lexicalPosting), docFreq: make(map[string]int), ready: make(chan struct{})}
}

func (idx *lexicalProjectIndex) markReady() {
	select {
	case <-idx.ready:
	default:
		close(idx.ready)
	}
}

func (idx *lexicalProjectIndex) isReady() bool {
	select {
	case <-idx.ready:
		return true
	default:
		return false
	}
}

func (idx *lexicalProjectIndex) addFact(fact *factsv1.GenericFact) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.facts = append(idx.facts, fact)
	weights := make(map[string]float64)
	for _, field := range factSearchFields(fact) {
		for _, token := range uniqueSearchTokens(tokenizeSearchText(field.value), false) {
			if field.weight > weights[token] {
				weights[token] = field.weight
			}
		}
	}
	for token, weight := range weights {
		idx.postings[token] = append(idx.postings[token], lexicalPosting{fact: fact, weight: weight})
		idx.docFreq[token]++
	}
	idx.docCount++
}

func (idx *lexicalProjectIndex) build() {
	defer idx.markReady()
	for _, root := range []string{filepath.Join(idx.root, "scenarios"), filepath.Join(idx.root, "packages")} {
		_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if entry.IsDir() {
				if path != root && shouldPruneDir(entry.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			switch filepath.Ext(path) {
			case ".go", ".ts", ".tsx", ".proto":
			default:
				return nil
			}
			payload, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			rel, _ := filepath.Rel(idx.root, path)
			for lineNumber, line := range strings.Split(string(payload), "\n") {
				subject := strings.TrimSpace(line)
				if !shouldIndexLexicalLine(filepath.ToSlash(rel), lineNumber+1, subject) {
					continue
				}
				family := factsv1.FactFamily_FACT_FAMILY_SYMBOLS
				if filepath.Ext(path) == ".proto" {
					family = factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION
				}
				fact := lexicalFact(filepath.ToSlash(rel), lineNumber+1, subject, family)
				idx.addFact(fact)
			}
			return nil
		})
	}
}

func (idx *lexicalProjectIndex) report(ctx context.Context, target *factsv1.CodeTarget, families []factsv1.FactFamily, query string) (*factsv1.CodeFactsReport, error) {
	terms := searchQueryTokens(query)
	idx.mu.RLock()
	var facts []*factsv1.GenericFact
	if len(idx.postings) > 0 {
		seen := make(map[*factsv1.GenericFact]struct{})
		for _, term := range terms {
			for _, posting := range idx.postings[term] {
				seen[posting.fact] = struct{}{}
			}
		}
		facts = make([]*factsv1.GenericFact, 0, len(seen))
		for fact := range seen {
			facts = append(facts, fact)
		}
		sort.Slice(facts, func(i, j int) bool { return facts[i].GetId() < facts[j].GetId() })
	} else {
		facts = append([]*factsv1.GenericFact(nil), idx.facts...)
	}
	idx.mu.RUnlock()
	report := &factsv1.CodeFactsReport{Target: &factsv1.TargetContext{Requested: target, ResolvedKind: target.GetKind(), RootPath: idx.root, RootPaths: []string{filepath.Join(idx.root, "scenarios"), filepath.Join(idx.root, "packages")}}}
	for _, fact := range facts {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !hasFamily(families, fact.GetFamily()) {
			continue
		}
		if len(idx.postings) > 0 || scoreSearchFact(fact, query, terms) > 0 {
			report.Facts = append(report.Facts, fact)
		}
	}
	return report, nil
}

func (idx *lexicalProjectIndex) scoreFacts(query string, queryTokens []string) map[*factsv1.GenericFact]float64 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	if len(idx.postings) == 0 || len(queryTokens) == 0 {
		return nil
	}
	type accumulated struct {
		matched int
		score   float64
	}
	accumulatedByFact := make(map[*factsv1.GenericFact]accumulated)
	compoundIdentifiers := searchCompoundIdentifiers(query)
	exactCompoundFacts := make(map[*factsv1.GenericFact]struct{})
	if len(compoundIdentifiers) > 0 {
		for _, identifier := range compoundIdentifiers {
			identifierTokens := tokenizeSearchText(identifier)
			if len(identifierTokens) == 0 {
				continue
			}
			for _, posting := range idx.postings[identifierTokens[0]] {
				if searchFactContainsCompoundIdentifier(posting.fact, identifier) {
					exactCompoundFacts[posting.fact] = struct{}{}
				}
			}
		}
	}
	for _, token := range queryTokens {
		termWeight := 1 + math.Log(float64(idx.docCount+1)/float64(idx.docFreq[token]+1))
		for _, posting := range idx.postings[token] {
			current := accumulatedByFact[posting.fact]
			current.matched++
			current.score += posting.weight * termWeight
			accumulatedByFact[posting.fact] = current
		}
	}
	totalTermWeight := 0.0
	for _, token := range queryTokens {
		totalTermWeight += 1 + math.Log(float64(idx.docCount+1)/float64(idx.docFreq[token]+1))
	}
	scores := make(map[*factsv1.GenericFact]float64, len(accumulatedByFact))
	for fact, current := range accumulatedByFact {
		if current.matched == 0 {
			continue
		}
		if len(exactCompoundFacts) > 0 {
			if _, ok := exactCompoundFacts[fact]; !ok {
				continue
			}
		}
		score := current.score / (10 * totalTermWeight)
		score += searchScoreBonusesIndexed(fact, query, queryTokens)
		coverage := float64(current.matched) / float64(len(queryTokens))
		score *= coverage * coverage
		if score > 1 {
			score = 1
		}
		scores[fact] = score
	}
	return scores
}

func lexicalFact(path string, line int, subject string, family factsv1.FactFamily) *factsv1.GenericFact {
	return &factsv1.GenericFact{
		Id: "code-facts:lexical:" + path + ":" + strconv.Itoa(line), Family: family, Kind: "lexical_source", Subject: subject,
		Attributes: map[string]string{"path": path, "line": strconv.Itoa(line), "analyzer": "code-facts.lexical"},
		Evidence:   []*factsv1.Evidence{{Status: factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, Confidence: 0.7, Analyzer: "code-facts.lexical", Message: "Matched source text in the bounded project lexical index.", Range: &factsv1.SourceRange{File: path, StartLine: int32(line), EndLine: int32(line)}}},
	}
}

// lexicalProjectReport is the bounded lexical leg for fleet search. It reads
// source lines directly instead of invoking one graph provider per module;
// this keeps identifier-shaped Search Hub queries inside the provider timeout
// while preserving source paths and an explicit evidence condition. Describe
// remains the authoritative graph-backed detail endpoint for callers that need
// references, calls, or analyzer-specific facts.
func lexicalProjectReport(ctx context.Context, repoRoot string, target *factsv1.CodeTarget, families []factsv1.FactFamily, query string) (*factsv1.CodeFactsReport, error) {
	terms := searchQueryTokens(query)
	report := &factsv1.CodeFactsReport{Target: &factsv1.TargetContext{Requested: target, ResolvedKind: target.GetKind(), RootPath: repoRoot, RootPaths: []string{filepath.Join(repoRoot, "scenarios"), filepath.Join(repoRoot, "packages")}}}
	allowed := map[string]bool{".go": true, ".ts": true, ".tsx": true, ".proto": true}
	roots := []string{filepath.Join(repoRoot, "scenarios"), filepath.Join(repoRoot, "packages")}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if entry.IsDir() {
				if path != root && shouldPruneDir(entry.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			if !allowed[filepath.Ext(path)] {
				return nil
			}
			payload, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			for lineNumber, line := range strings.Split(string(payload), "\n") {
				rel, _ := filepath.Rel(repoRoot, path)
				family := factsv1.FactFamily_FACT_FAMILY_SYMBOLS
				if filepath.Ext(path) == ".proto" {
					family = factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION
				}
				if !hasFamily(families, family) {
					continue
				}
				subject := strings.TrimSpace(line)
				fact := &factsv1.GenericFact{
					Id: "code-facts:lexical:" + filepath.ToSlash(rel) + ":" + strconv.Itoa(lineNumber+1), Family: family, Kind: "lexical_source", Subject: subject,
					Attributes: map[string]string{"path": filepath.ToSlash(rel), "line": strconv.Itoa(lineNumber + 1), "analyzer": "code-facts.lexical"},
					Evidence:   []*factsv1.Evidence{{Status: factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, Confidence: 0.7, Analyzer: "code-facts.lexical", Message: "Matched source text in the bounded project lexical index.", Range: &factsv1.SourceRange{File: filepath.ToSlash(rel), StartLine: int32(lineNumber + 1), EndLine: int32(lineNumber + 1)}}},
				}
				if scoreSearchFact(fact, query, terms) > 0 {
					report.Facts = append(report.Facts, fact)
				}
			}
			return nil
		})
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
	}
	sort.SliceStable(report.Facts, func(i, j int) bool { return report.Facts[i].GetId() < report.Facts[j].GetId() })
	return report, nil
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
		scenarios, err = listScenarioSlugs(repoRoot)
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
		graphPlan := graphCachePlan(target, unit, provider, unitSourceHash, unitConfigHash)
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
			result, err = provider.Extract(ctx, unit)
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
		graphPlan.GraphHash = result.GraphHash
		if err := s.cache.PutGraph(ctx, graphPlan, result); err != nil {
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

func graphCachePlan(target *factsv1.TargetContext, unit *factsv1.ParseUnit, provider GraphProvider, sourceHash string, configHash string) cacheEntry {
	providerVer := provider.AnalyzerName() + ":phase8"
	key := cacheKey(cacheScopeGraph, cacheAnalyzerVersion, cacheSchemaVersion, providerVer, target.GetRootPath(), unit.GetId(), unit.GetRootPath(), unit.GetConfigPath(), sourceHash, configHash)
	return cacheEntry{
		Key:           key,
		LogicalKey:    cacheKey(cacheScopeGraph, target.GetRootPath(), cacheAnalyzerVersion, unit.GetLanguage(), provider.AnalyzerName(), unit.GetId(), unit.GetRootPath(), unit.GetConfigPath()),
		Scope:         cacheScopeGraph,
		TargetRoot:    target.GetRootPath(),
		Analyzer:      cacheAnalyzerVersion,
		Provider:      provider.AnalyzerName(),
		ProviderVer:   providerVer,
		SchemaVersion: cacheSchemaVersion,
		SourceHash:    sourceHash,
		ConfigHash:    configHash,
		FamilyKey:     unit.GetLanguage(),
		Identity:      cacheKey(provider.AnalyzerName(), unit.GetId(), unit.GetRootPath(), unit.GetConfigPath()),
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

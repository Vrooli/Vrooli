package facts

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"code-facts/internal/catalog"
	internalfacts "code-facts/internal/facts"
	"code-facts/internal/fswatch"
	"code-facts/internal/gateway"
	"code-facts/internal/indexcontrol"
	"code-facts/internal/retrieval"

	aisearch "github.com/vrooli/ai-go/search"
	"github.com/vrooli/api-core/pathfilter"
	repocontract "github.com/vrooli/repo-contract-go"
	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

const (
	indexPolicy           = "code-facts-corpus-v3-polyglot-import-edges-v1"
	indexBatchSize        = 16
	maxIndexedFile        = 256 * 1024
	indexedChunkBytes     = 8 * 1024
	maxImportEdgesPerFile = 256
)

var errImportIndexUnavailable = errors.New("language import evidence is unavailable in the active index")

type wallClock struct{}

func (wallClock) Now() time.Time { return time.Now().UTC() }

// DescribeFileMetrics projects per-file line counts from the active Code Facts
// catalog. Content hashes fence the output to the generation; changed files are
// reported as unknown until catalog reconciliation completes.
func (index *ProductionIndex) DescribeFileMetrics(ctx context.Context, target *factsv1.TargetContext) ([]*factsv1.GenericFact, []*factsv1.Evidence, []*factsv1.Warning, error) {
	if target == nil || strings.TrimSpace(target.GetRootPath()) == "" {
		return nil, nil, []*factsv1.Warning{{Code: "file_metrics_target_required", Message: "FILE_METRICS requires a bounded target path.", Status: factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED}}, nil
	}
	rootRel, err := filepath.Rel(index.repoRoot, filepath.Clean(target.GetRootPath()))
	if err != nil || rootRel == ".." || strings.HasPrefix(rootRel, ".."+string(filepath.Separator)) || filepath.IsAbs(rootRel) {
		return nil, nil, []*factsv1.Warning{{Code: "file_metrics_target_outside_catalog", Message: "FILE_METRICS target is outside the active repository catalog.", Status: factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED}}, nil
	}
	if rootRel == "." {
		rootRel = ""
	}
	generation, err := index.catalog.Active(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("read active Code Facts catalog for file metrics: %w", err)
	}
	facts := make([]*factsv1.GenericFact, 0)
	warnings := make([]*factsv1.Warning, 0)
	allSourcePaths := make([]string, 0)
	for token := ""; ; {
		if err := ctx.Err(); err != nil {
			return nil, nil, nil, err
		}
		page, pageErr := index.catalog.PageFiles(ctx, generation.ID, token, 1024)
		if pageErr != nil {
			return nil, nil, nil, pageErr
		}
		for _, file := range page.Files {
			if rootRel != "" && file.Path != rootRel && !strings.HasPrefix(file.Path, rootRel+"/") {
				continue
			}
			if !pathfilter.IsSourceExt(filepath.Ext(file.Path)) || excludedCatalogPath(file.Path) {
				continue
			}
			metrics, hash, countErr := inspectCatalogFile(ctx, filepath.Join(index.repoRoot, filepath.FromSlash(file.Path)), file.Language)
			if countErr != nil {
				// A catalog row can outlive its file until reconciliation drops it.
				// A deleted file has no metrics to report, so skipping it keeps the
				// projection complete for every file that still exists; any other
				// failure is a real measurement problem the caller must see.
				if errors.Is(countErr, fs.ErrNotExist) {
					continue
				}
				warnings = append(warnings, &factsv1.Warning{Code: "file_metrics_unavailable", Message: fmt.Sprintf("%s: %v", file.Path, countErr), Status: factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN})
				continue
			}
			if hash != file.Hash {
				// The metrics above were computed from the file's current bytes, so
				// they are fresh even though the catalog generation is behind. Keep
				// the warning so callers know the generation is stale, but still
				// project the measurement instead of dropping the file.
				warnings = append(warnings, &factsv1.Warning{Code: "file_metrics_stale_catalog", Message: file.Path + " changed after the active catalog generation; reconcile Code Facts before treating its catalog evidence as current.", Status: factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN})
			}
			relPath := file.Path
			if rootRel != "" {
				relPath = strings.TrimPrefix(strings.TrimPrefix(file.Path, rootRel), "/")
			}
			allSourcePaths = append(allSourcePaths, relPath)
			attributes := metrics.attributes()
			attributes["file_path"] = relPath
			attributes["file_extension"] = filepath.Ext(file.Path)
			attributes["language"] = file.Language
			attributes["role"] = string(file.Role)
			attributes["size_bytes"] = strconv.FormatInt(file.Size, 10)
			attributes["source_hash"] = file.Hash
			attributes["generation"] = generation.ID
			facts = append(facts, &factsv1.GenericFact{
				Id: "code-facts:file-metrics:" + file.ID, Family: factsv1.FactFamily_FACT_FAMILY_FILE_METRICS,
				Kind: "catalog_lexical_file_metrics", Subject: relPath,
				Attributes: attributes,
				Evidence:   []*factsv1.Evidence{{Status: factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, Confidence: 1, Analyzer: "code-facts.catalog", Message: "Lexical file metrics were computed from a file whose content hash matches the active catalog."}},
			})
		}
		if page.NextToken == "" {
			break
		}
		token = page.NextToken
	}
	// Test-file classification and companion matching are source facts shared
	// by all FILE_METRICS consumers. Compute them from the same catalog snapshot
	// instead of making each consumer maintain its own filename policy.
	testFiles, sourceFiles := splitTestSourcePaths(allSourcePaths)
	for _, fact := range facts {
		path := fact.GetAttributes()["file_path"]
		isTest := isTestSourcePath(path)
		fact.Attributes["test_classification_version"] = "1"
		fact.Attributes["is_test_file"] = strconv.FormatBool(isTest)
		fact.Attributes["has_test_file"] = strconv.FormatBool(isTest || hasTestCompanion(path, testFiles, sourceFiles))
	}
	evidence := []*factsv1.Evidence{{Status: factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, Confidence: 1, Analyzer: "code-facts.catalog", Message: fmt.Sprintf("Returned current non-empty line counts for %d catalog source files in generation %s.", len(facts), generation.ID)}}
	return facts, evidence, warnings, nil
}

func excludedCatalogPath(path string) bool {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, part := range parts[:max(0, len(parts)-1)] {
		if pathfilter.SkipDir(part) {
			return true
		}
	}
	return false
}

func inspectCatalogFile(ctx context.Context, path, language string) (lexicalFileMetrics, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return lexicalFileMetrics{}, "", err
	}
	defer file.Close()
	hash := sha256.New()
	metrics, err := inspectLexicalFile(ctx, io.TeeReader(file, hash), language)
	if err != nil {
		return lexicalFileMetrics{}, "", err
	}
	return metrics, "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

// ProductionIndex owns the durable build path and the source-free query path.
// Source files are opened only by buildGeneration; Search uses the active
// SQLite generation and cannot fall through to repository scanning.
type ProductionIndex struct {
	db        *sql.DB
	repoRoot  string
	catalog   *catalog.SQLiteRepository
	inspector catalog.FileInspector
	lexical   *retrieval.SQLiteIndex
	vectors   *retrieval.SQLiteVectorStore
	admission *internalfacts.WeightedAdmission
	engine    retrieval.HybridEngine
	jobs      indexcontrol.JobStore
	clock     wallClock
	mu        sync.Mutex
	refreshMu sync.Mutex
	servingMu sync.RWMutex
	cancels   map[string]context.CancelFunc
	bootstrap sync.Once
}

func NewProductionIndex(db *sql.DB, admission *internalfacts.WeightedAdmission) (*ProductionIndex, error) {
	if db == nil {
		return nil, fmt.Errorf("production index requires database")
	}
	repoRoot, err := repocontract.ResolveRepoRoot()
	if err != nil {
		return nil, fmt.Errorf("resolve production index repository: %w", err)
	}
	return newProductionIndex(db, repoRoot, admission)
}

func newProductionIndex(db *sql.DB, repoRoot string, admission *internalfacts.WeightedAdmission) (*ProductionIndex, error) {
	if db == nil || strings.TrimSpace(repoRoot) == "" {
		return nil, fmt.Errorf("production index requires database and repository root")
	}
	clock := wallClock{}
	lexical := retrieval.NewSQLiteIndex(db)
	index := &ProductionIndex{
		db: db, repoRoot: filepath.Clean(repoRoot), catalog: catalog.NewSQLiteRepository(db, clock),
		inspector: catalog.OSFileInspector{}, lexical: lexical, vectors: retrieval.NewSQLiteVectorStore(db), admission: admission,
		jobs: indexcontrol.NewSQLiteJobStore(db), clock: clock,
		cancels: map[string]context.CancelFunc{},
	}
	index.engine = retrieval.HybridEngine{
		Planner: retrieval.QueryPlanner{}, Lexical: lexical, Semantic: gateway.GatewaySemanticSearcher{Store: retrieval.NewSQLiteVectorStore(db)}, Freshness: lexical,
		Admission: admission, Cache: retrieval.NewResultCache(128, 8*1024*1024),
		Flights: retrieval.NewQueryFlights(), RRFK: 60,
	}
	return index, nil
}

// Bootstrap is idempotent and non-blocking. It preserves an existing active
// generation; otherwise it builds and promotes a fresh generation in the
// background while status truthfully reports the durable job.
func (index *ProductionIndex) Bootstrap() {
	if index == nil {
		return
	}
	index.bootstrap.Do(func() {
		ctx := context.Background()
		interrupted, _ := index.jobs.RecoverInterrupted(ctx, index.clock.Now())
		for _, job := range interrupted {
			job.State = indexcontrol.JobFailed
			job.Error = "process restarted before completion; start a fresh shadow generation"
			job.UpdatedAt = index.clock.Now()
			_ = index.jobs.Update(ctx, job)
			_ = index.catalog.FailGeneration(ctx, job.Generation, job.Error)
		}
		go index.watch(ctx)
		if _, err := index.catalog.Active(ctx); err == nil {
			return
		} else if !errors.Is(err, catalog.ErrNoActiveGeneration) {
			return
		}
		generation := fmt.Sprintf("bootstrap-%d", index.clock.Now().UnixNano())
		_, _ = index.start(generation, true)
	})
}

func (index *ProductionIndex) Search(ctx context.Context, req *factsv1.SearchRequest) (*factsv1.SearchResponse, error) {
	index.servingMu.RLock()
	defer index.servingMu.RUnlock()
	if req == nil || strings.TrimSpace(req.GetQuery()) == "" {
		return nil, fmt.Errorf("query is required")
	}
	if err := validateImportSearchScope(req); err != nil {
		return nil, err
	}
	generation, err := index.catalog.Active(ctx)
	if err != nil {
		return nil, fmt.Errorf("indexed search unavailable: %w", err)
	}
	if hasRequestedFamily(req, factsv1.FactFamily_FACT_FAMILY_IMPORTS) && !generationSupportsImportLanguages(generation.Policy, req.GetLanguages()) {
		return nil, fmt.Errorf("%w; reconcile Code Facts before searching the requested import languages", errImportIndexUnavailable)
	}
	limit := int(req.GetLimit())
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	query := retrieval.Query{
		Text: req.GetQuery(), Limit: limit, Generation: generation.ID,
		Roles: append([]string(nil), req.GetRoles()...), Languages: append([]string(nil), req.GetLanguages()...),
	}
	query.Target, query.Scope = indexedTarget(req.GetTarget(), req.GetScope())
	query.Families, err = indexedFamilies(req.GetFamilies())
	if err != nil {
		return nil, err
	}
	if err := validateImportSearchQuery(req, &query); err != nil {
		return nil, err
	}
	if strings.HasPrefix(query.Scope, "scenario:") && containsString(query.Families, "contract") {
		query.Target = "packages/proto/schemas/" + strings.TrimPrefix(query.Scope, "scenario:")
		query.Scope = ""
	}
	engine := index.engine
	if engine.Planner.Plan(query).UseSemantic {
		policy, policyErr := aisearch.ResolveEmbeddingPolicy(ctx, retrieval.EmbeddingRole)
		if policyErr != nil {
			// Force a separate cache identity and leave the lexical leg available.
			query.EmbeddingModel = "policy-unavailable"
			engine.Semantic = nil
		} else {
			query.EmbeddingModel, query.EmbeddingDimensions = policy.Model, policy.Dimensions
		}
	}
	response, err := engine.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	result := &factsv1.SearchResponse{Generation: generation.ID, DegradedStages: append([]string(nil), response.Degraded...)}
	for _, candidate := range response.Results {
		if result.RetrievalRegime == "" {
			result.RetrievalRegime = string(candidate.Regime)
		}
		evidenceStatus := factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN
		if candidate.Kind == "imports_partial" {
			evidenceStatus = factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN
			result.DegradedStages = append(result.DegradedStages, "import_index_incomplete")
		}
		hit := &factsv1.SearchHit{
			Id: candidate.ID, Title: candidate.Title, Text: candidate.Text, Score: candidate.Score,
			Path: candidate.Path, Analyzer: "code-facts.sqlite-fts", EvidenceStatus: evidenceStatus,
			FactKind: candidate.Kind, SourceHash: candidate.SourceHash, Generation: candidate.Generation,
			Role: candidate.Role, Scope: candidate.Scope, RetrievalRegime: string(candidate.Regime),
			RetrievalExplanation: candidate.Explanation, ProofStatus: candidate.Proof,
			StartLine: int32(candidate.StartLine), EndLine: int32(candidate.EndLine),
		}
		factorNames := make([]string, 0, len(candidate.ScoreFactors))
		for name := range candidate.ScoreFactors {
			factorNames = append(factorNames, name)
		}
		sort.Strings(factorNames)
		for _, name := range factorNames {
			hit.RankFactors = append(hit.RankFactors, &factsv1.SearchRankFactor{Name: name, Value: candidate.ScoreFactors[name]})
		}
		for _, evidence := range candidate.RankEvidence {
			hit.RankFactors = append(hit.RankFactors, &factsv1.SearchRankFactor{Name: "retrieval_leg", Leg: evidence.Leg, Rank: int32(evidence.Rank), Value: evidence.Score})
		}
		result.Results = append(result.Results, hit)
	}
	result.DegradedStages = uniqueSortedStrings(result.DegradedStages)
	return result, nil
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func indexedTarget(target *factsv1.CodeTarget, scope string) (pathPrefix, indexedScope string) {
	scope = strings.TrimSpace(scope)
	switch {
	case strings.HasPrefix(scope, "path:"):
		return strings.TrimSpace(strings.TrimPrefix(scope, "path:")), ""
	case scope != "" && scope != "global" && scope != "project:":
		indexedScope = scope
	}
	if target == nil {
		return pathPrefix, indexedScope
	}
	switch target.GetKind() {
	case factsv1.TargetKind_TARGET_KIND_SCENARIO:
		if target.GetScenario() != "" {
			pathPrefix = "scenarios/" + target.GetScenario()
			if indexedScope == "" {
				indexedScope = "scenario:" + target.GetScenario()
			}
		}
	case factsv1.TargetKind_TARGET_KIND_PATH, factsv1.TargetKind_TARGET_KIND_MODULE:
		pathPrefix = filepath.ToSlash(strings.TrimPrefix(filepath.Clean(target.GetPath()), string(filepath.Separator)))
	}
	return pathPrefix, indexedScope
}

func indexedFamilies(families []factsv1.FactFamily) ([]string, error) {
	seen := map[string]struct{}{}
	for _, family := range families {
		var kind string
		switch family {
		case factsv1.FactFamily_FACT_FAMILY_SYMBOLS, factsv1.FactFamily_FACT_FAMILY_REFERENCES, factsv1.FactFamily_FACT_FAMILY_CALLS:
			kind = "source"
		case factsv1.FactFamily_FACT_FAMILY_IMPORTS:
			kind = "imports,imports_partial"
		case factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION, factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS:
			kind = "contract"
		case factsv1.FactFamily_FACT_FAMILY_ALL:
			return nil, nil
		default:
			return nil, fmt.Errorf("ranked Code Facts search does not index family %s", family)
		}
		if kind != "" {
			for _, indexedKind := range strings.Split(kind, ",") {
				seen[indexedKind] = struct{}{}
			}
		}
	}
	result := make([]string, 0, len(seen))
	for kind := range seen {
		result = append(result, kind)
	}
	sort.Strings(result)
	return result, nil
}

func validateImportSearchScope(req *factsv1.SearchRequest) error {
	importsRequested := false
	allRequested := false
	for _, family := range req.GetFamilies() {
		if family == factsv1.FactFamily_FACT_FAMILY_IMPORTS {
			importsRequested = true
		}
		if family == factsv1.FactFamily_FACT_FAMILY_ALL {
			allRequested = true
		}
	}
	if !importsRequested {
		return nil
	}
	if allRequested {
		return fmt.Errorf("FACT_FAMILY_IMPORTS cannot be combined with FACT_FAMILY_ALL")
	}
	target := req.GetTarget()
	if target == nil || target.GetKind() != factsv1.TargetKind_TARGET_KIND_SCENARIO || !validImportScenarioSlug(target.GetScenario()) {
		return fmt.Errorf("FACT_FAMILY_IMPORTS requires one valid TARGET_KIND_SCENARIO target")
	}
	if scope := strings.TrimSpace(req.GetScope()); scope != "" && scope != "scenario:"+target.GetScenario() {
		return fmt.Errorf("FACT_FAMILY_IMPORTS scope must match target scenario %q", target.GetScenario())
	}
	for _, language := range req.GetLanguages() {
		if !supportedImportLanguage(language) {
			return fmt.Errorf("FACT_FAMILY_IMPORTS ranked search supports Go, TypeScript, or JavaScript source")
		}
	}
	return nil
}

func validImportScenarioSlug(value string) bool {
	if value == "" || len(value) > 64 {
		return false
	}
	first := value[0]
	if (first < 'a' || first > 'z') && (first < '0' || first > '9') {
		return false
	}
	for _, char := range value {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
			return false
		}
	}
	return true
}

func validateImportSearchQuery(req *factsv1.SearchRequest, query *retrieval.Query) error {
	if req == nil || query == nil || !hasRequestedFamily(req, factsv1.FactFamily_FACT_FAMILY_IMPORTS) {
		return nil
	}
	if req.GetTarget() == nil {
		return fmt.Errorf("FACT_FAMILY_IMPORTS requires one valid TARGET_KIND_SCENARIO target")
	}
	normalized, err := retrieval.NormalizeQuery(*query)
	if err != nil {
		return err
	}
	target := "scenarios/" + req.GetTarget().GetScenario()
	scope := "scenario:" + req.GetTarget().GetScenario()
	if normalized.Target != target || normalized.Scope != scope {
		return fmt.Errorf("FACT_FAMILY_IMPORTS query filters must stay within %q", scope)
	}
	allowedFamilies, err := indexedFamilies(req.GetFamilies())
	if err != nil {
		return err
	}
	allowed := make(map[string]struct{}, len(allowedFamilies))
	for _, family := range allowedFamilies {
		allowed[family] = struct{}{}
	}
	for _, family := range normalized.Families {
		if _, exists := allowed[family]; !exists {
			return fmt.Errorf("FACT_FAMILY_IMPORTS query cannot add family filter %q", family)
		}
	}
	requestedLanguages := make(map[string]struct{}, len(req.GetLanguages()))
	for _, language := range req.GetLanguages() {
		requestedLanguages[strings.ToLower(strings.TrimSpace(language))] = struct{}{}
	}
	for _, language := range normalized.Languages {
		normalizedLanguage := strings.ToLower(strings.TrimSpace(language))
		if !supportedImportLanguage(normalizedLanguage) {
			return fmt.Errorf("FACT_FAMILY_IMPORTS ranked search supports Go, TypeScript, or JavaScript source")
		}
		if len(requestedLanguages) > 0 {
			if _, ok := requestedLanguages[normalizedLanguage]; !ok {
				return fmt.Errorf("FACT_FAMILY_IMPORTS query cannot override the requested language scope")
			}
		}
	}
	*query = normalized
	return nil
}

func hasRequestedFamily(req *factsv1.SearchRequest, expected factsv1.FactFamily) bool {
	if req == nil {
		return false
	}
	for _, family := range req.GetFamilies() {
		if family == expected {
			return true
		}
	}
	return false
}

func generationHasImportEdges(policy string) bool {
	return strings.Contains(policy, "polyglot-import-edges-v1")
}

func generationSupportsImportLanguages(policy string, languages []string) bool {
	if generationHasImportEdges(policy) {
		return true
	}
	if !generationHasGoImportEdges(policy) || len(languages) == 0 {
		return false
	}
	for _, language := range languages {
		if !strings.EqualFold(strings.TrimSpace(language), "go") {
			return false
		}
	}
	return true
}

func supportedImportLanguage(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "go", "typescript", "javascript":
		return true
	default:
		return false
	}
}

func generationHasGoImportEdges(policy string) bool {
	return strings.Contains(policy, "go-import-edges-v1")
}

func uniqueSortedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func (index *ProductionIndex) Reconcile(_ context.Context, _ string) (indexcontrol.Job, error) {
	generation := fmt.Sprintf("reconcile-%d", index.clock.Now().UnixNano())
	return index.start(generation, true)
}

func (index *ProductionIndex) StartShadow(_ context.Context, generation string) (indexcontrol.Job, error) {
	return index.start(generation, false)
}

func (index *ProductionIndex) start(generation string, promote bool) (indexcontrol.Job, error) {
	if strings.TrimSpace(generation) == "" {
		return indexcontrol.Job{}, fmt.Errorf("shadow generation is required")
	}
	now := index.clock.Now()
	descriptorDigest, err := fileDigest(filepath.Join(index.repoRoot, "packages", "proto", "gen", "descriptor", "image.binpb"))
	if err != nil {
		return indexcontrol.Job{}, fmt.Errorf("read descriptor image: %w", err)
	}
	if err := index.catalog.BeginGeneration(context.Background(), catalog.Generation{ID: generation, Policy: indexPolicy, DescriptorDigest: descriptorDigest, CreatedAt: now}); err != nil {
		return indexcontrol.Job{}, err
	}
	job := indexcontrol.Job{ID: "reindex-" + fmt.Sprint(now.UnixNano()), Kind: "reindex", State: indexcontrol.JobQueued, Generation: generation, CreatedAt: now, UpdatedAt: now}
	if err := index.jobs.Create(context.Background(), job); err != nil {
		_ = index.catalog.FailGeneration(context.Background(), generation, err.Error())
		return indexcontrol.Job{}, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	index.mu.Lock()
	index.cancels[job.ID] = cancel
	index.mu.Unlock()
	go index.runBuild(ctx, job, promote)
	return job, nil
}

func (index *ProductionIndex) runBuild(ctx context.Context, job indexcontrol.Job, promote bool) {
	defer func() {
		index.mu.Lock()
		delete(index.cancels, job.ID)
		index.mu.Unlock()
	}()
	job.State, job.UpdatedAt = indexcontrol.JobRunning, index.clock.Now()
	_ = index.jobs.Update(context.Background(), job)
	err := index.buildGeneration(ctx, &job)
	if err == nil && promote {
		// Replay working-tree events that may have landed after discovery began.
		// The refresh is generation-scoped and atomic per file.
		for pass := 0; pass < 2 && err == nil; pass++ {
			paths, listErr := index.dirtyPaths(ctx)
			if listErr != nil {
				err = listErr
				break
			}
			for _, path := range paths {
				err = index.refreshPath(ctx, job.Generation, path)
				if err != nil {
					break
				}
			}
		}
	}
	if err == nil && promote {
		err = index.Promote(ctx, job.Generation)
	}
	switch {
	case errors.Is(err, context.Canceled):
		job.State = indexcontrol.JobCancelled
		_ = index.catalog.FailGeneration(context.Background(), job.Generation, "index build cancelled")
	case err != nil:
		job.State, job.Error = indexcontrol.JobFailed, err.Error()
		_ = index.catalog.FailGeneration(context.Background(), job.Generation, job.Error)
	default:
		job.State = indexcontrol.JobSucceeded
	}
	job.UpdatedAt = index.clock.Now()
	_ = index.jobs.Update(context.Background(), job)
}

const (
	// watchDebounce is how long the loop waits after the last watcher event
	// before it reconciles; a save burst from an editor or a build collapses
	// into one refresh.
	watchDebounce = 2 * time.Second
	// watchMinInterval bounds how often a continuously dirty tree may shell
	// out to git status: at most one dirty-path discovery per interval, with
	// events that arrive during a run coalesced into a single follow-up run.
	watchMinInterval = 10 * time.Second
	// watchBackstop is the manifest audit period that repairs missed events.
	watchBackstop = 5 * time.Minute
)

// watchLoopConfig holds the timing seams of the watch loop so tests can drive
// the debounce and spacing rules with short intervals.
type watchLoopConfig struct {
	debounce    time.Duration
	minInterval time.Duration
	backstop    time.Duration
	now         func() time.Time
}

func (index *ProductionIndex) watch(ctx context.Context) {
	roots := []string{"scenarios", "packages", "cmd/vrooli", "internal", "resources"}
	absoluteRoots := make([]string, 0, len(roots))
	for _, root := range roots {
		absoluteRoots = append(absoluteRoots, filepath.Join(index.repoRoot, filepath.FromSlash(root)))
	}
	watcher, _ := fswatch.New(absoluteRoots)
	if watcher != nil {
		defer watcher.Close()
	}
	events := (<-chan struct{})(nil)
	if watcher != nil {
		events = watcher.Events()
	}
	refresh := func() {
		paths, err := index.dirtyPaths(ctx)
		if err == nil && len(paths) > 0 {
			if paths, err = index.filterDrift(ctx, paths); err == nil && len(paths) > 0 {
				if len(paths) > 256 {
					paths = paths[:256]
				}
				_ = index.refreshDirtyBatch(ctx, paths)
			}
		}
	}
	audit := func() { _ = index.auditManifest(ctx) }
	runWatchLoop(ctx, events, watchLoopConfig{debounce: watchDebounce, minInterval: watchMinInterval, backstop: watchBackstop}, refresh, audit)
}

// runWatchLoop turns raw watcher wake-ups into bounded refresh runs.
//
//   - The first run happens immediately: the watcher is started
//     asynchronously by the service bootstrap, so a source edit can race the
//     startup window and one reconciliation after the native watches are
//     installed closes it.
//   - Events arm a single debounce timer; a burst becomes one run.
//   - Consecutive runs are spaced at least minInterval apart, measured from
//     the start of the previous run. Events that arrive while a run is in
//     progress or inside the spacing window are coalesced into exactly one
//     follow-up run at the end of the window.
//   - The backstop audit repairs events the watcher missed.
func runWatchLoop(ctx context.Context, events <-chan struct{}, config watchLoopConfig, refresh, audit func()) {
	if config.now == nil {
		config.now = time.Now
	}
	if config.backstop <= 0 {
		config.backstop = watchBackstop
	}
	backstop := time.NewTimer(config.backstop)
	defer backstop.Stop()
	debounce := time.NewTimer(time.Hour)
	if !debounce.Stop() {
		<-debounce.C
	}
	defer debounce.Stop()
	pending := false
	lastRun := config.now()
	refresh()
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-events:
			if !ok {
				events = nil
				continue
			}
			if pending {
				continue
			}
			pending = true
			wait := config.debounce
			if remaining := config.minInterval - config.now().Sub(lastRun); remaining > wait {
				wait = remaining
			}
			debounce.Reset(wait)
		case <-debounce.C:
			pending = false
			lastRun = config.now()
			refresh()
		case <-backstop.C:
			audit()
			backstop.Reset(config.backstop)
		}
	}
}

func (index *ProductionIndex) filterDrift(ctx context.Context, paths []string) ([]string, error) {
	generation, err := index.catalog.Active(ctx)
	if err != nil {
		return nil, err
	}
	return index.filterGenerationDrift(ctx, generation.ID, paths)
}

func (index *ProductionIndex) filterGenerationDrift(ctx context.Context, generation string, paths []string) ([]string, error) {
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		path = filepath.ToSlash(filepath.Clean(path))
		id := catalog.StableFileID(path)
		var indexedHash string
		var indexedSize, indexedModTime int64
		dbErr := index.db.QueryRowContext(ctx, `SELECT content_hash,size_bytes,mod_time_unix_nano FROM code_facts_source_files WHERE generation_id=? AND id=?`, generation, id).Scan(&indexedHash, &indexedSize, &indexedModTime)
		if dbErr != nil && !errors.Is(dbErr, sql.ErrNoRows) {
			return nil, dbErr
		}
		absolutePath := filepath.Join(index.repoRoot, filepath.FromSlash(path))
		info, statErr := os.Stat(absolutePath)
		if dbErr == nil && statErr == nil && info.Mode().IsRegular() && info.Size() == indexedSize && info.ModTime().UnixNano() == indexedModTime {
			continue
		}
		snapshot, inspectErr := index.inspector.Inspect(ctx, absolutePath)
		if errors.Is(inspectErr, os.ErrNotExist) || errors.Is(inspectErr, catalog.ErrNotRegularFile) {
			if dbErr == nil {
				result = append(result, path)
			}
			continue
		}
		if inspectErr != nil {
			return nil, inspectErr
		}
		classification := catalog.Classify(path, snapshot.Prefix)
		if classification.Role == catalog.RoleTransient {
			if dbErr == nil {
				result = append(result, path)
			}
			continue
		}
		if dbErr != nil || indexedHash != snapshot.Hash {
			result = append(result, path)
		}
	}
	return result, nil
}

func (index *ProductionIndex) refreshBatch(ctx context.Context, paths []string) error {
	index.refreshMu.Lock()
	defer index.refreshMu.Unlock()
	generation, err := index.catalog.Active(ctx)
	if err != nil {
		return err
	}
	return index.refreshBatchLocked(ctx, generation.ID, paths)
}

func (index *ProductionIndex) refreshDirtyBatch(ctx context.Context, paths []string) error {
	index.refreshMu.Lock()
	defer index.refreshMu.Unlock()
	generation, err := index.catalog.Active(ctx)
	if err != nil {
		return err
	}
	if err := index.catalog.RecordGenerationDirtyPaths(ctx, generation.ID, paths); err != nil {
		return err
	}
	return index.refreshBatchLocked(ctx, generation.ID, paths)
}

func (index *ProductionIndex) refreshBatchLocked(ctx context.Context, generation string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	now := index.clock.Now()
	job := indexcontrol.Job{ID: fmt.Sprintf("reconcile-%d", now.UnixNano()), Kind: "reconcile", State: indexcontrol.JobRunning, Generation: generation, Total: int64(len(paths)), CreatedAt: now, UpdatedAt: now}
	if err := index.jobs.Create(ctx, job); err != nil {
		return err
	}
	for _, path := range paths {
		if err := index.refreshPath(ctx, generation, path); err != nil {
			job.State, job.Error, job.UpdatedAt = indexcontrol.JobFailed, err.Error(), index.clock.Now()
			_ = index.jobs.Update(context.WithoutCancel(ctx), job)
			return err
		}
		job.Progress++
		job.UpdatedAt = index.clock.Now()
		_ = index.jobs.Update(ctx, job)
	}
	job.State, job.UpdatedAt = indexcontrol.JobSucceeded, index.clock.Now()
	return index.jobs.Update(ctx, job)
}

func (index *ProductionIndex) refreshPath(ctx context.Context, generation, path string) error {
	index.servingMu.Lock()
	defer index.servingMu.Unlock()
	return index.refreshPathLocked(ctx, generation, path)
}

func (index *ProductionIndex) refreshPathLocked(ctx context.Context, generation, path string) error {
	path = filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
	if path == "." || path == "" || filepath.IsAbs(path) || strings.HasPrefix(path, "../") || !governedIndexPath(path) {
		return nil
	}
	id := catalog.StableFileID(path)
	snapshot, err := index.inspector.Inspect(ctx, filepath.Join(index.repoRoot, filepath.FromSlash(path)))
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, catalog.ErrNotRegularFile) {
		err = index.lexical.ApplySourceChange(ctx, generation, id, nil, nil)
		if err == nil {
			index.engine.Cache.Reset()
		}
		return err
	}
	if err != nil {
		return err
	}
	classification := catalog.Classify(path, snapshot.Prefix)
	if classification.Role == catalog.RoleTransient {
		err = index.lexical.ApplySourceChange(ctx, generation, id, nil, nil)
		if err == nil {
			index.engine.Cache.Reset()
		}
		return err
	}
	var currentHash string
	err = index.db.QueryRowContext(ctx, `SELECT content_hash FROM code_facts_source_files WHERE generation_id=? AND id=?`, generation, id).Scan(&currentHash)
	if err == nil && currentHash == snapshot.Hash {
		return nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	source := catalog.SourceFile{Generation: generation, ID: id, Path: path, Language: classification.Language.Name, Role: classification.Role, Scope: classification.Scope, Authority: classification.Authority, Owner: classification.Owner, Hash: snapshot.Hash, Size: snapshot.Size, ModTime: snapshot.ModTime, Searchable: classification.Searchable}
	var documents []retrieval.Document
	if source.Searchable {
		documents, err = index.documents(ctx, source)
		if err != nil {
			return err
		}
	}
	record := &retrieval.SourceRecord{ID: source.ID, Path: source.Path, Language: source.Language, Role: string(source.Role), Scope: source.Scope, Authority: source.Authority, Owner: source.Owner, Hash: source.Hash, Size: source.Size, ModTimeUnixNano: source.ModTime.UnixNano(), Searchable: source.Searchable}
	err = index.lexical.ApplySourceChange(ctx, generation, id, record, documents)
	if err == nil {
		index.engine.Cache.Reset()
	}
	return err
}

func governedIndexPath(path string) bool {
	for _, root := range []string{"scenarios/", "packages/", "cmd/vrooli/", "internal/", "resources/"} {
		if strings.HasPrefix(path, root) {
			return true
		}
	}
	return false
}

func (index *ProductionIndex) dirtyPaths(ctx context.Context) ([]string, error) {
	command := exec.CommandContext(ctx, "git", "status", "--porcelain=v1", "-z", "--untracked-files=all", "--", "scenarios", "packages", "cmd/vrooli", "internal", "resources")
	command.Dir = index.repoRoot
	payload, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("discover changed source paths: %w", err)
	}
	seen := map[string]struct{}{}
	records := bytes.Split(payload, []byte{0})
	for i := 0; i < len(records); i++ {
		record := string(records[i])
		if len(record) < 4 {
			continue
		}
		status, path := record[:2], record[3:]
		seen[filepath.ToSlash(path)] = struct{}{}
		if (strings.Contains(status, "R") || strings.Contains(status, "C")) && i+1 < len(records) {
			i++
			seen[filepath.ToSlash(string(records[i]))] = struct{}{}
		}
	}
	paths := make([]string, 0, len(seen))
	for path := range seen {
		if governedIndexPath(path) {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

type manifestAuditPlan struct {
	generation string
	revision   string
	paths      []string
}

func (index *ProductionIndex) manifestDrift(ctx context.Context) ([]string, error) {
	plan, err := index.planManifestAudit(ctx)
	return plan.paths, err
}

func (index *ProductionIndex) auditManifest(ctx context.Context) error {
	index.refreshMu.Lock()
	defer index.refreshMu.Unlock()
	plan, err := index.planManifestAudit(ctx)
	if err != nil {
		return err
	}
	paths, err := index.filterGenerationDrift(ctx, plan.generation, plan.paths)
	if err != nil {
		return err
	}
	if err := index.refreshBatchLocked(ctx, plan.generation, paths); err != nil {
		return err
	}
	if plan.revision == "" {
		return nil
	}
	currentRevision, err := index.gitRevision(ctx)
	if err != nil {
		return err
	}
	if currentRevision != plan.revision {
		return nil
	}
	dirty, err := index.dirtyPaths(ctx)
	if err != nil {
		return err
	}
	return index.catalog.CompleteGenerationAudit(ctx, plan.generation, plan.revision, dirty)
}

func (index *ProductionIndex) planManifestAudit(ctx context.Context) (manifestAuditPlan, error) {
	generation, err := index.catalog.Active(ctx)
	if err != nil {
		return manifestAuditPlan{}, err
	}
	return index.planGenerationAudit(ctx, generation.ID)
}

func (index *ProductionIndex) planGenerationAudit(ctx context.Context, generation string) (manifestAuditPlan, error) {
	plan := manifestAuditPlan{generation: generation}
	var err error
	plan.revision, err = index.gitRevision(ctx)
	if err != nil {
		return manifestAuditPlan{}, err
	}
	storedRevision, revisionErr := index.catalog.GenerationRevision(ctx, generation)
	if revisionErr != nil && !errors.Is(revisionErr, catalog.ErrRevisionNotFound) {
		return manifestAuditPlan{}, revisionErr
	}
	if errors.Is(revisionErr, catalog.ErrRevisionNotFound) || plan.revision == "" || !validGitRevision(storedRevision) {
		plan.paths, err = index.fullManifestDrift(ctx, generation)
		return plan, err
	}
	var candidates []string
	if storedRevision != plan.revision {
		candidates, err = index.gitChangedPaths(ctx, storedRevision, plan.revision)
		if err != nil {
			plan.paths, err = index.fullManifestDrift(ctx, generation)
			return plan, err
		}
	}
	dirty, err := index.dirtyPaths(ctx)
	if err != nil {
		return manifestAuditPlan{}, err
	}
	retained, err := index.catalog.GenerationDirtyPaths(ctx, generation)
	if err != nil {
		return manifestAuditPlan{}, err
	}
	inventory, err := index.inventoryDrift(ctx, generation)
	if err != nil {
		return manifestAuditPlan{}, err
	}
	plan.paths = uniqueSortedPaths(candidates, dirty, retained, inventory)
	return plan, nil
}

func (index *ProductionIndex) fullManifestDrift(ctx context.Context, generation string) ([]string, error) {
	iterator, err := (catalog.GitDiscoverer{RepoRoot: index.repoRoot, Starter: catalog.ExecCommandStarter{}, Inspector: index.inspector}).Open(ctx)
	if err != nil {
		return nil, err
	}
	defer iterator.Close()
	current := map[string]string{}
	for {
		source, ok, nextErr := iterator.Next(ctx)
		if nextErr != nil {
			return nil, nextErr
		}
		if !ok {
			break
		}
		current[source.Path] = source.Hash
	}
	indexed := map[string]string{}
	rows, err := index.db.QueryContext(ctx, `SELECT path,content_hash FROM code_facts_source_files WHERE generation_id=?`, generation)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var path, hash string
		if err := rows.Scan(&path, &hash); err != nil {
			return nil, err
		}
		indexed[path] = hash
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	for path, hash := range current {
		if indexed[path] != hash {
			seen[path] = struct{}{}
		}
	}
	for path := range indexed {
		if _, exists := current[path]; !exists {
			seen[path] = struct{}{}
		}
	}
	return sortedPathSet(seen), nil
}

func (index *ProductionIndex) inventoryDrift(ctx context.Context, generation string) ([]string, error) {
	current, err := index.gitPaths(ctx)
	if err != nil {
		return nil, err
	}
	indexed := map[string]struct{}{}
	rows, err := index.db.QueryContext(ctx, `SELECT path FROM code_facts_source_files WHERE generation_id=?`, generation)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		indexed[path] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(current)+len(indexed))
	for _, path := range current {
		if _, exists := indexed[path]; !exists {
			seen[path] = struct{}{}
		}
		delete(indexed, path)
	}
	for path := range indexed {
		seen[path] = struct{}{}
	}
	return sortedPathSet(seen), nil
}

func (index *ProductionIndex) gitRevision(ctx context.Context) (string, error) {
	command := exec.CommandContext(ctx, "git", "rev-parse", "--verify", "HEAD")
	command.Dir = index.repoRoot
	payload, err := command.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 128 {
			return "", nil
		}
		return "", fmt.Errorf("read repository revision: %w", err)
	}
	revision := strings.TrimSpace(string(payload))
	if !validGitRevision(revision) {
		return "", fmt.Errorf("git returned invalid repository revision %q", revision)
	}
	return revision, nil
}

func validGitRevision(revision string) bool {
	if len(revision) != 40 && len(revision) != 64 {
		return false
	}
	_, err := hex.DecodeString(revision)
	return err == nil
}

func (index *ProductionIndex) gitChangedPaths(ctx context.Context, fromRevision, toRevision string) ([]string, error) {
	if !validGitRevision(fromRevision) || !validGitRevision(toRevision) {
		return nil, fmt.Errorf("repository revision delta requires valid object ids")
	}
	command := exec.CommandContext(ctx, "git", "diff", "--name-only", "-z", "--no-renames", fromRevision, toRevision, "--", "scenarios", "packages", "cmd/vrooli", "internal", "resources")
	command.Dir = index.repoRoot
	payload, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("discover repository revision delta: %w", err)
	}
	return governedNULPaths(payload), nil
}

func (index *ProductionIndex) gitPaths(ctx context.Context) ([]string, error) {
	command := exec.CommandContext(ctx, "git", "ls-files", "-co", "--exclude-standard", "-z", "--", "scenarios", "packages", "cmd/vrooli", "internal", "resources")
	command.Dir = index.repoRoot
	payload, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("discover repository path inventory: %w", err)
	}
	return governedNULPaths(payload), nil
}

func governedNULPaths(payload []byte) []string {
	seen := map[string]struct{}{}
	for _, raw := range bytes.Split(payload, []byte{0}) {
		path := filepath.ToSlash(string(raw))
		if governedIndexPath(path) && catalog.Classify(path, nil).Role != catalog.RoleTransient {
			seen[path] = struct{}{}
		}
	}
	return sortedPathSet(seen)
}

func uniqueSortedPaths(groups ...[]string) []string {
	seen := map[string]struct{}{}
	for _, paths := range groups {
		for _, path := range paths {
			path = filepath.ToSlash(filepath.Clean(path))
			if governedIndexPath(path) {
				seen[path] = struct{}{}
			}
		}
	}
	return sortedPathSet(seen)
}

func sortedPathSet(seen map[string]struct{}) []string {
	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func (index *ProductionIndex) buildGeneration(ctx context.Context, job *indexcontrol.Job) error {
	embeddingPolicy, err := aisearch.ResolveEmbeddingPolicy(ctx, retrieval.EmbeddingRole)
	if err != nil {
		return fmt.Errorf("resolve Code Facts embedding role %q: %w", retrieval.EmbeddingRole, err)
	}
	cardPolicy := retrieval.DefaultCardPolicy()
	cardPolicy.Model, cardPolicy.Dimensions = embeddingPolicy.Model, embeddingPolicy.Dimensions
	cardPolicy.Version = fmt.Sprintf("%s:%s:%s", cardPolicy.Version, embeddingPolicy.Role, embeddingPolicy.PolicySchemaVersion)
	generationPolicy := fmt.Sprintf("%s|embedding_role=%s|embedding_model=%s|embedding_dimensions=%d|embedding_policy_schema=%s",
		indexPolicy, embeddingPolicy.Role, embeddingPolicy.Model, embeddingPolicy.Dimensions, embeddingPolicy.PolicySchemaVersion)
	if _, err := index.db.ExecContext(ctx, `UPDATE code_facts_generations SET policy=?,updated_at_unix=? WHERE id=? AND state='shadow'`, generationPolicy, index.clock.Now().Unix(), job.Generation); err != nil {
		return fmt.Errorf("record Code Facts embedding generation policy: %w", err)
	}
	cardIndexer := retrieval.CardIndexer{
		Embedder: gateway.NewGatewayEmbedder(embeddingPolicy.Model, embeddingPolicy.Dimensions, gateway.TaskPrefixForEmbeddingModel(embeddingPolicy.Model, false)),
		Store:    index.vectors, Admission: index.admission, BatchSize: 32,
	}
	buildRevision, err := index.gitRevision(ctx)
	if err != nil {
		return err
	}
	descriptorDigest, err := fileDigest(filepath.Join(index.repoRoot, "packages", "proto", "gen", "descriptor", "image.binpb"))
	if err != nil {
		return fmt.Errorf("read descriptor image: %w", err)
	}
	builder := catalog.Builder{
		Repository: index.catalog,
		Discoverer: catalog.GitDiscoverer{RepoRoot: index.repoRoot, Starter: catalog.ExecCommandStarter{}, Inspector: index.inspector},
		Clock:      index.clock, BatchSize: 256, SkipBegin: true,
	}
	build, err := builder.Build(ctx, catalog.Generation{ID: job.Generation, Policy: indexPolicy, DescriptorDigest: descriptorDigest})
	if err != nil {
		return err
	}
	job.Total = int64(build.Files)
	job.UpdatedAt = index.clock.Now()
	_ = index.jobs.Update(context.Background(), *job)
	token := ""
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		page, err := index.catalog.PageFiles(ctx, job.Generation, token, indexBatchSize)
		if err != nil {
			return err
		}
		documents := make([]retrieval.Document, 0, len(page.Files))
		for _, source := range page.Files {
			if !source.Searchable {
				job.Progress++
				continue
			}
			chunks, err := index.documents(ctx, source)
			if err != nil {
				return err
			}
			documents = append(documents, chunks...)
			job.Progress++
		}
		if len(documents) > 0 {
			if err := index.lexical.Upsert(ctx, job.Generation, documents); err != nil {
				return err
			}
			cardDocuments := make([]retrieval.Document, len(documents))
			copy(cardDocuments, documents)
			cards, _, extractErr := (retrieval.CardExtractor{Policy: cardPolicy}).Extract(job.Generation, cardDocuments)
			if extractErr != nil {
				return fmt.Errorf("extract semantic cards: %w", extractErr)
			}
			if _, err := cardIndexer.Index(ctx, cards); err != nil {
				return fmt.Errorf("index semantic cards through AI Gateway: %w", err)
			}
		}
		job.Cursor, job.UpdatedAt = page.NextToken, index.clock.Now()
		if err := index.jobs.Update(context.Background(), *job); err != nil {
			return err
		}
		if page.NextToken == "" {
			break
		}
		token = page.NextToken
	}
	if err := index.validateContent(ctx, job.Generation); err != nil {
		return err
	}
	currentRevision, revisionErr := index.gitRevision(ctx)
	dirty, dirtyErr := index.dirtyPaths(ctx)
	if revisionErr == nil && dirtyErr == nil && buildRevision != "" && currentRevision == buildRevision && len(dirty) == 0 {
		if err := index.catalog.SetGenerationRevision(ctx, job.Generation, currentRevision); err != nil {
			return err
		}
	}
	return nil
}

func (index *ProductionIndex) documents(ctx context.Context, source catalog.SourceFile) ([]retrieval.Document, error) {
	path := filepath.Join(index.repoRoot, filepath.FromSlash(source.Path))
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open indexed source %q: %w", source.Path, err)
	}
	defer file.Close()
	payload, err := io.ReadAll(io.LimitReader(&contextReader{ctx: ctx, reader: file}, maxIndexedFile))
	if err != nil {
		return nil, fmt.Errorf("read indexed source %q: %w", source.Path, err)
	}
	if len(bytes.TrimSpace(payload)) == 0 {
		return nil, nil
	}
	kind := "source"
	if source.Role == catalog.RoleContract {
		kind = "contract"
	}
	var documents []retrieval.Document
	if isLineIndexedSource(source.Path) {
		documents, err = lineDocuments(source, kind, payload)
		if err != nil {
			return nil, err
		}
	} else {
		documents, err = chunkDocuments(source, kind, payload)
		if err != nil {
			return nil, err
		}
	}
	imports, _, err := importDocuments(source, payload)
	if err != nil {
		return nil, fmt.Errorf("index import edges from %q: %w", source.Path, err)
	}
	return append(documents, imports...), nil
}

func chunkDocuments(source catalog.SourceFile, kind string, payload []byte) ([]retrieval.Document, error) {
	scanner := bufio.NewScanner(bytes.NewReader(payload))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	var documents []retrieval.Document
	var chunk strings.Builder
	startLine, line := 1, 0
	flush := func(endLine int) {
		if chunk.Len() == 0 {
			return
		}
		sequence := len(documents)
		title := filepath.Base(source.Path)
		documents = append(documents, retrieval.Document{
			ID: fmt.Sprintf("%s:chunk:%d", source.ID, sequence), SourceFileID: source.ID, SourceHash: source.Hash,
			Path: source.Path, Language: source.Language, Role: string(source.Role), Scope: source.Scope,
			Authority: source.Authority, Kind: kind, Title: title, ExactText: title,
			Body: chunk.String(), Aliases: []string{source.Path, strings.TrimSuffix(title, filepath.Ext(title))},
			StartLine: startLine, EndLine: endLine,
		})
		chunk.Reset()
		startLine = endLine + 1
	}
	for scanner.Scan() {
		line++
		text := scanner.Text()
		if chunk.Len() > 0 && chunk.Len()+len(text)+1 > indexedChunkBytes {
			flush(line - 1)
		}
		chunk.WriteString(text)
		chunk.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan indexed source %q: %w", source.Path, err)
	}
	flush(line)
	return documents, nil
}

func importDocuments(source catalog.SourceFile, payload []byte) ([]retrieval.Document, bool, error) {
	if source.Language == "typescript" || source.Language == "javascript" {
		return javascriptImportDocuments(source, payload)
	}
	return goImportDocuments(source, payload)
}

func goImportDocuments(source catalog.SourceFile, payload []byte) ([]retrieval.Document, bool, error) {
	if source.Language != "go" || strings.ToLower(filepath.Ext(source.Path)) != ".go" {
		return nil, false, nil
	}
	if len(payload) >= maxIndexedFile {
		line := bytes.Count(payload, []byte("\n")) + 1
		return []retrieval.Document{importIndexPartialDocument(source, line, "source reached the per-file indexed byte cap")}, true, nil
	}
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, source.Path, payload, parser.ImportsOnly)
	if err != nil {
		return []retrieval.Document{importIndexPartialDocument(source, 1, "Go import header could not be parsed")}, true, nil
	}
	documents := make([]retrieval.Document, 0, min(len(parsed.Imports), maxImportEdgesPerFile+1))
	for index, spec := range parsed.Imports {
		if index >= maxImportEdgesPerFile {
			position := fset.Position(spec.Pos())
			documents = append(documents, importIndexPartialDocument(source, position.Line, fmt.Sprintf("import indexing reached the per-file cap of %d edges", maxImportEdgesPerFile)))
			return documents, true, nil
		}
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil || importPath == "" {
			if err == nil {
				err = fmt.Errorf("empty import path")
			}
			return nil, false, fmt.Errorf("decode import path at line %d: %w", fset.Position(spec.Pos()).Line, err)
		}
		position := fset.Position(spec.Pos())
		title := fmt.Sprintf("%s imports %s", source.Path, importPath)
		body := title
		if spec.Name != nil && spec.Name.Name != "_" && spec.Name.Name != "." {
			body += " as " + spec.Name.Name
		}
		documents = append(documents, retrieval.Document{
			ID: fmt.Sprintf("code-facts:import:%s:%d:%s", source.Path, position.Line, importPath), SourceFileID: source.ID,
			SourceHash: source.Hash, Path: source.Path, Language: source.Language, Role: string(source.Role), Scope: source.Scope,
			Authority: source.Authority, Kind: "imports", Title: title, ExactText: importPath, Body: body,
			Aliases: []string{importPath, source.Path, filepath.Base(source.Path)}, StartLine: position.Line, EndLine: position.Line,
		})
	}
	return documents, false, nil
}

func importIndexPartialDocument(source catalog.SourceFile, line int, reason string) retrieval.Document {
	return retrieval.Document{
		ID: fmt.Sprintf("code-facts:imports-partial:%s:%d", source.Path, line), SourceFileID: source.ID,
		SourceHash: source.Hash, Path: source.Path, Language: source.Language, Role: string(source.Role), Scope: source.Scope,
		Authority: source.Authority, Kind: "imports_partial", Title: "Import evidence incomplete",
		ExactText: "Import evidence incomplete", Body: fmt.Sprintf("%s: %s; import results may be incomplete.", source.Path, reason),
		Aliases: []string{source.Path, filepath.Base(source.Path)}, StartLine: line, EndLine: line,
	}
}

func isLineIndexedSource(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".ts", ".tsx", ".mts", ".cts", ".js", ".jsx", ".mjs", ".cjs", ".proto":
		return true
	default:
		return false
	}
}

func lineDocuments(source catalog.SourceFile, kind string, payload []byte) ([]retrieval.Document, error) {
	scanner := bufio.NewScanner(bytes.NewReader(payload))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	documents := make([]retrieval.Document, 0, 64)
	line := 0
	for scanner.Scan() {
		line++
		subject := strings.TrimSpace(scanner.Text())
		if !shouldIndexLine(source.Path, line, subject) {
			continue
		}
		title, exact := sourceTitle(source.Path, line, subject)
		documents = append(documents, retrieval.Document{
			ID: fmt.Sprintf("code-facts:lexical:%s:%d", source.Path, line), SourceFileID: source.ID, SourceHash: source.Hash,
			Path: source.Path, Language: source.Language, Role: string(source.Role), Scope: source.Scope,
			Authority: source.Authority, Kind: kind, Title: title, ExactText: exact, Body: subject,
			Aliases: []string{source.Path, filepath.Base(source.Path)}, StartLine: line, EndLine: line,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan indexed source %q: %w", source.Path, err)
	}
	return documents, nil
}

func sourceTitle(path string, line int, subject string) (string, string) {
	if line == 1 {
		name := filepath.Base(path)
		return name, name
	}
	trimmed := strings.TrimSpace(strings.TrimLeft(subject, "/*"))
	if len(trimmed) > 240 {
		trimmed = trimmed[:240]
	}
	fields := strings.FieldsFunc(trimmed, func(r rune) bool {
		return !(r == '_' || r == '.' || r == '$' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z')
	})
	exact := ""
	for _, field := range fields {
		lower := strings.ToLower(field)
		if lower == "func" || lower == "type" || lower == "class" || lower == "service" || lower == "rpc" || lower == "message" || lower == "enum" || lower == "interface" || lower == "export" || lower == "const" || lower == "var" {
			continue
		}
		if len(field) >= 2 {
			exact = field
			break
		}
	}
	if exact == "" {
		exact = filepath.Base(path)
	}
	return trimmed, exact
}

func shouldIndexLine(path string, line int, subject string) bool {
	if subject == "" {
		return false
	}
	if line == 1 {
		return true
	}
	lower := strings.ToLower(subject)
	if strings.HasPrefix(subject, "//") || strings.HasPrefix(subject, "/*") || strings.HasPrefix(subject, "*") {
		return len(subject) >= 24 && containsAny(lower, "route", "handler", "provider", "search", "contract", "endpoint", "register", "demot", "adoption")
	}
	declaration := strings.TrimLeft(lower, " \t}")
	declaration = strings.TrimPrefix(declaration, "export ")
	declaration = strings.TrimPrefix(declaration, "default ")
	declaration = strings.TrimPrefix(declaration, "async ")
	if containsAnyPrefix(declaration, "func ", "type ", "interface ", "class ", "function ", "service ", "rpc ", "message ", "enum ") {
		return true
	}
	return containsAny(lower, "route", "endpoint", "register", "demot")
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func containsAnyPrefix(value string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func (index *ProductionIndex) validate(ctx context.Context, generation string) error {
	var activeJobs int64
	var sourceDigest, descriptorDigest string
	if err := index.db.QueryRowContext(ctx, `SELECT source_digest,descriptor_digest FROM code_facts_generations WHERE id=? AND state='shadow'`, generation).Scan(&sourceDigest, &descriptorDigest); err != nil {
		return fmt.Errorf("generation %q is not a completed shadow: %w", generation, err)
	}
	if sourceDigest == "" || descriptorDigest == "" {
		return fmt.Errorf("generation %q is missing source or descriptor digest", generation)
	}
	if err := index.validateContent(ctx, generation); err != nil {
		return err
	}
	if err := index.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM code_facts_index_jobs WHERE generation_id=? AND state IN ('queued','running','cancellation_requested','interrupted')`, generation).Scan(&activeJobs); err != nil {
		return err
	}
	if activeJobs != 0 {
		return fmt.Errorf("generation %q still has %d active build jobs", generation, activeJobs)
	}
	return nil
}

func (index *ProductionIndex) validateContent(ctx context.Context, generation string) error {
	var files, documents int64
	if err := index.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM code_facts_source_files WHERE generation_id=? AND searchable=1`, generation).Scan(&files); err != nil {
		return err
	}
	if err := index.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM code_facts_search_documents WHERE generation_id=?`, generation).Scan(&documents); err != nil {
		return err
	}
	if files == 0 || documents == 0 {
		return fmt.Errorf("generation %q is incomplete: searchable_files=%d documents=%d", generation, files, documents)
	}
	return nil
}

func (index *ProductionIndex) Cancel(ctx context.Context, id string) error {
	if err := index.jobs.RequestCancel(ctx, id, index.clock.Now()); err != nil {
		return err
	}
	index.mu.Lock()
	cancel := index.cancels[id]
	index.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

func (index *ProductionIndex) Promote(ctx context.Context, generation string) error {
	index.refreshMu.Lock()
	defer index.refreshMu.Unlock()
	if err := index.reconcileGenerationManifestLocked(ctx, generation); err != nil {
		return err
	}
	if err := index.validate(ctx, generation); err != nil {
		return err
	}
	return index.catalog.Activate(ctx, generation)
}

func (index *ProductionIndex) reconcileGenerationManifestLocked(ctx context.Context, generation string) error {
	for attempt := 0; attempt < 2; attempt++ {
		plan, err := index.planGenerationAudit(ctx, generation)
		if err != nil {
			return err
		}
		paths, err := index.filterGenerationDrift(ctx, generation, plan.paths)
		if err != nil {
			return err
		}
		if err := index.refreshBatchLocked(ctx, generation, paths); err != nil {
			return err
		}
		if plan.revision == "" {
			return nil
		}
		currentRevision, err := index.gitRevision(ctx)
		if err != nil {
			return err
		}
		if currentRevision != plan.revision {
			continue
		}
		dirty, err := index.dirtyPaths(ctx)
		if err != nil {
			return err
		}
		if err := index.catalog.CompleteGenerationAudit(ctx, generation, plan.revision, dirty); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("repository revision changed repeatedly while reconciling generation %q", generation)
}

func (index *ProductionIndex) Rollback(ctx context.Context, generation string) error {
	return index.catalog.Rollback(ctx, generation)
}

func (index *ProductionIndex) Cleanup(ctx context.Context) error {
	var victim string
	err := index.db.QueryRowContext(ctx, `SELECT g.id FROM code_facts_generations g
WHERE g.state='failed' OR (g.state='shadow' AND NOT EXISTS (
 SELECT 1 FROM code_facts_index_jobs j WHERE j.generation_id=g.id
 AND j.state IN ('queued','running','cancellation_requested','interrupted')
)) OR (g.state='retired' AND (
 g.id NOT IN (SELECT id FROM code_facts_generations WHERE state='retired' ORDER BY updated_at_unix DESC LIMIT 1)
 OR EXISTS (SELECT 1 FROM code_facts_generations shadow WHERE shadow.state='shadow')
)) ORDER BY CASE g.state WHEN 'failed' THEN 0 WHEN 'shadow' THEN 1 ELSE 2 END,g.updated_at_unix LIMIT 1`).Scan(&victim)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	// FTS delete triggers are intentionally paid in small transactions so
	// cleanup never monopolizes the scenario's single SQLite connection.
	result, err := index.db.ExecContext(ctx, `DELETE FROM code_facts_search_documents WHERE rowid IN (
 SELECT rowid FROM code_facts_search_documents WHERE generation_id=? LIMIT 2000
)`, victim)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected > 0 {
		return nil
	}
	_, err = index.db.ExecContext(ctx, `DELETE FROM code_facts_generations WHERE id=? AND state!='active'`, victim)
	return err
}

func fileDigest(path string) (string, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (reader *contextReader) Read(payload []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	return reader.reader.Read(payload)
}

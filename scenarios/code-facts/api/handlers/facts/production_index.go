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
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"code-facts/internal/catalog"
	internalfacts "code-facts/internal/facts"
	"code-facts/internal/indexcontrol"
	"code-facts/internal/retrieval"

	repocontract "github.com/vrooli/repo-contract-go"
	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

const (
	indexPolicy       = "code-facts-corpus-v1"
	indexBatchSize    = 16
	maxIndexedFile    = 256 * 1024
	indexedChunkBytes = 8 * 1024
)

type wallClock struct{}

func (wallClock) Now() time.Time { return time.Now().UTC() }

// ProductionIndex owns the durable build path and the source-free query path.
// Source files are opened only by buildGeneration; Search uses the active
// SQLite generation and cannot fall through to repository scanning.
type ProductionIndex struct {
	db        *sql.DB
	repoRoot  string
	catalog   *catalog.SQLiteRepository
	lexical   *retrieval.SQLiteIndex
	engine    retrieval.HybridEngine
	jobs      indexcontrol.JobStore
	clock     wallClock
	mu        sync.Mutex
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
		lexical: lexical, jobs: indexcontrol.NewSQLiteJobStore(db), clock: clock,
		cancels: map[string]context.CancelFunc{},
	}
	index.engine = retrieval.HybridEngine{
		Planner: retrieval.QueryPlanner{}, Lexical: lexical, Freshness: lexical,
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
			job.State = "failed"
			job.Error = "process restarted before completion; start a fresh shadow generation"
			job.UpdatedAt = index.clock.Now()
			_ = index.jobs.Update(ctx, job)
			_ = index.catalog.FailGeneration(ctx, job.Generation, job.Error)
		}
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
	if req == nil || strings.TrimSpace(req.GetQuery()) == "" {
		return nil, fmt.Errorf("query is required")
	}
	generation, err := index.catalog.Active(ctx)
	if err != nil {
		return nil, fmt.Errorf("indexed search unavailable: %w", err)
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
	query.Families = indexedFamilies(req.GetFamilies())
	if strings.HasPrefix(query.Scope, "scenario:") && containsString(query.Families, "contract") {
		query.Target = "packages/proto/schemas/" + strings.TrimPrefix(query.Scope, "scenario:")
		query.Scope = ""
	}
	response, err := index.engine.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	result := &factsv1.SearchResponse{
		Generation: generation.ID, DegradedStages: append([]string(nil), response.Degraded...),
	}
	for _, candidate := range response.Results {
		if result.RetrievalRegime == "" {
			result.RetrievalRegime = string(candidate.Regime)
		}
		hit := &factsv1.SearchHit{
			Id: candidate.ID, Title: candidate.Title, Text: candidate.Text, Score: candidate.Score,
			Path: candidate.Path, Analyzer: "code-facts.sqlite-fts", EvidenceStatus: factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN,
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

func indexedFamilies(families []factsv1.FactFamily) []string {
	seen := map[string]struct{}{}
	for _, family := range families {
		var kind string
		switch family {
		case factsv1.FactFamily_FACT_FAMILY_SYMBOLS, factsv1.FactFamily_FACT_FAMILY_REFERENCES, factsv1.FactFamily_FACT_FAMILY_CALLS:
			kind = "source"
		case factsv1.FactFamily_FACT_FAMILY_PROTO_ADOPTION, factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS:
			kind = "contract"
		case factsv1.FactFamily_FACT_FAMILY_ALL:
			return nil
		}
		if kind != "" {
			seen[kind] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for kind := range seen {
		result = append(result, kind)
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
	job := indexcontrol.Job{ID: "reindex-" + fmt.Sprint(now.UnixNano()), Kind: "reindex", State: "queued", Generation: generation, CreatedAt: now, UpdatedAt: now}
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
	job.State, job.UpdatedAt = "running", index.clock.Now()
	_ = index.jobs.Update(context.Background(), job)
	err := index.buildGeneration(ctx, &job)
	if err == nil && promote {
		job.State, job.UpdatedAt = "succeeded", index.clock.Now()
		_ = index.jobs.Update(context.Background(), job)
		err = index.Promote(ctx, job.Generation)
	}
	switch {
	case errors.Is(err, context.Canceled):
		job.State, job.CancellationRequested = "cancelled", true
		_ = index.catalog.FailGeneration(context.Background(), job.Generation, "index build cancelled")
	case err != nil:
		job.State, job.Error = "failed", err.Error()
		_ = index.catalog.FailGeneration(context.Background(), job.Generation, job.Error)
	default:
		job.State = "succeeded"
	}
	job.UpdatedAt = index.clock.Now()
	_ = index.jobs.Update(context.Background(), job)
}

func (index *ProductionIndex) buildGeneration(ctx context.Context, job *indexcontrol.Job) error {
	descriptorDigest, err := fileDigest(filepath.Join(index.repoRoot, "packages", "proto", "gen", "descriptor", "image.binpb"))
	if err != nil {
		return fmt.Errorf("read descriptor image: %w", err)
	}
	builder := catalog.Builder{
		Repository: index.catalog,
		Discoverer: catalog.GitDiscoverer{RepoRoot: index.repoRoot, Starter: catalog.ExecCommandStarter{}, Inspector: catalog.OSFileInspector{}},
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
	return index.validateContent(ctx, job.Generation)
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
	if isLineIndexedSource(source.Path) {
		return lineDocuments(source, kind, payload)
	}
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

func isLineIndexedSource(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".proto":
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
	if err := index.validate(ctx, generation); err != nil {
		return err
	}
	return index.catalog.Activate(ctx, generation)
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
)) OR (g.state='retired' AND g.id NOT IN (
 SELECT id FROM code_facts_generations WHERE state='retired' ORDER BY updated_at_unix DESC LIMIT 1
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

package files

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

// DefaultCorpus is the research home.
const DefaultCorpus = "research"

const (
	// readSyncBudget bounds the sync a query waits for. Files it could not
	// reach are excluded from results (never served stale) and reported.
	readSyncBudget = 4 * time.Second
	syncBatch      = 200
	candidatePool  = 40
	maxPerFile     = 3
	// rawWeight is the prior on raw material (isRaw). Measured on generated
	// paraphrase queries (owners/C/evidence/search-eval/gen): it roughly
	// doubles recall of curated targets and leaves exact-word ranking intact.
	rawWeight = 0.5
)

// Hit is one cited search result.
type Hit struct {
	Citation, Path, AbsPath, Topic, Feed string
	LineStart, LineEnd                   int
	Heading, Snippet                     string
	Score                                float64
	ModifiedAt                           time.Time
	Truncated                            bool // the file is longer than max_file_kb; only part is indexed
}

// SyncReport says what the read-time sync did.
type SyncReport struct {
	Seen, Changed, Removed, Pending int
	Duration                        time.Duration
	At                              time.Time
	Partial                         bool
	Problems                        []string
	Stats                           WalkStats
}

// Degradation is a skipped or partial leg; the answer is still valid.
type Degradation struct{ Reason, Leg, Detail string }

// Result is one search answer.
type Result struct {
	Hits         []Hit
	Sync         SyncReport
	Mode         string
	Degradations []Degradation
}

// Query selects what to search.
type Query struct {
	Text, Corpus, Topic, Feed string
	Limit                     int
	// IncludeRaw ranks raw material (isRaw) like curated pages instead of
	// lower.
	IncludeRaw bool
}

// Service owns the corpora roots and their shared index.
type Service struct {
	roots    map[string]string
	store    *store
	embedder Embedder
	logger   *log.Logger
	mu       sync.Mutex // one sync at a time
	vectors  vectorIndex
	prioMu   sync.Mutex
	priority []string // chunk hashes a query needed embedded first
	queued   map[string]bool
}

// NewService opens the index in db (a dedicated SQLite cache database) for
// the given corpus → root map. embedder may be nil.
func NewService(ctx context.Context, db *sql.DB, roots map[string]string, embedder Embedder, logger *log.Logger) (*Service, error) {
	st, err := openStore(ctx, db)
	if err != nil {
		return nil, err
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Service{roots: roots, store: st, embedder: embedder, logger: logger, queued: map[string]bool{}}, nil
}

func (s *Service) root(corpus string) (string, string, error) {
	if corpus == "" {
		corpus = DefaultCorpus
	}
	root, ok := s.roots[corpus]
	if !ok {
		return "", "", fmt.Errorf("unknown corpus %q", corpus)
	}
	return corpus, root, nil
}

func validTopic(topic string) error {
	if topic != "" && (strings.ContainsAny(topic, "/\\") || topic == "." || topic == "..") {
		return fmt.Errorf("invalid topic %q", topic)
	}
	return nil
}

// Search syncs the requested scope against the files, then ranks chunks
// lexically (FTS5 BM25) and, when embeddings answer, reranks the candidate
// pool semantically. Lexical search needs nothing but the files.
func (s *Service) Search(ctx context.Context, q Query) (Result, error) {
	corpus, root, err := s.root(q.Corpus)
	if err != nil {
		return Result{}, err
	}
	terms := queryTerms(q.Text)
	if len(terms) == 0 {
		return Result{}, errors.New("query has no searchable terms")
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	report, pending, err := s.Sync(ctx, corpus, q.Topic, readSyncBudget)
	if err != nil {
		return Result{}, err
	}
	result := Result{Sync: report, Mode: "lexical"}
	if report.Partial {
		result.Degradations = append(result.Degradations, Degradation{Reason: "SYNC_PARTIAL", Leg: "sync", Detail: fmt.Sprintf("%d changed files were not reindexed within %s and are excluded; the next read continues", report.Pending, readSyncBudget)})
	}
	if d := problemsDegradation(report.Problems); d != nil {
		result.Degradations = append(result.Degradations, *d)
	}
	pool, exact, err := s.candidates(ctx, corpus, terms, q, pending)
	if err != nil {
		return Result{}, err
	}
	pool, d := s.semantic(ctx, corpus, q, pool, exact > 0, pending)
	if d == nil || d.Reason == "SEMANTIC_PARTIAL" {
		result.Mode = "hybrid"
	}
	if d != nil {
		result.Degradations = append(result.Degradations, *d)
	}
	for i := range pool {
		pool[i].Rank *= authority(pool[i].Path, q.IncludeRaw)
	}
	normalise(pool)
	sort.SliceStable(pool, func(i, j int) bool { return pool[i].Rank > pool[j].Rank })
	for _, c := range distinctFilesFirst(pool, limit) {
		result.Hits = append(result.Hits, Hit{
			Citation: fmt.Sprintf("%s#L%d-L%d", c.Path, c.LineStart, c.LineEnd), Path: c.Path,
			AbsPath: filepath.Join(root, filepath.FromSlash(c.Path)), Topic: c.Topic, Feed: c.Feed,
			LineStart: c.LineStart, LineEnd: c.LineEnd, Heading: c.Heading, Snippet: c.Snippet,
			Score: math.Round(c.Rank*1000) / 1000, ModifiedAt: time.Unix(0, c.MtimeNS).UTC(), Truncated: c.Truncated,
		})
	}
	return result, nil
}

// ProblemsDegradations is problemsDegradation as a list (empty when none).
func ProblemsDegradations(problems []string) []Degradation {
	if d := problemsDegradation(problems); d != nil {
		return []Degradation{*d}
	}
	return nil
}

// problemsDegradation reports what the walk could not read or parse, so a
// broken feeds.json or an unreadable folder is never silent.
func problemsDegradation(problems []string) *Degradation {
	if len(problems) == 0 {
		return nil
	}
	shown := problems[:min(3, len(problems))]
	detail := strings.Join(shown, "; ")
	if len(problems) > len(shown) {
		detail += fmt.Sprintf("; and %d more", len(problems)-len(shown))
	}
	return &Degradation{Reason: "CORPUS_PROBLEMS", Leg: "walk", Detail: detail}
}

// distinctFilesFirst picks up to limit hits from a ranked pool: each file's
// best chunk first, so the top results name as many different files as they
// can, then further chunks (up to maxPerFile per file) in rank order.
func distinctFilesFirst(pool []candidate, limit int) []candidate {
	var out []candidate
	perFile := map[string]int{}
	taken := make([]bool, len(pool))
	for pass := 1; pass <= maxPerFile; pass++ {
		for i, c := range pool {
			if len(out) == limit {
				return out
			}
			if !taken[i] && perFile[c.Path] == pass-1 {
				taken[i] = true
				perFile[c.Path]++
				out = append(out, c)
			}
		}
	}
	return out
}

// candidates runs an all-terms query first and widens to any-term when that
// is too narrow. Rank is normalised to (0,1], higher is better. exact counts
// the leading candidates that hold every term.
func (s *Service) candidates(ctx context.Context, corpus string, terms []string, q Query, pending map[string]bool) (pool []candidate, exact int, err error) {
	quoted := make([]string, len(terms))
	for i, t := range terms {
		quoted[i] = `"` + t + `"`
	}
	seen := map[int64]bool{}
	for leg, match := range []string{strings.Join(quoted, " "), strings.Join(quoted, " OR ")} {
		found, err := s.store.search(ctx, corpus, match, q.Topic, q.Feed, candidatePool)
		if err != nil {
			return nil, 0, err
		}
		for _, c := range found {
			if !seen[c.ID] && !pending[c.Path] {
				seen[c.ID] = true
				pool = append(pool, c)
			}
		}
		if leg == 0 {
			exact = len(pool)
		}
		if len(pool) >= candidatePool/2 || len(terms) == 1 {
			break
		}
	}
	for i := range pool {
		pool[i].Rank = -pool[i].Rank // bm25: lower is better
	}
	normalise(pool)
	for i := exact; i < len(pool); i++ {
		pool[i].Rank *= partialMatchWeight
	}
	return pool, exact, nil
}

// normalise scales ranks so the best is 1.
func normalise(pool []candidate) {
	best := 0.0
	for _, c := range pool {
		best = math.Max(best, c.Rank)
	}
	for i := range pool {
		if best > 0 {
			pool[i].Rank /= best
		}
	}
}

// authority is a small prior on what a file is for: a topic's entry files
// (its mission, a skill, a map, a README) answer "what is X" better than the
// evidence about X. Raw material — run output, support copies, preimages,
// history, replayed copies, compiled files — ranks lower unless the caller asks for it, so a
// question finds the finding before the evidence and copies behind it.
func authority(rel string, includeRaw bool) float64 {
	w := 1.0
	if name := strings.ToUpper(path.Base(rel)); name == "SKILL.MD" || name == "MISSION.MD" || name == "README.MD" || strings.HasSuffix(name, "MAP.MD") {
		w = 1.25
	}
	if !includeRaw && isRaw(rel) {
		w *= rawWeight
	}
	return w
}

// isRaw reports raw material: anything under a runs/, support/, preimages/,
// history/ or replay/ (a review's copy of the home) folder, and compiled Python.
func isRaw(rel string) bool {
	if strings.HasSuffix(rel, ".pyc") {
		return true
	}
	for _, dir := range strings.Split(path.Dir(rel), "/") {
		switch dir {
		case "runs", "support", "preimages", "history", "replay":
			return true
		}
	}
	return false
}

// FeedSummary lists one feed of one topic.
type FeedSummary struct {
	Topic, Feed      string
	Files            int
	Bytes            int64
	LastModified     time.Time
	Recent           []RecentItem
	Chunks, Embedded int // semantic coverage
}

// RecentItem is a recently changed file in a feed.
type RecentItem struct {
	Path, Title string
	ModifiedAt  time.Time
}

// ListFeeds syncs and summarises each feed: size, freshness and the newest
// files, so an agent sees what a topic holds without reading it.
func (s *Service) ListFeeds(ctx context.Context, corpus, topic string, recent int) ([]FeedSummary, SyncReport, error) {
	corpus, _, err := s.root(corpus)
	if err != nil {
		return nil, SyncReport{}, err
	}
	if recent <= 0 {
		recent = 3
	}
	report, _, err := s.Sync(ctx, corpus, topic, readSyncBudget)
	if err != nil {
		return nil, report, err
	}
	rows, err := s.store.feeds(ctx, corpus, topic)
	if err != nil {
		return nil, report, err
	}
	coverage, err := s.store.coverage(ctx, corpus, topic)
	if err != nil {
		return nil, report, err
	}
	out := make([]FeedSummary, 0, len(rows))
	for _, r := range rows {
		c := coverage[[2]string{r.Topic, r.Feed}]
		f := FeedSummary{Topic: r.Topic, Feed: r.Feed, Files: r.Files, Bytes: r.Bytes, LastModified: time.Unix(0, r.LastNS).UTC(), Chunks: c[0], Embedded: c[1]}
		items, err := s.store.recent(ctx, corpus, r.Topic, r.Feed, recent)
		if err != nil {
			return nil, report, err
		}
		for _, it := range items {
			f.Recent = append(f.Recent, RecentItem{Path: it.Path, Title: it.Title, ModifiedAt: time.Unix(0, it.MtimeNS).UTC()})
		}
		out = append(out, f)
	}
	return out, report, nil
}

// Run keeps the index warm so reads rarely wait: a full sync at start and
// every interval, plus background embedding at a gentle rate when an
// embedder is configured. It returns when ctx ends.
func (s *Service) Run(ctx context.Context, interval time.Duration) {
	for {
		more := false
		for corpus := range s.roots {
			s.syncCorpus(ctx, corpus)
			if s.embedder != nil && s.embedBackground(ctx, corpus) {
				more = true
			}
		}
		wait := interval
		if more {
			wait = 0 // keep embedding, re-syncing between batches
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

// syncCorpus syncs a whole corpus in bounded slices so a long first build
// never makes a read wait behind it.
func (s *Service) syncCorpus(ctx context.Context, corpus string) {
	for ctx.Err() == nil {
		report, _, err := s.Sync(ctx, corpus, "", readSyncBudget)
		if err != nil {
			s.logger.Printf("files: sync %s: %v", corpus, err)
			return
		}
		if report.Changed > 0 || report.Removed > 0 {
			s.logger.Printf("files: synced %s: %d seen, %d changed, %d removed, %d pending in %s", corpus, report.Seen, report.Changed, report.Removed, report.Pending, report.Duration.Round(time.Millisecond))
		}
		if !report.Partial {
			return
		}
	}
}

// queryTerms tokenises like FTS5's unicode61 tokenizer and drops one-letter
// noise; quoting each term keeps FTS5 operators in user text inert.
func queryTerms(text string) []string {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	seen := map[string]bool{}
	var out []string
	for _, f := range fields {
		if len(f) > 1 && !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	return out
}

func chunkHash(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:16])
}

func encodeVector(v []float64) []byte {
	b := make([]byte, 4*len(v))
	for i, x := range v {
		binary.LittleEndian.PutUint32(b[4*i:], math.Float32bits(float32(x)))
	}
	return b
}

func decodeVector(b []byte) []float32 {
	v := make([]float32, len(b)/4)
	for i := range v {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[4*i:]))
	}
	return v
}

func toFloat32(v []float64) []float32 {
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = float32(x)
	}
	return out
}

func cosine(a []float64, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, aa, bb float64
	for i := range a {
		dot += a[i] * float64(b[i])
		aa += a[i] * a[i]
		bb += float64(b[i]) * float64(b[i])
	}
	if aa == 0 || bb == 0 {
		return 0
	}
	return dot / (math.Sqrt(aa) * math.Sqrt(bb))
}

package files

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// The semantic leg finds chunks by meaning, not shared words: the query
// vector is compared with every embedded chunk, and the nearest join the
// lexical candidates before one fused ranking. It is optional — without an
// embedder, or while ai-gateway is down, search stays lexical and says so.

const (
	semanticPool = 150 // nearest chunks that join the lexical candidates
	semanticScan = 200 // nearest hashes read before filtering to live chunks
	// Weights measured on paraphrase and identifier queries over the real
	// research home (owners/C/ROUNDS.md): a chunk holding every query term is
	// strong evidence; one sharing only some words is nearly none, so a
	// paraphrase ranks by meaning while an exact identifier still wins. When
	// no chunk holds every term the query is a paraphrase, and only meaning
	// ranks (paraphraseLexicalWeight).
	lexicalWeight           = 0.7
	paraphraseLexicalWeight = 0
	partialMatchWeight      = 0.1
	neutralSemantic         = 0.5 // stands in for a lexical candidate not yet embedded
	queryEmbedBudget        = 3 * time.Second
	// Background embedding: parallel batches through ai-gateway's typed
	// embedding call, each waiting for foreground inference to be quiet.
	embedWorkers = 4
	embedChunk   = 16
	embedBatch   = embedWorkers * embedChunk * 8 // chunks per pass between syncs
	priorityCap  = 4096
)

// Embedder is the optional semantic leg (Source Ledger's ai-gateway seam).
type Embedder interface {
	// EmbedQueryFast embeds a query in the foreground.
	EmbedQueryFast(context.Context, string) ([]float64, error)
	// EmbedDocumentsWhenIdle embeds a batch in the background, stepping aside
	// for foreground inference first.
	EmbedDocumentsWhenIdle(context.Context, []string) ([][]float64, error)
}

// vectorIndex holds every chunk vector in memory, normalised and quantised
// to int8 (a quarter of float32; ranking error far below the gap between a
// match and a miss), so a query scans the whole corpus in milliseconds.
type vectorIndex struct {
	mu     sync.RWMutex
	loaded bool
	dim    int
	hashes []string
	pos    map[string]int
	data   []int8
}

func (x *vectorIndex) ensure(ctx context.Context, st *store) error {
	x.mu.RLock()
	loaded := x.loaded
	x.mu.RUnlock()
	if loaded {
		return nil
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.loaded {
		return nil
	}
	x.pos = map[string]int{}
	err := st.eachVector(ctx, func(hash string, v []float32) {
		x.addLocked(hash, v)
	})
	if err != nil {
		return err
	}
	x.loaded = true
	return nil
}

func (x *vectorIndex) add(hash string, v []float64) {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.loaded {
		x.addLocked(hash, toFloat32(v))
	}
}

func (x *vectorIndex) addLocked(hash string, v []float32) {
	if len(v) == 0 || (x.dim != 0 && len(v) != x.dim) {
		return
	}
	if _, ok := x.pos[hash]; ok {
		return
	}
	x.dim = len(v)
	x.pos[hash] = len(x.hashes)
	x.hashes = append(x.hashes, hash)
	x.data = append(x.data, quantise(v)...)
}

func (x *vectorIndex) size() int {
	x.mu.RLock()
	defer x.mu.RUnlock()
	return len(x.hashes)
}

func quantise(v []float32) []int8 {
	var norm float64
	for _, f := range v {
		norm += float64(f) * float64(f)
	}
	out := make([]int8, len(v))
	if norm == 0 {
		return out
	}
	scale := 127 / math.Sqrt(norm)
	for i, f := range v {
		out[i] = int8(math.Round(float64(f) * scale))
	}
	return out
}

type near struct {
	hash string
	cos  float64
}

// nearest returns the k most similar hashes to q (any length-matching
// vector), restricted to allow when it is non-nil.
func (x *vectorIndex) nearest(q []float64, allow map[string]bool, k int) []near {
	x.mu.RLock()
	defer x.mu.RUnlock()
	if len(q) != x.dim || x.dim == 0 {
		return nil
	}
	var norm float64
	for _, f := range q {
		norm += f * f
	}
	if norm == 0 {
		return nil
	}
	qs := make([]float32, len(q))
	for i, f := range q {
		qs[i] = float32(f / math.Sqrt(norm) / 127)
	}
	out := make([]near, 0, k+1)
	for i, h := range x.hashes {
		if allow != nil && !allow[h] {
			continue
		}
		row := x.data[i*x.dim : (i+1)*x.dim]
		var dot float32
		for j, b := range row {
			dot += float32(b) * qs[j]
		}
		if len(out) < k || float64(dot) > out[len(out)-1].cos {
			out = append(out, near{hash: h, cos: float64(dot)})
			sort.Slice(out, func(a, b int) bool { return out[a].cos > out[b].cos })
			if len(out) > k {
				out = out[:k]
			}
		}
	}
	return out
}

// semantic adds the nearest chunks to pool and fuses lexical and semantic
// evidence into one rank. It returns a degradation when the leg is missing
// or partial; the pool stays valid either way.
func (s *Service) semantic(ctx context.Context, corpus string, q Query, pool []candidate, exactMatch bool, pending map[string]bool) ([]candidate, *Degradation) {
	if s.embedder == nil {
		return pool, &Degradation{Reason: "SEMANTIC_UNAVAILABLE", Leg: "semantic", Detail: "lexical ranking: no embedder configured"}
	}
	if err := s.vectors.ensure(ctx, s.store); err != nil {
		return pool, &Degradation{Reason: "SEMANTIC_UNAVAILABLE", Leg: "semantic", Detail: "lexical ranking: " + err.Error()}
	}
	embedded := s.vectors.size()
	if embedded == 0 {
		s.prioritise(pool)
		return pool, &Degradation{Reason: "SEMANTIC_WARMING", Leg: "semantic", Detail: "lexical ranking: no chunk vectors yet (background embedding continues)"}
	}
	qctx, cancel := context.WithTimeout(ctx, queryEmbedBudget)
	qv, err := s.embedder.EmbedQueryFast(qctx, q.Text)
	cancel()
	if err != nil {
		return pool, &Degradation{Reason: "SEMANTIC_UNAVAILABLE", Leg: "semantic", Detail: "lexical ranking: " + err.Error()}
	}
	var allow map[string]bool
	if q.Topic != "" || q.Feed != "" {
		if allow, err = s.store.hashesWhere(ctx, corpus, q.Topic, q.Feed); err != nil {
			return pool, &Degradation{Reason: "SEMANTIC_UNAVAILABLE", Leg: "semantic", Detail: "lexical ranking: " + err.Error()}
		}
	}
	nearest := s.vectors.nearest(qv, allow, semanticScan)
	cos := map[string]float64{}
	hashes := make([]string, 0, len(nearest))
	for _, n := range nearest {
		cos[n.hash] = n.cos
		hashes = append(hashes, n.hash)
	}
	extra, err := s.store.byHashes(ctx, corpus, hashes, q.Topic, q.Feed)
	if err != nil {
		return pool, &Degradation{Reason: "SEMANTIC_UNAVAILABLE", Leg: "semantic", Detail: "lexical ranking: " + err.Error()}
	}
	seen := map[int64]bool{}
	for _, c := range pool {
		seen[c.ID] = true
	}
	sort.SliceStable(extra, func(i, j int) bool { return cos[extra[i].Hash] > cos[extra[j].Hash] })
	added := 0
	for _, c := range extra {
		if added == semanticPool {
			break
		}
		if !seen[c.ID] && !pending[c.Path] {
			seen[c.ID] = true
			c.Rank = 0 // no lexical evidence
			pool = append(pool, c)
			added++
		}
	}
	// Lexical candidates outside the scanned neighbourhood get their exact
	// cosine when they have a vector; the rest count as neutral and are
	// embedded first by the background worker.
	var missing []string
	for _, c := range pool {
		if _, ok := cos[c.Hash]; !ok && isProse(c.Path) {
			missing = append(missing, c.Hash)
		}
	}
	if vecs, err := s.store.vectors(ctx, missing); err == nil {
		for h, v := range vecs {
			cos[h] = cosine(qv, v)
		}
	}
	hi, lo := 0.0, 1.0
	if len(nearest) > 0 {
		hi, lo = nearest[0].cos, nearest[len(nearest)-1].cos
	}
	lw := lexicalWeight
	if !exactMatch {
		lw = paraphraseLexicalWeight
	}
	var unembedded []candidate
	for i, c := range pool {
		sem := neutralSemantic
		if !exactMatch {
			sem = c.Rank // a paraphrase: without a vector, only its partial lexical evidence counts
		}
		if v, ok := cos[c.Hash]; ok {
			sem = 0
			if hi > lo {
				sem = math.Max(0, math.Min(1, (v-lo)/(hi-lo)))
			}
		} else if isProse(c.Path) {
			unembedded = append(unembedded, c)
		}
		pool[i].Rank = lw*c.Rank + (1-lw)*sem
	}
	s.prioritise(unembedded)
	total, err := s.store.chunkCount(ctx, corpus)
	if err == nil && embedded < total {
		return pool, &Degradation{Reason: "SEMANTIC_PARTIAL", Leg: "semantic", Detail: fmt.Sprintf("hybrid ranking over %d of %d prose chunks with vectors (%.0f%%); entry files and queried chunks are embedded first", embedded, total, 100*float64(embedded)/float64(total))}
	}
	return pool, nil
}

// embedEach retries a failed batch one text at a time. A text the model
// refuses twice while the model still answers (too long for its context,
// say) gets an empty vector, so it never stalls the queue again. When every
// text fails and a one-word probe fails too, the embedder is down: the error
// stands and nothing is marked.
func (s *Service) embedEach(ctx context.Context, texts []string) ([][]float64, error) {
	out := make([][]float64, len(texts))
	var lastErr error
	ok := 0
	for i, t := range texts {
		v, err := s.embedOne(ctx, t)
		if err != nil {
			lastErr = err
			continue
		}
		out[i] = v
		ok++
	}
	if ok == 0 && !s.embedderUp(ctx) {
		return nil, lastErr
	}
	if refused := len(texts) - ok; refused > 0 {
		s.logger.Printf("files: %d chunk(s) the embedding model refuses are skipped (last error: %v)", refused, lastErr)
	}
	return out, nil
}

// embedOne embeds one text, trying twice.
func (s *Service) embedOne(ctx context.Context, text string) (vec []float64, err error) {
	for range 2 {
		var v [][]float64
		if v, err = s.embedder.EmbedDocumentsWhenIdle(ctx, []string{text}); err == nil && len(v) == 1 && len(v[0]) > 0 {
			return v[0], nil
		}
	}
	if err == nil {
		err = fmt.Errorf("embedder returned no vectors")
	}
	return nil, err
}

// embedderUp probes the embedder with one word.
func (s *Service) embedderUp(ctx context.Context) bool {
	_, err := s.embedOne(ctx, "probe")
	return err == nil
}

// isProse reports whether path is in the semantic leg (see the store's
// prose filter); data files count as semantically neutral.
func isProse(path string) bool {
	k := kindByExt[strings.ToLower(filepath.Ext(path))]
	return k == KindMarkdown || k == KindText
}

// prioritise queues chunks a query needed but could not score semantically.
func (s *Service) prioritise(pool []candidate) {
	s.prioMu.Lock()
	defer s.prioMu.Unlock()
	for _, c := range pool {
		if len(s.priority) >= priorityCap {
			return
		}
		if !s.queued[c.Hash] {
			s.queued[c.Hash] = true
			s.priority = append(s.priority, c.Hash)
		}
	}
}

func (s *Service) takePriority(n int) []string {
	s.prioMu.Lock()
	defer s.prioMu.Unlock()
	n = min(n, len(s.priority))
	out := append([]string(nil), s.priority[:n]...)
	s.priority = s.priority[n:]
	for _, h := range out {
		delete(s.queued, h)
	}
	return out
}

// embedWork lists up to embedBatch (hash, text) pairs: queried chunks first,
// then the unembedded queue (entry files first).
func (s *Service) embedWork(ctx context.Context, corpus string) ([][2]string, error) {
	todo, err := s.store.textsByHash(ctx, corpus, s.takePriority(embedBatch))
	if err != nil || len(todo) >= embedBatch {
		return todo, err
	}
	rest, err := s.store.unembedded(ctx, corpus, embedBatch-len(todo))
	return append(todo, rest...), err
}

// embedAndStore embeds one batch (one text at a time if the batch fails) and
// stores the vectors; an empty vector marks a refused chunk.
func (s *Service) embedAndStore(ctx context.Context, batch [][2]string) error {
	texts := make([]string, len(batch))
	for i, item := range batch {
		texts[i] = item[1]
	}
	vecs, err := s.embedder.EmbedDocumentsWhenIdle(ctx, texts)
	if err != nil {
		if vecs, err = s.embedEach(ctx, texts); err != nil {
			return err
		}
	}
	for i, item := range batch {
		if s.store.putVector(ctx, item[0], vecs[i]) == nil && len(vecs[i]) > 0 {
			s.vectors.add(item[0], vecs[i])
		}
	}
	return nil
}

// embedBackground embeds one pass — queried chunks, then entry files, then
// the rest — in parallel batches, and reports whether more work remains.
func (s *Service) embedBackground(ctx context.Context, corpus string) bool {
	todo, err := s.embedWork(ctx, corpus)
	if err != nil {
		return false
	}
	work := make(chan [][2]string)
	var wg sync.WaitGroup
	var failed sync.Once
	stop := make(chan struct{})
	for range embedWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for batch := range work {
				if err := s.embedAndStore(ctx, batch); err != nil {
					failed.Do(func() {
						s.logger.Printf("files: background embedding paused: %v", err)
						close(stop)
					})
				}
			}
		}()
	}
	ok := true
feed:
	for i := 0; i < len(todo); i += embedChunk {
		select {
		case <-ctx.Done():
			ok = false
			break feed
		case <-stop:
			ok = false // the semantic leg is optional; retry next interval
			break feed
		case work <- todo[i:min(i+embedChunk, len(todo))]:
		}
	}
	close(work)
	wg.Wait()
	return ok && len(todo) == embedBatch
}

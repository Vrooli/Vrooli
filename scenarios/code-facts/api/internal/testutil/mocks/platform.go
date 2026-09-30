package mocks

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"code-facts/internal/analysis"
	"code-facts/internal/cache"
	"code-facts/internal/catalog"
	"code-facts/internal/indexcontrol"
	"code-facts/internal/retrieval"
	"code-facts/internal/targets"
)

type Clock struct{ Time time.Time }

func (c *Clock) Now() time.Time { return c.Time }

var (
	_ cache.Clock        = (*Clock)(nil)
	_ catalog.Clock      = (*Clock)(nil)
	_ indexcontrol.Clock = (*Clock)(nil)
)

type FileSystem struct {
	Files map[string][]byte
	Info  map[string]targets.FileInfo
}

func (f *FileSystem) Stat(_ context.Context, path string) (targets.FileInfo, error) {
	info, ok := f.Info[path]
	if !ok {
		return targets.FileInfo{}, errors.New("not found")
	}
	return info, nil
}

func (f *FileSystem) ReadFile(_ context.Context, path string) ([]byte, error) {
	payload, ok := f.Files[path]
	if !ok {
		return nil, errors.New("not found")
	}
	return append([]byte(nil), payload...), nil
}

func (f *FileSystem) Walk(ctx context.Context, roots []string, yield func(targets.FileInfo) error) error {
	paths := make([]string, 0, len(f.Info))
	for path := range f.Info {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := yield(f.Info[path]); err != nil {
			return err
		}
	}
	return nil
}

var _ targets.FileSystem = (*FileSystem)(nil)

type Analyzer struct {
	Result analysis.Result
	Err    error
	Calls  []analysis.Request
	mu     sync.Mutex
}

func (a *Analyzer) Analyze(_ context.Context, request analysis.Request) (analysis.Result, error) {
	a.mu.Lock()
	a.Calls = append(a.Calls, request)
	a.mu.Unlock()
	return a.Result, a.Err
}

var _ analysis.Analyzer = (*Analyzer)(nil)

type Embedder struct {
	Vectors [][]float32
	Err     error
}

func (e *Embedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	if e.Err != nil {
		return nil, e.Err
	}
	if len(e.Vectors) != len(texts) {
		return nil, errors.New("fake vector count does not match text count")
	}
	out := make([][]float32, len(e.Vectors))
	for i := range e.Vectors {
		out[i] = append([]float32(nil), e.Vectors[i]...)
	}
	return out, nil
}

var _ retrieval.Embedder = (*Embedder)(nil)

type VectorStore struct {
	Results []retrieval.Candidate
	Records map[string]retrieval.VectorRecord
	Err     error
}

func (v *VectorStore) Upsert(_ context.Context, records []retrieval.VectorRecord) error {
	if v.Err != nil {
		return v.Err
	}
	if v.Records == nil {
		v.Records = map[string]retrieval.VectorRecord{}
	}
	for _, record := range records {
		v.Records[record.ID] = record
	}
	return nil
}

func (v *VectorStore) Delete(_ context.Context, ids []string) error {
	for _, id := range ids {
		delete(v.Records, id)
	}
	return v.Err
}

func (v *VectorStore) Query(_ context.Context, _ []float32, _ retrieval.Query) ([]retrieval.Candidate, error) {
	return append([]retrieval.Candidate(nil), v.Results...), v.Err
}

var _ retrieval.VectorStore = (*VectorStore)(nil)

type Reranker struct {
	Results []retrieval.Candidate
	Err     error
}

func (r *Reranker) Rerank(_ context.Context, _ retrieval.Query, candidates []retrieval.Candidate) ([]retrieval.Candidate, error) {
	if r.Results == nil {
		return append([]retrieval.Candidate(nil), candidates...), r.Err
	}
	return append([]retrieval.Candidate(nil), r.Results...), r.Err
}

var _ retrieval.Reranker = (*Reranker)(nil)

type ProcessRunner struct {
	Output []byte
	Err    error
	Calls  [][]string
}

func (p *ProcessRunner) Run(_ context.Context, command string, args ...string) ([]byte, error) {
	p.Calls = append(p.Calls, append([]string{command}, args...))
	return append([]byte(nil), p.Output...), p.Err
}

var _ indexcontrol.ProcessRunner = (*ProcessRunner)(nil)

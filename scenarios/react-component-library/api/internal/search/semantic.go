package search

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	aisearch "github.com/vrooli/ai-go/search"
)

// semanticIndex is the ai-go adapter over the committed lexical projection.
// SQLite remains the source of truth; Qdrant is a separately reconciled
// retrieval projection and can therefore degrade without interrupting FTS.
type semanticIndex struct {
	engine         *aisearch.Service
	store          aisearch.VectorStore
	spec           aisearch.CollectionSpec
	reconciler     *aisearch.Reconciler
	reconcileGate  chan struct{}
	reconcileFunc  func(context.Context) error
	mode           aisearch.SearchMode
	lastSearchErr  atomic.Pointer[string]
	lastSyncErr    atomic.Pointer[string]
	readinessProbe func(context.Context) bool
	readinessMu    sync.Mutex
	readinessAt    time.Time
	readinessValue bool
	readinessWait  chan struct{}
}

// Backend reachability and point-count checks are network calls. Keep their
// result for at most this interval; status may be briefly stale, but never
// serves a snapshot older than the documented bound. Search/reconcile errors
// invalidate it immediately.
const (
	semanticReadinessTTL          = 30 * time.Second
	semanticReadinessProbeTimeout = 750 * time.Millisecond
)

type unavailableSemantic struct{ cause error }

// semanticQueryer is the smallest database surface needed to read the
// committed projection. Passing the request context through QueryContext lets
// RoutedDB select the same pool as the request being reconciled.
type semanticQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func UnavailableSemantic(cause error) Semantic {
	if cause == nil {
		cause = fmt.Errorf("semantic search is unavailable")
	}
	return unavailableSemantic{cause: cause}
}

func (s unavailableSemantic) Enabled() bool                   { return true }
func (s unavailableSemantic) Ready(context.Context) bool      { return false }
func (s unavailableSemantic) Reconcile(context.Context) error { return s.cause }
func (s unavailableSemantic) SearchSemantic(context.Context, aisearch.SearchQuery) ([]aisearch.SearchResult, error) {
	return nil, s.cause
}

// NewAISemantic assembles the shared ai-go engine and a projection-backed
// reconciler. A construction error returns an explicit degraded leg, allowing
// callers to retain lexical search while exposing the unavailable AI path.
func NewAISemantic(ctx context.Context, db semanticQueryer, cfg aisearch.Config, tuning aisearch.TuningConfig, collection string) (Semantic, error) {
	deps := aisearch.EngineDeps{
		QdrantURL: cfg.QdrantURL, QdrantAPIKey: cfg.QdrantAPIKey, Collection: collection,
		EmbedRole: cfg.EmbedRole, RerankerURL: cfg.RerankerURL, RerankerModel: cfg.RerankerModel, RerankRole: cfg.RerankRole,
	}
	resolved, err := aisearch.ResolveEngineDepsEmbedding(ctx, deps)
	if err != nil {
		return unavailableSemantic{cause: err}, err
	}
	engine := aisearch.NewServiceForTuning(tuning, resolved)
	options := engine.ServiceOptions()
	source := projectionSource{db: db}
	mode := aisearch.ModeDense
	var binding aisearch.SourceBinding
	if engine.SparseEncoder != nil {
		mode = aisearch.ModeHybrid
		binding = aisearch.NewHybridBinding("workbench", "rcl:", options.VectorStore, source,
			aisearch.NewIdentityChunker(), aisearch.NewIdentityComposer(), engine.SparseEncoder)
	} else {
		binding = aisearch.NewDenseBinding("workbench", "rcl:", options.VectorStore, source)
	}
	reconciler := aisearch.NewReconciler(options.Embedder, []aisearch.SourceBinding{binding}, cfg.ReconcileParallelism)
	options.Reconciler = reconciler
	return &semanticIndex{engine: aisearch.NewService(options), store: options.VectorStore, spec: engine.Spec, reconciler: reconciler, reconcileGate: make(chan struct{}, 1), mode: mode}, nil
}

func (s *semanticIndex) Enabled() bool { return s != nil && s.engine != nil }

func (s *semanticIndex) Ready(ctx context.Context) bool {
	if s == nil || s.lastSearchErr.Load() != nil || s.lastSyncErr.Load() != nil || (s.readinessProbe == nil && (s.engine == nil || s.store == nil)) {
		return false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	probeCtx, cancel := context.WithTimeout(ctx, semanticReadinessProbeTimeout)
	defer cancel()
	for {
		s.readinessMu.Lock()
		if s.lastSearchErr.Load() != nil || s.lastSyncErr.Load() != nil {
			s.readinessMu.Unlock()
			return false
		}
		if !s.readinessAt.IsZero() && time.Since(s.readinessAt) < semanticReadinessTTL {
			ready := s.readinessValue
			s.readinessMu.Unlock()
			return ready
		}
		if wait := s.readinessWait; wait != nil {
			s.readinessMu.Unlock()
			select {
			case <-wait:
				continue
			case <-probeCtx.Done():
				return false
			}
		}
		wait := make(chan struct{})
		s.readinessWait = wait
		s.readinessMu.Unlock()

		ready := false
		if s.readinessProbe != nil {
			ready = s.readinessProbe(probeCtx)
		} else {
			status := s.engine.Status(probeCtx)
			ready = status.Qdrant && status.Ollama && status.IndexedCount > 0
		}

		s.readinessMu.Lock()
		if s.lastSearchErr.Load() != nil || s.lastSyncErr.Load() != nil {
			ready = false
		}
		s.readinessValue = ready
		s.readinessAt = time.Now()
		s.readinessWait = nil
		close(wait)
		s.readinessMu.Unlock()
		return ready
	}
}

func (s *semanticIndex) Reconcile(ctx context.Context) error {
	if s == nil || (s.reconciler == nil && s.reconcileFunc == nil) {
		return fmt.Errorf("semantic index is not configured")
	}
	if s.reconcileGate == nil {
		return fmt.Errorf("semantic index reconciliation gate is not configured")
	}
	select {
	case s.reconcileGate <- struct{}{}:
		defer func() { <-s.reconcileGate }()
	case <-ctx.Done():
		// Cancellation while queued has not affected the active run or the
		// vector projection, so it must not poison readiness.
		return ctx.Err()
	}
	if s.reconcileFunc != nil {
		if err := s.reconcileFunc(ctx); err != nil {
			s.setSyncError(err)
			return err
		}
		s.lastSyncErr.Store(nil)
		s.invalidateReadiness()
		return nil
	}
	if err := aisearch.EnsureCollectionForBinding(ctx, s.store, s.reconciler.Bindings[0], s.spec); err != nil {
		s.setSyncError(err)
		return err
	}
	_, _, err := s.reconciler.RunOnce(ctx)
	if err != nil {
		s.setSyncError(err)
		return err
	}
	s.lastSyncErr.Store(nil)
	s.invalidateReadiness()
	return nil
}

func (s *semanticIndex) SearchSemantic(ctx context.Context, query aisearch.SearchQuery) ([]aisearch.SearchResult, error) {
	if s == nil || s.engine == nil {
		return nil, fmt.Errorf("semantic search is not configured")
	}
	query.Mode = s.mode
	query.Limit = maxSearchLimit
	response, err := s.engine.Search(ctx, query)
	if err != nil {
		s.setSearchError(err)
		return nil, err
	}
	s.lastSearchErr.Store(nil)
	s.invalidateReadiness()
	out := make([]aisearch.SearchResult, 0, len(response.Results))
	for _, result := range response.Results {
		id, _ := result.Payload["source_id"].(string)
		if strings.TrimSpace(id) == "" {
			continue
		}
		result.ID = id
		out = append(out, result)
	}
	return out, nil
}

func (s *semanticIndex) setSearchError(err error) {
	if err == nil {
		s.lastSearchErr.Store(nil)
		s.invalidateReadiness()
		return
	}
	message := err.Error()
	s.lastSearchErr.Store(&message)
	s.invalidateReadiness()
}

// setError preserves the query-operation helper used by readiness tests. Sync
// failures have a separate lifecycle and must only be cleared by reconciliation.
func (s *semanticIndex) setError(err error) { s.setSearchError(err) }

func (s *semanticIndex) setSyncError(err error) {
	if err == nil {
		s.lastSyncErr.Store(nil)
		s.invalidateReadiness()
		return
	}
	message := err.Error()
	s.lastSyncErr.Store(&message)
	s.invalidateReadiness()
}

func (s *semanticIndex) invalidateReadiness() {
	if s == nil {
		return
	}
	s.readinessMu.Lock()
	s.readinessAt = time.Time{}
	s.readinessValue = false
	s.readinessMu.Unlock()
}

// projectionSource materializes only the bounded, already-committed workbench
// index. ContentHash excludes volatile timestamps so routine reindexes do not
// trigger needless embeddings.
type projectionSource struct{ db semanticQueryer }

func (s projectionSource) LoadAll(ctx context.Context) ([]aisearch.SourceDoc, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, kind, title, subtitle, body, href, availability, badges_json, relations_json FROM search_docs ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("read semantic source projection: %w", err)
	}
	defer rows.Close()
	var docs []aisearch.SourceDoc
	for rows.Next() {
		var id, kind, title, subtitle, body, href, availability, badges, relations string
		if err := rows.Scan(&id, &kind, &title, &subtitle, &body, &href, &availability, &badges, &relations); err != nil {
			return nil, fmt.Errorf("scan semantic source: %w", err)
		}
		canonical := strings.Join([]string{kind, title, subtitle, body, href, availability, badges, relations}, "\x00")
		sum := sha256.Sum256([]byte(canonical))
		docs = append(docs, aisearch.SourceDoc{ID: id, Kind: "workbench", ContentHash: "sha256:" + hex.EncodeToString(sum[:]), Body: strings.Join([]string{title, subtitle, body, badges, relations}, "\n"), Meta: map[string]any{"kind": kind, "title": title, "subtitle": subtitle, "href": href, "availability": availability}})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate semantic source: %w", err)
	}
	return docs, nil
}

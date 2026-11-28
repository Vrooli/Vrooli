package search

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Reindex rebuilds the whole projection from the owned sources in one
// transaction. A failed rebuild rolls back and records last_error, leaving the
// previous index queryable. force=false is a no-op for a current, error-free
// index.
func (s *Service) Reindex(ctx context.Context, force bool) (int, time.Time, error) {
	if s == nil || s.db == nil {
		return 0, time.Time{}, ErrNotConfigured
	}
	s.reindexMu.Lock()
	defer s.reindexMu.Unlock()

	if !force {
		status, err := s.Status(ctx)
		if err != nil {
			return 0, time.Time{}, err
		}
		if status.GetIndexed() && !status.GetStale() {
			indexedAt, _ := time.Parse(time.RFC3339Nano, status.GetIndexedAt())
			s.reconcileSemanticAsync()
			return int(status.GetDocumentCount()), indexedAt, nil
		}
	}

	now := time.Now().UTC()
	// Catalog + availability are resolved before the transaction. They read
	// files and an injected resolver, and holding SQLite's single writer while
	// doing that would serialize the whole workbench behind a reindex.
	catalogDocs, warnings, err := s.catalogDocuments(ctx, now)
	if err != nil {
		return s.failReindex(ctx, err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return s.failReindex(ctx, fmt.Errorf("begin search reindex: %w", err))
	}
	// The rollback is explicit on every failure path before failReindex writes
	// last_error: SQLite allows one connection here, so recording the failure
	// while the transaction still holds that connection would deadlock.
	abort := func(cause error) (int, time.Time, error) {
		_ = tx.Rollback()
		return s.failReindex(ctx, cause)
	}
	defer func() { _ = tx.Rollback() }()

	docs := catalogDocs
	workspaceDocs, err := s.workspaceDocuments(ctx, tx, now)
	if err != nil {
		return abort(err)
	}
	docs = append(docs, workspaceDocs...)
	docs = append(docs, commandDocuments(now)...)
	findingDocs, err := s.findingDocuments(ctx, tx, now)
	if err != nil {
		return abort(err)
	}
	docs = append(docs, findingDocs...)

	normalized, err := normalizeDocuments(docs, now)
	if err != nil {
		return abort(err)
	}
	if err := writeDocuments(ctx, tx, normalized, now, strings.Join(warnings, "; ")); err != nil {
		return abort(err)
	}
	if err := tx.Commit(); err != nil {
		return s.failReindex(ctx, fmt.Errorf("commit search reindex: %w", err))
	}
	// SQLite committed first and remains authoritative. Vector reconciliation
	// is optional and can require many remote embeddings, so it must not hold
	// API startup or the reindex RPC open. Readiness remains degraded until the
	// background projection catches up.
	s.reconcileSemanticAsync()
	return len(normalized), now, nil
}

func (s *Service) reconcileSemanticAsync() {
	indexer, ok := s.semantic.(interface{ Reconcile(context.Context) error })
	if !ok {
		return
	}
	s.semanticReconcileMu.Lock()
	if s.semanticReconcileRunning {
		s.semanticReconcileDirty = true
		s.semanticReconcileMu.Unlock()
		return
	}
	s.semanticReconcileRunning = true
	s.semanticReconcileMu.Unlock()

	go func() {
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			err := indexer.Reconcile(ctx)
			cancel()
			if err != nil && s.logger != nil {
				s.logger.Printf("[search] semantic projection reconciliation failed; lexical search remains available: %v", err)
			}

			s.semanticReconcileMu.Lock()
			if s.semanticReconcileDirty {
				s.semanticReconcileDirty = false
				s.semanticReconcileMu.Unlock()
				continue
			}
			s.semanticReconcileRunning = false
			s.semanticReconcileMu.Unlock()
			return
		}
	}()
}

// failReindex records the failure after the transaction has rolled back. If the
// state write also fails (for example the schema is absent), the original error
// is still the one reported.
func (s *Service) failReindex(ctx context.Context, cause error) (int, time.Time, error) {
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	_ = setState(writeCtx, s.db, stateLastError, cause.Error())
	return 0, time.Time{}, cause
}

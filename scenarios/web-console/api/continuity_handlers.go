package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	"web-console/internal/continuity"
)

const continuityPublicationInterval = 30 * time.Second

// startContinuityPublisher drains the durable cross-scenario publication
// queue without making Agent Manager a boot dependency. The queue is the
// source of truth: a transient outage leaves records pending for the next
// attempt, while local Web Console lifecycle and search remain available.
func (s *Server) startContinuityPublisher() context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		publish := func() {
			report, err := s.Publish(ctx, 25)
			if err != nil {
				log.Printf("continuity publication: deferred: %v", err)
				return
			}
			if report.Attempted > 0 || report.Failed > 0 {
				log.Printf("continuity publication: attempted=%d published=%d failed=%d next=%d", report.Attempted, report.Published, report.Failed, report.Next)
			}
		}
		publish()
		ticker := time.NewTicker(continuityPublicationInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				publish()
			}
		}
	}()
	return cancel
}

func (s *Server) Integrity(ctx context.Context) (continuity.IntegrityReport, error) {
	report, err := continuity.NewIntegrityAuditor(s.db).Audit(ctx)
	if err == nil && s.metrics != nil {
		s.metrics.ContinuityOrphans.Store(report.OrphanConversations + report.OrphanCheckpoints + report.OrphanWorkspacePanes)
	}
	return report, err
}

func (s *Server) Catalog(ctx context.Context, lifecycleState string, limit int) ([]continuity.CatalogRecord, bool, error) {
	return continuity.NewSQLCatalogStore(s.db).List(ctx, lifecycleState, limit)
}

func (s *Server) Publish(ctx context.Context, limit int) (continuity.PublicationReport, error) {
	base, err := discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
	if err != nil {
		return continuity.PublicationReport{}, fmt.Errorf("resolve agent-manager: %w", err)
	}
	client := &continuity.AgentManagerPublisher{BaseURL: base}
	report, err := continuity.NewSQLCatalogStore(s.db).PublishPending(ctx, client, limit)
	if s.metrics != nil {
		s.metrics.ContinuityPublicationPending.Store(int64(report.Next))
		s.metrics.ContinuityPublicationFailures.Add(int64(report.Failed))
	}
	return report, err
}

func (s *Server) Search(ctx context.Context, query, state, agent, cwd, createdAfter, sessionID, agentSessionID, title, topicSummary string, limit int) ([]continuity.SearchMatch, bool, int64, int64, error) {
	var after time.Time
	var err error
	if strings.TrimSpace(createdAfter) != "" {
		after, err = time.Parse(time.RFC3339, createdAfter)
		if err != nil {
			return nil, false, 0, 0, fmt.Errorf("created_after must be RFC3339: %w", err)
		}
	}
	return continuity.SearchLocalWithOptions(ctx, s.db, continuity.SearchOptions{Query: query, State: state, AgentType: agent, CWD: cwd, CreatedAfter: after, Limit: limit, SessionID: sessionID, AgentSessionID: agentSessionID, Title: title, TopicSummary: topicSummary})
}

func (s *Server) Receipt(ctx context.Context, operationID string) (continuity.Receipt, error) {
	return continuity.NewSQLLedger(s.db).Get(ctx, operationID)
}

func (s *Server) Rollback(ctx context.Context, manifestHash, operationID string) (continuity.Receipt, error) {
	if strings.TrimSpace(manifestHash) == "" || strings.TrimSpace(operationID) == "" {
		return continuity.Receipt{}, fmt.Errorf("manifest_hash and operation_id are required")
	}
	ledger := continuity.NewSQLLedger(s.db)
	receipt, err := ledger.Put(ctx, continuity.Receipt{OperationID: operationID, SessionID: "catalog", ActorKind: "operator", Command: "rollback", FromState: continuity.StateRecoverable, ToState: continuity.StateArchived, ReasonCode: "reconciliation_rollback", Status: "pending", CreatedAt: time.Now().UTC()})
	if err != nil {
		return continuity.Receipt{}, err
	}
	if s.metrics != nil {
		s.metrics.ContinuityReceipts.Add(1)
	}
	if receipt.Status != "pending" {
		return receipt, nil
	}
	err = continuity.NewSQLCatalogStore(s.db).RollbackManifest(ctx, manifestHash)
	if err != nil {
		if s.metrics != nil {
			s.metrics.ContinuityFailures.Add(1)
		}
		completed, completeErr := ledger.Complete(ctx, operationID, "failed", "rollback_failed", time.Now().UTC())
		if completeErr != nil {
			return completed, errors.Join(err, fmt.Errorf("complete rollback receipt: %w", completeErr))
		}
		return completed, err
	}
	return ledger.Complete(ctx, operationID, "succeeded", "", time.Now().UTC())
}

func (s *Server) Reconcile(ctx context.Context, apply bool, generation, operationID, expectedManifest string, offset, batchSize int) ([]continuity.ReconcileItem, int, int, continuity.Receipt, string, int, bool, error) {
	store := continuity.NewSQLCatalogStore(s.db)
	evidence, err := store.Observe(ctx)
	if err != nil {
		return nil, 0, 0, continuity.Receipt{}, "", 0, false, err
	}
	existing, err := store.Existing(ctx)
	if err != nil {
		return nil, len(evidence), 0, continuity.Receipt{}, "", 0, false, err
	}
	if apply && strings.TrimSpace(operationID) == "" {
		return nil, len(evidence), 0, continuity.Receipt{}, "", 0, false, fmt.Errorf("operation_id is required when apply=true")
	}
	if generation != "" {
		report, err := continuity.NewIntegrityAuditor(s.db).Audit(ctx)
		if err != nil {
			return nil, len(evidence), 0, continuity.Receipt{}, "", 0, false, err
		}
		if report.Generation != generation {
			return nil, len(evidence), 0, continuity.Receipt{}, "", 0, false, fmt.Errorf("integrity generation changed: expected %s, observed %s", generation, report.Generation)
		}
	}
	var items []continuity.ReconcileItem
	if expectedManifest != "" {
		stored, loadErr := store.LoadManifest(ctx, expectedManifest)
		if loadErr == nil {
			if generation != "" && stored.Generation != generation {
				return nil, len(evidence), 0, continuity.Receipt{}, expectedManifest, 0, false, fmt.Errorf("manifest generation mismatch")
			}
			items = stored.Items
		} else if loadErr != sql.ErrNoRows {
			return nil, len(evidence), 0, continuity.Receipt{}, "", 0, false, loadErr
		}
	}
	if items == nil {
		items, err = continuity.PlanReconciliation(evidence, existing)
		if err != nil {
			return nil, len(evidence), 0, continuity.Receipt{}, "", 0, false, err
		}
	}
	manifestHash := continuity.ManifestHash(generation, items)
	if expectedManifest != "" && expectedManifest != manifestHash {
		return nil, len(evidence), 0, continuity.Receipt{}, manifestHash, 0, false, fmt.Errorf("reconciliation manifest changed: expected %s, observed %s", expectedManifest, manifestHash)
	}
	// A dry run is read-only with respect to catalog state, but its reviewed
	// manifest is durable audit state. Persisting the exact item set lets a
	// later bounded apply consume the reviewed plan instead of recomputing
	// against a moving projection and accidentally changing the repair scope.
	if !apply {
		previous := make(map[string]continuity.CatalogRecord, len(existing))
		for sessionID, record := range existing {
			previous[sessionID] = record
		}
		if err := store.SaveManifest(ctx, continuity.ReconciliationManifest{
			Hash: manifestHash, Generation: generation, Items: items, Previous: previous,
		}); err != nil {
			return nil, len(evidence), 0, continuity.Receipt{}, manifestHash, 0, false, err
		}
	}
	mutations := 0
	var receipt continuity.Receipt
	if apply {
		previous := make(map[string]continuity.CatalogRecord)
		for _, item := range items {
			if prior, ok := existing[item.Record.SessionID]; ok {
				previous[item.Record.SessionID] = prior
			}
		}
		if stored, loadErr := store.LoadManifest(ctx, manifestHash); loadErr == nil {
			previous = stored.Previous
		}
		if err := store.SaveManifest(ctx, continuity.ReconciliationManifest{Hash: manifestHash, Generation: generation, Items: items, Previous: previous}); err != nil {
			return nil, len(evidence), 0, continuity.Receipt{}, manifestHash, 0, false, err
		}
		ledger := continuity.NewSQLLedger(s.db)
		receipt = continuity.Receipt{OperationID: operationID, SessionID: "catalog", ActorKind: "operator", Command: "reconcile", FromState: continuity.StateArchived, ToState: continuity.StateRecoverable, ReasonCode: "catalog_reconciliation", Status: "pending", CreatedAt: time.Now().UTC()}
		if receipt, err = ledger.Put(ctx, receipt); err != nil {
			if s.metrics != nil {
				s.metrics.ContinuityFailures.Add(1)
			}
			return nil, len(evidence), 0, continuity.Receipt{}, manifestHash, 0, false, err
		}
		if s.metrics != nil {
			s.metrics.ContinuityReceipts.Add(1)
		}
		if receipt.Status == "failed" {
			if receipt, err = ledger.Reopen(ctx, operationID); err != nil {
				return items, len(evidence), 0, receipt, manifestHash, 0, false, err
			}
		}
		if receipt.Status == "succeeded" {
			return items, len(evidence), 0, receipt, manifestHash, len(items), true, nil
		}
		effectiveBatch := batchSize
		if effectiveBatch <= 0 {
			effectiveBatch = 500
		}
		progress, progressErr := store.LoadReconciliationProgress(ctx, operationID)
		if progressErr == nil {
			if progress.ManifestHash != manifestHash || progress.TotalItems != len(items) {
				return items, len(evidence), 0, receipt, manifestHash, 0, false, fmt.Errorf("reconciliation progress does not match manifest")
			}
			if offset == 0 {
				offset = progress.NextOffset
			} else if offset != progress.NextOffset {
				return items, len(evidence), 0, receipt, manifestHash, progress.NextOffset, false, fmt.Errorf("requested offset %d does not match durable offset %d", offset, progress.NextOffset)
			}
		} else if progressErr == sql.ErrNoRows {
			if err := store.SaveReconciliationProgress(ctx, continuity.ReconciliationProgress{OperationID: operationID, ManifestHash: manifestHash, ActorID: operationID, NextOffset: offset, TotalItems: len(items), Status: "pending", UpdatedAt: time.Now().UTC()}); err != nil {
				return items, len(evidence), 0, receipt, manifestHash, offset, false, err
			}
		} else if progressErr != nil {
			return items, len(evidence), 0, receipt, manifestHash, offset, false, progressErr
		}
		end := offset + effectiveBatch
		if end > len(items) {
			end = len(items)
		}
		nextOffset := offset
		for index := offset; index < end; index++ {
			item := items[index]
			itemReceipt, itemErr := store.LoadReconciliationItemReceipt(ctx, operationID, index)
			if itemErr != nil && itemErr != sql.ErrNoRows {
				err = itemErr
				break
			}
			if itemErr == nil && (itemReceipt.Status == "succeeded" || itemReceipt.Status == "quarantined") {
				nextOffset = index + 1
				continue
			}
			if err = store.SaveReconciliationItemReceipt(ctx, continuity.ReconciliationItemReceipt{OperationID: operationID, ItemIndex: index, ManifestHash: manifestHash, SessionID: item.Record.SessionID, Action: item.Action, Status: "pending", CreatedAt: time.Now().UTC()}); err != nil {
				break
			}
			changed, applyErr := continuity.ApplyReconciliationItem(ctx, store, item)
			if applyErr != nil {
				err = applyErr
				_ = store.SaveReconciliationItemReceipt(ctx, continuity.ReconciliationItemReceipt{OperationID: operationID, ItemIndex: index, ManifestHash: manifestHash, SessionID: item.Record.SessionID, Action: item.Action, Status: "failed", ErrorCode: "reconciliation_item_failed", CreatedAt: time.Now().UTC(), CompletedAt: time.Now().UTC()})
				break
			}
			status := "succeeded"
			if item.Action == continuity.ActionQuarantine {
				status = "quarantined"
			}
			if err = store.SaveReconciliationItemReceipt(ctx, continuity.ReconciliationItemReceipt{OperationID: operationID, ItemIndex: index, ManifestHash: manifestHash, SessionID: item.Record.SessionID, Action: item.Action, Status: status, CreatedAt: time.Now().UTC(), CompletedAt: time.Now().UTC()}); err != nil {
				break
			}
			if changed {
				mutations++
			}
			nextOffset = index + 1
			if err = store.SaveReconciliationProgress(ctx, continuity.ReconciliationProgress{OperationID: operationID, ManifestHash: manifestHash, ActorID: operationID, NextOffset: nextOffset, TotalItems: len(items), Status: "pending", UpdatedAt: time.Now().UTC()}); err != nil {
				break
			}
		}
		if err == nil && nextOffset < end {
			err = fmt.Errorf("reconciliation stopped before batch completed")
		}
		if err != nil {
			if s.metrics != nil {
				s.metrics.ContinuityFailures.Add(1)
			}
			completed, completeErr := ledger.Complete(ctx, operationID, "failed", "reconciliation_apply_failed", time.Now().UTC())
			receipt = completed
			if completeErr != nil {
				err = errors.Join(err, fmt.Errorf("complete reconciliation receipt: %w", completeErr))
			}
		} else if nextOffset >= len(items) {
			if err = store.SaveReconciliationProgress(ctx, continuity.ReconciliationProgress{OperationID: operationID, ManifestHash: manifestHash, ActorID: operationID, NextOffset: nextOffset, TotalItems: len(items), Status: "succeeded", UpdatedAt: time.Now().UTC()}); err != nil {
				return items, len(evidence), mutations, receipt, manifestHash, nextOffset, false, err
			}
			receipt, err = ledger.Complete(ctx, operationID, "succeeded", "", time.Now().UTC())
		}
		if s.metrics != nil {
			if pending, pendingErr := store.PendingPublicationCount(ctx); pendingErr == nil {
				s.metrics.ContinuityPublicationPending.Store(int64(pending))
			}
		}
		if !apply {
			nextOffset = 0
		}
		return items, len(evidence), mutations, receipt, manifestHash, nextOffset, nextOffset >= len(items), err
	}
	return items, len(evidence), mutations, receipt, manifestHash, 0, true, err
}

// handleConversationIntegrity is deliberately read-only. It exposes the same
// bounded audit used by reconciliation operators without making an integrity
// count a permission to mutate or delete anything.
func (s *Server) handleConversationIntegrity(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		http.Error(w, `{"error":"database unavailable"}`, http.StatusServiceUnavailable)
		return
	}
	report, err := s.Integrity(r.Context())
	if err != nil {
		http.Error(w, `{"error":"integrity audit failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(report)
}

// handleConversationPublish advances the durable, bounded publication queue.
// It is intentionally explicit and independent from local lifecycle requests:
// Agent Manager outage therefore cannot make a pane close or recovery fail.
func (s *Server) handleConversationPublish(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		http.Error(w, `{"error":"database unavailable"}`, http.StatusServiceUnavailable)
		return
	}
	limit := 25
	if value := r.URL.Query().Get("limit"); value != "" {
		if _, err := fmt.Sscanf(value, "%d", &limit); err != nil {
			http.Error(w, `{"error":"limit must be an integer"}`, http.StatusBadRequest)
			return
		}
	}
	report, err := s.Publish(r.Context(), limit)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(report)
}

// handleConversationCatalog reconciles Web Console metadata into the durable
// catalog projection. GET and POST without apply are dry runs; mutation is
// permitted only for POST with apply=true.
func (s *Server) handleConversationCatalog(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		http.Error(w, `{"error":"database unavailable"}`, http.StatusServiceUnavailable)
		return
	}
	apply := r.Method == http.MethodPost && strings.EqualFold(r.URL.Query().Get("apply"), "true")
	operationID := strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))
	if operationID == "" {
		operationID = strings.TrimSpace(r.URL.Query().Get("operation_id"))
	}
	generation := strings.TrimSpace(r.URL.Query().Get("generation"))
	items, observations, mutations, receipt, manifestHash, nextOffset, complete, err := s.Reconcile(r.Context(), apply, generation, operationID, r.URL.Query().Get("manifest_hash"), 0, 0)
	if err != nil {
		http.Error(w, `{"error":"catalog reconciliation failed"}`, http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	publicItems := make([]continuity.ReconcileItem, 0, len(items))
	for _, item := range items {
		publicItems = append(publicItems, continuity.PublicReconcileItem(item))
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"apply": apply, "observations": observations, "mutations": mutations, "receipt": receipt,
		"manifest_hash": manifestHash, "next_offset": nextOffset, "complete": complete, "items": publicItems,
	})
}

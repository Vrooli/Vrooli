package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/identity"
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

// Integrity is the explicit operator audit, including the event-content
// conservation hash. The hash reads every retained message body, so it is
// cached until the event inventory changes; health probes and the archive
// drawer use Health instead, which never reads bodies.
func (s *Server) Integrity(ctx context.Context) (continuity.IntegrityReport, error) {
	report, err := continuity.NewIntegrityAuditor(s.reads()).WithContentCache(s.contentReceipts).Audit(ctx)
	if err == nil && s.metrics != nil {
		s.metrics.ContinuityOrphans.Store(report.OrphanConversations + report.OrphanCheckpoints + report.OrphanWorkspacePanes)
	}
	return report, err
}

// Health reports the catalog-link drift counts in milliseconds without
// reading message bodies.
func (s *Server) Health(ctx context.Context) (continuity.HealthObservation, error) {
	return s.observeContinuityHealth(ctx)
}

func (s *Server) observeContinuityHealth(ctx context.Context) (continuity.HealthObservation, error) {
	observation, err := continuity.NewIntegrityAuditor(s.reads()).ObserveHealth(ctx)
	if err == nil && s.metrics != nil {
		s.metrics.ContinuityOrphans.Store(observation.OrphanConversations + observation.OrphanCheckpoints + observation.OrphanWorkspacePanes)
	}
	return observation, err
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
	return continuity.SearchLocalWithOptions(ctx, s.reads(), continuity.SearchOptions{Query: query, State: state, AgentType: agent, CWD: cwd, CreatedAfter: after, Limit: limit, SessionID: sessionID, AgentSessionID: agentSessionID, Title: title, TopicSummary: topicSummary})
}

func (s *Server) Receipt(ctx context.Context, operationID string) (continuity.Receipt, error) {
	return continuity.NewSQLLedger(s.db).Get(ctx, operationID)
}

func (s *Server) Rollback(ctx context.Context, manifestHash, operationID string) (continuity.Receipt, error) {
	if strings.TrimSpace(manifestHash) == "" || strings.TrimSpace(operationID) == "" {
		return continuity.Receipt{}, fmt.Errorf("manifest_hash and operation_id are required")
	}
	actor, err := authn.RequireOperation(ctx, "web-console:write", "continuity.rollback", "web-console:continuity/catalog")
	if err != nil {
		return continuity.Receipt{}, err
	}
	s.catalogApplyMu.Lock()
	defer s.catalogApplyMu.Unlock()
	ledger := continuity.NewSQLLedger(s.db)
	receipt, err := ledger.Put(ctx, continuity.Receipt{OperationID: operationID, SessionID: "catalog", ActorKind: string(actor.Kind), ActorID: actor.Subject, Command: "rollback", FromState: continuity.StateRecoverable, ToState: continuity.StateArchived, ReasonCode: "reconciliation_rollback", Status: "pending", CreatedAt: time.Now().UTC()})
	if err != nil {
		return continuity.Receipt{}, err
	}
	if receipt.Command != "rollback" || receipt.SessionID != "catalog" || receipt.ActorID != actor.Subject || receipt.ActorKind != string(actor.Kind) {
		return continuity.Receipt{}, fmt.Errorf("operation_id belongs to a different catalog operation or actor")
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

func (s *Server) Reconcile(ctx context.Context, apply bool, generation, operationID, expectedManifest string, offset, batchSize int) (continuity.ReconcileOutcome, error) {
	return s.reconcileCatalog(ctx, apply, generation, operationID, expectedManifest, offset, batchSize, false)
}

func (s *Server) reconcileCatalog(ctx context.Context, apply bool, generation, operationID, expectedManifest string, offset, batchSize int, missingOnly bool) (continuity.ReconcileOutcome, error) {
	const maxBatch = 500
	if offset < 0 || batchSize < 0 || batchSize > maxBatch {
		return continuity.ReconcileOutcome{}, fmt.Errorf("offset must be nonnegative and batch size must be between 0 and %d", maxBatch)
	}
	if apply && (strings.TrimSpace(operationID) == "" || strings.TrimSpace(generation) == "" || strings.TrimSpace(expectedManifest) == "") {
		return continuity.ReconcileOutcome{}, fmt.Errorf("operation_id, generation and reviewed manifest_hash are required for apply")
	}
	var actor identity.Principal
	target := "web-console:continuity/catalog"
	if missingOnly {
		target = "web-console:continuity/uncataloged"
	}
	if apply {
		var err error
		actor, err = authn.RequireOperation(ctx, "web-console:write", "continuity.reconcile", target)
		if err != nil {
			return continuity.ReconcileOutcome{}, err
		}
		s.catalogApplyMu.Lock()
		defer s.catalogApplyMu.Unlock()
	}
	store := continuity.NewSQLCatalogStore(s.db)
	var progress continuity.ReconciliationProgress
	progressErr := sql.ErrNoRows
	if apply {
		progress, progressErr = store.LoadReconciliationProgress(ctx, operationID)
		if progressErr != nil && progressErr != sql.ErrNoRows {
			return continuity.ReconcileOutcome{}, progressErr
		}
		if progressErr == sql.ErrNoRows && offset != 0 {
			return continuity.ReconcileOutcome{}, fmt.Errorf("new reconciliation must start at offset 0")
		}
		if progressErr == nil && progress.ManifestHash != expectedManifest {
			return continuity.ReconcileOutcome{}, fmt.Errorf("reconciliation operation is bound to a different manifest")
		}
	}
	evidence, err := store.Observe(ctx)
	if err != nil {
		return continuity.ReconcileOutcome{}, err
	}
	existing, err := store.Existing(ctx)
	if err != nil {
		return continuity.ReconcileOutcome{Observations: len(evidence)}, err
	}
	switch {
	case generation == "" && !apply:
		// A preview without a generation binds its manifest to the generation
		// it observes now and returns it. The token an apply presents then
		// dates from the plan it reviewed, not from an integrity audit that
		// spent minutes hashing message bodies while capture kept writing.
		observed, err := continuity.NewIntegrityAuditor(s.db).Generation(ctx)
		if err != nil {
			return continuity.ReconcileOutcome{Observations: len(evidence)}, err
		}
		generation = observed
	case generation != "" && (!missingOnly || progressErr != nil):
		// A scoped operation rechecks the missing-row and alias predicates for
		// each write. Unrelated live inventory changes cannot strand its cursor.
		observed, err := continuity.NewIntegrityAuditor(s.db).Generation(ctx)
		if err != nil {
			return continuity.ReconcileOutcome{Observations: len(evidence)}, err
		}
		if observed != generation {
			return continuity.ReconcileOutcome{Observations: len(evidence)}, fmt.Errorf("integrity generation changed: expected %s, observed %s", generation, observed)
		}
	}
	manifestGeneration := generation
	if missingOnly {
		manifestGeneration = "uncataloged:" + generation
	}
	var items []continuity.ReconcileItem
	if expectedManifest != "" {
		stored, loadErr := store.LoadManifest(ctx, expectedManifest)
		if loadErr == nil {
			if generation != "" && stored.Generation != manifestGeneration {
				return continuity.ReconcileOutcome{Observations: len(evidence), ManifestHash: expectedManifest}, fmt.Errorf("manifest generation mismatch")
			}
			items = stored.Items
		} else if loadErr != sql.ErrNoRows {
			return continuity.ReconcileOutcome{Observations: len(evidence)}, loadErr
		}
	}
	if items == nil {
		if missingOnly {
			missing, err := store.MissingConversationIDs(ctx)
			if err != nil {
				return continuity.ReconcileOutcome{Observations: len(evidence)}, err
			}
			filtered := evidence[:0]
			for _, observation := range evidence {
				if missing[observation.SessionID] {
					filtered = append(filtered, observation)
				}
			}
			evidence = filtered
		}
		items, err = continuity.PlanReconciliation(evidence, existing)
		if err != nil {
			return continuity.ReconcileOutcome{Observations: len(evidence)}, err
		}
	}
	manifestHash := continuity.ManifestHash(manifestGeneration, items)
	if expectedManifest != "" && expectedManifest != manifestHash {
		return continuity.ReconcileOutcome{Observations: len(evidence), ManifestHash: manifestHash}, fmt.Errorf("reconciliation manifest changed: expected %s, observed %s", expectedManifest, manifestHash)
	}

	mutations := 0
	var receipt continuity.Receipt
	if apply {
		if progressErr == nil && progress.TotalItems != len(items) {
			return continuity.ReconcileOutcome{Observations: len(evidence), ManifestHash: manifestHash}, fmt.Errorf("reconciliation progress does not match manifest")
		}
		previous := make(map[string]continuity.CatalogRecord)
		for _, item := range items {
			if prior, ok := existing[item.Record.SessionID]; ok {
				previous[item.Record.SessionID] = prior
			}
		}
		if stored, loadErr := store.LoadManifest(ctx, manifestHash); loadErr == nil {
			previous = stored.Previous
		}
		if err := store.SaveManifest(ctx, continuity.ReconciliationManifest{Hash: manifestHash, Generation: manifestGeneration, Items: items, Previous: previous}); err != nil {
			return continuity.ReconcileOutcome{Observations: len(evidence), ManifestHash: manifestHash}, err
		}
		ledger := continuity.NewSQLLedger(s.db)
		receipt = continuity.Receipt{OperationID: operationID, SessionID: "catalog", ActorKind: string(actor.Kind), ActorID: actor.Subject, Command: "reconcile", FromState: continuity.StateArchived, ToState: continuity.StateRecoverable, ReasonCode: "catalog_reconciliation", Status: "pending", CreatedAt: time.Now().UTC()}
		if receipt, err = ledger.Put(ctx, receipt); err != nil {
			if s.metrics != nil {
				s.metrics.ContinuityFailures.Add(1)
			}
			return continuity.ReconcileOutcome{Observations: len(evidence), ManifestHash: manifestHash}, err
		}
		if s.metrics != nil {
			s.metrics.ContinuityReceipts.Add(1)
		}
		if receipt.Command != "reconcile" || receipt.SessionID != "catalog" || receipt.ActorID != actor.Subject || receipt.ActorKind != string(actor.Kind) {
			return continuity.ReconcileOutcome{Observations: len(evidence), Receipt: receipt, ManifestHash: manifestHash}, fmt.Errorf("operation_id belongs to a different catalog operation or actor")
		}
		if receipt.Status == "failed" {
			if receipt, err = ledger.Reopen(ctx, operationID); err != nil {
				return continuity.ReconcileOutcome{Items: items, Observations: len(evidence), Receipt: receipt, ManifestHash: manifestHash}, err
			}
		}
		if receipt.Status == "succeeded" {
			return continuity.ReconcileOutcome{Observations: len(evidence), Receipt: receipt, ManifestHash: manifestHash, Generation: generation, NextOffset: len(items), Complete: true}, nil
		}
		effectiveBatch := batchSize
		if effectiveBatch <= 0 {
			effectiveBatch = maxBatch
		}
		if progressErr == nil {
			if offset == 0 {
				offset = progress.NextOffset
			} else if offset != progress.NextOffset {
				return continuity.ReconcileOutcome{Items: items, Observations: len(evidence), Receipt: receipt, ManifestHash: manifestHash, NextOffset: progress.NextOffset}, fmt.Errorf("requested offset %d does not match durable offset %d", offset, progress.NextOffset)
			}
		} else if progressErr == sql.ErrNoRows {
			if err := store.SaveReconciliationProgress(ctx, continuity.ReconciliationProgress{OperationID: operationID, ManifestHash: manifestHash, ActorID: actor.Subject, NextOffset: offset, TotalItems: len(items), Status: "pending", UpdatedAt: time.Now().UTC()}); err != nil {
				return continuity.ReconcileOutcome{Items: items, Observations: len(evidence), Receipt: receipt, ManifestHash: manifestHash, NextOffset: offset}, err
			}
		}
		end := offset + effectiveBatch
		if end > len(items) {
			end = len(items)
		}
		applyItem := func(ctx context.Context, item continuity.ReconcileItem) (bool, error) {
			return continuity.ApplyReconciliationItem(ctx, store, item)
		}
		if missingOnly {
			applyItem = store.ApplyMissingConversation
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
			changed, applyErr := applyItem(ctx, item)
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
			if err = store.SaveReconciliationProgress(ctx, continuity.ReconciliationProgress{OperationID: operationID, ManifestHash: manifestHash, ActorID: actor.Subject, NextOffset: nextOffset, TotalItems: len(items), Status: "pending", UpdatedAt: time.Now().UTC()}); err != nil {
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
			if err = store.SaveReconciliationProgress(ctx, continuity.ReconciliationProgress{OperationID: operationID, ManifestHash: manifestHash, ActorID: actor.Subject, NextOffset: nextOffset, TotalItems: len(items), Status: "succeeded", UpdatedAt: time.Now().UTC()}); err != nil {
				return continuity.ReconcileOutcome{Items: items, Observations: len(evidence), Mutations: mutations, Receipt: receipt, ManifestHash: manifestHash, NextOffset: nextOffset}, err
			}
			receipt, err = ledger.Complete(ctx, operationID, "succeeded", "", time.Now().UTC())
		}
		if s.metrics != nil {
			if pending, pendingErr := store.PendingPublicationCount(ctx); pendingErr == nil {
				s.metrics.ContinuityPublicationPending.Store(int64(pending))
			}
		}
		return continuity.ReconcileOutcome{Items: items[offset:end], Observations: len(evidence), Mutations: mutations, Receipt: receipt, ManifestHash: manifestHash, Generation: generation, NextOffset: nextOffset, Complete: nextOffset >= len(items)}, err
	}
	if offset > len(items) {
		return continuity.ReconcileOutcome{Observations: len(evidence), Receipt: receipt, ManifestHash: manifestHash}, fmt.Errorf("preview offset %d exceeds manifest length %d", offset, len(items))
	}
	if batchSize == 0 {
		batchSize = maxBatch
	}
	end := min(offset+batchSize, len(items))
	return continuity.ReconcileOutcome{Items: items[offset:end], Observations: len(evidence), Mutations: mutations, Receipt: receipt, ManifestHash: manifestHash, Generation: generation, NextOffset: end, Complete: end == len(items)}, err
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
	query := r.URL.Query()
	parsePageValue := func(key string) (int, error) {
		if query.Get(key) == "" {
			return 0, nil
		}
		return strconv.Atoi(query.Get(key))
	}
	offset, offsetErr := parsePageValue("offset")
	batchSize, batchErr := parsePageValue("batch_size")
	scope := query.Get("scope")
	if offsetErr != nil || batchErr != nil || (scope != "" && scope != "uncataloged") {
		http.Error(w, `{"error":"invalid catalog reconciliation scope or page"}`, http.StatusBadRequest)
		return
	}
	generation := strings.TrimSpace(query.Get("generation"))
	outcome, err := s.reconcileCatalog(r.Context(), apply, generation, operationID, query.Get("manifest_hash"), offset, batchSize, scope == "uncataloged")
	if err != nil {
		status := http.StatusConflict
		if errors.Is(err, authn.ErrCapability) || errors.Is(err, authn.ErrOperation) {
			status = http.StatusForbidden
		}
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	publicItems := make([]continuity.ReconcileItem, 0, len(outcome.Items))
	for _, item := range outcome.Items {
		publicItems = append(publicItems, continuity.PublicReconcileItem(item))
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"apply": apply, "observations": outcome.Observations, "mutations": outcome.Mutations, "receipt": outcome.Receipt,
		"manifest_hash": outcome.ManifestHash, "generation": outcome.Generation, "next_offset": outcome.NextOffset, "complete": outcome.Complete, "items": publicItems,
	})
}

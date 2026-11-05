package capture

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/provenance"
	learningv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/learning"
	learningconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/learning/learningv1connect"
)

type MemoryResolver func(context.Context) (string, error)

// RunMemoryWorker owns one bounded retry loop. It resolves Memory per pass,
// claims at most batchSize rows, and leaves failures leased/retryable for the
// next pass. Cancellation exits between passes and never deletes evidence.
func RunMemoryWorker(ctx context.Context, repo *Repository, resolve MemoryResolver, interval time.Duration, batchSize int, logger func(string, ...any)) {
	if interval <= 0 {
		interval = 15 * time.Second
	}
	if batchSize <= 0 || batchSize > 100 {
		batchSize = 10
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	drain := func() {
		baseURL, err := resolve(ctx)
		if err != nil {
			if logger != nil {
				logger("memory capture resolve failed: %v", err)
			}
			return
		}
		_, err = repo.Drain(ctx, batchSize, func(deliveryCtx context.Context, attempt Attempt) error {
			return DeliverToMemory(deliveryCtx, baseURL, attempt)
		})
		if err != nil && logger != nil {
			logger("memory capture drain failed: %v", err)
		}
	}
	drain()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			drain()
		}
	}
}

// DeliverToMemory decodes the exact outbox payload and submits it through the
// owner RPC. The owner performs scope and idempotency checks; this adapter only
// transports a bounded attempt record.
func DeliverToMemory(ctx context.Context, baseURL string, attempt Attempt) error {
	if PayloadHash(attempt.Payload) != attempt.PayloadSHA256 {
		return fmt.Errorf("attempt payload integrity check failed")
	}
	var payload struct {
		AttemptID          string   `json:"attempt_id"`
		TaskID             string   `json:"task_id"`
		Operation          string   `json:"operation"`
		ContextKey         string   `json:"context_key"`
		Outcome            string   `json:"outcome"`
		FailureFingerprint string   `json:"failure_fingerprint"`
		StartedAt          string   `json:"started_at"`
		FinishedAt         string   `json:"finished_at"`
		TaskStartedAt      string   `json:"task_started_at"`
		AttemptNumber      int32    `json:"attempt_number"`
		EvidenceRefs       []string `json:"evidence_refs"`
		RecallStatus       string   `json:"recall_status"`
		Provenance         string   `json:"provenance"`
		Trigger            string   `json:"trigger"`
		Approach           string   `json:"approach"`
		ObservationID      string   `json:"observation_id"`
		Disposition        string   `json:"disposition"`
		MethodRevision     string   `json:"method_revision"`
		ObservedAt         string   `json:"observed_at"`
	}
	if err := json.Unmarshal(attempt.Payload, &payload); err != nil {
		return fmt.Errorf("decode attempt payload: %w", err)
	}
	if payload.AttemptID == "" {
		return fmt.Errorf("attempt payload has no identity")
	}
	client := learningconnect.NewLearningServiceClient(&http.Client{Timeout: 30 * time.Second, Transport: provenance.ForwardingTransport{}}, baseURL)
	_, err := client.RecordAttempt(ctx, connect.NewRequest(&learningv1.RecordAttemptRequest{Scope: "web-search-usage", Attempt: &learningv1.Attempt{AttemptId: payload.AttemptID, TaskId: payload.TaskID, Operation: payload.Operation, ContextKey: payload.ContextKey, Outcome: payload.Outcome, FailureFingerprint: payload.FailureFingerprint, StartedAt: payload.StartedAt, FinishedAt: payload.FinishedAt, TaskStartedAt: payload.TaskStartedAt, AttemptNumber: payload.AttemptNumber, EvidenceRefs: payload.EvidenceRefs, RecallStatus: payload.RecallStatus, Provenance: payload.Provenance, Trigger: payload.Trigger, Approach: payload.Approach}}))
	if err != nil {
		return err
	}
	if payload.ObservationID == "" {
		return nil
	}
	_, err = client.RecordObservation(ctx, connect.NewRequest(&learningv1.RecordObservationRequest{Scope: "web-search-usage", Observation: &learningv1.Observation{ObservationId: payload.ObservationID, AttemptId: payload.AttemptID, Disposition: payload.Disposition, EvidenceRefs: payload.EvidenceRefs, MethodRevision: payload.MethodRevision, Provenance: payload.Provenance, ObservedAt: payload.ObservedAt}}))
	return err
}

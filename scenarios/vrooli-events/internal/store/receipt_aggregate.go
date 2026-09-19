package store

import (
	"context"
	"fmt"
	"sort"
	"time"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/proto"
)

const receiptEventType = "vrooli.events.receipt.v1"

// ReceiptAggregateFilter bounds a receipt projection to one UTC time window
// and optional target dimensions. The store keeps the grouping close to the
// durable event source so callers do not need to materialize the full stream.
type ReceiptAggregateFilter struct {
	Since          time.Time
	Until          time.Time
	TargetScenario string
	Operation      string
}

// ReceiptAggregateStore is the narrow owner seam used by the API and tests.
type ReceiptAggregateStore interface {
	AggregateReceipts(context.Context, ReceiptAggregateFilter) ([]*domain.ReceiptAggregate, error)
}

type receiptAggregate struct {
	target        string
	operation     string
	count         int64
	callers       map[string]struct{}
	unattributed  int64
	lastInvokedAt time.Time
}

// AggregateReceipts groups durable receipt envelopes by target scenario and
// operation. Every valid receipt contributes to invocation_count. Only the
// verified Agent Manager attribution written by the Events provenance
// middleware contributes to distinct_verified_callers; all other receipts
// remain visible in unattributed_remainder.
func (s *SQLiteStore) AggregateReceipts(ctx context.Context, filter ReceiptAggregateFilter) ([]*domain.ReceiptAggregate, error) {
	until := filter.Until.UTC()
	if until.IsZero() {
		until = s.config.Now().UTC()
	}
	since := filter.Since.UTC()
	if since.IsZero() {
		since = until.Add(-24 * time.Hour)
	}
	if !since.Before(until) {
		return []*domain.ReceiptAggregate{}, nil
	}
	query := `SELECT id, event_id, source_scenario, target_scenario, event_type, correlation_id, payload, metadata, created_at, expires_at
		FROM events WHERE event_type = ? AND created_at >= ? AND created_at < ?`
	args := []any{receiptEventType, since.Format(time.RFC3339Nano), until.Format(time.RFC3339Nano)}
	if filter.TargetScenario != "" {
		query += " AND target_scenario = ?"
		args = append(args, filter.TargetScenario)
	}
	query += " ORDER BY id ASC"
	events, err := s.scanEvents(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("read receipt events: %w", err)
	}
	groups := make(map[string]*receiptAggregate)
	for _, event := range events {
		var envelope domain.EventEnvelope
		if err := proto.Unmarshal(event.Payload, &envelope); err != nil || envelope.GetTarget() == nil || envelope.GetData() == nil {
			continue
		}
		var receipt domain.ReceiptData
		if len(envelope.GetData().GetValue()) == 0 || proto.Unmarshal(envelope.GetData().GetValue(), &receipt) != nil {
			continue
		}
		target := envelope.GetTarget().GetScenario()
		operation := envelope.GetTarget().GetOperation()
		if target == "" || operation == "" || (filter.Operation != "" && operation != filter.Operation) {
			continue
		}
		key := target + "\x00" + operation
		group := groups[key]
		if group == nil {
			group = &receiptAggregate{target: target, operation: operation, callers: make(map[string]struct{})}
			groups[key] = group
		}
		group.count++
		at := event.CreatedAt
		if envelope.GetOccurredAt() != nil {
			at = envelope.GetOccurredAt().AsTime().UTC()
		}
		if at.After(group.lastInvokedAt) {
			group.lastInvokedAt = at
		}
		attribution := envelope.GetAttribution()
		if attribution.GetVerified() && attribution.GetSubjectKind() == "agent" && attribution.GetSubjectId() != "" {
			group.callers[attribution.GetSubjectId()] = struct{}{}
		} else {
			group.unattributed++
		}
	}
	out := make([]*domain.ReceiptAggregate, 0, len(groups))
	for _, group := range groups {
		out = append(out, &domain.ReceiptAggregate{
			TargetScenario:          group.target,
			Operation:               group.operation,
			InvocationCount:         group.count,
			DistinctVerifiedCallers: int64(len(group.callers)),
			UnattributedRemainder:   group.unattributed,
			LastInvokedAt:           formatAggregateTime(group.lastInvokedAt),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].GetTargetScenario() != out[j].GetTargetScenario() {
			return out[i].GetTargetScenario() < out[j].GetTargetScenario()
		}
		return out[i].GetOperation() < out[j].GetOperation()
	})
	return out, nil
}

func formatAggregateTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

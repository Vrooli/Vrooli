package routing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
)

type Repository interface {
	Create(ctx context.Context, ev *routingv1.RouteEvidence) error
	List(ctx context.Context, filter EvidenceFilter) ([]*routingv1.RouteEvidence, error)
	Get(ctx context.Context, eventID string) (*routingv1.RouteEvidence, error)
}

type EvidenceFilter struct {
	Scenario string
	Limit    int
}

type SQLRepository struct {
	db *sql.DB
}

func NewSQLRepository(db *sql.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

func (r *SQLRepository) Create(ctx context.Context, ev *routingv1.RouteEvidence) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("route evidence repository is not configured")
	}
	policyReasons, err := json.Marshal(ev.GetPolicyReasons())
	if err != nil {
		return err
	}
	failureReasons, err := json.Marshal(ev.GetFailureReasons())
	if err != nil {
		return err
	}
	dimensions, err := json.Marshal(ev.GetAttachmentDimensions())
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO route_events (
event_id, request_id, scenario, operation, role, profile, privacy_class,
selected_provider, selected_locality, status, policy_reasons_json,
failure_reasons_json, fallback_used, prompt_redacted, response_redacted,
latency_ms, created_at,
breaker_state, failure_class, rejection_reason, capacity_verdict,
capacity_claim_id, capacity_required_bytes, capacity_granted_bytes,
capacity_reclaim_required, input_tokens, output_tokens, cost_estimate,
selected_model, image_count, attachment_bytes, attachment_sha256,
attachments_redacted, attachment_dimensions_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ev.GetEventId(),
		ev.GetRequestId(),
		ev.GetScenario(),
		ev.GetOperation(),
		ev.GetRole(),
		int32(ev.GetProfile()),
		int32(ev.GetPrivacyClass()),
		ev.GetSelectedProvider(),
		ev.GetSelectedLocality(),
		ev.GetStatus(),
		string(policyReasons),
		string(failureReasons),
		boolInt(ev.GetFallbackUsed()),
		boolInt(ev.GetPromptRedacted()),
		boolInt(ev.GetResponseRedacted()),
		ev.GetLatencyMs(),
		ev.GetCreatedAt(),
		ev.GetBreakerState(),
		ev.GetFailureClass(),
		ev.GetRejectionReason(),
		ev.GetCapacityVerdict(),
		ev.GetCapacityClaimId(),
		ev.GetCapacityRequiredBytes(),
		ev.GetCapacityGrantedBytes(),
		boolInt(ev.GetCapacityReclaimRequired()),
		ev.GetInputTokens(),
		ev.GetOutputTokens(),
		ev.GetCostEstimate(),
		ev.GetSelectedModel(),
		ev.GetImageCount(),
		ev.GetAttachmentBytes(),
		ev.GetAttachmentSha256(),
		boolInt(ev.GetAttachmentsRedacted()),
		string(dimensions),
	)
	return err
}

func (r *SQLRepository) List(ctx context.Context, filter EvidenceFilter) ([]*routingv1.RouteEvidence, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("route evidence repository is not configured")
	}
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows *sql.Rows
	var err error
	if filter.Scenario == "" {
		rows, err = r.db.QueryContext(ctx, listEvidenceSQL+" ORDER BY created_at DESC LIMIT ?", limit)
	} else {
		rows, err = r.db.QueryContext(ctx, listEvidenceSQL+" WHERE scenario = ? ORDER BY created_at DESC LIMIT ?", filter.Scenario, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*routingv1.RouteEvidence
	for rows.Next() {
		ev, err := scanEvidence(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}

func (r *SQLRepository) Get(ctx context.Context, eventID string) (*routingv1.RouteEvidence, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("route evidence repository is not configured")
	}
	row := r.db.QueryRowContext(ctx, listEvidenceSQL+" WHERE event_id = ?", eventID)
	return scanEvidence(row)
}

const listEvidenceSQL = `SELECT event_id, request_id, scenario, operation, role, profile, privacy_class,
selected_provider, selected_locality, status, policy_reasons_json, failure_reasons_json,
fallback_used, prompt_redacted, response_redacted, latency_ms, created_at,
breaker_state, failure_class, rejection_reason, capacity_verdict, capacity_claim_id,
capacity_required_bytes, capacity_granted_bytes, capacity_reclaim_required,
input_tokens, output_tokens, cost_estimate, selected_model, image_count,
attachment_bytes, attachment_sha256, attachments_redacted,
attachment_dimensions_json
FROM route_events`

type scanner interface {
	Scan(dest ...any) error
}

func scanEvidence(s scanner) (*routingv1.RouteEvidence, error) {
	var ev routingv1.RouteEvidence
	var policyJSON, failureJSON string
	var profile, privacy int32
	var fallbackUsed, promptRedacted, responseRedacted, capacityReclaimRequired, attachmentsRedacted int
	var dimensionsJSON string
	err := s.Scan(
		&ev.EventId,
		&ev.RequestId,
		&ev.Scenario,
		&ev.Operation,
		&ev.Role,
		&profile,
		&privacy,
		&ev.SelectedProvider,
		&ev.SelectedLocality,
		&ev.Status,
		&policyJSON,
		&failureJSON,
		&fallbackUsed,
		&promptRedacted,
		&responseRedacted,
		&ev.LatencyMs,
		&ev.CreatedAt,
		&ev.BreakerState,
		&ev.FailureClass,
		&ev.RejectionReason,
		&ev.CapacityVerdict,
		&ev.CapacityClaimId,
		&ev.CapacityRequiredBytes,
		&ev.CapacityGrantedBytes,
		&capacityReclaimRequired,
		&ev.InputTokens,
		&ev.OutputTokens,
		&ev.CostEstimate,
		&ev.SelectedModel,
		&ev.ImageCount,
		&ev.AttachmentBytes,
		&ev.AttachmentSha256,
		&attachmentsRedacted,
		&dimensionsJSON,
	)
	if err != nil {
		return nil, err
	}
	ev.Profile = profileEnum(profile)
	ev.PrivacyClass = privacyEnum(privacy)
	ev.FallbackUsed = fallbackUsed != 0
	ev.PromptRedacted = promptRedacted != 0
	ev.ResponseRedacted = responseRedacted != 0
	ev.CapacityReclaimRequired = capacityReclaimRequired != 0
	ev.AttachmentsRedacted = attachmentsRedacted != 0
	_ = json.Unmarshal([]byte(policyJSON), &ev.PolicyReasons)
	_ = json.Unmarshal([]byte(failureJSON), &ev.FailureReasons)
	_ = json.Unmarshal([]byte(dimensionsJSON), &ev.AttachmentDimensions)
	return &ev, nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

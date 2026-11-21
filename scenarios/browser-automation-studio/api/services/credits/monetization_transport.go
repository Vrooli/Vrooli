package credits

import (
	"context"
	"fmt"
	"strings"
	"time"

	monetization "github.com/vrooli/vrooli/packages/monetization-go"
)

// EnqueueJourneyUsage records a deterministic Class B event for the governed
// desktop proof. The event enters the same durable outbox used by real BAS
// usage; the journey never fabricates a delivery result.
func (s *Service) EnqueueJourneyUsage(ctx context.Context, userIdentity, operationID string) (monetization.Usage, error) {
	if s == nil || s.monetizationOutbox == nil {
		return monetization.Usage{}, fmt.Errorf("shared usage outbox is unavailable")
	}
	userIdentity = strings.ToLower(strings.TrimSpace(userIdentity))
	operationID = strings.TrimSpace(operationID)
	if userIdentity == "" || operationID == "" {
		return monetization.Usage{}, fmt.Errorf("journey usage identity and operation id are required")
	}
	usage := s.workflowExecutionUsage(userIdentity, operationID, "", "desktop.monetization.boundary", true)
	if err := s.monetizationOutbox.Enqueue(ctx, usage); err != nil {
		return monetization.Usage{}, err
	}
	return usage, nil
}

func (s *Service) workflowExecutionUsage(userIdentity, operationID, workflowID, operation string, journey bool) monetization.Usage {
	metadata := map[string]string{"operation": operation}
	if workflowID != "" {
		metadata["workflow_id"] = workflowID
	}
	if journey {
		metadata["journey"] = "true"
	}
	return monetization.Usage{
		OperationID:  operationID,
		UserIdentity: userIdentity,
		BundleKey:    "business_suite",
		AppKey:       s.appBundleKey,
		MeterKey:     "workflow_executions",
		Units:        1,
		OccurredAt:   time.Now().UTC(),
		Metadata:     metadata,
	}
}

// ReplayJourneyUsage sends the same immutable event a second time through the
// real LPBS transport. LPBS must deduplicate the operation ID; this is the
// upstream half of the exactly-once desktop proof.
func (s *Service) ReplayJourneyUsage(ctx context.Context, usage monetization.Usage) error {
	if s == nil || s.monetizationOutbox == nil || s.monetizationOutbox.Transport == nil {
		return fmt.Errorf("LPBS usage transport is unavailable")
	}
	return s.monetizationOutbox.Transport.Report(ctx, usage)
}

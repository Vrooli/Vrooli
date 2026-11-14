package credits

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	monetization "github.com/vrooli/vrooli/packages/monetization-go"
)

// lpbsUsageTransport is the scenario-owned adapter from the shared outbox
// event to BAS's existing LPBS usage endpoint. Retry and persistence remain in
// packages/monetization-go.
type lpbsUsageTransport struct {
	service *Service
}

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
	usage := monetization.Usage{
		OperationID:  operationID,
		UserIdentity: userIdentity,
		BundleKey:    "business_suite",
		AppKey:       s.appBundleKey,
		MeterKey:     "workflow_executions",
		Units:        1,
		OccurredAt:   time.Now().UTC(),
		Metadata:     map[string]string{"operation": "desktop.monetization.boundary", "journey": "true"},
	}
	if err := s.monetizationOutbox.Enqueue(ctx, usage); err != nil {
		return monetization.Usage{}, err
	}
	return usage, nil
}

// ReplayJourneyUsage sends the same immutable event a second time through the
// real LPBS transport. LPBS must deduplicate the operation ID; this is the
// upstream half of the exactly-once desktop proof.
func (s *Service) ReplayJourneyUsage(ctx context.Context, usage monetization.Usage) error {
	return (&lpbsUsageTransport{service: s}).Report(ctx, usage)
}

func (t *lpbsUsageTransport) Report(ctx context.Context, usage monetization.Usage) error {
	if t == nil || t.service == nil {
		return fmt.Errorf("LPBS usage transport is unavailable")
	}
	accessToken, err := t.service.resolveLPBSAccess(ctx)
	if err != nil {
		return err
	}
	return t.service.sendLPBSReport(ctx, lpbsReportFromUsage(usage), accessToken)
}

func usageFromLPBSReport(report LPBSUsageReport) monetization.Usage {
	metadata := map[string]string{
		"operation":     report.Metadata.Operation,
		"model":         report.Metadata.Model,
		"prompt_tokens": strconv.Itoa(report.Metadata.PromptTokens),
		"is_byok":       strconv.FormatBool(report.Metadata.IsBYOK),
	}
	return monetization.Usage{
		OperationID:  report.OperationID,
		UserIdentity: report.UserIdentity,
		BundleKey:    "business_suite",
		AppKey:       report.AppBundleKey,
		MeterKey:     report.LimitKey,
		Units:        report.UsageAmount,
		OccurredAt:   time.Now().UTC(),
		Metadata:     metadata,
	}
}

func lpbsReportFromUsage(usage monetization.Usage) LPBSUsageReport {
	metadata := usage.Metadata
	promptTokens, _ := strconv.Atoi(metadata["prompt_tokens"])
	byok, _ := strconv.ParseBool(metadata["is_byok"])
	return LPBSUsageReport{
		UserIdentity: usage.UserIdentity,
		LimitKey:     usage.MeterKey,
		UsageAmount:  usage.Units,
		Amount:       usage.Units,
		AppBundleKey: usage.AppKey,
		OperationID:  usage.OperationID,
		Metadata: LPBSUsageReportMetadata{
			Operation:    metadata["operation"],
			Model:        metadata["model"],
			PromptTokens: promptTokens,
			IsBYOK:       byok,
		},
	}
}

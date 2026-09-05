package wiring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/orchestration"
	"agent-manager/internal/runreport"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/eventbus"
	eventspb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

type receiptCaptureDeclaration struct {
	Policies []struct {
		TargetScenario string `json:"targetScenario"`
	} `json:"policies"`
}

type receiptRuntimeReader func(context.Context, string) (eventbus.RuntimeState, error)

func receiptCaptureDeclarationPath() string {
	root, _ := filepath.Abs(filepath.Join("..", "..", ".."))
	return filepath.Join(root, "scenarios", "agent-manager", ".vrooli", "vrooli-events", "receipt-capture.json")
}

func declaredReceiptTargets(path string) ([]string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read receipt capture declaration: %w", err)
	}
	var declaration receiptCaptureDeclaration
	if err := json.Unmarshal(body, &declaration); err != nil {
		return nil, fmt.Errorf("decode receipt capture declaration: %w", err)
	}
	unique := map[string]struct{}{}
	for _, policy := range declaration.Policies {
		if target := strings.TrimSpace(policy.TargetScenario); target != "" {
			unique[target] = struct{}{}
		}
	}
	targets := make([]string, 0, len(unique))
	for target := range unique {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	return targets, nil
}

func productionReceiptRuntimeReader(ctx context.Context, target string) (eventbus.RuntimeState, error) {
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, target)
	if err != nil {
		return eventbus.RuntimeState{}, err
	}
	return eventbus.ReadRuntimeState(ctx, baseURL, &http.Client{Timeout: 2 * time.Second})
}

// receiptRuntimeAvailability preserves the existing report vocabulary while
// making the former ambiguous empty ledger actionable. A target that has never
// received a policy is unavailable; a connected target with no policy is a
// policy declaration gap; only an armed target can honestly be unobserved.
func receiptRuntimeAvailability(ctx context.Context, targets []string, read receiptRuntimeReader) (runreport.ReceiptSummary, bool) {
	if len(targets) == 0 {
		return runreport.ReceiptSummary{Availability: runreport.Availability{State: runreport.AvailabilityPolicyAbsent, Reason: "agent-manager declares no receipt capture targets"}}, true
	}
	if read == nil {
		return runreport.ReceiptSummary{Availability: runreport.Availability{State: runreport.AvailabilityUnavailable, Reason: "receipt runtime state reader is not configured"}}, true
	}
	for _, target := range targets {
		state, err := read(ctx, target)
		if err != nil {
			return runreport.ReceiptSummary{Availability: runreport.Availability{State: runreport.AvailabilityUnavailable, Reason: fmt.Sprintf("receipt runtime for %s is unavailable", target)}}, true
		}
		switch state.State {
		case "never_connected":
			return runreport.ReceiptSummary{Availability: runreport.Availability{State: runreport.AvailabilityUnavailable, Reason: fmt.Sprintf("receipt runtime for %s has never connected to vrooli-events", target)}}, true
		case "connected_empty":
			return runreport.ReceiptSummary{Availability: runreport.Availability{State: runreport.AvailabilityPolicyAbsent, Reason: fmt.Sprintf("receipt runtime for %s is connected but has no capture policies", target)}}, true
		case "stale":
			return runreport.ReceiptSummary{Availability: runreport.Availability{State: runreport.AvailabilityUnavailable, Reason: fmt.Sprintf("receipt runtime for %s has a stale policy snapshot", target)}}, true
		case "armed":
			if !state.Armed {
				return runreport.ReceiptSummary{Availability: runreport.Availability{State: runreport.AvailabilityUnavailable, Reason: fmt.Sprintf("receipt runtime for %s is not armed", target)}}, true
			}
		default:
			return runreport.ReceiptSummary{Availability: runreport.Availability{State: runreport.AvailabilityUnavailable, Reason: fmt.Sprintf("receipt runtime for %s reported unknown state %q", target, state.State)}}, true
		}
	}
	return runreport.ReceiptSummary{}, false
}

func newReceiptSummaryReader(receipts eventbus.Client, targets []string, runtime receiptRuntimeReader) orchestration.ReceiptSummaryReader {
	return orchestration.ReceiptSummaryReaderFunc(func(ctx context.Context, id uuid.UUID) (runreport.ReceiptSummary, error) {
		if !receipts.Enabled() {
			return runreport.ReceiptSummary{Availability: runreport.Availability{State: runreport.AvailabilityUnavailable, Reason: "vrooli-events observations are not configured"}}, nil
		}
		observations, err := receipts.ReceiptQuery(ctx, id.String(), 100)
		if err != nil {
			return runreport.ReceiptSummary{}, err
		}
		verified := 0
		eventIDs := []string{}
		calls := []runreport.CrossScenarioCall{}
		for _, raw := range observations {
			envelope := &eventspb.EventEnvelope{}
			if protojson.Unmarshal(raw, envelope) != nil || envelope.EventType != eventbus.ReceiptEventType || envelope.Correlation == nil || envelope.Correlation.AgentRunId != id.String() {
				continue
			}
			call := runreport.CrossScenarioCall{TargetScenario: envelope.GetTarget().GetScenario(), Operation: envelope.GetTarget().GetOperation(), ReceiptEventID: envelope.EventId, Verified: envelope.GetAttribution() != nil && envelope.GetAttribution().GetVerified()}
			if call.Verified {
				call.MatchConfidence, call.MatchReason = "exact", "verified agent-run correlation"
			} else {
				call.MatchConfidence, call.MatchReason = "unmatched", "receipt correlation is not verified"
			}
			if occurredAt := envelope.GetOccurredAt(); occurredAt != nil {
				call.OccurredAt = occurredAt.AsTime()
			}
			receipt := &eventspb.ReceiptData{}
			if envelope.GetData() != nil && envelope.GetData().UnmarshalTo(receipt) == nil {
				call.Outcome, call.StatusCode, call.DurationMS = receipt.GetOutcome(), receipt.GetStatusCode(), receipt.GetDurationMs()
				call.PolicyVersion = receipt.GetPolicyVersion()
				if receipt.GetProjection() != nil {
					call.Projection, call.ProjectionDropCount = runreport.BoundProjection(receipt.GetProjection().AsMap())
				}
			}
			calls = append(calls, call)
			if call.Verified {
				verified++
				eventIDs = append(eventIDs, envelope.EventId)
			}
		}
		if verified == 0 {
			if summary, decisive := receiptRuntimeAvailability(ctx, targets, runtime); decisive {
				summary.Calls, summary.UnmatchedCount = calls, len(calls)
				return summary, nil
			}
			return runreport.ReceiptSummary{Availability: runreport.Availability{State: runreport.AvailabilityUnobserved, Reason: "no verified receipts correlated to this run"}, Calls: calls, UnmatchedCount: len(calls)}, nil
		}
		return runreport.ReceiptSummary{Availability: runreport.Availability{State: runreport.AvailabilityAvailable}, Count: verified, UnmatchedCount: len(calls) - verified, EventIDs: eventIDs, Calls: calls}, nil
	})
}

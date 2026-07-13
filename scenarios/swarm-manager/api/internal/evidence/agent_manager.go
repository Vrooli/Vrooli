package evidence

import (
	"context"
	"fmt"
	"strings"
	"time"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"swarm-manager/internal/agentmanager"
)

// AgentManagerReader is the bounded read surface used by the supplemental
// collector. It intentionally excludes raw transcript and command-output
// retention: only event identity, tool identity/outcome, and diff file names
// are normalized into ledger observations.
type AgentManagerReader interface {
	GetRunEvents(context.Context, string, agentmanager.RunEventsOptions) ([]*domainpb.RunEvent, bool, error)
	GetRunDiff(context.Context, string) (agentmanager.RunDiff, error)
}

// ReconcileAgentManager collects bounded, supplemental facts from the Agent
// Manager event and diff APIs. It never parses command text or tool output:
// tool calls/results prove only that a named tool was invoked or succeeded,
// while changed paths come from Agent Manager's diff endpoint.
func (s *Service) ReconcileAgentManager(ctx context.Context, reader AgentManagerReader, runID string) ([]IngestResult, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("evidence service is not configured")
	}
	if reader == nil || strings.TrimSpace(runID) == "" {
		return nil, fmt.Errorf("agent-manager reader and run id are required")
	}
	results, cursor, err := s.reconcileAgentManagerEvents(ctx, reader, runID)
	if err != nil {
		return nil, err
	}
	if err := s.store.SaveCheckpoint(ctx, Checkpoint{ProducerID: "agent-manager-events", RunID: runID, FactKind: "agent_tool", Cursor: cursor}); err != nil {
		return nil, fmt.Errorf("save agent-manager event checkpoint: %w", err)
	}
	if err := s.store.SaveWatermark(ctx, Watermark{ProducerID: "agent-manager-events", RunID: runID, FactKind: "agent_tool", Coverage: "complete paginated run event stream"}); err != nil {
		return nil, fmt.Errorf("save agent-manager event terminal watermark: %w", err)
	}

	diff, err := reader.GetRunDiff(ctx, runID)
	if err != nil {
		return results, fmt.Errorf("fetch agent-manager run diff: %w", err)
	}
	for index, file := range diff.Files {
		path := strings.TrimSpace(file.Path)
		if path == "" {
			continue
		}
		result, err := s.Ingest(ctx, Observation{
			SourceSystem: "agent-manager-diff", SourceEventID: fmt.Sprintf("%s:file:%d:%s", runID, index, path), RunID: runID,
			Subject: Subject{Kind: "repository_change", ID: path}, Action: defaultAction(file.ChangeType, "changed"),
			Confidence: ConfidenceObserved, Verification: VerificationVerified,
			Metadata: map[string]string{"sandbox_id": strings.TrimSpace(diff.SandboxID)}, ObservedAt: parseAgentManagerTime(diff.GeneratedAt),
		})
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}
	if err := s.store.SaveCheckpoint(ctx, Checkpoint{ProducerID: "agent-manager-diff", RunID: runID, FactKind: "repository_change", Cursor: strings.TrimSpace(diff.GeneratedAt)}); err != nil {
		return results, fmt.Errorf("save agent-manager diff checkpoint: %w", err)
	}
	if err := s.store.SaveWatermark(ctx, Watermark{ProducerID: "agent-manager-diff", RunID: runID, FactKind: "repository_change", Coverage: "complete run diff"}); err != nil {
		return results, fmt.Errorf("save agent-manager diff terminal watermark: %w", err)
	}
	return results, nil
}

func (s *Service) reconcileAgentManagerEvents(ctx context.Context, reader AgentManagerReader, runID string) ([]IngestResult, string, error) {
	var results []IngestResult
	var after int64
	cursor := "0"
	for {
		events, hasMore, err := reader.GetRunEvents(ctx, runID, agentmanager.RunEventsOptions{AfterSequence: after, Limit: 200})
		if err != nil {
			return results, cursor, fmt.Errorf("fetch agent-manager run events: %w", err)
		}
		for _, event := range events {
			if event.GetSequence() > after {
				after = event.GetSequence()
			}
			cursor = fmt.Sprintf("%d", after)
			if event.GetToolCall() != nil {
				result, err := s.Ingest(ctx, toolObservation(runID, event, "invoked", event.GetToolCall().GetToolName(), event.GetToolCall().GetToolCallId()))
				if err != nil {
					return results, cursor, err
				}
				results = append(results, result)
			}
			if resultEvent := event.GetToolResult(); resultEvent != nil && resultEvent.GetSuccess() {
				result, err := s.Ingest(ctx, toolObservation(runID, event, "succeeded", resultEvent.GetToolName(), resultEvent.GetToolCallId()))
				if err != nil {
					return results, cursor, err
				}
				results = append(results, result)
			}
		}
		if !hasMore || len(events) == 0 {
			return results, cursor, nil
		}
	}
}

func toolObservation(runID string, event *domainpb.RunEvent, action, toolName, toolCallID string) Observation {
	eventID := strings.TrimSpace(event.GetId())
	if eventID == "" {
		eventID = fmt.Sprintf("%s:event:%d", runID, event.GetSequence())
	}
	if strings.TrimSpace(toolCallID) == "" {
		toolCallID = eventID
	}
	observedAt := time.Now().UTC()
	if event.GetTimestamp() != nil {
		observedAt = event.GetTimestamp().AsTime().UTC()
	}
	return Observation{
		SourceSystem: "agent-manager-events", SourceEventID: eventID, RunID: runID,
		Subject: Subject{Kind: "agent_tool", ID: strings.TrimSpace(toolCallID)}, Action: action,
		Confidence: ConfidenceObserved, Verification: VerificationVerified,
		Metadata: map[string]string{"tool_name": strings.TrimSpace(toolName)}, ObservedAt: observedAt,
	}
}

func parseAgentManagerTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Now().UTC()
	}
	return parsed.UTC()
}

func defaultAction(value, fallback string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return fallback
}

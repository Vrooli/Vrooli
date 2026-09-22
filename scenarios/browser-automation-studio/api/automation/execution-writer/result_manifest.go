package executionwriter

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// buildResultManifestPayload owns the durable, human-readable execution
// manifest. FileWriter owns synchronization and file placement; this function
// owns the result contract, making future manifest versions independently
// testable without storage or database fixtures.
func buildResultManifestPayload(executionID uuid.UUID, result *ExecutionResultData, timeline *executionTimelineData) (map[string]any, error) {
	payload := map[string]any{}

	if timeline != nil && timeline.pb != nil {
		// Snapshot before shaping: the full timeline remains the artifact authority.
		timeline.mu.Lock()
		manifest := proto.Clone(timeline.pb).(*bastimeline.ExecutionTimeline)
		timeline.mu.Unlock()
		for _, entry := range manifest.Entries {
			if aggregates := entry.GetAggregates(); aggregates != nil {
				aggregates.Artifacts = nil
			}
		}
		raw, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: false}.Marshal(manifest)
		if err != nil {
			return nil, fmt.Errorf("marshal timeline payload: %w", err)
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, fmt.Errorf("parse timeline payload: %w", err)
		}
	}

	if len(payload) == 0 {
		payload["execution_id"] = executionID.String()
		if result != nil && strings.TrimSpace(result.WorkflowID) != "" {
			payload["workflow_id"] = result.WorkflowID
		}
		return payload, nil
	}

	if result != nil && result.Summary.TotalSteps > 0 {
		status := "EXECUTION_STATUS_RUNNING"
		if result.Summary.FailedSteps > 0 {
			status = "EXECUTION_STATUS_FAILED"
		} else if result.Summary.CompletedSteps >= result.Summary.TotalSteps {
			status = "EXECUTION_STATUS_COMPLETED"
		}
		payload["status"] = status
	}

	return payload, nil
}

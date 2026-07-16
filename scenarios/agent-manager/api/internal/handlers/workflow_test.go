package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestWorkflowExecutionProjectionRedactsPayloadsByDefault(t *testing.T) {
	execution := &domain.WorkflowExecution{
		ID: uuid.New(), Owner: "example", WorkflowKey: "example/review", DefinitionDigest: "sha256:test",
		Status: domain.WorkflowExecutionSucceeded, Input: json.RawMessage(`{"secret":"prompt"}`), Output: json.RawMessage(`{"secret":"result"}`),
		EdgeTraversals: map[string]int{}, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	redacted := workflowExecutionToProto(execution, false)
	if redacted.Input != nil || redacted.Output != nil {
		t.Fatalf("routine projection leaked payloads: input=%v output=%v", redacted.Input, redacted.Output)
	}
	authorized := workflowExecutionToProto(execution, true)
	if authorized.Input == nil || authorized.Output == nil {
		t.Fatalf("authorized projection omitted payloads: input=%v output=%v", authorized.Input, authorized.Output)
	}
}

func TestWorkflowAttemptProjectionExposesIdentityAndInputMetadataWithoutPayload(t *testing.T) {
	input := json.RawMessage(`{"customer":"sensitive"}`)
	attempt := &domain.WorkflowNodeAttempt{
		ID:              uuid.New(),
		ExecutionID:     uuid.New(),
		NodeID:          "review",
		Ordinal:         1,
		Strategy:        domain.WorkflowAttemptFreshRun,
		Status:          domain.WorkflowAttemptWaiting,
		InputSnapshot:   input,
		ProfileIdentity: "role:reviewer",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	projected := workflowAttemptToProto(attempt)
	digest := sha256.Sum256(input)
	if projected.GetProfileIdentity() != "role:reviewer" {
		t.Fatalf("profile identity = %q, want role:reviewer", projected.GetProfileIdentity())
	}
	if projected.GetInputSnapshotDigest() != fmt.Sprintf("sha256:%x", digest[:]) {
		t.Fatalf("input digest = %q", projected.GetInputSnapshotDigest())
	}
	if projected.GetInputSnapshotSizeBytes() != int64(len(input)) {
		t.Fatalf("input size = %d, want %d", projected.GetInputSnapshotSizeBytes(), len(input))
	}
}

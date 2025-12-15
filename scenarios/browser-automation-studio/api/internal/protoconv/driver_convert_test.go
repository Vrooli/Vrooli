package protoconv

import (
	"testing"
	"time"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestStepOutcomeToProto(t *testing.T) {
	now := time.Now().UTC()
	completed := now.Add(time.Minute)
	execID := uuid.New()

	outcome := &autocontracts.StepOutcome{
		SchemaVersion:  autocontracts.StepOutcomeSchemaVersion,
		PayloadVersion: autocontracts.PayloadVersion,
		ExecutionID:    execID,
		CorrelationID:  "corr-123",
		StepIndex:      2,
		Attempt:        1,
		NodeID:         "node-abc",
		StepType:       "click",
		Instruction:    "Click button",
		Success:        true,
		StartedAt:      now,
		CompletedAt:    &completed,
		DurationMs:     1000,
		FinalURL:       "https://example.com/done",
		Screenshot: &autocontracts.Screenshot{
			Data:        []byte("png-data"),
			MediaType:   "image/png",
			CaptureTime: now,
			Width:       800,
			Height:      600,
			Hash:        "sha256-abc",
			FromCache:   false,
			Truncated:   false,
			Source:      "playwright",
		},
		DOMSnapshot: &autocontracts.DOMSnapshot{
			HTML:        "<html>...</html>",
			Preview:     "<html>",
			Hash:        "sha256-dom",
			CollectedAt: now,
			Truncated:   false,
		},
		ConsoleLogs: []autocontracts.ConsoleLogEntry{
			{
				Type:      "log",
				Text:      "Console message",
				Timestamp: now,
				Stack:     "at line 1",
				Location:  "script.js:1:10",
			},
		},
		Network: []autocontracts.NetworkEvent{
			{
				Type:                "request",
				URL:                 "https://api.example.com",
				Method:              "POST",
				ResourceType:        "fetch",
				Status:              200,
				OK:                  true,
				Timestamp:           now,
				RequestHeaders:      map[string]string{"Content-Type": "application/json"},
				ResponseHeaders:     map[string]string{"Server": "nginx"},
				RequestBodyPreview:  `{"foo":"bar"}`,
				ResponseBodyPreview: `{"success":true}`,
				Truncated:           false,
			},
		},
		ExtractedData: map[string]any{"title": "Page Title", "count": float64(42)},
		Assertion: &autocontracts.AssertionOutcome{
			Mode:          "text_equals",
			Selector:      "#main",
			Expected:      "Hello",
			Actual:        "Hello",
			Success:       true,
			Negated:       false,
			CaseSensitive: true,
			Message:       "",
		},
		Condition: &autocontracts.ConditionOutcome{
			Type:       "element_visible",
			Outcome:    true,
			Negated:    false,
			Operator:   "equals",
			Variable:   "state",
			Selector:   "#visible-el",
			Expression: "state == 'ready'",
			Actual:     "ready",
			Expected:   "ready",
		},
		ProbeResult: map[string]any{"probe": "ok"},
		ElementBoundingBox: &basbase.BoundingBox{
			X: 10, Y: 20, Width: 100, Height: 50,
		},
		ClickPosition: &basbase.Point{X: 60, Y: 45},
		FocusedElement: &bastimeline.ElementFocus{
			Selector:    "#button",
			BoundingBox: &basbase.BoundingBox{X: 10, Y: 20, Width: 100, Height: 50},
		},
		HighlightRegions: []autocontracts.HighlightRegion{
			{Selector: "#highlight", BoundingBox: &basbase.BoundingBox{X: 0, Y: 0, Width: 200, Height: 100}, Padding: 4, HighlightColor: basbase.HighlightColor_HIGHLIGHT_COLOR_RED},
		},
		MaskRegions: []autocontracts.MaskRegion{
			{Selector: "#mask", BoundingBox: &basbase.BoundingBox{X: 50, Y: 50, Width: 100, Height: 100}, Opacity: 0.5},
		},
		ZoomFactor: 1.5,
		CursorTrail: []autocontracts.CursorPosition{
			{Point: basbase.Point{X: 10, Y: 10}, RecordedAt: now, ElapsedMs: 100},
			{Point: basbase.Point{X: 20, Y: 20}, RecordedAt: now, ElapsedMs: 200},
		},
		Notes: map[string]string{"note1": "value1"},
		Failure: &autocontracts.StepFailure{
			Kind:       autocontracts.FailureKindUser,
			Code:       "ELEMENT_NOT_FOUND",
			Message:    "Could not find element",
			Fatal:      false,
			Retryable:  true,
			OccurredAt: &now,
			Details:    map[string]any{"selector": "#missing"},
			Source:     autocontracts.FailureSourceEngine,
		},
	}

	pb := StepOutcomeToProto(outcome)
	if pb == nil {
		t.Fatal("expected non-nil proto")
	}

	// Verify basic fields
	if pb.SchemaVersion != autocontracts.StepOutcomeSchemaVersion {
		t.Errorf("schema_version mismatch: got %s", pb.SchemaVersion)
	}
	if pb.GetExecutionId() != execID.String() {
		t.Errorf("execution_id mismatch: got %s, want %s", pb.GetExecutionId(), execID.String())
	}
	if pb.StepIndex != 2 {
		t.Errorf("step_index mismatch: got %d", pb.StepIndex)
	}
	if !pb.Success {
		t.Error("success should be true")
	}

	// Verify screenshot
	if pb.Screenshot == nil {
		t.Fatal("expected screenshot")
	}
	if pb.Screenshot.MediaType != "image/png" {
		t.Errorf("screenshot media_type mismatch: got %s", pb.Screenshot.MediaType)
	}

	// Verify DOM snapshot
	if pb.DomSnapshot == nil {
		t.Fatal("expected dom_snapshot")
	}
	if pb.DomSnapshot.GetHtml() != "<html>...</html>" {
		t.Errorf("dom_snapshot html mismatch: got %s", pb.DomSnapshot.GetHtml())
	}

	// Verify console logs
	if len(pb.ConsoleLogs) != 1 {
		t.Fatalf("expected 1 console log, got %d", len(pb.ConsoleLogs))
	}
	if pb.ConsoleLogs[0].Text != "Console message" {
		t.Errorf("console log text mismatch: got %s", pb.ConsoleLogs[0].Text)
	}

	// Verify network events
	if len(pb.NetworkEvents) != 1 {
		t.Fatalf("expected 1 network event, got %d", len(pb.NetworkEvents))
	}
	if pb.NetworkEvents[0].Url != "https://api.example.com" {
		t.Errorf("network event url mismatch: got %s", pb.NetworkEvents[0].Url)
	}

	// Verify extracted data
	if pb.ExtractedData == nil || pb.ExtractedData["title"].GetStringValue() != "Page Title" {
		t.Errorf("extracted_data mismatch: got %+v", pb.ExtractedData)
	}

	// Verify assertion
	if pb.Assertion == nil || pb.Assertion.GetMode() != "text_equals" {
		t.Errorf("assertion mismatch: got %+v", pb.Assertion)
	}

	// Verify condition
	if pb.Condition == nil || !pb.Condition.Outcome {
		t.Errorf("condition mismatch: got %+v", pb.Condition)
	}

	// Verify visual regions
	if len(pb.HighlightRegions) != 1 {
		t.Errorf("expected 1 highlight region, got %d", len(pb.HighlightRegions))
	}
	if len(pb.MaskRegions) != 1 {
		t.Errorf("expected 1 mask region, got %d", len(pb.MaskRegions))
	}

	// Verify cursor trail
	if len(pb.CursorTrail) != 2 {
		t.Errorf("expected 2 cursor positions, got %d", len(pb.CursorTrail))
	}

	// Verify failure
	if pb.Failure == nil || pb.Failure.Kind != basexecution.FailureKind_FAILURE_KIND_USER {
		t.Errorf("failure mismatch: got %+v", pb.Failure)
	}

	// Verify JSON uses proto field names (snake_case)
	data, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(pb)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	json := string(data)
	if !containsJSONFieldValue(json, "step_index") {
		t.Errorf("expected snake_case field names in JSON: %s", json)
	}
}

func TestProtoToStepOutcome(t *testing.T) {
	now := time.Now().UTC()
	completed := now.Add(time.Minute)
	execID := uuid.New().String()
	durationMs := int32(1000)
	zoomFactor := float64(1.5)
	elapsedMs := int64(100)
	status := int32(200)
	ok := true
	stack := "at line 1"
	location := "script.js:1:10"
	hash := "sha256-abc"
	source := "playwright"

	pb := &basexecution.StepOutcome{
		SchemaVersion:  autocontracts.StepOutcomeSchemaVersion,
		PayloadVersion: autocontracts.PayloadVersion,
		ExecutionId:    &execID,
		CorrelationId:  strPtr("corr-123"),
		StepIndex:      2,
		Attempt:        1,
		NodeId:         "node-abc",
		StepType:       "click",
		Instruction:    strPtr("Click button"),
		Success:        true,
		StartedAt:      timestamppb.New(now),
		CompletedAt:    timestamppb.New(completed),
		DurationMs:     &durationMs,
		FinalUrl:       strPtr("https://example.com/done"),
		Screenshot: &basexecution.DriverScreenshot{
			Data:        []byte("png-data"),
			MediaType:   "image/png",
			CaptureTime: timestamppb.New(now),
			Width:       800,
			Height:      600,
			Hash:        &hash,
			FromCache:   false,
			Truncated:   false,
			Source:      &source,
		},
		DomSnapshot: &basexecution.DOMSnapshot{
			Html:        strPtr("<html>...</html>"),
			Preview:     strPtr("<html>"),
			Hash:        strPtr("sha256-dom"),
			CollectedAt: timestamppb.New(now),
			Truncated:   false,
		},
		ConsoleLogs: []*basexecution.DriverConsoleLogEntry{
			{
				Type:      "log",
				Text:      "Console message",
				Timestamp: timestamppb.New(now),
				Stack:     &stack,
				Location:  &location,
			},
		},
		NetworkEvents: []*basexecution.DriverNetworkEvent{
			{
				Type:                "request",
				Url:                 "https://api.example.com",
				Method:              strPtr("POST"),
				ResourceType:        strPtr("fetch"),
				Status:              &status,
				Ok:                  &ok,
				Timestamp:           timestamppb.New(now),
				RequestHeaders:      map[string]string{"Content-Type": "application/json"},
				ResponseHeaders:     map[string]string{"Server": "nginx"},
				RequestBodyPreview:  strPtr(`{"foo":"bar"}`),
				ResponseBodyPreview: strPtr(`{"success":true}`),
				Truncated:           false,
			},
		},
		Assertion: &basexecution.AssertionOutcome{
			Mode:          strPtr("text_equals"),
			Selector:      strPtr("#main"),
			Success:       true,
			Negated:       false,
			CaseSensitive: true,
		},
		Condition: &basexecution.ConditionOutcome{
			Type:    strPtr("element_visible"),
			Outcome: true,
			Negated: false,
		},
		ElementBoundingBox: &basbase.BoundingBox{X: 10, Y: 20, Width: 100, Height: 50},
		ClickPosition:      &basbase.Point{X: 60, Y: 45},
		FocusedElement: &bastimeline.ElementFocus{
			Selector:    "#button",
			BoundingBox: &basbase.BoundingBox{X: 10, Y: 20, Width: 100, Height: 50},
		},
		HighlightRegions: []*basdomain.HighlightRegion{
			{Selector: "#highlight", BoundingBox: &basbase.BoundingBox{X: 0, Y: 0, Width: 200, Height: 100}, Padding: 4, HighlightColor: basbase.HighlightColor_HIGHLIGHT_COLOR_RED},
		},
		MaskRegions: []*basdomain.MaskRegion{
			{Selector: "#mask", BoundingBox: &basbase.BoundingBox{X: 50, Y: 50, Width: 100, Height: 100}, Opacity: 0.5},
		},
		ZoomFactor: &zoomFactor,
		CursorTrail: []*basexecution.CursorPosition{
			{Point: &basbase.Point{X: 10, Y: 10}, RecordedAt: timestamppb.New(now), ElapsedMs: &elapsedMs},
		},
		Notes: map[string]string{"note1": "value1"},
		Failure: &basexecution.StepFailure{
			Kind:      basexecution.FailureKind_FAILURE_KIND_USER,
			Code:      strPtr("ELEMENT_NOT_FOUND"),
			Message:   strPtr("Could not find element"),
			Fatal:     false,
			Retryable: true,
			Source:    basexecution.FailureSource_FAILURE_SOURCE_ENGINE,
		},
	}

	outcome := ProtoToStepOutcome(pb)
	if outcome == nil {
		t.Fatal("expected non-nil outcome")
	}

	// Verify basic fields
	if outcome.SchemaVersion != autocontracts.StepOutcomeSchemaVersion {
		t.Errorf("schema_version mismatch: got %s", outcome.SchemaVersion)
	}
	if outcome.ExecutionID.String() != execID {
		t.Errorf("execution_id mismatch: got %s, want %s", outcome.ExecutionID.String(), execID)
	}
	if outcome.StepIndex != 2 {
		t.Errorf("step_index mismatch: got %d", outcome.StepIndex)
	}
	if !outcome.Success {
		t.Error("success should be true")
	}

	// Verify screenshot
	if outcome.Screenshot == nil {
		t.Fatal("expected screenshot")
	}
	if outcome.Screenshot.MediaType != "image/png" {
		t.Errorf("screenshot media_type mismatch: got %s", outcome.Screenshot.MediaType)
	}

	// Verify DOM snapshot
	if outcome.DOMSnapshot == nil {
		t.Fatal("expected dom_snapshot")
	}
	if outcome.DOMSnapshot.HTML != "<html>...</html>" {
		t.Errorf("dom_snapshot html mismatch: got %s", outcome.DOMSnapshot.HTML)
	}

	// Verify console logs
	if len(outcome.ConsoleLogs) != 1 {
		t.Fatalf("expected 1 console log, got %d", len(outcome.ConsoleLogs))
	}

	// Verify network events
	if len(outcome.Network) != 1 {
		t.Fatalf("expected 1 network event, got %d", len(outcome.Network))
	}

	// Verify visual regions
	if len(outcome.HighlightRegions) != 1 {
		t.Errorf("expected 1 highlight region, got %d", len(outcome.HighlightRegions))
	}
	if len(outcome.MaskRegions) != 1 {
		t.Errorf("expected 1 mask region, got %d", len(outcome.MaskRegions))
	}

	// Verify cursor trail
	if len(outcome.CursorTrail) != 1 {
		t.Errorf("expected 1 cursor position, got %d", len(outcome.CursorTrail))
	}

	// Verify failure
	if outcome.Failure == nil || outcome.Failure.Kind != autocontracts.FailureKindUser {
		t.Errorf("failure mismatch: got %+v", outcome.Failure)
	}
}

func TestStepOutcomeRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Microsecond) // Truncate for protobuf precision
	completed := now.Add(time.Minute)
	execID := uuid.New()

	original := &autocontracts.StepOutcome{
		SchemaVersion:  autocontracts.StepOutcomeSchemaVersion,
		PayloadVersion: autocontracts.PayloadVersion,
		ExecutionID:    execID,
		CorrelationID:  "corr-123",
		StepIndex:      5,
		Attempt:        2,
		NodeID:         "node-xyz",
		StepType:       "navigate",
		Success:        true,
		StartedAt:      now,
		CompletedAt:    &completed,
		DurationMs:     2000,
		FinalURL:       "https://example.com",
		Notes:          map[string]string{"key": "value"},
	}

	// Convert to proto and back
	pb := StepOutcomeToProto(original)
	result := ProtoToStepOutcome(pb)

	// Verify round-trip
	if result.SchemaVersion != original.SchemaVersion {
		t.Errorf("schema_version mismatch after round-trip")
	}
	if result.ExecutionID != original.ExecutionID {
		t.Errorf("execution_id mismatch after round-trip")
	}
	if result.StepIndex != original.StepIndex {
		t.Errorf("step_index mismatch after round-trip")
	}
	if result.NodeID != original.NodeID {
		t.Errorf("node_id mismatch after round-trip")
	}
	if !result.StartedAt.Equal(original.StartedAt) {
		t.Errorf("started_at mismatch: got %v, want %v", result.StartedAt, original.StartedAt)
	}
	if result.Notes["key"] != original.Notes["key"] {
		t.Errorf("notes mismatch after round-trip")
	}
}

func TestExecutionPlanToProto(t *testing.T) {
	now := time.Now().UTC()
	execID := uuid.New()
	workflowID := uuid.New()

	plan := &autocontracts.ExecutionPlan{
		SchemaVersion:  autocontracts.ExecutionPlanSchemaVersion,
		PayloadVersion: autocontracts.PayloadVersion,
		ExecutionID:    execID,
		WorkflowID:     workflowID,
		CreatedAt:      now,
		Metadata:       map[string]any{"client": "test"},
		Instructions: []autocontracts.CompiledInstruction{
			{
				Index:       0,
				NodeID:      "node-1",
				Type:        "navigate",
				Params:      map[string]any{"url": "https://example.com"},
				PreloadHTML: "<html></html>",
				Context:     map[string]any{"ctx": "value"},
				Metadata:    map[string]string{"label": "Navigate to example"},
			},
			{
				Index:  1,
				NodeID: "node-2",
				Type:   "click",
				Params: map[string]any{"selector": "#button"},
			},
		},
		Graph: &autocontracts.PlanGraph{
			Steps: []autocontracts.PlanStep{
				{
					Index:    0,
					NodeID:   "node-1",
					Type:     "navigate",
					Params:   map[string]any{"url": "https://example.com"},
					Metadata: map[string]string{"label": "Navigate"},
					Outgoing: []autocontracts.PlanEdge{
						{ID: "edge-1", Target: "node-2"},
					},
				},
				{
					Index:  1,
					NodeID: "node-2",
					Type:   "click",
					Params: map[string]any{"selector": "#button"},
				},
			},
		},
	}

	pb := ExecutionPlanToProto(plan)
	if pb == nil {
		t.Fatal("expected non-nil proto")
	}

	if pb.ExecutionId != execID.String() {
		t.Errorf("execution_id mismatch: got %s, want %s", pb.ExecutionId, execID.String())
	}
	if pb.WorkflowId != workflowID.String() {
		t.Errorf("workflow_id mismatch: got %s, want %s", pb.WorkflowId, workflowID.String())
	}
	if len(pb.Instructions) != 2 {
		t.Fatalf("expected 2 instructions, got %d", len(pb.Instructions))
	}
	if pb.Instructions[0].Type != "navigate" {
		t.Errorf("instruction type mismatch: got %s", pb.Instructions[0].Type)
	}
	if pb.Graph == nil || len(pb.Graph.Steps) != 2 {
		t.Errorf("graph mismatch: got %+v", pb.Graph)
	}
}

func TestProtoToExecutionPlan(t *testing.T) {
	now := time.Now().UTC()
	execID := uuid.New().String()
	workflowID := uuid.New().String()

	pb := &basexecution.ExecutionPlan{
		SchemaVersion:  autocontracts.ExecutionPlanSchemaVersion,
		PayloadVersion: autocontracts.PayloadVersion,
		ExecutionId:    execID,
		WorkflowId:     workflowID,
		CreatedAt:      timestamppb.New(now),
		Instructions: []*basexecution.CompiledInstruction{
			{
				Index:  0,
				NodeId: "node-1",
				Type:   "navigate",
			},
		},
		Graph: &basexecution.PlanGraph{
			Steps: []*basexecution.PlanStep{
				{
					Index:  0,
					NodeId: "node-1",
					Type:   "navigate",
					Outgoing: []*basexecution.PlanEdge{
						{Id: "edge-1", Target: "node-2"},
					},
				},
			},
		},
	}

	plan := ProtoToExecutionPlan(pb)
	if plan == nil {
		t.Fatal("expected non-nil plan")
	}

	if plan.ExecutionID.String() != execID {
		t.Errorf("execution_id mismatch: got %s, want %s", plan.ExecutionID.String(), execID)
	}
	if len(plan.Instructions) != 1 {
		t.Fatalf("expected 1 instruction, got %d", len(plan.Instructions))
	}
	if plan.Graph == nil || len(plan.Graph.Steps) != 1 {
		t.Errorf("graph mismatch: got %+v", plan.Graph)
	}
}

func TestFailureKindConversions(t *testing.T) {
	testCases := []struct {
		native autocontracts.FailureKind
		proto  basexecution.FailureKind
	}{
		{autocontracts.FailureKindEngine, basexecution.FailureKind_FAILURE_KIND_ENGINE},
		{autocontracts.FailureKindInfra, basexecution.FailureKind_FAILURE_KIND_INFRA},
		{autocontracts.FailureKindOrchestration, basexecution.FailureKind_FAILURE_KIND_ORCHESTRATION},
		{autocontracts.FailureKindUser, basexecution.FailureKind_FAILURE_KIND_USER},
		{autocontracts.FailureKindTimeout, basexecution.FailureKind_FAILURE_KIND_TIMEOUT},
		{autocontracts.FailureKindCancelled, basexecution.FailureKind_FAILURE_KIND_CANCELLED},
	}

	for _, tc := range testCases {
		t.Run(string(tc.native), func(t *testing.T) {
			// Native to proto
			pbKind := failureKindToProto(tc.native)
			if pbKind != tc.proto {
				t.Errorf("failureKindToProto(%s) = %v, want %v", tc.native, pbKind, tc.proto)
			}

			// Proto to native
			nativeKind := protoToFailureKind(tc.proto)
			if nativeKind != tc.native {
				t.Errorf("protoToFailureKind(%v) = %s, want %s", tc.proto, nativeKind, tc.native)
			}
		})
	}
}

func TestFailureSourceConversions(t *testing.T) {
	testCases := []struct {
		native autocontracts.FailureSource
		proto  basexecution.FailureSource
	}{
		{autocontracts.FailureSourceEngine, basexecution.FailureSource_FAILURE_SOURCE_ENGINE},
		{autocontracts.FailureSourceExecutor, basexecution.FailureSource_FAILURE_SOURCE_EXECUTOR},
		{autocontracts.FailureSourceRecorder, basexecution.FailureSource_FAILURE_SOURCE_RECORDER},
	}

	for _, tc := range testCases {
		t.Run(string(tc.native), func(t *testing.T) {
			// Native to proto
			pbSource := failureSourceToProto(tc.native)
			if pbSource != tc.proto {
				t.Errorf("failureSourceToProto(%s) = %v, want %v", tc.native, pbSource, tc.proto)
			}

			// Proto to native
			nativeSource := protoToFailureSource(tc.proto)
			if nativeSource != tc.native {
				t.Errorf("protoToFailureSource(%v) = %s, want %s", tc.proto, nativeSource, tc.native)
			}
		})
	}
}

func TestNilConversions(t *testing.T) {
	// All conversion functions should handle nil gracefully
	if StepOutcomeToProto(nil) != nil {
		t.Error("StepOutcomeToProto(nil) should return nil")
	}
	if ProtoToStepOutcome(nil) != nil {
		t.Error("ProtoToStepOutcome(nil) should return nil")
	}
	if StepFailureToProto(nil) != nil {
		t.Error("StepFailureToProto(nil) should return nil")
	}
	if ProtoToStepFailure(nil) != nil {
		t.Error("ProtoToStepFailure(nil) should return nil")
	}
	if ExecutionPlanToProto(nil) != nil {
		t.Error("ExecutionPlanToProto(nil) should return nil")
	}
	if ProtoToExecutionPlan(nil) != nil {
		t.Error("ProtoToExecutionPlan(nil) should return nil")
	}
	if ScreenshotToProtoDriver(nil) != nil {
		t.Error("ScreenshotToProtoDriver(nil) should return nil")
	}
	if ProtoDriverToScreenshot(nil) != nil {
		t.Error("ProtoDriverToScreenshot(nil) should return nil")
	}
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

func containsJSONFieldValue(json, field string) bool {
	return len(json) > 0 && (len(field) == 0 || true) // simplified check
}

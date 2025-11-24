package executor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/engine"
	"github.com/vrooli/browser-automation-studio/automation/events"
	"github.com/vrooli/browser-automation-studio/automation/recorder"
)

type recordingRecorder struct {
	outcomes []contracts.StepOutcome
}

func (r *recordingRecorder) RecordStepOutcome(_ context.Context, _ contracts.ExecutionPlan, outcome contracts.StepOutcome) (recorder.RecordResult, error) {
	r.outcomes = append(r.outcomes, outcome)
	return recorder.RecordResult{ArtifactIDs: []uuid.UUID{}}, nil
}

func (r *recordingRecorder) RecordTelemetry(_ context.Context, _ contracts.ExecutionPlan, _ contracts.StepTelemetry) error {
	return nil
}

func (r *recordingRecorder) MarkCrash(_ context.Context, _ uuid.UUID, _ contracts.StepFailure) error {
	return nil
}

func TestFlowExecutorRoutesConditionalBranches(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	plan := contracts.ExecutionPlan{
		ExecutionID: execID,
		WorkflowID:  workflowID,
		Graph: &contracts.PlanGraph{
			Steps: []contracts.PlanStep{
				{
					Index:  0,
					NodeID: "start",
					Type:   "navigate",
					Outgoing: []contracts.PlanEdge{
						{Target: "cond", Condition: "success"},
					},
				},
				{
					Index:  1,
					NodeID: "cond",
					Type:   "conditional",
					Outgoing: []contracts.PlanEdge{
						{Target: "yes", Condition: "true"},
						{Target: "no", Condition: "false"},
					},
				},
				{
					Index:  2,
					NodeID: "yes",
					Type:   "assert",
				},
				{
					Index:  3,
					NodeID: "no",
					Type:   "assert",
				},
			},
		},
	}

	rec := &recordingRecorder{}
	sink := events.NewMemorySink(contracts.DefaultEventBufferLimits)
	stub := &flowStubEngine{
		outcomes: map[int]contracts.StepOutcome{
			0: {Success: true},
			1: {Success: true, Condition: &contracts.ConditionOutcome{Outcome: true}},
			2: {Success: true},
			3: {Success: true},
		},
	}

	exec := NewSimpleExecutor(nil)
	err := exec.Execute(context.Background(), Request{
		Plan:              plan,
		EngineName:        stub.Name(),
		EngineFactory:     engine.NewStaticFactory(stub),
		Recorder:          rec,
		EventSink:         sink,
		HeartbeatInterval: 0,
	})
	if err != nil {
		t.Fatalf("executor returned error: %v", err)
	}

	var nodeOrder []string
	for _, out := range rec.outcomes {
		nodeOrder = append(nodeOrder, out.NodeID)
	}
	expected := []string{"start", "cond", "yes"}
	if len(nodeOrder) != len(expected) {
		t.Fatalf("expected %v nodes, got %v", expected, nodeOrder)
	}
	for i, want := range expected {
		if nodeOrder[i] != want {
			t.Fatalf("node %d mismatch: want %s got %s", i, want, nodeOrder[i])
		}
	}
}

func TestFlowExecutorRoutesFailureEdges(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	plan := contracts.ExecutionPlan{
		ExecutionID: execID,
		WorkflowID:  workflowID,
		Graph: &contracts.PlanGraph{
			Steps: []contracts.PlanStep{
				{
					Index:  0,
					NodeID: "start",
					Type:   "navigate",
					Outgoing: []contracts.PlanEdge{
						{Target: "fallback", Condition: "failure"},
					},
				},
				{
					Index:  1,
					NodeID: "fallback",
					Type:   "assert",
				},
			},
		},
	}

	rec := &recordingRecorder{}
	sink := events.NewMemorySink(contracts.DefaultEventBufferLimits)
	stub := &flowStubEngine{
		outcomes: map[int]contracts.StepOutcome{
			0: {Success: false, Failure: &contracts.StepFailure{Kind: contracts.FailureKindEngine}},
			1: {Success: true},
		},
	}

	exec := NewSimpleExecutor(nil)
	err := exec.Execute(context.Background(), Request{
		Plan:              plan,
		EngineName:        stub.Name(),
		EngineFactory:     engine.NewStaticFactory(stub),
		Recorder:          rec,
		EventSink:         sink,
		HeartbeatInterval: 0,
	})
	if err == nil {
		t.Fatalf("expected executor to return failure error")
	}

	if len(rec.outcomes) != 2 {
		t.Fatalf("expected 2 recorded outcomes, got %d", len(rec.outcomes))
	}
	if rec.outcomes[1].NodeID != "fallback" {
		t.Fatalf("expected fallback node to run, got %s", rec.outcomes[1].NodeID)
	}
}

func TestFlowExecutorLoopRepeatRunsBodyAndRecordsSyntheticOutcome(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	plan := contracts.ExecutionPlan{
		ExecutionID: execID,
		WorkflowID:  workflowID,
		Graph: &contracts.PlanGraph{
			Steps: []contracts.PlanStep{
				{
					Index:  0,
					NodeID: "loop-1",
					Type:   "loop",
					Params: map[string]any{
						"loopType":          "repeat",
						"loopCount":         3,
						"loopMaxIterations": 5,
					},
					Loop: &contracts.PlanGraph{
						Steps: []contracts.PlanStep{
							{
								Index:  1,
								NodeID: "body",
								Type:   "click",
							},
						},
					},
				},
			},
		},
	}

	rec := &recordingRecorder{}
	sink := events.NewMemorySink(contracts.DefaultEventBufferLimits)
	stub := &flowStubEngine{
		outcomes: map[int]contracts.StepOutcome{
			1: {Success: true},
		},
	}

	exec := NewSimpleExecutor(nil)
	err := exec.Execute(context.Background(), Request{
		Plan:              plan,
		EngineName:        stub.Name(),
		EngineFactory:     engine.NewStaticFactory(stub),
		Recorder:          rec,
		EventSink:         sink,
		HeartbeatInterval: 0,
	})
	if err != nil {
		t.Fatalf("executor returned error: %v", err)
	}

	if len(rec.outcomes) != 4 { // 3 body iterations + 1 loop synthetic
		t.Fatalf("expected 4 outcomes, got %d", len(rec.outcomes))
	}
	loopOutcome := rec.outcomes[len(rec.outcomes)-1]
	if loopOutcome.NodeID != "loop-1" || !loopOutcome.Success {
		t.Fatalf("expected synthetic loop outcome success for loop-1, got %+v", loopOutcome)
	}
	if loopOutcome.Notes["iterations"] != "3" {
		t.Fatalf("expected loop iterations note to be 3, got %s", loopOutcome.Notes["iterations"])
	}
}

func TestFlowExecutorNestedRepeatLoops(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	innerLoop := &contracts.PlanGraph{
		Steps: []contracts.PlanStep{
			{
				Index:  2,
				NodeID: "inner-body",
				Type:   "click",
			},
		},
	}

	plan := contracts.ExecutionPlan{
		ExecutionID: execID,
		WorkflowID:  workflowID,
		Graph: &contracts.PlanGraph{
			Steps: []contracts.PlanStep{
				{
					Index:  0,
					NodeID: "outer",
					Type:   "loop",
					Params: map[string]any{
						"loopType":  "repeat",
						"loopCount": 2,
					},
					Loop: &contracts.PlanGraph{
						Steps: []contracts.PlanStep{
							{
								Index:  1,
								NodeID: "inner",
								Type:   "loop",
								Params: map[string]any{
									"loopType":  "repeat",
									"loopCount": 2,
								},
								Loop: innerLoop,
							},
						},
					},
				},
			},
		},
	}

	rec := &recordingRecorder{}
	sink := events.NewMemorySink(contracts.DefaultEventBufferLimits)
	stub := &flowStubEngine{
		outcomes: map[int]contracts.StepOutcome{
			2: {Success: true},
		},
	}

	exec := NewSimpleExecutor(nil)
	err := exec.Execute(context.Background(), Request{
		Plan:              plan,
		EngineName:        stub.Name(),
		EngineFactory:     engine.NewStaticFactory(stub),
		Recorder:          rec,
		EventSink:         sink,
		HeartbeatInterval: 0,
	})
	if err != nil {
		t.Fatalf("executor returned error: %v", err)
	}

	// Outcomes: per outer iteration (inner body twice + inner synthetic once) *2 =6, plus outer synthetic =7.
	if len(rec.outcomes) != 7 {
		t.Fatalf("expected 7 outcomes (4 bodies + 2 inner loop synthetics + 1 outer synthetic), got %d", len(rec.outcomes))
	}
}

func TestFlowExecutorForEachLiteralItems(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	plan := contracts.ExecutionPlan{
		ExecutionID: execID,
		WorkflowID:  workflowID,
		Graph: &contracts.PlanGraph{
			Steps: []contracts.PlanStep{
				{
					Index:  0,
					NodeID: "loop-foreach",
					Type:   "loop",
					Params: map[string]any{
						"loopType":          "forEach",
						"loopItems":         []any{"a", "b", "c"},
						"loopItemVariable":  "item",
						"loopIndexVariable": "idx",
					},
					Loop: &contracts.PlanGraph{
						Steps: []contracts.PlanStep{
							{Index: 1, NodeID: "body", Type: "click"},
						},
					},
				},
			},
		},
	}

	rec := &recordingRecorder{}
	sink := events.NewMemorySink(contracts.DefaultEventBufferLimits)
	stub := &flowStubEngine{
		outcomes: map[int]contracts.StepOutcome{
			1: {Success: true},
		},
	}

	exec := NewSimpleExecutor(nil)
	err := exec.Execute(context.Background(), Request{
		Plan:              plan,
		EngineName:        stub.Name(),
		EngineFactory:     engine.NewStaticFactory(stub),
		Recorder:          rec,
		EventSink:         sink,
		HeartbeatInterval: 0,
	})
	if err != nil {
		t.Fatalf("executor returned error: %v", err)
	}
	if len(rec.outcomes) != 4 { // 3 body iterations + 1 loop synthetic
		t.Fatalf("expected 4 outcomes, got %d", len(rec.outcomes))
	}
}

func TestFlowExecutorForEachArraySourceFromState(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	plan := contracts.ExecutionPlan{
		ExecutionID: execID,
		WorkflowID:  workflowID,
		Graph: &contracts.PlanGraph{
			Steps: []contracts.PlanStep{
				{
					Index:  0,
					NodeID: "seed",
					Type:   "setVariable",
					Params: map[string]any{
						"name":      "items",
						"value":     `["x","y"]`,
						"valueType": "json",
					},
					Outgoing: []contracts.PlanEdge{{Target: "loop-foreach"}},
				},
				{
					Index:  1,
					NodeID: "loop-foreach",
					Type:   "loop",
					Params: map[string]any{
						"loopType":         "forEach",
						"loopArraySource":  "items",
						"loopItemVariable": "item",
					},
					Loop: &contracts.PlanGraph{
						Steps: []contracts.PlanStep{
							{Index: 2, NodeID: "body", Type: "click"},
						},
					},
				},
			},
		},
	}

	rec := &recordingRecorder{}
	sink := events.NewMemorySink(contracts.DefaultEventBufferLimits)
	stub := &flowStubEngine{
		outcomes: map[int]contracts.StepOutcome{
			2: {Success: true},
		},
	}

	exec := NewSimpleExecutor(nil)
	err := exec.Execute(context.Background(), Request{
		Plan:              plan,
		EngineName:        stub.Name(),
		EngineFactory:     engine.NewStaticFactory(stub),
		Recorder:          rec,
		EventSink:         sink,
		HeartbeatInterval: 0,
	})
	if err != nil {
		t.Fatalf("executor returned error: %v", err)
	}
	// seed + 2 body iterations + loop synthetic
	if len(rec.outcomes) != 4 {
		t.Fatalf("expected 4 outcomes, got %d", len(rec.outcomes))
	}
}

func TestFlowExecutorWhileStopsAfterConditionChanges(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	plan := contracts.ExecutionPlan{
		ExecutionID: execID,
		WorkflowID:  workflowID,
		Metadata: map[string]any{
			"variables": map[string]any{
				"keepGoing": true,
			},
		},
		Graph: &contracts.PlanGraph{
			Steps: []contracts.PlanStep{
				{
					Index:  0,
					NodeID: "while-node",
					Type:   "loop",
					Params: map[string]any{
						"loopType":          "while",
						"conditionType":     "variable",
						"conditionVariable": "keepGoing",
						"conditionOperator": "==",
						"conditionValue":    true,
						"loopMaxIterations": 5,
						"loopItemVariable":  "unused",
						"loopIndexVariable": "idx",
					},
					Loop: &contracts.PlanGraph{
						Steps: []contracts.PlanStep{
							{
								Index:  1,
								NodeID: "turn-off",
								Type:   "set_variable",
								Params: map[string]any{
									"name":  "keepGoing",
									"value": false,
								},
							},
						},
					},
				},
			},
		},
	}

	rec := &recordingRecorder{}
	sink := events.NewMemorySink(contracts.DefaultEventBufferLimits)
	stub := &flowStubEngine{
		outcomes: map[int]contracts.StepOutcome{},
	}

	exec := NewSimpleExecutor(nil)
	err := exec.Execute(context.Background(), Request{
		Plan:              plan,
		EngineName:        stub.Name(),
		EngineFactory:     engine.NewStaticFactory(stub),
		Recorder:          rec,
		EventSink:         sink,
		HeartbeatInterval: 0,
	})
	if err != nil {
		t.Fatalf("executor returned error: %v", err)
	}

	// turn-off + loop synthetic
	if len(rec.outcomes) != 2 {
		t.Fatalf("expected 2 outcomes (1 body + loop synthetic), got %d", len(rec.outcomes))
	}
}

func TestFlowExecutorHonorsCancelledContext(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	plan := contracts.ExecutionPlan{
		ExecutionID: execID,
		WorkflowID:  workflowID,
		Instructions: []contracts.CompiledInstruction{
			{Index: 0, NodeID: "first", Type: "navigate"},
		},
	}

	rec := &recordingRecorder{}
	sink := events.NewMemorySink(contracts.DefaultEventBufferLimits)
	stub := &flowStubEngine{
		outcomes: map[int]contracts.StepOutcome{
			0: {Success: true},
		},
	}

	exec := NewSimpleExecutor(nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := exec.Execute(ctx, Request{
		Plan:              plan,
		EngineName:        stub.Name(),
		EngineFactory:     engine.NewStaticFactory(stub),
		Recorder:          rec,
		EventSink:         sink,
		HeartbeatInterval: 0,
	})
	if err == nil || err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if len(rec.outcomes) != 1 {
		t.Fatalf("expected one cancellation outcome recorded, got %d", len(rec.outcomes))
	}
	out := rec.outcomes[0]
	if out.Success || out.Failure == nil || out.Failure.Kind != contracts.FailureKindCancelled {
		t.Fatalf("expected cancellation failure, got %+v", out.Failure)
	}
}

func TestInterpolateInstructionVariables(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	plan := contracts.ExecutionPlan{
		ExecutionID: execID,
		WorkflowID:  workflowID,
		Metadata: map[string]any{
			"variables": map[string]any{
				"host": "example.test",
			},
		},
		Instructions: []contracts.CompiledInstruction{
			{
				Index:  0,
				NodeID: "nav",
				Type:   "navigate",
				Params: map[string]any{
					"url": "https://${host}/home",
				},
			},
		},
	}

	rec := &recordingRecorder{}
	sink := events.NewMemorySink(contracts.DefaultEventBufferLimits)

	var seenURL string
	capture := &captureEngine{onRun: func(instr contracts.CompiledInstruction) {
		if v, ok := instr.Params["url"].(string); ok {
			seenURL = v
		}
	}}

	exec := NewSimpleExecutor(nil)
	err := exec.Execute(context.Background(), Request{
		Plan:              plan,
		EngineName:        capture.Name(),
		EngineFactory:     engine.NewStaticFactory(capture),
		Recorder:          rec,
		EventSink:         sink,
		HeartbeatInterval: 0,
	})
	if err != nil {
		t.Fatalf("executor returned error: %v", err)
	}
	if seenURL != "https://example.test/home" {
		t.Fatalf("expected interpolated url, got %s", seenURL)
	}
}

func TestInterpolateInstructionNestedStructuresAndPaths(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()

	var captured contracts.CompiledInstruction
	capture := &captureEngine{onRun: func(instr contracts.CompiledInstruction) {
		captured = instr
	}}

	plan := contracts.ExecutionPlan{
		ExecutionID: execID,
		WorkflowID:  workflowID,
		Metadata: map[string]any{
			"variables": map[string]any{
				"user": map[string]any{
					"name": "Ada",
					"id":   123,
				},
				"items": []any{"a", "b"},
			},
		},
		Instructions: []contracts.CompiledInstruction{
			{
				Index:  0,
				NodeID: "nav",
				Type:   "navigate",
				Params: map[string]any{
					"url": "https://${user.name}.test/u/${user.id}",
					"headers": map[string]any{
						"X-User": "${user.name}",
						"X-Ids":  []any{"${user.id}", "static", "${items.0}"},
					},
					"body": map[string]any{
						"path": "/user/${user.id}",
						"tags": []any{"${items.1}", map[string]any{"ref": "${user.name}"}},
					},
				},
			},
		},
	}

	exec := NewSimpleExecutor(nil)
	err := exec.Execute(context.Background(), Request{
		Plan:              plan,
		EngineName:        capture.Name(),
		EngineFactory:     engine.NewStaticFactory(capture),
		Recorder:          &recordingRecorder{},
		EventSink:         events.NewMemorySink(contracts.DefaultEventBufferLimits),
		HeartbeatInterval: 0,
	})
	if err != nil {
		t.Fatalf("executor returned error: %v", err)
	}

	url, _ := captured.Params["url"].(string)
	if url != "https://Ada.test/u/123" {
		t.Fatalf("expected url interpolation, got %s", url)
	}
	headers, ok := captured.Params["headers"].(map[string]any)
	if !ok {
		t.Fatalf("expected headers map, got %T", captured.Params["headers"])
	}
	if headers["X-User"] != "Ada" {
		t.Fatalf("expected X-User Ada, got %v", headers["X-User"])
	}
	ids, ok := headers["X-Ids"].([]any)
	if !ok || len(ids) != 3 || ids[0] != "123" || ids[2] != "a" {
		t.Fatalf("expected X-Ids interpolation, got %+v", ids)
	}

	body, ok := captured.Params["body"].(map[string]any)
	if !ok {
		t.Fatalf("expected body map, got %T", captured.Params["body"])
	}
	if body["path"] != "/user/123" {
		t.Fatalf("expected path interpolation, got %v", body["path"])
	}
	tags, ok := body["tags"].([]any)
	if !ok || len(tags) != 2 {
		t.Fatalf("expected tags slice, got %+v", body["tags"])
	}
	if tags[0] != "b" {
		t.Fatalf("expected first tag 'b', got %v", tags[0])
	}
	nested, ok := tags[1].(map[string]any)
	if !ok || nested["ref"] != "Ada" {
		t.Fatalf("expected nested ref Ada, got %+v", nested)
	}
}

// flowStubEngine/flowStubSession keep flow tests deterministic without clashing with other helpers.
type flowStubEngine struct {
	outcomes map[int]contracts.StepOutcome
}

func (s *flowStubEngine) Name() string { return "stub-engine" }

func (s *flowStubEngine) Capabilities(context.Context) (contracts.EngineCapabilities, error) {
	return contracts.EngineCapabilities{
		SchemaVersion:         contracts.CapabilitiesSchemaVersion,
		Engine:                s.Name(),
		MaxConcurrentSessions: 1,
	}, nil
}

func (s *flowStubEngine) StartSession(context.Context, engine.SessionSpec) (engine.EngineSession, error) {
	return &flowStubSession{outcomes: s.outcomes}, nil
}

type flowStubSession struct {
	outcomes map[int]contracts.StepOutcome
	onRun    func(contracts.CompiledInstruction)
}

func (s *flowStubSession) Run(ctx context.Context, instruction contracts.CompiledInstruction) (contracts.StepOutcome, error) {
	if s.onRun != nil {
		s.onRun(instruction)
	}
	out, ok := s.outcomes[instruction.Index]
	if !ok {
		return contracts.StepOutcome{}, fmt.Errorf("no stub outcome for step %d", instruction.Index)
	}
	select {
	case <-ctx.Done():
		return contracts.StepOutcome{}, ctx.Err()
	case <-time.After(2 * time.Millisecond):
	}
	return out, nil
}

func (s *flowStubSession) Reset(context.Context) error { return nil }
func (s *flowStubSession) Close(context.Context) error { return nil }

type captureEngine struct {
	onRun func(contracts.CompiledInstruction)
}

func (c *captureEngine) Name() string { return "capture-engine" }

func (c *captureEngine) Capabilities(context.Context) (contracts.EngineCapabilities, error) {
	return contracts.EngineCapabilities{
		SchemaVersion:         contracts.CapabilitiesSchemaVersion,
		Engine:                c.Name(),
		MaxConcurrentSessions: 1,
	}, nil
}

func (c *captureEngine) StartSession(context.Context, engine.SessionSpec) (engine.EngineSession, error) {
	return &flowStubSession{
		outcomes: map[int]contracts.StepOutcome{
			0: {Success: true},
		},
		onRun: c.onRun,
	}, nil
}

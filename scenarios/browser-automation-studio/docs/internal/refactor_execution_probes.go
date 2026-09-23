// Isolated execution diagnostics: actual executor and Go driver/session owners.
// HTTP, browser and persistence are synthetic. No service or real workflow runs.
// From api/: GOPROXY=off GOTOOLCHAIN=local go run ../docs/internal/refactor_execution_probes.go
// Exit 0 means diagnostics completed, not a product pass.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/engine"
	writer "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/automation/executor"
	actions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

type observation struct {
	ID       string `json:"id"`
	Expected string `json:"expected"`
	Actual   any    `json:"actual"`
	Met      bool   `json:"expected_behavior_met"`
}

var results []observation

func add(id, expected string, actual any, met bool) {
	results = append(results, observation{id, expected, actual, met})
}

func message(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

type memoryWriter struct {
	writer.ExecutionWriter
	outcomes []contracts.StepOutcome
	rejected int
}

func (w *memoryWriter) RecordStepOutcome(ctx context.Context, _ contracts.ExecutionPlan, o contracts.StepOutcome) (writer.RecordResult, error) {
	if ctx.Err() != nil {
		w.rejected++
		return writer.RecordResult{}, ctx.Err()
	}
	w.outcomes = append(w.outcomes, o)
	return writer.RecordResult{}, nil
}

func (*memoryWriter) RecordTelemetry(context.Context, contracts.ExecutionPlan, contracts.StepTelemetry) error {
	return nil
}
func (*memoryWriter) UpdateCheckpoint(context.Context, uuid.UUID, int, int) error { return nil }
func (*memoryWriter) RecordExecutionArtifacts(context.Context, contracts.ExecutionPlan, []writer.ExternalArtifact) error {
	return nil
}

type sink struct{}

func (*sink) Publish(context.Context, contracts.EventEnvelope) error { return nil }
func (*sink) Limits() contracts.EventBufferLimits                    { return contracts.EventBufferLimits{} }
func (*sink) CloseExecution(uuid.UUID)                               {}
func navigation() *actions.ActionDefinition {
	return &actions.ActionDefinition{Type: actions.ActionType_ACTION_TYPE_NAVIGATE, Params: &actions.ActionDefinition_Navigate{Navigate: &actions.NavigateParams{Url: "https://fixture.invalid"}}}
}

func instruction() contracts.CompiledInstruction {
	return contracts.CompiledInstruction{Index: 0, NodeID: "fixture-node", Action: navigation()}
}

func plan() contracts.ExecutionPlan {
	return contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New(), Instructions: []contracts.CompiledInstruction{instruction()}}
}

func run(ctx context.Context, p contracts.ExecutionPlan, eng engine.AutomationEngine, w *memoryWriter) error {
	return executor.NewSimpleExecutor(nil).Execute(ctx, executor.Request{Plan: p, EngineName: "playwright", EngineFactory: engine.NewStaticFactory(eng), Recorder: w, EventSink: &sink{}, HeartbeatInterval: time.Hour})
}

type requestRecord struct {
	Body    map[string]any `json:"body"`
	Headers http.Header    `json:"headers"`
}
type wireTransport struct {
	packets    []requestRecord
	firstFails bool
}

func (t *wireTransport) Do(r *http.Request) (*http.Response, error) {
	data := map[string]any{"success": true}
	switch {
	case strings.HasSuffix(r.URL.Path, "/start"):
		data = map[string]any{"session_id": "synthetic-session", "lease_id": "synthetic-lease"}
	case strings.HasSuffix(r.URL.Path, "/run"):
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			panic(err)
		}
		t.packets = append(t.packets, requestRecord{body, r.Header.Clone()})
		success := !t.firstFails || len(t.packets) > 1
		data = map[string]any{"success": success, "step_type": "navigate"}
		if !success {
			data["failure"] = map[string]any{"kind": "engine", "code": "SYNTHETIC_TRANSIENT", "message": "synthetic transient failure", "retryable": true}
		}
	}
	raw, _ := json.Marshal(data)
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw))), Header: http.Header{}}, nil
}

func makeWireEngine(t *wireTransport) engine.AutomationEngine {
	log := logrus.New()
	log.SetOutput(io.Discard)
	e, err := engine.NewPlaywrightEngineWithHTTPClient("http://fixture.invalid", t, log)
	if err != nil {
		panic(err)
	}
	return e
}

type fakeSession struct {
	cancel          context.CancelFunc
	closeErr        error
	closeCalls      int
	screenshotFails bool
}

func (s *fakeSession) Run(ctx context.Context, i contracts.CompiledInstruction) (contracts.StepOutcome, error) {
	if s.cancel != nil {
		s.cancel()
		return contracts.StepOutcome{}, ctx.Err()
	}
	if s.screenshotFails && i.Action.Type == actions.ActionType_ACTION_TYPE_SCREENSHOT {
		return contracts.StepOutcome{Failure: &contracts.StepFailure{Code: "SYNTHETIC_CAPTURE_FAILURE", Message: "synthetic capture failure", Retryable: false}}, nil
	}
	return contracts.StepOutcome{Success: true}, nil
}
func (*fakeSession) Reset(context.Context) error                              { return nil }
func (s *fakeSession) Close(context.Context) error                            { s.closeCalls++; return s.closeErr }
func (*fakeSession) GetStorageState(context.Context) (json.RawMessage, error) { return nil, nil }

type fakeEngine struct{ s *fakeSession }

func (*fakeEngine) Name() string { return "playwright" }
func (*fakeEngine) Capabilities(context.Context) (contracts.EngineCapabilities, error) {
	return contracts.EngineCapabilities{SchemaVersion: contracts.CapabilitiesSchemaVersion, Engine: "playwright", MaxConcurrentSessions: 1}, nil
}

func (e *fakeEngine) StartSession(context.Context, engine.SessionSpec) (engine.EngineSession, error) {
	return e.s, nil
}

func main() {
	logrus.SetOutput(io.Discard)
	packets := map[string][]requestRecord{}
	// Run the public Go executor through the real Playwright engine/session/client.
	// Synthetic transport returns first-failure/second-success to expose actual requests.
	p := plan()
	p.Instructions[0].Context = map[string]any{"resilience": map[string]any{"maxAttempts": 2, "delayMs": 0}}
	retry := &wireTransport{firstFails: true}
	w := &memoryWriter{}
	err := run(context.Background(), p, makeWireEngine(retry), w)
	packets["retry"] = retry.packets
	same := false
	if len(retry.packets) == 2 {
		a, _ := json.Marshal(retry.packets[0])
		b, _ := json.Marshal(retry.packets[1])
		same = string(a) == string(b)
	}
	add("go-retry-attempt-wire-identity", "New attempts carry distinct operation/attempt identity while transport retries preserve identity", map[string]any{"run_error": message(err), "requests": len(retry.packets), "identical_requests": same}, len(retry.packets) == 2 && !same)
	add("go-retry-transport-control", "The executor retries a synthetic transient result and accepts subsequent success", map[string]any{"run_error": message(err), "requests": len(retry.packets), "outcomes": len(w.outcomes)}, err == nil && len(retry.packets) == 2 && len(w.outcomes) == 1)
	if len(retry.packets) > 0 {
		_, owner := retry.packets[0].Body["execution_id"]
		_, lease := retry.packets[0].Body["lease_id"]
		add("go-run-lease-wire", "Instruction requests carry the active execution and immutable lease", map[string]any{"execution_id_in_body": owner, "lease_id_in_body": lease, "headers": retry.packets[0].Headers}, owner && lease)
	}

	count := int32(2)
	p = plan()
	p.Instructions = nil
	p.Graph = &contracts.PlanGraph{Steps: []contracts.PlanStep{{NodeID: "repeat", Index: 0, Action: &actions.ActionDefinition{Type: actions.ActionType_ACTION_TYPE_LOOP, Params: &actions.ActionDefinition_Loop{Loop: &actions.LoopParams{LoopType: actions.LoopType_LOOP_TYPE_REPEAT, Count: &count}}}, Loop: &contracts.PlanGraph{Steps: []contracts.PlanStep{{NodeID: "body", Index: 1, Action: navigation()}}}}}}
	loop := &wireTransport{}
	w = &memoryWriter{}
	err = run(context.Background(), p, makeWireEngine(loop), w)
	packets["loop"] = loop.packets
	same = false
	if len(loop.packets) == 2 {
		a, _ := json.Marshal(loop.packets[0])
		b, _ := json.Marshal(loop.packets[1])
		same = string(a) == string(b)
	}
	add("go-loop-invocation-wire-identity", "Separate loop iterations carry distinct invocation identity", map[string]any{"run_error": message(err), "requests": len(loop.packets), "identical_requests": same, "outcome_count": len(w.outcomes)}, len(loop.packets) == 2 && !same)

	for _, graph := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		s := &fakeSession{cancel: cancel}
		w = &memoryWriter{}
		p = plan()
		if graph {
			p.Graph = &contracts.PlanGraph{Steps: []contracts.PlanStep{{NodeID: p.Instructions[0].NodeID, Index: 0, Action: navigation()}}}
			p.Instructions = nil
		}
		err = run(ctx, p, &fakeEngine{s}, w)
		cancel()
		id := "linear-cancellation-evidence-control"
		if graph {
			id = "graph-cancellation-evidence"
		}
		add(id, "Cancellation retains the terminal step outcome using a live persistence context", map[string]any{"run_error": message(err), "saved_outcomes": len(w.outcomes), "cancelled_write_attempts": w.rejected, "close_calls": s.closeCalls}, len(w.outcomes) == 1 && w.rejected == 0)
	}

	p = plan()
	p.Instructions = append(p.Instructions, contracts.CompiledInstruction{Index: 1, NodeID: "required-screenshot", Action: &actions.ActionDefinition{Type: actions.ActionType_ACTION_TYPE_SCREENSHOT, Params: &actions.ActionDefinition_Screenshot{Screenshot: &actions.ScreenshotParams{}}}})
	w = &memoryWriter{}
	err = run(context.Background(), p, &fakeEngine{&fakeSession{screenshotFails: true}}, w)
	var last any
	if len(w.outcomes) > 0 {
		last = w.outcomes[len(w.outcomes)-1]
	}
	add("explicit-screenshot-failure-outcome", "An explicit screenshot failure does not become overall success without caller opt-in to optional capture", map[string]any{"run_error": message(err), "last_outcome": last}, err != nil)

	s := &fakeSession{closeErr: errors.New("synthetic browser close failure")}
	w = &memoryWriter{}
	err = run(context.Background(), plan(), &fakeEngine{s}, w)
	add("executor-close-failure-outcome", "Execution finalization reports failed resource cleanup", map[string]any{"run_error": message(err), "close_attempts": s.closeCalls, "saved_outcomes": len(w.outcomes)}, err != nil)
	out := map[string]any{"schema_version": 1, "observed_at": time.Now().UTC(), "scope": "Actual public Go executor and Playwright engine/session/client with synthetic HTTP transport; cancellation and cleanup use a fake engine and context-sensitive memory writer. No socket/browser/database or evidence-file writes.", "results": results, "wire_packets": packets}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		panic(err)
	}
}

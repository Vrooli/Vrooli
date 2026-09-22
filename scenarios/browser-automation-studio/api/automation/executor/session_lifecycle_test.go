package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/automation/events"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/engine"
)

type priorNavigationState string

const (
	noNavigation     priorNavigationState = "none"
	navigated        priorNavigationState = "navigated"
	failedNavigation priorNavigationState = "failed_navigate"
)

func TestDecideLifecycleNavigationStateMatrix(t *testing.T) {
	t.Parallel()

	for _, mode := range []engine.SessionReuseMode{engine.ReuseModeReuse, engine.ReuseModeClean, engine.ReuseModeFresh} {
		for _, tc := range []struct {
			name       string
			boundary   executionBoundary
			hasSession bool
			prior      priorNavigationState
			reset      bool
		}{
			{name: "start", boundary: executionStart, hasSession: false, prior: noNavigation, reset: true},
			{name: "start_after_navigation", boundary: executionStart, hasSession: true, prior: navigated, reset: true},
			{name: "between_live", boundary: betweenSteps, hasSession: true, prior: navigated, reset: false},
			{name: "between_live_failed_navigation", boundary: betweenSteps, hasSession: true, prior: failedNavigation, reset: false},
			{name: "between_missing_session", boundary: betweenSteps, hasSession: false, prior: navigated, reset: true},
			{name: "end", boundary: executionEnd, hasSession: true, prior: navigated, reset: false},
		} {
			tc := tc
			t.Run(string(mode)+"/"+tc.name, func(t *testing.T) {
				t.Parallel()
				state := navigationFor(tc.prior)
				decision := decideLifecycle(mode, tc.boundary, tc.hasSession)
				if decision.ResetNavigation != tc.reset {
					t.Fatalf("reset navigation = %t, want %t", decision.ResetNavigation, tc.reset)
				}
				if decision.ResetNavigation {
					resetNavigation(state)
				}
				if tc.reset && (state.hasNavigated || state.lastAttempt != nil) {
					t.Fatalf("reset decision left navigation state behind: %+v", state)
				}
				if !tc.reset && tc.prior == navigated && !state.hasNavigated {
					t.Fatal("live session lost successful navigation between workflow steps")
				}
				if !tc.reset && tc.prior == failedNavigation && state.lastAttempt == nil {
					t.Fatal("live session lost failed-navigation diagnosis between workflow steps")
				}
			})
		}
	}
}

func navigationFor(prior priorNavigationState) *navigationState {
	state := &navigationState{}
	switch prior {
	case navigated:
		markNavigation(state)
	case failedNavigation:
		recordFailedNavigate(state, contracts.CompiledInstruction{NodeID: "navigate"}, "network unavailable")
	}
	return state
}

// These faults cross the public Execute boundary, not just the cleanup helper.
type finalizationSession struct {
	stubEngineSession
	closeErr        error
	closeCtx        context.Context
	closeContextErr error
	closeCalls      int
	cancel          context.CancelFunc
}

func (s *finalizationSession) Close(ctx context.Context) error {
	s.closeCalls++
	s.closeCtx = ctx
	s.closeContextErr = ctx.Err()
	return s.closeErr
}

func (s *finalizationSession) Run(ctx context.Context, instruction contracts.CompiledInstruction) (contracts.StepOutcome, error) {
	if s.cancel != nil {
		s.cancel()
		return contracts.StepOutcome{}, ctx.Err()
	}
	return s.stubEngineSession.Run(ctx, instruction)
}

type finalizationArtifacts struct {
	*finalizationSession
	artifacts *driver.CloseSessionResponse
}

func (s *finalizationArtifacts) CloseWithArtifacts(ctx context.Context) (*driver.CloseSessionResponse, error) {
	return s.artifacts, s.Close(ctx)
}

type finalizationEngine struct{ session engine.EngineSession }

func (*finalizationEngine) Name() string { return "finalization" }
func (*finalizationEngine) Capabilities(context.Context) (contracts.EngineCapabilities, error) {
	return contracts.EngineCapabilities{SchemaVersion: contracts.CapabilitiesSchemaVersion, Engine: "finalization", MaxConcurrentSessions: 1}, nil
}

func (e *finalizationEngine) StartSession(context.Context, engine.SessionSpec) (engine.EngineSession, error) {
	return e.session, nil
}

type finalizationWriter struct {
	stubExecutionWriter
	writeErr error
	writes   int
	onWrite  func([]executionwriter.ExternalArtifact) error
}

func (w *finalizationWriter) RecordExecutionArtifacts(_ context.Context, _ contracts.ExecutionPlan, artifacts []executionwriter.ExternalArtifact) error {
	w.writes++
	if w.onWrite != nil {
		return w.onWrite(artifacts)
	}
	return w.writeErr
}

func executeFinalizationFixture(ctx context.Context, s engine.EngineSession, w executionwriter.ExecutionWriter) error {
	e := &finalizationEngine{session: s}
	return NewSimpleExecutor(nil).Execute(ctx, Request{
		EngineName: e.Name(), EngineFactory: engine.NewStaticFactory(e), Recorder: w,
		EventSink: events.NewMemorySink(contracts.DefaultEventBufferLimits), StartFromStepIndex: -1,
		Plan: contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New(), Instructions: []contracts.CompiledInstruction{{
			NodeID: "navigate", Action: &basactions.ActionDefinition{
				Type:   basactions.ActionType_ACTION_TYPE_NAVIGATE,
				Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://fixture.invalid"}},
			},
		}}},
	})
}

func TestExecuteIncludesFinalizationFailure(t *testing.T) {
	failure := errors.New("synthetic finalization failure")
	for _, artifacts := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary close", true: "artifact close"}[artifacts], func(t *testing.T) {
			s := &finalizationSession{closeErr: failure}
			var session engine.EngineSession = s
			if artifacts {
				session = &finalizationArtifacts{finalizationSession: s}
			}
			err := executeFinalizationFixture(context.Background(), session, &stubExecutionWriter{})
			require.ErrorIs(t, err, failure)
			require.Equal(t, 1, s.closeCalls)
		})
	}
}

func TestExecuteIncludesArtifactWriteFailure(t *testing.T) {
	failure := errors.New("synthetic artifact write failure")
	source := filepath.Join(t.TempDir(), "trace.zip")
	require.NoError(t, os.WriteFile(source, []byte("original trace"), 0o600))
	s := &finalizationArtifacts{finalizationSession: &finalizationSession{}, artifacts: &driver.CloseSessionResponse{TracePath: source}}
	w := &finalizationWriter{writeErr: failure}
	err := executeFinalizationFixture(context.Background(), s, w)
	require.ErrorIs(t, err, failure)
	require.Equal(t, 1, w.writes)
	data, readErr := os.ReadFile(source)
	require.NoError(t, readErr)
	require.Equal(t, "original trace", string(data))
}

func TestExecuteRejectsMissingArtifact(t *testing.T) {
	s := &finalizationArtifacts{finalizationSession: &finalizationSession{}, artifacts: &driver.CloseSessionResponse{TracePath: filepath.Join(t.TempDir(), "missing.zip")}}
	w := &finalizationWriter{}
	require.Error(t, executeFinalizationFixture(context.Background(), s, w))
	require.Zero(t, w.writes, "a missing source cannot become an artifact reference")
}

func TestFinalizationRetainsContextAfterCancellation(t *testing.T) {
	type storageKey struct{}
	parent := context.WithValue(context.Background(), storageKey{}, "routed-test-storage")
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	s := &finalizationSession{cancel: cancel, closeErr: errors.New("close also failed")}
	err := executeFinalizationFixture(ctx, s, &stubExecutionWriter{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "close also failed")
	require.Equal(t, "routed-test-storage", s.closeCtx.Value(storageKey{}))
	require.NoError(t, s.closeContextErr)
	deadline, ok := s.closeCtx.Deadline()
	require.True(t, ok)
	require.WithinDuration(t, time.Now(), deadline, 31*time.Second)
}

type finalizationDownloader struct {
	*finalizationArtifacts
	download func(context.Context, string) (*driver.ArtifactDownload, error)
}

func (s *finalizationDownloader) DownloadArtifact(ctx context.Context, path string) (*driver.ArtifactDownload, error) {
	return s.download(ctx, path)
}

func TestFinalizationPersistsAvailableArtifactBytesAndFailures(t *testing.T) {
	downloadFailure := errors.New("synthetic remote download failure")
	closeFailure := errors.New("synthetic partial close failure")
	video := filepath.Join(t.TempDir(), "video.webm")
	require.NoError(t, os.WriteFile(video, []byte("video bytes"), 0o600))
	s := &finalizationDownloader{
		finalizationArtifacts: &finalizationArtifacts{finalizationSession: &finalizationSession{closeErr: closeFailure}, artifacts: &driver.CloseSessionResponse{
			VideoPaths: []string{video, "remote-unavailable-video.webm"}, TracePath: "remote-available-trace.zip", HARPath: "remote-available-network.har",
		}},
		download: func(_ context.Context, path string) (*driver.ArtifactDownload, error) {
			if strings.Contains(path, "unavailable") {
				return nil, downloadFailure
			}
			return &driver.ArtifactDownload{Reader: io.NopCloser(strings.NewReader(path + " bytes")), ContentType: "application/octet-stream"}, nil
		},
	}
	var importedPaths []string
	w := &finalizationWriter{onWrite: func(artifacts []executionwriter.ExternalArtifact) error {
		require.Len(t, artifacts, 3)
		for i, expected := range []struct{ kind, label, source, contents string }{
			{"video_meta", "video-1", video, "video bytes"},
			{"trace_meta", "trace", "remote-available-trace.zip", "remote-available-trace.zip bytes"},
			{"har_meta", "har", "remote-available-network.har", "remote-available-network.har bytes"},
		} {
			artifact := artifacts[i]
			require.Equal(t, expected.kind, artifact.ArtifactType)
			require.Equal(t, expected.label, artifact.Label)
			require.Equal(t, expected.source, artifact.Payload["source_path"])
			data, err := os.ReadFile(artifact.Path)
			require.NoError(t, err)
			require.Equal(t, expected.contents, string(data))
			importedPaths = append(importedPaths, artifact.Path)
		}
		require.Equal(t, 0, artifacts[0].Payload["page_index"])
		return nil
	}}
	err := executeFinalizationFixture(context.Background(), s, w)
	require.ErrorIs(t, err, downloadFailure)
	require.ErrorIs(t, err, closeFailure)
	require.Equal(t, 1, w.writes)
	require.FileExists(t, video)
	for _, path := range importedPaths[1:] {
		require.NoFileExists(t, path, "temporary imports must not leak after persistence")
	}
}

func TestFinalizationRejectsDirectoriesAsArtifacts(t *testing.T) {
	s := &finalizationArtifacts{finalizationSession: &finalizationSession{}, artifacts: &driver.CloseSessionResponse{TracePath: t.TempDir()}}
	w := &finalizationWriter{}
	err := executeFinalizationFixture(context.Background(), s, w)
	require.ErrorContains(t, err, "not a regular file")
	require.Zero(t, w.writes)
}

func TestExecutePropagatesRealArtifactWriterFailure(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "malformed.har")
	require.NoError(t, os.WriteFile(source, []byte("malformed HAR"), 0o600))
	session := &finalizationArtifacts{finalizationSession: &finalizationSession{}, artifacts: &driver.CloseSessionResponse{HARPath: source}}
	writer := executionwriter.NewFileWriter(nil, nil, nil, executionwriter.NewStaticRoot(dir))
	err := executeFinalizationFixture(context.Background(), session, writer)
	require.ErrorContains(t, err, "import har_meta artifact")
	require.FileExists(t, source)
}

func TestExecuteFinalizationSuccessControl(t *testing.T) {
	s := &finalizationSession{}
	require.NoError(t, executeFinalizationFixture(context.Background(), s, &stubExecutionWriter{}))
	require.Equal(t, 1, s.closeCalls)
}

type outcomeRootKey struct{}

type cancellationRoot struct {
	dir string
}

func (r *cancellationRoot) Root(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if ctx.Value(outcomeRootKey{}) != r.dir {
		return "", errors.New("lost storage routing value")
	}
	return r.dir, nil
}
func (*cancellationRoot) RecordWrite(context.Context) {}

type outcomeSink struct {
	*events.MemorySink
	contextErrors []error
	deadlines     []time.Time
}

func (s *outcomeSink) Publish(ctx context.Context, event contracts.EventEnvelope) error {
	if event.Kind == contracts.EventKindStepCompleted || event.Kind == contracts.EventKindStepFailed {
		s.contextErrors = append(s.contextErrors, ctx.Err())
		deadline, _ := ctx.Deadline()
		s.deadlines = append(s.deadlines, deadline)
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return s.MemorySink.Publish(ctx, event)
}

type cancellationWriter struct {
	*executionwriter.FileWriter
	contexts []context.Context
}

func (w *cancellationWriter) RecordStepOutcome(ctx context.Context, plan contracts.ExecutionPlan, outcome contracts.StepOutcome) (executionwriter.RecordResult, error) {
	w.contexts = append(w.contexts, ctx)
	return w.FileWriter.RecordStepOutcome(ctx, plan, outcome)
}

// [REQ:BAS-RH-J18] A canceled action retains queryable evidence through the real
// file writer, including nested steps. Disk failure must preserve both causes.
func TestExecuteCancellationPersistsRealOutcomes(t *testing.T) {
	for _, shape := range []string{"linear", "graph", "loop", "subflow"} {
		for _, diskFailure := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/diskFailure=%t", shape, diskFailure), func(t *testing.T) {
				root := &cancellationRoot{dir: t.TempDir()}
				if diskFailure {
					root.dir = filepath.Join(root.dir, "not-a-directory")
					require.NoError(t, os.WriteFile(root.dir, []byte("preserve original"), 0600))
				}
				ctx, cancel := context.WithCancel(context.WithValue(context.Background(), outcomeRootKey{}, root.dir))
				defer cancel()
				sess := &finalizationSession{cancel: cancel}
				eng := &finalizationEngine{session: sess}
				writer := &cancellationWriter{FileWriter: executionwriter.NewFileWriter(nil, nil, nil, root)}
				action := &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://fixture.invalid"}}}
				instruction := contracts.CompiledInstruction{Index: 1, NodeID: "action", Action: action}
				plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New(), Instructions: []contracts.CompiledInstruction{instruction}}
				sink := &outcomeSink{MemorySink: events.NewMemorySink(contracts.DefaultEventBufferLimits)}
				req := Request{Plan: plan, EngineName: eng.Name(), EngineFactory: engine.NewStaticFactory(eng), Recorder: writer, EventSink: sink, StartFromStepIndex: -1}
				expectedEntries := 1
				if shape != "linear" {
					step := contracts.PlanStep{Index: 1, NodeID: "action", Action: action}
					if shape == "loop" {
						count := int32(2)
						step = contracts.PlanStep{Index: 0, NodeID: "parent", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_LOOP, Params: &basactions.ActionDefinition_Loop{Loop: &basactions.LoopParams{LoopType: basactions.LoopType_LOOP_TYPE_REPEAT, Count: &count}}}, Loop: &contracts.PlanGraph{Steps: []contracts.PlanStep{step}}}
						expectedEntries = 2
					}
					if shape == "subflow" {
						childID := uuid.New()
						req.WorkflowResolver = &stubWorkflowResolver{workflows: map[uuid.UUID]*basapi.WorkflowSummary{childID: {Id: childID.String(), Name: "child"}}}
						req.PlanCompiler = PlanCompilerFunc(func(context.Context, uuid.UUID, *basapi.WorkflowSummary) (contracts.ExecutionPlan, []contracts.CompiledInstruction, error) {
							child := plan
							child.WorkflowID = childID
							return child, child.Instructions, nil
						})
						step = contracts.PlanStep{Index: 0, NodeID: "parent", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_SUBFLOW, Params: &basactions.ActionDefinition_Subflow{Subflow: &basactions.SubflowParams{Target: &basactions.SubflowParams_WorkflowId{WorkflowId: childID.String()}}}}}
						expectedEntries = 2
					}
					req.Plan.Instructions = nil
					req.Plan.Graph = &contracts.PlanGraph{Steps: []contracts.PlanStep{step}}
				}
				err := NewSimpleExecutor(nil).Execute(ctx, req)
				require.ErrorIs(t, err, context.Canceled)
				require.Equal(t, 1, sess.closeCalls)
				require.NotEmpty(t, writer.contexts)
				for _, persistCtx := range writer.contexts {
					deadline, ok := persistCtx.Deadline()
					require.True(t, ok, "persistence must have a finite budget")
					require.WithinDuration(t, time.Now(), deadline, 31*time.Second)
					require.Equal(t, root.dir, persistCtx.Value(outcomeRootKey{}))
				}
				if diskFailure {
					require.Empty(t, sink.contextErrors, "failed writes cannot emit completion receipts")
					require.Contains(t, err.Error(), "not a directory")
					bytes, readErr := os.ReadFile(root.dir)
					require.NoError(t, readErr)
					require.Equal(t, "preserve original", string(bytes))
					return
				}
				data, readErr := os.ReadFile(filepath.Join(root.dir, plan.ExecutionID.String(), "result.json"))
				require.NoError(t, readErr)
				var result struct {
					Entries []struct {
						Context struct {
							Success bool   `json:"success"`
							Error   string `json:"error"`
						} `json:"context"`
					} `json:"entries"`
				}
				require.NoError(t, json.Unmarshal(data, &result))
				require.Len(t, result.Entries, expectedEntries)
				require.Len(t, sink.contextErrors, expectedEntries)
				for i, contextErr := range sink.contextErrors {
					require.NoError(t, contextErr)
					require.WithinDuration(t, time.Now(), sink.deadlines[i], 31*time.Second)
				}
				for _, event := range sink.Events() {
					if event.Kind == contracts.EventKindStepFailed {
						require.Equal(t, 1, *event.Attempt)
						payload := event.Payload.(map[string]any)
						require.NotNil(t, payload["timeline_artifact_id"])
					}
				}
				for _, entry := range result.Entries {
					require.False(t, entry.Context.Success)
					require.Contains(t, entry.Context.Error, "context canceled")
				}
			})
		}
	}
}

func TestExecuteSyntheticOutcomeRespectsCancellation(t *testing.T) {
	for _, graph := range []bool{false, true} {
		for _, canceled := range []bool{false, true} {
			t.Run(fmt.Sprintf("graph=%t/canceled=%t", graph, canceled), func(t *testing.T) {
				root := &cancellationRoot{dir: t.TempDir()}
				ctx, cancel := context.WithCancel(context.WithValue(context.Background(), outcomeRootKey{}, root.dir))
				defer cancel()
				if canceled {
					cancel()
				}
				session := &finalizationSession{}
				eng := &finalizationEngine{session: session}
				writer := executionwriter.NewFileWriter(nil, nil, nil, root)
				sink := &outcomeSink{MemorySink: events.NewMemorySink(contracts.DefaultEventBufferLimits)}
				action := &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_SET_VARIABLE, Params: &basactions.ActionDefinition_SetVariable{SetVariable: &basactions.SetVariableParams{Name: "sentinel"}}}
				plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New(), Instructions: []contracts.CompiledInstruction{{Index: 0, NodeID: "variable", Action: action}}}
				if graph {
					plan.Instructions = nil
					plan.Graph = &contracts.PlanGraph{Steps: []contracts.PlanStep{{Index: 0, NodeID: "variable", Action: action}}}
				}
				err := NewSimpleExecutor(nil).Execute(ctx, Request{Plan: plan, EngineName: eng.Name(), EngineFactory: engine.NewStaticFactory(eng), Recorder: writer, EventSink: sink, StartFromStepIndex: -1})
				if canceled {
					require.ErrorIs(t, err, context.Canceled)
					require.Zero(t, session.closeCalls, "canceled-before-admission must not create a browser")
				} else {
					require.NoError(t, err)
				}
				terminal := sink.Events()
				require.Len(t, terminal, 1)
				require.NoError(t, sink.contextErrors[0])
				outcome := terminal[0].Payload.(map[string]any)["outcome"].(contracts.StepOutcome)
				require.Equal(t, !canceled, outcome.Success)
				require.Equal(t, "variable", outcome.NodeID)
				require.FileExists(t, filepath.Join(root.dir, plan.ExecutionID.String(), "result.json"))
			})
		}
	}
}

// Exercise the public executor through the real HTTP client/session boundary.
func TestExecutePreservesInvocationAndTransportOwnership(t *testing.T) {
	for _, mode := range []string{"retry", "loop", "lost-response", "malformed-response"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			var packets []map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				switch {
				case strings.HasSuffix(r.URL.Path, "/start"):
					io.WriteString(w, `{"session_id":"wire-session","lease_id":"wire-lease"}`)
				case strings.HasSuffix(r.URL.Path, "/run"):
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
						return
					}
					body["header"] = r.Header.Get("X-Idempotency-Key")
					packets = append(packets, body)
					switch mode {
					case "retry":
						if len(packets) == 1 {
							io.WriteString(w, `{"success":false,"failure":{"kind":"engine","code":"TRANSIENT","retryable":true}}`)
							return
						}
					case "malformed-response":
						io.WriteString(w, `{"success":`)
						return
					case "lost-response":
						conn, _, err := w.(http.Hijacker).Hijack()
						if err != nil {
							t.Error(err)
							return
						}
						conn.Close()
						return
					}
					io.WriteString(w, `{"success":true}`)
				default:
					io.WriteString(w, `{"success":true}`)
				}
			}))
			defer server.Close()
			eng, err := engine.NewPlaywrightEngineWithHTTPClient(server.URL, server.Client(), nil)
			require.NoError(t, err)
			action := &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE,
				Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://fixture.invalid"}}}
			plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New(), Instructions: []contracts.CompiledInstruction{{
				NodeID: "same-node", Action: action, Context: map[string]any{"resilience": map[string]any{"maxAttempts": 2, "delayMs": 0}},
			}}}
			if mode == "loop" {
				count := int32(2)
				plan.Instructions = nil
				plan.Graph = &contracts.PlanGraph{Steps: []contracts.PlanStep{{NodeID: "repeat", Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_LOOP, Params: &basactions.ActionDefinition_Loop{Loop: &basactions.LoopParams{LoopType: basactions.LoopType_LOOP_TYPE_REPEAT, Count: &count}},
				}, Loop: &contracts.PlanGraph{Steps: []contracts.PlanStep{{NodeID: "same-node", Index: 1, Action: action}}}}}}
			}
			err = NewSimpleExecutor(nil).Execute(context.Background(), Request{Plan: plan, EngineName: "playwright", EngineFactory: engine.NewStaticFactory(eng), Recorder: &stubExecutionWriter{}, EventSink: events.NewMemorySink(contracts.DefaultEventBufferLimits)})
			mu.Lock()
			defer mu.Unlock()
			if mode == "lost-response" || mode == "malformed-response" {
				require.Error(t, err)
				// The HTTP transport may repeat a replayable packet; a new operation
				// would authorize another browser effect and is forbidden.
				require.NotEmpty(t, packets)
				for _, packet := range packets {
					require.Equal(t, packets[0], packet)
				}
				return
			}
			require.NoError(t, err)
			require.Len(t, packets, 2)
			require.NotEmpty(t, packets[0]["invocation_id"])
			require.Equal(t, float64(1), packets[0]["operation_sequence"])
			require.Equal(t, float64(2), packets[1]["operation_sequence"])
			require.Equal(t, "wire-lease:1", packets[0]["header"])
			require.Equal(t, "wire-lease:2", packets[1]["header"])
			if mode == "retry" {
				require.Equal(t, packets[0]["invocation_id"], packets[1]["invocation_id"])
				require.Equal(t, float64(2), packets[1]["attempt"])
			} else {
				require.NotEqual(t, packets[0]["invocation_id"], packets[1]["invocation_id"])
				require.Equal(t, float64(1), packets[1]["attempt"])
			}
		})
	}
}

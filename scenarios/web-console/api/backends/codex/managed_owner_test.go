package codex

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestManagedOwnerForkRetainsOriginalBranch(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"result":{"threadId":"thread-2"}}` + "\n"
	c := NewClient(writeCloser{io.Discard}, strings.NewReader(input))
	o, err := NewManagedOwner(c, "web-console:s1", "thread-1")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := o.ForkThrough(context.Background(), "turn-4"); err != nil || got != "thread-2" {
		t.Fatalf("fork = %q, %v", got, err)
	}
	if o.OriginalThreadID() != "thread-1" || o.ThreadID() != "thread-2" || o.LastVerifiedTurnID() != "turn-4" {
		t.Fatalf("owner lineage not retained: original=%q active=%q turn=%q", o.OriginalThreadID(), o.ThreadID(), o.LastVerifiedTurnID())
	}
}

type captureWriter struct {
	mu   sync.Mutex
	data strings.Builder
}

func (w *captureWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.data.Write(p)
}
func (w *captureWriter) Close() error   { return nil }
func (w *captureWriter) String() string { w.mu.Lock(); defer w.mu.Unlock(); return w.data.String() }

func TestManagedOwnerPromptDeliveryUsesNativeGenerationAndTurnCAS(t *testing.T) {
	w := &captureWriter{}
	provider, consumer := net.Pipe()
	t.Cleanup(func() { _ = provider.Close(); _ = consumer.Close() })
	go func() {
		decoder := json.NewDecoder(io.TeeReader(provider, w))
		encoder := json.NewEncoder(provider)
		for _, response := range []string{`{"thread":{"id":"thread-1","status":{"type":"idle"}}}`, `{"turnId":"turn-started"}`, `{"turnId":"turn-busy"}`} {
			var req Message
			if err := decoder.Decode(&req); err != nil {
				return
			}
			if err := encoder.Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": json.RawMessage(response)}); err != nil {
				return
			}
		}
	}()
	client := NewClient(consumer, consumer)
	defer client.Close()
	o, err := NewManagedOwner(client, "web-console:s1", "thread-1")
	if err != nil {
		t.Fatal(err)
	}
	if turn, delivery, err := o.DeliverPrompt(context.Background(), "thread-1", "", "start this"); err != nil || turn != "turn-started" || delivery != "started" {
		t.Fatalf("idle delivery = %q %q %v", turn, delivery, err)
	}
	o.mu.Lock()
	o.activeTurnID = "turn-busy"
	o.mu.Unlock()
	if turn, delivery, err := o.DeliverPrompt(context.Background(), "thread-1", "turn-busy", "steer this"); err != nil || turn != "turn-busy" || delivery != "steered" {
		t.Fatalf("busy delivery = %q %q %v", turn, delivery, err)
	}
	wire := w.String()
	for _, expected := range []string{`"method":"turn/start"`, `"threadId":"thread-1"`, `"method":"turn/steer"`, `"expectedTurnId":"turn-busy"`} {
		if !strings.Contains(wire, expected) {
			t.Fatalf("native request missing %q: %s", expected, wire)
		}
	}
	before := len(wire)
	if _, _, err := o.DeliverPrompt(context.Background(), "thread-replaced", "turn-busy", "must not route"); err != ErrGenerationChanged {
		t.Fatalf("generation mismatch error = %v", err)
	}
	if got := w.String(); len(got) != before {
		t.Fatalf("generation mismatch reached native transport: %s", got[before:])
	}
}

func TestManagedOwnerRejectsMissingBoundary(t *testing.T) {
	c := NewClient(writeCloser{io.Discard}, strings.NewReader(""))
	o, err := NewManagedOwner(c, "owner", "thread")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := o.ForkThrough(context.Background(), ""); err != ErrMissingTurn {
		t.Fatalf("error = %v", err)
	}
}

func TestManagedOwnerRecordsOnlyCompletedTurnForControl(t *testing.T) {
	o, err := NewManagedOwner(NewClient(writeCloser{io.Discard}, strings.NewReader("")), "owner", "thread")
	if err != nil {
		t.Fatal(err)
	}
	if o.MarkTurnCompleted("other-thread", "turn-1") {
		t.Fatal("foreign thread completion was accepted")
	}
	if !o.MarkTurnCompleted("thread", "turn-2") || o.LastVerifiedTurnID() != "turn-2" {
		t.Fatalf("owner = %+v", o)
	}
}

func TestManagedOwnerRestoresPersistedForkLineage(t *testing.T) {
	o, err := NewManagedOwner(NewClient(writeCloser{io.Discard}, strings.NewReader("")), "owner", "thread-active")
	if err != nil {
		t.Fatal(err)
	}
	if err := o.RestoreLineage("thread-original", "turn-4"); err != nil {
		t.Fatal(err)
	}
	if o.ThreadID() != "thread-active" || o.OriginalThreadID() != "thread-original" || o.LastVerifiedTurnID() != "turn-4" {
		t.Fatalf("restored lineage = active=%q original=%q turn=%q", o.ThreadID(), o.OriginalThreadID(), o.LastVerifiedTurnID())
	}
}

// scriptedGoalOwner uses private in-memory sockets; it never starts a provider.
func scriptedGoalOwner(t *testing.T, responses ...string) (*ManagedOwner, *captureWriter) {
	t.Helper()
	provider, consumer := net.Pipe()
	w := &captureWriter{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		decoder := json.NewDecoder(io.TeeReader(provider, w))
		encoder := json.NewEncoder(provider)
		for _, response := range responses {
			var req Message
			if err := decoder.Decode(&req); err != nil {
				return
			}
			if err := encoder.Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": json.RawMessage(response)}); err != nil {
				return
			}
		}
	}()
	client := NewClient(consumer, consumer)
	t.Cleanup(func() { _ = client.Close(); _ = provider.Close(); <-done })
	owner, err := NewManagedOwner(client, "owner", "thread-1")
	if err != nil {
		t.Fatal(err)
	}
	return owner, w
}

func TestManagedOwnerGoalMutationConfirmsProviderBinding(t *testing.T) {
	for _, origin := range []GoalMutationOrigin{GoalOriginUserDirected, GoalOriginAutomatic} {
		for _, tc := range []struct {
			name, response string
			wantError      bool
		}{
			{"exact", `{"goal":{"threadId":"thread-1","objective":"finish the admitted task","status":"active"}}`, false},
			{"wrong thread", `{"goal":{"threadId":"other","objective":"finish the admitted task","status":"active"}}`, true},
			{"wrong objective", `{"goal":{"threadId":"thread-1","objective":"different task","status":"active"}}`, true},
			{"wrong status", `{"goal":{"threadId":"thread-1","objective":"finish the admitted task","status":"paused"}}`, true},
			{"missing thread", `{"goal":{"objective":"finish the admitted task","status":"active"}}`, true},
			{"missing objective", `{"goal":{"threadId":"thread-1","status":"active"}}`, true},
			{"missing status", `{"goal":{"threadId":"thread-1","objective":"finish the admitted task"}}`, true},
			{"missing goal", `{}`, true},
		} {
			t.Run(string(origin)+"/"+tc.name, func(t *testing.T) {
				responses := []string{tc.response}
				if origin == GoalOriginAutomatic {
					responses = append([]string{`{"goal":{"threadId":"thread-1","objective":"finish the admitted task","status":"paused"}}`}, responses...)
				}
				owner, w := scriptedGoalOwner(t, responses...)
				got, err := owner.SetGoal(context.Background(), "thread-1", "finish the admitted task", "active", origin, nil)
				if (err != nil) != tc.wantError {
					t.Fatalf("goal=%+v err=%v", got, err)
				}
				if !tc.wantError && (got.ThreadID != "thread-1" || got.Objective != "finish the admitted task" || got.Status != "active") {
					t.Fatalf("provider binding lost: %+v", got)
				}
				if !strings.Contains(w.String(), `"origin":"`+string(origin)+`"`) {
					t.Fatalf("origin missing from wire: %s", w.String())
				}
			})
		}
	}
}

func TestManagedOwnerGoalMutationRejectsMalformedIntentBeforeSend(t *testing.T) {
	for _, tc := range []struct {
		name, thread, objective, status string
		origin                          GoalMutationOrigin
	}{
		{"missing origin", "thread-1", "objective", "active", ""},
		{"unknown origin", "thread-1", "objective", "active", "guessed"},
		{"wrong thread", "other", "objective", "active", GoalOriginAutomatic},
		{"empty active objective", "thread-1", "", "active", GoalOriginUserDirected},
		{"invalid status", "thread-1", "objective", "invented", GoalOriginAutomatic},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := &captureWriter{}
			client := NewClient(w, strings.NewReader(""))
			defer client.Close()
			owner, err := NewManagedOwner(client, "owner", "thread-1")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := owner.SetGoal(context.Background(), tc.thread, tc.objective, tc.status, tc.origin, nil); err == nil {
				t.Fatal("malformed mutation admitted")
			}
			if w.String() != "" {
				t.Fatalf("malformed mutation reached provider: %s", w.String())
			}
		})
	}
}

func TestManagedOwnerAutomaticStatusMutationPreservesProviderObjective(t *testing.T) {
	owner, w := scriptedGoalOwner(t,
		`{"goal":{"threadId":"thread-1","objective":"original admitted objective","status":"active"}}`,
		`{"goal":{"threadId":"thread-1","objective":"original admitted objective","status":"paused"}}`)
	goal, err := owner.SetGoal(context.Background(), "thread-1", "", "paused", GoalOriginAutomatic, nil)
	if err != nil || goal.Objective != "original admitted objective" {
		t.Fatalf("goal=%+v err=%v", goal, err)
	}
	if strings.Contains(w.String(), `"objective"`) || !strings.Contains(w.String(), `"origin":"automatic"`) {
		t.Fatalf("status mutation invented user/objective intent: %s", w.String())
	}
}

func TestAutomaticGoalRecoveryRefusesUnknownOrChangedIntentBeforeSet(t *testing.T) {
	for _, tc := range []struct{ name, response string }{
		{"user changed objective", `{"goal":{"threadId":"thread-1","objective":"user's newer goal","status":"paused"}}`},
		{"absent goal", `{"goal":null}`},
		{"missing goal", `{}`},
		{"unreadable goal", `{"goal":"unavailable"}`},
		{"wrong thread", `{"goal":{"threadId":"other","objective":"admitted objective","status":"paused"}}`},
		{"unknown status", `{"goal":{"threadId":"thread-1","objective":"admitted objective","status":"unknown"}}`},
		{"missing objective", `{"goal":{"threadId":"thread-1","status":"paused"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			owner, wire := scriptedGoalOwner(t, tc.response)
			if _, err := owner.SetGoal(context.Background(), "thread-1", "admitted objective", "active", GoalOriginAutomatic, nil); err == nil {
				t.Fatal("automatic recovery accepted unknown or changed user intent")
			}
			if !strings.Contains(wire.String(), `"method":"thread/goal/get"`) || strings.Contains(wire.String(), `"method":"thread/goal/set"`) {
				t.Fatalf("recovery mutated before checking authoritative intent: %s", wire.String())
			}
		})
	}
}

func TestAutomaticGoalRecoveryPreservesTextChangedBetweenReadAndSet(t *testing.T) {
	owner, wire := scriptedGoalOwner(t,
		`{"goal":{"threadId":"thread-1","objective":"admitted objective","status":"paused"}}`,
		`{"goal":{"threadId":"thread-1","objective":"user's newer goal","status":"active"}}`)
	if _, err := owner.SetGoal(context.Background(), "thread-1", "admitted objective", "active", GoalOriginAutomatic, nil); err == nil {
		t.Fatal("changed provider objective was accepted as the admitted objective")
	}
	if strings.Contains(wire.String(), `"objective"`) {
		t.Fatal("automatic mutation sent replacement objective text")
	}
}

func TestAutomaticGoalRecoveryReadFailureDoesNotSet(t *testing.T) {
	wire := &captureWriter{}
	client := NewClient(wire, strings.NewReader(`{"jsonrpc":"2.0","id":1,"error":{"code":-1,"message":"goal source unavailable"}}`+"\n"))
	defer client.Close()
	owner, err := NewManagedOwner(client, "owner", "thread-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := owner.SetGoal(context.Background(), "thread-1", "admitted objective", "active", GoalOriginAutomatic, nil); err == nil {
		t.Fatal("unavailable goal source was accepted")
	}
	if strings.Contains(wire.String(), `"method":"thread/goal/set"`) {
		t.Fatal("failed authoritative read reached goal mutation")
	}
}

func TestManagedOwnerRetirementRefusesRawPointerMutations(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	output := &captureWriter{}
	owner, err := NewManagedOwner(NewClient(output, reader), "generation", "thread")
	if err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if err := owner.CloseContext(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("close=%v", err)
	}
	if _, err := owner.ForkThrough(t.Context(), "completed-turn"); !errors.Is(err, ErrClosed) {
		t.Fatalf("fork=%v", err)
	}
	if _, _, err := owner.DeliverPrompt(t.Context(), "thread", "", "fresh work"); !errors.Is(err, ErrClosed) {
		t.Fatalf("prompt=%v", err)
	}
	if _, err := owner.SetGoal(t.Context(), "thread", "fresh goal", "active", GoalOriginAutomatic, nil); !errors.Is(err, ErrClosed) {
		t.Fatalf("goal=%v", err)
	}
	if _, err := owner.StartTurn(t.Context(), "fresh work"); !errors.Is(err, ErrClosed) {
		t.Fatalf("start=%v", err)
	}
	if got := output.String(); got != "" {
		t.Fatalf("retiring raw pointer wrote provider input: %s", got)
	}
	_ = writer.Close()
	ctx, stop := context.WithTimeout(t.Context(), time.Second)
	defer stop()
	if err := owner.CloseContext(ctx); err != nil {
		t.Fatal(err)
	}
}

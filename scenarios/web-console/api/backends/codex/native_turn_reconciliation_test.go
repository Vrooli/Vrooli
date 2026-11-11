package codex

import (
	"errors"
	"strings"
	"testing"
)

func TestManagedOwnerReconcilesGoalStartedTurnBeforeInitialTask(t *testing.T) {
	owner, wire := scriptedGoalOwner(t,
		`{"goal":{"threadId":"thread-1","objective":"finish task","status":"active"}}`,
		`{"thread":{"id":"thread-1","status":{"type":"active"}}}`,
		`{"goal":{"threadId":"thread-1","status":"active"}}`,
		`{"data":[{"id":"provider-goal-turn","status":"inProgress"}],"nextCursor":null}`,
		`{"turnId":"provider-goal-turn"}`,
	)
	if _, err := owner.SetGoal(t.Context(), "thread-1", "finish task", "active", GoalOriginUserDirected, nil); err != nil {
		t.Fatal(err)
	}
	states, err := owner.ReadNativeThreadStates(t.Context(), []string{"thread-1"})
	if err != nil || len(states) != 1 || states[0].Status != "active" || states[0].ActiveTurnID != "provider-goal-turn" {
		t.Fatalf("goal-started turn not recovered: %+v, %v", states, err)
	}
	turn, delivery, err := owner.DeliverPrompt(t.Context(), "thread-1", states[0].ActiveTurnID, "initial task with full context")
	if err != nil || turn != "provider-goal-turn" || delivery != "steered" {
		t.Fatalf("initial task delivery = %q %q %v", turn, delivery, err)
	}
	for _, fragment := range []string{`"method":"thread/turns/list"`, `"limit":1`, `"itemsView":"summary"`, `"expectedTurnId":"provider-goal-turn"`} {
		if !strings.Contains(wire.String(), fragment) {
			t.Fatalf("missing bounded receiver contract %s: %s", fragment, wire.String())
		}
	}
	if strings.Contains(wire.String(), `"includeTurns":true`) || strings.Contains(wire.String(), `"method":"turn/start"`) {
		t.Fatalf("provider active turn was replaced or full history hydrated: %s", wire.String())
	}
}

func TestManagedOwnerActiveTurnReadDoesNotInventIdentityFromCache(t *testing.T) {
	for _, turns := range []string{`{"data":[]}`, `{"data":[{"id":"old","status":"completed"}]}`, `{"data":[{"id":"","status":"inProgress"}]}`} {
		t.Run(turns, func(t *testing.T) {
			owner, _ := scriptedGoalOwner(t,
				`{"thread":{"id":"thread-1","status":{"type":"active"}}}`,
				`{"goal":null}`, turns,
			)
			owner.MarkTurnStarted("thread-1", "cached-turn")
			states, err := owner.ReadNativeThreadStates(t.Context(), []string{"thread-1"})
			if err != nil || states[0].Status != "active" || states[0].ActiveTurnID != "" {
				t.Fatalf("missing authoritative turn became deliverable: %+v %v", states, err)
			}
		})
	}
}

func TestManagedOwnerIdleInitialTaskStartsOnceAndRefusesStaleRetry(t *testing.T) {
	owner, wire := scriptedGoalOwner(t,
		`{"thread":{"id":"thread-1","status":{"type":"idle"}}}`,
		`{"goal":null}`,
		`{"thread":{"id":"thread-1","status":{"type":"idle"}}}`,
		`{"turnId":"initial-turn"}`,
		`{"thread":{"id":"thread-1","status":{"type":"active"}}}`,
	)
	states, err := owner.ReadNativeThreadStates(t.Context(), []string{"thread-1"})
	if err != nil || states[0].Status != "idle" || states[0].ActiveTurnID != "" {
		t.Fatalf("initial idle state = %+v %v", states, err)
	}
	if _, delivery, err := owner.DeliverPrompt(t.Context(), "thread-1", "", "first task"); err != nil || delivery != "started" {
		t.Fatalf("initial prompt failed: %q %v", delivery, err)
	}
	if _, _, err := owner.DeliverPrompt(t.Context(), "thread-1", "", "stale retry"); !errors.Is(err, ErrTurnBusy) {
		t.Fatalf("stale initial retry = %v", err)
	}
	if strings.Count(wire.String(), `"method":"turn/start"`) != 1 {
		t.Fatalf("initial prompt duplicated: %s", wire.String())
	}
}

func TestManagedOwnerIdleDispatchRechecksProviderDespiteDelayedStart(t *testing.T) {
	owner, wire := scriptedGoalOwner(t,
		`{"thread":{"id":"thread-1","status":{"type":"idle"}}}`,
		`{"goal":null}`,
		`{"thread":{"id":"thread-1","status":{"type":"idle"}}}`,
		`{"turnId":"fresh-initial"}`,
	)
	states, err := owner.ReadNativeThreadStates(t.Context(), []string{"thread-1"})
	if err != nil || states[0].Status != "idle" {
		t.Fatalf("idle observation = %+v %v", states, err)
	}
	owner.MarkTurnStarted("thread-1", "delayed-old-turn")
	if turn, delivery, err := owner.DeliverPrompt(t.Context(), "thread-1", "", "first task"); err != nil || turn != "fresh-initial" || delivery != "started" {
		t.Fatalf("stale notification vetoed current idle provider: %q %q %v", turn, delivery, err)
	}
	if strings.Count(wire.String(), `"method":"turn/start"`) != 1 {
		t.Fatalf("unexpected initial dispatch: %s", wire.String())
	}
}

func TestManagedOwnerTurnNotificationsPreserveCurrentTurnAndRetirement(t *testing.T) {
	owner, _ := scriptedGoalOwner(t)
	if owner.MarkTurnStarted("foreign-thread", "foreign") || owner.MarkTurnStarted("thread-1", "") {
		t.Fatal("unbound start accepted")
	}
	if !owner.MarkTurnStarted("thread-1", "first") || !owner.MarkTurnCompleted("thread-1", "first") {
		t.Fatal("provider lifecycle not recorded")
	}
	if owner.MarkTurnStarted("thread-1", "first") {
		t.Fatal("late started notification resurrected completed turn")
	}
	if !owner.MarkTurnStarted("thread-1", "next") || owner.MarkTurnCompleted("thread-1", "first") || owner.ActiveTurnID() != "next" {
		t.Fatal("late completion cleared a newer turn")
	}
	owner.BeginRetirement()
	if owner.MarkTurnStarted("thread-1", "after-close") || owner.MarkTurnCompleted("thread-1", "next") {
		t.Fatal("retiring owner accepted event")
	}
	if _, err := owner.ReadNativeThreadStates(t.Context(), []string{"thread-1"}); !errors.Is(err, ErrClosed) {
		t.Fatalf("retiring owner remained readable: %v", err)
	}
}

func TestManagedOwnerSteeringUsesProviderCASDespiteDelayedNotifications(t *testing.T) {
	owner, wire := scriptedGoalOwner(t,
		`{"thread":{"id":"thread-1","status":{"type":"active"}}}`,
		`{"goal":null}`,
		`{"data":[{"id":"current-B","status":"inProgress"}]}`,
		`{"turnId":"current-B"}`,
	)
	states, err := owner.ReadNativeThreadStates(t.Context(), []string{"thread-1"})
	if err != nil || states[0].ActiveTurnID != "current-B" {
		t.Fatalf("authoritative turn = %+v %v", states, err)
	}
	owner.MarkTurnStarted("thread-1", "delayed-A")
	owner.MarkTurnCompleted("thread-1", "delayed-A")
	if turn, delivery, err := owner.DeliverPrompt(t.Context(), "thread-1", states[0].ActiveTurnID, "initial task"); err != nil || turn != "current-B" || delivery != "steered" {
		t.Fatalf("notification cache vetoed provider CAS: %q %q %v", turn, delivery, err)
	}
	if !strings.Contains(wire.String(), `"expectedTurnId":"current-B"`) {
		t.Fatalf("lost exact expected turn: %s", wire.String())
	}
}

func TestManagedOwnerStaleExpectedTurnRemainsProviderRefusal(t *testing.T) {
	wire := &captureWriter{}
	client := NewClient(wire, strings.NewReader(`{"jsonrpc":"2.0","id":1,"error":{"code":-32602,"message":"active turn changed"}}`+"\n"))
	t.Cleanup(func() { _ = client.Close() })
	owner, err := NewManagedOwner(client, "owner", "thread-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := owner.DeliverPrompt(t.Context(), "thread-1", "stale-turn", "must not be restarted"); err == nil {
		t.Fatal("provider refused a stale turn but delivery succeeded")
	}
	if !strings.Contains(wire.String(), `"method":"turn/steer"`) || !strings.Contains(wire.String(), `"expectedTurnId":"stale-turn"`) || strings.Contains(wire.String(), `"method":"turn/start"`) {
		t.Fatalf("provider refusal bypassed: %s", wire.String())
	}
}

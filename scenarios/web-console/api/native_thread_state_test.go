package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
	"web-console/backends/codex"
	"web-console/internal/sessionstore"
)

type stateCaller func(context.Context, string, any, any) error

func (f stateCaller) Call(ctx context.Context, method string, args, out any) error {
	return f(ctx, method, args, out)
}

func TestNativeThreadStatesPreserveUnavailableAndNoGoal(t *testing.T) {
	calls := 0
	caller := stateCaller(func(_ context.Context, method string, args, out any) error {
		calls++
		id := args.(map[string]any)["threadId"].(string)
		if id == "missing" {
			return errors.New("unavailable")
		}
		var value any
		if method == "thread/read" {
			value = map[string]any{"thread": map[string]any{"id": id, "status": map[string]any{"type": "active", "activeFlags": []string{"waitingOnApproval"}}}}
		} else if id == "no-goal" {
			value = map[string]any{"goal": nil}
		} else if id == "bad-goal" {
			value = map[string]any{"goal": map[string]any{"threadId": "other", "status": "complete"}}
		} else {
			value = map[string]any{"goal": map[string]any{"threadId": id, "status": "active", "tokensUsed": 12, "timeUsedSeconds": 3, "tokenBudget": 100}}
		}
		data, _ := json.Marshal(value)
		return json.Unmarshal(data, out)
	})
	got := readNativeThreadStates(context.Background(), caller, []string{"active", "no-goal", "bad-goal", "missing"}).GetThreads()
	if len(got) != 4 || calls != 7 {
		t.Fatalf("rows=%d calls=%d", len(got), calls)
	}
	if got[0].GetStatus() != "active" || !got[0].GetHasGoal() || got[0].GetGoalStatus() != "active" || got[0].GetTokensUsed() != 12 || got[0].GetTokenBudget() != 100 || len(got[0].GetActiveFlags()) != 1 {
		t.Fatalf("active row=%+v", got[0])
	}
	if !got[1].GetGoalAvailable() || got[1].GetHasGoal() {
		t.Fatalf("missing goal row=%+v", got[1])
	}
	if got[2].GetGoalAvailable() || got[2].GetHasGoal() {
		t.Fatalf("mismatched goal row=%+v", got[2])
	}
	if got[3].GetStatus() != "unavailable" || got[3].GetGoalAvailable() {
		t.Fatalf("unavailable row=%+v", got[3])
	}
}

func TestNativeThreadStateRejectsInvalidInputsBeforeOwner(t *testing.T) {
	server := &Server{}
	valid := "01a182ed-c43e-7be0-9406-10c15448b4a7"
	for _, req := range []*sessionsv1.ReadNativeThreadStateRequest{
		{SessionId: "invalid", ThreadIds: []string{valid}},
		{SessionId: valid},
		{SessionId: valid, ThreadIds: []string{valid, valid}},
		{SessionId: valid, ThreadIds: []string{"invalid"}},
	} {
		if _, err := server.readSessionNativeThreadState(context.Background(), req); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("request=%v error=%v", req, err)
		}
	}
}

func TestNativeThreadStatesStopCallingOwnerAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	caller := stateCaller(func(context.Context, string, any, any) error { calls++; return nil })
	rows := readNativeThreadStates(ctx, caller, []string{"a", "b"}).GetThreads()
	if calls != 0 || len(rows) != 2 || rows[0].GetStatus() != "unavailable" || rows[1].GetStatus() != "unavailable" {
		t.Fatalf("calls=%d rows=%v", calls, rows)
	}
}

func TestNativeThreadStatesTreatMalformedGoalAsUnavailable(t *testing.T) {
	caller := stateCaller(func(_ context.Context, method string, _ any, out any) error {
		var data []byte
		if method == "thread/read" {
			data = []byte(`{"thread":{"id":"a","status":{"type":"idle"}}}`)
		} else {
			data = []byte(`{"goal":{"threadId":"a","status":""}}`)
		}
		return json.Unmarshal(data, out)
	})
	row := readNativeThreadStates(context.Background(), caller, []string{"a"}).GetThreads()[0]
	if row.GetGoalAvailable() || row.GetHasGoal() {
		t.Fatalf("malformed goal should be unavailable: %+v", row)
	}
}

func TestNativeThreadStatesRequireExplicitGoalEnvelope(t *testing.T) {
	for _, tc := range []struct {
		name, payload string
		known         bool
	}{{"explicit no goal", `{"goal":null}`, true}, {"omitted goal", `{}`, false}, {"null envelope", `null`, false}, {"invalid goal", `{"goal":[]}`, false}, {"unknown status", `{"goal":{"threadId":"a","status":"future"}}`, false}} {
		t.Run(tc.name, func(t *testing.T) {
			caller := stateCaller(func(_ context.Context, method string, _ any, out any) error {
				payload := tc.payload
				if method == "thread/read" {
					payload = `{"thread":{"id":"a","status":{"type":"idle"}}}`
				}
				return json.Unmarshal([]byte(payload), out)
			})
			row := readNativeThreadStates(t.Context(), caller, []string{"a"}).GetThreads()[0]
			if row.GetGoalAvailable() != tc.known || row.GetHasGoal() {
				t.Fatalf("unverified goal envelope became known absence: %+v", row)
			}
		})
	}
}

// A Codex TUI without an app-server daemon (a fresh launch, or a lead reopened
// from the archive) has no live owner for thread status, but the goal Codex
// persisted is readable; thread status is never inferred.
func TestTerminalCodexWithoutDaemonReportsStoredGoal(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WC_SESSION_STATE_ROOT", root)
	store := sessionstore.NewInMemory()
	meta := sessionstore.Metadata{ID: uuid.NewString(), Status: sessionstore.StatusLive, AgentType: sessionstore.AgentCodex, LaunchMode: sessionstore.LaunchModeTerminalPTY}
	if err := store.Save(t.Context(), meta); err != nil {
		t.Fatal(err)
	}
	server := &Server{sessionStore: store}
	lead, other := uuid.NewString(), uuid.NewString()
	req := &sessionsv1.ReadNativeThreadStateRequest{SessionId: meta.ID, ThreadIds: []string{lead, other}}

	if _, err := server.readSessionNativeThreadState(t.Context(), req); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("no daemon and no goal store: err = %v, want unavailable", err)
	}

	home := codex.SessionHomePath(resolveSessionStateRoot(), meta.ID)
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:"+filepath.Join(home, "goals_1.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`CREATE TABLE thread_goals (thread_id TEXT PRIMARY KEY NOT NULL, goal_id TEXT NOT NULL, objective TEXT NOT NULL, status TEXT NOT NULL, token_budget INTEGER, tokens_used INTEGER NOT NULL DEFAULT 0, time_used_seconds INTEGER NOT NULL DEFAULT 0, created_at_ms INTEGER NOT NULL, updated_at_ms INTEGER NOT NULL)`,
		`INSERT INTO thread_goals VALUES ('` + lead + `', 'g', 'objective', 'paused', NULL, 340295820, 86400, 0, 0)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()

	resp, err := server.readSessionNativeThreadState(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	rows := resp.GetThreads()
	if len(rows) != 2 {
		t.Fatalf("rows = %+v", rows)
	}
	if r := rows[0]; r.GetStatus() != "unavailable" || !r.GetGoalAvailable() || !r.GetHasGoal() || r.GetGoalStatus() != "paused" || r.GetTokensUsed() != 340295820 || r.GetTimeUsedSeconds() != 86400 {
		t.Fatalf("lead row = %+v, want unavailable status with the stored paused goal", r)
	}
	if r := rows[1]; r.GetStatus() != "unavailable" || !r.GetGoalAvailable() || r.GetHasGoal() {
		t.Fatalf("goal-less thread row = %+v, want no goal", r)
	}
}

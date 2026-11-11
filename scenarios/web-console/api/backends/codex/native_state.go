package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// NativeThreadState is observed provider state, independent of launch intent.
// A failed goal read leaves GoalAvailable false; it never implies no goal.
type NativeThreadState struct {
	ThreadID      string
	Status        string
	ActiveFlags   []string
	ActiveTurnID  string
	GoalAvailable bool
	Goal          *NativeGoalState
}

type NativeGoalState struct {
	ThreadID        string `json:"threadId"`
	Status          string `json:"status"`
	TokensUsed      int64  `json:"tokensUsed"`
	TimeUsedSeconds int64  `json:"timeUsedSeconds"`
	TokenBudget     *int64 `json:"tokenBudget"`
}

type NativeStateCaller interface {
	Call(context.Context, string, any, any) error
}

// readNativeThreadStatus reads only bounded status from the selected transport.
func readNativeThreadStatus(ctx context.Context, client NativeStateCaller, id string) NativeThreadState {
	out := NativeThreadState{ThreadID: id, Status: "unavailable"}
	if ctx.Err() != nil {
		return out
	}
	var read struct {
		Thread struct {
			ID     string `json:"id"`
			Status struct {
				Type        string   `json:"type"`
				ActiveFlags []string `json:"activeFlags"`
			} `json:"status"`
		} `json:"thread"`
	}
	if err := client.Call(ctx, "thread/read", map[string]any{"threadId": id, "includeTurns": false}, &read); err != nil || read.Thread.ID != id {
		return out
	}
	if read.Thread.Status.Type != "" {
		out.Status = read.Thread.Status.Type
		out.ActiveFlags = read.Thread.Status.ActiveFlags
	}
	return out
}

// ReadNativeThreadState reads status and goal from the already selected owner.
// It never creates a thread or chooses another owner.
func ReadNativeThreadState(ctx context.Context, client NativeStateCaller, id string) NativeThreadState {
	out := readNativeThreadStatus(ctx, client, id)
	if out.Status == "unavailable" {
		return out
	}
	var envelope map[string]json.RawMessage
	if err := client.Call(ctx, "thread/goal/get", map[string]any{"threadId": id}, &envelope); err != nil {
		if errors.Is(err, ErrClosed) || ctx.Err() != nil {
			out.Status = "unavailable"
			out.ActiveFlags = nil
		}
		return out
	}
	raw, present := envelope["goal"]
	if !present {
		return out
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		out.GoalAvailable = true
		return out
	}
	var goal NativeGoalState
	if err := json.Unmarshal(raw, &goal); err != nil || goal.ThreadID != id {
		return out
	}
	switch goal.Status {
	case "active", "paused", "blocked", "usageLimited", "budgetLimited", "complete":
	default:
		return out
	}
	out.GoalAvailable = true
	out.Goal = &goal
	return out
}

// ReadNativeThreadStates pins the owner's exact active/original lineage across
// all reads. Foreign subjects are refused before any provider RPC. Retirement
// cannot admit a read through a stale raw owner reference.
func (o *ManagedOwner) ReadNativeThreadStates(ctx context.Context, ids []string) ([]NativeThreadState, error) {
	if o == nil || o.client == nil || o.closing.Load() {
		return nil, ErrClosed
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closing.Load() {
		return nil, ErrClosed
	}
	for _, id := range ids {
		if id == "" || (id != o.threadID && id != o.originalThreadID) {
			return nil, ErrGenerationChanged
		}
	}
	out := make([]NativeThreadState, 0, len(ids))
	for _, id := range ids {
		state := ReadNativeThreadState(ctx, o.client, id)
		if o.closing.Load() {
			return nil, ErrClosed
		}
		if id == o.threadID {
			// Goal activation can start provider work without a local turn/start
			// call. Recover its identity from the same live provider, not our
			// submission cache or a hydrated conversation/transcript.
			switch state.Status {
			case "active":
				var turns struct {
					Data []TurnRef `json:"data"`
				}
				err := o.client.Call(ctx, "thread/turns/list", map[string]any{"threadId": id, "limit": 1, "sortDirection": "desc", "itemsView": "summary"}, &turns)
				if err == nil && len(turns.Data) == 1 && turns.Data[0].Status == "inProgress" && strings.TrimSpace(turns.Data[0].ID) != "" {
					state.ActiveTurnID = turns.Data[0].ID
					o.activeTurnID = state.ActiveTurnID
				}
				// Active with no confirmed turn stays active/unknown to callers;
				// neither an old cached ID nor a goal is proof of a live turn.
				if errors.Is(err, ErrClosed) || ctx.Err() != nil || o.closing.Load() {
					state.Status = "unavailable"
				}
			case "idle":
				o.activeTurnID = ""
			}
		}
		out = append(out, state)
	}
	return out, nil
}

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
	"web-console/backends/codex"
)

// The caller is the existing, session-owned daemon connection. It is read-only
// and never starts, resumes, interrupts, or modifies a native thread.
type nativeThreadCaller interface {
	Call(context.Context, string, any, any) error
}

func (s *Server) readSessionNativeThreadState(ctx context.Context, req *sessionsv1.ReadNativeThreadStateRequest) (*sessionsv1.ReadNativeThreadStateResponse, error) {
	if uuid.Validate(req.GetSessionId()) != nil || len(req.GetThreadIds()) == 0 || len(req.GetThreadIds()) > nativeUsageMaxThreads {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid session or thread count"))
	}
	seen := make(map[string]bool, len(req.GetThreadIds()))
	for _, id := range req.GetThreadIds() {
		if uuid.Validate(id) != nil || seen[id] {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid or duplicate thread id"))
		}
		seen[id] = true
	}
	if s.sessionStore == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("session store unavailable"))
	}
	if _, err := s.sessionStore.Get(ctx, req.GetSessionId()); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("session unavailable"))
	}
	path := filepath.Join(codex.SessionHomePath(resolveSessionStateRoot(), req.GetSessionId()), "app-server-control", "app-server-control.sock")
	if info, err := os.Stat(path); err != nil || info.Mode()&os.ModeSocket == 0 {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("native session owner unavailable"))
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client, err := codex.DialSessionDaemon(ctx, path)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("native session owner unavailable"))
	}
	defer client.Close()
	return readNativeThreadStates(ctx, client, req.GetThreadIds()), nil
}

func readNativeThreadStates(ctx context.Context, client nativeThreadCaller, ids []string) *sessionsv1.ReadNativeThreadStateResponse {
	out := &sessionsv1.ReadNativeThreadStateResponse{}
	for _, id := range ids {
		row := &sessionsv1.NativeThreadState{ThreadId: id, Status: "unavailable"}
		out.Threads = append(out.Threads, row)
		if ctx.Err() != nil {
			continue
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
			continue
		}
		if read.Thread.Status.Type != "" {
			row.Status = read.Thread.Status.Type
			row.ActiveFlags = read.Thread.Status.ActiveFlags
		}
		var result struct {
			Goal *struct {
				ThreadID        string `json:"threadId"`
				Status          string `json:"status"`
				TokensUsed      int64  `json:"tokensUsed"`
				TimeUsedSeconds int64  `json:"timeUsedSeconds"`
				TokenBudget     *int64 `json:"tokenBudget"`
			} `json:"goal"`
		}
		if err := client.Call(ctx, "thread/goal/get", map[string]any{"threadId": id}, &result); err != nil || (result.Goal != nil && (result.Goal.ThreadID != id || result.Goal.Status == "")) {
			continue
		}
		row.GoalAvailable = true
		if result.Goal != nil {
			row.HasGoal = true
			row.GoalStatus = result.Goal.Status
			row.TokensUsed = result.Goal.TokensUsed
			row.TimeUsedSeconds = result.Goal.TimeUsedSeconds
			row.TokenBudget = result.Goal.TokenBudget
		}
	}
	return out
}

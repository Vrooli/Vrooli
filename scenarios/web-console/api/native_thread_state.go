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
	"web-console/internal/sessionstore"
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
	meta, err := s.nativeObservationMetadata(ctx, req.GetSessionId())
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if meta.LaunchMode == sessionstore.LaunchModeCodexAppServer {
		states, err := s.managedCodex.ReadThreadStates(ctx, meta, req.GetThreadIds())
		if err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("managed native session owner binding unavailable"))
		}
		return nativeThreadStatesProto(states), nil
	}
	path := filepath.Join(codex.SessionHomePath(resolveSessionStateRoot(), req.GetSessionId()), "app-server-control", "app-server-control.sock")
	if info, err := os.Stat(path); err != nil || info.Mode()&os.ModeSocket == 0 {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("native session owner unavailable"))
	}
	client, err := codex.DialSessionDaemon(ctx, path)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("native session owner unavailable"))
	}
	defer client.Close()
	return readNativeThreadStates(ctx, client, req.GetThreadIds()), nil
}

func readNativeThreadStates(ctx context.Context, client nativeThreadCaller, ids []string) *sessionsv1.ReadNativeThreadStateResponse {
	states := make([]codex.NativeThreadState, 0, len(ids))
	for _, id := range ids {
		states = append(states, codex.ReadNativeThreadState(ctx, client, id))
	}
	return nativeThreadStatesProto(states)
}

func nativeThreadStatesProto(states []codex.NativeThreadState) *sessionsv1.ReadNativeThreadStateResponse {
	out := &sessionsv1.ReadNativeThreadStateResponse{}
	for _, state := range states {
		row := &sessionsv1.NativeThreadState{ThreadId: state.ThreadID, Status: state.Status, ActiveFlags: state.ActiveFlags, ActiveTurnId: state.ActiveTurnID, GoalAvailable: state.GoalAvailable}
		if state.Goal != nil {
			row.HasGoal = true
			row.GoalStatus = state.Goal.Status
			row.TokensUsed = state.Goal.TokensUsed
			row.TimeUsedSeconds = state.Goal.TimeUsedSeconds
			row.TokenBudget = state.Goal.TokenBudget
		}
		out.Threads = append(out.Threads, row)
	}
	return out
}

package main

import (
	"context"

	"web-console/backends/codex"
	sessionsH "web-console/handlers/sessions"
)

// sessionDaemons is the Web Console side of the per-session Codex app-server
// daemon: it finds the daemons that session homes leave running and stops them
// when their session is archived, dismissed, pruned or deleted. Process
// handling lives in the codex backend; this adapts it to the lifecycle owners.
type sessionDaemons struct {
	root func() string
}

func newSessionDaemons(root func() string) sessionsH.AgentDaemonControl {
	return sessionDaemons{root: root}
}

func (d sessionDaemons) Running(context.Context) (map[string]sessionsH.AgentDaemonUsage, error) {
	daemons, err := codex.SessionDaemons(d.root())
	if err != nil {
		return nil, err
	}
	out := make(map[string]sessionsH.AgentDaemonUsage, len(daemons))
	for id, usage := range codex.SessionDaemonUsage(daemons) {
		out[id] = sessionsH.AgentDaemonUsage{Processes: usage.Processes, MemoryBytes: usage.MemoryBytes}
	}
	return out, nil
}

func (d sessionDaemons) Stop(ctx context.Context, sessionID string) (sessionsH.AgentDaemonUsage, error) {
	usage, err := codex.StopSessionDaemon(ctx, d.root(), sessionID, codex.DefaultDaemonStopGrace)
	return sessionsH.AgentDaemonUsage{Processes: usage.Processes, MemoryBytes: usage.MemoryBytes}, err
}

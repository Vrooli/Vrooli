package main

import "time"

// Every terminal Web Console spawns carries WC_WEB_CONSOLE_SESSION_ID in its
// environment, and every process started inside that terminal inherits it. That
// makes the running agent process an authoritative, always-available answer to
// "which pane does this agent belong to" — one that does not depend on the
// agent cooperating by writing an identity record, and does not depend on a
// hook being registered correctly.
//
// This is the portable half of that lookup. The platform half is small enough
// to keep behind a build tag: Linux reads /proc, and hosts without a supported
// mechanism report nothing, which degrades identification back to the hook
// rather than breaking it.

// agentProcess is one running process that Web Console can attribute to a
// session it owns.
type agentProcess struct {
	// PID is retained for log lines; attribution never keys on it because pids
	// are reused.
	PID int
	// SessionID is the owning web-console session, read from the process
	// environment rather than inferred.
	SessionID string
	// WorkingDir is the process's current directory, used to locate the
	// directory an agent writes its transcript into.
	WorkingDir string
	// StartedAt bounds which transcripts this process could possibly have
	// written: a process cannot have produced a message before it existed.
	StartedAt time.Time
}

// discoverAgentProcesses returns the agent processes belonging to web-console
// sessions. It is a variable so tests can substitute a fixture without a
// process table, and so the platform implementation stays swappable.
var discoverAgentProcesses = platformDiscoverAgentProcesses

// agentProcessSessionEnvKey is the environment variable every session's PTY
// exports; see defaultSessionEnv in session_factory.go.
const agentProcessSessionEnvKey = "WC_WEB_CONSOLE_SESSION_ID"

// agentProcessNames are the executables worth inspecting. Keeping the list
// explicit avoids reading the environment of every process on the host, which
// would be both slow and needlessly invasive.
var agentProcessNames = map[string]bool{
	"claude":   true,
	"codex":    true,
	"grok":     true,
	"opencode": true,
}

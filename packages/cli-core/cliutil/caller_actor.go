package cliutil

import (
	"os"
	"os/user"
	"strings"
)

// callerAgentSignals names the agent runtimes an audit record can identify. Each
// is read directly so VROOLI_CALLER=human cannot hide a runtime from the record;
// DetectCallerKind honours that override because it answers a different question.
var callerAgentSignals = []struct {
	name, signal, value, session string
}{
	{name: "agent-manager run", signal: EnvIdentityToken, session: AgentSessionEnv},
	{name: "swarm-manager session", signal: "VROOLI_SWARM_MANAGER_SESSION_ID", session: "VROOLI_SWARM_MANAGER_SESSION_ID"},
	{name: "sandbox", signal: "VROOLI_SANDBOX_ID", session: "VROOLI_SANDBOX_ID"},
	{name: "claude-code", signal: "CLAUDECODE", value: "1", session: EnvClaudeCodeSessionID},
	{name: "codex", signal: EnvCodexThreadID, session: EnvCodexThreadID},
	{name: "grok", signal: "GROK_AGENT", value: "1", session: AgentSessionEnv},
	{name: "antigravity", signal: "ANTIGRAVITY_AGENT", value: "1", session: "ANTIGRAVITY_CONVERSATION_ID"},
}

// ResolveCallerActor names who is performing a host change, for an audit record.
// An agent signal always wins. onBehalfOf is what a trusted caller, such as Web
// Console after verifying its human, says; from an agent it is recorded as a
// claim. This is attribution, not authentication: a same-user process that scrubs
// its environment can still pass as local, so it must never authorize anything.
func ResolveCallerActor(onBehalfOf string) string {
	onBehalfOf = strings.TrimSpace(onBehalfOf)
	if agent := callerAgent(); agent != "" {
		if onBehalfOf != "" {
			return agent + " (claiming to act for " + onBehalfOf + ")"
		}
		return agent
	}
	if onBehalfOf != "" {
		return "operator " + onBehalfOf
	}
	if current, err := user.Current(); err == nil && current.Username != "" {
		return "local user " + current.Username + " (no agent signal)"
	}
	return "unidentified local caller (no agent signal)"
}

func callerAgent() string {
	for _, known := range callerAgentSignals {
		found := strings.TrimSpace(os.Getenv(known.signal))
		if found == "" || (known.value != "" && found != known.value) {
			continue
		}
		if session := strings.TrimSpace(os.Getenv(known.session)); session != "" && known.signal != EnvIdentityToken {
			return "agent " + known.name + " session " + session
		}
		return "agent " + known.name
	}
	if IsLikelyAgentContext() {
		return "agent " + DetectCallerKind().String()
	}
	return ""
}

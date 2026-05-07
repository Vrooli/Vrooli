package agentcontext

import "strings"

var agentContextEnvKeys = []string{
	"VROOLI_SANDBOX_ID",
	"VROOLI_SANDBOX_MERGED",
	"VROOLI_AGENT_MANAGER_RUN_ID",
	"VROOLI_SWARM_MANAGER_SESSION_ID",
	"VROOLI_AGENT_IDENTITY_TOKEN",
	"VROOLI_AGENT_MANAGER_API_BASE",
}

// IsAgentControlled reports whether env belongs to an agent-managed or
// workspace-sandboxed process. Values may be supplied as KEY=value entries or
// bare keys; empty values are ignored.
func IsAgentControlled(env []string) bool {
	values := map[string]string{}
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			key, value = entry, "1"
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	for _, key := range agentContextEnvKeys {
		if values[key] != "" {
			return true
		}
	}
	return false
}

package cliutil

import (
	"os"
	"strings"
)

var agentContextEnvKeys = []string{
	"VROOLI_SANDBOX_ID",
	"VROOLI_SANDBOX_MERGED",
	"VROOLI_AGENT_MANAGER_RUN_ID",
	"VROOLI_SWARM_MANAGER_SESSION_ID",
	"VROOLI_AGENT_IDENTITY_TOKEN",
}

func IsAgentControlledContext() bool {
	for _, key := range agentContextEnvKeys {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return true
		}
	}
	return false
}

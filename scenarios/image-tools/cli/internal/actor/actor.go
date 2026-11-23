// Package actor names who is running the CLI, for provenance the API records
// (an enabled-state override, a job rating). It is a declared label, not an
// authenticated identity: the API trusts the local caller, so this only keeps
// the record honest about which agent or person said it.
package actor

import (
	"os"
	"strings"
)

// Name returns the declared actor: IMAGE_TOOLS_ACTOR, else the agent identity
// the harness exports (AI_AGENT), else the OS user, else "cli".
func Name() string { return fromEnv(os.Getenv) }

func fromEnv(get func(string) string) string {
	for _, key := range []string{"IMAGE_TOOLS_ACTOR", "AI_AGENT", "USER"} {
		if v := strings.TrimSpace(get(key)); v != "" {
			return v
		}
	}
	return "cli"
}

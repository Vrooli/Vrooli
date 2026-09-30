package main

import (
	"fmt"
	"os"
	"strings"
)

// rejectRunIdentityLifecycleCommand avoids sending an operator-only request
// from an agent-manager run. The API validates the token and remains the
// authority; this client-side check makes the safe boundary clear before a
// workflow wastes a turn on a request that will be denied. stop, continue and
// wake are not listed: the API admits them for a run's own lineage (an
// orchestrator's direct children, a worker's parked parent) and denies the rest.
func rejectRunIdentityLifecycleCommand(subcommand string) error {
	token, _ := os.LookupEnv("VROOLI_AGENT_IDENTITY_TOKEN")
	if strings.TrimSpace(token) == "" {
		return nil
	}

	operatorOnly := map[string]struct{}{
		"apply-investigation": {},
		"approve":             {},
		"delete":              {},
		"investigate":         {},
		"quiesce":             {},
		"recover":             {},
		"reject":              {},
		"sandbox-sync":        {},
		"stop-all":            {},
		"stop-by-tag":         {},
	}
	if _, restricted := operatorOnly[subcommand]; !restricted {
		return nil
	}

	return fmt.Errorf("agent-manager run %s requires an operator context; run identities may inspect runs and use `run park`, but cannot perform this lifecycle operation", subcommand)
}

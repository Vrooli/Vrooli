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
// quiesce is not listed either: the API admits an effort's running orchestrator
// for the scenario its effort targets (a baseline promote) and refuses other
// runs with the reason. delete, stop-by-tag and stop-all are not listed: the API
// admits an approved run-deletion or agent.manage grant (stop-all: one covering
// every run) and otherwise answers with the `vrooli agent authorize` command.
func rejectRunIdentityLifecycleCommand(subcommand string) error {
	token, _ := os.LookupEnv("VROOLI_AGENT_IDENTITY_TOKEN")
	if strings.TrimSpace(token) == "" {
		return nil
	}

	operatorOnly := map[string]struct{}{
		"apply-investigation": {},
		"approve":             {},
		"reject":              {},
		"sandbox-sync":        {},
	}
	if _, restricted := operatorOnly[subcommand]; !restricted {
		return nil
	}

	return fmt.Errorf("agent-manager run %s records an operator decision (run review, an investigation's approved recommendations, or the sandbox's review result), so no agent grant covers it; the operator answers it in Agent Manager or the decision inbox", subcommand)
}

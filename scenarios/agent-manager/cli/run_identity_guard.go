package main

import (
	"fmt"
	"os"
	"strings"
)

// rejectRunIdentityLifecycleCommand avoids sending an operator-only request
// from an agent-manager run. The API validates the token and remains the
// authority; this client-side check makes the safe boundary clear before a
// workflow wastes a turn on a request that will be denied.
func rejectRunIdentityLifecycleCommand(subcommand string) error {
	if strings.TrimSpace(os.Getenv("VROOLI_AGENT_IDENTITY_TOKEN")) == "" {
		return nil
	}

	operatorOnly := map[string]struct{}{
		"apply-investigation": {},
		"approve":             {},
		"continue":            {},
		"create":              {},
		"delete":              {},
		"investigate":         {},
		"quiesce":             {},
		"recover":             {},
		"reject":              {},
		"sandbox-sync":        {},
		"stop":                {},
		"stop-all":            {},
		"stop-by-tag":         {},
		"wake":                {},
	}
	if _, restricted := operatorOnly[subcommand]; !restricted {
		return nil
	}

	return fmt.Errorf("agent-manager run %s requires an operator context; run identities may inspect runs and use `run park`, but cannot perform this lifecycle operation", subcommand)
}

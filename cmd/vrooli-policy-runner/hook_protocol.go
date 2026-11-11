package main

import (
	"fmt"
	"io"

	agentharness "github.com/vrooli/agentharness"
)

// codexDenyExit is the only exit status Codex reads as "block this tool call".
// Any other non-zero status is a non-blocking hook failure, so the generic
// deny class (20) would let the call run. Verified on Codex 0.167 with a
// throwaway CODEX_HOME: exit 20 ran the command, exit 2 stopped it.
const codexDenyExit = 2

// answerHook reports a decision the way the calling harness reads it.
//
// Codex parses a hook's stdout as its own hook-output JSON; Vrooli's decision
// document there marks every call "PreToolUse Failed". Its verdict is the exit
// status plus a stderr reason, so nothing is written to stdout. Other runners
// keep the decision document on stdout and the stable exit classes (zero
// continue, ten ask, twenty deny, thirty unavailable); their blocking
// convention is not verified here.
func answerHook(runner string, decision agentharness.Decision, stdout, stderr io.Writer) error {
	code := agentharness.ExitCode(decision.Action)
	if runner != "codex" {
		if err := writeJSON(stdout, decision); err != nil {
			return err
		}
		if code != 0 {
			return exitError{code: code}
		}
		return nil
	}
	switch decision.Action {
	case agentharness.ActionDeny:
		fmt.Fprintf(stderr, "Blocked by Vrooli agent policy: %s\n", decision.Reason)
		return exitError{code: codexDenyExit}
	case agentharness.ActionAsk:
		// Codex hooks cannot prompt; the call proceeds and the reason is shown.
		fmt.Fprintf(stderr, "Vrooli agent policy asks for operator confirmation, which Codex hooks cannot show: %s\n", decision.Reason)
	}
	return nil
}

// allowOnPanic turns a panic in a hook into an allow with a warning on stderr.
func allowOnPanic(stderr io.Writer, err *error) {
	if recovered := recover(); recovered != nil {
		fmt.Fprintf(stderr, "vrooli-policy-runner hook: allowed after internal error: %v\n", recovered)
		*err = nil
	}
}

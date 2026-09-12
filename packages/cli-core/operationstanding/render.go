// Package operationstanding renders the provider-owned durable-operation
// contract. It deliberately makes no lifecycle decisions: callers display the
// standing that the owner returned instead of inferring progress from elapsed
// time, phase transitions, or transport behavior.
package operationstanding

import (
	"fmt"
	"io"
)

// Standing is the generated OperationStanding's read-only surface. Keeping the
// renderer structural avoids making cli-core depend on a scenario/proto module.
type Standing interface {
	GetLifecycle() string
	GetActivePhase() string
	GetEtaKnown() bool
	GetEstimatedRemainingSeconds() int32
	GetDirective() string
	GetReattachCommand() string
}

// WriteText emits the shared human view used by durable-operation CLIs. JSON
// callers should emit the proto response unchanged so automation receives the
// same typed fields.
func WriteText(w io.Writer, standing Standing) error {
	if standing == nil {
		return nil
	}
	if _, err := fmt.Fprintf(w, "  lifecycle: %s\n", standing.GetLifecycle()); err != nil {
		return err
	}
	if phase := standing.GetActivePhase(); phase != "" {
		if _, err := fmt.Fprintf(w, "  active phase: %s\n", phase); err != nil {
			return err
		}
	}
	if standing.GetEtaKnown() {
		if _, err := fmt.Fprintf(w, "  estimated remaining: ~%ds\n", standing.GetEstimatedRemainingSeconds()); err != nil {
			return err
		}
	}
	if directive := standing.GetDirective(); directive != "" {
		if _, err := fmt.Fprintf(w, "  action: %s\n", directive); err != nil {
			return err
		}
		if directive == "wait" {
			if _, err := fmt.Fprintln(w, "  provider owns progress; wait once, do not poll."); err != nil {
				return err
			}
		}
	}
	if command := standing.GetReattachCommand(); command != "" {
		if _, err := fmt.Fprintf(w, "  reattach once: %s\n", command); err != nil {
			return err
		}
	}
	return nil
}

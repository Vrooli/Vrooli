// Package elevation delegates host service recovery to the control plane.
// Runtime code never invokes sudo or interprets a private sudoers grant.
package elevation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/vrooli/vrooli/internal/privilegebroker"
)

type State string

const (
	Granted     State = "granted"
	NeedsSetup  State = "needs_setup"
	Unsupported State = "unsupported"
	Refused     State = "refused"
)

type Outcome struct {
	State     State  `json:"state"`
	Command   string `json:"command,omitempty"`
	Reason    string `json:"reason,omitempty"`
	GrantName string `json:"grantName,omitempty"`
}
type Action string

const (
	ServiceStart   Action = "service.start"
	ServiceRestart Action = "service.restart"
	ServiceStop    Action = "service.stop"
)

type Executor interface {
	CombinedOutput(context.Context, string, ...string) ([]byte, error)
}

var osNameFn = func() string { return runtime.GOOS }

func Run(ctx context.Context, executor Executor, action Action, unit string) (Outcome, []byte, error) {
	if executor == nil {
		return Outcome{State: Refused, Reason: "no command executor configured"}, nil, errors.New("elevation: nil executor")
	}
	if osNameFn() != "linux" {
		return Outcome{State: Unsupported, Reason: "broker service recovery is Linux/systemd-only", GrantName: "privilege_broker"}, nil, nil
	}
	unit = strings.TrimSpace(unit)
	verb := map[Action]string{ServiceStart: "start", ServiceRestart: "restart", ServiceStop: "stop"}[action]
	if !privilegebroker.ServiceRecoveryAllowed(verb, unit) {
		return Outcome{State: Refused, Reason: "service action is outside the control-plane recovery registry", GrantName: "privilege_broker"}, nil, fmt.Errorf("elevation: unsupported service action %q for %q", action, unit)
	}
	args := []string{"host", "service", verb, unit, "--json"}
	command := "vrooli " + strings.Join(args, " ")
	output, runErr := executor.CombinedOutput(ctx, "vrooli", args...)
	var result privilegebroker.Result
	decodeErr := json.Unmarshal(output, &result)
	if decodeErr == nil && result.Version == privilegebroker.ProtocolVersion && result.Action == privilegebroker.ActionServiceRecover {
		if runErr == nil && result.Status == "verified" && result.Evidence.Active {
			return Outcome{State: Granted, Command: command, GrantName: "privilege_broker"}, output, nil
		}
		if result.Code != "action_not_allowed" {
			if runErr == nil {
				runErr = fmt.Errorf("host service recovery failed: %s", result.Code)
			}
			return Outcome{State: Refused, Command: command, Reason: runErr.Error(), GrantName: "privilege_broker"}, output, runErr
		}
	}
	// An old CLI/broker or a missing local broker is a setup gap, never a
	// reason to fall back to direct systemctl/sudo or to claim recovery.
	reason := "control-plane service recovery is unavailable or returned an invalid receipt"
	if runErr != nil {
		reason = runErr.Error()
	}
	return Outcome{State: NeedsSetup, Command: "vrooli setup broker --sudo-mode=ask", Reason: reason, GrantName: "privilege_broker"}, output, runErr
}

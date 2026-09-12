package vps

import (
	"context"
	"encoding/json"
	"path"
	"strings"
	"time"

	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
)

// Prober is the read-only probe surface every live-state, log and file
// inspection uses. It issues typed vrooli verbs and target-owner observation
// operations through reach; nothing here composes a shell string, and nothing
// here is effectful.
type Prober struct {
	Reach  reach.Reach
	Target identity.TargetRef
}

// DefaultProbeTimeout bounds one probe.
const DefaultProbeTimeout = 45 * time.Second

// Observe translates the legacy probe vocabulary into the target-owner's
// typed observation contract. The legacy name is local compatibility only and
// never becomes a remotely executable program.
func (p Prober) Observe(ctx context.Context, program string, args ...string) (reach.Result, error) {
	cmd, err := reach.NewObservation(program, args...)
	if err != nil {
		return reach.Result{}, err
	}
	cmd.Timeout = DefaultProbeTimeout
	result, err := p.Reach.Exec(ctx, p.Target, cmd)
	if err != nil {
		return result, err
	}
	// The target owner returns a stable JSON envelope so the transport never
	// exposes arbitrary command output as an untyped relay result.
	var envelope struct {
		Result struct {
			Stdout string `json:"stdout"`
			Stderr string `json:"stderr"`
			Exit   int    `json:"exit_code"`
		} `json:"result"`
	}
	if json.Unmarshal([]byte(result.Stdout), &envelope) == nil && envelope.Result.Stdout != "" {
		result.Stdout = envelope.Result.Stdout
		result.Stderr = envelope.Result.Stderr
		result.ExitCode = envelope.Result.Exit
	}
	return result, nil
}

// Verb runs one read-only vrooli verb in the bound workdir.
func (p Prober) Verb(ctx context.Context, verb string, args ...string) (reach.Result, error) {
	return p.Reach.Exec(ctx, p.Target, reach.Command{Verb: verb, Args: args, RequiredScope: "vrooli:read", Timeout: DefaultProbeTimeout})
}

// Effect runs one effectful vrooli verb (a lifecycle verb such as
// `scenario restart`) with the write scope. Host repairs never go through
// here; they are privilege-broker actions via `cloud-target host repair`.
func (p Prober) Effect(ctx context.Context, verb string, args ...string) (reach.Result, error) {
	return p.Reach.Exec(ctx, p.Target, reach.Command{Verb: verb, Args: args, RequiredScope: "vrooli:write", Effectful: true, Timeout: 2 * time.Minute})
}

// HomeDir is the login home the observation programs read user files from
// (`authorized_keys`). It follows the Ubuntu layout for the bound user.
func HomeDir(user string) string {
	user = strings.TrimSpace(user)
	if user == "" || user == "root" {
		return "/root"
	}
	return path.Join("/home", user)
}

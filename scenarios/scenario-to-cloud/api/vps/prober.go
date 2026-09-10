package vps

import (
	"context"
	"path"
	"strings"
	"time"

	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
)

// Prober is the read-only probe surface every live-state, log and file
// inspection uses. It issues typed vrooli verbs and bounded host observation
// programs through reach; nothing here composes a shell string, and nothing
// here is effectful.
type Prober struct {
	Reach  reach.Reach
	Target identity.TargetRef
}

// DefaultProbeTimeout bounds one probe.
const DefaultProbeTimeout = 45 * time.Second

// Observe runs one host observation program (reach.ObservationPrograms).
func (p Prober) Observe(ctx context.Context, program string, args ...string) (reach.Result, error) {
	return p.Reach.Exec(ctx, p.Target, reach.Command{Program: program, Args: args, RequiredScope: "vrooli:read", Timeout: DefaultProbeTimeout})
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

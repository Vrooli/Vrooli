package remote

import (
	"context"

	"github.com/vrooli/cli-core/cliutil"

	"treasury/cli/internal/clock"
	"treasury/cli/internal/envx"
)

// Dependencies supplies host-facing state to the remote command domain.
type Dependencies struct {
	Environment       envx.Reader
	Clock             clock.Clock
	ResolveAgentGrant func(context.Context, string, string) (cliutil.AgentCredentials, error)
	ResolveSessionKey func(context.Context, string, string) (string, error)
}

// ProductionDependencies binds remote commands to the current process and
// Agent Manager credential authority.
func ProductionDependencies() Dependencies {
	return Dependencies{
		Environment:       envx.OS{},
		Clock:             clock.System{},
		ResolveAgentGrant: cliutil.ResolveAgentCredentialsForOperation,
		ResolveSessionKey: cliutil.ResolveTreasurySessionKey,
	}
}

func (d Dependencies) withDefaults() Dependencies {
	if d.Environment == nil {
		d.Environment = envx.Empty{}
	}
	if d.Clock == nil {
		d.Clock = clock.System{}
	}
	if d.ResolveAgentGrant == nil {
		d.ResolveAgentGrant = cliutil.ResolveAgentCredentialsForOperation
	}
	if d.ResolveSessionKey == nil {
		d.ResolveSessionKey = cliutil.ResolveTreasurySessionKey
	}
	return d
}

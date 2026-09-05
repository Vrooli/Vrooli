// Package credentials owns the Bridge CLI's owner-facing credential grant
// commands. The manifest defines the public command shape; handlers bind the
// generated Connect client without ever rendering credential values.
package credentials

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "credentials"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"CredentialGrantService.CreateGrant":    h.grant,
		"CredentialGrantService.ListGrants":     h.list,
		"CredentialGrantService.RevokeGrant":    h.revoke,
		"CredentialGrantService.RotateAddress":  h.rotate,
		"CredentialGrantService.SyncNodeGrants": h.sync,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("credentials: load from manifest: %w", err)
	}
	return group, nil
}

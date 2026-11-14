package policy

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
	policyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy/policyconnect"
)

const GroupName = "policy"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	group, err := cliapp.LoadProtoPrimitiveGroupFromManifest(core, policyconnect.PolicyServiceName, manifest, GroupName, cliapp.ProtoBindingOptions{}, map[string]bool{
		"GetPolicy": true,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("policy: load from manifest: %w", err)
	}
	return group, nil
}

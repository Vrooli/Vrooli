package walletrecovery

import (
	"os"
	"scenario-to-cloud/cli/walletrecovery"

	"github.com/vrooli/cli-core/cliapp"
)

func Register() cliapp.SubcommandGroup {
	commands := walletrecovery.Commands{Out: os.Stdout}
	return cliapp.SubcommandGroup{
		Name:        "wallet-recovery",
		Description: "Bridge-lineage-bound off-host Treasury recovery through DBM, followed by a post-restart proof on the same bundle",
		NeedsAPI:    false,
		Subcommands: []cliapp.Command{
			{Name: "rotate", Description: "Publish and verify a signed Treasury revocation tombstone through an operator-selected off-host DBM repository before committing key deletion", NeedsAPI: false, Run: commands.Rotate},
			{Name: "restore", Description: "Preflight the durable Bridge Node/Machine/scope, fetch the newest exact DBM snapshot and restore the key-only bundle through Treasury", NeedsAPI: false, Run: commands.Restore},
			{Name: "prove-after-restart", Description: "Recheck Bridge lineage/readiness and DBM snapshot identity before verifying the witnessed address and Treasury control signature", NeedsAPI: false, Run: commands.ProveAfterRestart},
		},
	}
}

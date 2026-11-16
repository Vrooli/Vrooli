package walletrecovery

import (
	"fmt"
	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "wallet-recovery"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"WalletRecoveryAttest":            h.attest,
		"WalletRecoveryReservationFinish": h.finish,
		"WalletRecoveryReadLeaseAcquire":  h.readLeaseAcquire,
		"WalletRecoveryReadLeaseRenew":    h.readLeaseRenew,
		"WalletRecoveryReadLeaseRelease":  h.readLeaseRelease,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("wallet recovery: load from manifest: %w", err)
	}
	return group, nil
}

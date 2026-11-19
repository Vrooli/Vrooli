package runtimeapp

import (
	sharedscenariocli "github.com/vrooli/api-core/scenariocli"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// scenarioCLIInstallSchemaDigest pins the stored shape of the per-binary
// install metadata that records a scenario CLI's fingerprint. A deliberate
// change must bump the store's write version and re-pin.
const scenarioCLIInstallSchemaDigest = "67090bc95ca5c06f929782e5f68a27684541f48bc4599cd5c4f6f6e31618b58f"

// scenarioCLIInstallStateFormat describes the scenario CLI install metadata.
//
// The metadata file family (bin/*.build.meta) is declared and snapshotted by
// installation-inventory, which owns the installation record and the sidecar
// layout. Declaring it a second time would snapshot the same files twice, so
// this store is contract-only: it names the shape and vocabulary of a file
// another store carries.
func scenarioCLIInstallStateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(sharedscenariocli.PersistentTypes()...)
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:         "scenario-cli-install",
		Write:        1,
		Reads:        []stateformat.Version{1},
		Schemas:      map[stateformat.Version]string{1: digest},
		ContractOnly: true,
	}, nil
}

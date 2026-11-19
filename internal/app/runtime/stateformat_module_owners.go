package runtimeapp

import (
	"reflect"

	"github.com/vrooli/api-core/trustposture"

	"github.com/vrooli/vrooli/internal/credentialauthority"
	"github.com/vrooli/vrooli/internal/stateformat"
	"github.com/vrooli/vrooli/internal/tools/hyperframes"
	"github.com/vrooli/vrooli/internal/tools/pinnedrelease"
	harness "github.com/vrooli/vrooli/packages/harness-lifecycle"
)

// The owners below are declared here because their types live in shared
// modules. Each descriptor names the compiled shape of what the owner writes;
// none of them declares state paths, because the files sit in caller-chosen or
// host directories rather than the runtime home.

const (
	toolInstallationSchemaDigest   = "53ba977e8929b9926927722d6a581173bf4e201da07082ecdbdd4bce30dbeb97"
	breakGlassSchemaDigest         = "a71e7f45f97290ec61cf236d687f47745055f566598b281aaaabcbc9e56dd175"
	credentialRecoverySchemaDigest = "68fec36f48bfe8e7fc80caed4911fa019b9be81e13303b2e16725610b5e7b1c5"
	harnessLifecycleSchemaDigest   = "c9a5bb96434397bda10758c866569eb0ba58b3e095a5b5d318edfa579fa74ddc"
)

func toolInstallationStateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(
		reflect.TypeFor[pinnedrelease.Receipt](),
		reflect.TypeFor[hyperframes.Installation](),
	)
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:         "tool-installation",
		Write:        1,
		Reads:        []stateformat.Version{1},
		Schemas:      map[stateformat.Version]string{1: digest},
		ContractOnly: true,
	}, nil
}

func breakGlassStateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(
		reflect.TypeFor[trustposture.KeyMaterial](),
		reflect.TypeFor[trustposture.BreakGlassClaims](),
		reflect.TypeFor[trustposture.BreakGlassBinding](),
		reflect.TypeFor[trustposture.KeyStatus](),
	)
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:         "break-glass",
		Write:        1,
		Reads:        []stateformat.Version{1},
		Schemas:      map[stateformat.Version]string{1: digest},
		ContractOnly: true,
	}, nil
}

func credentialRecoveryStateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(
		reflect.TypeFor[credentialauthority.Status](),
		reflect.TypeFor[credentialauthority.Authority](),
		reflect.TypeFor[credentialauthority.RecoveryManifest](),
		reflect.TypeFor[credentialauthority.RecoveryEntry](),
		reflect.TypeFor[credentialauthority.RecoveryReceipt](),
		reflect.TypeFor[credentialauthority.RecoveryEpoch](),
	)
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:         "credential-recovery",
		Write:        1,
		Reads:        []stateformat.Version{1},
		Schemas:      map[stateformat.Version]string{1: digest},
		ContractOnly: true,
	}, nil
}

func harnessLifecycleStateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(
		reflect.TypeFor[harness.HarnessReport](),
		reflect.TypeFor[harness.Inventory](),
		reflect.TypeFor[harness.HarnessSource](),
		reflect.TypeFor[harness.HarnessCheck](),
		reflect.TypeFor[harness.CleanupPlan](),
	)
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:         "harness-lifecycle",
		Write:        1,
		Reads:        []stateformat.Version{1},
		Schemas:      map[stateformat.Version]string{1: digest},
		ContractOnly: true,
	}, nil
}

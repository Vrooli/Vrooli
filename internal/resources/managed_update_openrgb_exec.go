package resources

import (
	"context"
	"errors"
	"os"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

// observeOpenRGBFrozenExecWitness verifies access and the native launch context
// against frozen contract bytes. It deliberately does not materialize the
// resource config before the witness and durable launch claim.
func (c *Controller) observeOpenRGBFrozenExecWitness(ctx context.Context, contract ManagedUpdateContract, invocation managedUpdateInvocation, fence managedUpdateExecFence, ref platform.ServiceRef, incarnation string) (openRGBExecWitness, error) {
	role, err := c.openRGBFrozenExecRole(ctx, fence, contract)
	if err != nil {
		return openRGBExecWitness{}, err
	}
	if len(contract.ConfigBytes) == 0 {
		return openRGBExecWitness{}, errors.New("OpenRGB admitted role has no frozen config bytes")
	}
	if role.NativePolicy != nil {
		return observeOpenRGBNativeAdmittedWitness(ctx, *role.NativePolicy, contract, invocation, role.Catalog, contract.ConfigBytes, openRGBWitnessAdmission{Policy: fence.OpenRGBPolicy, Port: fence.Intent.Runtime.Port, ContractDigest: fence.Intent.ContractDigest, ClaimSHA256: fence.openRGBClaimDigest(ref)}, ref, incarnation, openRGBExecContext, hostreqkit.ObserveNativeOpenRGBDevicePolicy)
	}
	return observeOpenRGBExecWitness(ctx, role.Policy, contract, invocation, role.Catalog, contract.ConfigBytes, fence, ref, incarnation, os.Geteuid(), openRGBExecContext, hostreqkit.ObserveLinuxOpenRGBDevicePolicy)
}

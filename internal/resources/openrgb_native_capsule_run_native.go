//go:build darwin || windows

package resources

import (
	"context"
	"errors"
	"os"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/process"
)

// RunOpenRGBNativeCapsule is an early, bounded command dispatcher. It never
// initializes general CLI/operator state and does not read operator HOME.
func RunOpenRGBNativeCapsule(ctx context.Context, path string) error {
	if ctx == nil {
		return errors.New("native capsule context is required")
	}
	capsule, err := readOpenRGBNativeCapsule(ctx, path)
	if err != nil {
		return err
	}
	if err := prepareOpenRGBNativeWrapperCustody(ctx, capsule.Intent); err != nil {
		return err
	}
	admission, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return executeOpenRGBNativeCapsule(admission, capsule, openRGBNativeCapsuleExecOwner{
		lookup: platform.LookupOwnedService, incarnation: process.Incarnation, selfImage: platform.OpenOwnedLauncherImage, normalizeEnvironment: normalizeOpenRGBNativeEnvironment,
		connect: func(ctx context.Context, c openRGBNativeCapsule) (openRGBNativeCapsuleSession, error) {
			return dialOpenRGBNativeClaimOwner(ctx, c)
		},
		prepareImage: prepareOpenRGBNativeCapsuleImage,
		confineTransport: func(ctx context.Context, m openRGBNativeMaterial) (string, error) {
			return hostreqkit.ConfineNativeOpenRGBBackendProcess(ctx, m.StagePath)
		},
		observe: func(ctx context.Context, m openRGBNativeMaterial, ref platform.ServiceRef, incarnation string) (openRGBExecWitness, error) {
			return observeOpenRGBNativeAdmittedWitness(ctx, m.NativePolicy, m.Contract, m.Invocation, m.Catalog, m.Contract.ConfigBytes, openRGBWitnessAdmission{Policy: &m.Policy, Port: m.Runtime.Port, ContractDigest: m.Contract.Digest, ClaimSHA256: m.ClaimSHA256}, ref, incarnation, openRGBExecContext, hostreqkit.ObserveNativeOpenRGBDevicePolicy)
		},
		prepareChild: func(file *os.File, invocation managedUpdateInvocation) (openRGBNativePreparedChild, error) {
			child, err := platform.PrepareOwnedImageChild(file, append([]string{invocation.Path}, invocation.Args...), invocation.Env, invocation.Dir)
			if err != nil {
				return nil, err
			}
			return openRGBPlatformPreparedChild{child}, nil
		},
		verifyProcessOnly: platform.VerifyDarwinNoChildrenProcess,
		materialize:       materializeOpenRGBNativeCapsule, chdir: os.Chdir, replace: platform.ReplaceProcessImage,
	})
}

type openRGBPlatformPreparedChild struct{ *platform.OwnedImageChild }

func (c openRGBPlatformPreparedChild) identity() openRGBNativeExecutionChild {
	return openRGBNativeExecutionChild{PID: c.PID, Incarnation: c.Incarnation, ImageID: c.ImageID, ImageHandle: c.ImageHandle, SecurityProof: c.SecurityProof, NativeAccount: c.NativeAccount, ContainmentID: c.ContainmentID}
}

package resources

import (
	"context"
	"errors"
	"fmt"
	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"slices"
)

func observeOpenRGBNativeAdmittedWitness(ctx context.Context, policy hostreqkit.OpenRGBNativePolicy, contract ManagedUpdateContract, invocation managedUpdateInvocation, catalog, settings []byte, admission openRGBWitnessAdmission, ref platform.ServiceRef, incarnation string, observeContext func(platform.ServiceRef, string) (string, error), observeAccess func(context.Context, hostreqkit.OpenRGBNativePolicy) (hostreqkit.OpenRGBNativeEvidence, error)) (openRGBExecWitness, error) {
	if err := ctx.Err(); err != nil {
		return openRGBExecWitness{}, err
	}
	if err := verifyOpenRGBExecInvocation(invocation, admission.Port); err != nil {
		return openRGBExecWitness{}, err
	}
	binding, err := freezeOpenRGBNativeExecPolicy(policy, contract.Artifact.SHA256, catalog, settings)
	if err != nil {
		return openRGBExecWitness{}, err
	}
	if admission.Policy == nil || binding != *admission.Policy || binding.ConfigSHA256 != contract.ConfigSHA256 || string(settings) != invocation.ConfigContent || admission.ContractDigest != contract.Digest || !ref.NativeOwnedIdentityComplete() || ref.Native == nil || ref.Native.Platform != policy.Platform || !ref.MatchesIncarnation(incarnation) {
		return openRGBExecWitness{}, errors.New("OpenRGB native policy, configuration or containment differs from admission")
	}
	before, err := observeContext(ref, incarnation)
	if err != nil {
		return openRGBExecWitness{}, err
	}
	evidence, err := observeAccess(ctx, policy)
	if err != nil {
		return openRGBExecWitness{}, fmt.Errorf("verify native selected HID access in final account: %w", err)
	}
	if evidence.PolicyID != binding.ID || evidence.Account != binding.Account || !evidence.CurrentScopeVerified || len(evidence.SelectedPaths) != len(policy.Devices) || !updateDigest(evidence.DeviceGenerationSHA256) {
		return openRGBExecWitness{}, errors.New("OpenRGB native account did not verify all selected HID routes")
	}
	after, err := observeContext(ref, incarnation)
	if err != nil || before != after {
		return openRGBExecWitness{}, errors.Join(errors.New("OpenRGB native context changed during selected HID access observation"), err)
	}
	paths := slices.Clone(evidence.SelectedPaths)
	slices.Sort(paths)
	for i, p := range paths {
		if p == "" || i > 0 && paths[i-1] == p {
			return openRGBExecWitness{}, errors.New("OpenRGB native access witness has duplicate or empty routes")
		}
	}
	return openRGBExecWitness{Policy: binding, DeviceGenerationSHA256: evidence.DeviceGenerationSHA256, Incarnation: incarnation, ClaimSHA256: admission.ClaimSHA256, ContextSHA256: before, SelectedNodes: paths}, nil
}

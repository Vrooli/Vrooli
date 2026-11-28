//go:build linux

package hostwireless

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"slices"
	"strings"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"golang.org/x/sys/unix"
)

func runWirelessJournalOperation(ctx context.Context, root *os.Root, operation string, req WirelessOperationRequest, apply bool) (WirelessOperationResult, error) {
	r := WirelessOperationResult{Operation: operation, Profile: req.Profile, State: "prerequisites", Transaction: req.Transaction}
	if os.Geteuid() != 0 {
		wirelessResultCheck(&r, "root", false, "Inspect the root-owned transaction through the normal sudo host command")
		if apply {
			return r, hostreqkit.ErrElevationRequired
		}
		return r, nil
	}
	capability, err := acquireWirelessNamespace(ctx, root)
	if err != nil {
		return r, err
	}
	var running unix.Utsname
	if err = unix.Uname(&running); err != nil {
		return r, err
	}
	release := unix.ByteSliceToString(running.Release[:])
	capability.independentRecovery = release != nobleABI && (operation == "recover" || operation == "status")
	owner, err := openAdmittedDurableWireless(ctx, root, capability, !apply || operation == "status")
	if errors.Is(err, fs.ErrNotExist) {
		r.State = "not_staged"
		return r, nil
	}
	if err != nil {
		return r, err
	}
	defer owner.close()
	if owner.intent.ID != req.Transaction || owner.intent.Plan.Profile != req.Profile {
		return r, fmt.Errorf("requested transaction is inactive; current active profile %s transaction %s", owner.intent.Plan.Profile, owner.intent.ID)
	}
	r.Plan = &owner.intent.Plan
	if owner.intent.Context == nil {
		return r, errors.New("transaction lacks privileged admission context")
	}
	actor := wirelessActorUID()
	if actor != 0 && actor != owner.intent.Context.OperatorUID {
		return r, errors.New("transaction belongs to another invoking operator")
	}
	if release != nobleABI {
		if operation == "recover" {
			return runFallbackRecoveryWith(ctx, owner, release, apply, verifyIndependentWirelessFallback)
		}
		wirelessResultCheck(&r, "running_abi", false, "Target radio code is not running; use recover on the independently verified fallback for file-only undo")
		if operation == "status" {
			r.State = owner.phase
			return r, nil
		}
		return r, errors.New("radio effects require the exact target ABI; use file-only recovery on the verified fallback")
	}
	radio, radioErr := observeWirelessRadio(ctx, owner)
	r.Radio = &radio
	if operation == "status" {
		r.State = owner.phase
		r.Ready = radioErr == nil
		wirelessResultCheck(&r, "radio_observation", radioErr == nil, errorDetail(radioErr))
		return r, nil
	}
	if operation == "recover" {
		// Recovery is file-only compensation. Resume uses the original stage
		// request and all fresh staging prerequisites, never a stale permit.
		safe := radioErr == nil && len(radio.CandidateModulesLoaded) == 0 && !wirelessRadioPending(owner)
		wirelessResultCheck(&r, "no_live_candidate", safe, "Use explicit rollback with maintenance-window if candidate radio code is loaded or an activation is unfinished")
		r.Ready = safe
		if !apply {
			return r, nil
		}
		if !safe {
			return r, errors.New("file-only recovery cannot replace a live radio generation")
		}
		if err = owner.rollback(ctx); err != nil {
			r.State = "recovery_required"
			return r, err
		}
		r.State = "rolled_back"
		return r, nil
	}
	if operation == "rollback" {
		needed := len(radio.CandidateModulesLoaded) > 0 || wirelessRadioPending(owner)
		wirelessResultCheck(&r, "known_radio_identity", radioErr == nil, errorDetail(radioErr))
		wirelessResultCheck(&r, "maintenance_window", !needed || req.MaintenanceWindow, "A live radio rollback requires the explicitly coordinated maintenance window")
		r.Ready = wirelessChecksReady(&r)
		if !apply {
			return r, nil
		}
		if !r.Ready {
			return r, errors.New("live radio rollback requires coordinated maintenance-window")
		}
		if err = rollbackWirelessRadio(ctx, owner); err != nil {
			r.State = "recovery_required"
			return r, err
		}
		r.State = "rolled_back"
		observation, e := observeWirelessRadio(ctx, owner)
		r.Radio = &observation
		return r, e
	}
	if operation != "activate" {
		return r, errors.New("unsupported wireless journal operation")
	}
	wirelessResultCheck(&r, "published_files", owner.phase == "published", "Finish or recover any interrupted file publication before loading radio code")
	wirelessResultCheck(&r, "maintenance_window", req.MaintenanceWindow, "First radio qualification is an explicitly coordinated device effect")
	if err = freshWirelessActivationChecks(ctx, owner, &r); err != nil {
		return r, err
	}
	wirelessResultCheck(&r, "radio_consumers", radioErr == nil, errorDetail(radioErr))
	r.Ready = wirelessChecksReady(&r)
	if !apply {
		return r, nil
	}
	if !r.Ready {
		return r, errors.New("wireless activation prerequisites are not satisfied")
	}
	if err = activateWirelessRadio(ctx, owner); err != nil {
		r.State = "recovery_required"
		return r, err
	}
	observation, err := observeWirelessRadio(ctx, owner)
	r.Radio = &observation
	r.State = "activation_observed_function_unverified"
	if observation.FunctionVerified {
		r.State = "function_observed"
	}
	return r, err
}

func runFallbackRecoveryWith(ctx context.Context, owner *durableWireless, release string, apply bool, verify func(context.Context, RecoveryProofRequest) (RecoveryProof, error)) (WirelessOperationResult, error) {
	r := WirelessOperationResult{Operation: "recover", Profile: owner.intent.Plan.Profile, Transaction: owner.intent.ID, State: "prerequisites", Plan: &owner.intent.Plan}
	stored := owner.intent.Context
	if stored == nil || release == nobleABI || release != stored.Recovery.RecoveryRelease {
		return r, errors.New("file-only undo requires the exact independently proven fallback")
	}
	proof, err := verify(ctx, stored.Recovery)
	r.Recovery = &proof
	if err == nil && (!reflect.DeepEqual(proof.Request, stored.Recovery) || proof.MachineID != stored.MachineID) {
		err = errors.New("fallback proof does not match the staged machine and recovery identity")
	}
	wirelessResultCheck(&r, "independent_running_fallback", err == nil, errorDetail(err))
	r.Ready = wirelessChecksReady(&r)
	if !apply {
		return r, nil
	}
	if err != nil {
		return r, err
	}
	if err = recoverWirelessFilesOnFallback(ctx, owner); err != nil {
		r.State = "recovery_required"
		return r, err
	}
	r.State = "files_restored_on_verified_fallback"
	wirelessResultCheck(&r, "zero_radio_effects", true, "Exact target files restored without loading or unloading the independently running fallback's modules")
	return r, nil
}

func freshWirelessActivationChecks(ctx context.Context, owner *durableWireless, r *WirelessOperationResult) error {
	stored := owner.intent.Context
	if stored == nil {
		return errors.New("missing immutable wireless admission context")
	}
	if stored.ReviewedReceiptSHA256 != owner.intent.Plan.BundleReceiptSHA256 {
		return errors.New("reviewed wireless receipt binding differs")
	}
	var uname unix.Utsname
	if err := unix.Uname(&uname); err != nil {
		return err
	}
	wirelessResultCheck(r, "running_abi", unix.ByteSliceToString(uname.Release[:]) == nobleABI, "Candidate code targets only the exact reviewed running ABI")
	trust, err := EvaluateKernelTrust(ctx, "/"+owner.statePath("certificate.der"))
	if err != nil {
		return err
	}
	r.Trust = &trust
	wirelessResultCheck(r, "kernel_trust", trust.State == "verified" && trust.CertificateSHA256 == owner.intent.CertificateSHA256, trust.Issue)
	recovery, err := VerifyRecoveryProof(ctx, stored.Recovery)
	r.Recovery = &recovery
	wirelessResultCheck(r, "known_good_stock_recovery", err == nil, errorDetail(err))
	policyReq := FirmwarePolicyRequest{Root: owner.root, KernelRelease: nobleABI, Modules: slices.Clone(preparationProfiles[r.Profile].modules)}
	for _, pin := range durableFirmwarePins(r.Profile) {
		policyReq.Firmware = append(policyReq.Firmware, FirmwarePolicyTarget{Name: strings.TrimPrefix(pin.Path, "lib/firmware/"), SHA256: pin.SHA256})
	}
	policy, err := InspectFirmwarePolicy(ctx, policyReq)
	r.Firmware = &policy
	ready := err == nil && policy.StageAllowed && policy.ConfigSHA256 == stored.Firmware.ConfigSHA256 && policy.ParameterSHA256 == stored.Firmware.ParameterSHA256 && policy.BuiltinModulesSHA256 == stored.Firmware.BuiltinModulesSHA256 && policy.InitramfsSHA256 == stored.Firmware.InitramfsSHA256
	for _, selected := range policy.Selections {
		if selected.StageRequired {
			ready = false
		}
	}
	wirelessResultCheck(r, "firmware_selected", ready, strings.Join(append(policy.Reasons, errorDetail(err)), "; "))
	inv, err := hostinventory.NewIntegrityCollector(hostinventory.IntegrityCollectorOptions{}).Collect(ctx)
	if err != nil {
		return err
	}
	wirelessResultCheck(r, "secure_boot", inv.SecureBoot.Enabled && inv.SecureBoot.Error == "", "Preserve the enforcing boot policy")
	paths, err := admitWirelessRootPaths(owner.root, uint32(os.Geteuid()))
	if err != nil {
		return err
	}
	wirelessResultCheck(r, "physical_namespace", sameWirelessPaths(paths, owner.paths), "Distribution links and exact ABI headers must remain pinned")
	if err = owner.verifyProtected(ctx); err != nil {
		return fmt.Errorf("activation protected state: %w", err)
	}
	return nil
}

func sameWirelessPaths(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for name, value := range a {
		if b[name] != value {
			return false
		}
	}
	return true
}

//go:build linux

package hostwireless

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/packages/capacity"
	"golang.org/x/sys/unix"
)

type wirelessJournalContext struct {
	OperatorUID           uint32                   `json:"operator_uid"`
	SignedBundle          string                   `json:"signed_bundle"`
	ReviewedReceiptSHA256 string                   `json:"reviewed_receipt_sha256"`
	Recovery              RecoveryProofRequest     `json:"recovery_request"`
	Firmware              FirmwarePolicyEvidence   `json:"firmware_policy"`
	BootID                string                   `json:"stage_boot_id"`
	EthernetRoutes        []string                 `json:"ethernet_routes"`
	Device                string                   `json:"device"`
	StageModules          []wirelessModuleSnapshot `json:"stock_runtime_modules"`
	MachineID             string                   `json:"machine_id"`
}

type wirelessStageAdmission struct {
	result  WirelessOperationResult
	root    *os.Root
	signed  SignedBundleReceipt
	cert    *x509.Certificate
	images  []FirmwarePreflight
	context wirelessJournalContext
}

func wirelessResultCheck(r *WirelessOperationResult, name string, ok bool, detail string) {
	r.Checks = append(r.Checks, WirelessCheck{Name: name, Satisfied: ok, Detail: detail})
}
func wirelessChecksReady(r *WirelessOperationResult) bool {
	if len(r.Checks) == 0 {
		return false
	}
	for _, check := range r.Checks {
		if !check.Satisfied {
			return false
		}
	}
	return true
}
func wirelessActorUID() uint32 {
	if uid, _, ok := hostreqkit.InvokingUserIDs(); ok {
		return uint32(uid)
	}
	return uint32(os.Geteuid())
}
func errorDetail(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

func recoveryRequestForPlan(plan ActivationPlan, menu string) RecoveryProofRequest {
	var ethernet []string
	for _, module := range plan.WiredDrivers {
		ethernet = append(ethernet, module)
	}
	slices.Sort(ethernet)
	ethernet = slices.Compact(ethernet)
	return RecoveryProofRequest{TargetABI: nobleABI, RecoveryRelease: plan.RecoveryRelease, MenuID: menu, RequiredEthernetModules: ethernet, RequiredNVIDIAModules: slices.Clone(plan.NVIDIALoadedModules)}
}

func inspectWirelessStage(ctx context.Context, root *os.Root, req WirelessOperationRequest) (wirelessStageAdmission, error) {
	a := wirelessStageAdmission{root: root, result: WirelessOperationResult{Operation: "stage", Profile: req.Profile, State: "prerequisites"}}
	r := &a.result
	wirelessResultCheck(r, "root", os.Geteuid() == 0, "Use the normal elevated host command for privileged publication")
	plan, err := PreviewActivation(ctx, req.Profile, req.Bundle, req.RecoveryKernel)
	if err != nil {
		return a, err
	}
	r.Plan = &plan
	wirelessResultCheck(r, "reviewed_abi_device", plan.PreparationState == "eligible", plan.PreparationIssue)
	wirelessResultCheck(r, "reviewed_receipt", validSHA(req.ReceiptSHA256) && req.ReceiptSHA256 == plan.BundleReceiptSHA256, "Explicit independently reviewed preparation receipt must match the exact generation")
	protected := len(plan.Preserved) > 0
	for _, file := range append(slices.Clone(plan.Originals), plan.Preserved...) {
		if file.Issue != "" || !validSHA(file.SHA256) {
			protected = false
		}
	}
	wirelessResultCheck(r, "protected_boot_providers", protected, "Current/fallback boot, Ethernet, NVIDIA, headers and indexes must be readable and unchanged")
	if req.SignedBundle != "" {
		a.signed, err = VerifyPublishedSignedBundle(ctx, req.Profile, req.Bundle, req.SignedBundle)
		wirelessResultCheck(r, "signed_bundle", err == nil, errorDetail(err))
		if err == nil {
			r.SignedBundle = &a.signed
			wirelessResultCheck(r, "signed_receipt_binding", a.signed.BundleReceiptSHA256 == plan.BundleReceiptSHA256 && a.signed.SignaturesVerified, "Published signatures must authenticate the reviewed unsigned generation")
			a.cert, err = readPublicCertificate(a.signed.PublicCertificate)
			if err != nil {
				return a, err
			}
			trust, err := EvaluateKernelTrust(ctx, a.signed.PublicCertificate)
			if err != nil {
				return a, err
			}
			r.Trust = &trust
			wirelessResultCheck(r, "kernel_trust", trust.State == "verified" && trust.CertificateSHA256 == a.signed.CertificateSHA256, trust.Issue)
		}
	} else {
		wirelessResultCheck(r, "signed_bundle", false, "Publish or sign the exact reviewed generation and select its immutable --signed-bundle digest")
	}
	wirelessResultCheck(r, "secure_boot", plan.SecureBoot.Enabled && plan.SecureBoot.Error == "", "The reviewed live route preserves enabled Secure Boot and verifies running-kernel trust")
	policyReq := FirmwarePolicyRequest{Root: root, KernelRelease: nobleABI, Modules: slices.Clone(preparationProfiles[req.Profile].modules)}
	for _, image := range durableFirmwarePins(req.Profile) {
		policyReq.Firmware = append(policyReq.Firmware, FirmwarePolicyTarget{Name: strings.TrimPrefix(image.Path, "lib/firmware/"), SHA256: image.SHA256})
	}
	policy, policyErr := InspectFirmwarePolicy(ctx, policyReq)
	r.Firmware = &policy
	wirelessResultCheck(r, "firmware_precedence_initramfs", policyErr == nil && policy.StageAllowed, strings.Join(append(policy.Reasons, errorDetail(policyErr)), "; "))
	a.images, err = observeFirmwarePreflight(ctx, req.Profile)
	if err != nil {
		return a, err
	}
	firmwareVerified := len(a.images) > 0
	for _, image := range a.images {
		if !image.CandidateVerified {
			firmwareVerified = false
		}
	}
	wirelessResultCheck(r, "firmware_candidates", firmwareVerified, "Every required image must match its exact governed/local qualification pin")
	recoveryReq := recoveryRequestForPlan(plan, req.RecoveryEntry)
	recovery, recoveryErr := VerifyRecoveryProof(ctx, recoveryReq)
	r.Recovery = &recovery
	wirelessResultCheck(r, "known_good_stock_recovery", recoveryErr == nil, errorDetail(recoveryErr))
	boot, err := root.ReadFile("proc/sys/kernel/random/boot_id")
	if err != nil {
		return a, err
	}
	routes, err := readSmallRegular(root.FS(), "proc/net/route", 64<<10)
	if err != nil {
		return a, err
	}
	a.context = wirelessJournalContext{OperatorUID: wirelessActorUID(), SignedBundle: req.SignedBundle, ReviewedReceiptSHA256: req.ReceiptSHA256, Recovery: recoveryReq, Firmware: policy, BootID: strings.TrimSpace(string(boot)), EthernetRoutes: ethernetRouteLines(routes, plan.WiredInterfaces)}
	machine, e := readSmallRegular(root.FS(), "etc/machine-id", 4096)
	if e != nil {
		return a, e
	}
	a.context.MachineID = strings.TrimSpace(string(machine))
	prepared, e := Prepare(ctx, req.Profile, false)
	if e != nil {
		return a, e
	}
	ids := prepared.PCIDevices
	if req.Profile == NobleBluetoothProfile {
		ids = prepared.USBDevices
	}
	wirelessResultCheck(r, "exact_device", len(ids) == 1, "Exactly one scoped radio device is required")
	if len(ids) == 1 {
		a.context.Device = ids[0]
	}
	a.context.StageModules, e = moduleSnapshots(root, req.Profile)
	if e != nil {
		return a, e
	}
	r.Ready = wirelessChecksReady(r)
	if r.Ready {
		r.State = "ready"
	}
	return a, nil
}

func runNativeWirelessOperation(ctx context.Context, operation string, req WirelessOperationRequest, apply bool) (WirelessOperationResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	r := WirelessOperationResult{Operation: operation, Profile: req.Profile, State: "prerequisites"}
	if apply && os.Geteuid() != 0 {
		return r, fmt.Errorf("%w: use the normal sudo host wireless command", hostreqkit.ErrElevationRequired)
	}
	root, err := os.OpenRoot("/")
	if err != nil {
		return r, err
	}
	defer root.Close()
	if apply {
		release, err := lockWirelessPackageState(root)
		if err != nil {
			return r, err
		}
		defer release()
	}
	if operation == "record-recovery" {
		return runWirelessRecoveryRecording(ctx, req, apply)
	}
	if operation != "stage" {
		return runWirelessJournalOperation(ctx, root, operation, req, apply)
	}
	a, err := inspectWirelessStage(ctx, root, req)
	if err != nil {
		return a.result, err
	}
	if !apply {
		return a.result, nil
	}
	if !a.result.Ready {
		return a.result, errors.New("wireless publication prerequisites are not satisfied")
	}
	return applyWirelessStage(ctx, &a, req)
}

func acquireWirelessNamespace(ctx context.Context, root *os.Root) (*wirelessRootAdmission, error) {
	paths, err := admitWirelessRootPaths(root, uint32(os.Geteuid()))
	if err != nil {
		return nil, err
	}
	f, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var st unix.Stat_t
	if err = unix.Fstat(int(f.Fd()), &st); err != nil {
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	return &wirelessRootAdmission{device: uint64(st.Dev), inode: st.Ino, uid: uint32(os.Geteuid()), issued: time.Now(), paths: paths}, nil
}

func reserveWirelessStage(ctx context.Context) (func() error, error) {
	bytes, err := copiedTreeBudget(ctx, "/lib/modules/"+nobleABI)
	if err != nil {
		return nil, err
	}
	broker, err := capacity.NewBroker(ctx, "")
	if err != nil {
		return nil, err
	}
	lease, err := broker.ReserveDisk(ctx, "host-wireless:live-transaction", "/", 3*bytes+buildAllowance, 0)
	if err != nil {
		broker.Close()
		return nil, err
	}
	return func() error {
		defer broker.Close()
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return lease.Release(cleanup)
	}, nil
}

func applyWirelessStage(ctx context.Context, a *wirelessStageAdmission, req WirelessOperationRequest) (_ WirelessOperationResult, resultErr error) {
	r := a.result
	release, err := reserveWirelessStage(ctx)
	if err != nil {
		return r, err
	}
	defer func() { resultErr = errors.Join(resultErr, release()) }()
	capability, err := acquireWirelessNamespace(ctx, a.root)
	if err != nil {
		return r, err
	}
	owner, err := openAdmittedDurableWireless(ctx, a.root, capability)
	if err != nil {
		return r, err
	}
	defer owner.close()
	if owner.intent.ID != "" && owner.intent.Plan.Profile == req.Profile && owner.intent.Plan.Bundle == req.Bundle {
		if owner.intent.Context == nil || owner.intent.Context.SignedBundle != req.SignedBundle || owner.intent.Context.ReviewedReceiptSHA256 != req.ReceiptSHA256 {
			return r, errors.New("existing transaction has different reviewed signing identity")
		}
		if err = owner.apply(ctx); err != nil {
			return r, err
		}
		r.Transaction, r.State = owner.intent.ID, owner.phase
		return r, nil
	}
	// Native index proof is constructed under the same global mutation lock.
	if err = qualifyWirelessScratchParent(a.root); err != nil {
		return r, err
	}
	indexed, err := newIndexedActivationAt(ctx, a.root.FS(), *r.Plan, nativePrivateIndexes, "/var/tmp")
	if err != nil {
		return r, err
	}
	defer indexed.close()
	signedRoot, err := os.OpenRoot(a.signed.ModulesDirectory)
	if err != nil {
		return r, err
	}
	defer signedRoot.Close()
	if _, err = indexed.stage(ctx, bundleFiles{signedRoot}, a.cert); err != nil {
		return r, err
	}
	owner.context = &a.context
	if err = owner.begin(ctx, indexed, a.cert, a.images...); err != nil {
		return r, err
	}
	r.Transaction = owner.intent.ID
	if err = owner.apply(ctx); err != nil {
		r.State = "recovery_required"
		return r, err
	}
	r.State = owner.phase
	return r, nil
}

func runWirelessRecoveryRecording(ctx context.Context, req WirelessOperationRequest, apply bool) (WirelessOperationResult, error) {
	r := WirelessOperationResult{Operation: "record-recovery", Profile: req.Profile, State: "prerequisites"}
	inv, err := hostinventory.NewIntegrityCollector(hostinventory.IntegrityCollectorOptions{}).Collect(ctx)
	if err != nil {
		return r, err
	}
	plan := ActivationPlan{Profile: req.Profile, KernelRelease: nobleABI, RecoveryRelease: req.RecoveryKernel, WiredDrivers: map[string]string{}}
	for _, module := range inv.Kernel.LoadedModules {
		if module == "nvidia" || strings.HasPrefix(module, "nvidia_") {
			plan.NVIDIALoadedModules = append(plan.NVIDIALoadedModules, module)
		}
	}
	data, err := rootTextRoute()
	if err != nil {
		return r, err
	}
	for _, iface := range defaultIPv4Interfaces(data) {
		link, e := os.Readlink("/sys/class/net/" + iface + "/device/driver/module")
		if e == nil {
			plan.WiredDrivers[iface] = path.Base(link)
		}
	}
	recoveryReq := recoveryRequestForPlan(plan, req.RecoveryEntry)
	if !apply {
		r.Checks = []WirelessCheck{{Name: "independent_stock_boot", Satisfied: inv.Kernel.Release == req.RecoveryKernel, Detail: "Record at the active physical VT on this exact stock fallback; no boot change is performed"}, {Name: "root", Satisfied: os.Geteuid() == 0}}
		r.Ready = wirelessChecksReady(&r)
		return r, nil
	}
	proof, err := RecordRecoveryProof(ctx, recoveryReq)
	if err != nil {
		return r, err
	}
	r.Recovery = &proof
	r.State = "recorded"
	r.Ready = true
	return r, nil
}

func rootTextRoute() ([]byte, error) {
	return readSmallRegular(os.DirFS("/"), "proc/net/route", 64<<10)
}
func ethernetRouteLines(data []byte, interfaces []string) []string {
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 7 && slices.Contains(interfaces, fields[0]) {
			lines = append(lines, strings.Join(fields, " "))
		}
	}
	slices.Sort(lines)
	return lines
}

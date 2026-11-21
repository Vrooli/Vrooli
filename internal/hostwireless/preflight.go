package hostwireless

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	mt7927source "github.com/vrooli/vrooli/internal/tools/mt7927-source"
)

// ActivationPreflight binds read-only observations to one reviewed generation.
// It is a snapshot for review, never an authorization or reusable live permit.
type ActivationPreflight struct {
	Plan         ActivationPlan      `json:"plan"`
	SigningTrust *SigningTrust       `json:"signing_trust,omitempty"`
	SignedCopy   *SignedCopyEvidence `json:"signed_copy,omitempty"`
	Firmware     []FirmwarePreflight `json:"firmware"`
	Steps        []ActivationStep    `json:"steps"`
}

type FirmwarePreflight struct {
	Candidate         FileIdentity `json:"candidate"`
	Live              FileIdentity `json:"live"`
	ExpectedSHA256    string       `json:"expected_sha256"`
	CandidateVerified bool         `json:"candidate_verified"`
	LiveMatches       bool         `json:"live_matches"`
}

// ActivationStep states the next evidence and its real owner. No boolean
// supplied by a caller can substitute for a boot, trust or device observation.
type ActivationStep struct {
	Name             string `json:"name"`
	State            string `json:"state"`
	Owner            string `json:"owner"`
	RequiredEvidence string `json:"required_evidence"`
}

// PreflightActivation joins existing preparation, signature and host observers.
// An optional certificate enables read-only trust inspection; signedDir also
// authenticates signed copies. It never opens a signing key or changes host files.
func PreflightActivation(ctx context.Context, profile, generation, recovery, signedDir, certPath string) (ActivationPreflight, error) {
	var r ActivationPreflight
	if generation == "" || (signedDir != "" && certPath == "") {
		return r, errors.New("activation preflight requires --bundle; --signed-dir also requires --cert")
	}
	ctx, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()
	var err error
	r.Plan, err = PreviewActivation(ctx, profile, generation, recovery)
	if err != nil {
		return r, err
	}
	if r.Plan.PreparationState != "eligible" {
		return r, nil
	}
	if certPath != "" {
		trust, err := InspectSigningTrust(ctx, certPath)
		if err != nil {
			return r, err
		}
		r.SigningTrust = &trust
	}
	if signedDir != "" {
		signed, err := VerifySignedCopy(ctx, profile, generation, signedDir, certPath)
		if err != nil {
			return r, err
		}
		if signed.BundleReceiptSHA256 != r.Plan.BundleReceiptSHA256 || signed.CertificateSHA256 != r.SigningTrust.CertificateSHA256 {
			return r, errors.New("preflight bundle or public certificate changed during observation")
		}
		r.SignedCopy = &signed
	}
	r.Firmware, err = observeFirmwarePreflight(ctx, profile)
	if err != nil {
		return r, err
	}
	r.Steps = activationSteps(r)
	return r, ctx.Err()
}

func observeFirmwarePreflight(ctx context.Context, profile string) ([]FirmwarePreflight, error) {
	name := mt7927source.Name
	if profile == NobleBluetoothProfile {
		name = mt7927source.ASUSBluetoothName
	}
	source, _ := mt7927source.Lookup(name)
	stage, err := source.StagedPath()
	if err != nil {
		return nil, err
	}
	type image struct{ name, sha, candidate string }
	images := []image{
		{"WIFI_MT6639_PATCH_MCU_2_1_hdr.bin", "7a5aacf68ed8526c1d94ba962f84046496b3909562decf8bb690cfbd8de5b65e", stage},
		{"WIFI_RAM_CODE_MT6639_2_1.bin", "e841f5ed4fe547ff2f15275a2f330bb19a253db92e3e59cbdfd8581d72b2582d", stage},
	}
	if profile == NobleBluetoothProfile {
		images = []image{{bluetoothFirmwareName, bluetoothFirmwareSHA, filepath.Join(filepath.Dir(filepath.Dir(stage)), "host-wireless-firmware", "asus-mt6639", source.Version())}}
	}
	var result []FirmwarePreflight
	for _, image := range images {
		candidate := observeFile(ctx, filepath.Join(image.candidate, image.name))
		live := observeFile(ctx, filepath.Join("/lib/firmware/mediatek/mt7927", image.name))
		result = append(result, FirmwarePreflight{Candidate: candidate, Live: live, ExpectedSHA256: image.sha,
			CandidateVerified: candidate.Issue == "" && candidate.SHA256 == image.sha,
			LiveMatches:       live.Issue == "" && live.SHA256 == image.sha})
	}
	return result, nil
}

func activationSteps(r ActivationPreflight) []ActivationStep {
	firmwareEvidence := "Use only exact firmware candidate hashes; preserve absent/present original and lookup precedence including compressed images, override paths and initramfs."
	activationEvidence := "Coordinate radio-specific load/rebind only; never unload serving GPU/Ethernet or automatically reboot."
	if r.Plan.Profile == NobleBluetoothProfile {
		firmwareEvidence += " ASUS local Linux-use terms remain uncertain; redistribution unqualified."
		activationEvidence += " Inventory all btusb consumers and qualify exact USB/HCI transfer and reset behavior."
	} else {
		activationEvidence += " Wi-Fi PCIe loss containment remains a physical proof."
	}
	sign := "signed_copy_required"
	if r.SignedCopy != nil && r.SignedCopy.SignaturesVerified {
		sign = "authenticated_copies_observed"
	}
	protected := "unmeasured"
	if r.Plan.RecoveryRelease != "" {
		protected = "identities_observed_console_proof_required"
		for _, file := range append(append([]FileIdentity{}, r.Plan.Originals...), r.Plan.Preserved...) {
			if file.Issue != "" || !validSHA(file.SHA256) {
				protected = "protected_identity_unavailable"
			}
		}
	}
	firmware := "candidate_required"
	if len(r.Firmware) != 0 {
		firmware = "verified_candidates_require_transaction"
		for _, image := range r.Firmware {
			if !image.CandidateVerified {
				firmware = "candidate_required"
			}
		}
	}
	return []ActivationStep{
		{"stock_recovery", protected, "operator-console + hostwireless", "Hash readable current and distinct stock kernel/initramfs and GRUB configuration; prove an exact selectable boot entry, its Ethernet/NVIDIA providers, working console and successful stock boot. File existence is insufficient."},
		{"sign", sign, "Ubuntu sign-file + hostwireless signature verifier", "A supervisor-coordinated invocation signs copies of only this bundle with the existing module-only MOK; authenticate exact stripped payload and public DER certificate. Never generate or enroll trust implicitly."},
		{"kernel_trust", "independent_native_proof_required", "Ubuntu shim-signed + running kernel", "Record exact public certificate, Secure Boot/lockdown, pending/blocklist state and usable running kernel trust evidence. Enrollment or SKID lookup alone does not prove module acceptance."},
		{"firmware", firmware, "hostwireless + governed SDA source", firmwareEvidence},
		{"stage_and_rollback", "privileged_adapter_and_live_qualification_required", "hostwireless + hostreqkit privilege boundary", "Qualify the privilege adapter and live recovery around the implemented durable overlay/index/firmware journal, single writer, fresh drift checks and retained originals. Copied-root proof does not publish live module indexes atomically."},
		{"activate", "coordinated_device_effect_required", "operator + hostwireless", fmt.Sprintf("Revalidate ABI %s, all stock reverse dependencies, exact aliases, Ethernet/NVIDIA providers and recovery. %s", r.Plan.KernelRelease, activationEvidence)},
		{"verify_function", "physical_proof_required", "hostinventory + operator peer", "Record selected signed module and firmware initialization, usable interface/controller, Wi-Fi association and traffic or Bluetooth discovery/pairing, then suspend/resume and exact rollback. An observed activation is not proof of boot safety."},
	}
}

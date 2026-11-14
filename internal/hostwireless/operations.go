package hostwireless

import (
	"context"
	"errors"
	"slices"
)

// WirelessOperationRequest names exact reviewed artifacts and existing root
// journal lineage. No caller-provided boolean is a trust or recovery proof.
type WirelessOperationRequest struct {
	Profile           string `json:"profile"`
	Bundle            string `json:"bundle,omitempty"`
	SignedBundle      string `json:"signed_bundle,omitempty"`
	ReceiptSHA256     string `json:"reviewed_receipt_sha256,omitempty"`
	RecoveryKernel    string `json:"recovery_kernel,omitempty"`
	RecoveryEntry     string `json:"recovery_entry,omitempty"`
	Transaction       string `json:"transaction,omitempty"`
	MaintenanceWindow bool   `json:"maintenance_window"`
}

type WirelessCheck struct {
	Name      string `json:"name"`
	Satisfied bool   `json:"satisfied"`
	Detail    string `json:"detail,omitempty"`
}

type WirelessOperationResult struct {
	Operation    string                    `json:"operation"`
	Profile      string                    `json:"profile"`
	State        string                    `json:"state"`
	Ready        bool                      `json:"ready"`
	Transaction  string                    `json:"transaction,omitempty"`
	Checks       []WirelessCheck           `json:"checks"`
	Plan         *ActivationPlan           `json:"plan,omitempty"`
	SignedBundle *SignedBundleReceipt      `json:"signed_bundle,omitempty"`
	Firmware     *FirmwarePolicyEvidence   `json:"firmware,omitempty"`
	Recovery     *RecoveryProof            `json:"recovery,omitempty"`
	Trust        *KernelTrustEvidence      `json:"kernel_trust,omitempty"`
	Radio        *WirelessRadioObservation `json:"radio,omitempty"`
}

type WirelessRadioObservation struct {
	Device                   string   `json:"device,omitempty"`
	Driver                   string   `json:"driver,omitempty"`
	Interfaces               []string `json:"interfaces,omitempty"`
	Controllers              []string `json:"controllers,omitempty"`
	CandidateModulesVerified bool     `json:"candidate_modules_verified"`
	CandidateModulesLoaded   []string `json:"candidate_modules_loaded,omitempty"`
	LoadedModules            []string `json:"loaded_modules,omitempty"`
	BindingObserved          bool     `json:"binding_observed"`
	FunctionVerified         bool     `json:"function_verified"`
	Details                  []string `json:"details,omitempty"`
}

// RunWirelessOperation is the canonical typed transaction surface. Preview is
// read-only; effects require explicit apply and the native owner remeasures all
// operation-specific prerequisites. Other platforms return precise unsupported.
func RunWirelessOperation(ctx context.Context, operation string, req WirelessOperationRequest, apply bool) (WirelessOperationResult, error) {
	r := WirelessOperationResult{Operation: operation, Profile: req.Profile, State: "prerequisites"}
	if _, ok := preparationProfiles[req.Profile]; !ok {
		return r, errors.New("unknown wireless profile")
	}
	if !slices.Contains([]string{"stage", "activate", "recover", "rollback", "status", "record-recovery"}, operation) {
		return r, errors.New("unknown wireless transaction operation")
	}
	if req.Bundle != "" && !bundleID.MatchString(req.Bundle) {
		return r, errors.New("invalid preparation generation")
	}
	if req.SignedBundle != "" && !validSHA(req.SignedBundle) {
		return r, errors.New("invalid signed bundle digest")
	}
	if req.ReceiptSHA256 != "" && !validSHA(req.ReceiptSHA256) {
		return r, errors.New("invalid reviewed receipt SHA-256")
	}
	if req.Transaction != "" && !validOperationTransaction(req.Transaction) {
		return r, errors.New("invalid wireless transaction ID")
	}
	if operation == "stage" && (req.Bundle == "" || req.RecoveryKernel == "" || req.RecoveryEntry == "") {
		return r, errors.New("stage requires exact unsigned bundle, distinct recovery kernel and recovery entry")
	}
	if slices.Contains([]string{"activate", "recover", "rollback", "status"}, operation) && req.Transaction == "" {
		return r, errors.New("operation requires exact transaction ID")
	}
	if operation == "record-recovery" && (req.RecoveryKernel == "" || req.RecoveryEntry == "") {
		return r, errors.New("record-recovery requires exact fallback kernel and menu entry")
	}
	return runNativeWirelessOperation(ctx, operation, req, apply)
}

func validOperationTransaction(id string) bool {
	if len(id) != 32 {
		return false
	}
	for _, ch := range id {
		if !(ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f') {
			return false
		}
	}
	return true
}

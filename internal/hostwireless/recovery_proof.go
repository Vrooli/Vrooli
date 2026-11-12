package hostwireless

import "context"

// RecoveryProofRequest names the exact target and independently booted fallback.
// Provider sets come from target admission; callers cannot replace observations.
type RecoveryProofRequest struct {
	TargetABI               string   `json:"target_abi"`
	RecoveryRelease         string   `json:"recovery_release"`
	MenuID                  string   `json:"menu_id"`
	RequiredEthernetModules []string `json:"required_ethernet_modules"`
	RequiredNVIDIAModules   []string `json:"required_nvidia_modules"`
}
type RecoveryProof struct {
	SchemaVersion      int                  `json:"schema_version"`
	Request            RecoveryProofRequest `json:"request"`
	MachineID          string               `json:"machine_id"`
	BootID             string               `json:"observed_stock_boot_id"`
	Console            string               `json:"physical_console"`
	ObservedAt         string               `json:"observed_at"`
	Files              []FileIdentity       `json:"files"`
	EthernetInterfaces []string             `json:"ethernet_interfaces"`
}

// RecordRecoveryProof must run at a real active Linux VT while booted into the
// requested fallback. It writes a root-owned durable record only after measuring
// stock providers, a live Ethernet default route, boot artifacts and menu entry.
func RecordRecoveryProof(ctx context.Context, req RecoveryProofRequest) (RecoveryProof, error) {
	return recordNativeRecoveryProof(ctx, req)
}

// VerifyRecoveryProof requires the target ABI and a different boot ID, and reads
// the protected record again; a user boolean or supplied JSON is not evidence.
func VerifyRecoveryProof(ctx context.Context, req RecoveryProofRequest) (RecoveryProof, error) {
	return verifyNativeRecoveryProof(ctx, req)
}

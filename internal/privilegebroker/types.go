// Package privilegebroker implements the setup-managed, local-only host action
// boundary. It intentionally has no generic command execution API.
package privilegebroker

import "strings"

const (
	ProtocolVersion = "v1"
	BridgePort      = 18767
	BridgeScenario  = "vrooli-bridge"
	RuleComment     = "vrooli-bridge-admission-v1"

	ActionBridgeUFWInspect = "bridge.ufw.inspect"
	ActionBridgeUFWAllow   = "bridge.ufw.allow"
	ActionBridgeUFWVerify  = "bridge.ufw.verify"
	ActionBridgeUFWRevoke  = "bridge.ufw.revoke"

	// Volume actions repair a storage filesystem that the kernel refuses to
	// mount read/write. They exist only for hosts without udisks2, which
	// authorises the same operations for an active session with no elevation
	// at all; where udisks2 is present the caller must prefer it.
	//
	// The family is deliberately limited to block-device operations. Mount and
	// unmount are absent because this service runs with mount-namespace
	// isolation, so a mount performed here would not propagate to the host —
	// an action that silently does nothing is worse than no action.
	ActionVolumeFilesystemCheck      = "volume.filesystem.check"
	ActionVolumeFilesystemRepair     = "volume.filesystem.repair"
	ActionRuntimeHomeOwnershipRepair = "runtime-home.ownership.repair"
)

// Request is the complete v1 wire input. Deliberately, it has no command,
// argv, environment, working-directory, or path field.
type Request struct {
	Version   string  `json:"version"`
	RequestID string  `json:"request_id"`
	Action    string  `json:"action"`
	Subject   Subject `json:"subject"`
	// Volume carries the target of a volume.* action. It is a separate subject
	// rather than extra fields on Subject so each action family validates
	// against exactly the shape it needs and nothing else.
	Volume      *VolumeSubject      `json:"volume,omitempty"`
	RuntimeHome *RuntimeHomeSubject `json:"runtime_home,omitempty"`
}

// VolumeSubject identifies a storage volume for a volume.* action. There is no
// command, argument, option, or mountpoint field: the policy builds a fixed
// argv from the filesystem type alone.
type VolumeSubject struct {
	Device     string `json:"device"`
	Filesystem string `json:"filesystem"`
	// UUID and Serial bind the request to a specific disk. At least one is
	// required, so a renumbered or replugged device cannot inherit approval.
	UUID   string `json:"uuid,omitempty"`
	Serial string `json:"serial,omitempty"`
}

type RuntimeHomeSubject struct {
	Class       string `json:"class"`
	ExpectedUID uint32 `json:"expected_uid"`
	ExpectedGID uint32 `json:"expected_gid"`
}

type Subject struct {
	Scenario    string `json:"scenario"`
	CandidateIP string `json:"candidate_ip"`
	Port        int    `json:"port"`
}

type Result struct {
	Version   string   `json:"version"`
	RequestID string   `json:"request_id,omitempty"`
	Action    string   `json:"action,omitempty"`
	Status    string   `json:"status"`
	Code      string   `json:"code,omitempty"`
	Changed   bool     `json:"changed,omitempty"`
	Evidence  Evidence `json:"evidence,omitempty"`
}

type Evidence struct {
	Available bool   `json:"available"`
	Active    bool   `json:"active"`
	RuleFound bool   `json:"rule_found"`
	Managed   bool   `json:"managed"`
	Detail    string `json:"detail,omitempty"`
	// Volume-action evidence. Mounted and IdentityVerified record the broker's
	// own checks rather than the caller's claims; ExitCode preserves the tool's
	// status so a corrected-errors result is not mistaken for a failure.
	Mounted          bool   `json:"mounted,omitempty"`
	IdentityVerified bool   `json:"identity_verified,omitempty"`
	ExitCode         int    `json:"exit_code,omitempty"`
	Scanned          uint64 `json:"scanned,omitempty"`
	Repaired         uint64 `json:"repaired,omitempty"`
	Failed           uint64 `json:"failed,omitempty"`
}

func NewFailure(requestID, action, code string) Result {
	return Result{Version: ProtocolVersion, RequestID: strings.TrimSpace(requestID), Action: strings.TrimSpace(action), Status: "failed", Code: code}
}

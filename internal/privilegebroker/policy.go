package privilegebroker

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

const policyFilesystemFixer = "ntfsfix"

// Validate rejects every request shape outside the immutable v1 registry.
func Validate(req Request) error {
	if req.Version != ProtocolVersion {
		return fmt.Errorf("unsupported_version")
	}
	if strings.TrimSpace(req.RequestID) == "" || len(req.RequestID) > 128 {
		return fmt.Errorf("invalid_request_id")
	}
	switch req.Action {
	case ActionBridgeUFWInspect, ActionBridgeUFWAllow, ActionBridgeUFWVerify, ActionBridgeUFWRevoke:
		return validateBridge(req)
	case ActionVolumeFilesystemCheck, ActionVolumeFilesystemRepair:
		return validateVolume(req)
	case ActionRuntimeHomeOwnershipRepair:
		return validateRuntimeHome(req)
	case ActionLogRotateForce, ActionJournaldVacuum, ActionDockerPruneUnusedImages, ActionDockerPruneUnusedVolumes:
		return validateStorageAction(req)
	default:
		return fmt.Errorf("action_not_allowed")
	}
}

const managedLogStanza = "vrooli-log-volume-bounds"

func validateStorageAction(req Request) error {
	if req.Subject != (Subject{}) || req.Volume != nil || req.RuntimeHome != nil {
		return fmt.Errorf("subject_not_allowed")
	}
	switch req.Action {
	case ActionLogRotateForce:
		if req.Log == nil || req.Journal != nil || req.Docker != nil || strings.TrimSpace(req.Log.Stanza) != managedLogStanza {
			return fmt.Errorf("managed_log_stanza_required")
		}
	case ActionJournaldVacuum:
		if req.Journal == nil || req.Log != nil || req.Docker != nil || req.Journal.MaxUseBytes <= 0 {
			return fmt.Errorf("positive_journal_size_required")
		}
	case ActionDockerPruneUnusedImages:
		if req.Docker == nil || req.Log != nil || req.Journal != nil || len(req.Docker.VolumeNames) != 0 {
			return fmt.Errorf("image_prune_subject_required")
		}
	case ActionDockerPruneUnusedVolumes:
		if req.Docker == nil || req.Log != nil || req.Journal != nil || len(req.Docker.VolumeNames) == 0 {
			return fmt.Errorf("named_volume_list_required")
		}
		for _, name := range req.Docker.VolumeNames {
			name = strings.TrimSpace(name)
			if name == "" || name == "*" || strings.ContainsAny(name, "/\\ \t\r\n") || len(name) > 128 {
				return fmt.Errorf("invalid_volume_name")
			}
		}
	}
	return nil
}

func validateRuntimeHome(req Request) error {
	if req.Subject != (Subject{}) || req.Volume != nil || req.RuntimeHome == nil {
		return fmt.Errorf("runtime_home_subject_required")
	}
	switch strings.TrimSpace(req.RuntimeHome.Class) {
	case "bin", "cache", "logs", "metrics", "processes", "build", "test_runs", "backups", "artifacts", "secrets_enc":
	default:
		return fmt.Errorf("runtime_home_class_not_allowed")
	}
	if req.RuntimeHome.ExpectedUID == 0 || req.RuntimeHome.ExpectedGID == 0 {
		return fmt.Errorf("runtime_home_identity_required")
	}
	return nil
}

// validateBridge enforces the bridge admission shape. A bridge request that
// also carries a volume subject is rejected: each action family must arrive in
// exactly its own shape, never a union of both.
func validateBridge(req Request) error {
	if req.Volume != nil {
		return fmt.Errorf("subject_not_allowed")
	}
	if req.Subject.Scenario != BridgeScenario {
		return fmt.Errorf("scenario_not_allowed")
	}
	if req.Subject.Port != BridgePort {
		return fmt.Errorf("port_not_allowed")
	}
	ip := net.ParseIP(strings.TrimSpace(req.Subject.CandidateIP))
	if ip == nil || ip.IsUnspecified() || ip.IsLoopback() || ip.IsMulticast() {
		return fmt.Errorf("invalid_candidate_ip")
	}
	return nil
}

// volumeDevicePattern accepts a plain Linux block device path and nothing else,
// so a device string can never smuggle an option or a shell fragment into the
// argv the policy builds.
var volumeDevicePattern = regexp.MustCompile(`^/dev/[A-Za-z0-9][A-Za-z0-9._/-]{0,62}$`)

// volumeFilesystems maps an accepted filesystem type onto its repair family.
// Membership means a fixed argv exists for that family.
var volumeFilesystems = map[string]string{
	"ntfs": "ntfs", "ntfs3": "ntfs",
	"ext2": "ext", "ext3": "ext", "ext4": "ext",
	"exfat": "exfat",
	"vfat":  "vfat", "fat32": "vfat", "msdos": "vfat",
}

// validateVolume enforces the volume shape: a plain device path, a filesystem
// with a known adapter, and at least one identifier that survives a replug.
func validateVolume(req Request) error {
	if req.Volume == nil {
		return fmt.Errorf("volume_subject_required")
	}
	if req.Subject != (Subject{}) {
		return fmt.Errorf("subject_not_allowed")
	}
	device := strings.TrimSpace(req.Volume.Device)
	if device == "" || strings.Contains(device, "..") || !volumeDevicePattern.MatchString(device) {
		return fmt.Errorf("invalid_device")
	}
	if _, ok := volumeFilesystems[strings.ToLower(strings.TrimSpace(req.Volume.Filesystem))]; !ok {
		return fmt.Errorf("filesystem_not_allowed")
	}
	if strings.TrimSpace(req.Volume.UUID) == "" && strings.TrimSpace(req.Volume.Serial) == "" {
		// A device path alone cannot prove which disk is present. Refusing is
		// the only safe answer for an action that writes filesystem metadata.
		return fmt.Errorf("device_identity_required")
	}
	if len(req.Volume.UUID) > 128 || len(req.Volume.Serial) > 128 {
		return fmt.Errorf("invalid_device_identity")
	}
	return nil
}

// VolumeArgs returns the fixed tool and argv for a volume action, only after
// Validate has accepted the request. The check variants are the tools' explicit
// no-action modes and never write.
func VolumeArgs(req Request) (string, []string, error) {
	if err := Validate(req); err != nil {
		return "", nil, fmt.Errorf("validate privileged volume request: %w", err)
	}
	device := strings.TrimSpace(req.Volume.Device)
	family := volumeFilesystems[strings.ToLower(strings.TrimSpace(req.Volume.Filesystem))]
	repair := req.Action == ActionVolumeFilesystemRepair
	switch family {
	case "ntfs":
		if repair {
			return policyFilesystemFixer, []string{"-d", device}, nil
		}
		return policyFilesystemFixer, []string{"-n", device}, nil
	case "ext":
		if repair {
			return "e2fsck", []string{"-f", "-y", device}, nil
		}
		return "e2fsck", []string{"-f", "-n", device}, nil
	case "exfat":
		if repair {
			return "fsck.exfat", []string{"-y", device}, nil
		}
		return "fsck.exfat", []string{"-n", device}, nil
	case "vfat":
		if repair {
			return "fsck.fat", []string{"-a", device}, nil
		}
		return "fsck.fat", []string{"-n", device}, nil
	default:
		return "", nil, fmt.Errorf("filesystem_not_allowed")
	}
}

// UFWArgs returns a fixed argv only after Validate has accepted the request.
// The executor always invokes ufw directly; callers cannot influence a shell.
func UFWArgs(req Request) ([]string, error) {
	if err := Validate(req); err != nil {
		return nil, err
	}
	ip := net.ParseIP(strings.TrimSpace(req.Subject.CandidateIP)).String()
	base := []string{"from", ip, "to", "any", "port", "18767", "proto", "tcp", "comment", RuleComment}
	switch req.Action {
	case ActionBridgeUFWInspect, ActionBridgeUFWVerify:
		return []string{"status", "numbered"}, nil
	case ActionBridgeUFWAllow:
		return append([]string{"allow"}, base...), nil
	case ActionBridgeUFWRevoke:
		return append([]string{"delete", "allow"}, base...), nil
	default:
		return nil, fmt.Errorf("action_not_allowed")
	}
}

package driver

import "workspace-sandbox/internal/types"

// NamespaceWorkspacePath is the agent-visible mount point onto which a
// containment backend that provides a path illusion (bwrap on Linux)
// bind-mounts the sandbox merged dir. It is the single source of truth for
// the reported workspacePath: the exec argv binds the merged dir here (see
// exec.BuildBwrapArgs) and DeriveWorkspaceLayout reports it. Do not
// hard-code "/workspace" elsewhere.
const NamespaceWorkspacePath = "/workspace"

// backendNone is the backend id used when execution falls through to the
// direct (uncontained) path.
const backendNone = "none"

// SeatbeltBackendID is the containment backend id for the macOS Seatbelt
// (sandbox-exec) backend. Kept next to the neutral derivation so the darwin
// probe and the exec backend name it consistently.
const SeatbeltBackendID = "seatbelt"

// seatbeltContainmentInfo builds the containment report for the macOS
// Seatbelt backend, given whether sandbox-exec was found on the host and its
// resolved path. Pure so the darwin probe (containment_darwin.go) stays a
// thin LookPath call and the report is unit-tested on the Linux dev host.
//
// Seatbelt is honestly partial: it enforces filesystem write-containment and
// network denial, but provides NO path illusion and NO pid namespace, so
// those enforcements are deliberately absent from the list. When sandbox-exec
// is missing the report degrades to the direct-path backend ("none").
func seatbeltContainmentInfo(available bool, path string) *ContainmentInfo {
	if !available {
		return &ContainmentInfo{
			Backend:      backendNone,
			Available:    false,
			Enforcements: []string{},
		}
	}
	return &ContainmentInfo{
		Backend:   SeatbeltBackendID,
		Available: true,
		Path:      path,
		Enforcements: []string{
			EnforcementFilesystemWriteContainment,
			EnforcementNetworkDeny,
		},
	}
}

// String renders the containment level using the platform-neutral
// vocabulary reported on the wire (level field of SandboxContainment).
func (l ContainmentLevel) String() string {
	switch l {
	case ContainmentNone:
		return "none"
	case ContainmentPreferred:
		return "preferred"
	case ContainmentRequired:
		return "required"
	default:
		return "unknown"
	}
}

// predictedBackend reports which containment backend WILL carry out execs
// for a sandbox at the given level, mirroring exec.buildStartOpts's
// dispatch: level None (or an unavailable backend) falls through to the
// direct path ("none"); otherwise the host backend runs. Used to derive the
// sandbox-response containment before any launch has happened.
func predictedBackend(level ContainmentLevel, info *ContainmentInfo) string {
	if level == ContainmentNone || info == nil || !info.Available {
		return backendNone
	}
	return info.Backend
}

// EffectiveContainment builds the containment report for a launch that ran
// (or will run) under backendID. Enforcements are attributed only when a
// real backend ran and it matches the probed backend; the direct path
// ("none") enforces nothing. Shared by the sandbox-response derivation and
// per-launch provenance so both describe containment identically.
func EffectiveContainment(level ContainmentLevel, backendID string, info *ContainmentInfo) *types.SandboxContainment {
	enforcements := []string{}
	if backendID != backendNone && info != nil && info.Backend == backendID {
		enforcements = append(enforcements, info.Enforcements...)
	}
	return &types.SandboxContainment{
		Level:        level.String(),
		Backend:      backendID,
		Enforcements: enforcements,
	}
}

// AdjustForLaunch drops enforcements the launch configuration disables from
// a per-launch containment report. bwrap's network-deny comes from
// --unshare-net, which BuildBwrapArgs omits when the launch allows network
// — leaving the claim in place would over-state what this launch enforced.
// Sandbox-level (predicted) reports keep the full backend capability list;
// only per-launch provenance is adjusted. Mutates and returns cont.
func AdjustForLaunch(cont *types.SandboxContainment, allowNetwork bool) *types.SandboxContainment {
	if cont == nil || !allowNetwork {
		return cont
	}
	kept := make([]string, 0, len(cont.Enforcements))
	for _, e := range cont.Enforcements {
		if e != EnforcementNetworkDeny {
			kept = append(kept, e)
		}
	}
	cont.Enforcements = kept
	return cont
}

// DeriveWorkspaceLayout computes the negotiated workspace contract for a
// sandbox: the agent-visible workspace path, the path-illusion flag, and
// the effective containment. It is the ONE server-side derivation the
// sandbox-returning handlers use so workspacePath, pathIllusion, and
// containment can never drift apart.
//
// Layout invariant: pathIllusion is true exactly when the predicted
// backend provides path-illusion enforcement, and in that case
// workspacePath is NamespaceWorkspacePath; otherwise workspacePath is the
// host mergedDir (identity layout).
func DeriveWorkspaceLayout(level ContainmentLevel, info *ContainmentInfo, mergedDir string) (workspacePath string, pathIllusion bool, cont *types.SandboxContainment) {
	backendID := predictedBackend(level, info)
	cont = EffectiveContainment(level, backendID, info)
	pathIllusion = hasEnforcement(cont.Enforcements, EnforcementPathIllusion)
	workspacePath = mergedDir
	if pathIllusion {
		workspacePath = NamespaceWorkspacePath
	}
	return workspacePath, pathIllusion, cont
}

// hasEnforcement reports whether name is present in the enforcement list.
func hasEnforcement(enforcements []string, name string) bool {
	for _, e := range enforcements {
		if e == name {
			return true
		}
	}
	return false
}

// Package volumeremediation owns host-side repair of storage volumes: checking
// a filesystem, repairing it, and returning a volume to a read/write mount.
//
// It lives in the control plane on purpose. Detection and remediation of host
// state are control-plane responsibilities; a scenario may observe, request and
// report that state but must not carry a private host-repair implementation.
// Scenarios call this package (or its CLI surface) and render its typed result.
//
// Nothing here elevates privilege on its own. On Linux the primary path is
// udisks2, which polkit authorises for a non-system device from an active
// session with no password and no root. Hosts without that path fall back to
// the setup-installed privilege broker, whose immutable action registry is the
// only place elevation is ever granted. `vrooli setup` remains the single
// consent boundary either way.
package volumeremediation

import (
	"errors"
	"fmt"
	"strings"
)

// Action is the controlled vocabulary of volume remediation. It is deliberately
// small and contains no formatting, partitioning, or data-clearing action:
// those destroy backups rather than recovering them, and no remediation flow
// should be able to reach them by accident.
type Action string

const (
	// ActionInspect reports current volume state. Read-only.
	ActionInspect Action = "inspect"
	// ActionCheck verifies filesystem consistency without writing. Read-only.
	ActionCheck Action = "check"
	// ActionRepair writes filesystem metadata corrections. Destructive in the
	// narrow sense that it can discard inconsistent metadata.
	ActionRepair Action = "repair"
	// ActionUnmount detaches the volume so a repair can run against it.
	ActionUnmount Action = "unmount"
	// ActionMountReadWrite returns the volume to a writable mount.
	ActionMountReadWrite Action = "mount_read_write"
)

// Destructive reports whether an action can modify on-disk state. Only repair
// qualifies; unmount and mount change attachment, not content.
func (a Action) Destructive() bool { return a == ActionRepair }

// Known reports whether the action is in the registry at all.
func (a Action) Known() bool {
	switch a {
	case ActionInspect, ActionCheck, ActionRepair, ActionUnmount, ActionMountReadWrite:
		return true
	}
	return false
}

// Device identifies the volume a request targets. The caller supplies the
// identity it expects; the service re-observes and refuses on any mismatch, so
// a replugged or renumbered disk can never inherit another disk's approval.
type Device struct {
	// Path is the OS device specifier: /dev/sda1, /dev/disk2s1, or a Windows
	// drive letter such as "E:".
	Path string
	// Filesystem selects the repair adapter. An unrecognised filesystem is
	// refused rather than guessed at.
	Filesystem string
	// UUID and Serial are the identifiers that survive a replug. At least one
	// is required for a mutating action.
	UUID   string
	Serial string
	// Mountpoint is where the volume is currently attached, when it is.
	Mountpoint string
	// TotalBytes guards against a same-path different-disk swap.
	TotalBytes int64
}

// StableIdentity reports whether the device carries an identifier that survives
// a replug. Device paths are assignment-order artifacts and never qualify.
func (d Device) StableIdentity() bool {
	return strings.TrimSpace(d.UUID) != "" || strings.TrimSpace(d.Serial) != ""
}

// Matches reports whether an observed device is the one that was approved.
// Mountpoint is excluded on purpose: remediation unmounts the volume itself,
// so guarding on the mountpoint would reject the flow mid-sequence.
func (d Device) Matches(observed Device) bool {
	if !equalFoldOptional(d.Filesystem, observed.Filesystem) {
		return false
	}
	if d.TotalBytes != 0 && observed.TotalBytes != 0 && d.TotalBytes != observed.TotalBytes {
		return false
	}
	if d.StableIdentity() && observed.StableIdentity() {
		return equalFoldOptional(d.UUID, observed.UUID) && equalOptional(d.Serial, observed.Serial)
	}
	if strings.TrimSpace(d.Path) == "" || strings.TrimSpace(observed.Path) == "" {
		return false
	}
	return strings.TrimSpace(d.Path) == strings.TrimSpace(observed.Path)
}

// Describe renders a compact identity for confirmation phrases and audit rows.
func (d Device) Describe() string {
	out := fmt.Sprintf("device=%s fs=%s", strings.TrimSpace(d.Path), strings.TrimSpace(d.Filesystem))
	if d.UUID != "" {
		out += " uuid=" + d.UUID
	}
	if d.Serial != "" {
		out += " serial=" + d.Serial
	}
	if d.TotalBytes != 0 {
		out += fmt.Sprintf(" size=%d", d.TotalBytes)
	}
	return out
}

// State is the observed condition of a volume.
type State struct {
	Device Device
	// Mounted and ReadOnly describe attachment, not health.
	Mounted  bool
	ReadOnly bool
	// Dirty reports a filesystem that declares it needs a check. Unknown stays
	// unknown: a host that cannot tell must not report health.
	Dirty        Tristate
	Evidence     string
	Observations []string
}

// Tristate distinguishes "no" from "could not determine". Collapsing those two
// into a boolean is how an unverifiable volume comes to look healthy.
type Tristate string

const (
	TristateUnknown Tristate = "unknown"
	TristateYes     Tristate = "yes"
	TristateNo      Tristate = "no"
)

// Request asks for one remediation action against one device.
type Request struct {
	Action Action
	Device Device
	// DesiredMountpoint is honoured only by ActionMountReadWrite, and only
	// when it sits under a permitted media root.
	DesiredMountpoint string
	// AcknowledgeDataLoss must be set for a destructive action. The service
	// enforces this independently of any caller-side confirmation, because a
	// caller that forgets its own gate must still not be able to write.
	AcknowledgeDataLoss bool
	// DryRun runs every validation and reports the command that would run
	// without executing it.
	DryRun bool
}

// Result is the typed outcome of a remediation action.
type Result struct {
	Action Action
	Device Device
	// Status is one of: verified, changed, already_satisfied, refused,
	// unsupported, failed.
	Status string
	// Changed reports whether host state actually moved.
	Changed bool
	// DryRun echoes whether this was a rehearsal.
	DryRun bool
	// Command is the exact argv that ran, or would have run. Recording it is
	// what makes the operation auditable rather than a black box.
	Command []string
	// Backend names which execution path served the request.
	Backend string
	// Detail is bounded, redacted tool output.
	Detail string
	// Consistent is the verdict of a check action: whether the filesystem is
	// internally consistent. It is a tristate because a check that ran and
	// found problems, a check that ran and found none, and an action that
	// never checked at all are three different things — and only the middle
	// one means the volume is healthy.
	Consistent Tristate
	// State is the post-action observation when one could be taken.
	State State
}

const (
	StatusVerified         = "verified"
	StatusChanged          = "changed"
	StatusAlreadySatisfied = "already_satisfied"
	StatusRefused          = "refused"
	StatusUnsupported      = "unsupported"
	StatusFailed           = "failed"
)

// ErrRefused is returned when a safety gate rejects a request. It is separate
// from a tool failure because the two need different operator responses.
type ErrRefused struct{ Reason string }

func (e ErrRefused) Error() string { return "volume remediation refused: " + e.Reason }

// ErrUnsupported reports that this platform has no adapter for the request.
// It carries the native command an operator can run, so an unsupported host
// still leaves the operator with a way forward instead of a dead end.
type ErrUnsupported struct {
	Reason          string
	OperatorCommand string
}

func (e ErrUnsupported) Error() string {
	if e.OperatorCommand == "" {
		return "volume remediation unsupported: " + e.Reason
	}
	return "volume remediation unsupported: " + e.Reason + "; run: " + e.OperatorCommand
}

// ErrInvalid is a typed validation error.
type ErrInvalid struct {
	Field  string
	Reason string
}

func (e ErrInvalid) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }

// ErrDeviceChanged reports that the observed device is not the approved one.
var ErrDeviceChanged = errors.New("observed device identity does not match the approved device")

func equalOptional(a, b string) bool {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if a == "" || b == "" {
		return true
	}
	return a == b
}

func equalFoldOptional(a, b string) bool {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if a == "" || b == "" {
		return true
	}
	return strings.EqualFold(a, b)
}

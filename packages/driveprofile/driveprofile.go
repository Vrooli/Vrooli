// Package driveprofile is the single source of truth for how a storage
// filesystem behaves as a backup destination.
//
// It answers the questions that readiness, remediation, setup, and the UI must
// otherwise each re-derive:
//
//   - does the OS expose a trustworthy cleanliness signal for this filesystem?
//   - which tool checks it, and which tool repairs it?
//   - is the repair tool a real metadata repair or only a dirty-flag reset?
//   - is the filesystem a good backup destination on this platform?
//   - what should an operator do when it is not?
//
// Both the control plane (internal/volumeremediation) and Data Backup Manager
// read this table. Adding a filesystem means teaching one table, not editing a
// switch in every layer.
package driveprofile

import "strings"

// Family is the repair/behaviour family a filesystem belongs to. Several
// on-disk names share one family (ext2/ext3/ext4, ntfs/ntfs3, fat variants).
type Family string

const (
	FamilyExt   Family = "ext"
	FamilyXFS   Family = "xfs"
	FamilyNTFS  Family = "ntfs"
	FamilyExFAT Family = "exfat"
	FamilyFAT   Family = "fat"
	FamilyAPFS  Family = "apfs"
	FamilyHFS   Family = "hfs"
	FamilyOther Family = "other"
)

// StateSignal describes how, if at all, the running OS exposes whether the
// filesystem is clean or needs a check.
type StateSignal string

const (
	// StateSignalKernel means the mounted kernel driver publishes a bounded
	// error counter (for example ext4's sysfs errors_count). A zero count is
	// real evidence of cleanliness.
	StateSignalKernel StateSignal = "kernel"
	// StateSignalVolinfo means a kernel driver publishes a state summary whose
	// meaning is ambiguous for a read/write mount (for example ntfs3). It is
	// evidence, but not proof of clean.
	StateSignalVolinfo StateSignal = "volinfo"
	// StateSignalNone means the OS exposes nothing usable. A userspace driver
	// (ntfs-3g) or a filesystem without a state surface lands here.
	StateSignalNone StateSignal = "none"
)

// Suitability is the default posture for using the filesystem as a backup
// destination on this platform.
type Suitability string

const (
	// Recommended is a first-class choice with a real check/repair tool.
	Recommended Suitability = "recommended"
	// Usable is acceptable, with a documented tradeoff.
	Usable Suitability = "usable"
	// Caution works but has a known fault or repair limitation that should be
	// visible to the operator.
	Caution Suitability = "caution"
	// Avoid should not be used for a new backup destination.
	Avoid Suitability = "avoid"
)

// Profile is one filesystem's complete behaviour.
type Profile struct {
	// Name is the canonical display name.
	Name string
	// Family groups aliases that share repair behaviour.
	Family Family
	// Aliases are the on-disk/mount names that resolve to this profile.
	Aliases []string
	// StateSignal says whether this platform exposes cleanliness.
	StateSignal StateSignal
	// StateNote explains the signal or its absence in operator terms.
	StateNote string
	// RepairIsMetadata is true when the repair tool actually repairs
	// filesystem metadata. False means it only clears a dirty flag and fixes a
	// small set of known problems (ntfsfix).
	RepairIsMetadata bool
	// RepairLimitation is the operator-facing sentence that says, in one line,
	// what this platform's repair tool cannot do. Empty means the tool is a real
	// metadata repair, so there is nothing to warn about.
	RepairLimitation string
	// LinuxCheck / LinuxRepair are argv templates for native Linux tools. The
	// literal "{device}" is replaced with the device path.
	LinuxCheck  []string
	LinuxRepair []string
	// CrossPlatform is true when the filesystem is natively readable on
	// Windows and macOS as well as Linux.
	CrossPlatform bool
	// Suitability is the default backup-destination posture on Linux.
	Suitability Suitability
	// SuitabilityByOS overrides Suitability on another platform. A platform
	// where the filesystem is native (NTFS on Windows, APFS on macOS) is a
	// first-class choice there even when it is only usable on Linux.
	SuitabilityByOS map[string]Suitability
	// NoteByOS explains the platform difference in operator terms.
	NoteByOS map[string]string
	// Summary is one sentence for an operator.
	Summary string
	// NextAction is what to do when the profile is a caution or avoid.
	NextAction string
}

var profiles = []Profile{
	{
		Name: "ext4", Family: FamilyExt, Aliases: []string{"ext4", "ext3", "ext2"},
		StateSignal:      StateSignalKernel,
		StateNote:        "the kernel exposes an accumulated error count; zero is bounded evidence of a clean filesystem",
		RepairIsMetadata: true,
		LinuxCheck:       []string{"e2fsck", "-f", "-n", "{device}"},
		LinuxRepair:      []string{"e2fsck", "-f", "-y", "{device}"},
		CrossPlatform:    false,
		Suitability:      Recommended,
		SuitabilityByOS:  map[string]Suitability{"windows": Avoid, "darwin": Avoid},
		NoteByOS: map[string]string{
			"windows": "Windows has no native ext4 support",
			"darwin":  "macOS has no native ext4 support",
		},
		Summary: "ext4 is the recommended Linux backup filesystem: journaled, with a real offline check and repair tool.",
	},
	{
		Name: "exFAT", Family: FamilyExFAT, Aliases: []string{"exfat"},
		StateSignal:      StateSignalNone,
		StateNote:        "the kernel does not expose a cleanliness signal for exFAT",
		RepairIsMetadata: true,
		LinuxCheck:       []string{"fsck.exfat", "-n", "{device}"},
		LinuxRepair:      []string{"fsck.exfat", "-y", "{device}"},
		CrossPlatform:    true,
		Suitability:      Usable,
		Summary:          "exFAT is readable on Linux, Windows and macOS, but it has no journal; an unclean detach is more likely to need a check.",
		NextAction:       "prefer ext4 for a Linux-only drive; keep exFAT only when cross-platform readability is required",
	},
	{
		Name: "NTFS", Family: FamilyNTFS, Aliases: []string{"ntfs", "ntfs3"},
		StateSignal:      StateSignalVolinfo,
		StateNote:        "ntfs3's dirty flag cannot be distinguished from the current read/write mount's in-use marker; ntfs-3g exposes no state at all",
		RepairIsMetadata: false,
		RepairLimitation: "the only repair Linux offers for NTFS clears the dirty flag; it cannot repair a damaged metadata record",
		LinuxCheck:       []string{"ntfsfix", "-n", "{device}"},
		LinuxRepair:      []string{"ntfsfix", "-d", "{device}"},
		CrossPlatform:    true,
		Suitability:      Caution,
		SuitabilityByOS:  map[string]Suitability{"windows": Recommended},
		NoteByOS: map[string]string{
			"windows": "NTFS is native on Windows, where chkdsk can repair its metadata",
			"darwin":  "macOS reads NTFS but cannot write it without a third-party driver, and cannot repair it",
		},
		Summary:    "NTFS is journaled and cross-platform, but Linux repair (ntfsfix) only clears the dirty flag; it cannot repair unreadable metadata records.",
		NextAction: "prefer ext4 for a Linux-hosted backup drive, or exFAT when the drive must stay cross-platform; a Windows chkdsk /f is required for NTFS metadata damage",
	},
	{
		Name: "FAT32", Family: FamilyFAT, Aliases: []string{"vfat", "fat", "fat32", "msdos"},
		StateSignal:      StateSignalNone,
		StateNote:        "the kernel does not expose a cleanliness signal for FAT",
		RepairIsMetadata: true,
		LinuxCheck:       []string{"fsck.fat", "-n", "{device}"},
		LinuxRepair:      []string{"fsck.fat", "-a", "{device}"},
		CrossPlatform:    true,
		Suitability:      Avoid,
		Summary:          "FAT32 has a 4 GiB per-file limit and no journal; it is not suitable for a serious backup repository.",
		NextAction:       "choose ext4, or exFAT if cross-platform readability is required",
	},
	{
		Name: "APFS", Family: FamilyAPFS, Aliases: []string{"apfs"},
		StateSignal:     StateSignalNone,
		StateNote:       "APFS is managed by macOS; Linux has no native check or repair path",
		CrossPlatform:   false,
		Suitability:     Avoid,
		SuitabilityByOS: map[string]Suitability{"darwin": Recommended},
		NoteByOS:        map[string]string{"darwin": "APFS is native on macOS, where diskutil owns check and repair"},
		Summary:         "APFS is macOS-only for native repair; use it only on a macOS host where diskutil owns it.",
		NextAction:      "on macOS use `diskutil verifyVolume` / `diskutil repairVolume`; on Linux choose ext4 or exFAT",
	},
	{
		Name: "HFS+", Family: FamilyHFS, Aliases: []string{"hfs", "hfs+"},
		StateSignal:     StateSignalNone,
		StateNote:       "HFS+ is managed by macOS; Linux has no native check or repair path",
		CrossPlatform:   false,
		Suitability:     Avoid,
		SuitabilityByOS: map[string]Suitability{"darwin": Caution},
		NoteByOS:        map[string]string{"darwin": "HFS+ is legacy on macOS; APFS is preferred"},
		Summary:         "HFS+ is macOS-only for native repair; use it only on a macOS host where diskutil owns it.",
		NextAction:      "on macOS use `diskutil verifyVolume` / `diskutil repairVolume`; on Linux choose ext4 or exFAT",
	},
}

var byAlias = func() map[string]Profile {
	out := make(map[string]Profile, len(profiles)*3)
	for _, p := range profiles {
		for _, alias := range p.Aliases {
			out[alias] = p
		}
	}
	return out
}()

// Lookup resolves an on-disk filesystem name (as reported by the mount table)
// to its profile. Unknown filesystems are not guessed at.
func Lookup(filesystem string) (Profile, bool) {
	p, ok := byAlias[strings.ToLower(strings.TrimSpace(filesystem))]
	return p, ok
}

// All returns the canonical profiles (one per family).
func All() []Profile {
	out := make([]Profile, len(profiles))
	copy(out, profiles)
	return out
}

// Names returns every on-disk alias, sorted by the canonical order. It is used
// by validation that must reject an unlisted filesystem rather than guess.
func Names() []string {
	var out []string
	for _, p := range profiles {
		out = append(out, p.Aliases...)
	}
	return out
}

// NormalizeOS maps a Go runtime.GOOS value (or a person's spelling of it) to
// the key used by the per-platform fields. An unknown value is returned
// unchanged, so it simply falls back to the Linux default.
func NormalizeOS(goos string) string {
	switch strings.ToLower(strings.TrimSpace(goos)) {
	case "darwin", "macos", "mac", "osx":
		return "darwin"
	case "windows", "win", "win32":
		return "windows"
	case "linux":
		return "linux"
	default:
		return strings.ToLower(strings.TrimSpace(goos))
	}
}

// SuitabilityOn reports the posture for this filesystem on one platform.
func (p Profile) SuitabilityOn(goos string) Suitability {
	os := NormalizeOS(goos)
	if os == "linux" {
		return p.Suitability
	}
	if suitability, ok := p.SuitabilityByOS[os]; ok {
		return suitability
	}
	return p.Suitability
}

// NoteOn explains the platform difference, or reports the absence of one.
func (p Profile) NoteOn(goos string) string {
	return p.NoteByOS[NormalizeOS(goos)]
}

// DeviceCommand renders a template with the device path.
func DeviceCommand(template []string, device string) []string {
	if len(template) == 0 {
		return nil
	}
	out := make([]string, len(template))
	for i, part := range template {
		out[i] = strings.ReplaceAll(part, "{device}", device)
	}
	return out
}

// CheckCommand returns the native check argv for a filesystem, if one exists.
func CheckCommand(filesystem, device string) ([]string, bool) {
	p, ok := Lookup(filesystem)
	if !ok || len(p.LinuxCheck) == 0 {
		return nil, false
	}
	return DeviceCommand(p.LinuxCheck, device), true
}

// RepairCommand returns the native repair argv for a filesystem, if one exists.
func RepairCommand(filesystem, device string) ([]string, bool) {
	p, ok := Lookup(filesystem)
	if !ok || len(p.LinuxRepair) == 0 {
		return nil, false
	}
	return DeviceCommand(p.LinuxRepair, device), true
}

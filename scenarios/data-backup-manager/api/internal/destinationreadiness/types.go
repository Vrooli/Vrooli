// Package destinationreadiness evaluates whether a local filesystem location is
// a safe backup destination and gates any drive-preparation action behind
// explicit identity and confirmation checks.
package destinationreadiness

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"data-backup-manager/internal/sysmounts"
)

// CheckSeverity is the machine-readable readiness outcome for one check.
type CheckSeverity string

const (
	SeverityPass    CheckSeverity = "pass"
	SeverityWarning CheckSeverity = "warning"
	SeverityFail    CheckSeverity = "fail"
	SeverityUnknown CheckSeverity = "unknown"
)

// PreparationAction is the controlled vocabulary for destination preparation.
type PreparationAction string

const (
	ActionCreateSubdir   PreparationAction = "create_subdir"
	ActionRelabel        PreparationAction = "relabel"
	ActionClearDirectory PreparationAction = "clear_directory"
	ActionFormat         PreparationAction = "format"

	// Remediation actions return a destination the kernel refuses to mount
	// read/write to a usable state. They are executed by the control plane,
	// which owns host state; this scenario plans, confirms, and reports them.
	//
	// They are separate actions rather than one "fix it" call so each step is
	// individually confirmed, individually auditable, and individually
	// refusable — and so a partially-completed sequence can be resumed instead
	// of restarted.
	ActionUnmount          PreparationAction = "unmount"
	ActionCheckFilesystem  PreparationAction = "check_filesystem"
	ActionRepairFilesystem PreparationAction = "repair_filesystem"
	ActionMountReadWrite   PreparationAction = "mount_read_write"
)

// IsRemediation reports whether an action targets host volume state rather than
// the destination directory. Remediation actions intentionally change mount
// state, so they are inspected and identity-checked by device rather than by
// path.
func (a PreparationAction) IsRemediation() bool {
	switch a {
	case ActionUnmount, ActionCheckFilesystem, ActionRepairFilesystem, ActionMountReadWrite:
		return true
	default:
		return false
	}
}

// DeviceIdentity binds a preparation plan to the observed device state. It is
// deliberately redundant so unplug/replug, relabeling, or device-path drift can
// be detected before executing a plan.
type DeviceIdentity struct {
	DevicePath string
	Mountpoint string
	Label      string
	Filesystem string
	TotalBytes int64
	Model      string
	Serial     string
	UUID       string
}

// StableString is a compact, human-readable identity used in confirmation
// phrases and equality checks. Empty optional fields are skipped.
func (i DeviceIdentity) StableString() string {
	out := fmt.Sprintf("device=%s mount=%s fs=%s size=%d", i.DevicePath, i.Mountpoint, i.Filesystem, i.TotalBytes)
	if i.Label != "" {
		out += " label=" + i.Label
	}
	if i.Model != "" {
		out += " model=" + i.Model
	}
	if i.Serial != "" {
		out += " serial=" + i.Serial
	}
	if i.UUID != "" {
		out += " uuid=" + i.UUID
	}
	return out
}

// Matches reports whether the safety-critical identity fields still match.
func (i DeviceIdentity) Matches(other DeviceIdentity) bool {
	if normalizePath(i.DevicePath) != normalizePath(other.DevicePath) || normalizePath(i.Mountpoint) != normalizePath(other.Mountpoint) {
		return false
	}
	if !equalOptional(i.Filesystem, other.Filesystem, true) || !equalOptional(i.UUID, other.UUID, true) || !equalOptional(i.Serial, other.Serial, false) {
		return false
	}
	if i.TotalBytes != 0 && other.TotalBytes != 0 && i.TotalBytes != other.TotalBytes {
		return false
	}
	// Label and model are mutable/descriptive and are intentionally not identity
	// guards. A relabel operation must not make an unchanged disk look foreign.
	return true
}

// StrongIdentity reports whether this identity carries at least one field that
// survives a replug: a filesystem UUID or a device serial. Device paths and
// mountpoints are assignment-order artifacts, so a plan that mutates a volume
// must not rely on them alone.
func (i DeviceIdentity) StrongIdentity() bool {
	return strings.TrimSpace(i.UUID) != "" || strings.TrimSpace(i.Serial) != ""
}

// MatchesDevice reports whether the same physical volume is still present,
// deliberately ignoring the mountpoint. Remediation actions unmount the volume
// on purpose, so Matches — which guards the mountpoint — would reject a plan
// mid-flight for doing exactly what it was authorized to do.
//
// It is stricter than Matches where it matters: when both sides carry a strong
// identifier it must match, and when neither does the device path must match
// exactly rather than being waved through as "optional".
func (i DeviceIdentity) MatchesDevice(other DeviceIdentity) bool {
	if !equalOptional(i.Filesystem, other.Filesystem, true) {
		return false
	}
	if i.TotalBytes != 0 && other.TotalBytes != 0 && i.TotalBytes != other.TotalBytes {
		return false
	}
	if i.StrongIdentity() && other.StrongIdentity() {
		// Compare only the identifiers both sides actually published; a host
		// that exposes a UUID but no serial must still be able to match.
		return equalOptional(i.UUID, other.UUID, true) && equalOptional(i.Serial, other.Serial, false)
	}
	// Guard the raw values: normalizePath collapses an empty path to ".", so
	// comparing normalized forms would make two identity-free volumes match.
	if strings.TrimSpace(i.DevicePath) == "" || strings.TrimSpace(other.DevicePath) == "" {
		return false
	}
	return normalizePath(i.DevicePath) == normalizePath(other.DevicePath)
}

func normalizePath(p string) string {
	p = filepath.Clean(strings.TrimSpace(strings.ReplaceAll(p, "\\", string(filepath.Separator))))
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
	}
	return p
}

func equalOptional(a, b string, fold bool) bool {
	if a == "" || b == "" {
		return true
	}
	if fold {
		return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
	}
	return strings.TrimSpace(a) == strings.TrimSpace(b)
}

// Inspection is a read-only snapshot of a mounted destination candidate.
type Inspection struct {
	// LocationExists and LocationIsDirectory describe the requested candidate,
	// not merely its nearest mounted parent. A missing child under a mounted
	// volume must never be presented as a ready destination.
	LocationExists      bool
	LocationIsDirectory bool
	LocationError       string
	Identity            DeviceIdentity
	FreeBytes           int64
	ReadOnly            bool
	Removable           bool
	DriveClass          string
	MountOptions        []string
	// MountDriver is the kernel-side driver serving the mount, empty when the
	// volume is not mounted. Suitability is a property of the driver, not only
	// the on-disk format: the same NTFS volume is a kernel-panic risk under
	// `ntfs3` and an ordinary I/O risk under `ntfs-3g`.
	MountDriver     string
	FilesystemState sysmounts.FilesystemState
	// ReadOnlyCause attributes a read-only mount so a report can name the one
	// remediation that applies instead of listing every possibility.
	ReadOnlyCause sysmounts.ReadOnlyCause
	// DeviceWriteProtected marks block-layer write protection, which no
	// filesystem repair can clear.
	DeviceWriteProtected bool
	// StateEvidence names the source behind the state verdict.
	StateEvidence string
	// Mounted reports whether the volume is mounted at all. A device-scoped
	// inspection of an intentionally unmounted volume is a valid observation,
	// not an error, because remediation passes through that state.
	Mounted         bool
	TopLevelEntries []string
	NonEmptyRoot    bool
	InstallerMedia  bool
	Platform        string
	ReadDirError    string
}

// CheckResult is one structured readiness rule result.
type CheckResult struct {
	Code       string
	Severity   CheckSeverity
	Message    string
	NextAction string
}

// Report is the readiness result returned by Analyze.
type Report struct {
	Location                       string
	OverallSeverity                CheckSeverity
	Identity                       DeviceIdentity
	Checks                         []CheckResult
	RecommendedDestinationLocation string
	RecommendedAction              string
	Platform                       string
	Confidence                     string
	EvidenceSource                 string
	ObservedAt                     time.Time
	RepairSteps                    []string
	// ReadOnlyCause, FilesystemState, DeviceWriteProtected and Mounted are the
	// machine-readable state facts behind the checks. Consumers map a
	// remediation from these instead of pattern-matching on check prose.
	ReadOnlyCause        sysmounts.ReadOnlyCause
	FilesystemState      sysmounts.FilesystemState
	DeviceWriteProtected bool
	Mounted              bool
}

// AnalyzeInput controls a read-only readiness analysis.
type AnalyzeInput struct {
	Location              string
	ProposedSubdir        string
	SelectedTargetBytes   int64
	RetentionCopies       int
	ProtectedPaths        []string
	ExistingDestinations  []string
	CrossPlatformRequired bool
}

// PlanInput requests a non-mutating preparation plan.
type PlanInput struct {
	Location       string
	Action         PreparationAction
	DesiredSubdir  string
	DesiredLabel   string
	DesiredFS      string
	ExpectedDevice DeviceIdentity
	ProtectedPaths []string
}

// Plan is a preparation plan. It is data until Execute is called.
type Plan struct {
	ID                 string
	Action             PreparationAction
	Location           string
	TargetPath         string
	Identity           DeviceIdentity
	DesiredLabel       string
	DesiredFS          string
	RequiresConfirm    bool
	Destructive        bool
	ConfirmationPhrase string
	Supported          bool
	UnsupportedReason  string
}

// ExecuteInput executes or dry-runs a preparation plan.
type ExecuteInput struct {
	Plan                Plan
	Confirmation        string
	DryRun              bool
	AcknowledgeDataLoss bool
}

// ExecuteResult reports whether an execution was dry-run and what action would
// or did run. The remediation fields carry the control plane's typed outcome so
// a refusal or an unsupported platform reaches the operator with its reason and
// its next command intact, rather than collapsing into a bare failure.
type ExecuteResult struct {
	DryRun   bool
	Action   PreparationAction
	Location string

	Status          string
	Changed         bool
	Backend         string
	Command         []string
	Detail          string
	OperatorCommand string
	RefusalReason   string
	// Consistent is a check-filesystem verdict: "unknown" | "yes" | "no".
	Consistent string
}

// RemediationOutcome is the control plane's typed answer for one remediation
// action.
type RemediationOutcome struct {
	Status          string
	Changed         bool
	Backend         string
	Command         []string
	Detail          string
	OperatorCommand string
	RefusalReason   string
	// Consistent is a check-filesystem verdict: "unknown" | "yes" | "no". It is
	// separate from Status because a check that ran and found an inconsistent
	// filesystem is a successful action with a bad answer.
	Consistent string
}

// Satisfied reports whether the requested end state holds, whether or not this
// call produced it.
func (o RemediationOutcome) Satisfied() bool {
	switch o.Status {
	case "verified", "changed", "already_satisfied":
		return true
	default:
		return false
	}
}

// ErrInvalidReadiness is a typed validation error.
type ErrInvalidReadiness struct {
	Field  string
	Reason string
}

func (e ErrInvalidReadiness) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }

// ErrPreparationRefused is returned when safety checks reject execution.
type ErrPreparationRefused struct {
	Reason string
}

func (e ErrPreparationRefused) Error() string { return "preparation refused: " + e.Reason }

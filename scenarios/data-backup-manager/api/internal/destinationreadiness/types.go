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
)

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
	FilesystemState     sysmounts.FilesystemState
	TopLevelEntries     []string
	NonEmptyRoot        bool
	InstallerMedia      bool
	Platform            string
	ReadDirError        string
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
// or did run.
type ExecuteResult struct {
	DryRun   bool
	Action   PreparationAction
	Location string
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

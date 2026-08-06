package hostreqkit

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqspec"
)

var ErrUnsupportedPlatform = errors.New("unsupported platform")

type (
	SupportClass   string
	ExecutionState string
)

const (
	SupportSupported     SupportClass = "supported"
	SupportUnsupported   SupportClass = "unsupported"
	SupportNotApplicable SupportClass = "not_applicable"
	SupportManualOnly    SupportClass = "manual_only"
)

const (
	ExecutionPending              ExecutionState = "pending"
	ExecutionAlreadyPresent       ExecutionState = "already_present"
	ExecutionWouldInstall         ExecutionState = "would_install"
	ExecutionWouldApply           ExecutionState = "would_apply"
	ExecutionInstalled            ExecutionState = "installed"
	ExecutionApplied              ExecutionState = "applied"
	ExecutionRebootRequired       ExecutionState = "reboot_required"
	ExecutionManualActionRequired ExecutionState = "manual_action_required"
	ExecutionUnsupported          ExecutionState = "unsupported"
	ExecutionNotApplicable        ExecutionState = "not_applicable"
	ExecutionFailed               ExecutionState = "failed"
)

type Host struct {
	OS              string   `json:"os"`
	PackageManager  string   `json:"package_manager,omitempty"`
	SupportsSetup   bool     `json:"supports_setup"`
	SupportsDevelop bool     `json:"supports_develop"`
	SupportsSysctl  bool     `json:"supports_sysctl"`
	SupportsSystemd bool     `json:"supports_systemd"`
	Notes           []string `json:"notes,omitempty"`
}

type EnsureOptions struct {
	Environment string
	SudoMode    string
	DryRun      bool
	AutoInstall bool
	// IncludeOptional tells the runtime to also call Apply on optional
	// (Required=false) items that come back Pending from Inspect. By default
	// optionals are skipped — their Pending state is shown to the operator
	// but no automatic install is attempted. Operators opt in via
	// `vrooli setup --include-optional`.
	IncludeOptional bool
	// MaintenanceWindow acknowledges that a safeguard may interrupt an active
	// graphical or remote-desktop session while applying host state.
	MaintenanceWindow bool
	Stdout            io.Writer
	Stderr            io.Writer
}

// BlockingReason categorises *why* an item is not yet satisfied. It is
// orthogonal to ExecutionState — a `failed` item with reason `needs_sudo`
// renders very differently from a generic failure, even though the underlying
// state is the same. Empty value means "no specific blocker" (renderer falls
// back to the default Pending / Failed framing).
type BlockingReason string

const (
	// BlockingNone is the zero value. The renderer treats it as "no specific
	// reason recorded" and uses default group placement.
	BlockingNone BlockingReason = ""
	// BlockingNeedsSudo: an Apply call returned ErrSudoSkipped /
	// ErrSudoUnavailable. Re-running with sudo (or --sudo-mode=ask) unblocks.
	BlockingNeedsSudo BlockingReason = "needs_sudo"
	// BlockingOptionalSkipped: optional item still Pending after Inspect; the
	// runtime did not attempt Apply because IncludeOptional=false. Operator
	// opts in via --include-optional.
	BlockingOptionalSkipped BlockingReason = "optional_skipped"
	// BlockingOperatorChoiceMissing means an optional item is pending because
	// no durable operator choice has been recorded yet.
	BlockingOperatorChoiceMissing BlockingReason = "operator_choice_missing"
	// BlockingOperatorDeclined means the operator explicitly declined the
	// optional host change in operator-state.json.
	BlockingOperatorDeclined BlockingReason = "operator_declined"
	// BlockingInvalidParameter means declared safeguard config failed its
	// manifest schema and was not passed to the handler.
	BlockingInvalidParameter BlockingReason = "invalid_parameter"
	// BlockingNeedsEnv: handler reported it is waiting on an env var or other
	// operator-supplied configuration (e.g. netconsole target).
	BlockingNeedsEnv BlockingReason = "needs_env"
	// BlockingNeedsReboot: install/apply succeeded but the machine must be
	// rebooted before the change takes effect.
	BlockingNeedsReboot BlockingReason = "needs_reboot"
	// BlockingManual: handler is manual-only; operator must run the
	// documented procedure.
	BlockingManual BlockingReason = "manual"
	// BlockingNeedsMaintenanceWindow marks a deliberately withheld live change
	// while a remote desktop server is active.
	BlockingNeedsMaintenanceWindow BlockingReason = "maintenance_window"
)

type ItemStatus struct {
	Name                 string                     `json:"name"`
	Kind                 hostreqspec.Kind           `json:"kind"`
	Command              string                     `json:"command,omitempty"`
	Version              string                     `json:"version,omitempty"`
	Installed            bool                       `json:"installed"`
	Applied              bool                       `json:"applied,omitempty"`
	Required             bool                       `json:"required"`
	OperatorChoice       hostreqspec.OperatorChoice `json:"operator_choice"`
	Config               map[string]any             `json:"config,omitempty"`
	SelectedProvider     string                     `json:"selected_provider,omitempty"`
	ObservedProvider     string                     `json:"observed_provider,omitempty"`
	ObservedMode         string                     `json:"observed_mode,omitempty"`
	ObservedLive         bool                       `json:"observed_live,omitempty"`
	ObservedActive       bool                       `json:"observed_active,omitempty"`
	CredentialStoreState string                     `json:"credential_store_state,omitempty"`
	InstallSupported     bool                       `json:"install_supported"`
	PackageName          string                     `json:"package_name,omitempty"`
	SupportClass         SupportClass               `json:"support_class"`
	ExecutionState       ExecutionState             `json:"execution_state"`
	BlockingReason       BlockingReason             `json:"blocking_reason,omitempty"`
	Manual               bool                       `json:"manual"`
	Reasons              []string                   `json:"reasons,omitempty"`
	Notes                []string                   `json:"notes,omitempty"`
	Provenance           []hostreqspec.Provenance   `json:"provenance,omitempty"`
}

func (h Host) ValidateSetup() error {
	if h.SupportsSetup {
		return nil
	}
	return h.unsupportedError("setup")
}

func (h Host) ValidateDevelop() error {
	if h.SupportsDevelop {
		return nil
	}
	return h.unsupportedError("develop")
}

func (h Host) unsupportedError(command string) error {
	os := strings.TrimSpace(h.OS)
	if os == "" {
		os = "this platform"
	}
	if len(h.Notes) == 0 {
		return fmt.Errorf("%w: vrooli %s is not supported on %s", ErrUnsupportedPlatform, command, os)
	}
	return fmt.Errorf("%w: vrooli %s is not supported on %s (%s)", ErrUnsupportedPlatform, command, os, strings.Join(h.Notes, "; "))
}

type (
	ToolStatus      = ItemStatus
	SafeguardStatus = ItemStatus
)

type Report struct {
	Environment     string            `json:"environment"`
	Host            Host              `json:"host"`
	Tools           []ToolStatus      `json:"tools"`
	Safeguards      []SafeguardStatus `json:"safeguards,omitempty"`
	MissingRequired []string          `json:"missing_required,omitempty"`
	MissingOptional []string          `json:"missing_optional,omitempty"`
}

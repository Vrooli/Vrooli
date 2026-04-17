package hostreqkit

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqspec"
)

var ErrUnsupportedPlatform = errors.New("unsupported platform")

type SupportClass string
type ExecutionState string

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
	Stdout      io.Writer
	Stderr      io.Writer
}

type ItemStatus struct {
	Name             string                       `json:"name"`
	Kind             hostreqspec.Kind             `json:"kind"`
	Command          string                       `json:"command,omitempty"`
	Version          string                       `json:"version,omitempty"`
	Installed        bool                         `json:"installed"`
	Applied          bool                         `json:"applied,omitempty"`
	Required         bool                         `json:"required"`
	InstallSupported bool                         `json:"install_supported"`
	PackageName      string                       `json:"package_name,omitempty"`
	SupportClass     SupportClass                 `json:"support_class"`
	ExecutionState   ExecutionState               `json:"execution_state"`
	Manual           bool                         `json:"manual"`
	Reasons          []string                     `json:"reasons,omitempty"`
	Notes            []string                     `json:"notes,omitempty"`
	Provenance       []hostreqspec.Provenance     `json:"provenance,omitempty"`
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

type ToolStatus = ItemStatus
type SafeguardStatus = ItemStatus

type Report struct {
	Environment     string            `json:"environment"`
	Host            Host              `json:"host"`
	Tools           []ToolStatus      `json:"tools"`
	Safeguards      []SafeguardStatus `json:"safeguards,omitempty"`
	MissingRequired []string          `json:"missing_required,omitempty"`
	MissingOptional []string          `json:"missing_optional,omitempty"`
}

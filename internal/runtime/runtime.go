package runtime

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreq"
)

var ErrUnsupportedPlatform = errors.New("unsupported platform")

type EnsureOptions struct {
	Environment string
	SudoMode    string
	DryRun      bool
	AutoInstall bool
	Stdout      io.Writer
	Stderr      io.Writer
}

type Host struct {
	OS              string   `json:"os"`
	PackageManager  string   `json:"package_manager,omitempty"`
	SupportsSetup   bool     `json:"supports_setup"`
	SupportsDevelop bool     `json:"supports_develop"`
	SupportsSysctl  bool     `json:"supports_sysctl"`
	SupportsSystemd bool     `json:"supports_systemd"`
	Notes           []string `json:"notes,omitempty"`
}

func Current() Host {
	return currentHost()
}

func Inspect(environment string) (Report, error) {
	return Report{}, errors.New("runtime inspection requires explicit host requirements; use InspectRequirements")
}

func InspectRequirements(environment string, resolution hostreq.Resolution) (Report, error) {
	env := hostreq.NormalizeEnvironment(environment)
	return inspectResolution(Current(), env, resolution)
}

func Ensure(opts EnsureOptions) (Report, error) {
	return Report{}, errors.New("runtime ensure requires explicit host requirements; use EnsureRequirements")
}

func EnsureRequirements(opts EnsureOptions, resolution hostreq.Resolution) (Report, error) {
	opts.Environment = hostreq.NormalizeEnvironment(opts.Environment)
	return ensureResolution(opts, resolution)
}

func ensureResolution(opts EnsureOptions, resolution hostreq.Resolution) (Report, error) {
	report, err := inspectResolution(Current(), opts.Environment, resolution)
	if err != nil {
		return Report{}, err
	}
	if !opts.AutoInstall {
		return report, missingRequiredError(report, opts)
	}

	for index, status := range report.Tools {
		if requirementSatisfied(status) || !status.Required {
			continue
		}
		updated, applyErr := applyRequirement(report.Host, status, opts)
		if applyErr != nil {
			return Report{}, applyErr
		}
		report.Tools[index] = updated
	}
	for index, status := range report.Safeguards {
		if requirementSatisfied(status) || !status.Required {
			continue
		}
		updated, applyErr := applyRequirement(report.Host, status, opts)
		if applyErr != nil {
			return Report{}, applyErr
		}
		report.Safeguards[index] = updated
	}

	report = summarizeReport(report)
	return report, missingRequiredError(report, opts)
}

func inspectResolution(host Host, environment string, resolution hostreq.Resolution) (Report, error) {
	report := Report{
		Environment: environment,
		Host:        host,
		Tools:       make([]ToolStatus, 0, len(resolution.Tools)),
		Safeguards:  make([]SafeguardStatus, 0, len(resolution.Safeguards)),
	}
	for _, requirement := range resolution.Tools {
		report.Tools = append(report.Tools, inspectRequirement(host, requirement))
	}
	for _, requirement := range resolution.Safeguards {
		report.Safeguards = append(report.Safeguards, inspectRequirement(host, requirement))
	}
	return summarizeReport(report), nil
}

func inspectRequirement(host Host, requirement hostreq.ResolvedRequirement) ItemStatus {
	handler := lookupHandler(requirement.Kind, requirement.Name)
	if handler == nil {
		return unsupportedRequirementStatus(requirement, "no native runtime handler registered")
	}
	return handler.Inspect(host, requirement)
}

func applyRequirement(host Host, status ItemStatus, opts EnsureOptions) (ItemStatus, error) {
	handler := lookupHandler(status.Kind, status.Name)
	if handler == nil {
		status.Notes = append(status.Notes, "no native runtime handler registered")
		status.SupportClass = SupportUnsupported
		return status, nil
	}
	return handler.Apply(host, status, opts)
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
	if len(h.Notes) == 0 {
		return fmt.Errorf("%w: vrooli %s is not supported on %s", ErrUnsupportedPlatform, command, defaultOS(h.OS))
	}
	return fmt.Errorf("%w: vrooli %s is not supported on %s (%s)", ErrUnsupportedPlatform, command, defaultOS(h.OS), strings.Join(h.Notes, "; "))
}

func defaultOS(value string) string {
	if strings.TrimSpace(value) == "" {
		return "this platform"
	}
	return value
}

func summarizeReport(report Report) Report {
	report.MissingRequired = report.MissingRequired[:0]
	report.MissingOptional = report.MissingOptional[:0]
	for _, tool := range report.Tools {
		appendMissingRequirement(&report, tool)
	}
	for _, safeguard := range report.Safeguards {
		appendMissingRequirement(&report, safeguard)
	}
	return report
}

func appendMissingRequirement(report *Report, status ItemStatus) {
	if requirementSatisfied(status) {
		return
	}
	if status.Required {
		report.MissingRequired = append(report.MissingRequired, status.Name)
		return
	}
	report.MissingOptional = append(report.MissingOptional, status.Name)
}

func requirementSatisfied(status ItemStatus) bool {
	switch status.Kind {
	case hostreq.KindSafeguard:
		return status.Applied
	default:
		return status.Installed
	}
}

func missingRequiredError(report Report, opts EnsureOptions) error {
	if opts.DryRun {
		return nil
	}
	if len(report.MissingRequired) == 0 {
		return nil
	}
	return fmt.Errorf("missing required host requirements for %s: %s", hostreq.NormalizeEnvironment(report.Environment), strings.Join(report.MissingRequired, ", "))
}

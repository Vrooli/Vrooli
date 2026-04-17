package runtime

import (
	"errors"
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

var ErrUnsupportedPlatform = hostreqkit.ErrUnsupportedPlatform

type Host = hostreqkit.Host

type EnsureOptions = hostreqkit.EnsureOptions

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
	h := lookupHandler(requirement.Kind, requirement.Name)
	if h == nil {
		return hostreqkit.UnsupportedRequirementStatus(requirement, "no native runtime handler registered")
	}
	return h.Inspect(host, requirement)
}

func applyRequirement(host Host, status ItemStatus, opts EnsureOptions) (ItemStatus, error) {
	h := lookupHandler(status.Kind, status.Name)
	if h == nil {
		status.Notes = append(status.Notes, "no native runtime handler registered")
		status.SupportClass = SupportUnsupported
		return status, nil
	}
	return h.Apply(host, status, opts)
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

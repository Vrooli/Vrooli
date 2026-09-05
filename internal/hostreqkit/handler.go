package hostreqkit

import "github.com/vrooli/vrooli/internal/hostreqspec"

type Handler interface {
	Name() string
	Kind() hostreqspec.Kind
	Inspect(host Host, requirement hostreqspec.ResolvedRequirement) ItemStatus
	Apply(host Host, status ItemStatus, opts EnsureOptions) (ItemStatus, error)
}

// InstallerHandler adapts a shared installer into the host requirement
// handler contract. It keeps package-specific handlers focused on manifest
// values and artifact provenance callbacks.
type InstallerHandler struct {
	Manifest    ToolManifest
	KindValue   hostreqspec.Kind
	InspectFunc func(Host, hostreqspec.ResolvedRequirement) ItemStatus
	ApplyFunc   func(ItemStatus, EnsureOptions) (ItemStatus, error)
}

func (h InstallerHandler) Name() string           { return h.Manifest.Name }
func (h InstallerHandler) Kind() hostreqspec.Kind { return h.KindValue }
func (h InstallerHandler) Inspect(host Host, requirement hostreqspec.ResolvedRequirement) ItemStatus {
	return h.InspectFunc(host, requirement)
}

func (h InstallerHandler) Apply(_ Host, status ItemStatus, opts EnsureOptions) (ItemStatus, error) {
	return h.ApplyFunc(status, opts)
}

// ApplyStatus handles the status-only branches shared by host requirement
// handlers before they perform an installation or mutation.
func ApplyStatus(status ItemStatus) (ItemStatus, bool) {
	if status.Installed {
		status.ExecutionState = ExecutionAlreadyPresent
		return status, true
	}
	switch status.SupportClass {
	case SupportManualOnly:
		status.ExecutionState = ExecutionManualActionRequired
		return status, true
	case SupportUnsupported:
		status.ExecutionState = ExecutionUnsupported
		return status, true
	case SupportNotApplicable:
		status.ExecutionState = ExecutionNotApplicable
		return status, true
	default:
		return status, false
	}
}

// ApplyTool runs an installer's common status and error handling for tools
// whose installer does not need host details after inspection.
func ApplyTool(status ItemStatus, opts EnsureOptions, apply func(ItemStatus, EnsureOptions) (ItemStatus, error)) (ItemStatus, error) {
	if status, done := ApplyStatus(status); done {
		return status, nil
	}
	result, err := apply(status, opts)
	if err != nil {
		result.ExecutionState = ExecutionFailed
		result.Notes = append(result.Notes, err.Error())
	}
	return result, nil
}

// ApplyHostTool is the host-aware counterpart of ApplyTool.
func ApplyHostTool(host Host, status ItemStatus, opts EnsureOptions, apply func(Host, ItemStatus, EnsureOptions) (ItemStatus, error)) (ItemStatus, error) {
	if status, done := ApplyStatus(status); done {
		return status, nil
	}
	result, err := apply(host, status, opts)
	if err != nil {
		result.ExecutionState = ExecutionFailed
		result.Notes = append(result.Notes, err.Error())
	}
	return result, nil
}

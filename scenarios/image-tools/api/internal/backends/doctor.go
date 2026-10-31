package backends

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"image-tools/internal/models"
)

// Availability is the software-readiness verdict for one backend provider.
// Hardware fit is intentionally separate and remains owned by capabilities.Probe.
type Availability struct {
	Available bool
	Detail    string
	Provision string
}

// availabilityReporter is an optional Provider capability for providers that
// can return actionable readiness detail beyond a boolean.
type availabilityReporter interface {
	Availability(ctx context.Context) Availability
}

// BackendStatus is one row in the backend doctor report.
type BackendStatus struct {
	Name       string
	Operations []string
	Available  bool
	Standalone bool
	Cloud      bool
	GPUCapable bool
	Detail     string
	Provision  string
}

// DoctorReport is the complete backend software-availability report.
type DoctorReport struct {
	OK       bool
	Backends []BackendStatus
}

func providerAvailability(ctx context.Context, p Provider) Availability {
	if r, ok := p.(availabilityReporter); ok {
		a := r.Availability(ctx)
		if a.Detail == "" {
			a.Detail = availabilityDetail(a.Available)
		}
		if a.Provision == "" {
			a.Provision = defaultProvision
		}
		return a
	}
	available := p.Available(ctx)
	return Availability{
		Available: available,
		Detail:    availabilityDetail(available),
		Provision: defaultProvision,
	}
}

func availabilityDetail(available bool) string {
	if available {
		return "provider reported ready"
	}
	return "provider reported unavailable"
}

const defaultProvision = "see docs/reference/backends.md"

// Doctor checks registered backend software availability. Missing BYOK/cloud
// providers do not fail the report because they are opt-in fallback tiers; a
// missing local provider is a host provisioning gap.
func (r *Registry) Doctor(ctx context.Context) DoctorReport {
	return r.DoctorForModels(ctx, nil)
}

// DoctorForModels checks registered backend software availability and overlays
// the enabled model catalog's declared backend families. A declared-but-
// unregistered backend is a Phase-2 readiness gap: the catalog advertises a
// native backend family that the runtime cannot currently probe or select.
func (r *Registry) DoctorForModels(ctx context.Context, catalog []models.Model) DoctorReport {
	type aggregate struct {
		p   Provider
		ops map[string]struct{}
	}
	byProvider := map[string]*aggregate{}
	for op, providers := range r.byOp {
		for _, p := range providers {
			a := byProvider[providerDoctorKey(p)]
			if a == nil {
				a = &aggregate{p: p, ops: map[string]struct{}{}}
				byProvider[providerDoctorKey(p)] = a
			}
			a.ops[op] = struct{}{}
		}
	}

	out := DoctorReport{OK: true, Backends: make([]BackendStatus, 0, len(byProvider))}
	for _, a := range byProvider {
		ops := make([]string, 0, len(a.ops))
		for op := range a.ops {
			ops = append(ops, op)
		}
		sort.Strings(ops)
		avail := providerAvailability(ctx, a.p)
		status := BackendStatus{
			Name:       a.p.Name(),
			Operations: ops,
			Available:  avail.Available,
			Standalone: a.p.Standalone(),
			Cloud:      a.p.IsCloud(),
			GPUCapable: providerGPUCapable(a.p),
			Detail:     avail.Detail,
			Provision:  avail.Provision,
		}
		if !status.Available && !status.Cloud {
			out.OK = false
		}
		out.Backends = append(out.Backends, status)
	}
	registeredOps := make(map[string]map[string]struct{}, len(byProvider))
	for _, aggregate := range byProvider {
		ops := registeredOps[aggregate.p.Name()]
		if ops == nil {
			ops = make(map[string]struct{}, len(aggregate.ops))
			registeredOps[aggregate.p.Name()] = ops
		}
		for op := range aggregate.ops {
			ops[op] = struct{}{}
		}
	}
	for _, status := range undeclaredBackendStatuses(catalog, registeredOps) {
		out.OK = false
		out.Backends = append(out.Backends, status)
	}
	sort.Slice(out.Backends, func(i, j int) bool {
		if out.Backends[i].Name != out.Backends[j].Name {
			return out.Backends[i].Name < out.Backends[j].Name
		}
		return strings.Join(out.Backends[i].Operations, ",") < strings.Join(out.Backends[j].Operations, ",")
	})
	return out
}

func providerDoctorKey(p Provider) string {
	ops := p.Operations()
	sort.Strings(ops)
	return p.Name() + "\x00" + strings.Join(ops, "\x00")
}

func undeclaredBackendStatuses(catalog []models.Model, registered map[string]map[string]struct{}) []BackendStatus {
	if len(catalog) == 0 {
		return nil
	}
	opsByBackend := map[string]map[string]struct{}{}
	for _, m := range catalog {
		if !m.Enabled || strings.TrimSpace(m.Backend) == "" {
			continue
		}
		ops := opsByBackend[m.Backend]
		registeredOps := registered[m.Backend]
		for _, op := range m.Operations {
			if _, ok := registeredOps[op]; ok {
				continue
			}
			if ops == nil {
				ops = map[string]struct{}{}
				opsByBackend[m.Backend] = ops
			}
			ops[op] = struct{}{}
		}
	}
	out := make([]BackendStatus, 0, len(opsByBackend))
	for name, opsSet := range opsByBackend {
		ops := make([]string, 0, len(opsSet))
		for op := range opsSet {
			ops = append(ops, op)
		}
		sort.Strings(ops)
		out = append(out, BackendStatus{
			Name:       name,
			Operations: ops,
			Available:  false,
			Standalone: true,
			Cloud:      false,
			GPUCapable: declaredBackendGPUCapable(name),
			Detail:     "enabled model catalog declares this backend, but no runtime provider is registered to probe or execute it",
			Provision:  declaredBackendProvision(name),
		})
	}
	return out
}

func declaredBackendGPUCapable(name string) bool {
	switch name {
	case models.BackendBuiltin, models.BackendComputed, models.BackendLibraryGo, "library-cgo", "onnxruntime", "rembg":
		return false
	default:
		return true
	}
}

func declaredBackendProvision(name string) string {
	switch name {
	case models.BackendComputed:
		return "wire an in-process computed provider for this operation; no host provisioning required"
	case models.BackendLibraryGo:
		return "wire an in-process Go library provider for this operation; no host provisioning required"
	case "library-cgo":
		return "provision the host C/C++ library and data through Scenario Dependency Analyzer, then register a runtime provider"
	case "python-sidecar":
		return "embed/register the Python sidecar module and manage its runtime dependencies through Scenario Dependency Analyzer"
	case "llama.cpp":
		return "install/register llama.cpp tooling through Scenario Dependency Analyzer; do not use raw package managers"
	default:
		return defaultProvision
	}
}

func unavailableProviderDetails(ctx context.Context, providers []Provider) string {
	if len(providers) == 0 {
		return ""
	}
	lines := make([]string, 0, len(providers))
	for _, p := range providers {
		a := providerAvailability(ctx, p)
		lines = append(lines, fmt.Sprintf("%s: %s; provision: %s", p.Name(), a.Detail, a.Provision))
	}
	sort.Strings(lines)
	return strings.Join(lines, "; ")
}

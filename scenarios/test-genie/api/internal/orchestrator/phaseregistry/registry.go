// Package phaseregistry builds Test Genie's effective phase registry from
// provider-owned descriptors. It validates orchestration-level concerns that a
// pure descriptor loader cannot know, then projects descriptors into phase
// specs with Test Genie-owned runner bindings.
package phaseregistry

import (
	"fmt"
	"sort"
	"strings"

	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/orchestrator/providerdescriptor"

	"github.com/vrooli/vrooli/packages/proto/architecture/findingid"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

const SourceValidationProvider = "validation-provider"

type Diagnostic struct {
	Path    string `json:"path,omitempty"`
	Phase   string `json:"phase,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type RequiredPhase struct {
	Phase            string
	ProviderScenario string
}

type Options struct {
	Bindings       map[string]RunnerBinding
	RequiredPhases []RequiredPhase
}

type RunnerBinding func(providerdescriptor.Descriptor, architecturev1.FindingSource) (phases.Spec, error)

type Entry struct {
	Descriptor providerdescriptor.Descriptor
	Spec       phases.Spec
}

type Registry struct {
	entries map[phases.Name]Entry
	order   []phases.Name
}

type Result struct {
	Registry    *Registry
	Diagnostics []Diagnostic
}

func Build(descriptors []providerdescriptor.Descriptor, opts Options) Result {
	bindings := opts.Bindings
	if bindings == nil {
		bindings = DefaultBindings()
	}

	ordered := append([]providerdescriptor.Descriptor(nil), descriptors...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].OrderHint != ordered[j].OrderHint {
			return ordered[i].OrderHint < ordered[j].OrderHint
		}
		return ordered[i].Phase < ordered[j].Phase
	})

	registry := &Registry{
		entries: make(map[phases.Name]Entry, len(ordered)),
		order:   make([]phases.Name, 0, len(ordered)),
	}
	var diagnostics []Diagnostic
	seen := map[string]providerdescriptor.Descriptor{}
	for _, descriptor := range ordered {
		phaseName, ok := phases.NormalizeName(descriptor.Phase)
		if !ok {
			diagnostics = append(diagnostics, diagnostic(descriptor, "invalid_phase", "phase is not a normalized Test Genie phase name"))
			continue
		}
		if first, exists := seen[phaseName.String()]; exists {
			diagnostics = append(diagnostics, diagnostic(descriptor, "duplicate_phase", fmt.Sprintf("phase %q already declared by %s", phaseName, first.Path)))
			continue
		}
		seen[phaseName.String()] = descriptor

		bind, ok := bindings[descriptor.Source]
		if !ok {
			diagnostics = append(diagnostics, diagnostic(descriptor, "unsupported_source", fmt.Sprintf("source %q has no Test Genie runner binding", descriptor.Source)))
			continue
		}
		findingSource, ok := parseFindingSource(descriptor.FindingSource)
		if !ok {
			diagnostics = append(diagnostics, diagnostic(descriptor, "invalid_finding_source", fmt.Sprintf("findingSource %q is not a known architecture finding source token", descriptor.FindingSource)))
			continue
		}
		spec, err := bind(descriptor, findingSource)
		if err != nil {
			diagnostics = append(diagnostics, diagnostic(descriptor, "binding_failed", err.Error()))
			continue
		}
		registry.entries[phaseName] = Entry{Descriptor: descriptor, Spec: spec}
		registry.order = append(registry.order, phaseName)
	}

	diagnostics = append(diagnostics, requiredPhaseDiagnostics(opts.RequiredPhases, seen)...)
	if len(diagnostics) > 0 {
		return Result{Diagnostics: diagnostics}
	}
	return Result{Registry: registry}
}

func DefaultBindings() map[string]RunnerBinding {
	return map[string]RunnerBinding{
		SourceValidationProvider: validationProviderBinding,
	}
}

func validationProviderBinding(descriptor providerdescriptor.Descriptor, findingSource architecturev1.FindingSource) (phases.Spec, error) {
	return phases.ValidationProviderSpecFromDescriptor(descriptor, findingSource)
}

func (r *Registry) All() []Entry {
	if r == nil {
		return nil
	}
	out := make([]Entry, 0, len(r.order))
	for _, name := range r.order {
		out = append(out, r.entries[name])
	}
	return out
}

func (r *Registry) Specs() []phases.Spec {
	entries := r.All()
	out := make([]phases.Spec, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.Spec)
	}
	return out
}

func (r *Registry) Lookup(raw string) (Entry, bool) {
	if r == nil {
		return Entry{}, false
	}
	name, ok := phases.NormalizeName(raw)
	if !ok {
		return Entry{}, false
	}
	entry, ok := r.entries[name]
	return entry, ok
}

func requiredPhaseDiagnostics(required []RequiredPhase, seen map[string]providerdescriptor.Descriptor) []Diagnostic {
	var diagnostics []Diagnostic
	for _, requirement := range required {
		phase := strings.TrimSpace(requirement.Phase)
		if phase == "" {
			continue
		}
		descriptor, ok := seen[phases.NormalizeKey(phase)]
		if !ok {
			diagnostics = append(diagnostics, Diagnostic{
				Phase:   phase,
				Code:    "missing_required_descriptor",
				Message: fmt.Sprintf("required provider-backed phase %q has no descriptor", phase),
			})
			continue
		}
		if provider := strings.TrimSpace(requirement.ProviderScenario); provider != "" && descriptor.Scenario != provider {
			diagnostics = append(diagnostics, diagnostic(descriptor, "provider_mismatch", fmt.Sprintf("phase %q must be declared by provider %q", phase, provider)))
		}
	}
	return diagnostics
}

func diagnostic(descriptor providerdescriptor.Descriptor, code, message string) Diagnostic {
	return Diagnostic{
		Path:    descriptor.Path,
		Phase:   descriptor.Phase,
		Code:    code,
		Message: message,
	}
}

func parseFindingSource(token string) (architecturev1.FindingSource, bool) {
	token = strings.ToLower(strings.TrimSpace(token))
	if token == "" {
		return architecturev1.FindingSource_FINDING_SOURCE_UNSPECIFIED, true
	}
	for _, source := range []architecturev1.FindingSource{
		architecturev1.FindingSource_FINDING_SOURCE_UNSPECIFIED,
		architecturev1.FindingSource_FINDING_SOURCE_STRUCTURE,
		architecturev1.FindingSource_FINDING_SOURCE_CLI,
		architecturev1.FindingSource_FINDING_SOURCE_UI,
		architecturev1.FindingSource_FINDING_SOURCE_DOCS,
		architecturev1.FindingSource_FINDING_SOURCE_STANDARDS,
		architecturev1.FindingSource_FINDING_SOURCE_ARCHITECTURE,
		architecturev1.FindingSource_FINDING_SOURCE_TIDINESS,
		architecturev1.FindingSource_FINDING_SOURCE_COVERAGE,
		architecturev1.FindingSource_FINDING_SOURCE_SECURITY,
		architecturev1.FindingSource_FINDING_SOURCE_MEASURES,
		architecturev1.FindingSource_FINDING_SOURCE_BUSINESS,
		architecturev1.FindingSource_FINDING_SOURCE_PROTO,
		architecturev1.FindingSource_FINDING_SOURCE_DEPENDENCY,
		architecturev1.FindingSource_FINDING_SOURCE_STORAGE,
		architecturev1.FindingSource_FINDING_SOURCE_BRANDING,
		architecturev1.FindingSource_FINDING_SOURCE_WORKFLOW,
	} {
		if findingid.SourceToken(source) == token {
			return source, true
		}
	}
	return architecturev1.FindingSource_FINDING_SOURCE_UNSPECIFIED, false
}

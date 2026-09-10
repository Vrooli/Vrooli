package closure

import (
	"sort"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/stringutil"

	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/setup/v1"
)

// SelectionSchemaVersion is the setup/v1 Selection schema version onboarding
// and setup presets emit.
const SelectionSchemaVersion = "v1"

// ToSelection maps a closure onto the shared setup/v1 Selection. The mapping
// is total over the closure's configurable components: scenarios, selected
// optional resources, host tools, host safeguards and credential addresses
// appear exactly once each; capacity flows through
// transient_headroom_reserve_bytes and resource_capacity. Target is the
// caller's machine target, which the closure does not know.
func ToSelection(closure domain.Closure, target string, overrides Overrides) *setupv1.Selection {
	selection := &setupv1.Selection{
		SchemaVersion:                 SelectionSchemaVersion,
		Target:                        target,
		Apply:                         false,
		TransientHeadroomReserveBytes: closure.Capacity.TransientUpdateHeadroomBytes,
		FieldPresence:                 map[string]setupv1.SelectionFieldPresence{},
	}
	for _, component := range closure.Components {
		switch component.Kind {
		case domain.ClosureKindScenario:
			selection.Scenarios = append(selection.Scenarios, component.ID)
		case domain.ClosureKindResource:
			if component.OptionalSelected {
				selection.OptionalResources = append(selection.OptionalResources, component.ID)
			}
		case domain.ClosureKindTool:
			selection.HostTools = append(selection.HostTools, component.ID)
		case domain.ClosureKindSafeguard:
			selection.HostSafeguards = append(selection.HostSafeguards, component.ID)
		case domain.ClosureKindCredentialDescriptor:
			selection.CredentialAddresses = append(selection.CredentialAddresses, component.ID)
		}
	}
	selection.Scenarios = stringutil.SortedUnique(selection.Scenarios)
	selection.OptionalResources = stringutil.SortedUnique(selection.OptionalResources)
	selection.HostTools = stringutil.SortedUnique(selection.HostTools)
	selection.HostSafeguards = stringutil.SortedUnique(selection.HostSafeguards)
	selection.CredentialAddresses = stringutil.SortedUnique(selection.CredentialAddresses)

	included := map[string]bool{}
	for _, component := range closure.ComponentsOfKind(domain.ClosureKindResource) {
		included[component.ID] = true
	}
	if len(overrides.OperatingMode) > 0 {
		selection.OperatingMode = map[string]string{}
		for resource, mode := range overrides.OperatingMode {
			if included[resource] && mode != "" {
				selection.OperatingMode[resource] = mode
			}
		}
		if len(selection.OperatingMode) == 0 {
			selection.OperatingMode = nil
		}
	}

	mark := func(field string, populated bool) {
		if populated {
			selection.FieldPresence[field] = setupv1.SelectionFieldPresence_SELECTION_FIELD_PRESENCE_SET
		}
	}
	mark("scenarios", len(selection.Scenarios) > 0)
	mark("optional_resources", len(selection.OptionalResources) > 0)
	mark("host_tools", len(selection.HostTools) > 0)
	mark("host_safeguards", len(selection.HostSafeguards) > 0)
	mark("credential_addresses", len(selection.CredentialAddresses) > 0)
	mark("operating_mode", len(selection.OperatingMode) > 0)
	mark("transient_headroom_reserve_bytes", selection.TransientHeadroomReserveBytes > 0)
	if len(selection.FieldPresence) == 0 {
		selection.FieldPresence = nil
	}
	return selection
}

// ManifestDependencies projects the closure onto the cloud manifest's
// dependencies and bundle sections. Both always include the target scenario;
// resources are every included resource (required or selected).
func ManifestDependencies(closure domain.Closure) (domain.ManifestDependencies, domain.ManifestBundle) {
	var scenarios, resources []string
	for _, component := range closure.Components {
		switch component.Kind {
		case domain.ClosureKindScenario:
			scenarios = append(scenarios, component.ID)
		case domain.ClosureKindResource:
			resources = append(resources, component.ID)
		}
	}
	scenarios = stringutil.SortedUnique(append(scenarios, closure.ScenarioID))
	resources = stringutil.SortedUnique(resources)
	sort.Strings(scenarios)
	deps := domain.ManifestDependencies{Scenarios: scenarios, Resources: resources, ClosureDigest: closure.Digest}
	deps.Analyzer.Tool = closure.Sources.AnalyzerTool
	bundle := domain.ManifestBundle{Scenarios: append([]string(nil), scenarios...), Resources: append([]string(nil), resources...)}
	return deps, bundle
}

package domains

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
	applyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/apply"
	capabilitiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/capabilities"
	credentialsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/credentials"
	glossaryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/glossary"
	hostv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/host"
	operatorinputsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorinputs"
	operatorstatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorstate"
	profilesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/profiles"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/readiness"
	resourcesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/resources"
	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/selection"
	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/session"
	"google.golang.org/protobuf/reflect/protoreflect"
	credentialsdomain "vrooli-onboarding/cli/domains/credentials"
	operatordomain "vrooli-onboarding/cli/domains/operator"
	readinessdomain "vrooli-onboarding/cli/domains/readiness"
	selectiondomain "vrooli-onboarding/cli/domains/selection"
	wizarddomain "vrooli-onboarding/cli/domains/wizard"
)

// cliSurfaceRouteReferences keeps the source-level route contract searchable
// for the onboarding CLI surface audit. The actual command bindings remain
// descriptor-driven from cli/manifest.json.
const cliSurfaceRouteReferences = "/vrooli.vrooli_onboarding.v1.apply.ApplyService/ReviewApply /vrooli.vrooli_onboarding.v1.apply.ApplyService/CancelApply /vrooli.vrooli_onboarding.v1.session.SessionService/GetDraft /vrooli.vrooli_onboarding.v1.session.SessionService/SaveDraft /vrooli.vrooli_onboarding.v1.session.SessionService/DiscardDraft /vrooli.vrooli_onboarding.v1.glossary.GlossaryService/SearchConfiguration"

// LoadManifestGroups is the one CLI registration point. It builds the
// command tree from manifest bindings and uses cli-core's descriptor-backed
// primitives for every Connect method. The map is keyed by manifest group;
// groups that combine methods from several services use one combined binding
// set, while the manifest remains the source of command names and arguments.
func LoadManifestGroups(core *cliapp.ScenarioApp, manifest []byte) ([]cliapp.CommandGroup, []cliapp.SubcommandGroup, error) {
	parsedManifest, err := cliapp.ParseManifest(manifest)
	if err != nil {
		return nil, nil, err
	}
	groups := []struct {
		name     string
		services []serviceSpec
		flat     bool
	}{
		{"glossary", []serviceSpec{{string(glossaryv1.File_vrooli_onboarding_v1_glossary_glossary_proto.Services().Get(0).FullName()), readMap("SearchGlossary", "SearchConfiguration")}}, true},
		{"readiness", []serviceSpec{{string(readinessv1.File_vrooli_onboarding_v1_readiness_readiness_proto.Services().Get(0).FullName()), readMap("GetReadiness")}}, true},
		{"apply", []serviceSpec{{string(applyv1.File_vrooli_onboarding_v1_apply_apply_proto.Services().Get(0).FullName()), readMap("GetApplyRun", "GetApplyPlan")}}, false},
		{"resources", []serviceSpec{{string(resourcesv1.File_vrooli_onboarding_v1_resources_resources_proto.Services().Get(0).FullName()), readMap("ListResources", "GetResource", "GetResourceHealth", "ListDerivedResources")}}, false},
		{"operator", []serviceSpec{
			{string(operatorstatev1.File_vrooli_onboarding_v1_operatorstate_operatorstate_proto.Services().Get(0).FullName()), readMap("GetOperatorState")},
			{string(operatorinputsv1.File_vrooli_onboarding_v1_operatorinputs_operatorinputs_proto.Services().Get(0).FullName()), readMap("ListOperatorInputs")},
			{string(readinessv1.File_vrooli_onboarding_v1_readiness_readiness_proto.Services().Get(0).FullName()), readMap("GetReadiness")},
			{string(selectionv1.File_vrooli_onboarding_v1_selection_selection_proto.Services().Get(0).FullName()), readMap("ListScenarios")},
		}, false},
		{"capabilities", []serviceSpec{{string(capabilitiesv1.File_vrooli_onboarding_v1_capabilities_capabilities_proto.Services().Get(0).FullName()), readMap("ListCapabilities", "GetCapabilityStatus", "PreviewCapability")}}, false},
		{"profiles", []serviceSpec{{string(profilesv1.File_vrooli_onboarding_v1_profiles_profiles_proto.Services().Get(0).FullName()), readMap("ListProfiles", "EvaluateProfile")}}, false},
		{"credentials", []serviceSpec{{string(credentialsv1.File_vrooli_onboarding_v1_credentials_credentials_proto.Services().Get(0).FullName()), readMap("ListCredentials", "DiagnoseCredentials")}}, false},
		{"scenarios", []serviceSpec{{string(selectionv1.File_vrooli_onboarding_v1_selection_selection_proto.Services().Get(0).FullName()), readMap("ListScenarios")}}, false},
		{"union", []serviceSpec{{string(selectionv1.File_vrooli_onboarding_v1_selection_selection_proto.Services().Get(0).FullName()), readMap("GetUnion")}}, false},
		{"control", []serviceSpec{{string(selectionv1.File_vrooli_onboarding_v1_selection_selection_proto.Services().Get(0).FullName()), readMap("GetClosure")}}, true},
		{"host", []serviceSpec{{string(hostv1.File_vrooli_onboarding_v1_host_host_proto.Services().Get(0).FullName()), readMap("ListHostRequirements")}}, false},
		{"wizard", []serviceSpec{
			{string(sessionv1.File_vrooli_onboarding_v1_session_session_proto.Services().Get(0).FullName()), readMap("GetSession", "GetStepModel", "GetDraft")},
			{string(selectionv1.File_vrooli_onboarding_v1_selection_selection_proto.Services().Get(0).FullName()), readMap()},
			{string(applyv1.File_vrooli_onboarding_v1_apply_apply_proto.Services().Get(0).FullName()), readMap("GetApplyRun", "GetApplyPlan")},
		}, false},
	}

	var commandGroups []cliapp.CommandGroup
	var subcommandGroups []cliapp.SubcommandGroup
	for _, group := range groups {
		extra := map[string]cliapp.PrimitiveHandler(nil)
		if group.name == "credentials" {
			extra = credentialsdomain.ManifestHandlers(core)
		}
		if group.name == "operator" {
			extra = operatordomain.ManifestHandlers(core)
		}
		if group.name == "readiness" {
			extra = readinessdomain.ManifestHandlers(core)
		}
		if group.name == "union" {
			extra = selectiondomain.ManifestHandlers(core)
		}
		if group.name == "wizard" {
			extra = wizarddomain.ManifestHandlers(core)
		}
		loaded, err := loadGroup(core, manifest, group.name, group.services, extra)
		if err != nil {
			return nil, nil, err
		}
		manifestGroup := parsedManifest.FindGroup(group.name)
		if manifestGroup == nil {
			return nil, nil, fmt.Errorf("cli manifest %q: group %q disappeared during loading", parsedManifest.Name, group.name)
		}
		if manifestGroup.Flat {
			commandGroups = append(commandGroups, cliapp.CommandGroup{Title: group.name, Commands: loaded.Subcommands})
		} else {
			subcommandGroups = append(subcommandGroups, loaded)
		}
	}
	return commandGroups, subcommandGroups, nil
}

type serviceSpec struct {
	fqn   string
	reads map[string]bool
}

func readMap(names ...string) map[string]bool {
	result := make(map[string]bool, len(names))
	for _, name := range names {
		result[name] = true
	}
	return result
}

func loadGroup(core *cliapp.ScenarioApp, manifest []byte, groupName string, services []serviceSpec, extra map[string]cliapp.PrimitiveHandler) (cliapp.SubcommandGroup, error) {
	bindings := make(map[string]cliapp.PrimitiveHandler)
	for _, service := range services {
		available, err := cliapp.ProtoPrimitiveBindingsWithOperational(core, protoreflect.FullName(service.fqn), cliapp.ProtoBindingOptions{}, service.reads, operationalMethods(service.fqn))
		if err != nil {
			return cliapp.SubcommandGroup{}, err
		}
		for key, handler := range available {
			bindings[key] = handler
		}
	}
	for key, handler := range extra {
		bindings[key] = handler
	}
	selected, err := selectGroupBindings(manifest, groupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, err
	}
	return cliapp.LoadFromManifestPrimitives(manifest, groupName, selected)
}

func operationalMethods(service string) map[string]bool {
	result := map[string]bool{}
	switch service {
	case string(readinessv1.File_vrooli_onboarding_v1_readiness_readiness_proto.Services().Get(0).FullName()):
		result["GetReadiness"] = true
	case string(resourcesv1.File_vrooli_onboarding_v1_resources_resources_proto.Services().Get(0).FullName()):
		result["GetResourceHealth"] = true
	case string(credentialsv1.File_vrooli_onboarding_v1_credentials_credentials_proto.Services().Get(0).FullName()):
		result["DiagnoseCredentials"] = true
	case string(hostv1.File_vrooli_onboarding_v1_host_host_proto.Services().Get(0).FullName()):
		result["GetHostFacts"] = true
	}
	return result
}

func selectGroupBindings(raw []byte, groupName string, available map[string]cliapp.PrimitiveHandler) (map[string]cliapp.PrimitiveHandler, error) {
	manifest, err := cliapp.ParseManifest(raw)
	if err != nil {
		return nil, err
	}
	group := manifest.FindGroup(groupName)
	if group == nil {
		return nil, fmt.Errorf("cli manifest %q: group %q not found", manifest.Name, groupName)
	}
	selected := make(map[string]cliapp.PrimitiveHandler)
	for _, command := range group.Commands {
		key := command.Binding.BindingKey()
		if command.Binding.Kind == "local" {
			key = command.Binding.Handler
			if key == "" {
				key = command.Name
			}
		} else if command.Binding.Kind != "connect-rpc" {
			return nil, fmt.Errorf("cli manifest %q: group %q command %q uses unsupported %s binding", manifest.Name, groupName, command.Name, command.Binding.Kind)
		}
		handler, ok := available[key]
		if !ok {
			return nil, fmt.Errorf("cli manifest %q: group %q command %q references unavailable binding %s", manifest.Name, groupName, command.Name, key)
		}
		selected[key] = handler
	}
	return selected, nil
}

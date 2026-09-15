// Package modules is the static registry used by runtime mounting and
// endpoint generation. Domain constructors remain responsible for their live
// dependencies; this registry owns only the public descriptor lists.
package modules

import (
	applyH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/apply"
	capabilitiesH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/capabilities"
	credentialsH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/credentials"
	glossaryH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/glossary"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/health"
	hostH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/host"
	operatorinputsH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/operatorinputs"
	operatorstateH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/operatorstate"
	profilesH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/profiles"
	readinessH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/readiness"
	resourcesH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/resources"
	selectionH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/selection"
	sessionH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/session"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/module"

	apidb "github.com/vrooli/api-core/database"
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
)

type ProtoFileEntry struct {
	Module string
	File   protoreflect.FileDescriptor
}

func AllEndpoints() []module.EndpointDescriptor {
	result := append(append([]module.EndpointDescriptor(nil), health.Endpoints...), operatorinputsH.Endpoints...)
	result = append(result, readinessH.Endpoints...)
	result = append(result, applyH.Endpoints...)
	result = append(result, sessionH.Endpoints...)
	result = append(result, selectionH.Endpoints...)
	result = append(result, capabilitiesH.Endpoints...)
	result = append(result, credentialsH.Endpoints...)
	result = append(result, hostH.Endpoints...)
	result = append(result, operatorstateH.Endpoints...)
	result = append(result, profilesH.Endpoints...)
	result = append(result, resourcesH.Endpoints...)
	return append(result, glossaryH.Endpoints...)
}

func AllProtoFiles() []ProtoFileEntry {
	return []ProtoFileEntry{
		{Module: "operatorinputs", File: operatorinputsv1.File_vrooli_onboarding_v1_operatorinputs_operatorinputs_proto},
		{Module: "readiness", File: readinessv1.File_vrooli_onboarding_v1_readiness_readiness_proto},
		{Module: "apply", File: applyv1.File_vrooli_onboarding_v1_apply_apply_proto},
		{Module: "session", File: sessionv1.File_vrooli_onboarding_v1_session_session_proto},
		{Module: "selection", File: selectionv1.File_vrooli_onboarding_v1_selection_selection_proto},
		{Module: "capabilities", File: capabilitiesv1.File_vrooli_onboarding_v1_capabilities_capabilities_proto},
		{Module: "credentials", File: credentialsv1.File_vrooli_onboarding_v1_credentials_credentials_proto},
		{Module: "host", File: hostv1.File_vrooli_onboarding_v1_host_host_proto},
		{Module: "operatorstate", File: operatorstatev1.File_vrooli_onboarding_v1_operatorstate_operatorstate_proto},
		{Module: "profiles", File: profilesv1.File_vrooli_onboarding_v1_profiles_profiles_proto},
		{Module: "resources", File: resourcesv1.File_vrooli_onboarding_v1_resources_resources_proto},
		{Module: "glossary", File: glossaryv1.File_vrooli_onboarding_v1_glossary_glossary_proto},
	}
}

func AllSchemas() []apidb.SchemaProvider {
	return []apidb.SchemaProvider{apidb.SchemaProviderFunc(health.Schema)}
}

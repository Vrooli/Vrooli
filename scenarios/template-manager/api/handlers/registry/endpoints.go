package registry

import (
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/module"

	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/registry/registry_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "registry_list_templates",
		Path:        registryconnect.RegistryServiceListTemplatesProcedure,
		Method:      "POST",
		Summary:     "List template records",
		Description: "Returns the governed inventory of scenario, design, and resource templates.",
		Category:    "registry",
	},
	{
		ID:          "registry_get_template",
		Path:        registryconnect.RegistryServiceGetTemplateProcedure,
		Method:      "POST",
		Summary:     "Get a template record",
		Description: "Returns one template registry record by id.",
		Category:    "registry",
	},
}

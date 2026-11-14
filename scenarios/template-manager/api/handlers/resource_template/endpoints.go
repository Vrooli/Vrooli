package resourcetemplate

import (
	resourceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/resource_template/resource_template_v1connect"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "resource_template_list",
		Method:      "POST",
		Path:        resourceconnect.ResourceTemplateServiceListResourceTemplatesProcedure,
		Summary:     "List resource templates",
		Category:    "resource_template",
		Description: "Lists governed resource templates.",
	},
	{
		ID:          "resource_template_get",
		Method:      "POST",
		Path:        resourceconnect.ResourceTemplateServiceGetResourceTemplateProcedure,
		Summary:     "Get resource template",
		Category:    "resource_template",
		Description: "Shows one governed resource template.",
	},
	{
		ID:          "resource_template_validate",
		Method:      "POST",
		Path:        resourceconnect.ResourceTemplateServiceValidateResourceTemplatesProcedure,
		Summary:     "Validate resource templates",
		Category:    "resource_template",
		Description: "Validates governed resource templates.",
	},
	{
		ID:          "resource_template_generate",
		Method:      "POST",
		Path:        resourceconnect.ResourceTemplateServiceGenerateResourceTemplateProcedure,
		Summary:     "Generate resource template",
		Category:    "resource_template",
		Description: "Generates a resource from a governed template.",
	},
}

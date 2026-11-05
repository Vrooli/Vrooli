package guidance

import (
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/module"

	guidanceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/guidance/guidance_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "guidance_next_gate",
		Path:        guidanceconnect.GuidanceServiceNextGateProcedure,
		Method:      "POST",
		Summary:     "Get the next orientation gate",
		Description: "Returns the next incomplete orientation gate, checks, contract docs, and remediation pointers for a generated scenario.",
		Category:    "guidance",
	},
}

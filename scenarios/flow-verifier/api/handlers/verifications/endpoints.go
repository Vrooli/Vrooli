package verifications

import (
	"flow-verifier/internal/module"

	verificationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/verifications/verifications_v1connect"
)

// Endpoints describes the verifications Connect-RPC surface.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "verifications.start",
		Path:        verificationsconnect.VerificationsServiceStartVerificationProcedure,
		Method:      "POST",
		Summary:     "Start a verification",
		Description: "Kicks off pipeline (discover → compile → artifact → codegen → lint) for one or all flows under the given root and returns the recorded run rows.",
		Category:    "verifications",
	},
	{
		ID:          "verifications.get",
		Path:        verificationsconnect.VerificationsServiceGetVerificationProcedure,
		Method:      "POST",
		Summary:     "Verification status",
		Description: "Returns status + result for one verification run.",
		Category:    "verifications",
	},
}

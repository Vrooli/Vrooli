package manifest

import (
	"architecture-cartographer/internal/module"

	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/manifest/manifest_v1connect"
)

// Endpoints describes the manifest domain's Connect-RPC routes. Paths
// reference the generated *Procedure constants so validateTransport
// recognizes them as Connect routes.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "manifest.validate",
		Path:        manifest_v1connect.ManifestServiceValidateManifestProcedure,
		Method:      "POST",
		Summary:     "Validate a manifest source",
		Description: "Parses YAML or JSON manifest bytes, runs structural validation, and persists the result.",
		Category:    "manifest",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart manifest validate"},
	},
	{
		ID:          "manifest.get",
		Path:        manifest_v1connect.ManifestServiceGetManifestProcedure,
		Method:      "POST",
		Summary:     "Get the persisted manifest",
		Description: "Returns the most recently validated manifest for the scenario.",
		Category:    "manifest",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart manifest show"},
	},
	{
		ID:          "manifest.list-domains",
		Path:        manifest_v1connect.ManifestServiceListDomainsProcedure,
		Method:      "POST",
		Summary:     "List declared domains",
		Description: "Returns the declared domain specs for the scenario without re-downloading the full manifest.",
		Category:    "manifest",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart manifest list-domains"},
	},
}

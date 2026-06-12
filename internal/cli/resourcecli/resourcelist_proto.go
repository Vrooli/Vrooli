package resourcecli

import (
	"io"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/discovery"
	"github.com/vrooli/vrooli/internal/resources"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// ResourceListResponse maps the CLI's internal catalog types onto the
// vrooli.cli.v1 wire contract. It is the single producer-side translation from
// the rich internal `resources.Resource` to the shared, generated proto type
// that every consumer (EM, UI, …) decodes. A field rename in the proto breaks
// this mapping at compile time — that is the drift guard.
func ResourceListResponse(items []resources.Resource, failures []discovery.Failure) *cliv1.ResourceListResponse {
	resp := &cliv1.ResourceListResponse{Success: true}
	for _, item := range items {
		resp.Resources = append(resp.Resources, &cliv1.Resource{
			Name:       item.Name,
			Path:       item.Path,
			Exists:     item.Exists,
			Registered: item.Registered,
			Enabled:    item.Enabled,
			Required:   item.Required,
			HasCli:     item.HasCLI,
			Config: &cliv1.ResourceConfig{
				Enabled:     item.Config.Enabled,
				Required:    item.Config.Required,
				Description: item.Config.Description,
			},
			ControlMode:     item.ControlMode,
			Driver:          item.Driver,
			Template:        item.Template,
			PortabilityTier: item.PortabilityTier,
			ManifestPath:    item.ManifestPath,
		})
	}
	for _, failure := range failures {
		resp.DiscoveryFailures = append(resp.DiscoveryFailures, &cliv1.DiscoveryFailure{
			Kind:  failure.Kind,
			Name:  failure.Name,
			Path:  failure.Path,
			Stage: failure.Stage,
			Error: failure.Error,
		})
	}
	return resp
}

// writeResourceListJSON emits the resource-list wire contract as JSON.
func writeResourceListJSON(w io.Writer, items []resources.Resource, failures []discovery.Failure) error {
	return cliout.WriteProtoJSON(w, ResourceListResponse(items, failures))
}

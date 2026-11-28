package capability

import commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"

// ContractVersion is the version of the shared media-operation contract this
// package publishes. Bump it when a field's meaning changes; adding a field is
// compatible and needs no bump.
const ContractVersion = "1"

// ScenarioSlug is the owner slug this catalog publishes under.
const ScenarioSlug = "image-tools"

// Published projects the manifest set onto the shared cross-scenario contract.
// It is a projection, not a second declaration set: every field comes from the
// same records the CLI/UI generators and the safety policy read.
//
// A family filter is applied when non-empty so a builder can ask for just the
// generation operations without paging the whole catalog.
func (r *Registry) Published(family string) []*commonv1.MediaOperation {
	out := make([]*commonv1.MediaOperation, 0, len(r.manifests))
	for _, m := range r.manifests {
		if family != "" && string(m.Family) != family {
			continue
		}
		op := &commonv1.MediaOperation{
			Scenario:      ScenarioSlug,
			Id:            m.ID,
			Summary:       m.Summary,
			Family:        string(m.Family),
			Mechanism:     string(m.Mechanism),
			Output:        port(m.IO.Output),
			ConsentWeight: string(m.ConsentWeight),
			Backends:      append([]string(nil), m.Backends...),
			Requires: &commonv1.MediaRequirement{
				Process: m.Requires.Process,
				Weights: m.Requires.Weights,
				Network: m.Requires.Network,
			},
		}
		for _, in := range m.IO.Inputs {
			op.Inputs = append(op.Inputs, port(in))
		}
		if len(m.ParamsSchema) > 0 {
			op.ParamsSchemaJson = string(m.ParamsSchema)
		}
		out = append(out, op)
	}
	return out
}

func port(p PortSpec) *commonv1.MediaPort {
	return &commonv1.MediaPort{Name: p.Name, Kind: p.Kind, Required: p.Required}
}

// Published projects the default registry.
func Published(family string) []*commonv1.MediaOperation { return registry.Published(family) }

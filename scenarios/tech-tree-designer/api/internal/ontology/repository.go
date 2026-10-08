package ontology

import "context"

// seam: Repository persists capability ontology rows, edges, and fulfillment links.
// Production wires SQLiteRepository; tests wire mocks.FakeRepository.
type Repository interface {
	ListCapabilities(ctx context.Context, filter CapabilityFilter) ([]Capability, error)
	GetCapability(ctx context.Context, ref CapabilityRef) (Capability, error)
	UpsertCapability(ctx context.Context, capability Capability) (Capability, error)
	DeleteCapability(ctx context.Context, ref CapabilityRef) (bool, error)
	UpsertCapabilityEdge(ctx context.Context, edge CapabilityEdge) (CapabilityEdge, error)
	DeleteCapabilityEdge(ctx context.Context, edge CapabilityEdge) (bool, error)
	ListCapabilityEdges(ctx context.Context) ([]CapabilityEdge, error)
	LinkFulfillment(ctx context.Context, fulfillment Fulfillment) (Fulfillment, error)
	UnlinkFulfillment(ctx context.Context, capabilityID, scenarioSlug string) (bool, error)
	ListFulfillments(ctx context.Context, filter FulfillmentFilter) ([]Fulfillment, error)
}

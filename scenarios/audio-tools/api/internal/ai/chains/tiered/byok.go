package tiered

// AdapterIdentity is the part of a per-capability BYOK adapter that the shared
// dispatcher needs in order to stamp a result. Capability adapter interfaces
// embed these methods but keep their own capability-specific call signature.
type AdapterIdentity interface {
	ID() string
	Model() string
}

// BYOKDispatcher centralizes the per-request adapter resolution and result
// decoration that every BYOK provider chain repeats. Registries and error
// identities stay capability-owned; only the dispatch sequence is shared.
type BYOKDispatcher[Adapter AdapterIdentity, Response any] struct {
	registry   map[string]Adapter
	capability string
	missing    error
	unknown    error
	stamp      func(*Response, Adapter)
}

// NewBYOKDispatcher builds the shared dispatcher. stamp is called once per
// successful adapter call; a nil stamp skips decoration.
func NewBYOKDispatcher[Adapter AdapterIdentity, Response any](registry map[string]Adapter, capability string, missing, unknown error, stamp func(*Response, Adapter)) *BYOKDispatcher[Adapter, Response] {
	return &BYOKDispatcher[Adapter, Response]{
		registry:   registry,
		capability: capability,
		missing:    missing,
		unknown:    unknown,
		stamp:      stamp,
	}
}

// Configured reports whether any adapter is registered for this capability.
func (d *BYOKDispatcher[Adapter, Response]) Configured() bool {
	return d != nil && len(d.registry) > 0
}

// Resolve returns the adapter named by the request's provider header, applying
// the shared credential/provider guard.
func (d *BYOKDispatcher[Adapter, Response]) Resolve(key, provider string) (Adapter, error) {
	return ResolveBYOKAdapter(d.registry, key, provider, d.capability, d.missing, d.unknown)
}

// Execute resolves the adapter, performs the capability-specific call, and
// stamps the shared result identity.
func (d *BYOKDispatcher[Adapter, Response]) Execute(key, provider string, call func(Adapter) (*Response, error)) (*Response, error) {
	return ExecuteBYOK(d.registry, key, provider, d.capability, d.missing, d.unknown, call, d.stamp)
}

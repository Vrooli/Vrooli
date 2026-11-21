package tiered

import (
	"context"
	"fmt"
)

// LPBSProvider owns the provider-neutral client state shared by each Vrooli
// tier. Domain providers retain their typed execute method and response
// decoration, while availability and model identity have one implementation.
type LPBSProvider[Client any] struct {
	Client      Client
	isNil       func(Client) bool
	isAvailable func(Client, context.Context) bool
	model       func(Client) string
}

// NewLPBSProvider creates a typed LPBS provider state holder. isNil is
// injected because the concrete client is an interface in each capability.
func NewLPBSProvider[Client any](client Client, isNil func(Client) bool, isAvailable func(Client, context.Context) bool, model func(Client) string) *LPBSProvider[Client] {
	return &LPBSProvider[Client]{Client: client, isNil: isNil, isAvailable: isAvailable, model: model}
}

// LPBSClient is the client shape every capability's Vrooli tier exposes.
type LPBSClient interface {
	IsAvailable(context.Context) bool
	Model() string
}

// NewLPBSClientProvider builds LPBS provider state for an interface-typed
// client. The capability supplies only its nil test; availability and model
// delegation are identical for every LPBS tier.
func NewLPBSClientProvider[Client LPBSClient](client Client, isNil func(Client) bool) *LPBSProvider[Client] {
	return NewLPBSProvider(client, isNil,
		func(c Client, ctx context.Context) bool { return c.IsAvailable(ctx) },
		func(c Client) string { return c.Model() })
}

// Configured reports whether the capability supplied an LPBS client.
func (p *LPBSProvider[Client]) Configured() bool {
	return p != nil && p.isNil != nil && !p.isNil(p.Client)
}

// IsAvailable delegates the capability-specific health probe safely.
func (p *LPBSProvider[Client]) IsAvailable(ctx context.Context) bool {
	return p.Configured() && p.isAvailable(p.Client, ctx)
}

// Model returns the connected LPBS model identity, or empty when absent.
func (p *LPBSProvider[Client]) Model() string {
	if !p.Configured() {
		return ""
	}
	return p.model(p.Client)
}

// LPBSStateCarrier exposes the shared state held by a domain provider.
// Implementations must return nil for a nil receiver.
type LPBSStateCarrier[Client any] interface {
	LPBSState() *LPBSProvider[Client]
}

// SafeIsAvailable keeps provider-level nil receiver contracts intact when a
// domain provider embeds LPBSProvider. A promoted method cannot protect a nil
// outer provider because selecting the embedded field dereferences it first.
func SafeIsAvailable[Client any](carrier LPBSStateCarrier[Client], ctx context.Context) bool {
	if carrier == nil {
		return false
	}
	provider := carrier.LPBSState()
	return provider != nil && provider.IsAvailable(ctx)
}

// SafeModel is the nil-safe counterpart to SafeIsAvailable for model labels.
func SafeModel[Client any](carrier LPBSStateCarrier[Client]) string {
	if carrier == nil {
		return ""
	}
	provider := carrier.LPBSState()
	if provider == nil {
		return ""
	}
	return provider.Model()
}

// LPBSNotConfigured is the shared Vrooli-tier configuration error.
func LPBSNotConfigured(capability string) error {
	return fmt.Errorf("audio-tools/%s: vrooli client not configured", capability)
}

// LPBSTokenRequired is the shared Vrooli-tier credential error.
func LPBSTokenRequired(capability string) error {
	return fmt.Errorf("audio-tools/%s: LPBS token required", capability)
}

// StampLPBSResult stamps the shared Vrooli/LPBS identity on a result. A
// provider-reported id wins; the LPBS model is a fallback.
func StampLPBSResult[T ~string](tier *T, tierValue T, providerID, modelID *string, model string) {
	StampIdentity(tier, tierValue, providerID, modelID, "lpbs", model, false)
}

func lpbsGuard(configured bool, token string, notConfigured, tokenRequired error) error {
	if !configured {
		return notConfigured
	}
	if token == "" {
		return tokenRequired
	}
	return nil
}

// ExecuteLPBS centralizes the common Vrooli/LPBS guard sequence: an absent
// client and absent credential are terminal configuration errors, while the
// capability-specific call and result decoration remain injected.
func ExecuteLPBS[Response any](configured bool, token string, notConfigured, tokenRequired error, call func() (*Response, error), decorate func(*Response)) (*Response, error) {
	if err := lpbsGuard(configured, token, notConfigured, tokenRequired); err != nil {
		return nil, err
	}
	response, err := call()
	if err != nil {
		return nil, err
	}
	if decorate != nil {
		decorate(response)
	}
	return response, nil
}

// ExecuteLPBSCarrier is the nil-safe form used by capability providers: it
// resolves the carrier's LPBS state, applies the shared guards, invokes the
// capability call, and stamps the result identity.
func ExecuteLPBSCarrier[Client LPBSClient, Response any](carrier LPBSStateCarrier[Client], token string, notConfigured, tokenRequired error, call func(Client) (*Response, error), stamp func(*Response, *LPBSProvider[Client])) (*Response, error) {
	if carrier == nil {
		return nil, notConfigured
	}
	state := carrier.LPBSState()
	if state == nil || !state.Configured() {
		return nil, notConfigured
	}
	if err := lpbsGuard(true, token, notConfigured, tokenRequired); err != nil {
		return nil, err
	}
	response, err := call(state.Client)
	if err != nil {
		return nil, err
	}
	if stamp != nil {
		stamp(response, state)
	}
	return response, nil
}

// RegistryConfigured keeps the small registry-backed provider adapters from
// repeating the same availability check in every capability package.
func RegistryConfigured[Adapter any](registry map[string]Adapter) bool {
	return len(registry) > 0
}

// DispatchedModel is the stable model label for a provider selected from a
// per-request adapter registry.
func DispatchedModel() string { return "byok-dispatched" }

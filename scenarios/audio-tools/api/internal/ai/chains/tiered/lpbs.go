package tiered

import "context"

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

// ExecuteLPBS centralizes the common Vrooli/LPBS guard sequence: an absent
// client and absent credential are terminal configuration errors, while the
// capability-specific call and result decoration remain injected.
func ExecuteLPBS[Response any](configured bool, token string, notConfigured, tokenRequired error, call func() (*Response, error), decorate func(*Response)) (*Response, error) {
	if !configured {
		return nil, notConfigured
	}
	if token == "" {
		return nil, tokenRequired
	}
	response, err := call()
	if err != nil {
		return nil, err
	}
	decorate(response)
	return response, nil
}

package channels

import "context"

type ProbeResult struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

// Adapter is intentionally transport-neutral. Channel-native values must not
// cross this boundary.
type Adapter interface {
	ID() string
	Connect(context.Context, func(Envelope) error) error
	Send(context.Context, Outbound) error
	Probe(context.Context) ProbeResult
}

package effortauthority

import (
	"encoding/json"
	"net/http"
	"net/url"
)

// BrokerClientConfig is trusted owner startup data. NewBrokerClient freezes
// public metadata and TLS certificates/pools, retaining only signer handles.
type BrokerClientConfig struct {
	URL       string
	HTTP      *http.Client
	ServerPin string
	Bindings  map[string]Binding
}

func NewBrokerClient(cfg BrokerClientConfig) (*BrokerClient, error) {
	u, e := url.Parse(cfg.URL)
	if e != nil || u.Path != BrokerEndpoint || len(cfg.Bindings) == 0 {
		return nil, ErrRefused
	}
	var first Binding
	for id, b := range cfg.Bindings {
		if id != b.PolicyID {
			return nil, ErrRefused
		}
		first = b
	}
	// Reuse the same hardened private-channel TLS freezing contract. This
	// construction performs no request, enrollment, refresh or owner read.
	u.Path = ProfileContractEndpoint
	profile, e := NewProfileContractClient(ProfileContractClientConfig{URL: u.String(), HTTP: cfg.HTTP, ServerPin: cfg.ServerPin, Binding: first, Profiles: map[string]string{"private-broker": "private-broker"}})
	if e != nil {
		return nil, e
	}
	raw, e := json.Marshal(cfg.Bindings)
	if e != nil {
		return nil, ErrRefused
	}
	var bindings map[string]Binding
	if json.Unmarshal(raw, &bindings) != nil {
		return nil, ErrRefused
	}
	for _, b := range bindings {
		if b.PolicyID == "" || b.Epoch == 0 || b.Owner == "" || b.Client == "" || !validDigest(b.PolicyDigest) || b.Deadline.IsZero() {
			return nil, ErrRefused
		}
	}
	cfg.HTTP = profile.config.HTTP
	cfg.Bindings = bindings
	return &BrokerClient{frozen: &cfg}, nil
}
func (c *BrokerClient) snapshot() *BrokerClient {
	if c == nil || c.frozen == nil {
		return c
	}
	cfg := c.frozen
	return &BrokerClient{URL: cfg.URL, HTTP: cfg.HTTP, ServerPin: cfg.ServerPin, Bindings: cfg.Bindings}
}

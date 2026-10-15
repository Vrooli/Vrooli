package finiteinstallation

import (
	"context"
	"crypto/tls"
	"github.com/vrooli/api-core/effortauthority"
	ca "github.com/vrooli/vrooli/packages/credential-authority-go"
	"github.com/vrooli/vrooli/packages/credential-authority-go/finitesigning"
	"net/http"
	"net/url"
	"time"
)

// ExistingHandles retains only the exact protected owner-selected TLS handles.
// Construction reads existing credentials, but performs no HTTP request or listener effect.
type ExistingHandles struct {
	Publication       *effortauthority.ProtectedPublicationPlan
	Acceptance        *effortauthority.AcceptanceContractClient
	Profiles          *effortauthority.ProfileContractClient
	Scope, Commission *effortauthority.ScopeContractClient
	Broker            *effortauthority.BrokerClient
}

func (p *Plan) AcquireExistingHandles() (*ExistingHandles, error) {
	if p.checkWindow() != nil {
		return nil, effortauthority.ErrRefused
	}
	server, client, e := p.AcquireTLS()
	if e != nil {
		return nil, e
	}
	return p.composeHandles(server, client)
}
func (p *Plan) composeHandles(server, client *tls.Config) (*ExistingHandles, error) {
	cfg := p.bundle.Installation
	policy := cfg.Policy
	endpoint, _ := url.Parse(p.bundle.Endpoints[p.bundle.Role])
	publication, e := effortauthority.PrepareProtectedPublication(effortauthority.ProtectedPublicationConfig{Role: p.bundle.Role, Address: endpoint.Host, Installation: cfg, TLS: server})
	if e != nil {
		return nil, e
	}
	httpClient := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{TLSClientConfig: client, Proxy: nil}}
	binding := effortauthority.Binding{PolicyID: policy.ID, Epoch: policy.Epoch, PolicyDigest: effortauthority.Digest(policy), Owner: policy.Owner, Client: policy.Client, Deadline: policy.Deadline}
	accepted, e := effortauthority.NewAcceptanceContractClient(effortauthority.AcceptanceContractClientConfig{URL: p.bundle.Endpoints["acceptance"] + effortauthority.AcceptanceContractEndpoint, HTTP: httpClient, ServerPin: cfg.ServerPins["acceptance"], Binding: binding, Policy: policy})
	if e != nil {
		return nil, e
	}
	profiles := p.bundle.ProfileMappings
	profile, e := effortauthority.NewProfileContractClient(effortauthority.ProfileContractClientConfig{URL: p.bundle.Endpoints["am"] + effortauthority.ProfileContractEndpoint, HTTP: httpClient, ServerPin: cfg.ServerPins["am"], Binding: binding, Profiles: profiles})
	if e != nil {
		return nil, e
	}
	scopeCfg := effortauthority.AcceptanceContractClientConfig{URL: p.bundle.Endpoints["pm"] + effortauthority.ScopeContractEndpoint, HTTP: httpClient, ServerPin: cfg.ServerPins["pm"], Binding: binding, Policy: policy}
	scope, e := effortauthority.NewScopeContractClient(scopeCfg)
	if e != nil {
		return nil, e
	}
	scopeCfg.URL = p.bundle.Endpoints["pm"] + effortauthority.CommissionScopeEndpoint
	commission, e := effortauthority.NewCommissionScopeContractClient(scopeCfg)
	if e != nil {
		return nil, e
	}
	broker, e := effortauthority.NewBrokerClient(effortauthority.BrokerClientConfig{URL: p.bundle.Endpoints["issuer"] + effortauthority.BrokerEndpoint, HTTP: httpClient, ServerPin: cfg.ServerPins["issuer"], Bindings: map[string]effortauthority.Binding{policy.ID: binding}})
	if e != nil {
		return nil, e
	}
	return &ExistingHandles{publication, accepted, profile, scope, commission, broker}, nil
}

// AcquirePurpose keeps finite signing custody behind its existing concrete owner.
func (p *Plan) AcquirePurpose(ctx context.Context) (*finitesigning.Signer, error) {
	if p.checkWindow() != nil || p.bundle.Role != "issuer" {
		return nil, effortauthority.ErrRefused
	}
	authority, e := ca.Default()
	if e != nil {
		return nil, e
	}
	identity, e := ca.ParseIdentity(p.bundle.PurposeIdentity)
	if e != nil {
		return nil, e
	}
	signer, e := finitesigning.New(finitesigning.Config{Authority: authority, Descriptor: finitesigning.Descriptor{Identity: identity, HandleID: p.bundle.PurposeHandle, PublicKey: p.bundle.Installation.Policy.ClientKey, Policy: p.bundle.Installation.Policy}})
	if e != nil {
		return nil, e
	}
	if e = signer.CheckPolicy(ctx, p.bundle.Installation.Policy); e != nil {
		return nil, e
	}
	return signer, nil
}

// AcquirePublication reads only the existing server identity for the fourth
// read-only acceptance role; it requires no new client credential or witness.
func (p *Plan) AcquirePublication() (*effortauthority.ProtectedPublicationPlan, error) {
	if p.checkWindow() != nil || p.bundle.Role != "acceptance" {
		return nil, effortauthority.ErrRefused
	}
	authority, e := ca.Default()
	if e != nil {
		return nil, e
	}
	identity, e := ca.ParseIdentity(p.bundle.ServerIdentity)
	if e != nil {
		return nil, e
	}
	raw, e := authority.Resolve(identity, TLSField)
	if e != nil {
		return nil, e
	}
	server, e := p.parseTLS([]byte(raw), time.Now(), "server", p.bundle.Installation.ServerPins["acceptance"])
	if e != nil {
		return nil, e
	}
	endpoint, e := url.Parse(p.bundle.Endpoints["acceptance"])
	if e != nil {
		return nil, e
	}
	return effortauthority.PrepareProtectedPublication(effortauthority.ProtectedPublicationConfig{Role: "acceptance", Address: endpoint.Host, Installation: p.bundle.Installation, TLS: server})
}

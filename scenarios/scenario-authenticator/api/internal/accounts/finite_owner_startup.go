// finite_owner_startup.go binds finite publication to the actual existing issuer service graph.
package accounts

import (
	"context"
	"encoding/json"
	"github.com/vrooli/api-core/effortauthority"
	"github.com/vrooli/vrooli/packages/credential-authority-go/finitesigning"
	isolation "github.com/vrooli/vrooli/packages/nativeisolation"
	"net/http"
	"time"
)

type FiniteOwnerStartup struct {
	config            effortauthority.Installation
	scope, commission *effortauthority.ScopeContractClient
	purpose           *finitesigning.Signer
	witness           *isolation.Witness
	peers             map[string]map[string]effortauthority.BrokerGrant
	publication       *effortauthority.ProtectedPublicationPlan
	jwtKID            string
}

func PrepareFiniteOwnerStartup(ctx context.Context, cfg effortauthority.Installation, scope, commission *effortauthority.ScopeContractClient, purpose *finitesigning.Signer, witness *isolation.Witness, peers map[string]map[string]effortauthority.BrokerGrant, publication *effortauthority.ProtectedPublicationPlan, existingJWTKID string) (*FiniteOwnerStartup, error) {
	frozen, e := cfg.Freeze()
	if e != nil || !frozen.Enabled || existingJWTKID == "" || scope == nil || scope.CheckInstallation(frozen.Policy) != nil || commission == nil || commission.CheckApprovalInstallation(frozen.Policy) != nil || purpose == nil || purpose.CheckPolicy(ctx, frozen.Policy) != nil || witness == nil || witness.CheckPolicy(frozen.Policy) != nil || witness.Qualify(ctx) != nil || publication == nil || publication.Role() != "issuer" || publication.CheckInstallation(frozen) != nil {
		return nil, effortauthority.ErrRefused
	}
	raw, e := json.Marshal(peers)
	if e != nil {
		return nil, effortauthority.ErrRefused
	}
	var copy map[string]map[string]effortauthority.BrokerGrant
	if json.Unmarshal(raw, &copy) != nil || len(copy) == 0 {
		return nil, effortauthority.ErrRefused
	}
	return &FiniteOwnerStartup{frozen, scope, commission, purpose, witness, copy, publication, existingJWTKID}, nil
}
func (s *FiniteOwnerStartup) ExistingJWTKID() string {
	if s == nil {
		return ""
	}
	return s.jwtKID
}

// Compose uses the service's actual existing owner/authentication graph and
// canonical ledger. Only actual verified human consent may create a policy.
func (s *FiniteOwnerStartup) Compose(ctx context.Context, service *Service, ledger *effortauthority.SQLStore) (http.Handler, *effortauthority.ProtectedPublication, error) {
	if s == nil || service == nil || ledger == nil {
		return nil, nil, effortauthority.ErrRefused
	}
	prepared, e := service.PrepareFiniteInstallation(ctx, s.config, FiniteIssuerSetup{Store: ledger, Scope: s.scope, ApprovalScope: s.commission, Custody: &effortauthority.CustodySigner{Purpose: s.purpose}, Peers: s.peers, Proposals: map[string]effortauthority.Policy{s.config.Policy.ID: s.config.Policy}, Origin: s.config.Origin, Isolation: s.witness})
	if e != nil {
		return nil, nil, e
	}
	handlers := prepared.Handlers()
	if handlers.Consent == nil || handlers.Broker == nil {
		return nil, nil, effortauthority.ErrRefused
	}
	mux := http.NewServeMux()
	mux.Handle(effortauthority.BrokerEndpoint, handlers.Broker)
	private, e := s.publication.Start(mux)
	if e != nil {
		return nil, nil, e
	}
	return handlers.Consent, private, nil
}

// CheckStartup rechecks the original owner window before graph construction;
// it never renews a deadline or treats a readiness boolean as authority.
func (s *FiniteOwnerStartup) CheckStartup() error {
	if s == nil {
		return effortauthority.ErrRefused
	}
	now := time.Now()
	if !now.Before(s.config.Policy.Deadline) || (s.config.Enabled && now.Before(s.config.Policy.NotBefore)) {
		return effortauthority.ErrRefused
	}
	return nil
}

package accounts

import (
	"context"
	"crypto"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/vrooli/api-core/effortauthority"
	isolationowner "github.com/vrooli/vrooli/packages/nativeisolation"
)

// FiniteIssuerSetup is supplied only by an owner-approved installation. The
// identity service owns the store, approval and sign-only handle. Scope reads
// belong to their actual accepted-work/team/profile owners. No key generation,
// route registration, schema migration or privilege activation happens here.
type FiniteIssuerSetup struct {
	Store         effortauthority.Store
	Scope         effortauthority.ScopeOwner
	ApprovalScope *effortauthority.ScopeContractClient
	Custody       *effortauthority.CustodySigner
	Peers         map[string]map[string]effortauthority.BrokerGrant
	Proposals     map[string]effortauthority.Policy
	Origin        string
	// Isolation consumes the runtime owner's current isolation qualification,
	// not a submitted boolean. It must prove admitted native code cannot read or
	// control issuer/custody state. SameUID chmod/loopback fails qualification.
	Isolation effortauthority.NativeIsolationQualifier
}
type FiniteIssuerHandlers struct {
	Consent http.Handler
	Broker  http.Handler
}

func (s *Service) FiniteIssuer(setup FiniteIssuerSetup) (FiniteIssuerHandlers, error) {
	if s == nil || setup.Store == nil || setup.Scope == nil || setup.Custody == nil || len(setup.Peers) == 0 || len(setup.Proposals) == 0 || setup.Isolation == nil {
		return FiniteIssuerHandlers{}, effortauthority.ErrRefused
	}
	// Only concrete owner evidence is supported. No arbitrary success callback
	// can weaken the historical guard; prepared startup requires a live witness.
	switch guard := setup.Isolation.(type) {
	case *effortauthority.NativeUIDIsolation:
		if guard == nil {
			return FiniteIssuerHandlers{}, effortauthority.ErrRefused
		}
	case *isolationowner.Witness:
		if guard == nil {
			return FiniteIssuerHandlers{}, effortauthority.ErrRefused
		}
	default:
		return FiniteIssuerHandlers{}, effortauthority.ErrRefused
	}
	// Freeze nonsecret policy/peer tables and the custody handle lookup, never
	// export or copy key material. Later setup mutations cannot amend installation.
	raw, err := json.Marshal(struct {
		Proposals map[string]effortauthority.Policy
		Peers     map[string]map[string]effortauthority.BrokerGrant
	}{setup.Proposals, setup.Peers})
	if err != nil {
		return FiniteIssuerHandlers{}, effortauthority.ErrRefused
	}
	var tables struct {
		Proposals map[string]effortauthority.Policy
		Peers     map[string]map[string]effortauthority.BrokerGrant
	}
	if json.Unmarshal(raw, &tables) != nil {
		return FiniteIssuerHandlers{}, effortauthority.ErrRefused
	}
	setup.Proposals, setup.Peers = tables.Proposals, tables.Peers
	custody := &effortauthority.CustodySigner{Clients: map[string]crypto.Signer{}, Purpose: setup.Custody.Purpose}
	for id, handle := range setup.Custody.Clients {
		custody.Clients[id] = handle
	}
	setup.Custody = custody
	origin, err := url.Parse(setup.Origin)
	if err != nil || origin.Scheme != "https" || origin.Host == "" || origin.User != nil || origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" {
		return FiniteIssuerHandlers{}, effortauthority.ErrRefused
	}
	for id, p := range setup.Proposals {
		if p.ID != id || p.Validate() != nil || setup.Custody.CheckPolicy(context.Background(), p) != nil {
			return FiniteIssuerHandlers{}, effortauthority.ErrRefused
		}
	}
	authority := &effortauthority.Authority{Store: setup.Store, Owners: s, Approver: s, Scope: setup.Scope}
	if setup.ApprovalScope != nil {
		authority.ApprovalScope = setup.ApprovalScope
	}
	consent := &effortauthority.ConsentHandler{Authority: authority, Proposals: setup.Proposals, Origin: setup.Origin, CookieName: "vrooli_access_token"}
	broker := &effortauthority.BrokerHandler{Authority: authority, Custody: setup.Custody, Peers: setup.Peers}
	guard := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if setup.Isolation.Qualify(r.Context()) != nil {
				http.Error(w, "finite issuer isolation unqualified", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	return FiniteIssuerHandlers{Consent: guard(consent), Broker: guard(broker)}, nil
}

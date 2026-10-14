package accounts

import (
	"context"
	"encoding/hex"
	"github.com/vrooli/api-core/effortauthority"
	"github.com/vrooli/vrooli/packages/credential-authority-go/finitesigning"
	isolationowner "github.com/vrooli/vrooli/packages/nativeisolation"
)

type PreparedFiniteInstallation struct {
	config   effortauthority.Installation
	handlers FiniteIssuerHandlers
}

func (s *Service) PrepareFiniteInstallation(ctx context.Context, cfg effortauthority.Installation, setup FiniteIssuerSetup) (*PreparedFiniteInstallation, error) {
	frozen, e := cfg.Freeze()
	if e != nil {
		return nil, e
	}
	if !frozen.Enabled {
		return &PreparedFiniteInstallation{config: frozen}, nil
	}
	witness, ok := setup.Isolation.(*isolationowner.Witness)
	scope, scopeOK := setup.Scope.(*effortauthority.ScopeContractClient)
	var purpose *finitesigning.Signer
	if setup.Custody != nil {
		purpose, _ = setup.Custody.Purpose.(*finitesigning.Signer)
	}
	if !scopeOK || scope == nil || scope.CheckInstallation(frozen.Policy) != nil || setup.ApprovalScope == nil || setup.ApprovalScope.CheckApprovalInstallation(frozen.Policy) != nil || setup.ApprovalScope.CheckEffortScope(ctx, frozen.Policy) != nil || purpose == nil || setup.Custody == nil || setup.Custody.Purpose == nil || setup.Custody.CheckPolicy(ctx, frozen.Policy) != nil || !ok || witness == nil || witness.CheckPolicy(frozen.Policy) != nil || setup.Isolation.Qualify(ctx) != nil || setup.Origin != frozen.Origin || len(setup.Proposals) != 1 || effortauthority.Digest(setup.Proposals[frozen.Policy.ID]) != effortauthority.Digest(frozen.Policy) {
		return nil, effortauthority.ErrRefused
	}
	for pin, grants := range setup.Peers {
		raw, err := hex.DecodeString(pin)
		if err != nil || len(raw) != 32 || len(grants) == 0 {
			return nil, effortauthority.ErrRefused
		}
		for id, grant := range grants {
			if id != frozen.Policy.ID || grant.PolicyID != id || len(grant.Operations) == 0 {
				return nil, effortauthority.ErrRefused
			}
			for operation, allowed := range grant.Operations {
				if !allowed {
					return nil, effortauthority.ErrRefused
				}
				switch operation {
				case "check-binding", "check-proof", "prepare", "reserve", "accept", "settle", "observe-ingress", "read-reservation", "sign":
				default:
					return nil, effortauthority.ErrRefused
				}
			}
		}
	}
	h, e := s.FiniteIssuer(setup)
	if e != nil {
		return nil, e
	}
	return &PreparedFiniteInstallation{frozen, h}, nil
}

// Handlers returns nothing for disabled/incomplete startup; callers publish
// both only after their owner installation validates the private TLS listener.
func (p *PreparedFiniteInstallation) Handlers() FiniteIssuerHandlers {
	if p == nil || !p.config.Enabled {
		return FiniteIssuerHandlers{}
	}
	return p.handlers
}

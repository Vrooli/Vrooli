// finite_publication.go binds protected read-only acceptance to the canonical governed work owner.
package backlog

import (
	"context"
	"encoding/json"
	"github.com/vrooli/api-core/effortauthority"
	"time"
)

type FiniteReadPublication struct {
	config      effortauthority.Installation
	targets     map[string]FiniteCommissionTarget
	peers       map[string]map[string]effortauthority.AcceptanceContractGrant
	publication *effortauthority.ProtectedPublicationPlan
}

// Preparation captures only source-owner selected read targets. It neither
// accepts/revokes a record nor creates any public disposition endpoint.
func PrepareFiniteReadPublication(cfg effortauthority.Installation, targets map[string]FiniteCommissionTarget, peers map[string]map[string]effortauthority.AcceptanceContractGrant, publication *effortauthority.ProtectedPublicationPlan) (*FiniteReadPublication, error) {
	descriptor := cfg
	descriptor.Enabled = true
	frozen, e := descriptor.Freeze()
	if e != nil || cfg.Enabled || !time.Now().Before(frozen.Policy.Deadline) || len(targets) != 1 || publication == nil || publication.Role() != "acceptance" || publication.CheckInstallation(descriptor) != nil {
		return nil, effortauthority.ErrRefused
	}
	if _, ok := targets[frozen.Policy.Effort]; !ok {
		return nil, effortauthority.ErrRefused
	}
	for _, table := range peers {
		if len(table) != 1 {
			return nil, effortauthority.ErrRefused
		}
		grant, ok := table[frozen.Policy.ID]
		if !ok || effortauthority.Digest(grant.Policy) != effortauthority.Digest(frozen.Policy) {
			return nil, effortauthority.ErrRefused
		}
	}
	raw, e := json.Marshal(struct {
		Targets map[string]FiniteCommissionTarget
		Peers   map[string]map[string]effortauthority.AcceptanceContractGrant
	}{targets, peers})
	if e != nil {
		return nil, effortauthority.ErrRefused
	}
	var copy struct {
		Targets map[string]FiniteCommissionTarget
		Peers   map[string]map[string]effortauthority.AcceptanceContractGrant
	}
	if json.Unmarshal(raw, &copy) != nil {
		return nil, effortauthority.ErrRefused
	}
	frozen.Enabled = false
	return &FiniteReadPublication{frozen, copy.Targets, copy.Peers, publication}, nil
}
func (p *FiniteReadPublication) CheckStartup() error {
	if p == nil || !time.Now().Before(p.config.Policy.Deadline) {
		return effortauthority.ErrRefused
	}
	return nil
}

// Publish binds the actual newly composed canonical backlog Handler. Reads
// check current acceptance/revocation; construction performs no approval write.
func (p *FiniteReadPublication) Publish(ctx context.Context, h *Handler) (*effortauthority.ProtectedPublication, error) {
	if ctx.Err() != nil || p.CheckStartup() != nil {
		return nil, effortauthority.ErrRefused
	}
	owner, e := NewFiniteCommissionOwner(h, p.targets)
	if e != nil {
		return nil, e
	}
	handler, e := effortauthority.NewAcceptanceContractHandler(owner, p.peers)
	if e != nil {
		return nil, e
	}
	return p.publication.Start(handler)
}

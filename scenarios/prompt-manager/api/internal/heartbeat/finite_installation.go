package heartbeat

import (
	"context"
	"github.com/vrooli/api-core/effortauthority"
	isolationowner "github.com/vrooli/vrooli/packages/nativeisolation"
	"prompt-manager/internal/store"
	"slices"
	"strings"
)

// PreparedFiniteInstallation freezes configuration before publishing runtime
// ownership. No policy approval, queue write, client enrollment or start occurs.
type PreparedFiniteInstallation struct {
	config  effortauthority.Installation
	efforts *FiniteEffortAuthority
	teams   *store.FileTeamStore
	tasks   *AgentManagerClient
	scope   *FiniteEffortScope
	witness *isolationowner.Witness
}

func PrepareFiniteInstallation(ctx context.Context, cfg effortauthority.Installation, efforts *FiniteEffortAuthority, isolation *isolationowner.Witness, scope *FiniteEffortScope) (*PreparedFiniteInstallation, error) {
	frozen, e := cfg.Freeze()
	if e != nil {
		return nil, e
	}
	if !frozen.Enabled {
		return &PreparedFiniteInstallation{config: frozen}, nil
	}
	if scope == nil || scope.Accepted == nil || scope.Profiles == nil || scope.Teams == nil || scope.CheckEffortScope(ctx, frozen.Policy) != nil || efforts == nil || efforts.Authority == nil || efforts.Signer == nil || efforts.Tasks == nil || len(efforts.Bindings) == 0 || isolation == nil || isolation.CheckPolicy(frozen.Policy) != nil || isolation.Qualify(ctx) != nil {
		return nil, effortauthority.ErrRefused
	}
	frozenScope, e := freezeFiniteScope(frozen.Policy, scope)
	if e != nil {
		return nil, e
	}
	teams, ok := frozenScope.Teams.(*store.FileTeamStore)
	tasks, taskOK := efforts.Tasks.(*AgentManagerClient)
	if !ok || teams == nil || !taskOK || tasks == nil {
		return nil, effortauthority.ErrRefused
	}
	copy := *efforts
	copy.Bindings = map[string]effortauthority.Binding{}
	for key, b := range efforts.Bindings {
		parts := strings.Split(key, "/")
		current, err := efforts.Authority.CheckBinding(ctx, b)
		if len(parts) != 2 || parts[0] != frozen.Policy.Team || !slices.Contains(frozen.Policy.Members, parts[1]) || err != nil || effortauthority.Digest(current) != effortauthority.Digest(frozen.Policy) || b.PolicyID != frozen.Policy.ID || b.PolicyDigest != effortauthority.Digest(frozen.Policy) || !b.Deadline.Equal(frozen.Policy.Deadline) {
			return nil, effortauthority.ErrRefused
		}
		copy.Bindings[key] = b
	}
	return &PreparedFiniteInstallation{config: frozen, efforts: &copy, teams: teams, tasks: tasks, scope: frozenScope, witness: isolation}, nil
}
func (p *PreparedFiniteInstallation) Apply(r *FiniteLeaderRuntime) error {
	if p == nil || r == nil {
		return effortauthority.ErrRefused
	}
	if !p.config.Enabled {
		return nil
	}
	if r.Efforts != nil || p.tasks == nil || p.tasks.efforts != nil || r.Executor == nil || r.Executor.teamStore != p.teams || r.Executor.agentClient != p.tasks || p.scope == nil || p.witness == nil || p.witness.CheckPolicy(p.config.Policy) != nil || p.witness.Qualify(context.Background()) != nil || p.scope.CheckEffortScope(context.Background(), p.config.Policy) != nil {
		return effortauthority.ErrRefused
	}
	for _, b := range p.efforts.Bindings {
		current, e := p.efforts.Authority.CheckBinding(context.Background(), b)
		if e != nil || effortauthority.Digest(current) != effortauthority.Digest(p.config.Policy) {
			return effortauthority.ErrRefused
		}
	}
	p.tasks.efforts = p.efforts
	r.Efforts = p.efforts
	return nil
}

// The concrete owner-computed private reader satisfies the existing scope
// interface; portable read-only profile identity remains outside qualification.
var _ NativeProfileReader = (*effortauthority.ProfileContractClient)(nil)

// Frozen read composition is built before any scheduler/finite run authority.
// Private commissioning can inspect the disabled authored target; admission
// uses a distinct endpoint and still requires enabled current owner state.
func freezeFiniteScope(p effortauthority.Policy, scope *FiniteEffortScope) (*FiniteEffortScope, error) {
	if scope == nil || scope.Teams == nil {
		return nil, effortauthority.ErrRefused
	}
	accepted, ok := scope.Accepted.(*effortauthority.AcceptanceContractClient)
	profiles, profileOK := scope.Profiles.(*effortauthority.ProfileContractClient)
	if !ok || accepted == nil || accepted.CheckInstallation(p) != nil || !profileOK || profiles == nil || profiles.CheckInstallation(p) != nil {
		return nil, effortauthority.ErrRefused
	}
	copy := *scope
	return &copy, nil
}
func scopeGrantsMatch(p effortauthority.Policy, peers map[string]map[string]effortauthority.AcceptanceContractGrant) error {
	for _, grants := range peers {
		for _, g := range grants {
			if effortauthority.Digest(g.Policy) != effortauthority.Digest(p) {
				return effortauthority.ErrRefused
			}
		}
	}
	return nil
}
func PrepareFiniteScopeContract(ctx context.Context, p effortauthority.Policy, scope *FiniteEffortScope, peers map[string]map[string]effortauthority.AcceptanceContractGrant) (*effortauthority.ScopeContractHandler, error) {
	frozen, e := freezeFiniteScope(p, scope)
	if e != nil || scopeGrantsMatch(p, peers) != nil || frozen.CheckEffortScope(ctx, p) != nil {
		return nil, effortauthority.ErrRefused
	}
	return effortauthority.NewScopeContractHandler(frozen, peers)
}

type commissionScopeOwner struct{ scope *FiniteEffortScope }

func (s commissionScopeOwner) CheckEffortScope(ctx context.Context, p effortauthority.Policy) error {
	return s.scope.checkScope(ctx, p, false)
}
func PrepareFiniteCommissionScopeContract(ctx context.Context, p effortauthority.Policy, scope *FiniteEffortScope, peers map[string]map[string]effortauthority.AcceptanceContractGrant) (*effortauthority.ScopeContractHandler, error) {
	frozen, e := freezeFiniteScope(p, scope)
	if e != nil || scopeGrantsMatch(p, peers) != nil || frozen.checkScope(ctx, p, false) != nil {
		return nil, effortauthority.ErrRefused
	}
	return effortauthority.NewCommissionScopeContractHandler(commissionScopeOwner{frozen}, peers)
}

var _ AcceptedEffortReader = (*effortauthority.AcceptanceContractClient)(nil)

// PrepareFiniteScopeReadContracts composes private reads while the authored
// target stays disabled. Consent uses the commissioning read; live admission
// independently requires enabled current team/heartbeat state on every read.
// No scheduler, policy approval or queue is created by this phase.
func PrepareFiniteScopeReadContracts(ctx context.Context, p effortauthority.Policy, scope *FiniteEffortScope, peers map[string]map[string]effortauthority.AcceptanceContractGrant) (*effortauthority.ScopeContractHandler, *effortauthority.ScopeContractHandler, error) {
	frozen, e := freezeFiniteScope(p, scope)
	if e != nil || scopeGrantsMatch(p, peers) != nil || frozen.checkScope(ctx, p, false) != nil {
		return nil, nil, effortauthority.ErrRefused
	}
	commission, e := effortauthority.NewCommissionScopeContractHandler(commissionScopeOwner{frozen}, peers)
	if e != nil {
		return nil, nil, e
	}
	admission, e := effortauthority.NewScopeContractHandler(frozen, peers)
	if e != nil {
		return nil, nil, e
	}
	return commission, admission, nil
}

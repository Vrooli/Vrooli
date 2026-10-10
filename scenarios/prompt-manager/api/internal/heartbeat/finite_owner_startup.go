// finite_owner_startup.go composes protected owner reads and exact live graph installation before scheduler effects.
package heartbeat

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"github.com/vrooli/api-core/effortauthority"
	isolation "github.com/vrooli/vrooli/packages/nativeisolation"
	"net/http"
	"prompt-manager/internal/store"
	"time"
)

// FiniteOwnerStartup is selected by trusted executable composition, never by a
// request, environment flag, serialized witness or arbitrary success callback.
// Commissioning has no run authority; live requires current concrete owners.
type FiniteOwnerStartup struct {
	config      effortauthority.Installation
	accepted    *effortauthority.AcceptanceContractClient
	profiles    *effortauthority.ProfileContractClient
	broker      *effortauthority.BrokerClient
	witness     *isolation.Witness
	publication *effortauthority.ProtectedPublicationPlan
	peers       map[string]map[string]effortauthority.AcceptanceContractGrant
}

func PrepareFiniteOwnerStartup(ctx context.Context, cfg effortauthority.Installation, accepted *effortauthority.AcceptanceContractClient, profiles *effortauthority.ProfileContractClient, broker *effortauthority.BrokerClient, witness *isolation.Witness, publication *effortauthority.ProtectedPublicationPlan, peers map[string]map[string]effortauthority.AcceptanceContractGrant) (*FiniteOwnerStartup, error) {
	descriptor := cfg
	descriptor.Enabled = true
	frozen, e := descriptor.Freeze()
	if e != nil {
		return nil, e
	}
	frozen.Enabled = cfg.Enabled
	if accepted == nil || profiles == nil || accepted.CheckInstallation(frozen.Policy) != nil || profiles.CheckInstallation(frozen.Policy) != nil || publication == nil || publication.Role() != "pm" || publication.CheckInstallation(descriptor) != nil {
		return nil, effortauthority.ErrRefused
	}
	if frozen.Enabled {
		if broker == nil || witness == nil || witness.CheckPolicy(frozen.Policy) != nil || witness.Qualify(ctx) != nil {
			return nil, effortauthority.ErrRefused
		}
	} else if broker != nil || witness != nil {
		return nil, effortauthority.ErrRefused
	}
	raw, e := json.Marshal(peers)
	if e != nil {
		return nil, effortauthority.ErrRefused
	}
	var grants map[string]map[string]effortauthority.AcceptanceContractGrant
	if json.Unmarshal(raw, &grants) != nil || len(grants) == 0 {
		return nil, effortauthority.ErrRefused
	}
	for pin, table := range grants {
		b, e := hex.DecodeString(pin)
		if e != nil || len(b) != 32 || len(table) != 1 {
			return nil, effortauthority.ErrRefused
		}
		g, ok := table[frozen.Policy.ID]
		if !ok || effortauthority.Digest(g.Policy) != effortauthority.Digest(frozen.Policy) {
			return nil, effortauthority.ErrRefused
		}
	}
	return &FiniteOwnerStartup{config: frozen, accepted: accepted, profiles: profiles, broker: broker, witness: witness, publication: publication, peers: grants}, nil
}
func (s *FiniteOwnerStartup) Live() bool { return s != nil && s.config.Enabled }
func (s *FiniteOwnerStartup) Team() string {
	if s == nil {
		return ""
	}
	return s.config.Policy.Team
}

// PublishReadPhase binds only exact protected scope endpoints. The actual new
// team owner supplies authored state before issuer consent/current run checks.
// Explicit Start opens the supplied approved private address; construction did not.
func (s *FiniteOwnerStartup) PublishReadPhase(ctx context.Context, teams *store.FileTeamStore) (*effortauthority.ProtectedPublication, error) {
	if s == nil || teams == nil {
		return nil, effortauthority.ErrRefused
	}
	p := s.config.Policy
	scope := &FiniteEffortScope{Teams: teams, Accepted: s.accepted, Profiles: s.profiles}
	team, e := teams.Get(ctx, p.Team)
	if e != nil || team == nil {
		return nil, effortauthority.ErrRefused
	}
	cfg, e := teams.GetHeartbeatConfig(ctx, p.Team, team.Coordination.LeadAgentID)
	if e != nil || cfg == nil {
		return nil, effortauthority.ErrRefused
	}
	if !s.Live() && (team.Enabled || cfg.Enabled) {
		return nil, effortauthority.ErrRefused
	}
	commission, admission, e := PrepareFiniteScopeReadContracts(ctx, p, scope, s.peers)
	if e != nil {
		return nil, e
	}
	mux := http.NewServeMux()
	mux.Handle(effortauthority.CommissionScopeEndpoint, commission)
	if s.Live() {
		mux.Handle(effortauthority.ScopeContractEndpoint, admission)
	} else {
		mux.HandleFunc(effortauthority.ScopeContractEndpoint, func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "commissioning startup has no live admission authority", http.StatusForbidden)
		})
	}
	return s.publication.Start(mux)
}

// PrepareForGraph attaches exact existing task/run transport and team owner.
// It reads current approved authority only after scope publication is running.
func (s *FiniteOwnerStartup) PrepareForGraph(ctx context.Context, teams *store.FileTeamStore, tasks *AgentManagerClient) (*PreparedFiniteInstallation, error) {
	if s == nil || !s.Live() || teams == nil || tasks == nil {
		return nil, effortauthority.ErrRefused
	}
	p := s.config.Policy
	scope := &FiniteEffortScope{Teams: teams, Accepted: s.accepted, Profiles: s.profiles}
	binding := effortauthority.Binding{PolicyID: p.ID, PolicyDigest: effortauthority.Digest(p), Owner: p.Owner, Client: p.Client, Epoch: p.Epoch, Deadline: p.Deadline}
	effects := &FiniteEffortAuthority{Authority: s.broker, Signer: s.broker, Tasks: tasks, Bindings: map[string]effortauthority.Binding{}}
	for _, member := range p.Members {
		effects.Bindings[p.Team+"/"+member] = binding
	}
	prepared, e := PrepareFiniteInstallation(ctx, s.config, effects, s.witness, scope)
	if e != nil {
		return nil, e
	}
	return prepared, nil
}

func (s *FiniteOwnerStartup) Member(id string) bool {
	if s == nil {
		return false
	}
	for _, member := range s.config.Policy.Members {
		if member == id {
			return true
		}
	}
	return false
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

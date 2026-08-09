package teams

import (
	"context"
	"encoding/json"
	"net/http"

	"prompt-manager/interop"
	"prompt-manager/teamconfig"

	"github.com/gorilla/mux"
)

// teamDocReader is satisfied by stores that provide team member documents.
// This interface replaces a concrete *store.FileTeamStore type assertion,
// making the export handler testable with mocks.
type teamDocReader interface {
	GetResponsibilities(ctx context.Context, teamID, agentID string) (string, error)
	GetHeartbeatInstructions(ctx context.Context, teamID, agentID string) (string, error)
}

// ExportClaudeCode handles GET /teams/{id}/export/claude-code - exports PM team as CC config.
func (h *Handlers) ExportClaudeCode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	// Load team
	team, err := h.teamStore.Get(ctx, id)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}
	if !teamconfig.UsesSingleProcessInterop(team.Contract()) {
		http.Error(w, "Claude Code export is only supported for leader-led single-process teams", http.StatusBadRequest)
		return
	}

	// Load members
	members, err := h.relationStore.ListTeamMembers(ctx, id)
	if err != nil {
		http.Error(w, "failed to load members", http.StatusInternalServerError)
		return
	}

	// Build snapshot
	snapshot := &interop.PMTeamSnapshot{
		Team: *team,
	}

	// Load each member's agent and documents
	docReader, hasDocReader := h.teamStore.(teamDocReader)
	for _, rel := range members {
		pm := interop.PMTeamMember{
			Relation: rel,
		}
		if agent, err := h.agentStore.Get(ctx, rel.AgentID); err == nil {
			pm.Agent = *agent
		}
		if hasDocReader {
			if resp, err := docReader.GetResponsibilities(ctx, id, rel.AgentID); err == nil {
				pm.Responsibilities = resp
			}
			if instr, err := docReader.GetHeartbeatInstructions(ctx, id, rel.AgentID); err == nil {
				pm.HeartbeatInstr = instr
			}
		}
		snapshot.Members = append(snapshot.Members, pm)
	}

	// Load roles
	if roles, err := h.teamStore.GetRoles(ctx, id); err == nil {
		snapshot.Roles = roles.Roles
	}

	// Load org chart
	if org, err := h.teamStore.GetOrgChart(ctx, id); err == nil {
		snapshot.OrgEdges = org.Edges
	}

	// Convert to CC config
	converter := interop.ClaudeCodeConverter{}
	ccConfig, err := converter.FromPMTeam(snapshot)
	if err != nil {
		http.Error(w, "export conversion failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ccConfig)
}

package teams

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"prompt-manager/interop"
	"prompt-manager/store"
	"prompt-manager/validation"
	"strings"
)

// AvailableCCTeam represents a Claude Code team found on disk.
type AvailableCCTeam struct {
	Name        string `json:"name"`
	MemberCount int    `json:"memberCount"`
}

// ListAvailableCCTeams handles GET /teams/import/claude-code/available - lists CC teams on disk.
func (h *Handlers) ListAvailableCCTeams(w http.ResponseWriter, r *http.Request) {
	teams, err := h.listCCTeamDirs()
	if err != nil {
		// If the directory doesn't exist, return empty list (not an error)
		if os.IsNotExist(err) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]AvailableCCTeam{})
			return
		}
		http.Error(w, fmt.Sprintf("failed to list CC teams: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(teams)
}

// defaultListCCTeamDirs reads ~/.claude/teams/ and returns available teams.
func defaultListCCTeamDirs() ([]AvailableCCTeam, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine home directory: %w", err)
	}
	teamsDir := filepath.Join(homeDir, ".claude", "teams")

	entries, err := os.ReadDir(teamsDir)
	if err != nil {
		return nil, err
	}

	var result []AvailableCCTeam
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		team := AvailableCCTeam{Name: entry.Name()}

		// Try to read config.json and count members
		configPath := filepath.Join(teamsDir, entry.Name(), "config.json")
		data, err := os.ReadFile(configPath)
		if err == nil {
			var config struct {
				Members []json.RawMessage `json:"members"`
			}
			if json.Unmarshal(data, &config) == nil {
				team.MemberCount = len(config.Members)
			}
		}

		result = append(result, team)
	}

	if result == nil {
		result = []AvailableCCTeam{}
	}
	return result, nil
}

// ImportCCRequest is the request body for importing a Claude Code team.
type ImportCCRequest struct {
	TeamName string `json:"teamName"`
}

// ImportClaudeCode handles POST /teams/import/claude-code - imports a CC team.
func (h *Handlers) ImportClaudeCode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ImportCCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.TeamName) == "" {
		http.Error(w, "teamName is required", http.StatusBadRequest)
		return
	}

	// Read CC team config (uses injected reader for testability)
	data, err := h.readCCConfig(req.TeamName)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, fmt.Sprintf("Claude Code team %q not found", req.TeamName), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("failed to read CC team config: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse CC config into tool-agnostic format
	toolConfig, err := interop.ParseCCConfig(data, req.TeamName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Convert to PM import
	converter := interop.ClaudeCodeConverter{}
	pmImport, err := converter.ToPMTeam(toolConfig)
	if err != nil {
		http.Error(w, fmt.Sprintf("conversion failed: %v", err), http.StatusBadRequest)
		return
	}

	// Create the PM team
	if err := h.teamStore.Create(ctx, &pmImport.Team); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create agents that don't exist yet
	for i := range pmImport.Agents {
		agent := &pmImport.Agents[i]
		if existing, _ := h.agentStore.Get(ctx, agent.ID); existing != nil {
			continue // Agent already exists, skip
		}
		// Generate ID if needed
		if agent.ID == "" {
			agent.ID = validation.Slugify(agent.DisplayName)
		}
		if err := h.agentStore.Create(ctx, agent); err != nil {
			// Best effort: skip if agent creation fails
			continue
		}
	}

	// Create member relations
	for i := range pmImport.Members {
		rel := &pmImport.Members[i]
		if err := h.relationStore.SetTeamMember(ctx, rel); err != nil {
			continue // Best effort
		}
	}

	// Set org chart if there are edges
	if len(pmImport.OrgEdges) > 0 {
		org := &store.OrgChart{
			TeamID: pmImport.Team.ID,
			Edges:  pmImport.OrgEdges,
		}
		_ = h.teamStore.SetOrgChart(ctx, pmImport.Team.ID, org)
	}

	// Regenerate index
	if h.indexStore != nil {
		_ = h.indexStore.RegenerateTeams(ctx)
	}

	// Return created team details
	team, err := h.teamStore.Get(ctx, pmImport.Team.ID)
	if err != nil {
		http.Error(w, "team created but failed to fetch details", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(h.toDetailsResponse(ctx, team))
}

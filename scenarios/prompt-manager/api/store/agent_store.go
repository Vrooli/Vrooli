package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// FileAgentStore implements AgentStore using the file system
type FileAgentStore struct {
	storeDir      string
	relationStore RelationStore
	teamStore     TeamStore
}

// NewFileAgentStore creates a new file-based agent store
func NewFileAgentStore(storeDir string, relationStore RelationStore, teamStore TeamStore) *FileAgentStore {
	return &FileAgentStore{
		storeDir:      storeDir,
		relationStore: relationStore,
		teamStore:     teamStore,
	}
}

// agentsDir returns the path to the agents directory
func (s *FileAgentStore) agentsDir() string {
	return filepath.Join(s.storeDir, "agents")
}

// List returns all agents
func (s *FileAgentStore) List(ctx context.Context) ([]Agent, error) {
	agentDirs, err := ListDirectories(s.agentsDir())
	if err != nil {
		return nil, fmt.Errorf("listing agent directories: %w", err)
	}

	var agents []Agent
	for _, agentID := range agentDirs {
		agent, err := s.loadAgent(agentID)
		if err != nil {
			continue // Skip malformed agents
		}
		agents = append(agents, *agent)
	}

	return agents, nil
}

// Get retrieves an agent by ID
func (s *FileAgentStore) Get(ctx context.Context, id string) (*Agent, error) {
	return s.loadAgent(id)
}

// Create creates a new agent
func (s *FileAgentStore) Create(ctx context.Context, agent *Agent) error {
	// Check agent doesn't already exist
	if _, err := s.Get(ctx, agent.ID); err == nil {
		return fmt.Errorf("agent already exists: %s", agent.ID)
	}

	// Set up agent entity
	agent.Kind = KindAgent
	agent.SchemaVersion = CurrentSchemaVersion
	if agent.Status == "" {
		agent.Status = AgentStatusActive
	}
	agent.Timestamps = NewTimestamps()

	agentDir := filepath.Join(s.agentsDir(), agent.ID)

	// Create agent directory
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return fmt.Errorf("creating agent directory: %w", err)
	}

	// Write agent.json
	if err := SaveJSON(filepath.Join(agentDir, "agent.json"), agent); err != nil {
		return fmt.Errorf("writing agent.json: %w", err)
	}

	return nil
}

// Update updates an existing agent
func (s *FileAgentStore) Update(ctx context.Context, id string, updates *Agent) error {
	agent, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	// Apply updates
	if updates.DisplayName != "" {
		agent.DisplayName = updates.DisplayName
	}
	if updates.Status != "" {
		agent.Status = updates.Status
	}
	if updates.Appearance != nil {
		agent.Appearance = updates.Appearance
	}
	if updates.SkillPins != nil {
		agent.SkillPins = updates.SkillPins
	}
	if updates.Runtime != nil {
		agent.Runtime = updates.Runtime
	}

	agent.UpdateTimestamp()

	// Write updated agent.json
	agentPath := filepath.Join(s.agentsDir(), id, "agent.json")
	return SaveJSON(agentPath, agent)
}

// Delete removes an agent
func (s *FileAgentStore) Delete(ctx context.Context, id string) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}

	agentDir := filepath.Join(s.agentsDir(), id)
	return DeleteDirectory(agentDir)
}

// GetSkills returns skills assigned to an agent (through relations)
func (s *FileAgentStore) GetSkills(ctx context.Context, agentID string) ([]AgentSkillRelation, error) {
	if s.relationStore == nil {
		return nil, fmt.Errorf("relation store not configured")
	}
	return s.relationStore.ListAgentSkills(ctx, agentID)
}

// GetEffectiveSkills computes the effective skill set for an agent in a team context
//
// Resolution order:
// 1. Agent explicit skillPins
// 2. Agent-skill relations with enabled=true
// 3. Team role default skill grants from team.defaults.skillGrantsByRole
// 4. Subtract disabled relations
func (s *FileAgentStore) GetEffectiveSkills(ctx context.Context, agentID string, teamID *string) ([]string, error) {
	agent, err := s.Get(ctx, agentID)
	if err != nil {
		return nil, err
	}

	skillSet := make(map[string]bool)

	// 1. Add agent's explicit skillPins
	for _, pin := range agent.SkillPins {
		skillSet[pin.SkillID] = true
	}

	// 2. Add agent-skill relations
	if s.relationStore != nil {
		relations, err := s.relationStore.ListAgentSkills(ctx, agentID)
		if err == nil {
			for _, rel := range relations {
				if rel.Enabled {
					skillSet[rel.SkillID] = true
				} else {
					delete(skillSet, rel.SkillID)
				}
			}
		}
	}

	// 3. Add team role default grants if team context provided
	if teamID != nil && s.teamStore != nil && s.relationStore != nil {
		team, err := s.teamStore.Get(ctx, *teamID)
		if err == nil && team.Defaults != nil && team.Defaults.SkillGrantsByRole != nil {
			// Get agent's membership in the team
			membership, err := s.relationStore.GetTeamMember(ctx, *teamID, agentID)
			if err == nil {
				for _, role := range membership.Roles {
					if grants, ok := team.Defaults.SkillGrantsByRole[role]; ok {
						for _, skillID := range grants {
							skillSet[skillID] = true
						}
					}
				}
			}
		}
	}

	// Convert to slice
	var skills []string
	for skillID := range skillSet {
		skills = append(skills, skillID)
	}

	return skills, nil
}

// loadAgent loads an agent from the agents directory
func (s *FileAgentStore) loadAgent(agentID string) (*Agent, error) {
	agentPath := filepath.Join(s.agentsDir(), agentID, "agent.json")
	return LoadJSON[Agent](agentPath)
}

// GetSoul reads the SOUL.md content for an agent
func (s *FileAgentStore) GetSoul(ctx context.Context, agentID string) (string, error) {
	// Verify agent exists
	if _, err := s.Get(ctx, agentID); err != nil {
		return "", err
	}

	soulPath := filepath.Join(s.agentsDir(), agentID, "SOUL.md")
	if !FileExists(soulPath) {
		return "", nil // Return empty string if no SOUL.md exists
	}

	return ReadContent(soulPath)
}

// SetSoul writes the SOUL.md content for an agent
func (s *FileAgentStore) SetSoul(ctx context.Context, agentID string, content string) error {
	// Verify agent exists
	if _, err := s.Get(ctx, agentID); err != nil {
		return err
	}

	soulPath := filepath.Join(s.agentsDir(), agentID, "SOUL.md")
	return WriteContent(soulPath, content)
}

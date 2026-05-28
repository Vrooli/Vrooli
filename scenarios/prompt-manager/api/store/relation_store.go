package store

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// FileRelationStore implements RelationStore using the file system
type FileRelationStore struct {
	configDir string
}

// NewFileRelationStore creates a new file-based relation store
func NewFileRelationStore(configDir string) *FileRelationStore {
	return &FileRelationStore{configDir: configDir}
}

// relationsDir returns the path to the relations directory
func (s *FileRelationStore) relationsDir() string {
	return filepath.Join(s.configDir, "relations")
}

// teamMemberDir returns the path to team-member relations
func (s *FileRelationStore) teamMemberDir() string {
	return filepath.Join(s.relationsDir(), "team-member")
}

// TeamMember operations

// GetTeamMember retrieves a team-member relation
func (s *FileRelationStore) GetTeamMember(ctx context.Context, teamID, agentID string) (*TeamMemberRelation, error) {
	filename := fmt.Sprintf("%s__%s.json", teamID, agentID)
	path := filepath.Join(s.teamMemberDir(), filename)
	return LoadJSON[TeamMemberRelation](path)
}

// SetTeamMember creates or updates a team-member relation
func (s *FileRelationStore) SetTeamMember(ctx context.Context, rel *TeamMemberRelation) error {
	rel.Kind = KindTeamMember
	rel.SchemaVersion = CurrentSchemaVersion
	if rel.Status == "" {
		rel.Status = MemberStatusActive
	}

	filename := fmt.Sprintf("%s__%s.json", rel.TeamID, rel.AgentID)
	path := filepath.Join(s.teamMemberDir(), filename)
	return SaveJSON(path, rel)
}

// DeleteTeamMember removes a team-member relation
func (s *FileRelationStore) DeleteTeamMember(ctx context.Context, teamID, agentID string) error {
	filename := fmt.Sprintf("%s__%s.json", teamID, agentID)
	path := filepath.Join(s.teamMemberDir(), filename)
	return DeleteFile(path)
}

// ListTeamMembers returns all members of a team
func (s *FileRelationStore) ListTeamMembers(ctx context.Context, teamID string) ([]TeamMemberRelation, error) {
	files, err := ListFiles(s.teamMemberDir(), fmt.Sprintf("%s__*.json", teamID))
	if err != nil {
		return nil, fmt.Errorf("listing team members: %w", err)
	}

	var relations []TeamMemberRelation
	for _, file := range files {
		rel, err := LoadJSON[TeamMemberRelation](file)
		if err != nil {
			continue // Skip malformed relations
		}
		relations = append(relations, *rel)
	}

	return relations, nil
}

// ListAllTeamMembers returns all team-member relations
func (s *FileRelationStore) ListAllTeamMembers(ctx context.Context) ([]TeamMemberRelation, error) {
	files, err := ListFiles(s.teamMemberDir(), "*.json")
	if err != nil {
		return nil, fmt.Errorf("listing all team members: %w", err)
	}

	var relations []TeamMemberRelation
	for _, file := range files {
		rel, err := LoadJSON[TeamMemberRelation](file)
		if err != nil {
			continue // Skip malformed relations
		}
		relations = append(relations, *rel)
	}

	return relations, nil
}

// ListAgentTeams returns all teams an agent belongs to
func (s *FileRelationStore) ListAgentTeams(ctx context.Context, agentID string) ([]TeamMemberRelation, error) {
	files, err := ListFiles(s.teamMemberDir(), "*.json")
	if err != nil {
		return nil, fmt.Errorf("listing agent teams: %w", err)
	}

	var relations []TeamMemberRelation
	for _, file := range files {
		// Check if this relation involves the agent
		filename := filepath.Base(file)
		if !strings.Contains(filename, "__"+agentID+".json") {
			continue
		}

		rel, err := LoadJSON[TeamMemberRelation](file)
		if err != nil {
			continue
		}
		if rel.AgentID == agentID {
			relations = append(relations, *rel)
		}
	}

	return relations, nil
}

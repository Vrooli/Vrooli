package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// FileTeamStore implements TeamStore using the file system
type FileTeamStore struct {
	storeDir      string
	relationStore RelationStore
}

// NewFileTeamStore creates a new file-based team store
func NewFileTeamStore(storeDir string, relationStore RelationStore) *FileTeamStore {
	return &FileTeamStore{
		storeDir:      storeDir,
		relationStore: relationStore,
	}
}

// teamsDir returns the path to the teams directory
func (s *FileTeamStore) teamsDir() string {
	return filepath.Join(s.storeDir, "teams")
}

// List returns all teams
func (s *FileTeamStore) List(ctx context.Context) ([]Team, error) {
	teamDirs, err := ListDirectories(s.teamsDir())
	if err != nil {
		return nil, fmt.Errorf("listing team directories: %w", err)
	}

	var teams []Team
	for _, teamID := range teamDirs {
		team, err := s.loadTeam(teamID)
		if err != nil {
			continue // Skip malformed teams
		}
		teams = append(teams, *team)
	}

	return teams, nil
}

// Get retrieves a team by ID
func (s *FileTeamStore) Get(ctx context.Context, id string) (*Team, error) {
	return s.loadTeam(id)
}

// Create creates a new team
func (s *FileTeamStore) Create(ctx context.Context, team *Team) error {
	// Check team doesn't already exist
	if _, err := s.Get(ctx, team.ID); err == nil {
		return fmt.Errorf("team already exists: %s", team.ID)
	}

	// Set up team entity
	team.Kind = KindTeam
	team.SchemaVersion = CurrentSchemaVersion
	team.Timestamps = NewTimestamps()

	teamDir := filepath.Join(s.teamsDir(), team.ID)

	// Create team directory and shared folder
	if err := os.MkdirAll(filepath.Join(teamDir, "shared"), 0o755); err != nil {
		return fmt.Errorf("creating team directory: %w", err)
	}

	// Write team.json
	if err := SaveJSON(filepath.Join(teamDir, "team.json"), team); err != nil {
		return fmt.Errorf("writing team.json: %w", err)
	}

	// Create default roles.json
	roles := &TeamRoles{
		BaseEntity: BaseEntity{
			Kind:          KindTeamRoles,
			SchemaVersion: CurrentSchemaVersion,
		},
		TeamID: team.ID,
		Roles:  []Role{},
	}
	if err := SaveJSON(filepath.Join(teamDir, "roles.json"), roles); err != nil {
		return fmt.Errorf("writing roles.json: %w", err)
	}

	// Create default org.json
	org := &OrgChart{
		BaseEntity: BaseEntity{
			Kind:          KindOrgChart,
			SchemaVersion: CurrentSchemaVersion,
		},
		TeamID: team.ID,
		Edges:  []OrgEdge{},
	}
	if err := SaveJSON(filepath.Join(teamDir, "org.json"), org); err != nil {
		return fmt.Errorf("writing org.json: %w", err)
	}

	return nil
}

// Update updates an existing team
func (s *FileTeamStore) Update(ctx context.Context, id string, updates *Team) error {
	team, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	// Apply updates
	if updates.DisplayName != "" {
		team.DisplayName = updates.DisplayName
	}
	if updates.Mission != "" {
		team.Mission = updates.Mission
	}
	if updates.Shared != nil {
		team.Shared = updates.Shared
	}
	if updates.Defaults != nil {
		team.Defaults = updates.Defaults
	}

	team.UpdateTimestamp()

	// Write updated team.json
	teamPath := filepath.Join(s.teamsDir(), id, "team.json")
	return SaveJSON(teamPath, team)
}

// Delete removes a team
func (s *FileTeamStore) Delete(ctx context.Context, id string) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}

	teamDir := filepath.Join(s.teamsDir(), id)
	return DeleteDirectory(teamDir)
}

// GetRoles returns role definitions for a team
func (s *FileTeamStore) GetRoles(ctx context.Context, teamID string) (*TeamRoles, error) {
	rolesPath := filepath.Join(s.teamsDir(), teamID, "roles.json")
	return LoadJSON[TeamRoles](rolesPath)
}

// GetOrgChart returns the org chart for a team
func (s *FileTeamStore) GetOrgChart(ctx context.Context, teamID string) (*OrgChart, error) {
	orgPath := filepath.Join(s.teamsDir(), teamID, "org.json")
	return LoadJSON[OrgChart](orgPath)
}

// GetMembers returns all members of a team
func (s *FileTeamStore) GetMembers(ctx context.Context, teamID string) ([]TeamMemberRelation, error) {
	if s.relationStore == nil {
		return nil, fmt.Errorf("relation store not configured")
	}
	return s.relationStore.ListTeamMembers(ctx, teamID)
}

// SetRoles sets the role definitions for a team
func (s *FileTeamStore) SetRoles(ctx context.Context, teamID string, roles *TeamRoles) error {
	// Verify team exists
	if _, err := s.Get(ctx, teamID); err != nil {
		return err
	}

	roles.Kind = KindTeamRoles
	roles.SchemaVersion = CurrentSchemaVersion
	roles.TeamID = teamID

	rolesPath := filepath.Join(s.teamsDir(), teamID, "roles.json")
	return SaveJSON(rolesPath, roles)
}

// SetOrgChart sets the org chart for a team
func (s *FileTeamStore) SetOrgChart(ctx context.Context, teamID string, org *OrgChart) error {
	// Verify team exists
	if _, err := s.Get(ctx, teamID); err != nil {
		return err
	}

	org.Kind = KindOrgChart
	org.SchemaVersion = CurrentSchemaVersion
	org.TeamID = teamID

	orgPath := filepath.Join(s.teamsDir(), teamID, "org.json")
	return SaveJSON(orgPath, org)
}

// loadTeam loads a team from the teams directory
func (s *FileTeamStore) loadTeam(teamID string) (*Team, error) {
	teamPath := filepath.Join(s.teamsDir(), teamID, "team.json")
	return LoadJSON[Team](teamPath)
}

// memberDir returns the path to a member's directory within a team
func (s *FileTeamStore) memberDir(teamID, agentID string) string {
	return filepath.Join(s.teamsDir(), teamID, "members", agentID)
}

// EnsureMemberDir creates the member directory structure if it doesn't exist
func (s *FileTeamStore) EnsureMemberDir(ctx context.Context, teamID, agentID string) error {
	// Verify team exists
	if _, err := s.Get(ctx, teamID); err != nil {
		return err
	}

	memberDir := s.memberDir(teamID, agentID)
	logsDir := filepath.Join(memberDir, "logs")

	// Create directories
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return fmt.Errorf("creating member directories: %w", err)
	}

	return nil
}

// GetResponsibilities reads the RESPONSIBILITIES.md content for a team member
func (s *FileTeamStore) GetResponsibilities(ctx context.Context, teamID, agentID string) (string, error) {
	// Verify team exists
	if _, err := s.Get(ctx, teamID); err != nil {
		return "", err
	}

	respPath := filepath.Join(s.memberDir(teamID, agentID), "RESPONSIBILITIES.md")
	if !FileExists(respPath) {
		return "", nil
	}

	return ReadContent(respPath)
}

// SetResponsibilities writes the RESPONSIBILITIES.md content for a team member
func (s *FileTeamStore) SetResponsibilities(ctx context.Context, teamID, agentID, content string) error {
	// Ensure member directory exists
	if err := s.EnsureMemberDir(ctx, teamID, agentID); err != nil {
		return err
	}

	respPath := filepath.Join(s.memberDir(teamID, agentID), "RESPONSIBILITIES.md")
	return WriteContent(respPath, content)
}

// GetHeartbeatInstructions reads the HEARTBEAT.md content for a team member
func (s *FileTeamStore) GetHeartbeatInstructions(ctx context.Context, teamID, agentID string) (string, error) {
	// Verify team exists
	if _, err := s.Get(ctx, teamID); err != nil {
		return "", err
	}

	hbPath := filepath.Join(s.memberDir(teamID, agentID), "HEARTBEAT.md")
	if !FileExists(hbPath) {
		return "", nil
	}

	return ReadContent(hbPath)
}

// SetHeartbeatInstructions writes the HEARTBEAT.md content for a team member
func (s *FileTeamStore) SetHeartbeatInstructions(ctx context.Context, teamID, agentID, content string) error {
	// Ensure member directory exists
	if err := s.EnsureMemberDir(ctx, teamID, agentID); err != nil {
		return err
	}

	hbPath := filepath.Join(s.memberDir(teamID, agentID), "HEARTBEAT.md")
	return WriteContent(hbPath, content)
}

// GetHeartbeatConfig reads the heartbeat.json config for a team member
func (s *FileTeamStore) GetHeartbeatConfig(ctx context.Context, teamID, agentID string) (*HeartbeatConfig, error) {
	// Verify team exists
	if _, err := s.Get(ctx, teamID); err != nil {
		return nil, err
	}

	configPath := filepath.Join(s.memberDir(teamID, agentID), "heartbeat.json")
	if !FileExists(configPath) {
		return nil, nil
	}

	return LoadJSON[HeartbeatConfig](configPath)
}

// SetHeartbeatConfig writes the heartbeat.json config for a team member
func (s *FileTeamStore) SetHeartbeatConfig(ctx context.Context, teamID, agentID string, config *HeartbeatConfig) error {
	// Ensure member directory exists
	if err := s.EnsureMemberDir(ctx, teamID, agentID); err != nil {
		return err
	}

	// Set entity metadata
	config.Kind = KindHeartbeatConfig
	config.SchemaVersion = CurrentSchemaVersion
	config.TeamID = teamID
	config.AgentID = agentID

	if config.CreatedAt == "" {
		config.Timestamps = NewTimestamps()
	} else {
		config.UpdateTimestamp()
	}

	configPath := filepath.Join(s.memberDir(teamID, agentID), "heartbeat.json")
	return SaveJSON(configPath, config)
}

// DeleteHeartbeatConfig removes the heartbeat.json config for a team member
func (s *FileTeamStore) DeleteHeartbeatConfig(ctx context.Context, teamID, agentID string) error {
	configPath := filepath.Join(s.memberDir(teamID, agentID), "heartbeat.json")
	return DeleteFile(configPath)
}

// ListHeartbeatConfigs lists all heartbeat configs for a team
func (s *FileTeamStore) ListHeartbeatConfigs(ctx context.Context, teamID string) ([]HeartbeatConfig, error) {
	// Verify team exists
	if _, err := s.Get(ctx, teamID); err != nil {
		return nil, err
	}

	membersDir := filepath.Join(s.teamsDir(), teamID, "members")
	agentDirs, err := ListDirectories(membersDir)
	if err != nil {
		return nil, nil // No members directory yet
	}

	var configs []HeartbeatConfig
	for _, agentID := range agentDirs {
		config, err := s.GetHeartbeatConfig(ctx, teamID, agentID)
		if err == nil && config != nil {
			configs = append(configs, *config)
		}
	}

	return configs, nil
}

// GetMemberLogPath returns the path for a heartbeat execution log
func (s *FileTeamStore) GetMemberLogPath(teamID, agentID, timestamp string) string {
	return filepath.Join(s.memberDir(teamID, agentID), "logs", timestamp+".log")
}

// ListMemberLogs lists all log files for a team member
func (s *FileTeamStore) ListMemberLogs(ctx context.Context, teamID, agentID string) ([]string, error) {
	logsDir := filepath.Join(s.memberDir(teamID, agentID), "logs")
	files, err := ListFiles(logsDir, "*.log")
	if err != nil {
		return nil, nil // No logs yet
	}

	// Extract just the filenames
	var logs []string
	for _, f := range files {
		logs = append(logs, filepath.Base(f))
	}

	return logs, nil
}

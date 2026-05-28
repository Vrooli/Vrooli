package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"prompt-manager/teamconfig"
	"prompt-manager/teamcontract"
	"sort"
	"strings"
	"time"
)

// FileTeamStore implements TeamStore using the file system.
//
// Team data is split across two storage class roots:
//
//   - configRoot (Config class) holds authored, git-tracked content:
//     teams/<id>/{team,roles,org}.json, members/<a>/{RESPONSIBILITIES.md,
//     HEARTBEAT.md, topics.json}, and shared/TEAM.md.
//   - runtimeDataRoot (RuntimeData class) holds runtime execution state:
//     members/<a>/{heartbeat.json, last-handoff.md, inbox.json, logs/} and
//     shared/{tasks.json, decisions.jsonl, handoff-history.jsonl,
//     heartbeat-attempts.jsonl, knowledge.jsonl}.
//
// Path shapes under each root mirror the original single-tree layout (CD-2);
// only the prefix differs. The operator-facing shared/ tree is virtually
// merged by ListSharedFiles and single-file ops route by filename class.
type FileTeamStore struct {
	configRoot      string
	runtimeDataRoot string
	relationStore   RelationStore
}

// NewFileTeamStore creates a new file-based team store rooted at the given
// Config and RuntimeData class directories. Production callers pass
// roots.Config and roots.RuntimeData; tests that don't care about the split
// can pass the same tempdir for both.
func NewFileTeamStore(configRoot, runtimeDataRoot string, relationStore RelationStore) *FileTeamStore {
	return &FileTeamStore{
		configRoot:      configRoot,
		runtimeDataRoot: runtimeDataRoot,
		relationStore:   relationStore,
	}
}

// configTeamsDir returns the Config-class teams directory.
func (s *FileTeamStore) configTeamsDir() string {
	return filepath.Join(s.configRoot, "teams")
}

// runtimeTeamsDir returns the RuntimeData-class teams directory.
func (s *FileTeamStore) runtimeTeamsDir() string {
	return filepath.Join(s.runtimeDataRoot, "teams")
}

// teamsDir is kept as an alias for configTeamsDir — used by methods that
// only touch config-class files (team.json, roles.json, org.json).
func (s *FileTeamStore) teamsDir() string {
	return s.configTeamsDir()
}

// StoreDir returns the Config-class root. Callers (prompt_builder, topic
// contract loader, agent file loader) consume this to resolve authored
// content; runtime state never flows through this accessor.
func (s *FileTeamStore) StoreDir() string {
	return s.configRoot
}

// TeamFileEntry represents a file or directory within a team's shared folder.
type TeamFileEntry struct {
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
	Size  int64  `json:"size,omitempty"`
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
	if err := teamconfig.Validate(team.Contract()); err != nil {
		return err
	}
	if team.OperatingContract != nil {
		team.OperatingContract.Governance.DecisionMode = team.DecisionMode
	}
	if err := s.validateOperatingContract(context.Background(), team); err != nil {
		return err
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
	if updates.EnabledSet {
		team.Enabled = updates.Enabled
	}
	if updates.Runtime.Mode != "" {
		team.Runtime = updates.Runtime
	}
	if updates.Coordination.Pattern != "" {
		team.Coordination = updates.Coordination
	}
	if updates.Execution.QueuePolicy != "" {
		team.Execution = updates.Execution
	}
	if updates.DecisionMode != "" {
		team.DecisionMode = updates.DecisionMode
	}
	if updates.OperatingContract != nil {
		team.OperatingContract = updates.OperatingContract
	}
	if updates.DecisionMode != "" && team.OperatingContract != nil {
		team.OperatingContract.Governance.DecisionMode = updates.DecisionMode
	}
	if updates.Shared != nil {
		team.Shared = updates.Shared
	}
	if updates.Retention != nil {
		team.Retention = updates.Retention
	}
	if err := teamconfig.Validate(team.Contract()); err != nil {
		return err
	}
	if err := s.validateOperatingContract(ctx, team); err != nil {
		return err
	}

	team.UpdateTimestamp()

	// Write updated team.json
	teamPath := filepath.Join(s.teamsDir(), id, "team.json")
	return SaveJSON(teamPath, team)
}

// Delete removes a team from both Config and RuntimeData roots.
func (s *FileTeamStore) Delete(ctx context.Context, id string) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}

	configTeamDir := filepath.Join(s.configTeamsDir(), id)
	if err := DeleteDirectory(configTeamDir); err != nil {
		return err
	}
	runtimeTeamDir := filepath.Join(s.runtimeTeamsDir(), id)
	if _, err := os.Stat(runtimeTeamDir); err == nil {
		if err := DeleteDirectory(runtimeTeamDir); err != nil {
			return err
		}
	}
	return nil
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

// GetInbox returns the inbox for a team member.
func (s *FileTeamStore) GetInbox(ctx context.Context, teamID, agentID string) (*TeamInbox, error) {
	// Verify team exists
	if _, err := s.Get(ctx, teamID); err != nil {
		return nil, err
	}

	inboxPath := filepath.Join(s.runtimeMemberDir(teamID, agentID), "inbox.json")
	if !FileExists(inboxPath) {
		return &TeamInbox{
			BaseEntity: BaseEntity{
				Kind:          KindTeamInbox,
				SchemaVersion: CurrentSchemaVersion,
			},
			TeamID:   teamID,
			AgentID:  agentID,
			Messages: []TeamMessage{},
		}, nil
	}

	return LoadJSON[TeamInbox](inboxPath)
}

// SetInbox stores the inbox for a team member.
func (s *FileTeamStore) SetInbox(ctx context.Context, teamID, agentID string, inbox *TeamInbox) error {
	if err := s.EnsureMemberDir(ctx, teamID, agentID); err != nil {
		return err
	}

	inbox.Kind = KindTeamInbox
	inbox.SchemaVersion = CurrentSchemaVersion
	inbox.TeamID = teamID
	inbox.AgentID = agentID
	if inbox.Messages == nil {
		inbox.Messages = []TeamMessage{}
	}

	inboxPath := filepath.Join(s.runtimeMemberDir(teamID, agentID), "inbox.json")
	return SaveJSON(inboxPath, inbox)
}

// loadTeam loads a team from the teams directory
func (s *FileTeamStore) loadTeam(teamID string) (*Team, error) {
	teamPath := filepath.Join(s.teamsDir(), teamID, "team.json")
	team, err := LoadJSON[Team](teamPath)
	if err != nil {
		return nil, err
	}
	if err := teamconfig.Validate(team.Contract()); err != nil {
		return nil, err
	}
	if err := s.validateOperatingContract(context.Background(), team); err != nil {
		return nil, err
	}
	return team, nil
}

func (s *FileTeamStore) validateOperatingContract(ctx context.Context, team *Team) error {
	if team == nil {
		return fmt.Errorf("team is required")
	}
	var memberIDs []string
	if s.relationStore != nil {
		members, err := s.relationStore.ListTeamMembers(ctx, team.ID)
		if err == nil {
			for _, member := range members {
				if member.Status == "" || member.Status == MemberStatusActive {
					memberIDs = append(memberIDs, member.AgentID)
				}
			}
		}
	}
	return teamcontract.Validate(team.OperatingContract, teamcontract.ValidationInput{
		TeamID:       team.ID,
		DecisionMode: team.DecisionMode,
		MemberIDs:    memberIDs,
		StoreDir:     s.configRoot,
	})
}

// configMemberDir returns the Config-class member directory (holds
// RESPONSIBILITIES.md, HEARTBEAT.md, topics.json).
func (s *FileTeamStore) configMemberDir(teamID, agentID string) string {
	return filepath.Join(s.configTeamsDir(), teamID, "members", agentID)
}

// runtimeMemberDir returns the RuntimeData-class member directory (holds
// heartbeat.json, last-handoff.md, inbox.json, logs/).
func (s *FileTeamStore) runtimeMemberDir(teamID, agentID string) string {
	return filepath.Join(s.runtimeTeamsDir(), teamID, "members", agentID)
}

// DeleteMemberData removes all stored files for a team member (heartbeat
// config, inbox, logs, docs) from both Config and RuntimeData roots.
func (s *FileTeamStore) DeleteMemberData(ctx context.Context, teamID, agentID string) error {
	if _, err := s.Get(ctx, teamID); err != nil {
		return err
	}

	for _, dir := range []string{s.configMemberDir(teamID, agentID), s.runtimeMemberDir(teamID, agentID)} {
		if _, err := os.Stat(dir); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("stat member directory: %w", err)
		}
		if err := DeleteDirectory(dir); err != nil {
			return err
		}
	}
	return nil
}

// EnsureMemberDir creates the RuntimeData member directory structure
// (logs/ subdir) so subsequent runtime writes succeed. The Config member
// directory is owned by the authored repo layout and is not created here.
func (s *FileTeamStore) EnsureMemberDir(ctx context.Context, teamID, agentID string) error {
	if _, err := s.Get(ctx, teamID); err != nil {
		return err
	}

	logsDir := filepath.Join(s.runtimeMemberDir(teamID, agentID), "logs")
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

	respPath := filepath.Join(s.configMemberDir(teamID, agentID), "RESPONSIBILITIES.md")
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

	respPath := filepath.Join(s.configMemberDir(teamID, agentID), "RESPONSIBILITIES.md")
	return WriteContent(respPath, content)
}

// GetHeartbeatInstructions reads the HEARTBEAT.md content for a team member
func (s *FileTeamStore) GetHeartbeatInstructions(ctx context.Context, teamID, agentID string) (string, error) {
	// Verify team exists
	if _, err := s.Get(ctx, teamID); err != nil {
		return "", err
	}

	hbPath := filepath.Join(s.configMemberDir(teamID, agentID), "HEARTBEAT.md")
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

	hbPath := filepath.Join(s.configMemberDir(teamID, agentID), "HEARTBEAT.md")
	return WriteContent(hbPath, content)
}

// GetHeartbeatConfig reads the heartbeat.json config for a team member
func (s *FileTeamStore) GetHeartbeatConfig(ctx context.Context, teamID, agentID string) (*HeartbeatConfig, error) {
	// Verify team exists
	if _, err := s.Get(ctx, teamID); err != nil {
		return nil, err
	}

	configPath := filepath.Join(s.runtimeMemberDir(teamID, agentID), "heartbeat.json")
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

	configPath := filepath.Join(s.runtimeMemberDir(teamID, agentID), "heartbeat.json")
	return SaveJSON(configPath, config)
}

// DeleteHeartbeatConfig removes the heartbeat.json config for a team member
func (s *FileTeamStore) DeleteHeartbeatConfig(ctx context.Context, teamID, agentID string) error {
	configPath := filepath.Join(s.runtimeMemberDir(teamID, agentID), "heartbeat.json")
	return DeleteFile(configPath)
}

// ListHeartbeatConfigs lists all heartbeat configs for a team
func (s *FileTeamStore) ListHeartbeatConfigs(ctx context.Context, teamID string) ([]HeartbeatConfig, error) {
	// Verify team exists
	if _, err := s.Get(ctx, teamID); err != nil {
		return nil, err
	}

	// Members exist in either root: Config holds the authored
	// member subtree (RESPONSIBILITIES.md, topics.json, …) and
	// RuntimeData holds the heartbeat.json itself. Union both so a
	// purely-runtime member (test fixtures) and a fully-authored
	// member (production) both show up.
	seen := map[string]struct{}{}
	for _, root := range []string{
		filepath.Join(s.configTeamsDir(), teamID, "members"),
		filepath.Join(s.runtimeTeamsDir(), teamID, "members"),
	} {
		dirs, _ := ListDirectories(root)
		for _, d := range dirs {
			seen[d] = struct{}{}
		}
	}

	var configs []HeartbeatConfig
	for agentID := range seen {
		config, err := s.GetHeartbeatConfig(ctx, teamID, agentID)
		if err == nil && config != nil {
			configs = append(configs, *config)
		}
	}

	return configs, nil
}

// AppendHeartbeatAttempt appends a durable heartbeat dispatch attempt record.
func (s *FileTeamStore) AppendHeartbeatAttempt(_ context.Context, teamID string, entry *HeartbeatAttempt) error {
	sharedDir := filepath.Join(s.runtimeTeamsDir(), teamID, "shared")
	if err := os.MkdirAll(sharedDir, 0o755); err != nil {
		return fmt.Errorf("creating shared directory: %w", err)
	}
	return AppendJSONL(filepath.Join(sharedDir, "heartbeat-attempts.jsonl"), entry)
}

// ListHeartbeatAttempts returns newest-first heartbeat dispatch attempts across
// all teams, optionally filtered by team, agent, status, and profile key.
func (s *FileTeamStore) ListHeartbeatAttempts(_ context.Context, teamID, agentID, status, profileKey string, limit, offset int) ([]HeartbeatAttempt, int, error) {
	teamIDs := []string{}
	if teamID != "" {
		teamIDs = append(teamIDs, teamID)
	} else {
		dirs, err := ListDirectories(s.teamsDir())
		if err != nil {
			return nil, 0, fmt.Errorf("listing team directories: %w", err)
		}
		teamIDs = dirs
	}

	var entries []HeartbeatAttempt
	for _, id := range teamIDs {
		path := filepath.Join(s.runtimeTeamsDir(), id, "shared", "heartbeat-attempts.jsonl")
		if !FileExists(path) {
		} else {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, 0, fmt.Errorf("reading heartbeat attempts: %w", err)
			}
			for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				var entry HeartbeatAttempt
				if err := json.Unmarshal([]byte(line), &entry); err != nil {
					continue
				}
				if agentID != "" && entry.AgentID != agentID {
					continue
				}
				if status != "" && entry.Status != status {
					continue
				}
				if profileKey != "" && entry.ProfileKey != profileKey {
					continue
				}
				entries = append(entries, entry)
			}
		}

		configs, _ := s.ListHeartbeatConfigs(context.Background(), id)
		for _, config := range configs {
			if config.LastExecution == nil || config.LastExecution.RunID != "" {
				continue
			}
			entry := HeartbeatAttempt{
				ID:            "last-" + config.TeamID + "-" + config.AgentID + "-" + strings.ReplaceAll(config.LastExecution.StartedAt, ":", "-"),
				TeamID:        config.TeamID,
				AgentID:       config.AgentID,
				ProfileKey:    config.ProfileKey,
				Status:        config.LastExecution.Status,
				Phase:         "pre_run_failure",
				StartedAt:     config.LastExecution.StartedAt,
				EndedAt:       config.LastExecution.EndedAt,
				ErrorCategory: "unknown",
				Error:         config.LastExecution.Error,
				Recovery:      "inspect_heartbeat_config",
			}
			if agentID != "" && entry.AgentID != agentID {
				continue
			}
			if status != "" && entry.Status != status {
				continue
			}
			if profileKey != "" && entry.ProfileKey != profileKey {
				continue
			}
			duplicate := false
			for _, existing := range entries {
				if existing.TeamID == entry.TeamID && existing.AgentID == entry.AgentID && existing.StartedAt == entry.StartedAt && existing.RunID == "" {
					duplicate = true
					break
				}
			}
			if !duplicate {
				entries = append(entries, entry)
			}
		}
	}

	sort.SliceStable(entries, func(i, j int) bool {
		left := entries[i].StartedAt
		if entries[i].EndedAt != "" {
			left = entries[i].EndedAt
		}
		right := entries[j].StartedAt
		if entries[j].EndedAt != "" {
			right = entries[j].EndedAt
		}
		return left > right
	})

	total := len(entries)
	if offset < 0 {
		offset = 0
	}
	if limit < 0 {
		limit = 0
	}
	if offset >= len(entries) {
		return []HeartbeatAttempt{}, total, nil
	}
	if limit > 0 && offset+limit < len(entries) {
		entries = entries[offset : offset+limit]
	} else {
		entries = entries[offset:]
	}
	return entries, total, nil
}

// GetMemberLogPath returns the path for a heartbeat execution log
func (s *FileTeamStore) GetMemberLogPath(teamID, agentID, timestamp string) string {
	return filepath.Join(s.runtimeMemberDir(teamID, agentID), "logs", timestamp+".log")
}

// ListMemberLogs lists all log files for a team member
func (s *FileTeamStore) ListMemberLogs(ctx context.Context, teamID, agentID string) ([]string, error) {
	logsDir := filepath.Join(s.runtimeMemberDir(teamID, agentID), "logs")
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

// ListSharedFiles returns all files and directories under a team's shared
// folder by walking both class roots and merging on relative path. Runtime
// entries override config on the (rare) collision; the merge is alphabetical
// by relative path.
func (s *FileTeamStore) ListSharedFiles(ctx context.Context, teamID string) ([]TeamFileEntry, error) {
	team, err := s.Get(ctx, teamID)
	if err != nil {
		return nil, err
	}

	byPath := map[string]TeamFileEntry{}
	walkRoot := func(root string) error {
		if _, err := os.Stat(root); err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("stat shared directory: %w", err)
		}
		return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if path == root {
				return nil
			}
			if isHiddenFile(d.Name()) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			info, err := d.Info()
			size := int64(0)
			if err == nil {
				size = info.Size()
			}
			byPath[filepath.ToSlash(rel)] = TeamFileEntry{
				Path:  filepath.ToSlash(rel),
				IsDir: d.IsDir(),
				Size:  size,
			}
			return nil
		})
	}

	if err := walkRoot(s.configSharedDir(team)); err != nil {
		return nil, err
	}
	if err := walkRoot(s.runtimeSharedDir(team)); err != nil {
		return nil, err
	}

	entries := make([]TeamFileEntry, 0, len(byPath))
	for _, e := range byPath {
		entries = append(entries, e)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})

	return entries, nil
}

// ReadSharedFile returns file content for a path within the team's shared folder.
func (s *FileTeamStore) ReadSharedFile(ctx context.Context, teamID, relPath string) (string, error) {
	team, err := s.Get(ctx, teamID)
	if err != nil {
		return "", err
	}

	fullPath, _, err := s.resolveSharedPath(team, relPath)
	if err != nil {
		return "", err
	}

	return ReadContent(fullPath)
}

// WriteSharedFile overwrites file content within the team's shared folder.
func (s *FileTeamStore) WriteSharedFile(ctx context.Context, teamID, relPath, content string) error {
	team, err := s.Get(ctx, teamID)
	if err != nil {
		return err
	}

	fullPath, _, err := s.resolveSharedPath(team, relPath)
	if err != nil {
		return err
	}

	return WriteContent(fullPath, content)
}

// CreateSharedFile creates a new file or directory within the team's shared folder.
func (s *FileTeamStore) CreateSharedFile(ctx context.Context, teamID, relPath, content string, isDir bool) error {
	team, err := s.Get(ctx, teamID)
	if err != nil {
		return err
	}

	fullPath, _, err := s.resolveSharedPath(team, relPath)
	if err != nil {
		return err
	}
	if FileExists(fullPath) {
		return fmt.Errorf("file already exists: %s", relPath)
	}

	if isDir {
		return os.MkdirAll(fullPath, 0o755)
	}

	return WriteContent(fullPath, content)
}

// RenameSharedFile renames or moves a file within the team's shared folder.
func (s *FileTeamStore) RenameSharedFile(ctx context.Context, teamID, fromPath, toPath string) error {
	team, err := s.Get(ctx, teamID)
	if err != nil {
		return err
	}

	fromFull, _, err := s.resolveSharedPath(team, fromPath)
	if err != nil {
		return err
	}
	toFull, _, err := s.resolveSharedPath(team, toPath)
	if err != nil {
		return err
	}

	if !FileExists(fromFull) {
		return fmt.Errorf("file not found: %s", fromPath)
	}
	if FileExists(toFull) {
		return fmt.Errorf("target already exists: %s", toPath)
	}

	if err := os.MkdirAll(filepath.Dir(toFull), 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	return os.Rename(fromFull, toFull)
}

// DeleteSharedFile deletes a file or directory within the team's shared folder.
func (s *FileTeamStore) DeleteSharedFile(ctx context.Context, teamID, relPath string) error {
	team, err := s.Get(ctx, teamID)
	if err != nil {
		return err
	}

	fullPath, _, err := s.resolveSharedPath(team, relPath)
	if err != nil {
		return err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("stat %s: %w", relPath, err)
	}

	if info.IsDir() {
		return DeleteDirectory(fullPath)
	}
	return DeleteFile(fullPath)
}

// sharedSubpath returns the (cleaned, traversal-safe) relative subpath that
// the team's shared/ folder uses under each class root.
func (s *FileTeamStore) sharedSubpath(team *Team) string {
	sharedPath := "shared"
	if team.Shared != nil && strings.TrimSpace(team.Shared.Path) != "" {
		sharedPath = team.Shared.Path
	}

	clean := filepath.Clean(filepath.FromSlash(sharedPath))
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		clean = "shared"
	}
	return clean
}

// configSharedDir returns the Config-class shared directory (holds TEAM.md
// and any other operator-authored docs).
func (s *FileTeamStore) configSharedDir(team *Team) string {
	return filepath.Join(s.configTeamsDir(), team.ID, s.sharedSubpath(team))
}

// runtimeSharedDir returns the RuntimeData-class shared directory (holds
// tasks.json and the *.jsonl append-logs).
func (s *FileTeamStore) runtimeSharedDir(team *Team) string {
	return filepath.Join(s.runtimeTeamsDir(), team.ID, s.sharedSubpath(team))
}

// runtimeSharedBasenames is the set of file basenames that live under the
// RuntimeData shared/ root. Operator file ops use it to route a request to
// the correct class root.
var runtimeSharedBasenames = map[string]struct{}{
	"tasks.json":                {},
	"decisions.jsonl":           {},
	"handoff-history.jsonl":     {},
	"heartbeat-attempts.jsonl":  {},
	"knowledge.jsonl":           {},
}

// isRuntimeSharedRel reports whether the given (cleaned) relative path under
// shared/ targets a RuntimeData-class file. Currently driven by basename
// only — runtime files live at the top of shared/.
func isRuntimeSharedRel(cleanRel string) bool {
	_, ok := runtimeSharedBasenames[filepath.Base(cleanRel)]
	return ok
}

// resolveSharedPath validates relPath, picks the right class root for the
// file's basename, and returns (fullPath, cleanRel, error).
func (s *FileTeamStore) resolveSharedPath(team *Team, relPath string) (string, string, error) {
	if strings.TrimSpace(relPath) == "" {
		return "", "", fmt.Errorf("path is required")
	}

	clean := filepath.Clean(filepath.FromSlash(relPath))
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return "", "", fmt.Errorf("invalid path: %s", relPath)
	}

	var sharedDir string
	if isRuntimeSharedRel(clean) {
		sharedDir = s.runtimeSharedDir(team)
	} else {
		sharedDir = s.configSharedDir(team)
	}
	fullPath := filepath.Join(sharedDir, clean)

	rel, err := filepath.Rel(sharedDir, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", "", fmt.Errorf("invalid path: %s", relPath)
	}

	return fullPath, filepath.ToSlash(clean), nil
}

// --- Handoff methods ---

// GetLastHandoff reads the last handoff markdown for a team member.
func (s *FileTeamStore) GetLastHandoff(_ context.Context, teamID, agentID string) (string, error) {
	path := filepath.Join(s.runtimeMemberDir(teamID, agentID), "last-handoff.md")
	if !FileExists(path) {
		return "", nil
	}
	return ReadContent(path)
}

// SetLastHandoff writes the last handoff markdown for a team member.
func (s *FileTeamStore) SetLastHandoff(ctx context.Context, teamID, agentID, content string) error {
	if err := s.EnsureMemberDir(ctx, teamID, agentID); err != nil {
		return err
	}
	path := filepath.Join(s.runtimeMemberDir(teamID, agentID), "last-handoff.md")
	return WriteContent(path, content)
}

// AppendHandoffHistory appends a handoff entry to the team's handoff history.
func (s *FileTeamStore) AppendHandoffHistory(_ context.Context, teamID string, entry *HandoffEntry) error {
	sharedDir := filepath.Join(s.runtimeTeamsDir(), teamID, "shared")
	if err := os.MkdirAll(sharedDir, 0o755); err != nil {
		return fmt.Errorf("creating shared directory: %w", err)
	}
	path := filepath.Join(sharedDir, "handoff-history.jsonl")

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling handoff entry: %w", err)
	}
	data = append(data, '\n')

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening handoff history: %w", err)
	}
	defer f.Close()

	_, err = f.Write(data)
	return err
}

// GetHandoffHistory reads handoff history entries, optionally filtered by agent and limited.
func (s *FileTeamStore) GetHandoffHistory(_ context.Context, teamID, agentID string, last int) ([]HandoffEntry, error) {
	path := filepath.Join(s.runtimeTeamsDir(), teamID, "shared", "handoff-history.jsonl")
	if !FileExists(path) {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading handoff history: %w", err)
	}

	var entries []HandoffEntry
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry HandoffEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // skip malformed lines
		}
		if agentID != "" && entry.AgentID != agentID {
			continue
		}
		entries = append(entries, entry)
	}

	// Return the last N entries
	if last > 0 && len(entries) > last {
		entries = entries[len(entries)-last:]
	}

	// Reverse to newest-first order
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	return entries, nil
}

// ClearHandoffHistory removes handoff history entries. If agentID is empty, all
// entries are removed. Otherwise only entries for the given agent are removed.
func (s *FileTeamStore) ClearHandoffHistory(_ context.Context, teamID, agentID string) error {
	path := filepath.Join(s.runtimeTeamsDir(), teamID, "shared", "handoff-history.jsonl")
	if !FileExists(path) {
		return nil
	}

	if agentID == "" {
		return os.Remove(path)
	}

	// Read all, filter out the target agent, rewrite.
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading handoff history: %w", err)
	}

	var kept [][]byte
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry HandoffEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			kept = append(kept, []byte(line))
			continue
		}
		if entry.AgentID == agentID {
			continue
		}
		kept = append(kept, []byte(line))
	}

	if len(kept) == 0 {
		return os.Remove(path)
	}

	tmp := path + ".tmp"
	lines := make([]string, len(kept))
	for i, b := range kept {
		lines[i] = string(b)
	}
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing temp handoff history: %w", err)
	}
	return os.Rename(tmp, path)
}

// ClearLastHandoff removes the last handoff file for a team member.
func (s *FileTeamStore) ClearLastHandoff(_ context.Context, teamID, agentID string) error {
	path := filepath.Join(s.runtimeMemberDir(teamID, agentID), "last-handoff.md")
	if !FileExists(path) {
		return nil
	}
	return os.Remove(path)
}

// --- Task Board methods ---

// GetTaskBoard reads the full task board for a team.
func (s *FileTeamStore) GetTaskBoard(_ context.Context, teamID string) (*TeamTaskBoard, error) {
	path := filepath.Join(s.runtimeTeamsDir(), teamID, "shared", "tasks.json")
	if !FileExists(path) {
		return &TeamTaskBoard{Tasks: []TeamTask{}}, nil
	}
	return LoadJSON[TeamTaskBoard](path)
}

// SaveTaskBoard writes the full task board for a team.
func (s *FileTeamStore) SaveTaskBoard(_ context.Context, teamID string, board *TeamTaskBoard) error {
	sharedDir := filepath.Join(s.runtimeTeamsDir(), teamID, "shared")
	if err := os.MkdirAll(sharedDir, 0o755); err != nil {
		return fmt.Errorf("creating shared directory: %w", err)
	}
	path := filepath.Join(sharedDir, "tasks.json")
	return SaveJSON(path, board)
}

// GetTask returns a single task by ID from the team's task board.
func (s *FileTeamStore) GetTask(ctx context.Context, teamID, taskID string) (*TeamTask, error) {
	board, err := s.GetTaskBoard(ctx, teamID)
	if err != nil {
		return nil, err
	}
	for i := range board.Tasks {
		if board.Tasks[i].ID == taskID {
			return &board.Tasks[i], nil
		}
	}
	return nil, nil
}

// UpdateTask performs an atomic read-modify-write on a single task.
func (s *FileTeamStore) UpdateTask(ctx context.Context, teamID, taskID string, updater func(*TeamTask)) error {
	board, err := s.GetTaskBoard(ctx, teamID)
	if err != nil {
		return err
	}

	found := false
	for i := range board.Tasks {
		if board.Tasks[i].ID == taskID {
			updater(&board.Tasks[i])
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("task not found: %s", taskID)
	}

	return s.SaveTaskBoard(ctx, teamID, board)
}

// DeleteTask removes a task by ID from the team's task board.
func (s *FileTeamStore) DeleteTask(ctx context.Context, teamID, taskID string) error {
	board, err := s.GetTaskBoard(ctx, teamID)
	if err != nil {
		return err
	}

	filtered := make([]TeamTask, 0, len(board.Tasks))
	found := false
	for _, task := range board.Tasks {
		if task.ID == taskID {
			found = true
			continue
		}
		filtered = append(filtered, task)
	}
	if !found {
		return fmt.Errorf("task not found: %s", taskID)
	}

	board.Tasks = filtered
	return s.SaveTaskBoard(ctx, teamID, board)
}

// --- Decision Log methods ---

// AppendDecision appends a decision entry to the team's decision log.
func (s *FileTeamStore) AppendDecision(_ context.Context, teamID string, entry *DecisionEntry) error {
	sharedDir := filepath.Join(s.runtimeTeamsDir(), teamID, "shared")
	if err := os.MkdirAll(sharedDir, 0o755); err != nil {
		return fmt.Errorf("creating shared directory: %w", err)
	}
	path := filepath.Join(sharedDir, "decisions.jsonl")

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling decision entry: %w", err)
	}
	data = append(data, '\n')

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening decision log: %w", err)
	}
	defer f.Close()

	_, err = f.Write(data)
	return err
}

// GetDecisions reads decision entries, optionally filtered by context tag, status, and limited.
// Returns the sliced entries, the total count after filtering (before slicing), and any error.
func (s *FileTeamStore) GetDecisions(_ context.Context, teamID, contextTag, statusFilter string, last int) ([]DecisionEntry, int, error) {
	entries, _, err := s.readAllDecisions(teamID)
	if err != nil {
		return nil, 0, err
	}

	if contextTag != "" {
		filtered := make([]DecisionEntry, 0, len(entries))
		for _, e := range entries {
			if e.Context == contextTag {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	if statusFilter != "" {
		filtered := make([]DecisionEntry, 0, len(entries))
		for _, e := range entries {
			status := e.Status
			// Treat empty status as "pending" for backward compatibility
			// with decisions created before the status field was added.
			if status == "" {
				status = DecisionStatusPending
			}
			if status == statusFilter {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	total := len(entries)

	if last > 0 && len(entries) > last {
		entries = entries[len(entries)-last:]
	}

	return entries, total, nil
}

// readAllDecisions reads all decision entries from the JSONL file.
func (s *FileTeamStore) readAllDecisions(teamID string) ([]DecisionEntry, string, error) {
	path := filepath.Join(s.runtimeTeamsDir(), teamID, "shared", "decisions.jsonl")
	if !FileExists(path) {
		return nil, path, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, path, fmt.Errorf("reading decision log: %w", err)
	}
	var entries []DecisionEntry
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry DecisionEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, path, nil
}

// writeAllDecisions rewrites the full decisions JSONL file.
func (s *FileTeamStore) writeAllDecisions(path string, entries []DecisionEntry) error {
	var buf strings.Builder
	for _, e := range entries {
		data, err := json.Marshal(e)
		if err != nil {
			return fmt.Errorf("marshaling decision entry: %w", err)
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(buf.String()), 0o644)
}

// UpdateDecision updates a decision entry by ID using the provided updater function.
func (s *FileTeamStore) UpdateDecision(_ context.Context, teamID, decisionID string, updater func(*DecisionEntry)) error {
	entries, path, err := s.readAllDecisions(teamID)
	if err != nil {
		return err
	}
	found := false
	for i := range entries {
		if entries[i].ID == decisionID {
			updater(&entries[i])
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("decision not found: %s", decisionID)
	}
	return s.writeAllDecisions(path, entries)
}

// DeleteDecision removes a decision entry by ID.
func (s *FileTeamStore) DeleteDecision(_ context.Context, teamID, decisionID string) error {
	entries, path, err := s.readAllDecisions(teamID)
	if err != nil {
		return err
	}
	filtered := make([]DecisionEntry, 0, len(entries))
	found := false
	for _, e := range entries {
		if e.ID == decisionID {
			found = true
			continue
		}
		filtered = append(filtered, e)
	}
	if !found {
		return fmt.Errorf("decision not found: %s", decisionID)
	}
	return s.writeAllDecisions(path, filtered)
}

// --- Knowledge Log methods ---

// AppendKnowledge appends a knowledge entry to the team's knowledge log.
func (s *FileTeamStore) AppendKnowledge(_ context.Context, teamID string, entry *KnowledgeEntry) error {
	sharedDir := filepath.Join(s.runtimeTeamsDir(), teamID, "shared")
	if err := os.MkdirAll(sharedDir, 0o755); err != nil {
		return fmt.Errorf("creating shared directory: %w", err)
	}
	path := filepath.Join(sharedDir, "knowledge.jsonl")

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling knowledge entry: %w", err)
	}
	data = append(data, '\n')

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening knowledge log: %w", err)
	}
	defer f.Close()

	_, err = f.Write(data)
	return err
}

// GetKnowledge reads knowledge entries, optionally filtered by exact topic
// match, topic prefix match, and limited to the most recent N entries. When
// both topicFilter and topicPrefix are non-empty, an entry must satisfy both
// to be returned; in practice upstream layers enforce mutual exclusion so the
// AND semantics rarely matter.
func (s *FileTeamStore) GetKnowledge(_ context.Context, teamID, topicFilter, topicPrefix string, last int) ([]KnowledgeEntry, error) {
	entries, _, err := s.readAllKnowledge(teamID)
	if err != nil {
		return nil, err
	}

	if topicFilter != "" || topicPrefix != "" {
		filtered := make([]KnowledgeEntry, 0, len(entries))
		for _, e := range entries {
			if topicFilter != "" && e.Topic != topicFilter {
				continue
			}
			if topicPrefix != "" && !strings.HasPrefix(e.Topic, topicPrefix) {
				continue
			}
			filtered = append(filtered, e)
		}
		entries = filtered
	}

	if last > 0 && len(entries) > last {
		entries = entries[len(entries)-last:]
	}

	return entries, nil
}

// readAllKnowledge reads all knowledge entries from the JSONL file.
func (s *FileTeamStore) readAllKnowledge(teamID string) ([]KnowledgeEntry, string, error) {
	path := filepath.Join(s.runtimeTeamsDir(), teamID, "shared", "knowledge.jsonl")
	if !FileExists(path) {
		return nil, path, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, path, fmt.Errorf("reading knowledge log: %w", err)
	}
	var entries []KnowledgeEntry
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry KnowledgeEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, path, nil
}

// writeAllKnowledge rewrites the full knowledge JSONL file.
func (s *FileTeamStore) writeAllKnowledge(path string, entries []KnowledgeEntry) error {
	var buf strings.Builder
	for _, e := range entries {
		data, err := json.Marshal(e)
		if err != nil {
			return fmt.Errorf("marshaling knowledge entry: %w", err)
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(buf.String()), 0o644)
}

// UpdateKnowledge updates a knowledge entry by ID using the provided updater function.
func (s *FileTeamStore) UpdateKnowledge(_ context.Context, teamID, knowledgeID string, updater func(*KnowledgeEntry)) error {
	entries, path, err := s.readAllKnowledge(teamID)
	if err != nil {
		return err
	}
	found := false
	for i := range entries {
		if entries[i].ID == knowledgeID {
			updater(&entries[i])
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("knowledge entry not found: %s", knowledgeID)
	}
	return s.writeAllKnowledge(path, entries)
}

// DeleteKnowledge removes a knowledge entry by ID.
func (s *FileTeamStore) DeleteKnowledge(_ context.Context, teamID, knowledgeID string) error {
	entries, path, err := s.readAllKnowledge(teamID)
	if err != nil {
		return err
	}
	filtered := make([]KnowledgeEntry, 0, len(entries))
	found := false
	for _, e := range entries {
		if e.ID == knowledgeID {
			found = true
			continue
		}
		filtered = append(filtered, e)
	}
	if !found {
		return fmt.Errorf("knowledge entry not found: %s", knowledgeID)
	}
	return s.writeAllKnowledge(path, filtered)
}

// --- Retention / Prune ---

// PruneResult reports how many items were removed during pruning.
type PruneResult struct {
	TasksRemoved     int `json:"tasksRemoved"`
	DecisionsRemoved int `json:"decisionsRemoved"`
	KnowledgeRemoved int `json:"knowledgeRemoved"`
}

// PruneSharedState removes stale completed tasks, decisions, and knowledge
// entries according to the team's retention policy.
func (s *FileTeamStore) PruneSharedState(ctx context.Context, teamID string) (*PruneResult, error) {
	team, err := s.Get(ctx, teamID)
	if err != nil {
		return nil, err
	}
	retention := team.EffectiveRetention()
	result := &PruneResult{}
	now := time.Now().UTC()

	// 1. Prune completed tasks
	if err := s.pruneCompletedTasks(ctx, teamID, retention.Tasks, now, result); err != nil {
		return result, fmt.Errorf("pruning tasks: %w", err)
	}

	// 2. Prune decisions
	if err := s.pruneDecisions(teamID, retention.Decisions, now, result); err != nil {
		return result, fmt.Errorf("pruning decisions: %w", err)
	}

	// 3. Prune knowledge
	if err := s.pruneKnowledge(teamID, retention.Knowledge, now, result); err != nil {
		return result, fmt.Errorf("pruning knowledge: %w", err)
	}

	return result, nil
}

func (s *FileTeamStore) pruneCompletedTasks(ctx context.Context, teamID string, cfg *TaskRetention, now time.Time, result *PruneResult) error {
	if cfg == nil {
		return nil
	}

	board, err := s.GetTaskBoard(ctx, teamID)
	if err != nil {
		return err
	}

	var active, completed []TeamTask
	for _, t := range board.Tasks {
		if t.Status == "done" {
			completed = append(completed, t)
		} else {
			active = append(active, t)
		}
	}

	if len(completed) == 0 {
		return nil
	}

	// Sort completed by UpdatedAt desc (newest first)
	sort.Slice(completed, func(i, j int) bool {
		return completed[i].UpdatedAt > completed[j].UpdatedAt
	})

	kept := make([]TeamTask, 0, len(completed))
	for i, t := range completed {
		// Apply maxCompleted: keep only N newest
		if cfg.MaxCompleted > 0 && i >= cfg.MaxCompleted {
			continue
		}
		// Apply maxAgeDays: remove completed tasks older than cutoff
		if cfg.MaxAgeDays > 0 {
			parsed, parseErr := time.Parse(time.RFC3339, t.UpdatedAt)
			if parseErr == nil && now.Sub(parsed) > time.Duration(cfg.MaxAgeDays)*24*time.Hour {
				continue
			}
		}
		kept = append(kept, t)
	}

	removed := len(completed) - len(kept)
	if removed == 0 {
		return nil
	}

	result.TasksRemoved = removed
	board.Tasks = append(active, kept...)
	return s.SaveTaskBoard(ctx, teamID, board)
}

func (s *FileTeamStore) pruneDecisions(teamID string, cfg *EntryRetention, now time.Time, result *PruneResult) error {
	if cfg == nil {
		return nil
	}

	entries, path, err := s.readAllDecisions(teamID)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}

	kept := pruneEntries(entries, cfg, now, func(e DecisionEntry) string { return e.At })
	removed := len(entries) - len(kept)
	if removed == 0 {
		return nil
	}

	result.DecisionsRemoved = removed
	return s.writeAllDecisions(path, kept)
}

func (s *FileTeamStore) pruneKnowledge(teamID string, cfg *EntryRetention, now time.Time, result *PruneResult) error {
	if cfg == nil {
		return nil
	}

	entries, path, err := s.readAllKnowledge(teamID)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}

	kept := pruneEntries(entries, cfg, now, func(e KnowledgeEntry) string { return e.At })
	removed := len(entries) - len(kept)
	if removed == 0 {
		return nil
	}

	result.KnowledgeRemoved = removed
	return s.writeAllKnowledge(path, kept)
}

// pruneEntries is a generic helper that applies maxEntries and maxAgeDays to a slice.
// The getAt function extracts the timestamp string from each entry.
// Entries are assumed to be in chronological order (oldest first), which is the
// natural order for append-only JSONL files.
func pruneEntries[T any](entries []T, cfg *EntryRetention, now time.Time, getAt func(T) string) []T {
	// Apply maxEntries: keep N newest (from the end)
	if cfg.MaxEntries > 0 && len(entries) > cfg.MaxEntries {
		entries = entries[len(entries)-cfg.MaxEntries:]
	}

	// Apply maxAgeDays: remove entries older than cutoff
	if cfg.MaxAgeDays > 0 {
		cutoff := now.Add(-time.Duration(cfg.MaxAgeDays) * 24 * time.Hour)
		kept := make([]T, 0, len(entries))
		for _, e := range entries {
			parsed, parseErr := time.Parse(time.RFC3339, getAt(e))
			if parseErr != nil {
				// Keep entries with unparseable timestamps
				kept = append(kept, e)
				continue
			}
			if !parsed.Before(cutoff) {
				kept = append(kept, e)
			}
		}
		entries = kept
	}

	return entries
}

package store

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"prompt-manager/internal/finding"
	"prompt-manager/internal/sourceledger"

	"prompt-manager/internal/teamconfig"
	"prompt-manager/internal/teamcontract"

	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
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
//     shared/tasks.json and member runtime files. Durable team corpus lives in
//     source-ledger scopes.
//
// Path shapes under each root mirror the original single-tree layout (CD-2);
// only the prefix differs. The operator-facing shared/ tree is virtually
// merged by ListSharedFiles and single-file ops route by filename class.
type FileTeamStore struct {
	configRoot      string
	runtimeDataRoot string
	relationStore   RelationStore
	routedRoots     *filerouting.RoutedRoots
	ledger          *sourceledger.Client
	eventsBase      string
	corpus          *memoryCorpus
}

// NewFileTeamStore creates a new file-based team store rooted at the given
// Config and RuntimeData class directories. Production callers pass
// roots.Config and roots.RuntimeData; tests that don't care about the split
// can pass the same tempdir for both.
func NewFileTeamStore(configRoot, runtimeDataRoot string, relationStore RelationStore, routedRoots ...*filerouting.RoutedRoots) *FileTeamStore {
	var roots *filerouting.RoutedRoots
	if len(routedRoots) > 0 {
		roots = routedRoots[0]
	}
	return &FileTeamStore{
		configRoot:      configRoot,
		runtimeDataRoot: runtimeDataRoot,
		relationStore:   relationStore,
		routedRoots:     roots,
		corpus:          newMemoryCorpus(),
	}
}

// SetSourceLedger attaches the shared durable corpus used by production team
// runtime paths. Nil is retained only for isolated file-store tests.
func (s *FileTeamStore) SetSourceLedger(client *sourceledger.Client) { s.ledger = client }

// HasSourceLedger reports whether this store is wired to the production
// Source Ledger dependency. Isolated file-store fixtures intentionally return
// false so prompt tests do not need a live ledger.
func (s *FileTeamStore) HasSourceLedger() bool { return s != nil && s.ledger != nil }

// EnsureTeamScope makes the Source Ledger dependency explicit at the member
// boundary. Production stores have a ledger client; isolated file-store tests
// intentionally keep the nil client and exercise their local fixture seam.
func (s *FileTeamStore) EnsureTeamScope(ctx context.Context, teamID string) error {
	if s == nil || s.ledger == nil {
		return nil
	}
	return s.ledger.EnsureTeamScope(ctx, teamID)
}

// SetEventsEndpoint attaches the durable event stream used for heartbeat
// attempts. Production never falls back to a local attempts file; nil is
// retained only for isolated fixture stores.
func (s *FileTeamStore) SetEventsEndpoint(base string) {
	s.eventsBase = strings.TrimRight(strings.TrimSpace(base), "/")
}

// forContext returns a short-lived, root-resolved view for a request. The
// returned clone deliberately has no routedRoots, preventing recursive
// resolution while keeping concurrent requests isolated from one another.
func (s *FileTeamStore) forContext(ctx context.Context) *FileTeamStore {
	if s.routedRoots == nil {
		return s
	}
	configRoot, err := s.routedRoots.Pick(ctx, storage.ClassConfig)
	if err != nil {
		return s
	}
	runtimeRoot, err := s.routedRoots.Pick(ctx, storage.ClassData)
	if err != nil {
		return s
	}
	cloned := NewFileTeamStore(configRoot, runtimeRoot, s.relationStore)
	cloned.ledger = s.ledger
	cloned.eventsBase = s.eventsBase
	cloned.corpus = s.corpus
	return cloned
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
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.List(ctx)
	}
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
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.Get(ctx, id)
	}
	return s.loadTeam(id)
}

// Create creates a new team
func (s *FileTeamStore) Create(ctx context.Context, team *Team) error {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.Create(ctx, team)
	}
	// Check team doesn't already exist
	if _, err := s.Get(ctx, team.ID); err == nil {
		return fmt.Errorf("team already exists: %s", team.ID)
	}
	if err := teamconfig.Validate(team.Contract()); err != nil {
		return err
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
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.Update(ctx, id, updates)
	}
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
	if updates.OperatingContract != nil {
		team.OperatingContract = updates.OperatingContract
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
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.Delete(ctx, id)
	}
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
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.GetRoles(ctx, teamID)
	}
	rolesPath := filepath.Join(s.teamsDir(), teamID, "roles.json")
	return LoadJSON[TeamRoles](rolesPath)
}

// GetOrgChart returns the org chart for a team
func (s *FileTeamStore) GetOrgChart(ctx context.Context, teamID string) (*OrgChart, error) {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.GetOrgChart(ctx, teamID)
	}
	orgPath := filepath.Join(s.teamsDir(), teamID, "org.json")
	return LoadJSON[OrgChart](orgPath)
}

// GetMembers returns all members of a team
func (s *FileTeamStore) GetMembers(ctx context.Context, teamID string) ([]TeamMemberRelation, error) {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.GetMembers(ctx, teamID)
	}
	if s.relationStore == nil {
		return nil, fmt.Errorf("relation store not configured")
	}
	return s.relationStore.ListTeamMembers(ctx, teamID)
}

// SetRoles sets the role definitions for a team
func (s *FileTeamStore) SetRoles(ctx context.Context, teamID string, roles *TeamRoles) error {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.SetRoles(ctx, teamID, roles)
	}
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
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.SetOrgChart(ctx, teamID, org)
	}
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
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.GetInbox(ctx, teamID, agentID)
	}
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
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.SetInbox(ctx, teamID, agentID, inbox)
	}
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
	for _, finding := range teamconfig.ValidateFindings(team.Contract()) {
		team.ValidationFindings = append(team.ValidationFindings, TeamValidationFinding{
			Source:  "teamconfig",
			Field:   finding.Field,
			Message: finding.Message,
		})
	}
	for _, f := range s.validateOperatingContractFindings(context.Background(), team) {
		team.ValidationFindings = append(team.ValidationFindings, TeamValidationFinding{
			Source:  "teamcontract",
			Rule:    f.Rule,
			Field:   f.Path,
			Message: f.Detail,
		})
	}
	return team, nil
}

func (s *FileTeamStore) validateOperatingContract(ctx context.Context, team *Team) error {
	findings := s.validateOperatingContractFindings(ctx, team)
	if len(findings) == 0 {
		return nil
	}
	return fmt.Errorf("%s", findings[0].Detail)
}

func (s *FileTeamStore) validateOperatingContractFindings(ctx context.Context, team *Team) []finding.Finding {
	if team == nil {
		return []finding.Finding{{Rule: "contract_missing", Severity: finding.SeverityError, Path: "team", Detail: "team is required"}}
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
	return teamcontract.ValidateFindings(team.OperatingContract, teamcontract.ValidationInput{
		TeamID:    team.ID,
		MemberIDs: memberIDs,
		StoreDir:  s.configRoot,
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
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.DeleteMemberData(ctx, teamID, agentID)
	}
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
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.EnsureMemberDir(ctx, teamID, agentID)
	}
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
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.GetResponsibilities(ctx, teamID, agentID)
	}
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
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.SetResponsibilities(ctx, teamID, agentID, content)
	}
	// Ensure member directory exists
	if err := s.EnsureMemberDir(ctx, teamID, agentID); err != nil {
		return err
	}

	respPath := filepath.Join(s.configMemberDir(teamID, agentID), "RESPONSIBILITIES.md")
	return WriteContent(respPath, content)
}

// GetHeartbeatInstructions reads the HEARTBEAT.md content for a team member
func (s *FileTeamStore) GetHeartbeatInstructions(ctx context.Context, teamID, agentID string) (string, error) {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.GetHeartbeatInstructions(ctx, teamID, agentID)
	}
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
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.SetHeartbeatInstructions(ctx, teamID, agentID, content)
	}
	// Ensure member directory exists
	if err := s.EnsureMemberDir(ctx, teamID, agentID); err != nil {
		return err
	}

	hbPath := filepath.Join(s.configMemberDir(teamID, agentID), "HEARTBEAT.md")
	return WriteContent(hbPath, content)
}

// GetHeartbeatConfig reads the heartbeat.json config for a team member
func (s *FileTeamStore) GetHeartbeatConfig(ctx context.Context, teamID, agentID string) (*HeartbeatConfig, error) {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.GetHeartbeatConfig(ctx, teamID, agentID)
	}
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
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.SetHeartbeatConfig(ctx, teamID, agentID, config)
	}
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
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.DeleteHeartbeatConfig(ctx, teamID, agentID)
	}
	configPath := filepath.Join(s.runtimeMemberDir(teamID, agentID), "heartbeat.json")
	return DeleteFile(configPath)
}

// ListHeartbeatConfigs lists all heartbeat configs for a team
func (s *FileTeamStore) ListHeartbeatConfigs(ctx context.Context, teamID string) ([]HeartbeatConfig, error) {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.ListHeartbeatConfigs(ctx, teamID)
	}
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
func (s *FileTeamStore) AppendHeartbeatAttempt(ctx context.Context, teamID string, entry *HeartbeatAttempt) error {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.AppendHeartbeatAttempt(ctx, teamID, entry)
	}
	if s.ledger != nil {
		return s.appendHeartbeatAttemptEvent(ctx, entry)
	}
	s.corpus.mu.Lock()
	defer s.corpus.mu.Unlock()
	s.corpus.attempts[teamID] = append(s.corpus.attempts[teamID], *entry)
	return nil
}

const heartbeatAttemptEventType = "prompt-manager.heartbeat.attempt.v1"

func (s *FileTeamStore) appendHeartbeatAttemptEvent(ctx context.Context, entry *HeartbeatAttempt) error {
	if strings.TrimSpace(s.eventsBase) == "" {
		return fmt.Errorf("vrooli-events unavailable: heartbeat attempt persistence requires the shared event stream")
	}
	if strings.TrimSpace(entry.ID) == "" {
		entry.ID = fmt.Sprintf("heartbeat-attempt-%d", time.Now().UTC().UnixNano())
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("encode heartbeat attempt: %w", err)
	}
	var values map[string]any
	if err := json.Unmarshal(data, &values); err != nil {
		return fmt.Errorf("prepare heartbeat attempt event: %w", err)
	}
	structData, err := structpb.NewStruct(values)
	if err != nil {
		return fmt.Errorf("pack heartbeat attempt event: %w", err)
	}
	packed, err := anypb.New(structData)
	if err != nil {
		return fmt.Errorf("pack heartbeat attempt payload: %w", err)
	}
	env := &domain.EventEnvelope{
		EventId: entry.ID, EventType: heartbeatAttemptEventType, OccurredAt: timestamppb.Now(),
		Source: &domain.EventSource{Scenario: "prompt-manager", ActorKind: "system"},
		Target: &domain.EventTarget{Scenario: "prompt-manager", Operation: "heartbeat-attempt", Protocol: "http"},
		Data:   packed,
	}
	if entry.RunID != "" || entry.TaskID != "" {
		env.Correlation = &domain.EventCorrelation{AgentRunId: entry.RunID, TaskId: entry.TaskID}
	}
	body, err := (protojson.MarshalOptions{}).Marshal(env)
	if err != nil {
		return fmt.Errorf("encode heartbeat attempt event envelope: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.eventsBase+"/api/v1/events", strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("create heartbeat attempt event request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("publish heartbeat attempt event: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusConflict {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("publish heartbeat attempt event: %s: %s", resp.Status, strings.TrimSpace(string(responseBody)))
	}
	return nil
}

// ListHeartbeatAttempts returns newest-first heartbeat dispatch attempts across
// all teams, optionally filtered by team, agent, status, and profile key.
func (s *FileTeamStore) ListHeartbeatAttempts(ctx context.Context, teamID, agentID, status, profileKey string, limit, offset int) ([]HeartbeatAttempt, int, error) {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.ListHeartbeatAttempts(ctx, teamID, agentID, status, profileKey, limit, offset)
	}
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
	if s.ledger != nil {
		var err error
		entries, err = s.listHeartbeatAttemptEvents(ctx, teamID, agentID, status, profileKey, limit, offset)
		if err != nil {
			return nil, 0, err
		}
	} else {
		s.corpus.mu.Lock()
		for _, id := range teamIDs {
			attempts := append([]HeartbeatAttempt(nil), s.corpus.attempts[id]...)
			for _, entry := range attempts {
				if agentID != "" && entry.AgentID != agentID || status != "" && entry.Status != status || profileKey != "" && entry.ProfileKey != profileKey {
					continue
				}
				entries = append(entries, entry)
			}
		}
		s.corpus.mu.Unlock()
	}
	for _, id := range teamIDs {

		configs, _ := s.ListHeartbeatConfigs(ctx, id)
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

func (s *FileTeamStore) listHeartbeatAttemptEvents(ctx context.Context, teamID, agentID, status, profileKey string, limit, offset int) ([]HeartbeatAttempt, error) {
	u, err := url.Parse(s.eventsBase + "/api/v1/events")
	if err != nil {
		return nil, fmt.Errorf("parse vrooli-events endpoint: %w", err)
	}
	query := u.Query()
	query.Set("type", heartbeatAttemptEventType)
	query.Set("source", "prompt-manager")
	query.Set("limit", "100")
	u.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create heartbeat attempt query: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query heartbeat attempt events: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("query heartbeat attempt events: %s", resp.Status)
	}
	var raw []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode heartbeat attempt events: %w", err)
	}
	entries := make([]HeartbeatAttempt, 0, len(raw))
	for _, item := range raw {
		env := &domain.EventEnvelope{}
		if err := protojson.Unmarshal(item, env); err != nil || env.GetData() == nil {
			continue
		}
		payload := &structpb.Struct{}
		if err := anypb.UnmarshalTo(env.GetData(), payload, proto.UnmarshalOptions{}); err != nil {
			continue
		}
		body, err := json.Marshal(payload.AsMap())
		if err != nil {
			continue
		}
		var entry HeartbeatAttempt
		if err := json.Unmarshal(body, &entry); err != nil {
			continue
		}
		if entry.ID == "" {
			entry.ID = env.GetEventId()
		}
		if teamID != "" && entry.TeamID != teamID || agentID != "" && entry.AgentID != agentID || status != "" && entry.Status != status || profileKey != "" && entry.ProfileKey != profileKey {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// GetMemberLogPath returns the path for a heartbeat execution log
func (s *FileTeamStore) GetMemberLogPath(teamID, agentID, timestamp string) string {
	return filepath.Join(s.runtimeMemberDir(teamID, agentID), "logs", timestamp+".log")
}

// GetMemberLogPathForContext resolves the runtime log path through the same
// per-request root selection as all mutating team operations. New request
// handlers and executors must use this rather than the legacy context-free
// accessor, which remains for non-request compatibility callers.
func (s *FileTeamStore) GetMemberLogPathForContext(ctx context.Context, teamID, agentID, timestamp string) string {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.GetMemberLogPathForContext(ctx, teamID, agentID, timestamp)
	}
	return s.GetMemberLogPath(teamID, agentID, timestamp)
}

// ListMemberLogs lists all log files for a team member
func (s *FileTeamStore) ListMemberLogs(ctx context.Context, teamID, agentID string) ([]string, error) {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.ListMemberLogs(ctx, teamID, agentID)
	}
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
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.ListSharedFiles(ctx, teamID)
	}
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
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.ReadSharedFile(ctx, teamID, relPath)
	}
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
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.WriteSharedFile(ctx, teamID, relPath, content)
	}
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
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.CreateSharedFile(ctx, teamID, relPath, content, isDir)
	}
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
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.RenameSharedFile(ctx, teamID, fromPath, toPath)
	}
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
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.DeleteSharedFile(ctx, teamID, relPath)
	}
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
// mutable tasks.json only).
func (s *FileTeamStore) runtimeSharedDir(team *Team) string {
	return filepath.Join(s.runtimeTeamsDir(), team.ID, s.sharedSubpath(team))
}

// runtimeSharedBasenames is the set of file basenames that live under the
// RuntimeData shared/ root. Operator file ops use it to route a request to
// the correct class root.
var runtimeSharedBasenames = map[string]struct{}{
	"tasks.json": {},
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
func (s *FileTeamStore) GetLastHandoff(ctx context.Context, teamID, agentID string) (string, error) {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.GetLastHandoff(ctx, teamID, agentID)
	}
	if s.ledger != nil {
		entries, err := s.ledger.List(ctx, sourceLedgerScope(teamID), 500)
		if err != nil {
			return "", err
		}
		for i := len(entries) - 1; i >= 0; i-- {
			if entries[i].Kind != sourceLedgerHandoffSnapshotKind {
				continue
			}
			handoff, ok := decodeHandoff(entries[i].Body)
			if ok && handoff.AgentID == agentID {
				return handoff.Content, nil
			}
		}
		return "", nil
	}
	path := filepath.Join(s.runtimeMemberDir(teamID, agentID), "last-handoff.md")
	if !FileExists(path) {
		return "", nil
	}
	return ReadContent(path)
}

// SetLastHandoff writes the last handoff markdown for a team member.
func (s *FileTeamStore) SetLastHandoff(ctx context.Context, teamID, agentID, content string) error {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.SetLastHandoff(ctx, teamID, agentID, content)
	}
	if s.ledger != nil {
		body, err := encodeHandoff(HandoffEntry{AgentID: agentID, Content: content, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)})
		if err != nil {
			return err
		}
		_, err = s.ledger.Append(ctx, sourceLedgerScope(teamID), body, sourceLedgerHandoffSnapshotKind)
		return err
	}
	if err := s.EnsureMemberDir(ctx, teamID, agentID); err != nil {
		return err
	}
	path := filepath.Join(s.runtimeMemberDir(teamID, agentID), "last-handoff.md")
	return WriteContent(path, content)
}

// AppendHandoffHistory appends a handoff entry to the team's handoff history.
func (s *FileTeamStore) AppendHandoffHistory(ctx context.Context, teamID string, entry *HandoffEntry) error {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.AppendHandoffHistory(ctx, teamID, entry)
	}
	if s.ledger != nil {
		body, err := encodeHandoff(*entry)
		if err != nil {
			return err
		}
		_, err = s.ledger.Append(ctx, sourceLedgerScope(teamID), body, sourceLedgerHandoffKind)
		return err
	}
	s.corpus.mu.Lock()
	defer s.corpus.mu.Unlock()
	s.corpus.handoffs[teamID] = append(s.corpus.handoffs[teamID], *entry)
	return nil
}

// GetHandoffHistory reads handoff history entries, optionally filtered by agent and limited.
func (s *FileTeamStore) GetHandoffHistory(ctx context.Context, teamID, agentID string, last int) ([]HandoffEntry, error) {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.GetHandoffHistory(ctx, teamID, agentID, last)
	}
	if s.ledger != nil {
		entries, err := s.ledger.List(ctx, sourceLedgerScope(teamID), 500)
		if err != nil {
			return nil, err
		}
		history := make([]HandoffEntry, 0, len(entries))
		for _, item := range entries {
			if item.Kind != sourceLedgerHandoffKind {
				continue
			}
			handoff, ok := decodeHandoff(item.Body)
			if !ok || (agentID != "" && handoff.AgentID != agentID) {
				continue
			}
			history = append(history, handoff)
		}
		if last > 0 && len(history) > last {
			history = history[len(history)-last:]
		}
		for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
			history[i], history[j] = history[j], history[i]
		}
		return history, nil
	}
	s.corpus.mu.Lock()
	entries := append([]HandoffEntry(nil), s.corpus.handoffs[teamID]...)
	s.corpus.mu.Unlock()
	filtered := entries[:0]
	for _, entry := range entries {
		if agentID == "" || entry.AgentID == agentID {
			filtered = append(filtered, entry)
		}
	}
	entries = filtered

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
func (s *FileTeamStore) ClearHandoffHistory(ctx context.Context, teamID, agentID string) error {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.ClearHandoffHistory(ctx, teamID, agentID)
	}
	if s.ledger != nil {
		return fmt.Errorf("source-ledger handoff history is append-only; clearing is unsupported")
	}
	s.corpus.mu.Lock()
	defer s.corpus.mu.Unlock()
	if agentID == "" {
		s.corpus.handoffs[teamID] = nil
		return nil
	}
	kept := s.corpus.handoffs[teamID][:0]
	for _, entry := range s.corpus.handoffs[teamID] {
		if entry.AgentID != agentID {
			kept = append(kept, entry)
		}
	}
	s.corpus.handoffs[teamID] = kept
	return nil
}

// ClearLastHandoff removes the last handoff file for a team member.
func (s *FileTeamStore) ClearLastHandoff(ctx context.Context, teamID, agentID string) error {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.ClearLastHandoff(ctx, teamID, agentID)
	}
	if s.ledger != nil {
		return fmt.Errorf("source-ledger handoff snapshots are append-only; clearing is unsupported")
	}
	path := filepath.Join(s.runtimeMemberDir(teamID, agentID), "last-handoff.md")
	if !FileExists(path) {
		return nil
	}
	return os.Remove(path)
}

// --- Task Board methods ---

// GetTaskBoard reads the full task board for a team.
func (s *FileTeamStore) GetTaskBoard(ctx context.Context, teamID string) (*TeamTaskBoard, error) {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.GetTaskBoard(ctx, teamID)
	}
	path := filepath.Join(s.runtimeTeamsDir(), teamID, "shared", "tasks.json")
	if !FileExists(path) {
		return &TeamTaskBoard{Tasks: []TeamTask{}}, nil
	}
	return LoadJSON[TeamTaskBoard](path)
}

// SaveTaskBoard writes the full task board for a team.
func (s *FileTeamStore) SaveTaskBoard(ctx context.Context, teamID string, board *TeamTaskBoard) error {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.SaveTaskBoard(ctx, teamID, board)
	}
	sharedDir := filepath.Join(s.runtimeTeamsDir(), teamID, "shared")
	if err := os.MkdirAll(sharedDir, 0o755); err != nil {
		return fmt.Errorf("creating shared directory: %w", err)
	}
	path := filepath.Join(sharedDir, "tasks.json")
	return SaveJSON(path, board)
}

// GetTask returns a single task by ID from the team's task board.
func (s *FileTeamStore) GetTask(ctx context.Context, teamID, taskID string) (*TeamTask, error) {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.GetTask(ctx, teamID, taskID)
	}
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
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.UpdateTask(ctx, teamID, taskID, updater)
	}
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
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.DeleteTask(ctx, teamID, taskID)
	}
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

// --- Team corpus methods ---

// AppendTeamCorpus appends durable team context to the team's source-ledger scope.
func (s *FileTeamStore) AppendTeamCorpus(ctx context.Context, teamID string, entry *KnowledgeEntry) error {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.AppendTeamCorpus(ctx, teamID, entry)
	}
	if s.ledger != nil {
		body, err := encodeKnowledge(*entry)
		if err != nil {
			return err
		}
		_, err = s.ledger.Append(ctx, sourceLedgerScope(teamID), body, sourceLedgerKnowledgeKind)
		return err
	}
	s.corpus.mu.Lock()
	defer s.corpus.mu.Unlock()
	s.corpus.knowledge[teamID] = append(s.corpus.knowledge[teamID], *entry)
	return nil
}

// ListTeamCorpus reads team context, optionally filtered by exact topic
// match, topic prefix match, and limited to the most recent N entries. When
// both topicFilter and topicPrefix are non-empty, an entry must satisfy both
// to be returned; in practice upstream layers enforce mutual exclusion so the
// AND semantics rarely matter.
func (s *FileTeamStore) ListTeamCorpus(ctx context.Context, teamID, topicFilter, topicPrefix string, last int) ([]KnowledgeEntry, error) {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.ListTeamCorpus(ctx, teamID, topicFilter, topicPrefix, last)
	}
	if s.ledger != nil {
		entries, err := s.ledger.List(ctx, sourceLedgerScope(teamID), 500)
		if err != nil {
			return nil, err
		}
		knowledge := make([]KnowledgeEntry, 0, len(entries))
		for _, item := range entries {
			if item.Kind != sourceLedgerKnowledgeKind {
				continue
			}
			entry, ok := decodeKnowledge(item.Body)
			if !ok {
				continue
			}
			if entry.At == "" {
				entry.At = item.CreatedAt
			}
			if topicFilter != "" && entry.Topic != topicFilter {
				continue
			}
			if topicPrefix != "" && !strings.HasPrefix(entry.Topic, topicPrefix) {
				continue
			}
			knowledge = append(knowledge, entry)
		}
		if last > 0 && len(knowledge) > last {
			knowledge = knowledge[len(knowledge)-last:]
		}
		return knowledge, nil
	}
	s.corpus.mu.Lock()
	entries := append([]KnowledgeEntry(nil), s.corpus.knowledge[teamID]...)
	s.corpus.mu.Unlock()

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

// WakeTeamCorpus returns the bounded ambient context for a team from its
// Source Ledger scope. A nil result is intentional for isolated file-store
// fixtures that do not attach the production ledger client.
func (s *FileTeamStore) WakeTeamCorpus(ctx context.Context, teamID string) (sourceledger.WakeResult, error) {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.WakeTeamCorpus(ctx, teamID)
	}
	if s.ledger == nil {
		return sourceledger.WakeResult{}, nil
	}
	return s.ledger.Wake(ctx, sourceLedgerScope(teamID), 0)
}

// UpdateTeamCorpus updates a team-corpus entry by ID using the provided updater function.
func (s *FileTeamStore) UpdateTeamCorpus(ctx context.Context, teamID, knowledgeID string, updater func(*KnowledgeEntry)) error {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.UpdateTeamCorpus(ctx, teamID, knowledgeID, updater)
	}
	if s.ledger != nil {
		entries, err := s.ListTeamCorpus(ctx, teamID, "", "", 0)
		if err != nil {
			return err
		}
		for i := range entries {
			if entries[i].ID == knowledgeID {
				updater(&entries[i])
				return s.AppendTeamCorpus(ctx, teamID, &entries[i])
			}
		}
		return fmt.Errorf("knowledge entry not found: %s", knowledgeID)
	}
	s.corpus.mu.Lock()
	defer s.corpus.mu.Unlock()
	entries := append([]KnowledgeEntry(nil), s.corpus.knowledge[teamID]...)
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
	s.corpus.knowledge[teamID] = entries
	return nil
}

// DeleteTeamCorpus removes a team-corpus entry by ID in fixture mode.
func (s *FileTeamStore) DeleteTeamCorpus(ctx context.Context, teamID, knowledgeID string) error {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.DeleteTeamCorpus(ctx, teamID, knowledgeID)
	}
	if s.ledger != nil {
		return fmt.Errorf("source-ledger knowledge is append-only; file a superseding entry instead of deleting %s", knowledgeID)
	}
	s.corpus.mu.Lock()
	defer s.corpus.mu.Unlock()
	entries := append([]KnowledgeEntry(nil), s.corpus.knowledge[teamID]...)
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
	s.corpus.knowledge[teamID] = filtered
	return nil
}

// SaveBugDraft persists a private repairable intake draft. Drafts have their
// own fixture store and are intentionally never returned by ListTeamCorpus.
func (s *FileTeamStore) SaveBugDraft(ctx context.Context, teamID string, draft BugDraft) error {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.SaveBugDraft(ctx, teamID, draft)
	}
	if s.ledger != nil {
		data, err := json.Marshal(draft)
		if err != nil {
			return fmt.Errorf("marshal bug draft: %w", err)
		}
		_, err = s.ledger.Append(ctx, sourceLedgerScope(teamID), string(data), sourceLedgerBugDraftKind)
		return err
	}
	s.corpus.mu.Lock()
	defer s.corpus.mu.Unlock()
	drafts := append([]BugDraft(nil), s.corpus.drafts[teamID]...)
	replaced := false
	for i := range drafts {
		if drafts[i].ID == draft.ID {
			drafts[i] = draft
			replaced = true
			break
		}
	}
	if !replaced {
		drafts = append(drafts, draft)
	}
	s.corpus.drafts[teamID] = drafts
	return nil
}

// GetBugDraft returns one private draft. It is intentionally a direct-id read;
// there is no broad draft listing in the normal knowledge/inbox surface.
func (s *FileTeamStore) GetBugDraft(ctx context.Context, teamID, id string) (BugDraft, error) {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.GetBugDraft(ctx, teamID, id)
	}
	if s.ledger != nil {
		entries, err := s.ledger.List(ctx, sourceLedgerScope(teamID), 500)
		if err != nil {
			return BugDraft{}, err
		}
		for _, item := range entries {
			if item.Kind != sourceLedgerBugDraftKind {
				continue
			}
			var draft BugDraft
			if err := json.Unmarshal([]byte(item.Body), &draft); err == nil && draft.ID == id {
				return draft, nil
			}
		}
		return BugDraft{}, fmt.Errorf("bug draft not found: %s", id)
	}
	s.corpus.mu.Lock()
	defer s.corpus.mu.Unlock()
	drafts := append([]BugDraft(nil), s.corpus.drafts[teamID]...)
	for _, draft := range drafts {
		if draft.ID == id {
			return draft, nil
		}
	}
	return BugDraft{}, fmt.Errorf("bug draft not found: %s", id)
}

// DeleteBugDraft removes a draft only after its corresponding bug-inbox entry
// has been durably appended.
func (s *FileTeamStore) DeleteBugDraft(ctx context.Context, teamID, id string) error {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.DeleteBugDraft(ctx, teamID, id)
	}
	if s.ledger != nil {
		return fmt.Errorf("source-ledger bug drafts are append-only; close %s through the swarm-manager work item", id)
	}
	s.corpus.mu.Lock()
	defer s.corpus.mu.Unlock()
	drafts := append([]BugDraft(nil), s.corpus.drafts[teamID]...)
	filtered := drafts[:0]
	found := false
	for _, draft := range drafts {
		if draft.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, draft)
	}
	if !found {
		return fmt.Errorf("bug draft not found: %s", id)
	}
	s.corpus.drafts[teamID] = filtered
	return nil
}

// --- Retention / Prune ---

// PruneResult reports how many mutable task items were removed during pruning.
type PruneResult struct {
	TasksRemoved int `json:"tasksRemoved"`
}

// PruneSharedState removes stale completed tasks according to the team's
// retention policy. Source Ledger team-corpus entries are append-only and are
// deliberately excluded.
func (s *FileTeamStore) PruneSharedState(ctx context.Context, teamID string) (*PruneResult, error) {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.PruneSharedState(ctx, teamID)
	}
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

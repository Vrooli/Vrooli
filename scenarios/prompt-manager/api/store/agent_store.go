package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

	// Seed default agent files (optional, user can delete/rename)
	if err := s.ensureDefaultAgentFiles(agentDir); err != nil {
		return fmt.Errorf("creating default agent files: %w", err)
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

// AgentFileEntry represents a file or directory within an agent folder.
type AgentFileEntry struct {
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
	Size  int64  `json:"size,omitempty"`
}

// ListFiles returns all files and directories under an agent folder.
func (s *FileAgentStore) ListFiles(ctx context.Context, agentID string) ([]AgentFileEntry, error) {
	if _, err := s.Get(ctx, agentID); err != nil {
		return nil, err
	}

	agentDir := filepath.Join(s.agentsDir(), agentID)
	var entries []AgentFileEntry

	err := filepath.WalkDir(agentDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == agentDir {
			return nil
		}
		if isHiddenFile(d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(agentDir, path)
		if err != nil {
			return err
		}

		info, err := d.Info()
		size := int64(0)
		if err == nil {
			size = info.Size()
		}

		entries = append(entries, AgentFileEntry{
			Path:  filepath.ToSlash(rel),
			IsDir: d.IsDir(),
			Size:  size,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listing agent files: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})

	return entries, nil
}

// ReadFile returns file content for a path within an agent folder.
func (s *FileAgentStore) ReadFile(ctx context.Context, agentID, relPath string) (string, error) {
	if _, err := s.Get(ctx, agentID); err != nil {
		return "", err
	}

	fullPath, _, err := s.resolveAgentPath(agentID, relPath)
	if err != nil {
		return "", err
	}

	return ReadContent(fullPath)
}

// WriteFile overwrites file content within an agent folder.
func (s *FileAgentStore) WriteFile(ctx context.Context, agentID, relPath, content string) error {
	if _, err := s.Get(ctx, agentID); err != nil {
		return err
	}
	if isReservedAgentFile(relPath) {
		return fmt.Errorf("cannot modify reserved file: %s", relPath)
	}

	fullPath, _, err := s.resolveAgentPath(agentID, relPath)
	if err != nil {
		return err
	}

	return WriteContent(fullPath, content)
}

// CreateFile creates a new file or directory within an agent folder.
func (s *FileAgentStore) CreateFile(ctx context.Context, agentID, relPath, content string, isDir bool) error {
	if _, err := s.Get(ctx, agentID); err != nil {
		return err
	}
	if isReservedAgentFile(relPath) {
		return fmt.Errorf("cannot create reserved file: %s", relPath)
	}

	fullPath, _, err := s.resolveAgentPath(agentID, relPath)
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

// RenameFile renames or moves a file within an agent folder.
func (s *FileAgentStore) RenameFile(ctx context.Context, agentID, fromPath, toPath string) error {
	if _, err := s.Get(ctx, agentID); err != nil {
		return err
	}
	if isReservedAgentFile(fromPath) || isReservedAgentFile(toPath) {
		return fmt.Errorf("cannot rename reserved file")
	}

	fromFull, _, err := s.resolveAgentPath(agentID, fromPath)
	if err != nil {
		return err
	}
	toFull, _, err := s.resolveAgentPath(agentID, toPath)
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

// DeleteFile deletes a file or directory within an agent folder.
func (s *FileAgentStore) DeleteFile(ctx context.Context, agentID, relPath string) error {
	if _, err := s.Get(ctx, agentID); err != nil {
		return err
	}
	if isReservedAgentFile(relPath) {
		return fmt.Errorf("cannot delete reserved file: %s", relPath)
	}

	fullPath, _, err := s.resolveAgentPath(agentID, relPath)
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

func (s *FileAgentStore) ensureDefaultAgentFiles(agentDir string) error {
	defaultFiles := map[string]string{
		"SOUL.md":   "# SOUL\n\nWho I am, how I communicate, and my boundaries.\n",
		"AGENTS.md": "# AGENTS\n\nOperating procedures for this agent.\n",
		"TOOLS.md":  "# TOOLS\n\nTooling notes and preferences.\n",
	}

	for name, content := range defaultFiles {
		path := filepath.Join(agentDir, name)
		if FileExists(path) {
			continue
		}
		if err := WriteContent(path, content); err != nil {
			return err
		}
	}

	return nil
}

func (s *FileAgentStore) resolveAgentPath(agentID, relPath string) (string, string, error) {
	if strings.TrimSpace(relPath) == "" {
		return "", "", fmt.Errorf("path is required")
	}

	clean := filepath.Clean(filepath.FromSlash(relPath))
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return "", "", fmt.Errorf("invalid path: %s", relPath)
	}

	agentDir := filepath.Join(s.agentsDir(), agentID)
	fullPath := filepath.Join(agentDir, clean)

	rel, err := filepath.Rel(agentDir, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", "", fmt.Errorf("invalid path: %s", relPath)
	}

	return fullPath, filepath.ToSlash(clean), nil
}

func isReservedAgentFile(path string) bool {
	return strings.EqualFold(filepath.Base(filepath.FromSlash(path)), "agent.json")
}

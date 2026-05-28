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
	configDir string
}

var defaultAgentMarkdownOrder = []string{"SOUL.md", "AGENTS.md", "TOOLS.md"}

var defaultAgentMarkdownFiles = []struct {
	name    string
	content string
}{
	{
		name:    "SOUL.md",
		content: "# SOUL\n\nWho I am, how I communicate, and my boundaries.\n",
	},
	{
		name: "AGENTS.md",
		content: "# AGENTS\n\nOperating procedures for this agent.\n\n## Skills\n\n" +
			"List skills as markdown references.\n\nExample:\n- e2e-testing: `prompt-manager skill read e2e-testing`\n",
	},
	{
		name:    "TOOLS.md",
		content: "# TOOLS\n\nTooling notes and preferences.\n",
	},
}

// NewFileAgentStore creates a new file-based agent store
func NewFileAgentStore(configDir string) *FileAgentStore {
	return &FileAgentStore{
		configDir: configDir,
	}
}

// agentsDir returns the path to the agents directory
func (s *FileAgentStore) agentsDir() string {
	return filepath.Join(s.configDir, "agents")
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
	if len(agent.FileOrder) == 0 {
		agent.FileOrder = append([]string{}, defaultAgentMarkdownOrder...)
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
	if updates.FileOrder != nil {
		agent.FileOrder = updates.FileOrder
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
	info, err := os.Stat(fromFull)
	if err != nil {
		return fmt.Errorf("stat %s: %w", fromPath, err)
	}
	isDir := info.IsDir()
	if FileExists(toFull) {
		return fmt.Errorf("target already exists: %s", toPath)
	}

	if err := os.MkdirAll(filepath.Dir(toFull), 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	if err := os.Rename(fromFull, toFull); err != nil {
		return err
	}

	if err := s.updateAgentFileOrderForRename(ctx, agentID, fromPath, toPath, isDir); err != nil {
		return err
	}

	return nil
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
	for _, entry := range defaultAgentMarkdownFiles {
		path := filepath.Join(agentDir, entry.name)
		if FileExists(path) {
			continue
		}
		if err := WriteContent(path, entry.content); err != nil {
			return err
		}
	}

	return nil
}

func (s *FileAgentStore) updateAgentFileOrderForRename(ctx context.Context, agentID, fromPath, toPath string, isDir bool) error {
	agent, err := s.Get(ctx, agentID)
	if err != nil {
		return err
	}

	updatedOrder, changed := updateFileOrderForRename(agent.FileOrder, fromPath, toPath, isDir)
	if !changed {
		return nil
	}

	agent.FileOrder = updatedOrder
	agent.UpdateTimestamp()

	agentPath := filepath.Join(s.agentsDir(), agentID, "agent.json")
	return SaveJSON(agentPath, agent)
}

func updateFileOrderForRename(fileOrder []string, fromPath, toPath string, isDir bool) ([]string, bool) {
	if len(fileOrder) == 0 {
		return fileOrder, false
	}

	fromNormalized := strings.Trim(filepath.ToSlash(fromPath), "/")
	toNormalized := strings.Trim(filepath.ToSlash(toPath), "/")
	fromLower := strings.ToLower(fromNormalized)

	updated := make([]string, 0, len(fileOrder))
	changed := false

	for _, entry := range fileOrder {
		entryNormalized := strings.Trim(filepath.ToSlash(entry), "/")
		entryLower := strings.ToLower(entryNormalized)

		switch {
		case entryLower == fromLower:
			changed = true
			if strings.HasSuffix(strings.ToLower(toNormalized), ".md") {
				updated = append(updated, toNormalized)
			}
		case isDir && strings.HasPrefix(entryLower, fromLower+"/"):
			changed = true
			suffix := entryNormalized[len(fromNormalized):]
			next := toNormalized + suffix
			if strings.HasSuffix(strings.ToLower(next), ".md") {
				updated = append(updated, next)
			}
		default:
			updated = append(updated, entry)
		}
	}

	if !changed {
		return fileOrder, false
	}

	return updated, true
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

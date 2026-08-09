package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

// FileAgentStore implements AgentStore using the file system
type FileAgentStore struct {
	configDir string
	roots     *filerouting.RoutedRoots
}

func NewRoutedFileAgentStore(roots *filerouting.RoutedRoots) *FileAgentStore {
	return &FileAgentStore{roots: roots}
}

func (s *FileAgentStore) forContext(ctx context.Context) (*FileAgentStore, error) {
	if s.roots == nil {
		return s, nil
	}
	configDir, err := s.roots.Pick(ctx, storage.ClassConfig)
	if err != nil {
		return nil, fmt.Errorf("resolve routed config root: %w", err)
	}
	copy := *s
	copy.configDir, copy.roots = configDir, nil
	return &copy, nil
}

func (s *FileAgentStore) recordWrite(ctx context.Context) {
	if s.roots != nil {
		s.roots.RecordWrite(ctx)
	}
}

// agentProseFile is the single standing-prose file an agent owns. It replaced
// SOUL.md + AGENTS.md + TOOLS.md, which were three files averaging seven lines
// each and whose AGENTS.md body was byte-identical across 21 of 24 agents.
const agentProseFile = "AGENT.md"

var defaultAgentMarkdownOrder = []string{agentProseFile}

var defaultAgentMarkdownFiles = []struct {
	name    string
	content string
}{
	{
		name: agentProseFile,
		content: "# SOUL\n\nWho I am, how I communicate, and my boundaries.\n\n" +
			"# TOOLS\n\nTooling notes and preferences.\n",
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
	var err error
	s, err = s.forContext(ctx)
	if err != nil {
		return nil, err
	}
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
	var err error
	s, err = s.forContext(ctx)
	if err != nil {
		return nil, err
	}
	return s.loadAgent(id)
}

// Create creates a new agent
func (s *FileAgentStore) Create(ctx context.Context, agent *Agent) error {
	original := s
	var err error
	s, err = s.forContext(ctx)
	if err != nil {
		return err
	}
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

	original.recordWrite(ctx)
	return nil
}

// Update updates an existing agent
func (s *FileAgentStore) Update(ctx context.Context, id string, updates *Agent) error {
	original := s
	var err error
	s, err = s.forContext(ctx)
	if err != nil {
		return err
	}
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
	if err := SaveJSON(agentPath, agent); err != nil {
		return err
	}
	original.recordWrite(ctx)
	return nil
}

// Delete removes an agent
func (s *FileAgentStore) Delete(ctx context.Context, id string) error {
	original := s
	var err error
	s, err = s.forContext(ctx)
	if err != nil {
		return err
	}
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}

	agentDir := filepath.Join(s.agentsDir(), id)
	if err := DeleteDirectory(agentDir); err != nil {
		return err
	}
	original.recordWrite(ctx)
	return nil
}

// loadAgent loads an agent from the agents directory
func (s *FileAgentStore) loadAgent(agentID string) (*Agent, error) {
	agentPath := filepath.Join(s.agentsDir(), agentID, "agent.json")
	return LoadJSON[Agent](agentPath)
}

// GetProse reads an agent's standing-prose content. Callers depend on the
// concept ("the agent's standing prose"), not on a filename: agents are the
// only indexed entity whose embedding source used to be a hardcoded file, while
// skills and topics already take a generic content string.
func (s *FileAgentStore) GetProse(ctx context.Context, agentID string) (string, error) {
	var err error
	s, err = s.forContext(ctx)
	if err != nil {
		return "", err
	}
	// Verify agent exists
	if _, err := s.Get(ctx, agentID); err != nil {
		return "", err
	}

	prosePath := filepath.Join(s.agentsDir(), agentID, agentProseFile)
	if !FileExists(prosePath) {
		return "", nil // Return empty string if the agent has no standing prose
	}

	return ReadContent(prosePath)
}

// SetProse writes an agent's standing-prose content.
func (s *FileAgentStore) SetProse(ctx context.Context, agentID string, content string) error {
	original := s
	var err error
	s, err = s.forContext(ctx)
	if err != nil {
		return err
	}
	// Verify agent exists
	if _, err := s.Get(ctx, agentID); err != nil {
		return err
	}

	prosePath := filepath.Join(s.agentsDir(), agentID, agentProseFile)
	if err := WriteContent(prosePath, content); err != nil {
		return err
	}
	original.recordWrite(ctx)
	return nil
}

// AgentFileEntry represents a file or directory within an agent folder.
type AgentFileEntry struct {
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
	Size  int64  `json:"size,omitempty"`
}

// ListFiles returns all files and directories under an agent folder.
func (s *FileAgentStore) ListFiles(ctx context.Context, agentID string) ([]AgentFileEntry, error) {
	var err error
	s, err = s.forContext(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := s.Get(ctx, agentID); err != nil {
		return nil, err
	}

	agentDir := filepath.Join(s.agentsDir(), agentID)
	var entries []AgentFileEntry

	err = filepath.WalkDir(agentDir, func(path string, d os.DirEntry, err error) error {
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
	var err error
	s, err = s.forContext(ctx)
	if err != nil {
		return "", err
	}
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
	original := s
	var err error
	s, err = s.forContext(ctx)
	if err != nil {
		return err
	}
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

	if err := WriteContent(fullPath, content); err != nil {
		return err
	}
	original.recordWrite(ctx)
	return nil
}

// CreateFile creates a new file or directory within an agent folder.
func (s *FileAgentStore) CreateFile(ctx context.Context, agentID, relPath, content string, isDir bool) error {
	original := s
	var err error
	s, err = s.forContext(ctx)
	if err != nil {
		return err
	}
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
		if err := os.MkdirAll(fullPath, 0o755); err != nil {
			return err
		}
		original.recordWrite(ctx)
		return nil
	}
	if err := WriteContent(fullPath, content); err != nil {
		return err
	}
	original.recordWrite(ctx)
	return nil
}

// RenameFile renames or moves a file within an agent folder.
func (s *FileAgentStore) RenameFile(ctx context.Context, agentID, fromPath, toPath string) error {
	original := s
	var err error
	s, err = s.forContext(ctx)
	if err != nil {
		return err
	}
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

	original.recordWrite(ctx)
	return nil
}

// DeleteFile deletes a file or directory within an agent folder.
func (s *FileAgentStore) DeleteFile(ctx context.Context, agentID, relPath string) error {
	original := s
	var err error
	s, err = s.forContext(ctx)
	if err != nil {
		return err
	}
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
		if err := DeleteDirectory(fullPath); err != nil {
			return err
		}
	} else if err := DeleteFile(fullPath); err != nil {
		return err
	}
	original.recordWrite(ctx)
	return nil
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

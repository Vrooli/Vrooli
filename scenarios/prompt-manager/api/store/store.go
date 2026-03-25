package store

import (
	"context"
	"log"
	"os"
	"path/filepath"
)

// FileStore is the combined implementation of all store interfaces
type FileStore struct {
	storeDir  string
	skills    *FileSkillStore
	agents    *FileAgentStore
	teams     *FileTeamStore
	topics    *FileTopicStore
	relations *FileRelationStore
	indexes   *FileIndexStore
}

// NewFileStore creates a new file-based store
func NewFileStore(storeDir string) *FileStore {
	// Ensure store directories exist
	ensureStoreDirectories(storeDir)

	// Create relation store first (needed by others)
	relationStore := NewFileRelationStore(storeDir)

	// Create entity stores
	skillStore := NewFileSkillStore(storeDir)
	teamStore := NewFileTeamStore(storeDir, relationStore)
	agentStore := NewFileAgentStore(storeDir)
	topicStore := NewFileTopicStore(storeDir)

	// Create index store (needs all entity stores)
	indexStore := NewFileIndexStore(storeDir, skillStore, agentStore, teamStore, topicStore, relationStore)

	return &FileStore{
		storeDir:  storeDir,
		skills:    skillStore,
		agents:    agentStore,
		teams:     teamStore,
		topics:    topicStore,
		relations: relationStore,
		indexes:   indexStore,
	}
}

// Skills returns the skill store
func (s *FileStore) Skills() SkillStore {
	return s.skills
}

// FileSkills returns the concrete file skill store (for adapters that need direct access)
func (s *FileStore) FileSkills() *FileSkillStore {
	return s.skills
}

// Agents returns the agent store
func (s *FileStore) Agents() AgentStore {
	return s.agents
}

// Teams returns the team store
func (s *FileStore) Teams() TeamStore {
	return s.teams
}

// Topics returns the topic store
func (s *FileStore) Topics() TopicStore {
	return s.topics
}

// FileTopics returns the concrete file topic store
func (s *FileStore) FileTopics() *FileTopicStore {
	return s.topics
}

// Relations returns the relation store
func (s *FileStore) Relations() RelationStore {
	return s.relations
}

// Indexes returns the index store
func (s *FileStore) Indexes() IndexStore {
	return s.indexes
}

// RegenerateIndexes regenerates all indexes
func (s *FileStore) RegenerateIndexes(ctx context.Context) error {
	return s.indexes.RegenerateAll(ctx)
}

// ensureStoreDirectories creates the required directory structure
func ensureStoreDirectories(storeDir string) {
	dirs := []string{
		filepath.Join(storeDir, "skills", "packs", "core"),
		filepath.Join(storeDir, "skills", "packs", "local"),
		filepath.Join(storeDir, "skills", "packs", "drafts"),
		filepath.Join(storeDir, "templates", "agent-files"),
		filepath.Join(storeDir, "agents"),
		filepath.Join(storeDir, "teams"),
		filepath.Join(storeDir, "topics"),
		filepath.Join(storeDir, "relations", "team-member"),
		filepath.Join(storeDir, "indexes"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Printf("Warning: failed to create directory %s: %v", dir, err)
		}
	}

	// Ensure pack order file exists
	packOrderPath := filepath.Join(storeDir, "skills", "_pack-order.json")
	if !FileExists(packOrderPath) {
		defaultOrder := &PackOrder{
			ActivePacks:   []string{"local", "core"},
			InactivePacks: []string{"drafts"},
		}
		if err := SaveJSON(packOrderPath, defaultOrder); err != nil {
			log.Printf("Warning: failed to create pack order file: %v", err)
		}
	}
}

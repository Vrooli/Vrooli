package store

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/vrooli/api-core/filerouting"

	// Registers the sqlite driver for every binary that links this package.
	// Without it only package main had the driver, so store and teams test
	// binaries failed on "sql: unknown driver sqlite".
	"prompt-manager/internal/paths"
	"prompt-manager/internal/sourceledger"

	_ "modernc.org/sqlite"
)

// FileStore is the combined implementation of all store interfaces.
//
// FileStore is the seam between the three storage classes (Config,
// RuntimeData, RuntimeCache) and the per-entity stores it composes. Each
// per-entity store receives the class root it actually writes to: the index
// store gets RuntimeCache; everything else (skills, actions,
// agents, teams, topics, relations) currently gets Config, with
// individual stores being retargeted to RuntimeData in later phases of the
// runtime/config storage split (see docs/concepts/STORAGE_CLASSES.md).
type FileStore struct {
	roots     paths.Roots
	skills    *FileSkillStore
	actions   *FileActionStore
	agents    *FileAgentStore
	teams     *FileTeamStore
	topics    *FileTopicStore
	relations *FileRelationStore
	indexes   *FileIndexStore
}

// NewFileStore creates a new file-based store rooted at the given Roots.
// It returns initialization failures to the caller rather than panicking so
// tests and command startup can report the underlying storage problem.
func NewFileStore(roots paths.Roots, routedRoots ...*filerouting.RoutedRoots) (*FileStore, error) {
	// Ensure on-disk layout exists for every class FileStore writes to.
	ensureStoreDirectories(roots)

	// Create relation store first (needed by others)
	relationStore := NewFileRelationStore(roots.Config)
	if len(routedRoots) > 0 && routedRoots[0] != nil {
		relationStore = NewRoutedFileRelationStore(routedRoots[0])
	}

	// Create entity stores
	skillStore := NewFileSkillStoreWithScenarioRootsAndCache(roots.Config, roots.RuntimeCache, roots.ScenariosDir)
	actionStore := NewFileActionStore(roots.Config)
	if len(routedRoots) > 0 && routedRoots[0] != nil {
		skillStore = NewRoutedFileSkillStore(routedRoots[0])
		skillStore.scenarioRoots = []string{roots.ScenariosDir}
		actionStore = NewRoutedFileActionStore(routedRoots[0])
	}
	teamStore := NewFileTeamStore(roots.Config, roots.RuntimeData, relationStore, routedRoots...)
	teamStore.repoRoot = roots.RepoRoot
	agentStore := NewFileAgentStore(roots.Config)
	topicStore := NewFileTopicStore(roots.Config)
	if len(routedRoots) > 0 && routedRoots[0] != nil {
		topicStore = NewRoutedFileTopicStore(routedRoots[0])
		agentStore = NewRoutedFileAgentStore(routedRoots[0])
	}

	// Index store writes derived caches under the RuntimeCache class root.
	indexStore := NewFileIndexStore(roots.RuntimeCache, skillStore, agentStore, teamStore, topicStore, relationStore)
	if len(routedRoots) > 0 && routedRoots[0] != nil {
		indexStore = NewRoutedFileIndexStore(routedRoots[0], skillStore, agentStore, teamStore, topicStore, relationStore)
	}

	return &FileStore{
		roots:     roots,
		skills:    skillStore,
		actions:   actionStore,
		agents:    agentStore,
		teams:     teamStore,
		topics:    topicStore,
		relations: relationStore,
		indexes:   indexStore,
	}, nil
}

// SetSourceLedger installs the shared source-ledger corpus client on the team
// store. Production startup calls this before wiring heartbeat handlers.
func (s *FileStore) SetSourceLedger(client *sourceledger.Client) {
	if s == nil || s.teams == nil {
		return
	}
	s.teams.SetSourceLedger(client)
}

// SetEventsEndpoint installs the shared event stream used for durable
// heartbeat-attempt records.
func (s *FileStore) SetEventsEndpoint(base string) {
	if s == nil || s.teams == nil {
		return
	}
	s.teams.SetEventsEndpoint(base)
}

// Roots returns the storage roots this FileStore was constructed against.
// Used by callers that need to compose paths against a class root (e.g.
// handlers that take a file-relative path from the operator and resolve it
// against Config or RuntimeData).
func (s *FileStore) Roots() paths.Roots {
	return s.roots
}

// Actions returns the Action store
func (s *FileStore) Actions() ActionStore {
	return s.actions
}

// FileActions returns the concrete file Action store (for adapters that need direct access)
func (s *FileStore) FileActions() *FileActionStore {
	return s.actions
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

// ensureStoreDirectories creates the required directory structure under each
// storage class root the FileStore writes to. Config subtree mirrors the
// authored repo layout; RuntimeCache holds the indexes directory.
func ensureStoreDirectories(roots paths.Roots) {
	configDirs := []string{
		filepath.Join(roots.Config, "skills", "packs", "core"),
		filepath.Join(roots.Config, "skills", "packs", "local"),
		filepath.Join(roots.Config, "skills", "packs", "drafts"),
		filepath.Join(roots.Config, "actions", "packs", "core"),
		filepath.Join(roots.Config, "actions", "packs", "local"),
		filepath.Join(roots.Config, "actions", "packs", "drafts"),
		filepath.Join(roots.Config, "templates", "agent-files"),
		filepath.Join(roots.Config, "agents"),
		filepath.Join(roots.Config, "teams"),
		filepath.Join(roots.Config, "topics"),
		filepath.Join(roots.Config, "relations", "team-member"),
	}
	runtimeDataDirs := []string{}
	cacheDirs := []string{
		filepath.Join(roots.RuntimeCache, "indexes"),
	}

	allDirs := append(configDirs, runtimeDataDirs...)
	allDirs = append(allDirs, cacheDirs...)
	for _, dir := range allDirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Printf("Warning: failed to create directory %s: %v", dir, err)
		}
	}

	// Ensure pack order files exist under the config root.
	packOrderPath := filepath.Join(roots.Config, "skills", "_pack-order.json")
	if !FileExists(packOrderPath) {
		defaultOrder := &PackOrder{
			ActivePacks:   []string{"local", "core"},
			InactivePacks: []string{"drafts"},
		}
		if err := SaveJSON(packOrderPath, defaultOrder); err != nil {
			log.Printf("Warning: failed to create pack order file: %v", err)
		}
	}

	actionPackOrderPath := filepath.Join(roots.Config, "actions", "_pack-order.json")
	if !FileExists(actionPackOrderPath) {
		defaultOrder := &PackOrder{
			ActivePacks:   []string{"local", "core"},
			InactivePacks: []string{"drafts"},
		}
		if err := SaveJSON(actionPackOrderPath, defaultOrder); err != nil {
			log.Printf("Warning: failed to create action pack order file: %v", err)
		}
	}
}

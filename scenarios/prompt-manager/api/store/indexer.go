package store

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

// FileIndexStore implements IndexStore using the file system. Indexes are
// derived cache artifacts (RuntimeCache class), reconstructable in full from
// the authored entity stores via RegenerateAll, so they live under the
// scenario's api-core/storage ClassCache root.
type FileIndexStore struct {
	cacheRoot     string
	skillStore    SkillStore
	agentStore    AgentStore
	teamStore     TeamStore
	topicStore    TopicStore
	relationStore RelationStore
	roots         *filerouting.RoutedRoots
}

// NewRoutedFileIndexStore routes derived cache writes per request.
func NewRoutedFileIndexStore(roots *filerouting.RoutedRoots, skillStore SkillStore, agentStore AgentStore, teamStore TeamStore, topicStore TopicStore, relationStore RelationStore) *FileIndexStore {
	return &FileIndexStore{skillStore: skillStore, agentStore: agentStore, teamStore: teamStore, topicStore: topicStore, relationStore: relationStore, roots: roots}
}

func (s *FileIndexStore) forContext(ctx context.Context) *FileIndexStore {
	if s.roots == nil {
		return s
	}
	cacheRoot, err := s.roots.Pick(ctx, storage.ClassCache)
	if err != nil {
		return s
	}
	copy := *s
	copy.cacheRoot = cacheRoot
	return &copy
}

// NewFileIndexStore creates a new file-based index store. cacheRoot is the
// runtime cache class root for this scenario (see paths.Roots).
func NewFileIndexStore(cacheRoot string, skillStore SkillStore, agentStore AgentStore, teamStore TeamStore, topicStore TopicStore, relationStore RelationStore) *FileIndexStore {
	return &FileIndexStore{
		cacheRoot:     cacheRoot,
		skillStore:    skillStore,
		agentStore:    agentStore,
		teamStore:     teamStore,
		topicStore:    topicStore,
		relationStore: relationStore,
	}
}

// indexesDir returns the path to the indexes directory under the cache root.
func (s *FileIndexStore) indexesDir() string {
	return filepath.Join(s.cacheRoot, "indexes")
}

// RegenerateAll regenerates all indexes from entity files
func (s *FileIndexStore) RegenerateAll(ctx context.Context) error {
	s = s.forContext(ctx)
	if err := s.RegenerateSkills(ctx); err != nil {
		return fmt.Errorf("regenerating skills index: %w", err)
	}
	if err := s.RegenerateAgents(ctx); err != nil {
		return fmt.Errorf("regenerating agents index: %w", err)
	}
	if err := s.RegenerateTeams(ctx); err != nil {
		return fmt.Errorf("regenerating teams index: %w", err)
	}
	if err := s.RegenerateTopics(ctx); err != nil {
		return fmt.Errorf("regenerating topics index: %w", err)
	}
	return nil
}

// RegenerateSkills regenerates the skills index
func (s *FileIndexStore) RegenerateSkills(ctx context.Context) error {
	s = s.forContext(ctx)
	skills, err := s.skillStore.List(ctx)
	if err != nil {
		return fmt.Errorf("listing skills: %w", err)
	}

	index := &SkillsIndex{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Items:       make([]SkillsIndexEntry, 0, len(skills)),
	}

	for _, skill := range skills {
		entry := SkillsIndexEntry{
			ID:     skill.ID,
			Pack:   skill.Pack,
			Name:   skill.Name,
			Modes:  skill.Modes,
			Tags:   skill.Tags,
			Status: skill.Status,
		}
		index.Items = append(index.Items, entry)
	}

	indexPath := filepath.Join(s.indexesDir(), "skills.index.json")
	err = SaveJSON(indexPath, index)
	if err == nil && s.roots != nil {
		s.roots.RecordWrite(ctx)
	}
	return err
}

// RegenerateAgents regenerates the agents index
func (s *FileIndexStore) RegenerateAgents(ctx context.Context) error {
	s = s.forContext(ctx)
	agents, err := s.agentStore.List(ctx)
	if err != nil {
		return fmt.Errorf("listing agents: %w", err)
	}

	index := &AgentsIndex{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Items:       make([]AgentsIndexEntry, 0, len(agents)),
	}

	for _, agent := range agents {
		entry := AgentsIndexEntry{
			ID:          agent.ID,
			DisplayName: agent.DisplayName,
			Status:      agent.Status,
		}
		index.Items = append(index.Items, entry)
	}

	indexPath := filepath.Join(s.indexesDir(), "agents.index.json")
	err = SaveJSON(indexPath, index)
	if err == nil && s.roots != nil {
		s.roots.RecordWrite(ctx)
	}
	return err
}

// RegenerateTeams regenerates the teams index
func (s *FileIndexStore) RegenerateTeams(ctx context.Context) error {
	s = s.forContext(ctx)
	teams, err := s.teamStore.List(ctx)
	if err != nil {
		return fmt.Errorf("listing teams: %w", err)
	}

	index := &TeamsIndex{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Items:       make([]TeamsIndexEntry, 0, len(teams)),
	}

	for _, team := range teams {
		memberCount := 0
		if s.relationStore != nil {
			members, err := s.relationStore.ListTeamMembers(ctx, team.ID)
			if err == nil {
				memberCount = len(members)
			}
		}

		entry := TeamsIndexEntry{
			ID:          team.ID,
			DisplayName: team.DisplayName,
			MemberCount: memberCount,
		}
		index.Items = append(index.Items, entry)
	}

	indexPath := filepath.Join(s.indexesDir(), "teams.index.json")
	err = SaveJSON(indexPath, index)
	if err == nil && s.roots != nil {
		s.roots.RecordWrite(ctx)
	}
	return err
}

// GetSkillsIndex returns the current skills index
func (s *FileIndexStore) GetSkillsIndex(ctx context.Context) (*SkillsIndex, error) {
	s = s.forContext(ctx)
	indexPath := filepath.Join(s.indexesDir(), "skills.index.json")
	if !FileExists(indexPath) {
		// Generate if doesn't exist
		if err := s.RegenerateSkills(ctx); err != nil {
			return nil, err
		}
	}
	return LoadJSON[SkillsIndex](indexPath)
}

// GetAgentsIndex returns the current agents index
func (s *FileIndexStore) GetAgentsIndex(ctx context.Context) (*AgentsIndex, error) {
	s = s.forContext(ctx)
	indexPath := filepath.Join(s.indexesDir(), "agents.index.json")
	if !FileExists(indexPath) {
		// Generate if doesn't exist
		if err := s.RegenerateAgents(ctx); err != nil {
			return nil, err
		}
	}
	return LoadJSON[AgentsIndex](indexPath)
}

// GetTeamsIndex returns the current teams index
func (s *FileIndexStore) GetTeamsIndex(ctx context.Context) (*TeamsIndex, error) {
	s = s.forContext(ctx)
	indexPath := filepath.Join(s.indexesDir(), "teams.index.json")
	if !FileExists(indexPath) {
		// Generate if doesn't exist
		if err := s.RegenerateTeams(ctx); err != nil {
			return nil, err
		}
	}
	return LoadJSON[TeamsIndex](indexPath)
}

// RegenerateTopics regenerates the topics index
func (s *FileIndexStore) RegenerateTopics(ctx context.Context) error {
	s = s.forContext(ctx)
	topics, err := s.topicStore.List(ctx)
	if err != nil {
		return fmt.Errorf("listing topics: %w", err)
	}

	index := &TopicsIndex{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Items:       make([]TopicsIndexEntry, 0, len(topics)),
	}

	for _, topic := range topics {
		entry := TopicsIndexEntry{
			ID:            topic.ID,
			Name:          topic.Name,
			ParentTopicID: topic.ParentTopicID,
			SkillCount:    len(topic.Skills),
			Status:        topic.Status,
		}
		index.Items = append(index.Items, entry)
	}

	indexPath := filepath.Join(s.indexesDir(), "topics.index.json")
	err = SaveJSON(indexPath, index)
	if err == nil && s.roots != nil {
		s.roots.RecordWrite(ctx)
	}
	return err
}

// GetTopicsIndex returns the current topics index
func (s *FileIndexStore) GetTopicsIndex(ctx context.Context) (*TopicsIndex, error) {
	s = s.forContext(ctx)
	indexPath := filepath.Join(s.indexesDir(), "topics.index.json")
	if !FileExists(indexPath) {
		if err := s.RegenerateTopics(ctx); err != nil {
			return nil, err
		}
	}
	return LoadJSON[TopicsIndex](indexPath)
}

package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// maxAncestorDepth guards against circular parent references.
const maxAncestorDepth = 20

// FileTopicStore implements TopicStore using the file system.
type FileTopicStore struct {
	storeDir string
}

// NewFileTopicStore creates a new file-based topic store.
func NewFileTopicStore(storeDir string) *FileTopicStore {
	return &FileTopicStore{
		storeDir: storeDir,
	}
}

func (s *FileTopicStore) topicsDir() string {
	return filepath.Join(s.storeDir, "topics")
}

// List returns all topics.
func (s *FileTopicStore) List(ctx context.Context) ([]Topic, error) {
	topicDirs, err := ListDirectories(s.topicsDir())
	if err != nil {
		return nil, fmt.Errorf("listing topic directories: %w", err)
	}

	var topics []Topic
	for _, topicID := range topicDirs {
		topic, err := s.loadTopic(topicID)
		if err != nil {
			continue // Skip malformed topics
		}
		topics = append(topics, *topic)
	}

	return topics, nil
}

// Get retrieves a topic by ID.
func (s *FileTopicStore) Get(ctx context.Context, id string) (*Topic, error) {
	return s.loadTopic(id)
}

// GetWithContent retrieves a topic with its markdown content.
func (s *FileTopicStore) GetWithContent(ctx context.Context, id string) (*Topic, string, error) {
	topic, err := s.loadTopic(id)
	if err != nil {
		return nil, "", err
	}

	contentPath := filepath.Join(s.topicsDir(), id, "TOPIC.md")
	if !FileExists(contentPath) {
		return topic, "", nil
	}

	content, err := ReadContent(contentPath)
	if err != nil {
		return topic, "", fmt.Errorf("reading topic content: %w", err)
	}

	return topic, content, nil
}

// Create creates a new topic.
func (s *FileTopicStore) Create(ctx context.Context, topic *Topic, content string) error {
	if _, err := s.Get(ctx, topic.ID); err == nil {
		return fmt.Errorf("topic already exists: %s", topic.ID)
	}

	// Validate parent exists if specified
	if topic.ParentTopicID != nil && *topic.ParentTopicID != "" {
		if _, err := s.Get(ctx, *topic.ParentTopicID); err != nil {
			return fmt.Errorf("parent topic not found: %s", *topic.ParentTopicID)
		}
	}

	topic.Kind = KindTopic
	topic.SchemaVersion = CurrentSchemaVersion
	if topic.Status == "" {
		topic.Status = StatusActive
	}
	topic.Timestamps = NewTimestamps()

	topicDir := filepath.Join(s.topicsDir(), topic.ID)
	if err := os.MkdirAll(topicDir, 0o755); err != nil {
		return fmt.Errorf("creating topic directory: %w", err)
	}

	if err := SaveJSON(filepath.Join(topicDir, "topic.json"), topic); err != nil {
		return fmt.Errorf("writing topic.json: %w", err)
	}

	if content != "" {
		if err := WriteContent(filepath.Join(topicDir, "TOPIC.md"), content); err != nil {
			return fmt.Errorf("writing TOPIC.md: %w", err)
		}
	}

	return nil
}

// Update updates an existing topic.
func (s *FileTopicStore) Update(ctx context.Context, id string, updates *Topic, content *string) error {
	topic, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	if updates.Name != "" {
		topic.Name = updates.Name
	}
	if updates.Description != "" {
		topic.Description = updates.Description
	}
	if updates.ParentTopicID != nil {
		parentID := *updates.ParentTopicID
		if parentID == "" {
			topic.ParentTopicID = nil
		} else {
			// Validate parent exists and is not self
			if parentID == id {
				return fmt.Errorf("topic cannot be its own parent")
			}
			if _, err := s.Get(ctx, parentID); err != nil {
				return fmt.Errorf("parent topic not found: %s", parentID)
			}
			// Check for cycles: walk ancestors of new parent to ensure we don't appear
			ancestors, err := s.GetAncestors(ctx, parentID)
			if err != nil {
				return fmt.Errorf("checking for cycles: %w", err)
			}
			for _, a := range ancestors {
				if a.ID == id {
					return fmt.Errorf("setting parent would create a cycle")
				}
			}
			topic.ParentTopicID = &parentID
		}
	}
	if updates.Skills != nil {
		topic.Skills = updates.Skills
	}
	if updates.Icon != "" {
		topic.Icon = updates.Icon
	}
	if updates.Status != "" {
		topic.Status = updates.Status
	}

	topic.UpdateTimestamp()

	topicPath := filepath.Join(s.topicsDir(), id, "topic.json")
	if err := SaveJSON(topicPath, topic); err != nil {
		return err
	}

	if content != nil {
		contentPath := filepath.Join(s.topicsDir(), id, "TOPIC.md")
		if err := WriteContent(contentPath, *content); err != nil {
			return fmt.Errorf("writing TOPIC.md: %w", err)
		}
	}

	return nil
}

// Delete removes a topic.
func (s *FileTopicStore) Delete(ctx context.Context, id string) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}

	topicDir := filepath.Join(s.topicsDir(), id)
	return DeleteDirectory(topicDir)
}

// GetAncestors returns the ancestor chain for a topic (parent, grandparent, etc.).
func (s *FileTopicStore) GetAncestors(ctx context.Context, id string) ([]Topic, error) {
	topic, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	var ancestors []Topic
	visited := map[string]bool{id: true}
	current := topic

	for depth := 0; depth < maxAncestorDepth; depth++ {
		if current.ParentTopicID == nil || *current.ParentTopicID == "" {
			break
		}

		parentID := *current.ParentTopicID
		if visited[parentID] {
			break // Cycle detected
		}
		visited[parentID] = true

		parent, err := s.loadTopic(parentID)
		if err != nil {
			break // Parent not found, stop walking
		}

		ancestors = append(ancestors, *parent)
		current = parent
	}

	return ancestors, nil
}

// GetChildren returns direct children of a topic.
func (s *FileTopicStore) GetChildren(ctx context.Context, id string) ([]Topic, error) {
	// Verify topic exists
	if _, err := s.Get(ctx, id); err != nil {
		return nil, err
	}

	allTopics, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	var children []Topic
	for _, t := range allTopics {
		if t.ParentTopicID != nil && *t.ParentTopicID == id {
			children = append(children, t)
		}
	}

	return children, nil
}

// AccumulateSkills returns deduplicated skills from a topic and all its ancestors.
func (s *FileTopicStore) AccumulateSkills(ctx context.Context, id string) ([]string, error) {
	topic, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	ancestors, err := s.GetAncestors(ctx, id)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var skills []string

	// Add this topic's skills first
	for _, skillID := range topic.Skills {
		if !seen[skillID] {
			seen[skillID] = true
			skills = append(skills, skillID)
		}
	}

	// Add ancestor skills
	for _, ancestor := range ancestors {
		for _, skillID := range ancestor.Skills {
			if !seen[skillID] {
				seen[skillID] = true
				skills = append(skills, skillID)
			}
		}
	}

	return skills, nil
}

func (s *FileTopicStore) loadTopic(topicID string) (*Topic, error) {
	topicPath := filepath.Join(s.topicsDir(), topicID, "topic.json")
	return LoadJSON[Topic](topicPath)
}

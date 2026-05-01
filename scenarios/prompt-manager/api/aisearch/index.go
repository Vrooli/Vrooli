package aisearch

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"sort"
	"strings"

	"prompt-manager/skills"
	"prompt-manager/store"
)

// IndexSkill indexes a single skill into the vector store.
// This is designed to be called asynchronously after skill CRUD operations.
func (s *Service) IndexSkill(ctx context.Context, skillID string) error {
	skill, folder, err := s.skillStore.FindByID(skillID)
	if err != nil {
		return fmt.Errorf("skill not found: %w", err)
	}

	// Load content
	_, filename := extractFolderAndFile(skill.File)
	content, err := s.skillStore.GetContent(folder, filename)
	if err != nil {
		return fmt.Errorf("failed to load content: %w", err)
	}

	// Compose text for embedding
	embeddingText := composeEmbeddingText(skill, content)

	// Generate embedding
	vector, err := s.embedder.Embed(ctx, embeddingText)
	if err != nil {
		return fmt.Errorf("embedding failed: %w", err)
	}

	// Build payload
	payload := map[string]interface{}{
		"skill_id":    skill.ID,
		"name":        skill.Name,
		"description": skill.Description,
		"folder":      folder,
		"tags":        skill.Tags,
		"modes":       skill.Modes,
	}

	// Upsert into vector store
	if err := s.vectorStore.Upsert(ctx, qdrantPointID(skill.ID), vector, payload); err != nil {
		return fmt.Errorf("upsert failed: %w", err)
	}

	log.Printf("[aisearch] Indexed skill: %s (%s)", skill.Name, skill.ID)
	return nil
}

// DeleteFromIndex removes a skill from the vector index.
func (s *Service) DeleteFromIndex(ctx context.Context, skillID string) error {
	if err := s.vectorStore.Delete(ctx, qdrantPointID(skillID)); err != nil {
		return fmt.Errorf("delete from index failed: %w", err)
	}
	log.Printf("[aisearch] Deleted skill from index: %s", skillID)
	return nil
}

// ReindexAll rebuilds the entire vector index from all skills.
func (s *Service) ReindexAll(ctx context.Context) (*ReindexResponse, error) {
	return s.reindexAllWithProgress(ctx, nil, nil)
}

func (s *Service) reindexAllWithProgress(
	ctx context.Context,
	progress func(indexed, skipped, errors int),
	setTotal func(total int),
) (*ReindexResponse, error) {
	// Ensure collection exists
	if err := s.vectorStore.EnsureCollection(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure collection: %w", err)
	}

	// Get all skills
	allSkills, err := s.skillStore.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get skills: %w", err)
	}
	if setTotal != nil {
		setTotal(len(allSkills))
	}

	var indexed, skipped, errors int

	for _, skill := range allSkills {
		if err := ctx.Err(); err != nil {
			return &ReindexResponse{
				Indexed: indexed,
				Skipped: skipped,
				Errors:  errors,
				Message: fmt.Sprintf("Indexed %d skills, skipped %d, errors %d", indexed, skipped, errors),
			}, err
		}

		// Extract folder from file path
		folder, filename := extractFolderAndFile(skill.File)
		if folder == "" {
			skipped++
			if progress != nil {
				progress(indexed, skipped, errors)
			}
			continue
		}

		// Load content
		content, err := s.skillStore.GetContent(folder, filename)
		if err != nil {
			log.Printf("[aisearch] Failed to load content for %s: %v", skill.ID, err)
			errors++
			if progress != nil {
				progress(indexed, skipped, errors)
			}
			continue
		}

		// Compose text for embedding
		embeddingText := composeEmbeddingText(&skill, content)

		// Generate embedding
		vector, err := s.embedder.Embed(ctx, embeddingText)
		if err != nil {
			log.Printf("[aisearch] Failed to embed %s: %v", skill.ID, err)
			errors++
			if progress != nil {
				progress(indexed, skipped, errors)
			}
			continue
		}

		// Build payload
		payload := map[string]interface{}{
			"skill_id":    skill.ID,
			"name":        skill.Name,
			"description": skill.Description,
			"folder":      folder,
			"tags":        skill.Tags,
			"modes":       skill.Modes,
		}

		// Upsert into vector store
		if err := s.vectorStore.Upsert(ctx, qdrantPointID(skill.ID), vector, payload); err != nil {
			log.Printf("[aisearch] Failed to upsert %s: %v", skill.ID, err)
			errors++
			if progress != nil {
				progress(indexed, skipped, errors)
			}
			continue
		}

		indexed++
		if progress != nil {
			progress(indexed, skipped, errors)
		}
	}

	// Index agents if configured
	if s.agentVectorStore != nil && s.agentStore != nil {
		if err := s.agentVectorStore.EnsureCollection(ctx); err != nil {
			log.Printf("[aisearch] Failed to ensure agent collection: %v", err)
		} else {
			agents, err := s.agentStore.List(ctx)
			if err != nil {
				log.Printf("[aisearch] Failed to list agents: %v", err)
			} else {
				if setTotal != nil {
					setTotal(len(allSkills) + len(agents))
				}
				for _, agent := range agents {
					if err := ctx.Err(); err != nil {
						break
					}
					if err := s.IndexAgent(ctx, agent.ID); err != nil {
						log.Printf("[aisearch] Failed to index agent %s: %v", agent.ID, err)
						errors++
					} else {
						indexed++
					}
					if progress != nil {
						progress(indexed, skipped, errors)
					}
				}
			}
		}
	}

	// Index teams if configured
	if s.teamVectorStore != nil && s.teamStore != nil {
		if err := s.teamVectorStore.EnsureCollection(ctx); err != nil {
			log.Printf("[aisearch] Failed to ensure team collection: %v", err)
		} else {
			teams, err := s.teamStore.List(ctx)
			if err != nil {
				log.Printf("[aisearch] Failed to list teams: %v", err)
			} else {
				if setTotal != nil {
					currentTotal := len(allSkills)
					if s.agentStore != nil {
						if agents, err := s.agentStore.List(ctx); err == nil {
							currentTotal += len(agents)
						}
					}
					setTotal(currentTotal + len(teams))
				}
				for _, team := range teams {
					if err := ctx.Err(); err != nil {
						break
					}
					if err := s.IndexTeam(ctx, team.ID); err != nil {
						log.Printf("[aisearch] Failed to index team %s: %v", team.ID, err)
						errors++
					} else {
						indexed++
					}
					if progress != nil {
						progress(indexed, skipped, errors)
					}
				}
			}
		}
	}

	// Index topics if configured
	if s.topicVectorStore != nil && s.topicStore != nil {
		if err := s.topicVectorStore.EnsureCollection(ctx); err != nil {
			log.Printf("[aisearch] Failed to ensure topic collection: %v", err)
		} else {
			topics, err := s.topicStore.List(ctx)
			if err != nil {
				log.Printf("[aisearch] Failed to list topics: %v", err)
			} else {
				for _, topic := range topics {
					if err := ctx.Err(); err != nil {
						break
					}
					if err := s.IndexTopic(ctx, topic.ID); err != nil {
						log.Printf("[aisearch] Failed to index topic %s: %v", topic.ID, err)
						errors++
					} else {
						indexed++
					}
					if progress != nil {
						progress(indexed, skipped, errors)
					}
				}
			}
		}
	}

	// Index Actions if configured
	if s.actionVectorStore != nil && s.actionStore != nil {
		if err := s.actionVectorStore.EnsureCollection(ctx); err != nil {
			log.Printf("[aisearch] Failed to ensure action collection: %v", err)
		} else {
			actions, err := s.actionStore.List(ctx)
			if err != nil {
				log.Printf("[aisearch] Failed to list actions: %v", err)
			} else {
				for _, action := range actions {
					if err := ctx.Err(); err != nil {
						break
					}
					if err := s.IndexAction(ctx, action.ID); err != nil {
						log.Printf("[aisearch] Failed to index action %s: %v", action.ID, err)
						errors++
					} else {
						indexed++
					}
					if progress != nil {
						progress(indexed, skipped, errors)
					}
				}
			}
		}
	}

	return &ReindexResponse{
		Indexed: indexed,
		Skipped: skipped,
		Errors:  errors,
		Message: fmt.Sprintf("Indexed %d entities, skipped %d, errors %d", indexed, skipped, errors),
	}, nil
}

// composeEmbeddingText creates a rich text representation for embedding.
func composeEmbeddingText(skill *skills.Metadata, content string) string {
	var parts []string

	// Name first for high relevance
	parts = append(parts, skill.Name)

	// Description
	if skill.Description != "" {
		parts = append(parts, skill.Description)
	}

	// Tags
	if len(skill.Tags) > 0 {
		parts = append(parts, "Tags: "+strings.Join(skill.Tags, ", "))
	}

	// Modes (categories)
	if len(skill.Modes) > 0 {
		parts = append(parts, "Categories: "+strings.Join(skill.Modes, " / "))
	}

	// Content (truncated to avoid token limits)
	if content != "" {
		truncated := content
		if len(truncated) > 2000 {
			truncated = truncated[:2000] + "..."
		}
		parts = append(parts, truncated)
	}

	return strings.Join(parts, "\n\n")
}

// extractFolderAndFile splits a file path like "local/skill.md" into folder and filename.
func extractFolderAndFile(file string) (folder, filename string) {
	parts := strings.SplitN(file, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", file
}

var qdrantNamespace = [16]byte{
	0x6b, 0xa7, 0xb8, 0x10,
	0x9d, 0xad, 0x11, 0xd1,
	0x80, 0xb4, 0x00, 0xc0,
	0x4f, 0xd4, 0x30, 0xc8,
}

func qdrantPointID(skillID string) string {
	name := strings.TrimSpace(skillID)
	if name == "" {
		name = "unknown"
	}
	return uuidV5(qdrantNamespace, "prompt-manager:"+name)
}

func uuidV5(namespace [16]byte, name string) string {
	hash := sha1.New()
	_, _ = hash.Write(namespace[:])
	_, _ = hash.Write([]byte(name))
	sum := hash.Sum(nil)

	var uuid [16]byte
	copy(uuid[:], sum[:16])

	// Set version (5) and RFC4122 variant bits.
	uuid[6] = (uuid[6] & 0x0f) | 0x50
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	hexStr := hex.EncodeToString(uuid[:])
	return hexStr[0:8] + "-" + hexStr[8:12] + "-" + hexStr[12:16] + "-" + hexStr[16:20] + "-" + hexStr[20:32]
}

// --- Agent indexing ---

func agentPointID(agentID string) string {
	name := strings.TrimSpace(agentID)
	if name == "" {
		name = "unknown"
	}
	return uuidV5(qdrantNamespace, "prompt-manager-agent:"+name)
}

// composeAgentEmbeddingText creates a rich text representation for agent embedding.
func composeAgentEmbeddingText(agent *store.Agent, soulContent string) string {
	var parts []string

	parts = append(parts, agent.DisplayName)

	if agent.Description != "" {
		parts = append(parts, agent.Description)
	}

	if len(agent.Tags) > 0 {
		parts = append(parts, "Tags: "+strings.Join(agent.Tags, ", "))
	}

	if agent.Status != "" {
		parts = append(parts, "Status: "+agent.Status)
	}

	if soulContent != "" {
		truncated := soulContent
		if len(truncated) > 2000 {
			truncated = truncated[:2000] + "..."
		}
		parts = append(parts, truncated)
	}

	return strings.Join(parts, "\n\n")
}

// IndexAgent indexes a single agent into the vector store.
func (s *Service) IndexAgent(ctx context.Context, agentID string) error {
	if s.agentVectorStore == nil || s.agentStore == nil {
		return nil
	}

	agent, err := s.agentStore.Get(ctx, agentID)
	if err != nil {
		return fmt.Errorf("agent not found: %w", err)
	}

	// Try to load SOUL.md content
	var soulContent string
	if soulReader, ok := s.agentStore.(AgentSoulReader); ok {
		content, err := soulReader.GetSoul(ctx, agentID)
		if err == nil {
			soulContent = content
		}
	}

	embeddingText := composeAgentEmbeddingText(agent, soulContent)

	vector, err := s.embedder.Embed(ctx, embeddingText)
	if err != nil {
		return fmt.Errorf("embedding failed: %w", err)
	}

	payload := map[string]interface{}{
		"agent_id":     agent.ID,
		"display_name": agent.DisplayName,
		"description":  agent.Description,
		"status":       agent.Status,
		"tags":         agent.Tags,
	}

	if err := s.agentVectorStore.Upsert(ctx, agentPointID(agent.ID), vector, payload); err != nil {
		return fmt.Errorf("upsert failed: %w", err)
	}

	log.Printf("[aisearch] Indexed agent: %s (%s)", agent.DisplayName, agent.ID)
	return nil
}

// DeleteAgentFromIndex removes an agent from the vector index.
func (s *Service) DeleteAgentFromIndex(ctx context.Context, agentID string) error {
	if s.agentVectorStore == nil {
		return nil
	}
	if err := s.agentVectorStore.Delete(ctx, agentPointID(agentID)); err != nil {
		return fmt.Errorf("delete from index failed: %w", err)
	}
	log.Printf("[aisearch] Deleted agent from index: %s", agentID)
	return nil
}

// --- Team indexing ---

func teamPointID(teamID string) string {
	name := strings.TrimSpace(teamID)
	if name == "" {
		name = "unknown"
	}
	return uuidV5(qdrantNamespace, "prompt-manager-team:"+name)
}

// composeTeamEmbeddingText creates a rich text representation for team embedding.
func composeTeamEmbeddingText(team *store.Team, memberNames []string) string {
	var parts []string

	parts = append(parts, team.DisplayName)

	if team.Mission != "" {
		parts = append(parts, team.Mission)
	}

	if len(memberNames) > 0 {
		parts = append(parts, "Members: "+strings.Join(memberNames, ", "))
	}

	return strings.Join(parts, "\n\n")
}

// IndexTeam indexes a single team into the vector store.
func (s *Service) IndexTeam(ctx context.Context, teamID string) error {
	if s.teamVectorStore == nil || s.teamStore == nil {
		return nil
	}

	team, err := s.teamStore.Get(ctx, teamID)
	if err != nil {
		return fmt.Errorf("team not found: %w", err)
	}

	// Get member names for richer embedding
	var memberNames []string
	if s.teamRelStore != nil {
		members, err := s.teamRelStore.ListTeamMembers(ctx, teamID)
		if err == nil {
			for _, m := range members {
				memberNames = append(memberNames, m.AgentID)
			}
		}
	}

	embeddingText := composeTeamEmbeddingText(team, memberNames)

	vector, err := s.embedder.Embed(ctx, embeddingText)
	if err != nil {
		return fmt.Errorf("embedding failed: %w", err)
	}

	memberCount := len(memberNames)
	payload := map[string]interface{}{
		"team_id":      team.ID,
		"display_name": team.DisplayName,
		"mission":      team.Mission,
		"enabled":      team.Enabled,
		"member_count": memberCount,
	}

	if err := s.teamVectorStore.Upsert(ctx, teamPointID(team.ID), vector, payload); err != nil {
		return fmt.Errorf("upsert failed: %w", err)
	}

	log.Printf("[aisearch] Indexed team: %s (%s)", team.DisplayName, team.ID)
	return nil
}

// DeleteTeamFromIndex removes a team from the vector index.
func (s *Service) DeleteTeamFromIndex(ctx context.Context, teamID string) error {
	if s.teamVectorStore == nil {
		return nil
	}
	if err := s.teamVectorStore.Delete(ctx, teamPointID(teamID)); err != nil {
		return fmt.Errorf("delete from index failed: %w", err)
	}
	log.Printf("[aisearch] Deleted team from index: %s", teamID)
	return nil
}

// --- Topic indexing ---

func topicPointID(topicID string) string {
	name := strings.TrimSpace(topicID)
	if name == "" {
		name = "unknown"
	}
	return uuidV5(qdrantNamespace, "prompt-manager-topic:"+name)
}

// --- Action indexing ---

func actionPointID(actionID string) string {
	name := strings.TrimSpace(actionID)
	if name == "" {
		name = "unknown"
	}
	return uuidV5(qdrantNamespace, "prompt-manager-action:"+name)
}

func composeActionEmbeddingText(action *store.Action) string {
	var parts []string
	parts = append(parts, action.Name)
	if action.Description != "" {
		parts = append(parts, action.Description)
	}
	if len(action.Tags) > 0 {
		parts = append(parts, "Tags: "+strings.Join(action.Tags, ", "))
	}
	if action.Owner.Type != "" || action.Owner.ID != "" {
		parts = append(parts, "Owner: "+strings.Trim(action.Owner.Type+":"+action.Owner.ID, ":"))
	}
	if len(action.Command.Argv) > 0 {
		parts = append(parts, "Command: "+strings.Join(action.Command.Argv, " "))
	}
	if len(action.Inputs) > 0 {
		inputs := make([]string, 0, len(action.Inputs))
		for name, input := range action.Inputs {
			inputs = append(inputs, name+":"+input.Type)
		}
		sort.Strings(inputs)
		parts = append(parts, "Inputs: "+strings.Join(inputs, ", "))
	}
	if len(action.Outputs) > 0 {
		outputs := make([]string, 0, len(action.Outputs))
		for name, output := range action.Outputs {
			outputs = append(outputs, name+":"+output.Type)
		}
		sort.Strings(outputs)
		parts = append(parts, "Outputs: "+strings.Join(outputs, ", "))
	}
	return strings.Join(parts, "\n\n")
}

// IndexAction indexes a single Action into the vector store.
func (s *Service) IndexAction(ctx context.Context, actionID string) error {
	if s.actionVectorStore == nil || s.actionStore == nil {
		return nil
	}
	action, err := s.actionStore.Get(ctx, actionID)
	if err != nil {
		return fmt.Errorf("action not found: %w", err)
	}
	embeddingText := composeActionEmbeddingText(action)
	vector, err := s.embedder.Embed(ctx, embeddingText)
	if err != nil {
		return fmt.Errorf("embedding failed: %w", err)
	}
	payload := map[string]interface{}{
		"action_id":   action.ID,
		"name":        action.Name,
		"description": action.Description,
		"status":      action.Status,
		"owner":       strings.Trim(action.Owner.Type+":"+action.Owner.ID, ":"),
		"command":     strings.Join(action.Command.Argv, " "),
		"tags":        action.Tags,
	}
	if err := s.actionVectorStore.Upsert(ctx, actionPointID(action.ID), vector, payload); err != nil {
		return fmt.Errorf("upsert failed: %w", err)
	}
	log.Printf("[aisearch] Indexed action: %s (%s)", action.Name, action.ID)
	return nil
}

// DeleteActionFromIndex removes an Action from the vector index.
func (s *Service) DeleteActionFromIndex(ctx context.Context, actionID string) error {
	if s.actionVectorStore == nil {
		return nil
	}
	if err := s.actionVectorStore.Delete(ctx, actionPointID(actionID)); err != nil {
		return fmt.Errorf("delete from action index failed: %w", err)
	}
	log.Printf("[aisearch] Deleted action from index: %s", actionID)
	return nil
}

// composeTopicEmbeddingText creates a rich text representation for topic embedding.
func composeTopicEmbeddingText(topic *store.Topic, content string) string {
	var parts []string

	parts = append(parts, topic.Name)

	if topic.Description != "" {
		parts = append(parts, topic.Description)
	}

	if len(topic.Skills) > 0 {
		parts = append(parts, "Skills: "+strings.Join(topic.Skills, ", "))
	}

	if content != "" {
		parts = append(parts, content)
	}

	return strings.Join(parts, "\n\n")
}

// IndexTopic indexes a single topic into the vector store.
func (s *Service) IndexTopic(ctx context.Context, topicID string) error {
	if s.topicVectorStore == nil || s.topicStore == nil {
		return nil
	}

	topic, content, err := s.topicStore.GetWithContent(ctx, topicID)
	if err != nil {
		return fmt.Errorf("topic not found: %w", err)
	}

	embeddingText := composeTopicEmbeddingText(topic, content)

	vector, err := s.embedder.Embed(ctx, embeddingText)
	if err != nil {
		return fmt.Errorf("embedding failed: %w", err)
	}

	payload := map[string]interface{}{
		"topic_id":    topic.ID,
		"name":        topic.Name,
		"description": topic.Description,
		"skills":      topic.Skills,
	}
	if topic.ParentTopicID != nil {
		payload["parent_topic_id"] = *topic.ParentTopicID
	}

	if err := s.topicVectorStore.Upsert(ctx, topicPointID(topic.ID), vector, payload); err != nil {
		return fmt.Errorf("upsert failed: %w", err)
	}

	log.Printf("[aisearch] Indexed topic: %s (%s)", topic.Name, topic.ID)
	return nil
}

// DeleteTopicFromIndex removes a topic from the vector index.
func (s *Service) DeleteTopicFromIndex(ctx context.Context, topicID string) error {
	if s.topicVectorStore == nil {
		return nil
	}
	if err := s.topicVectorStore.Delete(ctx, topicPointID(topicID)); err != nil {
		return fmt.Errorf("delete from index failed: %w", err)
	}
	log.Printf("[aisearch] Deleted topic from index: %s", topicID)
	return nil
}

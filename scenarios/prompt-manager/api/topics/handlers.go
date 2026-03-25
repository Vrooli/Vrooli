// Package topics provides HTTP handlers for topic CRUD and skill accumulation.
//
// DOC: docs/reference/api-endpoints.md#topics
package topics

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"prompt-manager/store"
	"prompt-manager/validation"

	"github.com/gorilla/mux"
)

// GraphInvalidator allows triggering graph index invalidation.
type GraphInvalidator interface {
	Invalidate()
}

// AITopicIndexer defines the interface for AI search index operations on topics.
type AITopicIndexer interface {
	IndexTopic(ctx context.Context, topicID string) error
	DeleteTopicFromIndex(ctx context.Context, topicID string) error
}

// TopicMatchFunc performs AI-powered topic matching and skill accumulation.
// Returns matched topics, accumulated skill IDs, search method, and error.
type TopicMatchFunc func(ctx context.Context, queries []string, limit int) ([]MatchedTopic, []string, string, error)

// Handlers provides HTTP handlers for topic operations.
type Handlers struct {
	topicStore       store.TopicStore
	indexStore       store.IndexStore
	graphInvalidator GraphInvalidator
	aiIndexer        AITopicIndexer
	topicMatchFn     TopicMatchFunc
}

// NewHandlers creates a new topics handler.
func NewHandlers(topicStore store.TopicStore, indexStore store.IndexStore) *Handlers {
	return &Handlers{
		topicStore: topicStore,
		indexStore: indexStore,
	}
}

// SetGraphInvalidator sets the graph invalidator.
func (h *Handlers) SetGraphInvalidator(inv GraphInvalidator) {
	h.graphInvalidator = inv
}

// SetAIIndexer sets the AI search indexer for async index updates.
func (h *Handlers) SetAIIndexer(indexer AITopicIndexer) {
	h.aiIndexer = indexer
}

// SetTopicMatchFn sets the function used by the Match handler for AI topic search.
func (h *Handlers) SetTopicMatchFn(fn TopicMatchFunc) {
	h.topicMatchFn = fn
}

func (h *Handlers) invalidateGraph() {
	if h.graphInvalidator != nil {
		h.graphInvalidator.Invalidate()
	}
}

func (h *Handlers) asyncIndexTopic(ctx context.Context, topicID string) {
	if h.aiIndexer == nil {
		return
	}
	go func() {
		if err := h.aiIndexer.IndexTopic(ctx, topicID); err != nil {
			log.Printf("[topics] AI index update failed for %s: %v", topicID, err)
		}
	}()
}

func (h *Handlers) asyncDeleteTopicFromIndex(ctx context.Context, topicID string) {
	if h.aiIndexer == nil {
		return
	}
	go func() {
		if err := h.aiIndexer.DeleteTopicFromIndex(ctx, topicID); err != nil {
			log.Printf("[topics] AI index delete failed for %s: %v", topicID, err)
		}
	}()
}

// List handles GET /topics - returns all topics.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	topics, err := h.topicStore.List(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	responses := make([]Response, 0, len(topics))
	for _, t := range topics {
		responses = append(responses, toResponse(&t))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(responses)
}

// Get handles GET /topics/{id} - returns a single topic.
func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	topic, err := h.topicStore.Get(ctx, id)
	if err != nil {
		http.Error(w, "Topic not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toResponse(topic))
}

// Create handles POST /topics - creates a new topic.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	id := req.ID
	if id == "" {
		id = validation.Slugify(req.Name)
	}

	topic := &store.Topic{
		ID:            id,
		Name:          req.Name,
		Description:   req.Description,
		ParentTopicID: req.ParentTopicID,
		Skills:        req.Skills,
		Icon:          req.Icon,
	}

	if err := h.topicStore.Create(ctx, topic, req.Content); err != nil {
		if isConflict(err) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.invalidateGraph()
	h.asyncIndexTopic(ctx, id)

	if err := h.indexStore.RegenerateTopics(ctx); err != nil {
		log.Printf("[topics] index regeneration failed: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toResponse(topic))
}

// Update handles PUT /topics/{id} - updates a topic.
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	updates := &store.Topic{}
	if req.Name != nil {
		updates.Name = *req.Name
	}
	if req.Description != nil {
		updates.Description = *req.Description
	}
	if req.ParentTopicID != nil {
		updates.ParentTopicID = req.ParentTopicID
	}
	if req.Skills != nil {
		updates.Skills = req.Skills
	}
	if req.Icon != nil {
		updates.Icon = *req.Icon
	}
	if req.Status != nil {
		updates.Status = *req.Status
	}

	if err := h.topicStore.Update(ctx, id, updates, req.Content); err != nil {
		if isNotFound(err) {
			http.Error(w, "Topic not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.invalidateGraph()
	h.asyncIndexTopic(ctx, id)

	if err := h.indexStore.RegenerateTopics(ctx); err != nil {
		log.Printf("[topics] index regeneration failed: %v", err)
	}

	topic, err := h.topicStore.Get(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toResponse(topic))
}

// Delete handles DELETE /topics/{id} - deletes a topic.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	if err := h.topicStore.Delete(ctx, id); err != nil {
		if isNotFound(err) {
			http.Error(w, "Topic not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.invalidateGraph()
	h.asyncDeleteTopicFromIndex(ctx, id)

	if err := h.indexStore.RegenerateTopics(ctx); err != nil {
		log.Printf("[topics] index regeneration failed: %v", err)
	}

	w.WriteHeader(http.StatusNoContent)
}

// AccumulatedSkills handles GET /topics/{id}/skills - returns deduplicated skills from a topic and its ancestors.
func (h *Handlers) AccumulatedSkills(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	topic, err := h.topicStore.Get(ctx, id)
	if err != nil {
		http.Error(w, "Topic not found", http.StatusNotFound)
		return
	}

	skills, err := h.topicStore.AccumulateSkills(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ancestors, err := h.topicStore.GetAncestors(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ancestry := make([]string, 0, len(ancestors))
	for _, a := range ancestors {
		ancestry = append(ancestry, a.ID)
	}

	resp := AccumulatedSkillsResponse{
		TopicID:  topic.ID,
		Ancestry: ancestry,
		Skills:   skills,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// Match handles POST /topics/match - AI search for topics, returns accumulated skills.
func (h *Handlers) Match(w http.ResponseWriter, r *http.Request) {
	var req MatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Queries) == 0 {
		http.Error(w, "queries is required", http.StatusBadRequest)
		return
	}

	if h.topicMatchFn == nil {
		resp := MatchResponse{
			Topics: []MatchedTopic{},
			Skills: []string{},
			Method: "none",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}

	matchedTopics, skills, method, err := h.topicMatchFn(r.Context(), req.Queries, limit)
	if err != nil {
		log.Printf("[topics] match error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := MatchResponse{
		Topics: matchedTopics,
		Skills: skills,
		Method: method,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func toResponse(t *store.Topic) Response {
	return Response{
		ID:            t.ID,
		Name:          t.Name,
		Description:   t.Description,
		ParentTopicID: t.ParentTopicID,
		Skills:        t.Skills,
		Icon:          t.Icon,
		Status:        t.Status,
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
	}
}

func isConflict(err error) bool {
	return err != nil && contains(err.Error(), "already exists")
}

func isNotFound(err error) bool {
	return err != nil && (contains(err.Error(), "not found") || contains(err.Error(), "no such file"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

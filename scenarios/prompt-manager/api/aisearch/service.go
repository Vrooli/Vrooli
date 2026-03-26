package aisearch

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"prompt-manager/search"
	"prompt-manager/skills"
	"prompt-manager/store"
)

// Service provides AI-powered search with graceful fallback to text search.
type Service struct {
	embedder      *Embedder
	vectorStore   *VectorStore
	skillStore    skills.SkillStore
	searchService *search.Service
	threshold     float64
	reindex       *reindexState

	// Multi-entity support
	agentVectorStore *VectorStore
	agentStore       AgentStoreReader
	agentSearchSvc   *search.AgentSearchService
	teamVectorStore  *VectorStore
	teamStore        TeamStoreReader
	teamRelStore     TeamRelReader
	teamSearchSvc    *search.TeamSearchService

	// Topic support
	topicVectorStore *VectorStore
	topicStore       TopicStoreReader

	// Budget configuration
	budgetConfig BudgetConfigProvider
}

// AgentStoreReader provides read access to agents for AI search.
type AgentStoreReader interface {
	List(ctx context.Context) ([]store.Agent, error)
	Get(ctx context.Context, id string) (*store.Agent, error)
}

// AgentSoulReader provides access to agent SOUL.md content.
type AgentSoulReader interface {
	GetSoul(ctx context.Context, agentID string) (string, error)
}

// TeamStoreReader provides read access to teams for AI search.
type TeamStoreReader interface {
	List(ctx context.Context) ([]store.Team, error)
	Get(ctx context.Context, id string) (*store.Team, error)
}

// TeamRelReader provides access to team member relations.
type TeamRelReader interface {
	ListTeamMembers(ctx context.Context, teamID string) ([]store.TeamMemberRelation, error)
}

// TopicStoreReader provides read access to topics for AI search.
type TopicStoreReader interface {
	List(ctx context.Context) ([]store.Topic, error)
	Get(ctx context.Context, id string) (*store.Topic, error)
	GetWithContent(ctx context.Context, id string) (*store.Topic, string, error)
	GetAncestors(ctx context.Context, id string) ([]store.Topic, error)
	AccumulateSkills(ctx context.Context, id string) ([]string, error)
}

// ComplexityBudgets maps complexity levels to character budgets for skill discovery.
var ComplexityBudgets = map[string]int{
	"minor":         4000,
	"moderate":      8000,
	"major":         12000,
	"architectural": 18000,
}

// ValidComplexity returns true if the given complexity string is recognized.
func ValidComplexity(c string) bool {
	_, ok := ComplexityBudgets[c]
	return ok
}

// SearchOptions controls optional output formatting for AI search responses.
type SearchOptions struct {
	Output      string
	Format      string
	RenderLimit int
}

type reindexState struct {
	mu         sync.Mutex
	running    bool
	canceled   bool
	startedAt  time.Time
	finishedAt time.Time
	indexed    int
	skipped    int
	errors     int
	total      int
	message    string
	lastError  string
	cancel     context.CancelFunc
}

// NewService creates a new AI search service.
func NewService(
	embedder *Embedder,
	vectorStore *VectorStore,
	skillStore skills.SkillStore,
	searchService *search.Service,
	threshold float64,
) *Service {
	if threshold <= 0 {
		threshold = 0.5
	}
	return &Service{
		embedder:      embedder,
		vectorStore:   vectorStore,
		skillStore:    skillStore,
		searchService: searchService,
		threshold:     threshold,
		reindex:       &reindexState{},
	}
}

// Search performs AI semantic search with fallback to text search.
func (s *Service) Search(ctx context.Context, query string, limit int) (*AISearchResponse, error) {
	if limit <= 0 {
		limit = 5
	}

	// Try AI search first
	if s.embedder == nil {
		return s.fallbackToTextSearch(ctx, query, limit)
	}
	vector, err := s.embedder.Embed(ctx, query)
	if err != nil {
		log.Printf("[aisearch] Embedding failed, falling back to text search: %v", err)
		return s.fallbackToTextSearch(ctx, query, limit)
	}

	results, err := s.vectorStore.Search(ctx, vector, limit, s.threshold)
	if err != nil {
		log.Printf("[aisearch] Vector search failed, falling back to text search: %v", err)
		return s.fallbackToTextSearch(ctx, query, limit)
	}

	// Convert to AI search results
	aiResults := make([]AISearchResult, 0, len(results))
	for _, r := range results {
		aiResults = append(aiResults, s.toAISearchResult(r))
	}

	return &AISearchResponse{
		Results: aiResults,
		Total:   len(aiResults),
		Query:   query,
		Method:  "ai",
		Output:  "results",
	}, nil
}

// SearchWithOptions performs AI search and optionally renders combined skill output.
func (s *Service) SearchWithOptions(ctx context.Context, query string, limit int, options SearchOptions) (*AISearchResponse, error) {
	resp, err := s.Search(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	output := normalizeSearchOutput(options.Output)
	resp.Output = output

	if !outputIncludesCombined(output) {
		return resp, nil
	}

	renderLimit := options.RenderLimit
	if renderLimit <= 0 || renderLimit > len(resp.Results) {
		renderLimit = len(resp.Results)
	}

	ids := make([]string, 0, renderLimit)
	for i := 0; i < renderLimit; i++ {
		ids = append(ids, resp.Results[i].ID)
	}

	responses := s.loadResponsesByIDs(ids)
	combined, normalizedFormat, err := skills.RenderCombined(responses, options.Format)
	if err != nil {
		return nil, err
	}

	resp.Combined = combined
	resp.SkillCount = len(responses)
	resp.TotalTokens = (len(combined) + 3) / 4
	resp.Format = normalizedFormat

	return resp, nil
}

// SearchMultiWithOptions performs multiple AI searches (one per query), merges and deduplicates results.
func (s *Service) SearchMultiWithOptions(ctx context.Context, queries []string, limit int, options SearchOptions) (*AISearchResponse, error) {
	if len(queries) == 0 {
		return nil, fmt.Errorf("at least one query is required")
	}
	if len(queries) == 1 {
		return s.SearchWithOptions(ctx, queries[0], limit, options)
	}

	// Search each query independently, collect all results
	seen := make(map[string]AISearchResult) // key: skill ID, value: best result
	method := "ai"

	for _, query := range queries {
		resp, err := s.Search(ctx, query, limit)
		if err != nil {
			log.Printf("[aisearch] Multi-query search failed for %q: %v", query, err)
			continue
		}
		if resp.Method == "text" {
			method = "text"
		}
		for _, r := range resp.Results {
			if existing, ok := seen[r.ID]; !ok || r.Score > existing.Score {
				seen[r.ID] = r
			}
		}
	}

	// Collect and sort by score descending
	results := make([]AISearchResult, 0, len(seen))
	for _, r := range seen {
		results = append(results, r)
	}
	sortAIResults(results)

	// Apply limit
	if len(results) > limit {
		results = results[:limit]
	}

	resp := &AISearchResponse{
		Results: results,
		Total:   len(results),
		Query:   strings.Join(queries, " | "),
		Method:  method,
		Output:  normalizeSearchOutput(options.Output),
	}

	// Render combined output if requested
	if outputIncludesCombined(resp.Output) {
		renderLimit := options.RenderLimit
		if renderLimit <= 0 || renderLimit > len(resp.Results) {
			renderLimit = len(resp.Results)
		}
		ids := make([]string, 0, renderLimit)
		for i := 0; i < renderLimit; i++ {
			ids = append(ids, resp.Results[i].ID)
		}
		responses := s.loadResponsesByIDs(ids)
		combined, normalizedFormat, err := skills.RenderCombined(responses, options.Format)
		if err != nil {
			return nil, err
		}
		resp.Combined = combined
		resp.SkillCount = len(responses)
		resp.TotalTokens = (len(combined) + 3) / 4
		resp.Format = normalizedFormat
	}

	return resp, nil
}

// sortAIResults sorts AI search results by score descending.
func sortAIResults(results []AISearchResult) {
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].Score > results[j-1].Score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
}

// SearchTopics performs AI semantic search for topics with graceful fallback.
func (s *Service) SearchTopics(ctx context.Context, query string, limit int) ([]SearchResult, string, error) {
	if limit <= 0 {
		limit = 5
	}
	if s.topicVectorStore == nil {
		return nil, "none", nil
	}

	vector, err := s.embedder.Embed(ctx, query)
	if err != nil {
		log.Printf("[aisearch] Topic embedding failed: %v", err)
		return nil, "none", nil
	}

	results, err := s.topicVectorStore.Search(ctx, vector, limit, s.threshold)
	if err != nil {
		log.Printf("[aisearch] Topic vector search failed: %v", err)
		return nil, "none", nil
	}

	return results, "ai", nil
}

// Discover performs unified topic + skill discovery.
// For each query: searches topics (accumulates their skills) and searches skills directly.
// Results are deduplicated, sorted by topic depth then search score, and annotated with content sizes.
func (s *Service) Discover(ctx context.Context, queries []string, complexity string, limit int) (*DiscoverResponse, error) {
	if len(queries) == 0 {
		return nil, fmt.Errorf("at least one query is required")
	}
	if limit <= 0 {
		limit = 10
	}

	type skillEntry struct {
		result DiscoverResult
	}
	seen := make(map[string]*skillEntry)
	topicNames := make(map[string]string) // cache topic ID → name
	method := "ai"

	// Step 1: Topic search per query
	for _, query := range queries {
		topicResults, topicMethod, err := s.SearchTopics(ctx, query, 3)
		if err != nil {
			log.Printf("[aisearch] Discover topic search failed for %q: %v", query, err)
			continue
		}
		if topicMethod == "none" {
			if method == "ai" {
				method = "mixed"
			}
		}

		for _, tr := range topicResults {
			topicID, _ := tr.Payload["topic_id"].(string)
			if topicID == "" || s.topicStore == nil {
				continue
			}

			// Resolve and cache topic name
			if _, cached := topicNames[topicID]; !cached {
				if t, nameErr := s.topicStore.Get(ctx, topicID); nameErr == nil && t != nil {
					topicNames[topicID] = t.Name
				}
			}

			// Get ancestors to compute depth
			ancestors, err := s.topicStore.GetAncestors(ctx, topicID)
			if err != nil {
				log.Printf("[aisearch] Discover get ancestors failed for topic %s: %v", topicID, err)
				continue
			}
			topicDepth := len(ancestors)

			// Accumulate skills from this topic + ancestors
			skillIDs, err := s.topicStore.AccumulateSkills(ctx, topicID)
			if err != nil {
				log.Printf("[aisearch] Discover accumulate skills failed for topic %s: %v", topicID, err)
				continue
			}

			for _, skillID := range skillIDs {
				if existing, exists := seen[skillID]; exists {
					// Keep the shallowest depth
					if existing.result.TopicDepth != nil && topicDepth < *existing.result.TopicDepth {
						d := topicDepth
						existing.result.TopicDepth = &d
						existing.result.TopicID = topicID
						existing.result.TopicName = topicNames[topicID]
					}
					continue
				}

				meta, folder, findErr := s.skillStore.FindByID(skillID)
				if findErr != nil || meta == nil {
					continue
				}

				d := topicDepth
				seen[skillID] = &skillEntry{
					result: DiscoverResult{
						ID:           meta.ID,
						Name:         meta.Name,
						Description:  meta.Description,
						Tags:         meta.Tags,
						Modes:        meta.Modes,
						Score:        tr.Score,
						ScorePercent: int(tr.Score * 100),
						Source:       "topic",
						TopicDepth:   &d,
						TopicID:      topicID,
						TopicName:    topicNames[topicID],
					},
				}

				// Load content size
				content, contentErr := s.skillStore.GetContent(folder, meta.File)
				if contentErr == nil {
					seen[skillID].result.ContentChars = len(content)
				}
			}
		}
	}

	// Step 2: Skill search per query
	for _, query := range queries {
		resp, err := s.Search(ctx, query, limit)
		if err != nil {
			log.Printf("[aisearch] Discover skill search failed for %q: %v", query, err)
			continue
		}
		if resp.Method == "text" {
			method = "mixed"
		}

		for _, r := range resp.Results {
			if existing, exists := seen[r.ID]; exists {
				// Topic source wins; for search dupes keep higher score
				if existing.result.Source == "search" && r.Score > existing.result.Score {
					existing.result.Score = r.Score
					existing.result.ScorePercent = r.ScorePercent
				}
				continue
			}

			entry := &skillEntry{
				result: DiscoverResult{
					ID:           r.ID,
					Name:         r.Name,
					Description:  r.Description,
					Tags:         r.Tags,
					Modes:        r.Modes,
					Score:        r.Score,
					ScorePercent: r.ScorePercent,
					Source:       "search",
				},
			}

			// Load content size
			meta, folder, findErr := s.skillStore.FindByID(r.ID)
			if findErr == nil && meta != nil {
				content, contentErr := s.skillStore.GetContent(folder, meta.File)
				if contentErr == nil {
					entry.result.ContentChars = len(content)
				}
			}

			seen[r.ID] = entry
		}
	}

	// Step 3: Sort - topic-sourced first (depth asc, score desc), then search-sourced (score desc)
	var topicResults, searchResults []DiscoverResult
	for _, entry := range seen {
		if entry.result.Source == "topic" {
			topicResults = append(topicResults, entry.result)
		} else {
			searchResults = append(searchResults, entry.result)
		}
	}

	sortDiscoverTopicResults(topicResults)
	sortDiscoverSearchResults(searchResults)

	results := append(topicResults, searchResults...)
	if len(results) > limit {
		results = results[:limit]
	}

	// Step 4: Build response
	totalChars := 0
	ids := make([]string, 0, len(results))
	for _, r := range results {
		totalChars += r.ContentChars
		ids = append(ids, r.ID)
	}

	readCommand := ""
	if len(ids) > 0 {
		readCommand = "prompt-manager skill read " + strings.Join(ids, " ")
	}

	resp := &DiscoverResponse{
		Results:           results,
		Total:             len(results),
		Query:             strings.Join(queries, " | "),
		Method:            method,
		TotalContentChars: totalChars,
		ReadCommand:       readCommand,
	}

	// Step 5: Budget calculation
	if complexity != "" {
		budgetChars := 0
		if s.budgetConfig != nil {
			if cfg, cfgErr := s.budgetConfig.Get(ctx); cfgErr == nil {
				budgetChars, _ = cfg.ForTier(complexity)
			}
		}
		if budgetChars == 0 {
			budgetChars = ComplexityBudgets[complexity]
		}
		if ok := budgetChars > 0; ok {
			resp.BudgetChars = budgetChars
			resp.Complexity = complexity

			switch {
			case totalChars == budgetChars:
				resp.BudgetStatus = "at"
			case totalChars < budgetChars:
				resp.BudgetStatus = "under"
			default:
				resp.BudgetStatus = "over"
				trimmedIDs := []string{}
				cumChars := 0
				for _, r := range results {
					if cumChars+r.ContentChars > budgetChars {
						break
					}
					cumChars += r.ContentChars
					trimmedIDs = append(trimmedIDs, r.ID)
				}
				if len(trimmedIDs) > 0 {
					resp.RecommendedReadCommand = "prompt-manager skill read " + strings.Join(trimmedIDs, " ")
				}
			}
		}
	}

	return resp, nil
}

func sortDiscoverTopicResults(results []DiscoverResult) {
	for i := 1; i < len(results); i++ {
		for j := i; j > 0; j-- {
			a, b := results[j], results[j-1]
			aDepth, bDepth := 0, 0
			if a.TopicDepth != nil {
				aDepth = *a.TopicDepth
			}
			if b.TopicDepth != nil {
				bDepth = *b.TopicDepth
			}
			if aDepth < bDepth || (aDepth == bDepth && a.Score > b.Score) {
				results[j], results[j-1] = results[j-1], results[j]
			} else {
				break
			}
		}
	}
}

func sortDiscoverSearchResults(results []DiscoverResult) {
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].Score > results[j-1].Score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
}

// fallbackToTextSearch uses the existing text search when AI is unavailable.
func (s *Service) fallbackToTextSearch(ctx context.Context, query string, limit int) (*AISearchResponse, error) {
	textResp, err := s.searchService.Search(search.SearchQuery{Query: query})
	if err != nil {
		return nil, fmt.Errorf("text search failed: %w", err)
	}

	// Limit results
	results := textResp.Results
	if len(results) > limit {
		results = results[:limit]
	}

	// Convert to AI search result format
	aiResults := make([]AISearchResult, 0, len(results))
	for _, r := range results {
		aiResults = append(aiResults, AISearchResult{
			ID:           r.ID,
			Name:         r.Name,
			Description:  r.Description,
			Folder:       r.Folder,
			Tags:         r.Tags,
			Modes:        r.Modes,
			Score:        r.Score / 10.0, // Normalize text search score to 0-1 range
			ScorePercent: int(r.Score * 10),
		})
	}

	return &AISearchResponse{
		Results: aiResults,
		Total:   len(aiResults),
		Query:   query,
		Method:  "text",
		Output:  "results",
	}, nil
}

// toAISearchResult converts a vector search result to an AI search result.
func (s *Service) toAISearchResult(r SearchResult) AISearchResult {
	payload := r.Payload

	// Extract fields from payload
	resultID := r.ID
	if payloadSkillID, ok := payload["skill_id"].(string); ok && payloadSkillID != "" {
		resultID = payloadSkillID
	}
	name, _ := payload["name"].(string)
	description, _ := payload["description"].(string)
	folder, _ := payload["folder"].(string)

	var tags []string
	if tagsRaw, ok := payload["tags"].([]interface{}); ok {
		for _, t := range tagsRaw {
			if ts, ok := t.(string); ok {
				tags = append(tags, ts)
			}
		}
	}

	var modes []string
	if modesRaw, ok := payload["modes"].([]interface{}); ok {
		for _, m := range modesRaw {
			if ms, ok := m.(string); ok {
				modes = append(modes, ms)
			}
		}
	}

	// Convert cosine similarity (0-1) to percentage
	scorePercent := int(r.Score * 100)
	if scorePercent > 100 {
		scorePercent = 100
	}

	return AISearchResult{
		ID:           resultID,
		Name:         name,
		Description:  description,
		Folder:       folder,
		Tags:         tags,
		Modes:        modes,
		Score:        r.Score,
		ScorePercent: scorePercent,
	}
}

func (s *Service) loadResponsesByIDs(ids []string) []skills.Response {
	responses := make([]skills.Response, 0, len(ids))
	for _, id := range ids {
		meta, folder, err := s.skillStore.FindByID(id)
		if err != nil || meta == nil {
			continue
		}

		content, err := s.skillStore.GetContent(folder, meta.File)
		if err != nil {
			continue
		}

		responses = append(responses, skills.Response{
			ID:           meta.ID,
			File:         meta.File,
			Name:         meta.Name,
			Description:  meta.Description,
			Content:      content,
			Modes:        meta.Modes,
			Tags:         meta.Tags,
			Icon:         meta.Icon,
			TargetToolID: meta.TargetToolID,
			Draft:        meta.Draft,
			Folder:       folder,
			CreatedAt:    meta.CreatedAt,
			UpdatedAt:    meta.UpdatedAt,
		})
	}

	return responses
}

func normalizeSearchOutput(output string) string {
	out := strings.ToLower(strings.TrimSpace(output))
	if out == "" {
		return "results"
	}
	if out == "results" || out == "combined" || out == "both" {
		return out
	}
	return ""
}

func outputIncludesCombined(output string) bool {
	return output == "combined" || output == "both"
}

// GetStatus returns the availability status of the AI search system.
func (s *Service) GetStatus(ctx context.Context) *AvailabilityStatus {
	ollamaAvailable := s.embedder.Available(ctx)
	qdrantAvailable := s.vectorStore.Available(ctx)

	var indexedCount int
	if qdrantAvailable {
		count, err := s.vectorStore.CountPoints(ctx)
		if err == nil {
			indexedCount = count
		}
	}

	available := ollamaAvailable && qdrantAvailable

	var message string
	if !available {
		var missing []string
		if !ollamaAvailable {
			missing = append(missing, "Ollama")
		}
		if !qdrantAvailable {
			missing = append(missing, "Qdrant")
		}
		message = fmt.Sprintf("AI search unavailable: %s not reachable", strings.Join(missing, " and "))
	}

	return &AvailabilityStatus{
		Available:    available,
		Ollama:       ollamaAvailable,
		Qdrant:       qdrantAvailable,
		IndexedCount: indexedCount,
		Message:      message,
	}
}

// Available returns true if AI search is available.
func (s *Service) Available(ctx context.Context) bool {
	return s.embedder.Available(ctx) && s.vectorStore.Available(ctx)
}

// StartReindex begins a singleton reindex job if one is not already running.
// Returns the current status and whether a new job was started.
func (s *Service) StartReindex() (ReindexStatus, bool) {
	s.reindex.mu.Lock()
	defer s.reindex.mu.Unlock()

	if s.reindex.running {
		return s.reindex.statusLocked(), false
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.reindex.running = true
	s.reindex.canceled = false
	s.reindex.startedAt = time.Now()
	s.reindex.finishedAt = time.Time{}
	s.reindex.indexed = 0
	s.reindex.skipped = 0
	s.reindex.errors = 0
	s.reindex.total = 0
	s.reindex.message = "Reindex started"
	s.reindex.lastError = ""
	s.reindex.cancel = cancel

	go s.runReindexJob(ctx)

	return s.reindex.statusLocked(), true
}

// CancelReindex requests cancellation of the active reindex job.
func (s *Service) CancelReindex() ReindexStatus {
	s.reindex.mu.Lock()
	defer s.reindex.mu.Unlock()

	if s.reindex.running && s.reindex.cancel != nil {
		s.reindex.canceled = true
		s.reindex.message = "Reindex cancel requested"
		s.reindex.cancel()
	}

	return s.reindex.statusLocked()
}

// ReindexStatus returns the current reindex status.
func (s *Service) ReindexStatus() ReindexStatus {
	s.reindex.mu.Lock()
	defer s.reindex.mu.Unlock()
	return s.reindex.statusLocked()
}

func (s *Service) runReindexJob(ctx context.Context) {
	resp, err := s.reindexAllWithProgress(ctx, func(indexed, skipped, errorsCount int) {
		s.reindex.mu.Lock()
		s.reindex.indexed = indexed
		s.reindex.skipped = skipped
		s.reindex.errors = errorsCount
		s.reindex.mu.Unlock()
	}, func(total int) {
		s.reindex.mu.Lock()
		s.reindex.total = total
		s.reindex.mu.Unlock()
	})

	s.reindex.mu.Lock()
	defer s.reindex.mu.Unlock()

	if resp != nil {
		s.reindex.indexed = resp.Indexed
		s.reindex.skipped = resp.Skipped
		s.reindex.errors = resp.Errors
		s.reindex.message = resp.Message
	}

	if err != nil {
		if errors.Is(err, context.Canceled) {
			s.reindex.canceled = true
			if s.reindex.message == "" {
				s.reindex.message = "Reindex canceled"
			}
		} else {
			s.reindex.lastError = err.Error()
		}
	}

	s.reindex.running = false
	s.reindex.finishedAt = time.Now()
	s.reindex.cancel = nil
}

func formatReindexTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// NeedsReindex compares indexed count against on-disk entity counts across all collections.
// Returns (needsReindex, totalIndexedCount, totalDiskCount, error).
func (s *Service) NeedsReindex(ctx context.Context) (bool, int, int, error) {
	indexedCount, err := s.vectorStore.CountPoints(ctx)
	if err != nil {
		return false, 0, 0, err
	}
	allSkills, err := s.skillStore.GetAll()
	if err != nil {
		return false, 0, 0, err
	}
	diskCount := len(allSkills)

	totalIndexed := indexedCount
	totalDisk := diskCount

	// Check agent collection
	if s.agentVectorStore != nil && s.agentStore != nil {
		agentIndexed, err := s.agentVectorStore.CountPoints(ctx)
		if err == nil {
			agents, err := s.agentStore.List(ctx)
			if err == nil {
				totalIndexed += agentIndexed
				totalDisk += len(agents)
			}
		}
	}

	// Check team collection
	if s.teamVectorStore != nil && s.teamStore != nil {
		teamIndexed, err := s.teamVectorStore.CountPoints(ctx)
		if err == nil {
			teams, err := s.teamStore.List(ctx)
			if err == nil {
				totalIndexed += teamIndexed
				totalDisk += len(teams)
			}
		}
	}

	// Check topic collection
	if s.topicVectorStore != nil && s.topicStore != nil {
		topicIndexed, err := s.topicVectorStore.CountPoints(ctx)
		if err == nil {
			topics, err := s.topicStore.List(ctx)
			if err == nil {
				totalIndexed += topicIndexed
				totalDisk += len(topics)
			}
		}
	}

	return totalIndexed != totalDisk, totalIndexed, totalDisk, nil
}

// StartPeriodicSync runs a background goroutine that periodically checks for
// index staleness and triggers a reindex when counts diverge.
func (s *Service) StartPeriodicSync(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				needs, indexed, disk, err := s.NeedsReindex(ctx)
				if err != nil {
					log.Printf("[aisearch] Periodic sync staleness check failed: %v", err)
					continue
				}
				if needs {
					log.Printf("[aisearch] Periodic sync: index out of sync (indexed=%d, on-disk=%d), reindexing...", indexed, disk)
					s.StartReindex()
				}
			}
		}
	}()
}

// SetAgentSearch configures agent AI search support.
func (s *Service) SetAgentSearch(vectorStore *VectorStore, agentStore AgentStoreReader, searchSvc *search.AgentSearchService) {
	s.agentVectorStore = vectorStore
	s.agentStore = agentStore
	s.agentSearchSvc = searchSvc
}

// SetTeamSearch configures team AI search support.
func (s *Service) SetTeamSearch(vectorStore *VectorStore, teamStore TeamStoreReader, relStore TeamRelReader, searchSvc *search.TeamSearchService) {
	s.teamVectorStore = vectorStore
	s.teamStore = teamStore
	s.teamRelStore = relStore
	s.teamSearchSvc = searchSvc
}

// SetTopicSearch configures topic AI search support.
func (s *Service) SetTopicSearch(vectorStore *VectorStore, topicStore TopicStoreReader) {
	s.topicVectorStore = vectorStore
	s.topicStore = topicStore
}

// SetBudgetConfig sets the budget configuration provider.
func (s *Service) SetBudgetConfig(p BudgetConfigProvider) {
	s.budgetConfig = p
}

// SearchAgents performs AI semantic search for agents with fallback to text search.
func (s *Service) SearchAgents(ctx context.Context, query string, limit int) (*AIAgentSearchResponse, error) {
	if limit <= 0 {
		limit = 5
	}

	if s.agentVectorStore == nil {
		return s.fallbackToAgentTextSearch(ctx, query, limit)
	}

	vector, err := s.embedder.Embed(ctx, query)
	if err != nil {
		log.Printf("[aisearch] Agent embedding failed, falling back to text search: %v", err)
		return s.fallbackToAgentTextSearch(ctx, query, limit)
	}

	results, err := s.agentVectorStore.Search(ctx, vector, limit, s.threshold)
	if err != nil {
		log.Printf("[aisearch] Agent vector search failed, falling back to text search: %v", err)
		return s.fallbackToAgentTextSearch(ctx, query, limit)
	}

	aiResults := make([]AIAgentSearchResult, 0, len(results))
	for _, r := range results {
		aiResults = append(aiResults, toAIAgentSearchResult(r))
	}

	return &AIAgentSearchResponse{
		Results: aiResults,
		Total:   len(aiResults),
		Query:   query,
		Method:  "ai",
	}, nil
}

// SearchTeams performs AI semantic search for teams with fallback to text search.
func (s *Service) SearchTeams(ctx context.Context, query string, limit int) (*AITeamSearchResponse, error) {
	if limit <= 0 {
		limit = 5
	}

	if s.teamVectorStore == nil {
		return s.fallbackToTeamTextSearch(ctx, query, limit)
	}

	vector, err := s.embedder.Embed(ctx, query)
	if err != nil {
		log.Printf("[aisearch] Team embedding failed, falling back to text search: %v", err)
		return s.fallbackToTeamTextSearch(ctx, query, limit)
	}

	results, err := s.teamVectorStore.Search(ctx, vector, limit, s.threshold)
	if err != nil {
		log.Printf("[aisearch] Team vector search failed, falling back to text search: %v", err)
		return s.fallbackToTeamTextSearch(ctx, query, limit)
	}

	aiResults := make([]AITeamSearchResult, 0, len(results))
	for _, r := range results {
		aiResults = append(aiResults, toAITeamSearchResult(r))
	}

	return &AITeamSearchResponse{
		Results: aiResults,
		Total:   len(aiResults),
		Query:   query,
		Method:  "ai",
	}, nil
}

func (s *Service) fallbackToAgentTextSearch(ctx context.Context, query string, limit int) (*AIAgentSearchResponse, error) {
	if s.agentSearchSvc == nil {
		return &AIAgentSearchResponse{
			Results: []AIAgentSearchResult{},
			Total:   0,
			Query:   query,
			Method:  "text",
		}, nil
	}

	textResp, err := s.agentSearchSvc.Search(ctx, search.AgentSearchQuery{Query: query})
	if err != nil {
		return nil, fmt.Errorf("agent text search failed: %w", err)
	}

	results := textResp.Results
	if len(results) > limit {
		results = results[:limit]
	}

	aiResults := make([]AIAgentSearchResult, 0, len(results))
	for _, r := range results {
		aiResults = append(aiResults, AIAgentSearchResult{
			ID:           r.ID,
			DisplayName:  r.DisplayName,
			Description:  r.Description,
			Status:       r.Status,
			Tags:         r.Tags,
			Score:        r.Score / 10.0,
			ScorePercent: int(r.Score * 10),
		})
	}

	return &AIAgentSearchResponse{
		Results: aiResults,
		Total:   len(aiResults),
		Query:   query,
		Method:  "text",
	}, nil
}

func (s *Service) fallbackToTeamTextSearch(ctx context.Context, query string, limit int) (*AITeamSearchResponse, error) {
	if s.teamSearchSvc == nil {
		return &AITeamSearchResponse{
			Results: []AITeamSearchResult{},
			Total:   0,
			Query:   query,
			Method:  "text",
		}, nil
	}

	textResp, err := s.teamSearchSvc.Search(ctx, search.TeamSearchQuery{Query: query})
	if err != nil {
		return nil, fmt.Errorf("team text search failed: %w", err)
	}

	results := textResp.Results
	if len(results) > limit {
		results = results[:limit]
	}

	aiResults := make([]AITeamSearchResult, 0, len(results))
	for _, r := range results {
		aiResults = append(aiResults, AITeamSearchResult{
			ID:           r.ID,
			DisplayName:  r.DisplayName,
			Mission:      r.Mission,
			Enabled:      r.Enabled,
			MemberCount:  r.MemberCount,
			Score:        r.Score / 10.0,
			ScorePercent: int(r.Score * 10),
		})
	}

	return &AITeamSearchResponse{
		Results: aiResults,
		Total:   len(aiResults),
		Query:   query,
		Method:  "text",
	}, nil
}

func toAIAgentSearchResult(r SearchResult) AIAgentSearchResult {
	payload := r.Payload
	resultID := r.ID
	if pid, ok := payload["agent_id"].(string); ok && pid != "" {
		resultID = pid
	}
	displayName, _ := payload["display_name"].(string)
	description, _ := payload["description"].(string)
	status, _ := payload["status"].(string)

	var tags []string
	if tagsRaw, ok := payload["tags"].([]interface{}); ok {
		for _, t := range tagsRaw {
			if ts, ok := t.(string); ok {
				tags = append(tags, ts)
			}
		}
	}

	scorePercent := int(r.Score * 100)
	if scorePercent > 100 {
		scorePercent = 100
	}

	return AIAgentSearchResult{
		ID:           resultID,
		DisplayName:  displayName,
		Description:  description,
		Status:       status,
		Tags:         tags,
		Score:        r.Score,
		ScorePercent: scorePercent,
	}
}

func toAITeamSearchResult(r SearchResult) AITeamSearchResult {
	payload := r.Payload
	resultID := r.ID
	if pid, ok := payload["team_id"].(string); ok && pid != "" {
		resultID = pid
	}
	displayName, _ := payload["display_name"].(string)
	mission, _ := payload["mission"].(string)
	enabled, _ := payload["enabled"].(bool)
	memberCount := 0
	if mc, ok := payload["member_count"].(float64); ok {
		memberCount = int(mc)
	}

	scorePercent := int(r.Score * 100)
	if scorePercent > 100 {
		scorePercent = 100
	}

	return AITeamSearchResult{
		ID:           resultID,
		DisplayName:  displayName,
		Mission:      mission,
		Enabled:      enabled,
		MemberCount:  memberCount,
		Score:        r.Score,
		ScorePercent: scorePercent,
	}
}

func (rs *reindexState) statusLocked() ReindexStatus {
	status := ReindexStatus{
		Running:    rs.running,
		StartedAt:  formatReindexTime(rs.startedAt),
		FinishedAt: formatReindexTime(rs.finishedAt),
		Indexed:    rs.indexed,
		Skipped:    rs.skipped,
		Errors:     rs.errors,
		Total:      rs.total,
		Message:    rs.message,
		Canceled:   rs.canceled,
	}
	if rs.lastError != "" {
		status.Error = rs.lastError
	}
	return status
}

package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/rs/cors"
)

// Data models
type API struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Provider         string    `json:"provider"`
	Description      string    `json:"description"`
	BaseURL          string    `json:"base_url"`
	DocumentationURL string    `json:"documentation_url"`
	PricingURL       string    `json:"pricing_url"`
	Category         string    `json:"category"`
	Status           string    `json:"status"`
	AuthType         string    `json:"auth_type"`
	Tags             []string  `json:"tags"`
	Capabilities     []string  `json:"capabilities"`
	Version          string    `json:"version"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	LastRefreshed    time.Time `json:"last_refreshed"`
	SourceURL        string    `json:"source_url"`
}

type Note struct {
	ID        string    `json:"id"`
	APIID     string    `json:"api_id"`
	Content   string    `json:"content"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
}

type PricingTier struct {
	ID               string  `json:"id"`
	APIID            string  `json:"api_id"`
	Name             string  `json:"name"`
	PricePerRequest  float64 `json:"price_per_request"`
	PricePerMB       float64 `json:"price_per_mb"`
	MonthlyCost      float64 `json:"monthly_cost"`
	FreeTierRequests int     `json:"free_tier_requests"`
}

type APIVersion struct {
	ID              string    `json:"id"`
	APIID           string    `json:"api_id"`
	Version         string    `json:"version"`
	ChangeSummary   string    `json:"change_summary"`
	BreakingChanges bool      `json:"breaking_changes"`
	CreatedAt       time.Time `json:"created_at"`
}

type SearchFilters struct {
	Configured *bool    `json:"configured,omitempty"`
	MaxPrice   *float64 `json:"max_price,omitempty"`
	Categories []string `json:"categories,omitempty"`
}

type SearchRequest struct {
	Query      string         `json:"query"`
	Limit      int            `json:"limit"`
	Filters    *SearchFilters `json:"filters,omitempty"`
	Configured *bool          `json:"configured,omitempty"`
	MaxPrice   *float64       `json:"max_price,omitempty"`
	Categories []string       `json:"categories,omitempty"`
}

type SearchParams struct {
	Query             string
	Limit             int
	RequireConfigured bool
	ConfiguredValue   bool
	MaxPrice          *float64
	Categories        []string
}

type SearchResult struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Provider       string   `json:"provider"`
	Description    string   `json:"description"`
	RelevanceScore float64  `json:"relevance_score"`
	Configured     bool     `json:"configured"`
	PricingSummary string   `json:"pricing_summary"`
	Category       string   `json:"category"`
	MinPrice       *float64 `json:"min_price,omitempty"`
}

type SemanticSearchClient struct {
	QdrantURL        string
	QdrantAPIKey     string
	QdrantCollection string
	OllamaURL        string
	OllamaModel      string
	HTTPClient       *http.Client
}

type ResearchRequest struct {
	Capability   string                 `json:"capability"`
	Requirements map[string]interface{} `json:"requirements"`
}

type ResearchResponse struct {
	ResearchID    string `json:"research_id"`
	Status        string `json:"status"`
	EstimatedTime int    `json:"estimated_time"`
}

// Integration Snippet models
type IntegrationSnippet struct {
	ID                   string                 `json:"id"`
	APIID                string                 `json:"api_id"`
	Title                string                 `json:"title"`
	Description          string                 `json:"description"`
	Language             string                 `json:"language"`
	Framework            string                 `json:"framework,omitempty"`
	Code                 string                 `json:"code"`
	Dependencies         map[string]interface{} `json:"dependencies,omitempty"`
	EnvironmentVariables []string               `json:"environment_variables,omitempty"`
	Prerequisites        string                 `json:"prerequisites,omitempty"`
	SnippetType          string                 `json:"snippet_type"`
	Tags                 []string               `json:"tags,omitempty"`
	Tested               bool                   `json:"tested"`
	Official             bool                   `json:"official"`
	CommunityVerified    bool                   `json:"community_verified"`
	UsageCount           int                    `json:"usage_count"`
	HelpfulCount         int                    `json:"helpful_count"`
	NotHelpfulCount      int                    `json:"not_helpful_count"`
	EndpointID           *string                `json:"endpoint_id,omitempty"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
	CreatedBy            string                 `json:"created_by"`
	SourceURL            string                 `json:"source_url,omitempty"`
	Version              string                 `json:"version,omitempty"`
	LastVerified         *time.Time             `json:"last_verified,omitempty"`

	// Additional fields from JOIN
	APIName     string `json:"api_name,omitempty"`
	APIProvider string `json:"api_provider,omitempty"`
}

type IntegrationRecipe struct {
	ID                   string                   `json:"id"`
	APIID                string                   `json:"api_id"`
	Name                 string                   `json:"name"`
	Description          string                   `json:"description"`
	UseCase              string                   `json:"use_case"`
	Steps                []map[string]interface{} `json:"steps"`
	Prerequisites        map[string]interface{}   `json:"prerequisites,omitempty"`
	ExpectedOutcome      string                   `json:"expected_outcome"`
	EstimatedTimeMinutes int                      `json:"estimated_time_minutes"`
	DifficultyLevel      string                   `json:"difficulty_level"`
	RelatedAPIIDs        []string                 `json:"related_api_ids,omitempty"`
	TimesUsed            int                      `json:"times_used"`
	SuccessRate          float64                  `json:"success_rate"`
	Rating               float64                  `json:"rating"`
	RatingCount          int                      `json:"rating_count"`
	CreatedAt            time.Time                `json:"created_at"`
	UpdatedAt            time.Time                `json:"updated_at"`
	CreatedBy            string                   `json:"created_by"`
	LastUpdatedBy        string                   `json:"last_updated_by"`
	Tags                 []string                 `json:"tags"`
}

type SnippetRating struct {
	ID           string    `json:"id"`
	SnippetID    *string   `json:"snippet_id,omitempty"`
	RecipeID     *string   `json:"recipe_id,omitempty"`
	Rating       int       `json:"rating"`
	Comment      string    `json:"comment"`
	Helpful      bool      `json:"helpful"`
	ScenarioName string    `json:"scenario_name"`
	CreatedAt    time.Time `json:"created_at"`
}

type SnippetExecutionLog struct {
	ID                string    `json:"id"`
	SnippetID         *string   `json:"snippet_id,omitempty"`
	RecipeID          *string   `json:"recipe_id,omitempty"`
	ScenarioName      string    `json:"scenario_name"`
	ExecutionStatus   string    `json:"execution_status"`
	ExecutionTimeMs   int       `json:"execution_time_ms"`
	ErrorMessage      string    `json:"error_message,omitempty"`
	ModificationsMade string    `json:"modifications_made,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

var db *sql.DB
var semanticClient *SemanticSearchClient
var redisClient *redis.Client
var cacheEnabled bool
var webhookManager *WebhookManager
var healthMonitor *HealthMonitor
var rateLimiter *RateLimiter

// RateLimiter manages API rate limiting
type RateLimiter struct {
	mu      sync.RWMutex
	clients map[string]*ClientRateInfo
}

// ClientRateInfo tracks rate limit info for a client
type ClientRateInfo struct {
	Requests   []time.Time
	LastReset  time.Time
	BucketSize int
	Window     time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*ClientRateInfo),
	}

	// Start cleanup goroutine to remove old entries
	go rl.cleanup()

	return rl
}

// cleanup removes old client entries periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for clientID, info := range rl.clients {
			if now.Sub(info.LastReset) > 1*time.Hour {
				delete(rl.clients, clientID)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request is allowed for the given client
func (rl *RateLimiter) Allow(clientID string, bucketSize int, window time.Duration) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Get or create client info
	info, exists := rl.clients[clientID]
	if !exists {
		info = &ClientRateInfo{
			Requests:   make([]time.Time, 0, bucketSize),
			LastReset:  now,
			BucketSize: bucketSize,
			Window:     window,
		}
		rl.clients[clientID] = info
	}

	// Remove old requests outside the window
	cutoff := now.Add(-window)
	validRequests := make([]time.Time, 0, len(info.Requests))
	for _, reqTime := range info.Requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	info.Requests = validRequests

	// Check if we can allow this request
	if len(info.Requests) >= bucketSize {
		return false
	}

	// Add the current request
	info.Requests = append(info.Requests, now)
	return true
}

// GetClientID extracts a client identifier from the request
func getClientID(r *http.Request) string {
	// Try to get API key from header
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "" {
		return "key:" + apiKey
	}

	// Fall back to IP address
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}

	// Check for X-Forwarded-For header (for proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip = strings.TrimSpace(ips[0])
		}
	}

	return "ip:" + ip
}

// rateLimitMiddleware applies rate limiting to requests
func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for health endpoints
		if strings.HasSuffix(r.URL.Path, "/health") {
			next.ServeHTTP(w, r)
			return
		}

		clientID := getClientID(r)

		// Different rate limits for different operations
		var bucketSize int
		var window time.Duration

		switch {
		case strings.Contains(r.URL.Path, "/search"):
			// Search endpoints: 100 requests per minute
			bucketSize = 100
			window = 1 * time.Minute
		case r.Method == "POST" || r.Method == "PUT" || r.Method == "DELETE":
			// Write operations: 30 requests per minute
			bucketSize = 30
			window = 1 * time.Minute
		default:
			// Read operations: 300 requests per minute
			bucketSize = 300
			window = 1 * time.Minute
		}

		// Check rate limit
		if !rateLimiter.Allow(clientID, bucketSize, window) {
			// Add rate limit headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(bucketSize))
			w.Header().Set("X-RateLimit-Window", window.String())
			w.Header().Set("X-RateLimit-Retry-After", window.String())

			http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}

		// Add rate limit headers for successful requests
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(bucketSize))
		w.Header().Set("X-RateLimit-Window", window.String())

		next.ServeHTTP(w, r)
	})
}

func normalizeSearchParams(req SearchRequest) SearchParams {
	params := SearchParams{
		Query: strings.TrimSpace(req.Query),
		Limit: req.Limit,
	}

	if params.Limit <= 0 {
		params.Limit = 10
	} else if params.Limit > 50 {
		params.Limit = 50
	}

	mergeCategories := func(dst *[]string, values []string) {
		if len(values) == 0 {
			return
		}
		seen := make(map[string]struct{}, len(*dst))
		for _, existing := range *dst {
			seen[existing] = struct{}{}
		}
		for _, raw := range values {
			trimmed := strings.TrimSpace(raw)
			if trimmed == "" {
				continue
			}
			if _, exists := seen[trimmed]; exists {
				continue
			}
			seen[trimmed] = struct{}{}
			*dst = append(*dst, trimmed)
		}
	}

	if req.Filters != nil {
		if req.Filters.Configured != nil {
			params.RequireConfigured = true
			params.ConfiguredValue = *req.Filters.Configured
		}
		if req.Filters.MaxPrice != nil {
			value := *req.Filters.MaxPrice
			params.MaxPrice = &value
		}
		mergeCategories(&params.Categories, req.Filters.Categories)
	}

	if req.Configured != nil {
		params.RequireConfigured = true
		params.ConfiguredValue = *req.Configured
	}
	if req.MaxPrice != nil {
		value := *req.MaxPrice
		params.MaxPrice = &value
	}
	mergeCategories(&params.Categories, req.Categories)

	return params
}

func formatPricingSummary(value *float64) string {
	if value == nil {
		return "Pricing not available"
	}
	v := *value
	if v <= 0 {
		return "Free tier available"
	}
	formatted := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", v), "0"), ".")
	if formatted == "" {
		formatted = "0"
	}
	return fmt.Sprintf("Starts at $%s", formatted)
}

func executeFullTextSearch(ctx context.Context, params SearchParams) ([]SearchResult, error) {
	log.Printf("🔍 Executing full-text search for query: %s", params.Query)
	whereClauses := []string{"a.search_vector @@ plainto_tsquery('english', $1)"}
	args := []interface{}{params.Query}
	argIndex := 2

	if params.RequireConfigured {
		if params.ConfiguredValue {
			// Looking for configured APIs
			whereClauses = append(whereClauses, fmt.Sprintf("c.is_configured = $%d", argIndex))
			args = append(args, true)
		} else {
			// Looking for non-configured APIs (including those with no credentials entry)
			whereClauses = append(whereClauses, fmt.Sprintf("(c.is_configured IS NULL OR c.is_configured = $%d)", argIndex))
			args = append(args, false)
		}
		argIndex++
	}

	if len(params.Categories) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("a.category = ANY($%d)", argIndex))
		args = append(args, pq.Array(params.Categories))
		argIndex++
	}

	if params.MaxPrice != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("p.min_price IS NOT NULL AND p.min_price <= $%d", argIndex))
		args = append(args, *params.MaxPrice)
		argIndex++
	}

	query := fmt.Sprintf(`
		SELECT 
			a.id, a.name, a.provider, a.description, a.category,
			ts_rank(a.search_vector, plainto_tsquery('english', $1)) AS relevance,
			COALESCE(c.is_configured, false) AS configured,
			p.min_price
		FROM apis a
		LEFT JOIN api_credentials c ON a.id = c.api_id AND c.environment = 'development'
		LEFT JOIN (
			SELECT api_id, MIN(COALESCE(price_per_request, monthly_cost)) AS min_price
			FROM pricing_tiers
			GROUP BY api_id
		) p ON a.id = p.api_id
		WHERE %s
		ORDER BY relevance DESC
		LIMIT $%d
	`, strings.Join(whereClauses, " AND "), argIndex)

	args = append(args, params.Limit)

	log.Printf("📝 Query: %s", query)
	log.Printf("📝 Args: %v", args)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	rowCount := 0
	for rows.Next() {
		rowCount++
		var (
			result      SearchResult
			configured  bool
			minPriceVal sql.NullFloat64
		)

		err := rows.Scan(
			&result.ID,
			&result.Name,
			&result.Provider,
			&result.Description,
			&result.Category,
			&result.RelevanceScore,
			&configured,
			&minPriceVal,
		)
		if err != nil {
			log.Printf("Failed to scan search row: %v", err)
			continue
		}

		result.Configured = configured
		if minPriceVal.Valid {
			value := minPriceVal.Float64
			result.MinPrice = &value
		}
		result.PricingSummary = formatPricingSummary(result.MinPrice)

		results = append(results, result)
	}

	log.Printf("📊 Found %d rows, returned %d results", rowCount, len(results))

	return results, rows.Err()
}

func (c *SemanticSearchClient) isReady() bool {
	return c != nil && c.QdrantURL != "" && c.OllamaURL != ""
}

func (c *SemanticSearchClient) SemanticSearch(ctx context.Context, params SearchParams) ([]SearchResult, error) {
	if !c.isReady() {
		return nil, fmt.Errorf("semantic search not configured")
	}

	embedding, err := c.generateEmbedding(ctx, params.Query)
	if err != nil {
		return nil, err
	}

	ids, scores, err := c.searchQdrant(ctx, embedding, params)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []SearchResult{}, nil
	}

	return hydrateSearchResults(ctx, ids, scores, params)
}

func (c *SemanticSearchClient) generateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("empty query")
	}

	model := c.OllamaModel
	if model == "" {
		model = "nomic-embed-text"
	}

	payload := map[string]interface{}{
		"model":  model,
		"prompt": text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimRight(c.OllamaURL, "/") + "/api/embeddings"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama embeddings failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}

	var result struct {
		Embedding []float32 `json:"embedding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Embedding) == 0 {
		return nil, fmt.Errorf("embedding response was empty")
	}

	return result.Embedding, nil
}

func (c *SemanticSearchClient) searchQdrant(ctx context.Context, vector []float32, params SearchParams) ([]string, map[string]float64, error) {
	requestBody := map[string]interface{}{
		"vector":       vector,
		"limit":        params.Limit,
		"with_payload": true,
		"with_vector":  false,
	}

	must := make([]map[string]interface{}, 0)
	if params.RequireConfigured {
		must = append(must, map[string]interface{}{
			"key": "configured",
			"match": map[string]interface{}{
				"value": params.ConfiguredValue,
			},
		})
	}
	if len(params.Categories) > 0 {
		must = append(must, map[string]interface{}{
			"key": "category",
			"match": map[string]interface{}{
				"any": params.Categories,
			},
		})
	}
	if len(must) > 0 {
		requestBody["filter"] = map[string]interface{}{"must": must}
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, nil, err
	}

	endpoint := fmt.Sprintf("%s/collections/%s/points/search", strings.TrimRight(c.QdrantURL, "/"), c.QdrantCollection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.QdrantAPIKey != "" {
		req.Header.Set("api-key", c.QdrantAPIKey)
	}

	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("qdrant search failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}

	var qdrantResp struct {
		Result []struct {
			ID      interface{}            `json:"id"`
			Score   float64                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&qdrantResp); err != nil {
		return nil, nil, err
	}

	ids := make([]string, 0, len(qdrantResp.Result))
	scores := make(map[string]float64, len(qdrantResp.Result))

	for _, item := range qdrantResp.Result {
		var apiID string
		if payloadID, ok := item.Payload["api_id"].(string); ok && payloadID != "" {
			apiID = payloadID
		} else if strID, ok := item.ID.(string); ok && strID != "" {
			apiID = strID
		} else {
			continue
		}

		ids = append(ids, apiID)
		scores[apiID] = item.Score
	}

	return ids, scores, nil
}

func hydrateSearchResults(ctx context.Context, ids []string, scores map[string]float64, params SearchParams) ([]SearchResult, error) {
	filteredIDs := filterValidUUIDs(ids)
	if len(filteredIDs) == 0 {
		return []SearchResult{}, nil
	}

	query := `
		SELECT 
			a.id, a.name, a.provider, a.description, a.category,
			COALESCE(c.is_configured, false) AS configured,
			p.min_price
		FROM apis a
		LEFT JOIN api_credentials c ON a.id = c.api_id AND c.environment = 'development'
		LEFT JOIN (
			SELECT api_id, MIN(COALESCE(price_per_request, monthly_cost)) AS min_price
			FROM pricing_tiers
			GROUP BY api_id
		) p ON a.id = p.api_id
		WHERE a.id = ANY($1::uuid[])
		ORDER BY array_position($1::uuid[], a.id)
	`

	rows, err := db.QueryContext(ctx, query, pq.Array(filteredIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resultMap := make(map[string]SearchResult, len(filteredIDs))

	for rows.Next() {
		var (
			res         SearchResult
			configured  bool
			minPriceVal sql.NullFloat64
		)

		err := rows.Scan(
			&res.ID,
			&res.Name,
			&res.Provider,
			&res.Description,
			&res.Category,
			&configured,
			&minPriceVal,
		)
		if err != nil {
			log.Printf("Failed to scan semantic search row: %v", err)
			continue
		}

		res.Configured = configured
		if minPriceVal.Valid {
			value := minPriceVal.Float64
			res.MinPrice = &value
		}
		res.PricingSummary = formatPricingSummary(res.MinPrice)
		res.RelevanceScore = scores[res.ID]

		resultMap[res.ID] = res
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(filteredIDs))
	for _, id := range filteredIDs {
		res, ok := resultMap[id]
		if !ok {
			continue
		}
		if params.RequireConfigured && res.Configured != params.ConfiguredValue {
			continue
		}
		if params.MaxPrice != nil {
			if res.MinPrice == nil || *res.MinPrice > *params.MaxPrice {
				continue
			}
		}
		results = append(results, res)
		if len(results) >= params.Limit {
			break
		}
	}

	return results, nil
}

func filterValidUUIDs(ids []string) []string {
	valid := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, err := uuid.Parse(strings.TrimSpace(id)); err == nil {
			valid = append(valid, strings.TrimSpace(id))
		}
	}
	return valid
}

func populateInitialEmbeddings() {
	ctx := context.Background()
	time.Sleep(2 * time.Second) // Wait for service to fully initialize

	log.Println("🔄 Checking for APIs needing embeddings...")

	// Get all APIs
	rows, err := db.Query(`
		SELECT id, name, provider, description, category
		FROM apis
		WHERE status != 'deprecated'
	`)
	if err != nil {
		log.Printf("Failed to fetch APIs for embedding: %v", err)
		return
	}
	defer rows.Close()

	var apisToEmbed []struct {
		ID          string
		Name        string
		Provider    string
		Description string
		Category    string
	}

	for rows.Next() {
		var api struct {
			ID          string
			Name        string
			Provider    string
			Description string
			Category    string
		}
		if err := rows.Scan(&api.ID, &api.Name, &api.Provider, &api.Description, &api.Category); err != nil {
			continue
		}
		apisToEmbed = append(apisToEmbed, api)
	}

	if len(apisToEmbed) == 0 {
		log.Println("No APIs found to embed")
		return
	}

	// Check which APIs already have embeddings
	for _, api := range apisToEmbed {
		// Create embedding text from API information
		embeddingText := fmt.Sprintf("%s %s %s %s", api.Name, api.Provider, api.Description, api.Category)

		// Generate embedding
		embedding, err := semanticClient.generateEmbedding(ctx, embeddingText)
		if err != nil {
			log.Printf("Failed to generate embedding for %s: %v", api.Name, err)
			continue
		}

		// Store in Qdrant
		if err := semanticClient.storeEmbedding(ctx, api.ID, embedding, api.Name, api.Provider, api.Category); err != nil {
			log.Printf("Failed to store embedding for %s: %v", api.Name, err)
			continue
		}

		log.Printf("✅ Created embedding for: %s", api.Name)
	}

	log.Println("✨ Embedding population complete")
}

// startPricingRefresh starts a goroutine that refreshes API pricing periodically
func startPricingRefresh() {
	// Initial refresh after 1 minute
	time.Sleep(1 * time.Minute)
	refreshAPIPricing()

	// Then refresh every 24 hours
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		refreshAPIPricing()
	}
}

// refreshAPIPricing fetches and updates pricing for all APIs
func refreshAPIPricing() {
	log.Println("🔄 Starting API pricing refresh...")

	// Get all APIs with pricing URLs
	rows, err := db.Query(`
		SELECT id, name, pricing_url 
		FROM apis 
		WHERE pricing_url IS NOT NULL AND pricing_url != ''
	`)
	if err != nil {
		log.Printf("Failed to fetch APIs for pricing refresh: %v", err)
		return
	}
	defer rows.Close()

	refreshedCount := 0
	for rows.Next() {
		var id, name, pricingURL string
		if err := rows.Scan(&id, &name, &pricingURL); err != nil {
			continue
		}

		// Fetch and parse pricing from URL
		pricing := fetchPricingFromURL(pricingURL)
		if pricing != nil {
			// Update pricing tiers
			updatePricingTiers(id, pricing)
			refreshedCount++
			log.Printf("✅ Updated pricing for %s", name)
		}
	}

	// Update last_refreshed timestamp for all processed APIs
	_, err = db.Exec(`
		UPDATE apis 
		SET last_refreshed = NOW() 
		WHERE pricing_url IS NOT NULL AND pricing_url != ''
	`)
	if err != nil {
		log.Printf("Failed to update last_refreshed timestamp: %v", err)
	}

	log.Printf("✨ Pricing refresh complete. Updated %d APIs", refreshedCount)
}

// fetchPricingFromURL attempts to fetch pricing information from a URL
func fetchPricingFromURL(url string) map[string]interface{} {
	// Simple implementation - in production this would use more sophisticated parsing
	resp, err := http.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	// For now, return a placeholder - real implementation would parse the pricing page
	// This could be enhanced to scrape common pricing patterns or use APIs
	return map[string]interface{}{
		"updated": true,
	}
}

// updatePricingTiers updates the pricing tiers for an API
func updatePricingTiers(apiID string, pricing map[string]interface{}) {
	// For demonstration, update the timestamp
	// Real implementation would parse and store actual pricing data
	_, err := db.Exec(`
		UPDATE pricing_tiers 
		SET updated_at = NOW() 
		WHERE api_id = $1
	`, apiID)
	if err != nil {
		log.Printf("Failed to update pricing tiers for API %s: %v", apiID, err)
	}
}

// calculateCostHandler calculates estimated costs based on usage patterns
func calculateCostHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		APIID            string  `json:"api_id"`
		RequestsPerMonth int     `json:"requests_per_month"`
		DataPerRequestMB float64 `json:"data_per_request_mb"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get pricing tiers for the API
	rows, err := db.Query(`
		SELECT name, price_per_request, price_per_mb, monthly_cost, free_tier_requests
		FROM pricing_tiers
		WHERE api_id = $1
		ORDER BY price_per_request ASC NULLS LAST
	`, req.APIID)
	if err != nil {
		http.Error(w, "Failed to fetch pricing tiers", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var bestTier struct {
		Name          string             `json:"name"`
		EstimatedCost float64            `json:"estimated_cost"`
		CostBreakdown map[string]float64 `json:"cost_breakdown"`
	}

	minCost := math.MaxFloat64
	for rows.Next() {
		var tier PricingTier
		if err := rows.Scan(&tier.Name, &tier.PricePerRequest, &tier.PricePerMB, &tier.MonthlyCost, &tier.FreeTierRequests); err != nil {
			continue
		}

		// Calculate cost for this tier
		billableRequests := math.Max(0, float64(req.RequestsPerMonth-tier.FreeTierRequests))
		requestCost := billableRequests * tier.PricePerRequest
		dataCost := float64(req.RequestsPerMonth) * req.DataPerRequestMB * tier.PricePerMB
		totalCost := tier.MonthlyCost + requestCost + dataCost

		if totalCost < minCost {
			minCost = totalCost
			bestTier.Name = tier.Name
			bestTier.EstimatedCost = totalCost
			bestTier.CostBreakdown = map[string]float64{
				"monthly_base": tier.MonthlyCost,
				"request_cost": requestCost,
				"data_cost":    dataCost,
			}
		}
	}

	// Compare with other similar APIs
	alternatives := findCostEffectiveAlternatives(req.APIID, minCost)

	response := map[string]interface{}{
		"recommended_tier": bestTier,
		"alternatives":     alternatives,
		"savings_tip":      generateSavingsTip(req.RequestsPerMonth),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// findCostEffectiveAlternatives finds cheaper alternatives with similar capabilities
func findCostEffectiveAlternatives(apiID string, currentCost float64) []map[string]interface{} {
	// Get the category of the current API
	var category string
	err := db.QueryRow("SELECT category FROM apis WHERE id = $1", apiID).Scan(&category)
	if err != nil {
		return nil
	}

	// Find alternatives in the same category
	query := `
		SELECT DISTINCT a.id, a.name, a.provider, 
		       MIN(pt.price_per_request) as min_price
		FROM apis a
		LEFT JOIN pricing_tiers pt ON a.id = pt.api_id
		WHERE a.category = $1 AND a.id != $2
		GROUP BY a.id, a.name, a.provider
		HAVING MIN(pt.price_per_request) < $3 OR MIN(pt.price_per_request) IS NULL
		ORDER BY min_price ASC NULLS LAST
		LIMIT 3
	`

	rows, err := db.Query(query, category, apiID, currentCost/10000) // Rough estimate
	if err != nil {
		return nil
	}
	defer rows.Close()

	var alternatives []map[string]interface{}
	for rows.Next() {
		var id, name, provider string
		var minPrice sql.NullFloat64
		if err := rows.Scan(&id, &name, &provider, &minPrice); err != nil {
			continue
		}

		alt := map[string]interface{}{
			"id":       id,
			"name":     name,
			"provider": provider,
		}
		if minPrice.Valid {
			alt["estimated_savings"] = fmt.Sprintf("%.2f%%", (currentCost-minPrice.Float64*10000)/currentCost*100)
		}
		alternatives = append(alternatives, alt)
	}

	return alternatives
}

// generateSavingsTip generates cost-saving recommendations based on usage
func generateSavingsTip(requestsPerMonth int) string {
	switch {
	case requestsPerMonth > 1000000:
		return "Consider negotiating an enterprise agreement for volume discounts"
	case requestsPerMonth > 100000:
		return "Look for APIs with better bulk pricing or consider caching frequently used responses"
	case requestsPerMonth > 10000:
		return "Implement request batching and response caching to reduce API calls"
	default:
		return "Monitor usage patterns to identify optimization opportunities"
	}
}

// trackAPIVersion tracks version changes for APIs
func trackAPIVersion(apiID string, newVersion string, changeSummary string, breakingChanges bool) error {
	// Check if version changed
	var currentVersion string
	err := db.QueryRow("SELECT version FROM apis WHERE id = $1", apiID).Scan(&currentVersion)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	// Only track if version actually changed
	if currentVersion != newVersion {
		versionID := uuid.New().String()
		_, err := db.Exec(`
			INSERT INTO api_versions (id, api_id, version, change_summary, breaking_changes, created_at)
			VALUES ($1, $2, $3, $4, $5, NOW())
		`, versionID, apiID, newVersion, changeSummary, breakingChanges)
		if err != nil {
			return err
		}

		// Update current version in APIs table
		_, err = db.Exec("UPDATE apis SET version = $1 WHERE id = $2", newVersion, apiID)
		if err != nil {
			return err
		}

		log.Printf("📝 Tracked version change for API %s: %s -> %s", apiID, currentVersion, newVersion)
	}

	return nil
}

// getAPIVersionsHandler returns version history for an API
func getAPIVersionsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	rows, err := db.Query(`
		SELECT id, version, change_summary, breaking_changes, created_at
		FROM api_versions
		WHERE api_id = $1
		ORDER BY created_at DESC
	`, apiID)
	if err != nil {
		http.Error(w, "Failed to fetch version history", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var versions []APIVersion
	for rows.Next() {
		var v APIVersion
		if err := rows.Scan(&v.ID, &v.Version, &v.ChangeSummary, &v.BreakingChanges, &v.CreatedAt); err != nil {
			continue
		}
		v.APIID = apiID
		versions = append(versions, v)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(versions)
}

// updateAPIVersionHandler updates API version with tracking
func updateAPIVersionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	var req struct {
		Version         string `json:"version"`
		ChangeSummary   string `json:"change_summary"`
		BreakingChanges bool   `json:"breaking_changes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Version == "" {
		http.Error(w, "Version is required", http.StatusBadRequest)
		return
	}

	// Track the version change
	if err := trackAPIVersion(apiID, req.Version, req.ChangeSummary, req.BreakingChanges); err != nil {
		http.Error(w, "Failed to track version change", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c *SemanticSearchClient) storeEmbedding(ctx context.Context, apiID string, embedding []float32, name, provider, category string) error {
	// Convert float32 to float64 for JSON marshaling
	embedFloat64 := make([]float64, len(embedding))
	for i, v := range embedding {
		embedFloat64[i] = float64(v)
	}

	// Prepare the point for Qdrant
	point := map[string]interface{}{
		"id":     apiID,
		"vector": embedFloat64,
		"payload": map[string]interface{}{
			"api_id":   apiID,
			"name":     name,
			"provider": provider,
			"category": category,
		},
	}

	// Prepare the request
	reqBody := map[string]interface{}{
		"points": []map[string]interface{}{point},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal point: %w", err)
	}

	// Store in Qdrant
	url := fmt.Sprintf("%s/collections/%s/points?wait=true", c.QdrantURL, c.QdrantCollection)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to store embedding: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to store embedding (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start api-library

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	initDB()
	semanticClient = initSemanticSearchClient()
	initRedisCache()

	// Initialize webhook manager
	webhookManager = NewWebhookManager(db)

	// Initialize rate limiter
	rateLimiter = NewRateLimiter()

	// Initialize embeddings for existing APIs if semantic search is enabled
	if semanticClient != nil && semanticClient.isReady() {
		go populateInitialEmbeddings()
	}

	// Start periodic pricing refresh (every 24 hours)
	go startPricingRefresh()

	router := setupRouter()

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(router)

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	log.Printf("API Library service starting on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func initDB() {
	// Database configuration - support both POSTGRES_URL and individual components
	psqlInfo := os.Getenv("POSTGRES_URL")
	if psqlInfo == "" {
		// Try to build from individual components - REQUIRED, no defaults
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")

		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Database configuration missing. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}

		psqlInfo = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbPassword, dbName)
		log.Printf("📊 Connecting to database: %s on %s:%s", dbName, dbHost, dbPort)
	}
	log.Printf("🔗 Database connection string: %s", psqlInfo)

	var err error
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Set connection pool settings with better values
	db.SetMaxOpenConns(50)                  // Increased from 25
	db.SetMaxIdleConns(10)                  // Increased from 5
	db.SetConnMaxLifetime(30 * time.Minute) // Increased from 5 minutes
	db.SetConnMaxIdleTime(10 * time.Minute) // Added idle timeout

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("📆 Database URL configured")

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt+1)
			break
		}

		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay)*math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))

		// Add progressive jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))
		actualDelay := delay + jitter

		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt+1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)

		// Provide detailed status every few attempts
		if attempt > 0 && attempt%3 == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt+1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt*2)*baseDelay)
			log.Printf("   - Current delay: %v (with jitter: %v)", delay, jitter)
		}

		time.Sleep(actualDelay)
	}

	if pingErr != nil {
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}

	log.Println("🎉 Database connection pool established successfully!")

	// Initialize database schema
	if err := initializeSchema(); err != nil {
		log.Printf("⚠️  Warning: Failed to initialize schema: %v", err)
		// Continue anyway as schema might already exist
	}
}

func initRedisCache() {
	// Redis is optional - if not available, caching will be disabled
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}

	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}

	redisDB := 0
	if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
		if parsed, err := strconv.Atoi(dbStr); err == nil {
			redisDB = parsed
		}
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		DB:       redisDB,
		Password: os.Getenv("REDIS_PASSWORD"),
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("⚠️  Redis not available: %v - caching disabled", err)
		redisClient = nil
		cacheEnabled = false
	} else {
		log.Printf("✅ Redis cache enabled at %s:%s", redisHost, redisPort)
		cacheEnabled = true
	}
}

// Cache helper functions
func getCacheKey(prefix string, params interface{}) string {
	// Create a deterministic cache key from the parameters
	data, _ := json.Marshal(params)
	hash := md5.Sum(data)
	return fmt.Sprintf("%s:%s", prefix, hex.EncodeToString(hash[:]))
}

func getFromCache(ctx context.Context, key string) ([]byte, error) {
	if !cacheEnabled || redisClient == nil {
		return nil, fmt.Errorf("cache disabled")
	}

	result, err := redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("key not found")
	} else if err != nil {
		return nil, err
	}

	return []byte(result), nil
}

func setCache(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if !cacheEnabled || redisClient == nil {
		return nil // Silently skip if cache is disabled
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return redisClient.Set(ctx, key, data, expiration).Err()
}

func invalidateCachePattern(ctx context.Context, pattern string) error {
	if !cacheEnabled || redisClient == nil {
		return nil
	}

	// Get all keys matching the pattern
	keys, err := redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return redisClient.Del(ctx, keys...).Err()
	}

	return nil
}

func initializeSchema() error {
	// Check if tables exist, if not create them
	var tableCount int
	err := db.QueryRow(`
		SELECT COUNT(*) 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_name IN ('apis', 'endpoints', 'pricing_tiers', 'notes', 'api_credentials')
	`).Scan(&tableCount)

	if err != nil {
		return fmt.Errorf("failed to check existing tables: %v", err)
	}

	// If all tables exist, skip initialization
	if tableCount >= 5 {
		log.Println("✅ Database schema already initialized")
		return nil
	}

	log.Println("📝 Initializing database schema...")

	// Create tables directly with individual statements to handle partial failures better
	tables := []string{
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,
		`CREATE TABLE IF NOT EXISTS apis (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			name VARCHAR(255) NOT NULL UNIQUE,
			provider VARCHAR(255) NOT NULL,
			description TEXT,
			base_url VARCHAR(500),
			documentation_url VARCHAR(500),
			pricing_url VARCHAR(500),
			category VARCHAR(100),
			status VARCHAR(50) DEFAULT 'active',
			sunset_date DATE,
			auth_type VARCHAR(50),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_refreshed TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			source_url VARCHAR(500),
			tags TEXT[],
			capabilities TEXT[],
			search_vector tsvector
		)`,
		`CREATE TABLE IF NOT EXISTS endpoints (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
			path VARCHAR(500) NOT NULL,
			method VARCHAR(10) NOT NULL,
			description TEXT,
			rate_limit_requests INTEGER,
			rate_limit_period VARCHAR(50),
			request_schema JSONB,
			response_schema JSONB,
			requires_auth BOOLEAN DEFAULT true,
			auth_details JSONB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(api_id, path, method)
		)`,
		`CREATE TABLE IF NOT EXISTS pricing_tiers (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			price_per_request DECIMAL(10, 6),
			price_per_mb DECIMAL(10, 6),
			price_per_minute DECIMAL(10, 6),
			monthly_cost DECIMAL(10, 2),
			annual_cost DECIMAL(10, 2),
			free_tier_requests INTEGER,
			free_tier_mb INTEGER,
			max_requests_per_month INTEGER,
			pricing_details JSONB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(api_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS notes (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
			content TEXT NOT NULL,
			type VARCHAR(50) NOT NULL,
			endpoint_id UUID REFERENCES endpoints(id) ON DELETE CASCADE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_by VARCHAR(255) DEFAULT 'system',
			scenario_source VARCHAR(255),
			helpful_count INTEGER DEFAULT 0,
			not_helpful_count INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS api_credentials (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
			is_configured BOOLEAN DEFAULT false,
			configuration_date TIMESTAMP,
			last_verified TIMESTAMP,
			environment VARCHAR(50) DEFAULT 'development',
			last_used TIMESTAMP,
			usage_count INTEGER DEFAULT 0,
			configuration_notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(api_id, environment)
		)`,
	}

	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			log.Printf("⚠️  Warning: Failed to create table: %v", err)
			// Continue with other tables
		}
	}

	// Create indexes
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_apis_status ON apis(status)`,
		`CREATE INDEX IF NOT EXISTS idx_apis_category ON apis(category)`,
		`CREATE INDEX IF NOT EXISTS idx_apis_provider ON apis(provider)`,
		`CREATE INDEX IF NOT EXISTS idx_endpoints_api_id ON endpoints(api_id)`,
		`CREATE INDEX IF NOT EXISTS idx_pricing_api_id ON pricing_tiers(api_id)`,
		`CREATE INDEX IF NOT EXISTS idx_notes_api_id ON notes(api_id)`,
		`CREATE INDEX IF NOT EXISTS idx_credentials_api_id ON api_credentials(api_id)`,
		`CREATE INDEX IF NOT EXISTS idx_credentials_configured ON api_credentials(is_configured)`,
	}

	for _, index := range indexes {
		if _, err := db.Exec(index); err != nil {
			log.Printf("⚠️  Warning: Failed to create index: %v", err)
		}
	}

	log.Println("✅ Database schema initialized successfully")

	// Insert minimal seed data to get started
	seedData := []string{
		`INSERT INTO apis (name, provider, description, base_url, documentation_url, category, auth_type, tags, capabilities)
		 VALUES ('OpenAI API', 'OpenAI', 'Access to GPT models for text generation, embeddings, and more', 
		         'https://api.openai.com', 'https://platform.openai.com/docs', 'AI/ML', 'bearer',
		         ARRAY['ai', 'ml', 'nlp', 'gpt'], ARRAY['text-generation', 'embeddings', 'chat'])
		 ON CONFLICT (name) DO NOTHING`,
		`INSERT INTO apis (name, provider, description, base_url, documentation_url, category, auth_type, tags, capabilities)
		 VALUES ('Stripe API', 'Stripe', 'Payment processing and financial services', 
		         'https://api.stripe.com', 'https://stripe.com/docs', 'Payment', 'bearer',
		         ARRAY['payment', 'finance', 'billing'], ARRAY['payment-processing', 'subscriptions', 'invoicing'])
		 ON CONFLICT (name) DO NOTHING`,
	}

	for _, seed := range seedData {
		if _, err := db.Exec(seed); err != nil {
			log.Printf("⚠️  Warning: Failed to insert seed data: %v", err)
		}
	}

	return nil
}

func setupRouter() *mux.Router {
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", healthHandler).Methods("GET")

	// API v1 routes with rate limiting
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.Use(rateLimitMiddleware)

	// Search endpoints
	v1.HandleFunc("/search", searchAPIsHandler).Methods("GET", "POST")

	// API CRUD
	v1.HandleFunc("/apis", listAPIsHandler).Methods("GET")
	v1.HandleFunc("/apis", createAPIHandler).Methods("POST")
	v1.HandleFunc("/apis/{id}", getAPIHandler).Methods("GET")
	v1.HandleFunc("/apis/{id}", updateAPIHandler).Methods("PUT")
	v1.HandleFunc("/apis/{id}", deleteAPIHandler).Methods("DELETE")

	// Notes
	v1.HandleFunc("/apis/{id}/notes", getNotesHandler).Methods("GET")
	v1.HandleFunc("/apis/{id}/notes", addNoteHandler).Methods("POST")

	// Endpoints
	v1.HandleFunc("/apis/{id}/endpoints", getAPIEndpointsHandler).Methods("GET")

	// Configuration tracking
	v1.HandleFunc("/configured", getConfiguredAPIsHandler).Methods("GET")
	v1.HandleFunc("/apis/{id}/configure", markConfiguredHandler).Methods("POST")

	// Research integration
	v1.HandleFunc("/request-research", requestResearchHandler).Methods("POST")

	// Cost calculator
	v1.HandleFunc("/calculate-cost", calculateCostHandler).Methods("POST")

	// Version tracking
	v1.HandleFunc("/apis/{id}/versions", getAPIVersionsHandler).Methods("GET")
	v1.HandleFunc("/apis/{id}/versions", updateAPIVersionHandler).Methods("POST")

	// Export capabilities
	v1.HandleFunc("/export", exportAPIsHandler).Methods("GET")
	v1.HandleFunc("/apis/{id}/status", updateAPIStatusHandler).Methods("PUT")

	// Categories and tags
	v1.HandleFunc("/categories", getCategoriesHandler).Methods("GET")
	v1.HandleFunc("/tags", getTagsHandler).Methods("GET")
	v1.HandleFunc("/apis/{id}/tags", updateAPITagsHandler).Methods("PUT")

	// Integration snippets
	v1.HandleFunc("/apis/{id}/snippets", getSnippetsHandler).Methods("GET")
	v1.HandleFunc("/apis/{id}/snippets", createSnippetHandler).Methods("POST")
	v1.HandleFunc("/snippets/popular", getPopularSnippetsHandler).Methods("GET")
	v1.HandleFunc("/snippets/{snippet_id}", getSnippetByIDHandler).Methods("GET")
	v1.HandleFunc("/snippets/{snippet_id}/vote", voteSnippetHandler).Methods("POST")

	// Integration recipes
	v1.HandleFunc("/apis/{id}/recipes", getRecipesHandler).Methods("GET")
	v1.HandleFunc("/apis/{id}/recipes", createRecipeHandler).Methods("POST")
	v1.HandleFunc("/recipes/{id}", getRecipeHandler).Methods("GET")
	v1.HandleFunc("/recipes/{id}", updateRecipeHandler).Methods("PUT")
	v1.HandleFunc("/recipes/{id}", deleteRecipeHandler).Methods("DELETE")
	v1.HandleFunc("/recipes/successful", getSuccessfulRecipesHandler).Methods("GET")

	// API Comparison Matrix
	v1.HandleFunc("/compare", compareAPIsHandler).Methods("POST")

	// Usage Analytics
	v1.HandleFunc("/apis/{id}/usage", trackUsageHandler).Methods("POST")
	v1.HandleFunc("/apis/{id}/analytics", getAnalyticsHandler).Methods("GET")
	v1.HandleFunc("/recommendations", getRecommendationsHandler).Methods("GET")

	// Webhook Management
	v1.HandleFunc("/webhooks", createWebhookHandler).Methods("POST")
	v1.HandleFunc("/webhooks", listWebhooksHandler).Methods("GET")
	v1.HandleFunc("/webhooks/{id}", deleteWebhookHandler).Methods("DELETE")
	v1.HandleFunc("/webhooks/{id}/test", testWebhookHandler).Methods("POST")

	// Code Generator Integration
	v1.HandleFunc("/codegen/apis/{id}", getCodeGenSpecHandler).Methods("GET")
	v1.HandleFunc("/codegen/search", searchCodeGenAPIsHandler).Methods("POST")
	v1.HandleFunc("/codegen/templates/{language}", getCodeGenTemplatesHandler).Methods("GET")

	// Snippet/Recipe ratings and execution logs
	// v1.HandleFunc("/snippets/{id}/rate", rateSnippetHandler).Methods("POST")
	// v1.HandleFunc("/recipes/{id}/rate", rateRecipeHandler).Methods("POST")
	// v1.HandleFunc("/snippets/{id}/execute", logSnippetExecutionHandler).Methods("POST")
	// v1.HandleFunc("/recipes/{id}/execute", logRecipeExecutionHandler).Methods("POST")

	return router
}

// Handlers

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "api-library",
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func searchAPIsHandler(w http.ResponseWriter, r *http.Request) {
	var searchReq SearchRequest

	if r.Method == http.MethodPost {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&searchReq); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
	} else {
		query := strings.TrimSpace(r.URL.Query().Get("query"))
		if query == "" {
			query = strings.TrimSpace(r.URL.Query().Get("q"))
		}
		searchReq.Query = query

		if limitStr := strings.TrimSpace(r.URL.Query().Get("limit")); limitStr != "" {
			if parsed, err := strconv.Atoi(limitStr); err == nil {
				searchReq.Limit = parsed
			}
		}

		if configuredStr := strings.TrimSpace(r.URL.Query().Get("configured")); configuredStr != "" {
			if parsed, err := strconv.ParseBool(configuredStr); err == nil {
				searchReq.Configured = &parsed
			}
		}

		if configuredOnly := strings.TrimSpace(r.URL.Query().Get("configured_only")); configuredOnly != "" {
			if parsed, err := strconv.ParseBool(configuredOnly); err == nil {
				searchReq.Configured = &parsed
			}
		}

		if maxPriceStr := strings.TrimSpace(r.URL.Query().Get("max_price")); maxPriceStr != "" {
			if parsed, err := strconv.ParseFloat(maxPriceStr, 64); err == nil {
				searchReq.MaxPrice = &parsed
			}
		}

		allCategories := r.URL.Query()["category"]
		if categoriesCSV := strings.TrimSpace(r.URL.Query().Get("categories")); categoriesCSV != "" {
			allCategories = append(allCategories, strings.Split(categoriesCSV, ",")...)
		}
		if len(allCategories) > 0 {
			searchReq.Categories = allCategories
		}
	}

	params := normalizeSearchParams(searchReq)
	if params.Query == "" {
		http.Error(w, "Query parameter is required", http.StatusBadRequest)
		return
	}

	// Try to get from cache first
	cacheKey := getCacheKey("search", params)
	if cached, err := getFromCache(r.Context(), cacheKey); err == nil {
		// Cache hit - return cached response
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write(cached)
		return
	}

	// Cache miss - perform search
	results, method, err := performSearch(r.Context(), params)
	if err != nil {
		log.Printf("Search query failed: %v", err)
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"results": results,
		"count":   len(results),
		"query":   params.Query,
		"method":  method,
	}

	// Cache the response for 5 minutes
	setCache(r.Context(), cacheKey, response, 5*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	json.NewEncoder(w).Encode(response)
}

func performSearch(ctx context.Context, params SearchParams) ([]SearchResult, string, error) {
	// Temporarily disable semantic search until embeddings are synchronized
	// TODO: Re-enable after populating embeddings with correct database IDs
	// if semanticClient != nil && semanticClient.isReady() {
	// 	results, err := semanticClient.SemanticSearch(ctx, params)
	// 	if err == nil && len(results) > 0 {
	// 		return results, "semantic", nil
	// 	}
	// 	if err != nil {
	// 		log.Printf("Semantic search attempt failed: %v", err)
	// 	}
	// }

	results, err := executeFullTextSearch(ctx, params)
	if err != nil {
		return nil, "keyword", err
	}
	return results, "keyword", nil
}

func initSemanticSearchClient() *SemanticSearchClient {
	qdrantURL := strings.TrimSpace(os.Getenv("QDRANT_URL"))
	if qdrantURL == "" {
		qdrantURL = strings.TrimSpace(os.Getenv("QDRANT_BASE_URL"))
	}
	if qdrantURL == "" {
		if port := strings.TrimSpace(os.Getenv("RESOURCE_PORT_QDRANT")); port != "" {
			qdrantURL = fmt.Sprintf("http://localhost:%s", port)
		}
	}
	qdrantURL = strings.TrimRight(qdrantURL, "/")

	ollamaURL := strings.TrimSpace(os.Getenv("OLLAMA_URL"))
	if ollamaURL == "" {
		ollamaURL = strings.TrimSpace(os.Getenv("OLLAMA_BASE_URL"))
	}
	if ollamaURL == "" {
		if port := strings.TrimSpace(os.Getenv("RESOURCE_PORT_OLLAMA")); port != "" {
			ollamaURL = fmt.Sprintf("http://localhost:%s", port)
		}
	}
	ollamaURL = strings.TrimRight(ollamaURL, "/")

	if qdrantURL == "" || ollamaURL == "" {
		log.Println("🧠 Semantic search disabled (Qdrant or Ollama URL missing)")
		return nil
	}

	collection := strings.TrimSpace(os.Getenv("QDRANT_COLLECTION"))
	if collection == "" {
		collection = "api_embeddings"
	}

	model := strings.TrimSpace(os.Getenv("OLLAMA_EMBED_MODEL"))
	if model == "" {
		model = "nomic-embed-text"
	}

	client := &SemanticSearchClient{
		QdrantURL:        qdrantURL,
		QdrantAPIKey:     strings.TrimSpace(os.Getenv("QDRANT_API_KEY")),
		QdrantCollection: collection,
		OllamaURL:        ollamaURL,
		OllamaModel:      model,
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}

	log.Printf("🔍 Semantic search enabled (Qdrant: %s, Collection: %s, Model: %s)", client.QdrantURL, client.QdrantCollection, client.OllamaModel)
	return client
}

func listAPIsHandler(w http.ResponseWriter, r *http.Request) {
	// Add context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	category := r.URL.Query().Get("category")
	status := r.URL.Query().Get("status")

	query := `SELECT id, name, provider, description, category, status FROM apis WHERE 1=1`
	var args []interface{}
	argCount := 0

	if category != "" {
		argCount++
		query += fmt.Sprintf(" AND category = $%d", argCount)
		args = append(args, category)
	}

	if status != "" {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
	}

	query += " ORDER BY name"

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("Failed to fetch APIs: %v", err)
		http.Error(w, "Failed to fetch APIs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var apis []API
	for rows.Next() {
		var api API
		err := rows.Scan(&api.ID, &api.Name, &api.Provider, &api.Description, &api.Category, &api.Status)
		if err != nil {
			log.Printf("Failed to scan API: %v", err)
			continue
		}
		apis = append(apis, api)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apis)
}

// sanitizeString escapes HTML to prevent XSS attacks
func sanitizeString(s string) string {
	return html.EscapeString(s)
}

// sanitizeStringSlice escapes HTML in all slice elements
func sanitizeStringSlice(slice []string) []string {
	if slice == nil {
		return nil
	}
	result := make([]string, len(slice))
	for i, s := range slice {
		result[i] = html.EscapeString(s)
	}
	return result
}

func createAPIHandler(w http.ResponseWriter, r *http.Request) {
	var api API
	if err := json.NewDecoder(r.Body).Decode(&api); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Sanitize all string inputs to prevent XSS
	api.Name = sanitizeString(api.Name)
	api.Provider = sanitizeString(api.Provider)
	api.Description = sanitizeString(api.Description)
	api.BaseURL = sanitizeString(api.BaseURL)
	api.DocumentationURL = sanitizeString(api.DocumentationURL)
	api.PricingURL = sanitizeString(api.PricingURL)
	api.Category = sanitizeString(api.Category)
	api.Status = sanitizeString(api.Status)
	api.AuthType = sanitizeString(api.AuthType)
	api.SourceURL = sanitizeString(api.SourceURL)
	api.Tags = sanitizeStringSlice(api.Tags)
	api.Capabilities = sanitizeStringSlice(api.Capabilities)

	api.ID = uuid.New().String()
	api.CreatedAt = time.Now()
	api.UpdatedAt = time.Now()
	api.LastRefreshed = time.Now()

	query := `
		INSERT INTO apis (id, name, provider, description, base_url, documentation_url, 
			pricing_url, category, status, auth_type, tags, capabilities, source_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING created_at, updated_at
	`

	err := db.QueryRow(query, api.ID, api.Name, api.Provider, api.Description,
		api.BaseURL, api.DocumentationURL, api.PricingURL, api.Category,
		api.Status, api.AuthType, pq.Array(api.Tags), pq.Array(api.Capabilities),
		api.SourceURL).Scan(&api.CreatedAt, &api.UpdatedAt)

	if err != nil {
		log.Printf("Failed to create API: %v", err)
		http.Error(w, "Failed to create API", http.StatusInternalServerError)
		return
	}

	// Invalidate search cache when new API is added
	invalidateCachePattern(r.Context(), "search:*")

	// Trigger webhook event for API creation
	if webhookManager != nil {
		webhookManager.TriggerEvent("api.created", map[string]interface{}{
			"api_id":   api.ID,
			"name":     api.Name,
			"provider": api.Provider,
			"category": api.Category,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(api)
}

func getAPIHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	var api API
	var baseURL, docURL, pricingURL, sourceURL sql.NullString
	var description sql.NullString

	query := `
		SELECT id, name, provider, COALESCE(description, ''), 
			base_url, documentation_url, pricing_url, 
			COALESCE(category, 'general'), COALESCE(status, 'active'), COALESCE(auth_type, 'none'), 
			tags, capabilities,
			created_at, updated_at, last_refreshed, source_url
		FROM apis WHERE id = $1
	`

	err := db.QueryRow(query, apiID).Scan(
		&api.ID, &api.Name, &api.Provider, &description,
		&baseURL, &docURL, &pricingURL,
		&api.Category, &api.Status, &api.AuthType,
		pq.Array(&api.Tags), pq.Array(&api.Capabilities),
		&api.CreatedAt, &api.UpdatedAt, &api.LastRefreshed, &sourceURL)

	if err == sql.ErrNoRows {
		http.Error(w, "API not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error fetching API %s: %v", apiID, err)
		http.Error(w, "Failed to fetch API", http.StatusInternalServerError)
		return
	}

	// Handle nullable fields
	if description.Valid {
		api.Description = description.String
	}
	if baseURL.Valid {
		api.BaseURL = baseURL.String
	}
	if docURL.Valid {
		api.DocumentationURL = docURL.String
	}
	if pricingURL.Valid {
		api.PricingURL = pricingURL.String
	}
	if sourceURL.Valid {
		api.SourceURL = sourceURL.String
	}

	// Ensure arrays are not nil
	if api.Tags == nil {
		api.Tags = []string{}
	}
	if api.Capabilities == nil {
		api.Capabilities = []string{}
	}

	// Set default version if not in database
	if api.Version == "" {
		api.Version = "1.0.0"
	}

	// Fetch notes
	notesQuery := `SELECT id, content, type, created_at, created_by FROM notes WHERE api_id = $1`
	rows, err := db.Query(notesQuery, apiID)

	var notes []Note
	if err == nil && rows != nil {
		defer rows.Close()
		for rows.Next() {
			var note Note
			rows.Scan(&note.ID, &note.Content, &note.Type, &note.CreatedAt, &note.CreatedBy)
			notes = append(notes, note)
		}
	}

	response := map[string]interface{}{
		"api":   api,
		"notes": notes,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getAPIEndpointsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	// Fetch endpoints
	rows, err := db.Query(`
		SELECT id, path, method, description, rate_limit_requests, rate_limit_period,
		       request_schema, response_schema, requires_auth, auth_details,
		       created_at, updated_at
		FROM endpoints 
		WHERE api_id = $1
		ORDER BY method, path`, apiID)

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var endpoints []map[string]interface{}
	for rows.Next() {
		var id, path, method string
		var description, rateLimitPeriod sql.NullString
		var rateLimitRequests sql.NullInt64
		var requestSchema, responseSchema, authDetails sql.NullString
		var requiresAuth bool
		var createdAt, updatedAt time.Time

		err := rows.Scan(&id, &path, &method, &description, &rateLimitRequests,
			&rateLimitPeriod, &requestSchema, &responseSchema, &requiresAuth,
			&authDetails, &createdAt, &updatedAt)
		if err != nil {
			continue
		}

		endpoint := map[string]interface{}{
			"id":            id,
			"path":          path,
			"method":        method,
			"requires_auth": requiresAuth,
			"created_at":    createdAt,
			"updated_at":    updatedAt,
		}

		if description.Valid {
			endpoint["description"] = description.String
		}
		if rateLimitRequests.Valid {
			endpoint["rate_limit_requests"] = rateLimitRequests.Int64
		}
		if rateLimitPeriod.Valid {
			endpoint["rate_limit_period"] = rateLimitPeriod.String
		}
		if requestSchema.Valid {
			endpoint["request_schema"] = json.RawMessage(requestSchema.String)
		}
		if responseSchema.Valid {
			endpoint["response_schema"] = json.RawMessage(responseSchema.String)
		}
		if authDetails.Valid {
			endpoint["auth_details"] = json.RawMessage(authDetails.String)
		}

		endpoints = append(endpoints, endpoint)
	}

	// If no endpoints, return empty array instead of null
	if endpoints == nil {
		endpoints = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(endpoints)
}

func updateAPIHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	var api API
	if err := json.NewDecoder(r.Body).Decode(&api); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := `
		UPDATE apis 
		SET name = $2, provider = $3, description = $4, base_url = $5,
			documentation_url = $6, pricing_url = $7, category = $8,
			status = $9, auth_type = $10, tags = $11, capabilities = $12,
			source_url = $13, updated_at = NOW()
		WHERE id = $1
	`

	_, err := db.Exec(query, apiID, api.Name, api.Provider, api.Description,
		api.BaseURL, api.DocumentationURL, api.PricingURL, api.Category,
		api.Status, api.AuthType, pq.Array(api.Tags), pq.Array(api.Capabilities),
		api.SourceURL)

	if err != nil {
		http.Error(w, "Failed to update API", http.StatusInternalServerError)
		return
	}

	// Invalidate cache when API is updated
	invalidateCachePattern(r.Context(), "search:*")

	w.WriteHeader(http.StatusNoContent)
}

func deleteAPIHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	_, err := db.Exec("DELETE FROM apis WHERE id = $1", apiID)
	if err != nil {
		http.Error(w, "Failed to delete API", http.StatusInternalServerError)
		return
	}

	// Invalidate cache when API is deleted
	invalidateCachePattern(r.Context(), "search:*")

	w.WriteHeader(http.StatusNoContent)
}

func getNotesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	query := `SELECT id, content, type, created_at, created_by FROM notes WHERE api_id = $1 ORDER BY created_at DESC`
	rows, err := db.Query(query, apiID)
	if err != nil {
		http.Error(w, "Failed to fetch notes", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		err := rows.Scan(&note.ID, &note.Content, &note.Type, &note.CreatedAt, &note.CreatedBy)
		if err != nil {
			continue
		}
		note.APIID = apiID
		notes = append(notes, note)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notes)
}

func addNoteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	var note Note
	if err := json.NewDecoder(r.Body).Decode(&note); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Sanitize inputs to prevent XSS
	note.Content = sanitizeString(note.Content)
	note.Type = sanitizeString(note.Type)
	note.CreatedBy = sanitizeString(note.CreatedBy)

	note.ID = uuid.New().String()
	note.APIID = apiID
	note.CreatedAt = time.Now()
	if note.CreatedBy == "" {
		note.CreatedBy = "user"
	}

	query := `
		INSERT INTO notes (id, api_id, content, type, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at
	`

	err := db.QueryRow(query, note.ID, note.APIID, note.Content, note.Type, note.CreatedBy).Scan(&note.CreatedAt)
	if err != nil {
		log.Printf("Error adding note to API %s: %v", apiID, err)
		http.Error(w, "Failed to add note", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(note)
}

func getConfiguredAPIsHandler(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT a.id, a.name, a.provider, a.description, a.category, c.environment, c.configuration_date
		FROM apis a
		JOIN api_credentials c ON a.id = c.api_id
		WHERE c.is_configured = true
		ORDER BY a.name
	`

	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, "Failed to fetch configured APIs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id, name, provider, description, category, environment string
		var configDate time.Time

		err := rows.Scan(&id, &name, &provider, &description, &category, &environment, &configDate)
		if err != nil {
			continue
		}

		results = append(results, map[string]interface{}{
			"id":                 id,
			"name":               name,
			"provider":           provider,
			"description":        description,
			"category":           category,
			"environment":        environment,
			"configuration_date": configDate,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func markConfiguredHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	var config struct {
		Environment string `json:"environment"`
		Notes       string `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if config.Environment == "" {
		config.Environment = "development"
	}

	query := `
		INSERT INTO api_credentials (api_id, is_configured, environment, configuration_notes, configuration_date)
		VALUES ($1, true, $2, $3, NOW())
		ON CONFLICT (api_id, environment) 
		DO UPDATE SET is_configured = true, configuration_notes = $3, configuration_date = NOW()
	`

	_, err := db.Exec(query, apiID, config.Environment, config.Notes)
	if err != nil {
		http.Error(w, "Failed to mark API as configured", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func requestResearchHandler(w http.ResponseWriter, r *http.Request) {
	var req ResearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Capability == "" {
		http.Error(w, "Capability is required", http.StatusBadRequest)
		return
	}

	// Create research request in database
	researchID := uuid.New().String()
	query := `
		INSERT INTO research_requests (id, capability, requirements, status)
		VALUES ($1, $2, $3, 'queued')
	`

	reqJSON, _ := json.Marshal(req.Requirements)
	_, err := db.Exec(query, researchID, req.Capability, reqJSON)
	if err != nil {
		log.Printf("Error creating research request: %v", err)
		http.Error(w, "Failed to create research request", http.StatusInternalServerError)
		return
	}

	// Trigger research-assistant scenario if available
	go triggerResearchAssistant(researchID, req.Capability, req.Requirements)

	response := ResearchResponse{
		ResearchID:    researchID,
		Status:        "queued",
		EstimatedTime: 300, // 5 minutes estimate
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// triggerResearchAssistant calls the research-assistant API to perform research
func triggerResearchAssistant(researchID string, capability string, requirements map[string]interface{}) {
	// First check if research-assistant is running
	researchPort := getResearchAssistantPort()
	if researchPort == "" {
		log.Printf("Research-assistant not running, cannot trigger research for ID: %s", researchID)
		updateResearchStatus(researchID, "failed", "Research-assistant service not available")
		return
	}

	// Build search query from capability
	searchQuery := fmt.Sprintf("APIs for %s", capability)
	if req, ok := requirements["features"]; ok {
		if features, ok := req.([]string); ok && len(features) > 0 {
			searchQuery += " with " + strings.Join(features, ", ")
		}
	}

	// Call research-assistant search API
	searchReq := map[string]interface{}{
		"query":    searchQuery,
		"category": "web",
		"limit":    10,
	}

	searchJSON, _ := json.Marshal(searchReq)
	resp, err := http.Post(
		fmt.Sprintf("http://localhost:%s/api/search", researchPort),
		"application/json",
		bytes.NewBuffer(searchJSON),
	)
	if err != nil {
		log.Printf("Failed to call research-assistant: %v", err)
		updateResearchStatus(researchID, "failed", "Failed to connect to research service")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Research-assistant returned status %d", resp.StatusCode)
		updateResearchStatus(researchID, "failed", "Research service returned an error")
		return
	}

	// Parse research results and store discovered APIs
	var searchResults map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&searchResults); err != nil {
		log.Printf("Failed to parse research results: %v", err)
		updateResearchStatus(researchID, "failed", "Failed to parse research results")
		return
	}

	// Process and store discovered APIs
	processResearchResults(researchID, capability, searchResults)
}

// getResearchAssistantPort attempts to get the port of the running research-assistant
func getResearchAssistantPort() string {
	// Try using vrooli CLI to get the port
	cmd := "vrooli scenario port research-assistant api 2>/dev/null"
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err == nil && len(out) > 0 {
		return strings.TrimSpace(string(out))
	}

	// Fallback to checking if default port is in use
	// Research-assistant typically uses port range 16000-16999
	for port := 16100; port <= 16110; port++ {
		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
		if err == nil {
			conn.Close()
			// Try to verify it's the research-assistant
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", port))
			if err == nil {
				defer resp.Body.Close()
				var health map[string]interface{}
				if json.NewDecoder(resp.Body).Decode(&health) == nil {
					if service, ok := health["service"].(string); ok && service == "research-assistant" {
						return strconv.Itoa(port)
					}
				}
			}
		}
	}

	return ""
}

// updateResearchStatus updates the status of a research request in the database
func updateResearchStatus(researchID string, status string, errorMsg string) {
	query := `UPDATE research_requests SET status = $1, error_message = $2, updated_at = NOW() WHERE id = $3`
	if _, err := db.Exec(query, status, errorMsg, researchID); err != nil {
		log.Printf("Failed to update research status: %v", err)
	}
}

// processResearchResults processes the results from research-assistant and stores discovered APIs
func processResearchResults(researchID string, capability string, results map[string]interface{}) {
	// Extract search results
	searchResults, ok := results["results"].([]interface{})
	if !ok || len(searchResults) == 0 {
		updateResearchStatus(researchID, "completed", "No APIs found for the requested capability")
		return
	}

	apiCount := 0
	for _, result := range searchResults {
		if res, ok := result.(map[string]interface{}); ok {
			// Extract API information from search result
			title, _ := res["title"].(string)
			urlStr, _ := res["url"].(string)
			snippet, _ := res["snippet"].(string)

			// Try to identify if this is an API documentation page
			if strings.Contains(strings.ToLower(title+snippet), "api") {
				// Create a new API entry
				apiID := uuid.New().String()
				apiName := extractAPIName(title, urlStr)
				provider := extractProvider(urlStr)

				insertQuery := `
					INSERT INTO apis (id, name, provider, description, documentation_url, category, status, created_at, updated_at)
					VALUES ($1, $2, $3, $4, $5, $6, 'active', NOW(), NOW())
					ON CONFLICT (name, provider) DO UPDATE SET 
						description = EXCLUDED.description,
						documentation_url = EXCLUDED.documentation_url,
						updated_at = NOW()
				`

				_, err := db.Exec(insertQuery, apiID, apiName, provider, snippet, urlStr, capability)
				if err == nil {
					apiCount++

					// Add a note about automatic discovery
					noteID := uuid.New().String()
					noteQuery := `
						INSERT INTO notes (id, api_id, content, type, created_by, created_at)
						VALUES ($1, $2, $3, 'tip', 'research-assistant', NOW())
					`
					noteContent := fmt.Sprintf("Automatically discovered via research for '%s' capability", capability)
					db.Exec(noteQuery, noteID, apiID, noteContent)
				}
			}
		}
	}

	// Update research request status
	if apiCount > 0 {
		msg := fmt.Sprintf("Successfully discovered %d APIs", apiCount)
		updateResearchStatus(researchID, "completed", msg)
	} else {
		updateResearchStatus(researchID, "completed", "Research completed but no suitable APIs identified")
	}
}

// extractAPIName attempts to extract a clean API name from title and URL
func extractAPIName(title string, urlStr string) string {
	// Try to extract from title first
	if title != "" {
		// Remove common suffixes
		name := strings.TrimSuffix(title, " API Documentation")
		name = strings.TrimSuffix(name, " API")
		name = strings.TrimSuffix(name, " Documentation")
		name = strings.TrimSuffix(name, " Docs")
		if name != "" {
			return name
		}
	}

	// Fallback to extracting from URL
	u, err := url.Parse(urlStr)
	if err == nil && u.Host != "" {
		parts := strings.Split(u.Host, ".")
		if len(parts) > 0 {
			// Use the first subdomain or domain name
			name := parts[0]
			if name == "www" && len(parts) > 1 {
				name = parts[1]
			}
			return strings.Title(name)
		}
	}

	return "Unknown API"
}

// extractProvider attempts to extract the provider from URL
func extractProvider(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err == nil && u.Host != "" {
		// Remove www. prefix
		host := strings.TrimPrefix(u.Host, "www.")
		// Get domain without TLD
		parts := strings.Split(host, ".")
		if len(parts) > 0 {
			return strings.Title(parts[0])
		}
	}
	return "Unknown Provider"
}

// getEnv removed to prevent hardcoded defaults

// Export handlers
func exportAPIsHandler(w http.ResponseWriter, r *http.Request) {
	format := strings.ToLower(r.URL.Query().Get("format"))
	if format == "" {
		format = "json"
	}

	// Check if database is connected
	if db == nil {
		log.Println("Export handler error: database connection is nil")
		http.Error(w, "Database not connected", http.StatusInternalServerError)
		return
	}

	// Query all APIs with their details
	query := `
		SELECT a.id, a.name, a.provider, a.description, a.category, 
			   a.base_url, a.documentation_url, a.pricing_url, a.status,
			   a.tags, a.capabilities, a.created_at, a.updated_at
		FROM apis a
		ORDER BY a.name
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Export handler database query error: %v", err)
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var apis []API
	for rows.Next() {
		var api API
		var tags, capabilities pq.StringArray
		var docURL, pricingURL sql.NullString
		err := rows.Scan(
			&api.ID, &api.Name, &api.Provider, &api.Description, &api.Category,
			&api.BaseURL, &docURL, &pricingURL, &api.Status,
			&tags, &capabilities, &api.CreatedAt, &api.UpdatedAt,
		)
		if err != nil {
			log.Printf("Export handler row scan error: %v", err)
			continue
		}
		if docURL.Valid {
			api.DocumentationURL = docURL.String
		}
		if pricingURL.Valid {
			api.PricingURL = pricingURL.String
		}
		api.Tags = []string(tags)
		api.Capabilities = []string(capabilities)
		apis = append(apis, api)
	}

	log.Printf("Export handler: Found %d APIs to export", len(apis))

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=api-library.csv")

		// Write CSV header
		fmt.Fprintln(w, "Name,Provider,Category,Status,Description,BaseURL,DocumentationURL,Tags")

		// Write data rows
		for _, api := range apis {
			tags := strings.Join(api.Tags, ";")
			fmt.Fprintf(w, "%s,%s,%s,%s,%s,%s,%s,%s\n",
				api.Name, api.Provider, api.Category, api.Status,
				api.Description, api.BaseURL, api.DocumentationURL, tags)
		}
	default:
		// JSON format (default)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=api-library.json")
		json.NewEncoder(w).Encode(apis)
	}
}

// Category and tag handlers
func getCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT DISTINCT category, COUNT(*) as count
		FROM apis
		WHERE category IS NOT NULL AND category != ''
		GROUP BY category
		ORDER BY count DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type CategoryCount struct {
		Category string `json:"category"`
		Count    int    `json:"count"`
	}

	var categories []CategoryCount
	for rows.Next() {
		var cat CategoryCount
		if err := rows.Scan(&cat.Category, &cat.Count); err == nil {
			categories = append(categories, cat)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}

func getTagsHandler(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT UNNEST(tags) as tag, COUNT(*) as count
		FROM apis
		WHERE tags IS NOT NULL
		GROUP BY tag
		ORDER BY count DESC
		LIMIT 50
	`

	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type TagCount struct {
		Tag   string `json:"tag"`
		Count int    `json:"count"`
	}

	var tags []TagCount
	for rows.Next() {
		var tag TagCount
		if err := rows.Scan(&tag.Tag, &tag.Count); err == nil {
			tags = append(tags, tag)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tags)
}

func updateAPITagsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	var request struct {
		Tags []string `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := `
		UPDATE apis 
		SET tags = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, tags
	`

	var updatedAPI struct {
		ID   string         `json:"id"`
		Name string         `json:"name"`
		Tags pq.StringArray `json:"tags"`
	}

	err := db.QueryRow(query, apiID, pq.Array(request.Tags)).Scan(
		&updatedAPI.ID, &updatedAPI.Name, &updatedAPI.Tags,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "API not found", http.StatusNotFound)
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedAPI)
}

func updateAPIStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	var request struct {
		Status     string  `json:"status"`
		SunsetDate *string `json:"sunset_date,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate status
	validStatuses := []string{"active", "deprecated", "sunset", "beta"}
	isValid := false
	for _, s := range validStatuses {
		if request.Status == s {
			isValid = true
			break
		}
	}

	if !isValid {
		http.Error(w, "Invalid status. Must be one of: active, deprecated, sunset, beta", http.StatusBadRequest)
		return
	}

	query := `
		UPDATE apis 
		SET status = $2, sunset_date = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, status, sunset_date
	`

	var updatedAPI struct {
		ID         string     `json:"id"`
		Name       string     `json:"name"`
		Status     string     `json:"status"`
		SunsetDate *time.Time `json:"sunset_date,omitempty"`
	}

	err := db.QueryRow(query, apiID, request.Status, request.SunsetDate).Scan(
		&updatedAPI.ID, &updatedAPI.Name, &updatedAPI.Status, &updatedAPI.SunsetDate,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "API not found", http.StatusNotFound)
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedAPI)
}

// Recipe handlers

func getRecipesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	query := `
		SELECT id, name, description, use_case, steps, prerequisites,
			   expected_outcome, estimated_time_minutes, difficulty_level,
			   related_api_ids, times_used, success_rate, rating, rating_count,
			   created_at, updated_at, created_by, last_updated_by, tags
		FROM integration_recipes
		WHERE api_id = $1
		ORDER BY rating DESC, times_used DESC
	`

	rows, err := db.Query(query, apiID)
	if err != nil {
		http.Error(w, "Failed to fetch recipes", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var recipes []IntegrationRecipe
	for rows.Next() {
		var r IntegrationRecipe
		var steps, prerequisites sql.NullString
		var relatedIDs, tags sql.NullString

		err := rows.Scan(
			&r.ID, &r.Name, &r.Description, &r.UseCase,
			&steps, &prerequisites, &r.ExpectedOutcome,
			&r.EstimatedTimeMinutes, &r.DifficultyLevel,
			&relatedIDs, &r.TimesUsed, &r.SuccessRate,
			&r.Rating, &r.RatingCount, &r.CreatedAt,
			&r.UpdatedAt, &r.CreatedBy, &r.LastUpdatedBy, &tags,
		)
		if err != nil {
			continue
		}

		r.APIID = apiID
		if steps.Valid {
			json.Unmarshal([]byte(steps.String), &r.Steps)
		}
		if prerequisites.Valid {
			json.Unmarshal([]byte(prerequisites.String), &r.Prerequisites)
		}
		if relatedIDs.Valid {
			json.Unmarshal([]byte(relatedIDs.String), &r.RelatedAPIIDs)
		}
		if tags.Valid {
			json.Unmarshal([]byte(tags.String), &r.Tags)
		}

		recipes = append(recipes, r)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"recipes": recipes,
		"count":   len(recipes),
	})
}

func createRecipeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	var recipe IntegrationRecipe
	if err := json.NewDecoder(r.Body).Decode(&recipe); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if recipe.Name == "" || recipe.Description == "" || recipe.UseCase == "" {
		http.Error(w, "Name, description, and use_case are required", http.StatusBadRequest)
		return
	}

	recipe.ID = uuid.New().String()
	recipe.APIID = apiID
	recipe.CreatedAt = time.Now()
	recipe.UpdatedAt = time.Now()
	if recipe.CreatedBy == "" {
		recipe.CreatedBy = "api"
	}
	recipe.LastUpdatedBy = recipe.CreatedBy
	if recipe.DifficultyLevel == "" {
		recipe.DifficultyLevel = "intermediate"
	}

	stepsJSON, _ := json.Marshal(recipe.Steps)
	prereqJSON, _ := json.Marshal(recipe.Prerequisites)
	relatedJSON, _ := json.Marshal(recipe.RelatedAPIIDs)
	tagsJSON, _ := json.Marshal(recipe.Tags)

	query := `
		INSERT INTO integration_recipes (
			id, api_id, name, description, use_case, steps, prerequisites,
			expected_outcome, estimated_time_minutes, difficulty_level,
			related_api_ids, tags, created_by, last_updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id
	`

	err := db.QueryRow(
		query, recipe.ID, apiID, recipe.Name, recipe.Description, recipe.UseCase,
		string(stepsJSON), string(prereqJSON), recipe.ExpectedOutcome,
		recipe.EstimatedTimeMinutes, recipe.DifficultyLevel,
		string(relatedJSON), string(tagsJSON), recipe.CreatedBy, recipe.LastUpdatedBy,
	).Scan(&recipe.ID)

	if err != nil {
		http.Error(w, "Failed to create recipe", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(recipe)
}

func getRecipeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	recipeID := vars["id"]

	query := `
		SELECT id, api_id, name, description, use_case, steps, prerequisites,
			   expected_outcome, estimated_time_minutes, difficulty_level,
			   related_api_ids, times_used, success_rate, rating, rating_count,
			   created_at, updated_at, created_by, last_updated_by, tags
		FROM integration_recipes
		WHERE id = $1
	`

	var recipe IntegrationRecipe
	var steps, prerequisites sql.NullString
	var relatedIDs, tags sql.NullString

	err := db.QueryRow(query, recipeID).Scan(
		&recipe.ID, &recipe.APIID, &recipe.Name, &recipe.Description, &recipe.UseCase,
		&steps, &prerequisites, &recipe.ExpectedOutcome,
		&recipe.EstimatedTimeMinutes, &recipe.DifficultyLevel,
		&relatedIDs, &recipe.TimesUsed, &recipe.SuccessRate,
		&recipe.Rating, &recipe.RatingCount, &recipe.CreatedAt,
		&recipe.UpdatedAt, &recipe.CreatedBy, &recipe.LastUpdatedBy, &tags,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Recipe not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to fetch recipe", http.StatusInternalServerError)
		return
	}

	if steps.Valid {
		json.Unmarshal([]byte(steps.String), &recipe.Steps)
	}
	if prerequisites.Valid {
		json.Unmarshal([]byte(prerequisites.String), &recipe.Prerequisites)
	}
	if relatedIDs.Valid {
		json.Unmarshal([]byte(relatedIDs.String), &recipe.RelatedAPIIDs)
	}
	if tags.Valid {
		json.Unmarshal([]byte(tags.String), &recipe.Tags)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipe)
}

func updateRecipeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	recipeID := vars["id"]

	var recipe IntegrationRecipe
	if err := json.NewDecoder(r.Body).Decode(&recipe); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	recipe.UpdatedAt = time.Now()
	if recipe.LastUpdatedBy == "" {
		recipe.LastUpdatedBy = "api"
	}

	stepsJSON, _ := json.Marshal(recipe.Steps)
	prereqJSON, _ := json.Marshal(recipe.Prerequisites)
	relatedJSON, _ := json.Marshal(recipe.RelatedAPIIDs)
	tagsJSON, _ := json.Marshal(recipe.Tags)

	query := `
		UPDATE integration_recipes
		SET name = $2, description = $3, use_case = $4, steps = $5,
			prerequisites = $6, expected_outcome = $7, estimated_time_minutes = $8,
			difficulty_level = $9, related_api_ids = $10, tags = $11,
			updated_at = $12, last_updated_by = $13
		WHERE id = $1
		RETURNING id
	`

	err := db.QueryRow(
		query, recipeID, recipe.Name, recipe.Description, recipe.UseCase,
		string(stepsJSON), string(prereqJSON), recipe.ExpectedOutcome,
		recipe.EstimatedTimeMinutes, recipe.DifficultyLevel,
		string(relatedJSON), string(tagsJSON),
		recipe.UpdatedAt, recipe.LastUpdatedBy,
	).Scan(&recipe.ID)

	if err == sql.ErrNoRows {
		http.Error(w, "Recipe not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to update recipe", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipe)
}

func deleteRecipeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	recipeID := vars["id"]

	query := `DELETE FROM integration_recipes WHERE id = $1`
	result, err := db.Exec(query, recipeID)
	if err != nil {
		http.Error(w, "Failed to delete recipe", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		http.Error(w, "Recipe not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getSuccessfulRecipesHandler(w http.ResponseWriter, r *http.Request) {
	minSuccessRate := 0.7
	if rate := r.URL.Query().Get("min_success_rate"); rate != "" {
		if parsed, err := strconv.ParseFloat(rate, 64); err == nil {
			minSuccessRate = parsed
		}
	}

	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	query := `
		SELECT id, api_id, name, description, use_case, 
			   success_rate, rating, times_used, difficulty_level
		FROM integration_recipes
		WHERE success_rate >= $1 AND times_used >= 10
		ORDER BY success_rate DESC, rating DESC
		LIMIT $2
	`

	rows, err := db.Query(query, minSuccessRate, limit)
	if err != nil {
		http.Error(w, "Failed to fetch successful recipes", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var recipes []map[string]interface{}
	for rows.Next() {
		var id, apiID, name, description, useCase, difficulty string
		var successRate, rating float64
		var timesUsed int

		err := rows.Scan(&id, &apiID, &name, &description, &useCase,
			&successRate, &rating, &timesUsed, &difficulty)
		if err != nil {
			continue
		}

		recipes = append(recipes, map[string]interface{}{
			"id":               id,
			"api_id":           apiID,
			"name":             name,
			"description":      description,
			"use_case":         useCase,
			"success_rate":     successRate,
			"rating":           rating,
			"times_used":       timesUsed,
			"difficulty_level": difficulty,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"recipes": recipes,
		"count":   len(recipes),
	})
}

// compareAPIsHandler generates a comparison matrix for multiple APIs
func compareAPIsHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		APIIDs     []string `json:"api_ids"`
		Attributes []string `json:"attributes,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.APIIDs) < 2 {
		http.Error(w, "At least 2 API IDs required for comparison", http.StatusBadRequest)
		return
	}

	// Default attributes if none specified
	if len(req.Attributes) == 0 {
		req.Attributes = []string{"pricing", "rate_limits", "auth_type", "regions", "features", "support"}
	}

	// Build comparison matrix
	matrix := make(map[string]map[string]interface{})

	for _, apiID := range req.APIIDs {
		// Get API details
		var api API
		var pricingTiers []PricingTier

		// Get API info
		var docURL, pricingURL sql.NullString
		err := db.QueryRow(`
			SELECT id, name, provider, description, base_url, documentation_url, 
			       pricing_url, category, status, auth_type, version
			FROM apis WHERE id = $1`, apiID).Scan(
			&api.ID, &api.Name, &api.Provider, &api.Description, &api.BaseURL,
			&docURL, &pricingURL, &api.Category,
			&api.Status, &api.AuthType, &api.Version,
		)

		if docURL.Valid {
			api.DocumentationURL = docURL.String
		}
		if pricingURL.Valid {
			api.PricingURL = pricingURL.String
		}

		if err != nil {
			continue // Skip APIs that don't exist
		}

		// Get pricing tiers
		pRows, _ := db.Query(`
			SELECT name, price_per_request, price_per_mb, monthly_cost, free_tier_requests
			FROM pricing_tiers WHERE api_id = $1`, apiID)
		if pRows != nil {
			for pRows.Next() {
				var tier PricingTier
				pRows.Scan(&tier.Name, &tier.PricePerRequest, &tier.PricePerMB,
					&tier.MonthlyCost, &tier.FreeTierRequests)
				pricingTiers = append(pricingTiers, tier)
			}
			pRows.Close()
		}

		// Build comparison data
		apiData := make(map[string]interface{})
		apiData["name"] = api.Name
		apiData["provider"] = api.Provider

		for _, attr := range req.Attributes {
			switch attr {
			case "pricing":
				if len(pricingTiers) > 0 {
					apiData["pricing"] = pricingTiers[0] // Show primary tier
				} else {
					apiData["pricing"] = "No pricing data"
				}
			case "rate_limits":
				// Get from endpoints table if available
				var rateLimit string
				db.QueryRow(`
					SELECT rate_limit->>'limit' 
					FROM endpoints 
					WHERE api_id = $1 
					LIMIT 1`, apiID).Scan(&rateLimit)
				apiData["rate_limits"] = rateLimit
			case "auth_type":
				apiData["auth_type"] = api.AuthType
			case "status":
				apiData["status"] = api.Status
			case "category":
				apiData["category"] = api.Category
			default:
				apiData[attr] = "N/A"
			}
		}

		matrix[apiID] = apiData
	}

	// Generate comparison summary
	response := map[string]interface{}{
		"comparison_matrix": matrix,
		"attributes":        req.Attributes,
		"api_count":         len(matrix),
		"generated_at":      time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// trackUsageHandler tracks API usage for analytics
func trackUsageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	var req struct {
		Requests int     `json:"requests"`
		DataMB   float64 `json:"data_mb"`
		Errors   int     `json:"errors,omitempty"`
		UserID   string  `json:"user_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Insert usage data
	_, err := db.Exec(`
		INSERT INTO api_usage_logs (api_id, user_id, requests, data_mb, errors, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		apiID, req.UserID, req.Requests, req.DataMB, req.Errors, time.Now())

	if err != nil {
		log.Printf("Failed to track usage: %v", err)
		http.Error(w, "Failed to track usage", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "usage tracked",
		"api_id": apiID,
	})
}

// getAnalyticsHandler returns usage analytics for an API
func getAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	// Get time range from query params
	timeRange := r.URL.Query().Get("range")
	if timeRange == "" {
		timeRange = "30d"
	}

	var since time.Time
	switch timeRange {
	case "24h":
		since = time.Now().Add(-24 * time.Hour)
	case "7d":
		since = time.Now().Add(-7 * 24 * time.Hour)
	case "30d":
		since = time.Now().Add(-30 * 24 * time.Hour)
	default:
		since = time.Now().Add(-30 * 24 * time.Hour)
	}

	// Get aggregated analytics
	var totalRequests, totalErrors int
	var totalDataMB float64
	var uniqueUsers int

	err := db.QueryRow(`
		SELECT 
			COALESCE(SUM(requests), 0) as total_requests,
			COALESCE(SUM(data_mb), 0) as total_data_mb,
			COALESCE(SUM(errors), 0) as total_errors,
			COUNT(DISTINCT user_id) as unique_users
		FROM api_usage_logs
		WHERE api_id = $1 AND timestamp >= $2`,
		apiID, since).Scan(&totalRequests, &totalDataMB, &totalErrors, &uniqueUsers)

	if err != nil {
		log.Printf("Failed to get analytics: %v", err)
		http.Error(w, "Failed to retrieve analytics", http.StatusInternalServerError)
		return
	}

	// Calculate error rate
	errorRate := float64(0)
	if totalRequests > 0 {
		errorRate = float64(totalErrors) / float64(totalRequests) * 100
	}

	// Get daily breakdown
	rows, err := db.Query(`
		SELECT DATE(timestamp) as day, 
		       SUM(requests) as daily_requests,
		       SUM(data_mb) as daily_data_mb
		FROM api_usage_logs
		WHERE api_id = $1 AND timestamp >= $2
		GROUP BY DATE(timestamp)
		ORDER BY day DESC`, apiID, since)

	if err != nil {
		log.Printf("Failed to get daily analytics: %v", err)
	}
	defer rows.Close()

	var dailyStats []map[string]interface{}
	for rows.Next() {
		var day time.Time
		var requests int
		var dataMB float64

		if err := rows.Scan(&day, &requests, &dataMB); err == nil {
			dailyStats = append(dailyStats, map[string]interface{}{
				"date":     day.Format("2006-01-02"),
				"requests": requests,
				"data_mb":  dataMB,
			})
		}
	}

	response := map[string]interface{}{
		"api_id":     apiID,
		"time_range": timeRange,
		"since":      since,
		"summary": map[string]interface{}{
			"total_requests": totalRequests,
			"total_data_mb":  totalDataMB,
			"total_errors":   totalErrors,
			"error_rate":     errorRate,
			"unique_users":   uniqueUsers,
		},
		"daily_breakdown": dailyStats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getRecommendationsHandler provides API recommendations based on usage patterns
func getRecommendationsHandler(w http.ResponseWriter, r *http.Request) {
	capability := r.URL.Query().Get("capability")
	maxPrice := r.URL.Query().Get("max_price")

	var recommendations []map[string]interface{}

	// Get popular APIs based on usage (simplified to avoid column issues)
	query := `
		SELECT a.id, a.name, a.provider, a.description, 
		       COUNT(u.id) as usage_count,
		       0.0 as avg_error_rate
		FROM apis a
		LEFT JOIN api_usage_logs u ON a.id = u.api_id
		WHERE a.status = 'active'`

	args := []interface{}{}
	argCount := 0

	if capability != "" {
		argCount++
		query += fmt.Sprintf(" AND (a.description ILIKE $%d OR a.name ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+capability+"%")
	}

	if maxPrice != "" {
		if price, err := strconv.ParseFloat(maxPrice, 64); err == nil {
			argCount++
			query += fmt.Sprintf(` AND EXISTS (
				SELECT 1 FROM pricing_tiers pt 
				WHERE pt.api_id = a.id 
				AND pt.price_per_request <= $%d
			)`, argCount)
			args = append(args, price)
		}
	}

	query += ` GROUP BY a.id, a.name, a.provider, a.description
	           ORDER BY usage_count DESC, avg_error_rate ASC
	           LIMIT 10`

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Failed to get recommendations: %v", err)
		http.Error(w, "Failed to generate recommendations", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, name, provider, description string
		var usageCount int
		var avgErrorRate float64

		if err := rows.Scan(&id, &name, &provider, &description, &usageCount, &avgErrorRate); err == nil {
			// Get pricing info
			var lowestPrice float64
			db.QueryRow(`
				SELECT MIN(price_per_request) 
				FROM pricing_tiers 
				WHERE api_id = $1`, id).Scan(&lowestPrice)

			recommendation := map[string]interface{}{
				"api_id":               id,
				"name":                 name,
				"provider":             provider,
				"description":          description,
				"usage_count":          usageCount,
				"reliability":          fmt.Sprintf("%.1f%%", (1-avgErrorRate)*100),
				"lowest_price":         lowestPrice,
				"recommendation_score": calculateRecommendationScore(usageCount, avgErrorRate, lowestPrice),
			}

			recommendations = append(recommendations, recommendation)
		}
	}

	// If no results yet (no usage data), fall back to basic API listing
	if len(recommendations) == 0 {
		fallbackQuery := `
			SELECT id, name, provider, description, category
			FROM apis
			WHERE status != 'deprecated' AND status != 'sunset'`

		fallbackArgs := []interface{}{}
		argCount := 0

		if capability != "" {
			argCount++
			fallbackQuery += fmt.Sprintf(" AND (description ILIKE $%d OR name ILIKE $%d OR category ILIKE $%d)", argCount, argCount, argCount)
			fallbackArgs = append(fallbackArgs, "%"+capability+"%")
		}

		fallbackQuery += " ORDER BY name ASC LIMIT 10"

		fRows, err := db.Query(fallbackQuery, fallbackArgs...)
		if err == nil {
			defer fRows.Close()
			for fRows.Next() {
				var id, name, provider, description, category string
				if err := fRows.Scan(&id, &name, &provider, &description, &category); err == nil {
					recommendation := map[string]interface{}{
						"api_id":               id,
						"name":                 name,
						"provider":             provider,
						"description":          description,
						"category":             category,
						"usage_count":          0,
						"reliability":          "No data",
						"lowest_price":         0,
						"recommendation_score": 50.0,
					}
					recommendations = append(recommendations, recommendation)
				}
			}
		}
	}

	response := map[string]interface{}{
		"recommendations": recommendations,
		"criteria": map[string]interface{}{
			"capability": capability,
			"max_price":  maxPrice,
		},
		"generated_at": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper function to calculate recommendation score
func calculateRecommendationScore(usageCount int, errorRate, price float64) float64 {
	// Score based on: usage popularity (40%), reliability (40%), price (20%)
	usageScore := math.Min(float64(usageCount)/100, 1.0) * 40
	reliabilityScore := (1 - errorRate) * 40
	priceScore := 20.0
	if price > 0 {
		priceScore = math.Max(0, 20-price*100) // Lower price = higher score
	}

	return usageScore + reliabilityScore + priceScore
}

// Code Generator Integration Handlers

// getCodeGenSpecHandler returns API specification optimized for code generation
func getCodeGenSpecHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	// Get API details
	var api API
	var endpoints []map[string]interface{}

	// Fetch API info with proper NULL handling
	var (
		docURL     sql.NullString
		pricingURL sql.NullString
		authType   sql.NullString
		version    sql.NullString
	)
	
	err := db.QueryRow(`
		SELECT id, name, provider, COALESCE(description, ''), COALESCE(base_url, ''), 
		       documentation_url, pricing_url, COALESCE(category, 'general'), 
		       COALESCE(status, 'active'), auth_type, version
		FROM apis WHERE id = $1`,
		apiID).Scan(
		&api.ID, &api.Name, &api.Provider, &api.Description, &api.BaseURL,
		&docURL, &pricingURL, &api.Category,
		&api.Status, &authType, &version,
	)
	
	// Handle nullable fields
	if docURL.Valid {
		api.DocumentationURL = docURL.String
	}
	if pricingURL.Valid {
		api.PricingURL = pricingURL.String
	}
	if authType.Valid {
		api.AuthType = authType.String
	} else {
		api.AuthType = "none"
	}
	if version.Valid {
		api.Version = version.String
	} else {
		api.Version = "1.0.0"
	}

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "API not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Fetch endpoints
	rows, err := db.Query(`
		SELECT path, method, description, parameters, response_schema, rate_limit
		FROM endpoints WHERE api_id = $1
		ORDER BY method, path`, apiID)

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var endpoint map[string]interface{}
		var path, method, description string
		var parameters, responseSchema, rateLimit sql.NullString

		err := rows.Scan(&path, &method, &description, &parameters, &responseSchema, &rateLimit)
		if err != nil {
			continue
		}

		endpoint = map[string]interface{}{
			"path":        path,
			"method":      method,
			"description": description,
		}

		if parameters.Valid {
			var params interface{}
			json.Unmarshal([]byte(parameters.String), &params)
			endpoint["parameters"] = params
		}

		if responseSchema.Valid {
			var schema interface{}
			json.Unmarshal([]byte(responseSchema.String), &schema)
			endpoint["response_schema"] = schema
		}

		if rateLimit.Valid {
			var limit interface{}
			json.Unmarshal([]byte(rateLimit.String), &limit)
			endpoint["rate_limit"] = limit
		}

		endpoints = append(endpoints, endpoint)
	}

	// Fetch code snippets for this API
	var snippets []map[string]interface{}
	snippetRows, err := db.Query(`
		SELECT id, language, code, description, dependencies, created_at, vote_count
		FROM snippets 
		WHERE api_id = $1 AND vote_count > 0
		ORDER BY vote_count DESC
		LIMIT 5`, apiID)

	if err == nil {
		defer snippetRows.Close()
		for snippetRows.Next() {
			var snippet map[string]interface{}
			var id, language, code, description string
			var dependencies sql.NullString
			var createdAt time.Time
			var voteCount int

			err := snippetRows.Scan(&id, &language, &code, &description, &dependencies, &createdAt, &voteCount)
			if err != nil {
				continue
			}

			snippet = map[string]interface{}{
				"id":          id,
				"language":    language,
				"code":        code,
				"description": description,
				"vote_count":  voteCount,
			}

			if dependencies.Valid {
				var deps interface{}
				json.Unmarshal([]byte(dependencies.String), &deps)
				snippet["dependencies"] = deps
			}

			snippets = append(snippets, snippet)
		}
	}

	// Build code generation specification
	codeGenSpec := map[string]interface{}{
		"api": map[string]interface{}{
			"id":            api.ID,
			"name":          api.Name,
			"provider":      api.Provider,
			"description":   api.Description,
			"base_url":      api.BaseURL,
			"documentation": api.DocumentationURL,
			"auth_type":     api.AuthType,
			"version":       api.Version,
		},
		"endpoints":     endpoints,
		"code_snippets": snippets,
		"generation_hints": map[string]interface{}{
			"preferred_languages": []string{"python", "javascript", "go", "java"},
			"auth_pattern":        getAuthPattern(api.AuthType),
			"error_handling":      "exponential_backoff_recommended",
			"sdk_available":       checkSDKAvailability(api.Name),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(codeGenSpec)
}

// searchCodeGenAPIsHandler searches for APIs suitable for code generation
func searchCodeGenAPIsHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Capability string   `json:"capability"`
		Languages  []string `json:"languages"`
		MaxPrice   float64  `json:"max_price"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Use semantic search to find relevant APIs
	searchReq := SearchRequest{
		Query: req.Capability,
		Limit: 10,
	}

	if req.MaxPrice > 0 {
		searchReq.Filters = &SearchFilters{
			MaxPrice: &req.MaxPrice,
		}
	}

	// Perform search using existing search logic
	params := normalizeSearchParams(searchReq)
	searchResults, _, err := performSearch(context.Background(), params)
	if err != nil {
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	var results []interface{}
	for _, r := range searchResults {
		results = append(results, r)
	}

	// Enhance results with code generation metadata
	var enhancedResults []map[string]interface{}
	for _, result := range results {
		apiMap, ok := result.(map[string]interface{})
		if !ok {
			continue
		}

		apiID, ok := apiMap["id"].(string)
		if !ok {
			continue
		}

		// Check if we have code snippets for requested languages
		var supportedLanguages []string
		if len(req.Languages) > 0 {
			for _, lang := range req.Languages {
				var count int
				db.QueryRow(`
					SELECT COUNT(*) FROM snippets 
					WHERE api_id = $1 AND language = $2`,
					apiID, lang).Scan(&count)

				if count > 0 {
					supportedLanguages = append(supportedLanguages, lang)
				}
			}
		}

		apiMap["code_generation"] = map[string]interface{}{
			"supported_languages": supportedLanguages,
			"snippet_count":       getSnippetCount(apiID),
			"sdk_available":       checkSDKAvailability(apiMap["name"].(string)),
		}

		enhancedResults = append(enhancedResults, apiMap)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": enhancedResults,
		"total":   len(enhancedResults),
	})
}

// getCodeGenTemplatesHandler returns code generation templates for a language
func getCodeGenTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	language := vars["language"]

	// Define templates for common patterns
	templates := map[string]map[string]interface{}{
		"python": {
			"client_class": `import requests
from typing import Dict, Any, Optional

class {api_name}Client:
    def __init__(self, api_key: str, base_url: str = "{base_url}"):
        self.api_key = api_key
        self.base_url = base_url
        self.session = requests.Session()
        self.session.headers.update({auth_header})
    
    def _request(self, method: str, endpoint: str, **kwargs) -> Dict[str, Any]:
        url = f"{self.base_url}{endpoint}"
        response = self.session.request(method, url, **kwargs)
        response.raise_for_status()
        return response.json()`,
			"auth_patterns": map[string]string{
				"bearer":  `{"Authorization": f"Bearer {self.api_key}"}`,
				"api_key": `{"X-API-Key": self.api_key}`,
				"basic":   `{"Authorization": f"Basic {self.api_key}"}`,
			},
		},
		"javascript": {
			"client_class": `class {api_name}Client {
    constructor(apiKey, baseUrl = '{base_url}') {
        this.apiKey = apiKey;
        this.baseUrl = baseUrl;
        this.headers = {auth_header};
    }
    
    async request(method, endpoint, options = {}) {
        const url = ` + "`${this.baseUrl}${endpoint}`" + `;
        const response = await fetch(url, {
            method,
            headers: this.headers,
            ...options
        });
        
        if (!response.ok) {
            throw new Error(` + "`HTTP error! status: ${response.status}`" + `);
        }
        
        return await response.json();
    }
}`,
			"auth_patterns": map[string]string{
				"bearer":  "'Authorization': `Bearer ${this.apiKey}`",
				"api_key": "'X-API-Key': this.apiKey",
				"basic":   "'Authorization': `Basic ${this.apiKey}`",
			},
		},
		"go": {
			"client_struct": `package {package_name}

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type Client struct {
    APIKey  string
    BaseURL string
    client  *http.Client
}

func NewClient(apiKey string) *Client {
    return &Client{
        APIKey:  apiKey,
        BaseURL: "{base_url}",
        client:  &http.Client{},
    }
}

func (c *Client) request(method, endpoint string, body interface{}) ([]byte, error) {
    url := c.BaseURL + endpoint
    
    var reqBody []byte
    if body != nil {
        var err error
        reqBody, err = json.Marshal(body)
        if err != nil {
            return nil, err
        }
    }
    
    req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
    if err != nil {
        return nil, err
    }
    
    {auth_header}
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
    }
    
    return io.ReadAll(resp.Body)
}`,
			"auth_patterns": map[string]string{
				"bearer":  `req.Header.Set("Authorization", "Bearer " + c.APIKey)`,
				"api_key": `req.Header.Set("X-API-Key", c.APIKey)`,
				"basic":   `req.Header.Set("Authorization", "Basic " + c.APIKey)`,
			},
		},
	}

	template, exists := templates[language]
	if !exists {
		http.Error(w, "Language not supported", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"language":             language,
		"templates":            template,
		"supported_auth_types": []string{"bearer", "api_key", "basic"},
	})
}

// Helper functions for code generation

func getAuthPattern(authType string) string {
	patterns := map[string]string{
		"bearer_token": "Authorization: Bearer <token>",
		"api_key":      "X-API-Key: <key>",
		"basic_auth":   "Authorization: Basic <base64>",
		"oauth2":       "OAuth 2.0 flow required",
	}

	if pattern, ok := patterns[authType]; ok {
		return pattern
	}
	return "Custom authentication"
}

func checkSDKAvailability(apiName string) bool {
	// Check if known SDKs exist for popular APIs
	knownSDKs := map[string]bool{
		"OpenAI":       true,
		"Stripe":       true,
		"Twilio":       true,
		"SendGrid":     true,
		"AWS S3":       true,
		"Firebase":     true,
		"Google Cloud": true,
	}

	return knownSDKs[apiName]
}

func getSnippetCount(apiID string) int {
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM snippets WHERE api_id = $1`, apiID).Scan(&count)
	return count
}

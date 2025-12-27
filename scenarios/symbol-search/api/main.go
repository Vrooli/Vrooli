package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type Character struct {
	Codepoint      string                 `json:"codepoint" db:"codepoint"`
	Decimal        int                    `json:"decimal" db:"decimal"`
	Name           string                 `json:"name" db:"name"`
	Category       string                 `json:"category" db:"category"`
	Block          string                 `json:"block" db:"block"`
	UnicodeVersion string                 `json:"unicode_version" db:"unicode_version"`
	Description    *string                `json:"description,omitempty" db:"description"`
	HTMLEntity     *string                `json:"html_entity,omitempty" db:"html_entity"`
	CSSContent     *string                `json:"css_content,omitempty" db:"css_content"`
	Properties     map[string]interface{} `json:"properties,omitempty" db:"properties"`
}

type CharacterBlock struct {
	ID             int    `json:"id" db:"id"`
	Name           string `json:"name" db:"name"`
	StartCodepoint int    `json:"start_codepoint" db:"start_codepoint"`
	EndCodepoint   int    `json:"end_codepoint" db:"end_codepoint"`
	Description    string `json:"description" db:"description"`
	CharacterCount int    `json:"character_count,omitempty"`
}

type Category struct {
	Code           string `json:"code" db:"code"`
	Name           string `json:"name" db:"name"`
	Description    string `json:"description" db:"description"`
	CharacterCount int    `json:"character_count,omitempty"`
}

type SearchResponse struct {
	Characters     []Character            `json:"characters"`
	Total          int                    `json:"total"`
	QueryTimeMs    float64                `json:"query_time_ms"`
	FiltersApplied map[string]interface{} `json:"filters_applied"`
}

type CharacterDetailResponse struct {
	Character         Character   `json:"character"`
	RelatedCharacters []Character `json:"related_characters,omitempty"`
	UsageExamples     []string    `json:"usage_examples,omitempty"`
}

type BulkRangeRequest struct {
	Ranges []struct {
		Start  string `json:"start"`
		End    string `json:"end"`
		Format string `json:"format,omitempty"`
	} `json:"ranges"`
}

type BulkRangeResponse struct {
	Characters      []Character `json:"characters"`
	TotalCharacters int         `json:"total_characters"`
	RangesProcessed int         `json:"ranges_processed"`
}

type API struct {
	db *sql.DB
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "symbol-search",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Connect to database with exponential backoff
	db, err := database.Connect(context.Background(), database.Config{
		Driver: "postgres",
	})
	if err != nil {
		log.Fatalf("❌ Database connection failed: %v", err)
	}

	log.Println("🎉 Database connection pool established successfully!")

	api := &API{db: db}

	bootstrapCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := api.ensureUnicodeData(bootstrapCtx); err != nil {
		log.Fatalf("failed to populate unicode data: %v", err)
	}

	// Initialize Gin router
	router := gin.Default()

	// Enable CORS
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check endpoint
	router.GET("/health", gin.WrapF(health.New().Version("1.0.0").Check(health.DB(api.db), health.Critical).Handler()))

	// API routes
	apiGroup := router.Group("/api")
	{
		apiGroup.GET("/search", api.searchCharacters)
		apiGroup.GET("/character/:codepoint", api.getCharacterDetail)
		apiGroup.GET("/categories", api.getCategories)
		apiGroup.GET("/blocks", api.getBlocks)
		apiGroup.POST("/bulk/range", api.getBulkRange)
	}

	log.Printf("Starting Symbol Search API server")
	if err := server.Run(server.Config{
		Handler: router,
		Cleanup: func(ctx context.Context) error {
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func (api *API) searchCharacters(c *gin.Context) {
	start := time.Now()

	// Parse query parameters
	query := c.DefaultQuery("q", "")
	category := c.Query("category")
	block := c.Query("block")
	unicodeVersion := c.Query("unicode_version")
	limitStr := c.DefaultQuery("limit", "100")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 1000 {
		limit = 100
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Build SQL query
	var whereClauses []string
	var args []interface{}
	argIndex := 1

	if query != "" {
		whereClauses = append(whereClauses,
			fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d OR codepoint ILIKE $%d)",
				argIndex, argIndex+1, argIndex+2))
		searchTerm := "%" + query + "%"
		args = append(args, searchTerm, searchTerm, searchTerm)
		argIndex += 3
	}

	if category != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, category)
		argIndex++
	}

	if block != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("block = $%d", argIndex))
		args = append(args, block)
		argIndex++
	}

	if unicodeVersion != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("unicode_version = $%d", argIndex))
		args = append(args, unicodeVersion)
		argIndex++
	}

	// Build WHERE clause
	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Count total results
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM characters %s", whereSQL)
	var total int
	err = api.db.QueryRow(countSQL, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to count characters",
		})
		return
	}

	// Get paginated results
	searchSQL := fmt.Sprintf(`
		SELECT codepoint, decimal, name, category, block, unicode_version, 
			   description, html_entity, css_content, properties
		FROM characters %s 
		ORDER BY 
			CASE WHEN name ILIKE $%d THEN 1 ELSE 2 END,
			LENGTH(name), 
			name
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIndex, argIndex+1, argIndex+2)

	// Add exact name match for ranking
	if query != "" {
		args = append(args, "%"+query+"%", limit, offset)
	} else {
		args = append(args, "", limit, offset)
	}

	rows, err := api.db.Query(searchSQL, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to search characters",
		})
		return
	}
	defer rows.Close()

	var characters []Character
	for rows.Next() {
		var char Character
		var propertiesJSON []byte

		err := rows.Scan(
			&char.Codepoint, &char.Decimal, &char.Name, &char.Category,
			&char.Block, &char.UnicodeVersion, &char.Description,
			&char.HTMLEntity, &char.CSSContent, &propertiesJSON,
		)
		if err != nil {
			log.Printf("Error scanning character: %v", err)
			continue
		}

		// Parse properties JSON
		if len(propertiesJSON) > 0 {
			json.Unmarshal(propertiesJSON, &char.Properties)
		}

		characters = append(characters, char)
	}

	queryTime := time.Since(start).Seconds() * 1000

	// Build filters applied
	filtersApplied := make(map[string]interface{})
	if query != "" {
		filtersApplied["query"] = query
	}
	if category != "" {
		filtersApplied["category"] = category
	}
	if block != "" {
		filtersApplied["block"] = block
	}
	if unicodeVersion != "" {
		filtersApplied["unicode_version"] = unicodeVersion
	}
	filtersApplied["limit"] = limit
	filtersApplied["offset"] = offset

	response := SearchResponse{
		Characters:     characters,
		Total:          total,
		QueryTimeMs:    queryTime,
		FiltersApplied: filtersApplied,
	}

	c.JSON(http.StatusOK, response)
}

func (api *API) getCharacterDetail(c *gin.Context) {
	codepoint := c.Param("codepoint")

	// Handle both Unicode format (U+1F600) and decimal format
	var query string
	var arg interface{}

	if strings.HasPrefix(strings.ToUpper(codepoint), "U+") {
		query = "SELECT codepoint, decimal, name, category, block, unicode_version, description, html_entity, css_content, properties FROM characters WHERE codepoint = $1"
		arg = strings.ToUpper(codepoint)
	} else {
		// Try decimal format
		decimal, err := strconv.Atoi(codepoint)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid codepoint format. Use U+XXXX or decimal number",
			})
			return
		}
		query = "SELECT codepoint, decimal, name, category, block, unicode_version, description, html_entity, css_content, properties FROM characters WHERE decimal = $1"
		arg = decimal
	}

	var char Character
	var propertiesJSON []byte

	err := api.db.QueryRow(query, arg).Scan(
		&char.Codepoint, &char.Decimal, &char.Name, &char.Category,
		&char.Block, &char.UnicodeVersion, &char.Description,
		&char.HTMLEntity, &char.CSSContent, &propertiesJSON,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Character not found",
		})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get character",
		})
		return
	}

	// Parse properties JSON
	if len(propertiesJSON) > 0 {
		json.Unmarshal(propertiesJSON, &char.Properties)
	}

	// Get related characters (same block, first 5)
	relatedQuery := `
		SELECT codepoint, decimal, name, category, block, unicode_version, 
			   description, html_entity, css_content, properties
		FROM characters 
		WHERE block = $1 AND codepoint != $2 
		ORDER BY decimal 
		LIMIT 5
	`

	rows, err := api.db.Query(relatedQuery, char.Block, char.Codepoint)
	if err == nil {
		defer rows.Close()

		var relatedCharacters []Character
		for rows.Next() {
			var related Character
			var relatedPropertiesJSON []byte

			err := rows.Scan(
				&related.Codepoint, &related.Decimal, &related.Name, &related.Category,
				&related.Block, &related.UnicodeVersion, &related.Description,
				&related.HTMLEntity, &related.CSSContent, &relatedPropertiesJSON,
			)
			if err == nil {
				if len(relatedPropertiesJSON) > 0 {
					json.Unmarshal(relatedPropertiesJSON, &related.Properties)
				}
				relatedCharacters = append(relatedCharacters, related)
			}
		}

		response := CharacterDetailResponse{
			Character:         char,
			RelatedCharacters: relatedCharacters,
			UsageExamples:     generateUsageExamples(char),
		}

		c.JSON(http.StatusOK, response)
	} else {
		// Return without related characters if query fails
		response := CharacterDetailResponse{
			Character:     char,
			UsageExamples: generateUsageExamples(char),
		}
		c.JSON(http.StatusOK, response)
	}
}

func (api *API) getCategories(c *gin.Context) {
	query := `
		SELECT c.code, c.name, c.description, COUNT(ch.codepoint) as character_count
		FROM categories c
		LEFT JOIN characters ch ON ch.category = c.code
		GROUP BY c.code, c.name, c.description
		ORDER BY c.name
	`

	rows, err := api.db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get categories",
		})
		return
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var category Category
		err := rows.Scan(&category.Code, &category.Name, &category.Description, &category.CharacterCount)
		if err != nil {
			log.Printf("Error scanning category: %v", err)
			continue
		}
		categories = append(categories, category)
	}

	c.JSON(http.StatusOK, gin.H{
		"categories": categories,
	})
}

func (api *API) getBlocks(c *gin.Context) {
	query := `
		SELECT b.id, b.name, b.start_codepoint, b.end_codepoint, b.description,
			   COUNT(ch.codepoint) as character_count
		FROM character_blocks b
		LEFT JOIN characters ch ON ch.block = b.name
		GROUP BY b.id, b.name, b.start_codepoint, b.end_codepoint, b.description
		ORDER BY b.start_codepoint
	`

	rows, err := api.db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get blocks",
		})
		return
	}
	defer rows.Close()

	var blocks []CharacterBlock
	for rows.Next() {
		var block CharacterBlock
		err := rows.Scan(&block.ID, &block.Name, &block.StartCodepoint,
			&block.EndCodepoint, &block.Description, &block.CharacterCount)
		if err != nil {
			log.Printf("Error scanning block: %v", err)
			continue
		}
		blocks = append(blocks, block)
	}

	c.JSON(http.StatusOK, gin.H{
		"blocks": blocks,
	})
}

func (api *API) getBulkRange(c *gin.Context) {
	var req BulkRangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	if len(req.Ranges) == 0 || len(req.Ranges) > 10 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Must specify 1-10 ranges",
		})
		return
	}

	var allCharacters []Character
	rangesProcessed := 0

	for _, r := range req.Ranges {
		start, end, err := parseCodepointRange(r.Start, r.End)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Invalid range format: %v", err),
			})
			return
		}

		query := `
			SELECT codepoint, decimal, name, category, block, unicode_version,
				   description, html_entity, css_content, properties
			FROM characters 
			WHERE decimal >= $1 AND decimal <= $2
			ORDER BY decimal
			LIMIT 1000
		`

		rows, err := api.db.Query(query, start, end)
		if err != nil {
			log.Printf("Error querying range %d-%d: %v", start, end, err)
			continue
		}

		for rows.Next() {
			var char Character
			var propertiesJSON []byte

			err := rows.Scan(
				&char.Codepoint, &char.Decimal, &char.Name, &char.Category,
				&char.Block, &char.UnicodeVersion, &char.Description,
				&char.HTMLEntity, &char.CSSContent, &propertiesJSON,
			)
			if err != nil {
				log.Printf("Error scanning character in range: %v", err)
				continue
			}

			if len(propertiesJSON) > 0 {
				json.Unmarshal(propertiesJSON, &char.Properties)
			}

			allCharacters = append(allCharacters, char)
		}
		rows.Close()
		rangesProcessed++
	}

	response := BulkRangeResponse{
		Characters:      allCharacters,
		TotalCharacters: len(allCharacters),
		RangesProcessed: rangesProcessed,
	}

	c.JSON(http.StatusOK, response)
}

// Helper functions

// getEnv removed to prevent hardcoded defaults

func parseCodepointRange(start, end string) (int, int, error) {
	startDecimal, err := parseCodepoint(start)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid start codepoint: %v", err)
	}

	endDecimal, err := parseCodepoint(end)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid end codepoint: %v", err)
	}

	if startDecimal > endDecimal {
		return 0, 0, fmt.Errorf("start codepoint must be <= end codepoint")
	}

	return startDecimal, endDecimal, nil
}

func parseCodepoint(codepoint string) (int, error) {
	if strings.HasPrefix(strings.ToUpper(codepoint), "U+") {
		// Parse Unicode format (U+1F600)
		hex := strings.TrimPrefix(strings.ToUpper(codepoint), "U+")
		val, err := strconv.ParseInt(hex, 16, 32)
		if err != nil {
			return 0, err
		}
		return int(val), nil
	} else {
		// Parse decimal format
		return strconv.Atoi(codepoint)
	}
}

func generateUsageExamples(char Character) []string {
	examples := []string{
		fmt.Sprintf("HTML: &#%d;", char.Decimal),
		fmt.Sprintf("Unicode: %s", char.Codepoint),
	}

	if char.HTMLEntity != nil && *char.HTMLEntity != "" {
		examples = append(examples, fmt.Sprintf("HTML Entity: %s", *char.HTMLEntity))
	}

	if char.CSSContent != nil && *char.CSSContent != "" {
		examples = append(examples, fmt.Sprintf("CSS: content: \"%s\";", *char.CSSContent))
	}

	return examples
}

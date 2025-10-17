package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RSS Feed structures
type RSS struct {
	Channel Channel `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

// FeedProcessor handles RSS feed fetching and processing
type FeedProcessor struct {
	db          *sql.DB
	ollamaURL   string
	redisClient *redis.Client
	mu          sync.Mutex
}

// NewFeedProcessor creates a new feed processor
func NewFeedProcessor(database *sql.DB, ollamaURL string, redisClient *redis.Client) *FeedProcessor {
	return &FeedProcessor{
		db:          database,
		ollamaURL:   ollamaURL,
		redisClient: redisClient,
	}
}

// FetchRSSFeed fetches and parses an RSS feed
func (fp *FeedProcessor) FetchRSSFeed(feedURL string) (*RSS, error) {
	resp, err := http.Get(feedURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rss RSS
	err = xml.Unmarshal(body, &rss)
	if err != nil {
		return nil, err
	}

	return &rss, nil
}

// ProcessFeed processes a single RSS feed
func (fp *FeedProcessor) ProcessFeed(feed Feed) error {
	log.Printf("Processing feed: %s", feed.Name)

	// Fetch RSS content
	rss, err := fp.FetchRSSFeed(feed.URL)
	if err != nil {
		log.Printf("Error fetching feed %s: %v", feed.Name, err)
		return err
	}

	// Process each article
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // Process 5 articles concurrently

	for _, item := range rss.Channel.Items {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(item Item) {
			defer wg.Done()
			defer func() { <-semaphore }()

			article := Article{
				Title:       item.Title,
				URL:         item.Link,
				Source:      feed.Name,
				Summary:     item.Description,
				FetchedAt:   time.Now(),
			}

			// Parse published date
			if pubDate, err := time.Parse(time.RFC1123Z, item.PubDate); err == nil {
				article.PublishedAt = pubDate
			} else {
				article.PublishedAt = time.Now()
			}

			// Check if article already exists
			if !fp.articleExists(article.URL) {
				// Enrich article with AI analysis
				fp.enrichArticle(&article)

				// Analyze bias
				fp.analyzeArticleBias(&article)

				// Store article
				fp.storeArticle(article)

				// Broadcast update
				broadcast <- article
			}
		}(item)
	}

	wg.Wait()
	return nil
}

// ProcessAllFeeds processes all active feeds
func (fp *FeedProcessor) ProcessAllFeeds() error {
	feeds, err := fp.getActiveFeeds()
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	for _, feed := range feeds {
		wg.Add(1)
		go func(f Feed) {
			defer wg.Done()
			fp.ProcessFeed(f)
		}(feed)
	}

	wg.Wait()
	return nil
}

// EnrichArticle enriches article with AI-generated summary and metadata
func (fp *FeedProcessor) enrichArticle(article *Article) {
	prompt := fmt.Sprintf(`Analyze this news article and provide:
	1. A concise summary (2-3 sentences)
	2. Key topics and entities mentioned
	3. The main perspective or stance
	
	Title: %s
	Content: %s
	
	Format as JSON with keys: summary, topics, perspective`, 
		article.Title, article.Summary)

	payload := map[string]interface{}{
		"model":  "llama3.2",
		"prompt": prompt,
		"format": "json",
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(fp.ollamaURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to enrich article: %v", err)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if response, ok := result["response"].(string); ok {
		var enrichment map[string]interface{}
		if err := json.Unmarshal([]byte(response), &enrichment); err == nil {
			if summary, ok := enrichment["summary"].(string); ok {
				article.Summary = summary
			}
		}
	}
}

// AnalyzeArticleBias analyzes the bias in an article
func (fp *FeedProcessor) analyzeArticleBias(article *Article) {
	prompt := fmt.Sprintf(`Analyze the bias in this news article:
	Title: %s
	Source: %s
	Content: %s
	
	Provide:
	1. Bias score (-100 to 100, where -100 is far left, 0 is center, 100 is far right)
	2. Bias analysis explaining the score
	3. Loaded language or biased framing identified
	4. Missing perspectives or context
	
	Format as JSON with keys: bias_score, analysis, loaded_language, missing_perspectives`,
		article.Title, article.Source, article.Summary)

	payload := map[string]interface{}{
		"model":  "llama3.2",
		"prompt": prompt,
		"format": "json",
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(fp.ollamaURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to analyze bias: %v", err)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if response, ok := result["response"].(string); ok {
		var biasAnalysis map[string]interface{}
		if err := json.Unmarshal([]byte(response), &biasAnalysis); err == nil {
			if score, ok := biasAnalysis["bias_score"].(float64); ok {
				article.BiasScore = score
			}
			if analysis, ok := biasAnalysis["analysis"].(string); ok {
				article.BiasAnalysis = analysis
			}
		}
	}
}

// FactCheck performs fact-checking on article claims
func (fp *FeedProcessor) factCheckArticle(article *Article) (*FactCheckResult, error) {
	prompt := fmt.Sprintf(`Fact-check the main claims in this article:
	Title: %s
	Content: %s
	
	For each claim:
	1. Identify the claim
	2. Assess verifiability (can it be fact-checked?)
	3. Provide a verdict (true, false, misleading, unverifiable)
	4. Explain the reasoning
	
	Format as JSON with array of claims, each having: claim, verifiable, verdict, reasoning`,
		article.Title, article.Summary)

	payload := map[string]interface{}{
		"model":  "llama3.2",
		"prompt": prompt,
		"format": "json",
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(fp.ollamaURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	factCheckResult := &FactCheckResult{
		ArticleID:   article.ID,
		CheckedAt:   time.Now(),
		Claims:      []Claim{},
	}

	if response, ok := result["response"].(string); ok {
		var checkResult map[string]interface{}
		if err := json.Unmarshal([]byte(response), &checkResult); err == nil {
			if claims, ok := checkResult["claims"].([]interface{}); ok {
				for _, c := range claims {
					if claimMap, ok := c.(map[string]interface{}); ok {
						claim := Claim{
							Text:       getString(claimMap, "claim"),
							Verifiable: getBool(claimMap, "verifiable"),
							Verdict:    getString(claimMap, "verdict"),
							Reasoning:  getString(claimMap, "reasoning"),
						}
						factCheckResult.Claims = append(factCheckResult.Claims, claim)
					}
				}
			}
		}
	}

	return factCheckResult, nil
}

// AggregatePerspectives finds and aggregates different perspectives on a topic
func (fp *FeedProcessor) aggregatePerspectives(topic string) (*PerspectiveAggregation, error) {
	// Fetch articles related to the topic
	articles, err := fp.fetchArticlesByTopic(topic)
	if err != nil {
		return nil, err
	}

	// Group articles by source bias rating
	perspectiveGroups := make(map[string][]Article)
	for _, article := range articles {
		biasCategory := fp.categorizeBias(article.BiasScore)
		perspectiveGroups[biasCategory] = append(perspectiveGroups[biasCategory], article)
	}

	// Analyze common themes and differences
	prompt := fmt.Sprintf(`Analyze these different perspectives on "%s":
	
	Left-leaning sources say: %s
	Center sources say: %s
	Right-leaning sources say: %s
	
	Identify:
	1. Common ground across perspectives
	2. Key disagreements
	3. Unique points from each perspective
	4. Overall narrative differences
	
	Format as JSON with keys: common_ground, disagreements, unique_points, narrative_differences`,
		topic,
		fp.summarizePerspective(perspectiveGroups["left"]),
		fp.summarizePerspective(perspectiveGroups["center"]),
		fp.summarizePerspective(perspectiveGroups["right"]))

	payload := map[string]interface{}{
		"model":  "llama3.2",
		"prompt": prompt,
		"format": "json",
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(fp.ollamaURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	aggregation := &PerspectiveAggregation{
		Topic:         topic,
		ArticleCount:  len(articles),
		Perspectives:  perspectiveGroups,
		AnalyzedAt:    time.Now(),
	}

	if response, ok := result["response"].(string); ok {
		var analysis map[string]interface{}
		if err := json.Unmarshal([]byte(response), &analysis); err == nil {
			aggregation.CommonGround = getStringSlice(analysis, "common_ground")
			aggregation.Disagreements = getStringSlice(analysis, "disagreements")
			aggregation.UniquePoints = analysis["unique_points"]
			aggregation.NarrativeDifferences = getString(analysis, "narrative_differences")
		}
	}

	return aggregation, nil
}

// Helper functions
func (fp *FeedProcessor) articleExists(url string) bool {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM articles WHERE url = $1)`
	fp.db.QueryRow(query, url).Scan(&exists)
	return exists
}

func (fp *FeedProcessor) storeArticle(article Article) error {
	query := `
		INSERT INTO articles (title, url, source, published_at, summary, 
		                      bias_score, bias_analysis, fetched_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (url) DO UPDATE SET
			summary = $5,
			bias_score = $6,
			bias_analysis = $7,
			fetched_at = $8
	`
	_, err := fp.db.Exec(query, article.Title, article.URL, article.Source,
		article.PublishedAt, article.Summary, article.BiasScore,
		article.BiasAnalysis, article.FetchedAt)
	return err
}

func (fp *FeedProcessor) getActiveFeeds() ([]Feed, error) {
	query := `SELECT id, name, url, category, bias_rating FROM feeds WHERE active = true`
	rows, err := fp.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []Feed
	for rows.Next() {
		var feed Feed
		err := rows.Scan(&feed.ID, &feed.Name, &feed.URL, &feed.Category, &feed.BiasRating)
		if err != nil {
			continue
		}
		feed.Active = true
		feeds = append(feeds, feed)
	}

	return feeds, nil
}

func (fp *FeedProcessor) fetchArticlesByTopic(topic string) ([]Article, error) {
	query := `
		SELECT id, title, url, source, published_at, summary, bias_score, bias_analysis
		FROM articles
		WHERE title ILIKE $1 OR summary ILIKE $1
		ORDER BY published_at DESC
		LIMIT 50
	`
	searchTerm := "%" + topic + "%"
	rows, err := fp.db.Query(query, searchTerm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []Article
	for rows.Next() {
		var article Article
		err := rows.Scan(&article.ID, &article.Title, &article.URL, &article.Source,
			&article.PublishedAt, &article.Summary, &article.BiasScore, &article.BiasAnalysis)
		if err != nil {
			continue
		}
		articles = append(articles, article)
	}

	return articles, nil
}

func (fp *FeedProcessor) categorizeBias(score float64) string {
	if score < -33 {
		return "left"
	} else if score > 33 {
		return "right"
	}
	return "center"
}

func (fp *FeedProcessor) summarizePerspective(articles []Article) string {
	if len(articles) == 0 {
		return "No articles from this perspective"
	}

	summaries := []string{}
	for _, article := range articles {
		if len(summaries) < 3 { // Limit to 3 articles
			summaries = append(summaries, article.Summary)
		}
	}

	return strings.Join(summaries, " | ")
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func getStringSlice(m map[string]interface{}, key string) []string {
	result := []string{}
	if v, ok := m[key].([]interface{}); ok {
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
	}
	return result
}
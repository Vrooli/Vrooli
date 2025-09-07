package platforms

import (
	"fmt"
	"strings"
	"time"
)

// TwitterIntegration handles X (Twitter) bookmark extraction and monitoring
type TwitterIntegration struct {
	config TwitterConfig
}

// TwitterConfig holds Twitter-specific configuration
type TwitterConfig struct {
	SessionCookie    string `json:"session_cookie"`    // For web scraping approach
	BrowserlessURL   string `json:"browserless_url"`   // Fallback browserless instance
	UserAgent       string `json:"user_agent"`
	RateLimit       int    `json:"rate_limit"`        // Requests per 15 minutes
	FallbackEnabled bool   `json:"fallback_enabled"`
}

// TwitterTweet represents a Twitter tweet/bookmark
type TwitterTweet struct {
	ID            string    `json:"id"`
	Text          string    `json:"text"`
	AuthorName    string    `json:"author_name"`
	Username      string    `json:"username"`
	CreatedAt     time.Time `json:"created_at"`
	Retweets      int       `json:"retweets"`
	Likes         int       `json:"likes"`
	Replies       int       `json:"replies"`
	IsRetweet     bool      `json:"is_retweet"`
	HasMedia      bool      `json:"has_media"`
	MediaURLs     []string  `json:"media_urls"`
	Hashtags      []string  `json:"hashtags"`
	Mentions      []string  `json:"mentions"`
	URLs          []string  `json:"urls"`
	ConversationID string   `json:"conversation_id"`
	InReplyToID   string    `json:"in_reply_to_id"`
}

// NewTwitterIntegration creates a new Twitter integration instance
func NewTwitterIntegration(config TwitterConfig) *TwitterIntegration {
	return &TwitterIntegration{
		config: config,
	}
}

// GetBookmarks retrieves bookmarked tweets from Twitter
func (t *TwitterIntegration) GetBookmarks(limit int) ([]StandardBookmark, error) {
	// TODO: Implement Twitter bookmark scraping
	// This requires either:
	// 1. Web scraping with session cookies + browserless fallback
	// 2. Twitter API access (difficult to obtain for individual use)
	
	bookmarks := []StandardBookmark{}
	
	// Placeholder implementation - actual version would:
	// 1. Use Huginn agents to scrape bookmarks page
	// 2. Fall back to browserless for JavaScript-heavy content
	// 3. Parse tweet data from HTML/JSON
	// 4. Convert to StandardBookmark format
	
	return bookmarks, nil
}

// ProcessBookmark converts a Twitter tweet to our standard bookmark format
func (t *TwitterIntegration) ProcessBookmark(rawData interface{}) StandardBookmark {
	tweet, ok := rawData.(TwitterTweet)
	if !ok {
		// Handle conversion error
		return StandardBookmark{}
	}
	
	// Create full Twitter URL
	fullURL := fmt.Sprintf("https://x.com/%s/status/%s", tweet.Username, tweet.ID)
	
	// Determine content type based on media presence
	contentType := ContentTypeText
	if tweet.HasMedia {
		if len(tweet.MediaURLs) > 0 {
			// Simple heuristic based on URL patterns
			mediaURL := tweet.MediaURLs[0]
			if strings.Contains(mediaURL, "video") {
				contentType = ContentTypeVideo
			} else {
				contentType = ContentTypeImage
			}
		}
	} else if len(tweet.URLs) > 0 {
		contentType = ContentTypeLink
	}
	
	// Prepare metadata
	metadata := map[string]interface{}{
		"tweet_id":        tweet.ID,
		"username":        tweet.Username,
		"retweets":        tweet.Retweets,
		"likes":          tweet.Likes,
		"replies":         tweet.Replies,
		"is_retweet":      tweet.IsRetweet,
		"has_media":       tweet.HasMedia,
		"media_urls":      tweet.MediaURLs,
		"hashtags":        tweet.Hashtags,
		"mentions":        tweet.Mentions,
		"urls":           tweet.URLs,
		"conversation_id": tweet.ConversationID,
		"extracted_by":    "twitter-integration",
		"api_version":     "v1",
	}
	
	// Generate hash signature for deduplication
	hashInput := fmt.Sprintf("twitter:%s:%s", tweet.ID, tweet.CreatedAt.Format(time.RFC3339))
	
	return StandardBookmark{
		ID:              tweet.ID,
		Platform:        "twitter",
		OriginalURL:     fullURL,
		Title:          fmt.Sprintf("Tweet by @%s", tweet.Username),
		Content:        tweet.Text,
		Author:         tweet.AuthorName,
		AuthorUsername: tweet.Username,
		CreatedAt:      tweet.CreatedAt,
		BookmarkedAt:   time.Now(), // Would be actual bookmark time in real implementation
		ContentType:    contentType,
		Metadata:       metadata,
		HashSignature:  generateHash(hashInput),
	}
}

// ValidateConfig checks if the Twitter configuration is valid
func (t *TwitterIntegration) ValidateConfig() error {
	if t.config.SessionCookie == "" {
		return fmt.Errorf("twitter session_cookie is required for web scraping")
	}
	if t.config.BrowserlessURL == "" {
		return fmt.Errorf("browserless_url is required for fallback scraping")
	}
	if t.config.UserAgent == "" {
		t.config.UserAgent = "Mozilla/5.0 (compatible; BookmarkIntelligenceBot/1.0)"
	}
	if t.config.RateLimit <= 0 {
		t.config.RateLimit = 300 // Default to 300 requests per 15 minutes
	}
	return nil
}

// GetPlatformInfo returns metadata about this platform integration
func (t *TwitterIntegration) GetPlatformInfo() PlatformInfo {
	return PlatformInfo{
		Name:                "twitter",
		DisplayName:         "X (Twitter)",
		Icon:                "fab fa-twitter",
		Color:              "#1DA1F2",
		SupportedFeatures:  []string{"bookmarks", "user_timeline", "mentions"},
		AuthMethod:         "session",
		RateLimits:         map[string]int{"per_15min": t.config.RateLimit},
		RequiredCredentials: []string{"session_cookie", "browserless_url"},
		OptionalCredentials: []string{"user_agent"},
		DefaultSyncInterval: 60 * time.Minute,
		ContentTypes:       []string{"text", "link", "image", "video"},
		Categories:         []string{"news", "tech", "entertainment", "politics", "sports"},
		Notes:              "Uses web scraping with browserless fallback due to API limitations",
	}
}

// TestConnection tests the Twitter connection
func (t *TwitterIntegration) TestConnection() error {
	if err := t.ValidateConfig(); err != nil {
		return fmt.Errorf("configuration invalid: %w", err)
	}
	
	// TODO: Implement actual connection test
	// This would make a request to Twitter bookmarks page to verify session
	
	return nil
}

// EstimateCategories provides category suggestions based on Twitter content
func (t *TwitterIntegration) EstimateCategories(bookmark StandardBookmark) []CategorySuggestion {
	suggestions := []CategorySuggestion{}
	
	// Extract tweet data from metadata
	hashtags, _ := bookmark.Metadata["hashtags"].([]string)
	mentions, _ := bookmark.Metadata["mentions"].([]string)
	hasMedia, _ := bookmark.Metadata["has_media"].(bool)
	
	// Hashtag-based categorization
	if len(hashtags) > 0 {
		hashtagCategories := map[string]string{
			"coding":        "Programming",
			"programming":   "Programming",
			"javascript":    "Programming",
			"python":        "Programming",
			"webdev":        "Programming",
			"recipe":        "Recipes",
			"cooking":       "Recipes",
			"food":          "Recipes",
			"fitness":       "Fitness",
			"workout":       "Fitness",
			"gym":           "Fitness",
			"travel":        "Travel",
			"vacation":      "Travel",
			"wanderlust":    "Travel",
			"news":          "News",
			"breaking":      "News",
			"politics":      "News",
			"tech":          "Tech",
			"ai":            "Tech",
			"technology":    "Tech",
		}
		
		matchedHashtags := []string{}
		for _, hashtag := range hashtags {
			normalizedTag := strings.ToLower(strings.TrimPrefix(hashtag, "#"))
			if category, exists := hashtagCategories[normalizedTag]; exists {
				matchedHashtags = append(matchedHashtags, hashtag)
				suggestions = append(suggestions, CategorySuggestion{
					Category:    category,
					Confidence:  0.8,
					Reason:      fmt.Sprintf("Based on hashtag: %s", hashtag),
					Method:      "hashtag_mapping",
					Keywords:    []string{hashtag},
				})
			}
		}
	}
	
	// Content-based keyword matching
	content := strings.ToLower(bookmark.Content)
	title := strings.ToLower(bookmark.Title)
	combined := content + " " + title
	
	keywordCategories := map[string][]string{
		"Programming": {"code", "programming", "developer", "api", "javascript", "python", "react", "coding"},
		"News":        {"breaking", "news", "just in", "update", "reported", "according to"},
		"Tech":        {"ai", "technology", "tech", "startup", "innovation", "digital"},
		"Entertainment": {"movie", "tv", "show", "music", "game", "entertainment", "celebrity"},
		"Education":   {"learn", "education", "tutorial", "course", "study", "tip", "how to"},
		"Business":    {"business", "economy", "market", "stock", "investment", "company"},
	}
	
	for category, keywords := range keywordCategories {
		matchCount := 0
		matchedKeywords := []string{}
		
		for _, keyword := range keywords {
			if strings.Contains(combined, keyword) {
				matchCount++
				matchedKeywords = append(matchedKeywords, keyword)
			}
		}
		
		if matchCount > 0 {
			confidence := float64(matchCount) / float64(len(keywords))
			if confidence > 0.2 { // Lower threshold for Twitter due to brevity
				suggestions = append(suggestions, CategorySuggestion{
					Category:    category,
					Confidence:  confidence * 0.6, // Lower confidence for keyword matching
					Reason:      fmt.Sprintf("Matched %d keywords", matchCount),
					Method:      "keyword_matching",
					Keywords:    matchedKeywords,
				})
			}
		}
	}
	
	// Media-based categorization
	if hasMedia {
		suggestions = append(suggestions, CategorySuggestion{
			Category:    "Entertainment",
			Confidence:  0.4,
			Reason:      "Contains media content",
			Method:      "media_detection",
		})
	}
	
	// Author-based categorization (if known tech/news accounts)
	username := bookmark.AuthorUsername
	if username != "" {
		authorCategories := map[string]string{
			"techcrunch":    "Tech",
			"verge":         "Tech", 
			"engadget":      "Tech",
			"cnn":           "News",
			"bbcnews":       "News",
			"nytimes":       "News",
			"foodnetwork":   "Recipes",
			"buzzfeedtasty": "Recipes",
		}
		
		normalizedUsername := strings.ToLower(username)
		for account, category := range authorCategories {
			if strings.Contains(normalizedUsername, account) {
				suggestions = append(suggestions, CategorySuggestion{
					Category:    category,
					Confidence:  0.9,
					Reason:      fmt.Sprintf("Known %s account: @%s", category, username),
					Method:      "author_recognition",
				})
				break
			}
		}
	}
	
	return suggestions
}

// GetSuggestedActions returns suggested actions based on Twitter content
func (t *TwitterIntegration) GetSuggestedActions(bookmark StandardBookmark) []ActionSuggestion {
	actions := []ActionSuggestion{}
	
	// Extract metadata for action suggestions
	hashtags, _ := bookmark.Metadata["hashtags"].([]string)
	urls, _ := bookmark.Metadata["urls"].([]string)
	hasMedia, _ := bookmark.Metadata["has_media"].(bool)
	
	// Analyze content for action suggestions
	content := strings.ToLower(bookmark.Content)
	
	// Recipe-related actions
	if containsAny(content, []string{"recipe", "cooking", "food", "ingredients"}) ||
		containsAnyHashtag(hashtags, []string{"recipe", "cooking", "food"}) {
		actions = append(actions, ActionSuggestion{
			ActionType:      ActionAddToRecipeBook,
			TargetScenario:  "recipe-book",
			Confidence:      0.75,
			Priority:        8,
			ActionData: map[string]interface{}{
				"title":       bookmark.Title,
				"source_url":  bookmark.OriginalURL,
				"platform":    "twitter",
				"hashtags":    hashtags,
				"author":      "@" + bookmark.AuthorUsername,
			},
			Description: "Save this recipe or cooking tip",
		})
	}
	
	// Fitness-related actions
	if containsAny(content, []string{"workout", "fitness", "exercise", "gym"}) ||
		containsAnyHashtag(hashtags, []string{"fitness", "workout", "gym"}) {
		actions = append(actions, ActionSuggestion{
			ActionType:      ActionScheduleWorkout,
			TargetScenario:  "workout-plan-generator",
			Confidence:      0.7,
			Priority:        7,
			ActionData: map[string]interface{}{
				"workout_name": bookmark.Title,
				"source_url":   bookmark.OriginalURL,
				"platform":     "twitter",
				"has_media":    hasMedia,
			},
			Description: "Add this workout or fitness tip to your plan",
		})
	}
	
	// Programming/Tech-related actions
	if containsAny(content, []string{"code", "programming", "api", "javascript", "python"}) ||
		containsAnyHashtag(hashtags, []string{"coding", "programming", "webdev", "javascript", "python"}) {
		actions = append(actions, ActionSuggestion{
			ActionType:      ActionAddToCodeLibrary,
			TargetScenario:  "code-library",
			Confidence:      0.8,
			Priority:        9,
			ActionData: map[string]interface{}{
				"title":       bookmark.Title,
				"source_url":  bookmark.OriginalURL,
				"content":     bookmark.Content,
				"hashtags":    hashtags,
				"language":    detectProgrammingLanguage(content, hashtags),
			},
			Description: "Save this programming resource or tip",
		})
	}
	
	// News-related actions
	if containsAny(content, []string{"breaking", "news", "just in", "update"}) ||
		containsAnyHashtag(hashtags, []string{"news", "breaking", "update"}) {
		actions = append(actions, ActionSuggestion{
			ActionType:      ActionAddToResearch,
			TargetScenario:  "research-assistant",
			Confidence:      0.6,
			Priority:        5,
			ActionData: map[string]interface{}{
				"title":       bookmark.Title,
				"content":     bookmark.Content,
				"source_url":  bookmark.OriginalURL,
				"category":    "news",
				"platform":    "twitter",
				"timestamp":   bookmark.CreatedAt,
			},
			Description: "Save this news item for research",
		})
	}
	
	// Travel-related actions
	if containsAny(content, []string{"travel", "trip", "vacation", "destination"}) ||
		containsAnyHashtag(hashtags, []string{"travel", "vacation", "wanderlust"}) {
		actions = append(actions, ActionSuggestion{
			ActionType:      ActionAddToTravelList,
			TargetScenario:  "travel-planner",
			Confidence:      0.65,
			Priority:        6,
			ActionData: map[string]interface{}{
				"destination": extractDestination(bookmark.Content),
				"source_url":  bookmark.OriginalURL,
				"notes":       bookmark.Content,
				"has_media":   hasMedia,
			},
			Description: "Add this travel tip or destination",
		})
	}
	
	// Link-sharing action for tweets with external links
	if len(urls) > 0 {
		actions = append(actions, ActionSuggestion{
			ActionType:      ActionShareContent,
			TargetScenario:  "content-aggregator",
			Confidence:      0.5,
			Priority:        3,
			ActionData: map[string]interface{}{
				"original_tweet": bookmark.OriginalURL,
				"shared_urls":    urls,
				"context":        bookmark.Content,
				"author":         "@" + bookmark.AuthorUsername,
			},
			Description: "Share this content with referenced links",
		})
	}
	
	// Always offer general research archival
	actions = append(actions, ActionSuggestion{
		ActionType:      ActionAddToResearch,
		TargetScenario:  "research-assistant",
		Confidence:      0.4,
		Priority:        2,
		ActionData: map[string]interface{}{
			"title":       bookmark.Title,
			"content":     bookmark.Content,
			"source_url":  bookmark.OriginalURL,
			"platform":    "twitter",
			"author":      "@" + bookmark.AuthorUsername,
			"hashtags":    hashtags,
			"created_at":  bookmark.CreatedAt,
		},
		Description: "Archive this tweet for future reference",
	})
	
	return actions
}

// Helper functions

func containsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func containsAnyHashtag(hashtags []string, keywords []string) bool {
	for _, hashtag := range hashtags {
		normalized := strings.ToLower(strings.TrimPrefix(hashtag, "#"))
		for _, keyword := range keywords {
			if strings.Contains(normalized, keyword) {
				return true
			}
		}
	}
	return false
}

func detectProgrammingLanguage(content string, hashtags []string) string {
	languages := map[string][]string{
		"javascript": {"javascript", "js", "react", "vue", "angular", "node"},
		"python":     {"python", "django", "flask", "pandas"},
		"go":         {"golang", "go"},
		"rust":       {"rust", "rustlang"},
		"java":       {"java", "spring", "kotlin"},
		"php":        {"php", "laravel"},
		"ruby":       {"ruby", "rails"},
	}
	
	combined := strings.ToLower(content + " " + strings.Join(hashtags, " "))
	
	for lang, keywords := range languages {
		for _, keyword := range keywords {
			if strings.Contains(combined, keyword) {
				return lang
			}
		}
	}
	
	return "unknown"
}

func extractDestination(content string) string {
	// Simple destination extraction - could be enhanced with NLP
	words := strings.Fields(content)
	for i, word := range words {
		normalized := strings.ToLower(word)
		if normalized == "in" || normalized == "to" || normalized == "visit" {
			if i+1 < len(words) {
				return words[i+1]
			}
		}
	}
	return ""
}

func generateHash(input string) string {
	// Simple hash implementation - would use crypto/md5 or similar in real implementation
	return fmt.Sprintf("%x", len(input)+int(input[0]))
}
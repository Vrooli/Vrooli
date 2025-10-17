package platforms

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// RedditIntegration handles Reddit bookmark extraction and monitoring
type RedditIntegration struct {
	config RedditConfig
	client *http.Client
}

// RedditConfig holds Reddit-specific configuration
type RedditConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	UserAgent   string `json:"user_agent"`
	RateLimit   int    `json:"rate_limit"` // Requests per hour
}

// RedditPost represents a Reddit post/bookmark
type RedditPost struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"selftext"`
	URL         string    `json:"url"`
	Permalink   string    `json:"permalink"`
	Author      string    `json:"author"`
	Subreddit   string    `json:"subreddit"`
	Score       int       `json:"score"`
	Comments    int       `json:"num_comments"`
	CreatedUTC  float64   `json:"created_utc"`
	IsText      bool      `json:"is_self"`
	Saved       bool      `json:"saved"`
	Thumbnail   string    `json:"thumbnail"`
	Domain      string    `json:"domain"`
}

// RedditBookmark represents a processed Reddit bookmark for our system
type RedditBookmark struct {
	Platform        string                 `json:"platform"`
	OriginalURL     string                 `json:"original_url"`
	Title          string                 `json:"title"`
	Content        string                 `json:"content"`
	Author         string                 `json:"author"`
	CreatedAt      time.Time              `json:"created_at"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// NewRedditIntegration creates a new Reddit integration instance
func NewRedditIntegration(config RedditConfig) *RedditIntegration {
	return &RedditIntegration{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetSavedPosts retrieves saved posts from Reddit
func (r *RedditIntegration) GetSavedPosts(limit int) ([]RedditBookmark, error) {
	// TODO: Implement OAuth2 authentication with Reddit API
	// For now, return mock data structure
	
	bookmarks := []RedditBookmark{}
	
	// This is a placeholder - actual implementation would:
	// 1. Authenticate with Reddit OAuth2
	// 2. Fetch saved posts from /user/{username}/saved
	// 3. Parse the JSON response
	// 4. Convert to our bookmark format
	
	return bookmarks, nil
}

// ProcessBookmark converts a Reddit post to our standard bookmark format
func (r *RedditIntegration) ProcessBookmark(post RedditPost) RedditBookmark {
	// Determine content - use selftext for text posts, URL for link posts
	content := post.Content
	if !post.IsText && content == "" {
		content = post.URL
	}
	
	// Create full Reddit URL
	fullURL := fmt.Sprintf("https://reddit.com%s", post.Permalink)
	
	// Prepare metadata
	metadata := map[string]interface{}{
		"subreddit":     post.Subreddit,
		"post_id":       post.ID,
		"score":         post.Score,
		"num_comments":  post.Comments,
		"post_type":     map[bool]string{true: "text", false: "link"}[post.IsText],
		"domain":        post.Domain,
		"thumbnail":     post.Thumbnail,
		"extracted_by":  "reddit-integration",
		"api_version":   "v1",
	}
	
	return RedditBookmark{
		Platform:    "reddit",
		OriginalURL: fullURL,
		Title:       post.Title,
		Content:     content,
		Author:      fmt.Sprintf("/u/%s", post.Author),
		CreatedAt:   time.Unix(int64(post.CreatedUTC), 0),
		Metadata:    metadata,
	}
}

// ValidateConfig checks if the Reddit configuration is valid
func (r *RedditIntegration) ValidateConfig() error {
	if r.config.ClientID == "" {
		return fmt.Errorf("reddit client_id is required")
	}
	if r.config.ClientSecret == "" {
		return fmt.Errorf("reddit client_secret is required")
	}
	if r.config.Username == "" {
		return fmt.Errorf("reddit username is required")
	}
	if r.config.Password == "" {
		return fmt.Errorf("reddit password is required")
	}
	if r.config.UserAgent == "" {
		r.config.UserAgent = "BookmarkIntelligenceHub/1.0"
	}
	if r.config.RateLimit <= 0 {
		r.config.RateLimit = 1000 // Default to 1000 requests per hour
	}
	return nil
}

// GetPlatformInfo returns metadata about this platform integration
func (r *RedditIntegration) GetPlatformInfo() PlatformInfo {
	return PlatformInfo{
		Name:                "reddit",
		DisplayName:         "Reddit",
		Icon:                "fab fa-reddit",
		Color:              "#FF4500",
		SupportedFeatures:  []string{"saved_posts", "user_posts", "subreddit_monitoring"},
		AuthMethod:         "oauth2",
		RateLimits:         map[string]int{"hourly": r.config.RateLimit},
		RequiredCredentials: []string{"client_id", "client_secret", "username", "password"},
		OptionalCredentials: []string{"user_agent"},
		DefaultSyncInterval: 30 * time.Minute,
		ContentTypes:       []string{"text_posts", "link_posts", "image_posts"},
		Categories:         []string{"programming", "recipes", "news", "education", "entertainment"},
	}
}

// TestConnection tests the Reddit API connection
func (r *RedditIntegration) TestConnection() error {
	// TODO: Implement actual connection test
	// This would make a simple API call to verify credentials
	if err := r.ValidateConfig(); err != nil {
		return fmt.Errorf("configuration invalid: %w", err)
	}
	
	// Placeholder for actual connection test
	return nil
}

// EstimateCategories provides category suggestions based on Reddit content
func (r *RedditIntegration) EstimateCategories(post RedditPost) []CategorySuggestion {
	suggestions := []CategorySuggestion{}
	
	// Subreddit-based categorization
	subreddit := strings.ToLower(post.Subreddit)
	
	categoryMap := map[string]string{
		"programming":     "Programming",
		"webdev":         "Programming", 
		"javascript":     "Programming",
		"python":         "Programming",
		"recipes":        "Recipes",
		"cooking":        "Recipes",
		"baking":         "Recipes",
		"fitness":        "Fitness",
		"bodyweightfitness": "Fitness",
		"travel":         "Travel",
		"solotravel":     "Travel",
		"todayilearned":  "Education",
		"explainlikeimfive": "Education",
		"news":           "News",
		"worldnews":      "News",
		"movies":         "Entertainment",
		"gaming":         "Entertainment",
		"funny":          "Entertainment",
	}
	
	if category, exists := categoryMap[subreddit]; exists {
		suggestions = append(suggestions, CategorySuggestion{
			Category:    category,
			Confidence:  0.85,
			Reason:      fmt.Sprintf("Based on subreddit: r/%s", post.Subreddit),
			Method:      "subreddit_mapping",
		})
	}
	
	// Title-based keyword matching
	title := strings.ToLower(post.Title)
	content := strings.ToLower(post.Content)
	combined := title + " " + content
	
	keywordCategories := map[string][]string{
		"Programming": {"code", "programming", "developer", "api", "javascript", "python", "react"},
		"Recipes":     {"recipe", "cooking", "food", "baking", "ingredients", "delicious"},
		"Fitness":     {"workout", "fitness", "exercise", "gym", "training", "muscle"},
		"Travel":      {"travel", "trip", "vacation", "destination", "flight", "hotel"},
		"Education":   {"learn", "education", "tutorial", "course", "study", "knowledge"},
		"News":        {"news", "breaking", "politics", "world", "economy", "business"},
	}
	
	for category, keywords := range keywordCategories {
		score := 0
		for _, keyword := range keywords {
			if strings.Contains(combined, keyword) {
				score++
			}
		}
		
		if score > 0 {
			confidence := float64(score) / float64(len(keywords))
			if confidence > 0.3 { // Only suggest if reasonable confidence
				suggestions = append(suggestions, CategorySuggestion{
					Category:    category,
					Confidence:  confidence * 0.7, // Slightly lower than subreddit-based
					Reason:      fmt.Sprintf("Matched %d keywords", score),
					Method:      "keyword_matching",
				})
			}
		}
	}
	
	return suggestions
}

// GetSuggestedActions returns suggested actions based on Reddit content
func (r *RedditIntegration) GetSuggestedActions(bookmark RedditBookmark) []ActionSuggestion {
	actions := []ActionSuggestion{}
	
	// Extract category from metadata for action suggestions
	category := ""
	if subreddit, ok := bookmark.Metadata["subreddit"].(string); ok {
		// Simple category mapping based on subreddit
		categoryMap := map[string]string{
			"recipes": "Recipes",
			"cooking": "Recipes", 
			"fitness": "Fitness",
			"programming": "Programming",
			"travel": "Travel",
		}
		category = categoryMap[strings.ToLower(subreddit)]
	}
	
	// Suggest actions based on category
	switch category {
	case "Recipes":
		actions = append(actions, ActionSuggestion{
			ActionType:      "add_to_recipe_book",
			TargetScenario:  "recipe-book",
			Confidence:      0.8,
			ActionData: map[string]interface{}{
				"title":       bookmark.Title,
				"source_url":  bookmark.OriginalURL,
				"platform":    "reddit",
				"subreddit":   bookmark.Metadata["subreddit"],
			},
			Description: "Add this recipe to your recipe book",
		})
	case "Fitness":
		actions = append(actions, ActionSuggestion{
			ActionType:      "schedule_workout",
			TargetScenario:  "workout-plan-generator",
			Confidence:      0.75,
			ActionData: map[string]interface{}{
				"workout_name": bookmark.Title,
				"source_url":   bookmark.OriginalURL,
				"platform":     "reddit",
			},
			Description: "Add this workout to your fitness plan",
		})
	case "Programming":
		actions = append(actions, ActionSuggestion{
			ActionType:      "add_to_code_library",
			TargetScenario:  "code-library",
			Confidence:      0.85,
			ActionData: map[string]interface{}{
				"title":       bookmark.Title,
				"source_url":  bookmark.OriginalURL,
				"language":    "multiple", // Could be enhanced with language detection
				"topic":       strings.ToLower(bookmark.Metadata["subreddit"].(string)),
			},
			Description: "Save this programming resource to your code library",
		})
	case "Travel":
		actions = append(actions, ActionSuggestion{
			ActionType:      "add_to_travel_list",
			TargetScenario:  "travel-planner",
			Confidence:      0.7,
			ActionData: map[string]interface{}{
				"destination": bookmark.Title,
				"source_url":  bookmark.OriginalURL,
				"notes":       bookmark.Content,
			},
			Description: "Add this destination to your travel wishlist",
		})
	}
	
	// Always suggest adding to research assistant for knowledge preservation
	actions = append(actions, ActionSuggestion{
		ActionType:      "add_to_research",
		TargetScenario:  "research-assistant",
		Confidence:      0.6,
		ActionData: map[string]interface{}{
			"title":       bookmark.Title,
			"content":     bookmark.Content,
			"source_url":  bookmark.OriginalURL,
			"platform":    "reddit",
			"category":    category,
		},
		Description: "Save this content for future research",
	})
	
	return actions
}
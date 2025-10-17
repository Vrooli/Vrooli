package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// PlatformAdapter interface for all social media platforms
type PlatformAdapter interface {
	GetName() string
	OptimizeContent(content string, mediaURLs []string) (*OptimizedContent, error)
	ValidateContent(content *OptimizedContent) error
	Post(ctx context.Context, content *OptimizedContent, credentials *SocialCredentials) (*PostResult, error)
	GetEngagementMetrics(ctx context.Context, postID string, credentials *SocialCredentials) (*EngagementMetrics, error)
	RefreshToken(ctx context.Context, refreshToken string) (*TokenRefresh, error)
}

// OptimizedContent represents platform-optimized content
type OptimizedContent struct {
	Platform     string   `json:"platform"`
	Content      string   `json:"content"`
	MediaURLs    []string `json:"media_urls"`
	Hashtags     []string `json:"hashtags"`
	CharCount    int      `json:"char_count"`
	IsValid      bool     `json:"is_valid"`
	Warnings     []string `json:"warnings,omitempty"`
	Suggestions  []string `json:"suggestions,omitempty"`
}

// PostResult represents the result of posting to a platform
type PostResult struct {
	Success        bool   `json:"success"`
	PlatformPostID string `json:"platform_post_id,omitempty"`
	URL            string `json:"url,omitempty"`
	Error          string `json:"error,omitempty"`
	Timestamp      string `json:"timestamp"`
}

// EngagementMetrics represents platform-specific engagement data
type EngagementMetrics struct {
	Platform       string                 `json:"platform"`
	PostID         string                 `json:"post_id"`
	Timestamp      string                 `json:"timestamp"`
	Likes          int                    `json:"likes"`
	Shares         int                    `json:"shares"`
	Comments       int                    `json:"comments"`
	Impressions    int                    `json:"impressions"`
	Reach          int                    `json:"reach"`
	Clicks         int                    `json:"clicks"`
	EngagementRate float64                `json:"engagement_rate"`
	CustomMetrics  map[string]interface{} `json:"custom_metrics,omitempty"`
}

// TokenRefresh represents refreshed OAuth tokens
type TokenRefresh struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// SocialCredentials holds OAuth credentials for a platform
type SocialCredentials struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
}

// PlatformManager manages all platform adapters
type PlatformManager struct {
	adapters  map[string]PlatformAdapter
	ollamaURL string
	httpClient *http.Client
}

// NewPlatformManager creates a new platform manager
func NewPlatformManager(ollamaURL string) *PlatformManager {
	pm := &PlatformManager{
		adapters:  make(map[string]PlatformAdapter),
		ollamaURL: ollamaURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Register platform adapters
	pm.adapters["twitter"] = &TwitterAdapter{manager: pm}
	pm.adapters["instagram"] = &InstagramAdapter{manager: pm}
	pm.adapters["linkedin"] = &LinkedInAdapter{manager: pm}
	pm.adapters["facebook"] = &FacebookAdapter{manager: pm}

	return pm
}

// GetAdapter returns the adapter for a specific platform
func (pm *PlatformManager) GetAdapter(platform string) (PlatformAdapter, error) {
	adapter, exists := pm.adapters[platform]
	if !exists {
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
	return adapter, nil
}

// GetSupportedPlatforms returns list of supported platforms
func (pm *PlatformManager) GetSupportedPlatforms() []string {
	platforms := make([]string, 0, len(pm.adapters))
	for platform := range pm.adapters {
		platforms = append(platforms, platform)
	}
	return platforms
}

// OptimizeWithAI uses Ollama to optimize content for a specific platform
func (pm *PlatformManager) OptimizeWithAI(content, platform string) (string, error) {
	prompt := fmt.Sprintf(`Optimize this social media content for %s:

Original content: %s

Platform-specific requirements:
%s

Return only the optimized content without any explanations or metadata.`, 
		platform, content, pm.getPlatformRequirements(platform))

	payload := map[string]interface{}{
		"model":  "llama3.2",
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature": 0.7,
			"max_tokens":  500,
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal AI request: %w", err)
	}

	resp, err := pm.httpClient.Post(pm.ollamaURL+"/api/generate", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("AI optimization request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI service returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read AI response: %w", err)
	}

	var aiResponse struct {
		Response string `json:"response"`
	}

	if err := json.Unmarshal(body, &aiResponse); err != nil {
		return "", fmt.Errorf("failed to parse AI response: %w", err)
	}

	return strings.TrimSpace(aiResponse.Response), nil
}

func (pm *PlatformManager) getPlatformRequirements(platform string) string {
	requirements := map[string]string{
		"twitter": `- Maximum 280 characters
- Use engaging hashtags (#) but limit to 2-3 relevant ones
- Include emojis for better engagement
- Clear call-to-action if appropriate
- Maintain conversational tone`,
		
		"instagram": `- Engaging caption with storytelling approach
- Use 5-10 relevant hashtags at the end
- Include emojis throughout for visual appeal
- Ask questions to encourage comments
- Maintain authentic, personal tone`,
		
		"linkedin": `- Professional tone while remaining engaging
- Include industry-relevant insights or value
- Use 1-3 professional hashtags
- Structure with clear paragraphs
- Include call-to-action for professional engagement`,
		
		"facebook": `- Conversational and community-focused tone
- Encourage discussion and sharing
- Can be longer-form content
- Include relevant hashtags (1-2)
- Use emojis appropriately for engagement`,
	}

	if req, exists := requirements[platform]; exists {
		return req
	}
	return "- Optimize for general social media best practices"
}

// Twitter Adapter
type TwitterAdapter struct {
	manager *PlatformManager
}

func (t *TwitterAdapter) GetName() string {
	return "twitter"
}

func (t *TwitterAdapter) OptimizeContent(content string, mediaURLs []string) (*OptimizedContent, error) {
	// Use AI to optimize content for Twitter
	optimized, err := t.manager.OptimizeWithAI(content, "twitter")
	if err != nil {
		// Fallback to basic optimization
		optimized = t.basicTwitterOptimization(content)
	}

	// Extract hashtags
	hashtags := t.extractHashtags(optimized)

	result := &OptimizedContent{
		Platform:  "twitter",
		Content:   optimized,
		MediaURLs: mediaURLs,
		Hashtags:  hashtags,
		CharCount: len(optimized),
	}

	// Validate content
	if err := t.ValidateContent(result); err != nil {
		result.IsValid = false
		result.Warnings = append(result.Warnings, err.Error())
	} else {
		result.IsValid = true
	}

	return result, nil
}

func (t *TwitterAdapter) basicTwitterOptimization(content string) string {
	// Basic Twitter optimization without AI
	if len(content) <= 280 {
		return content
	}

	// Truncate and add ellipsis
	truncated := content[:270] + "..."
	
	// Try to truncate at word boundary
	if lastSpace := strings.LastIndex(truncated[:270], " "); lastSpace > 200 {
		truncated = content[:lastSpace] + "..."
	}

	return truncated
}

func (t *TwitterAdapter) ValidateContent(content *OptimizedContent) error {
	if len(content.Content) > 280 {
		return fmt.Errorf("content exceeds 280 character limit (%d characters)", len(content.Content))
	}
	
	if len(content.MediaURLs) > 4 {
		return fmt.Errorf("twitter supports maximum 4 media attachments")
	}

	return nil
}

func (t *TwitterAdapter) Post(ctx context.Context, content *OptimizedContent, credentials *SocialCredentials) (*PostResult, error) {
	// Twitter API v2 posting implementation
	tweetData := map[string]interface{}{
		"text": content.Content,
	}

	// Add media if present
	if len(content.MediaURLs) > 0 {
		// TODO: Implement media upload to Twitter
		// This requires uploading media first, then attaching media_ids
	}

	jsonData, err := json.Marshal(tweetData)
	if err != nil {
		return &PostResult{Success: false, Error: err.Error()}, nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.twitter.com/2/tweets", bytes.NewBuffer(jsonData))
	if err != nil {
		return &PostResult{Success: false, Error: err.Error()}, nil
	}

	req.Header.Set("Authorization", "Bearer "+credentials.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.manager.httpClient.Do(req)
	if err != nil {
		return &PostResult{Success: false, Error: err.Error()}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &PostResult{Success: false, Error: "failed to read response"}, nil
	}

	if resp.StatusCode != http.StatusCreated {
		return &PostResult{
			Success: false,
			Error:   fmt.Sprintf("Twitter API error: %s", string(body)),
		}, nil
	}

	var twitterResponse struct {
		Data struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &twitterResponse); err != nil {
		return &PostResult{Success: false, Error: "failed to parse response"}, nil
	}

	return &PostResult{
		Success:        true,
		PlatformPostID: twitterResponse.Data.ID,
		URL:            fmt.Sprintf("https://twitter.com/%s/status/%s", credentials.Username, twitterResponse.Data.ID),
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (t *TwitterAdapter) GetEngagementMetrics(ctx context.Context, postID string, credentials *SocialCredentials) (*EngagementMetrics, error) {
	// Twitter API v2 metrics endpoint
	url := fmt.Sprintf("https://api.twitter.com/2/tweets/%s?tweet.fields=public_metrics,created_at", postID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+credentials.AccessToken)

	resp, err := t.manager.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse metrics and return EngagementMetrics struct
	// Implementation details for parsing Twitter metrics...

	return &EngagementMetrics{
		Platform:    "twitter",
		PostID:      postID,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		// ... populate metrics
	}, nil
}

func (t *TwitterAdapter) RefreshToken(ctx context.Context, refreshToken string) (*TokenRefresh, error) {
	// Twitter OAuth 2.0 token refresh implementation
	return nil, fmt.Errorf("token refresh not implemented for Twitter")
}

func (t *TwitterAdapter) extractHashtags(content string) []string {
	re := regexp.MustCompile(`#\w+`)
	matches := re.FindAllString(content, -1)
	return matches
}

// Instagram Adapter (similar structure)
type InstagramAdapter struct {
	manager *PlatformManager
}

func (i *InstagramAdapter) GetName() string {
	return "instagram"
}

func (i *InstagramAdapter) OptimizeContent(content string, mediaURLs []string) (*OptimizedContent, error) {
	optimized, err := i.manager.OptimizeWithAI(content, "instagram")
	if err != nil {
		optimized = content // Use original content if AI fails
	}

	hashtags := i.extractHashtags(optimized)

	result := &OptimizedContent{
		Platform:  "instagram",
		Content:   optimized,
		MediaURLs: mediaURLs,
		Hashtags:  hashtags,
		CharCount: len(optimized),
	}

	if err := i.ValidateContent(result); err != nil {
		result.IsValid = false
		result.Warnings = append(result.Warnings, err.Error())
	} else {
		result.IsValid = true
	}

	return result, nil
}

func (i *InstagramAdapter) ValidateContent(content *OptimizedContent) error {
	if len(content.Content) > 2200 {
		return fmt.Errorf("content exceeds Instagram's 2200 character limit")
	}
	
	if len(content.Hashtags) > 30 {
		return fmt.Errorf("Instagram supports maximum 30 hashtags")
	}

	if len(content.MediaURLs) == 0 {
		return fmt.Errorf("Instagram posts require at least one media attachment")
	}

	return nil
}

func (i *InstagramAdapter) Post(ctx context.Context, content *OptimizedContent, credentials *SocialCredentials) (*PostResult, error) {
	// Instagram Basic Display API implementation
	// This is a simplified implementation - full version would handle media upload
	return &PostResult{
		Success:   false,
		Error:     "Instagram posting requires media upload implementation",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (i *InstagramAdapter) GetEngagementMetrics(ctx context.Context, postID string, credentials *SocialCredentials) (*EngagementMetrics, error) {
	return nil, fmt.Errorf("Instagram metrics not implemented")
}

func (i *InstagramAdapter) RefreshToken(ctx context.Context, refreshToken string) (*TokenRefresh, error) {
	return nil, fmt.Errorf("Instagram token refresh not implemented")
}

func (i *InstagramAdapter) extractHashtags(content string) []string {
	re := regexp.MustCompile(`#\w+`)
	matches := re.FindAllString(content, -1)
	return matches
}

// LinkedIn Adapter (similar structure)
type LinkedInAdapter struct {
	manager *PlatformManager
}

func (l *LinkedInAdapter) GetName() string {
	return "linkedin"
}

func (l *LinkedInAdapter) OptimizeContent(content string, mediaURLs []string) (*OptimizedContent, error) {
	optimized, err := l.manager.OptimizeWithAI(content, "linkedin")
	if err != nil {
		optimized = content
	}

	hashtags := l.extractHashtags(optimized)

	result := &OptimizedContent{
		Platform:  "linkedin",
		Content:   optimized,
		MediaURLs: mediaURLs,
		Hashtags:  hashtags,
		CharCount: len(optimized),
		IsValid:   true,
	}

	if err := l.ValidateContent(result); err != nil {
		result.IsValid = false
		result.Warnings = append(result.Warnings, err.Error())
	}

	return result, nil
}

func (l *LinkedInAdapter) ValidateContent(content *OptimizedContent) error {
	if len(content.Content) > 3000 {
		return fmt.Errorf("content exceeds LinkedIn's 3000 character limit")
	}
	return nil
}

func (l *LinkedInAdapter) Post(ctx context.Context, content *OptimizedContent, credentials *SocialCredentials) (*PostResult, error) {
	// LinkedIn API v2 implementation
	return &PostResult{
		Success:   false,
		Error:     "LinkedIn posting implementation pending",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (l *LinkedInAdapter) GetEngagementMetrics(ctx context.Context, postID string, credentials *SocialCredentials) (*EngagementMetrics, error) {
	return nil, fmt.Errorf("LinkedIn metrics not implemented")
}

func (l *LinkedInAdapter) RefreshToken(ctx context.Context, refreshToken string) (*TokenRefresh, error) {
	return nil, fmt.Errorf("LinkedIn token refresh not implemented")
}

func (l *LinkedInAdapter) extractHashtags(content string) []string {
	re := regexp.MustCompile(`#\w+`)
	matches := re.FindAllString(content, -1)
	return matches
}

// Facebook Adapter (similar structure)
type FacebookAdapter struct {
	manager *PlatformManager
}

func (f *FacebookAdapter) GetName() string {
	return "facebook"
}

func (f *FacebookAdapter) OptimizeContent(content string, mediaURLs []string) (*OptimizedContent, error) {
	optimized, err := f.manager.OptimizeWithAI(content, "facebook")
	if err != nil {
		optimized = content
	}

	hashtags := f.extractHashtags(optimized)

	result := &OptimizedContent{
		Platform:  "facebook",
		Content:   optimized,
		MediaURLs: mediaURLs,
		Hashtags:  hashtags,
		CharCount: len(optimized),
		IsValid:   true,
	}

	if err := f.ValidateContent(result); err != nil {
		result.IsValid = false
		result.Warnings = append(result.Warnings, err.Error())
	}

	return result, nil
}

func (f *FacebookAdapter) ValidateContent(content *OptimizedContent) error {
	if len(content.Content) > 63206 {
		return fmt.Errorf("content exceeds Facebook's character limit")
	}
	return nil
}

func (f *FacebookAdapter) Post(ctx context.Context, content *OptimizedContent, credentials *SocialCredentials) (*PostResult, error) {
	// Facebook Graph API implementation
	return &PostResult{
		Success:   false,
		Error:     "Facebook posting implementation pending",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (f *FacebookAdapter) GetEngagementMetrics(ctx context.Context, postID string, credentials *SocialCredentials) (*EngagementMetrics, error) {
	return nil, fmt.Errorf("Facebook metrics not implemented")
}

func (f *FacebookAdapter) RefreshToken(ctx context.Context, refreshToken string) (*TokenRefresh, error) {
	return nil, fmt.Errorf("Facebook token refresh not implemented")
}

func (f *FacebookAdapter) extractHashtags(content string) []string {
	re := regexp.MustCompile(`#\w+`)
	matches := re.FindAllString(content, -1)
	return matches
}
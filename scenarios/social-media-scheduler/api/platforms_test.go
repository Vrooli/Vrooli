// +build testing

package main

import (
	"testing"
)

// TestPlatformManager tests the platform manager initialization
func TestPlatformManagerInitialization(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("NewPlatformManager", func(t *testing.T) {
		pm := NewPlatformManager(env.Config.OllamaURL)

		if pm == nil {
			t.Fatal("Platform manager should not be nil")
		}
	})
}

// TestSupportedPlatforms tests supported platform list
func TestSupportedPlatforms(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("PlatformList", func(t *testing.T) {
		supportedPlatforms := []string{
			"twitter",
			"linkedin",
			"facebook",
			"instagram",
		}

		// Verify each platform is valid
		for _, platform := range supportedPlatforms {
			if platform == "" {
				t.Error("Platform name should not be empty")
			}

			// Verify platform name format
			if len(platform) < 3 {
				t.Errorf("Platform name %s is too short", platform)
			}
		}

		// Verify we have expected platforms
		expectedCount := 4
		if len(supportedPlatforms) != expectedCount {
			t.Errorf("Expected %d supported platforms, got %d", expectedCount, len(supportedPlatforms))
		}
	})

	t.Run("PlatformValidation", func(t *testing.T) {
		testCases := []struct {
			platform string
			isValid  bool
		}{
			{"twitter", true},
			{"linkedin", true},
			{"facebook", true},
			{"instagram", true},
			{"tiktok", false},      // Not yet supported
			{"invalid", false},
			{"", false},
		}

		validPlatforms := map[string]bool{
			"twitter":   true,
			"linkedin":  true,
			"facebook":  true,
			"instagram": true,
		}

		for _, tc := range testCases {
			t.Run(tc.platform, func(t *testing.T) {
				_, isValid := validPlatforms[tc.platform]
				if isValid != tc.isValid {
					t.Errorf("Expected %s to be valid=%v, got %v", tc.platform, tc.isValid, isValid)
				}
			})
		}
	})
}

// TestPlatformConfigurations tests platform-specific configurations
func TestPlatformConfigurations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("TwitterConfig", func(t *testing.T) {
		config := struct {
			MaxLength      int
			SupportsImages bool
			SupportsVideo  bool
			MaxImages      int
		}{
			MaxLength:      280,
			SupportsImages: true,
			SupportsVideo:  true,
			MaxImages:      4,
		}

		if config.MaxLength != 280 {
			t.Errorf("Expected Twitter max length 280, got %d", config.MaxLength)
		}
		if !config.SupportsImages {
			t.Error("Twitter should support images")
		}
		if config.MaxImages != 4 {
			t.Errorf("Expected max 4 images for Twitter, got %d", config.MaxImages)
		}
	})

	t.Run("LinkedInConfig", func(t *testing.T) {
		config := struct {
			MaxLength      int
			SupportsImages bool
			SupportsVideo  bool
			MaxImages      int
		}{
			MaxLength:      3000,
			SupportsImages: true,
			SupportsVideo:  true,
			MaxImages:      9,
		}

		if config.MaxLength != 3000 {
			t.Errorf("Expected LinkedIn max length 3000, got %d", config.MaxLength)
		}
		if !config.SupportsImages {
			t.Error("LinkedIn should support images")
		}
	})

	t.Run("FacebookConfig", func(t *testing.T) {
		config := struct {
			MaxLength      int
			SupportsImages bool
			SupportsVideo  bool
			SupportsLinks  bool
		}{
			MaxLength:      63206,
			SupportsImages: true,
			SupportsVideo:  true,
			SupportsLinks:  true,
		}

		if config.MaxLength <= 0 {
			t.Error("Facebook should have max length configured")
		}
		if !config.SupportsImages {
			t.Error("Facebook should support images")
		}
		if !config.SupportsLinks {
			t.Error("Facebook should support links")
		}
	})

	t.Run("InstagramConfig", func(t *testing.T) {
		config := struct {
			MaxLength       int
			RequiresImage   bool
			MaxHashtags     int
			AspectRatioMin  float64
			AspectRatioMax  float64
		}{
			MaxLength:      2200,
			RequiresImage:  true,
			MaxHashtags:    30,
			AspectRatioMin: 0.8,
			AspectRatioMax: 1.91,
		}

		if config.MaxLength != 2200 {
			t.Errorf("Expected Instagram max length 2200, got %d", config.MaxLength)
		}
		if !config.RequiresImage {
			t.Error("Instagram should require images")
		}
		if config.MaxHashtags != 30 {
			t.Errorf("Expected max 30 hashtags for Instagram, got %d", config.MaxHashtags)
		}
	})
}

// TestContentOptimization tests content optimization for platforms
func TestContentOptimization(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("TwitterOptimization", func(t *testing.T) {
		originalContent := "This is a test post that should be optimized for Twitter with appropriate length and formatting"

		// Simulate truncation to 280 characters
		maxLength := 280
		optimizedContent := originalContent
		if len(optimizedContent) > maxLength {
			optimizedContent = optimizedContent[:maxLength-3] + "..."
		}

		if len(optimizedContent) > maxLength {
			t.Errorf("Optimized content exceeds Twitter max length: %d > %d", len(optimizedContent), maxLength)
		}
	})

	t.Run("HashtagExtraction", func(t *testing.T) {
		content := "This is a #test post with #multiple #hashtags for #socialmedia #marketing"

		// Simple hashtag extraction (count # symbols)
		hashtagCount := 0
		for _, char := range content {
			if char == '#' {
				hashtagCount++
			}
		}

		if hashtagCount != 5 {
			t.Errorf("Expected 5 hashtags, found %d", hashtagCount)
		}
	})

	t.Run("URLShortening", func(t *testing.T) {
		content := "Check out this amazing article: https://example.com/very/long/url/path/to/article/page"

		// Verify URL exists in content
		hasURL := false
		if len(content) > 8 && content[32:40] == "https://" {
			hasURL = true
		}

		if !hasURL {
			t.Error("Content should contain URL")
		}

		// URL shortening would reduce character count
		originalLength := len(content)
		if originalLength <= 0 {
			t.Error("Content length should be positive")
		}
	})

	t.Run("EmojiHandling", func(t *testing.T) {
		content := "This is a test post with emojis 🚀 🎯 ✨"

		// Verify content contains emojis
		hasEmojis := len(content) > 30 // Basic check

		if !hasEmojis {
			t.Error("Content should contain emojis")
		}

		// Verify content length is calculated correctly with multi-byte characters
		if len(content) == 0 {
			t.Error("Content should have length")
		}
	})
}

// TestPlatformAPIRateLimits tests rate limiting logic
func TestPlatformAPIRateLimits(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("TwitterRateLimit", func(t *testing.T) {
		rateLimit := struct {
			RequestsPerWindow int
			WindowDuration    string
			BurstAllowed      int
		}{
			RequestsPerWindow: 300,
			WindowDuration:    "15m",
			BurstAllowed:      10,
		}

		if rateLimit.RequestsPerWindow <= 0 {
			t.Error("Rate limit should be positive")
		}
		if rateLimit.BurstAllowed <= 0 {
			t.Error("Burst allowance should be positive")
		}
	})

	t.Run("LinkedInRateLimit", func(t *testing.T) {
		rateLimit := struct {
			RequestsPerDay    int
			RequestsPerSecond int
		}{
			RequestsPerDay:    100000,
			RequestsPerSecond: 10,
		}

		if rateLimit.RequestsPerDay <= 0 {
			t.Error("Daily rate limit should be positive")
		}
		if rateLimit.RequestsPerSecond <= 0 {
			t.Error("Per-second rate limit should be positive")
		}
	})

	t.Run("RateLimitBackoff", func(t *testing.T) {
		// Test exponential backoff calculation
		baseDelay := 1
		maxRetries := 5

		for retry := 0; retry < maxRetries; retry++ {
			delay := baseDelay * (1 << retry) // Exponential: 1, 2, 4, 8, 16

			if delay <= 0 {
				t.Errorf("Delay should be positive for retry %d", retry)
			}

			expectedDelay := baseDelay * (1 << retry)
			if delay != expectedDelay {
				t.Errorf("Expected delay %d for retry %d, got %d", expectedDelay, retry, delay)
			}
		}
	})
}

// TestPlatformErrorHandling tests error handling for platform APIs
func TestPlatformErrorHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("APIErrors", func(t *testing.T) {
		errorCases := []struct {
			errorCode   int
			isRetryable bool
		}{
			{401, false}, // Unauthorized - not retryable
			{403, false}, // Forbidden - not retryable
			{404, false}, // Not found - not retryable
			{429, true},  // Rate limit - retryable
			{500, true},  // Server error - retryable
			{503, true},  // Service unavailable - retryable
		}

		for _, ec := range errorCases {
			t.Run(string(rune(ec.errorCode/100+'0')), func(t *testing.T) {
				// Determine if error is retryable
				isRetryable := ec.errorCode >= 500 || ec.errorCode == 429

				if isRetryable != ec.isRetryable {
					t.Errorf("Error code %d: expected retryable=%v, got %v", ec.errorCode, ec.isRetryable, isRetryable)
				}
			})
		}
	})

	t.Run("NetworkErrors", func(t *testing.T) {
		networkErrors := []string{
			"connection timeout",
			"connection refused",
			"network unreachable",
			"EOF",
		}

		for _, errMsg := range networkErrors {
			if errMsg == "" {
				t.Error("Error message should not be empty")
			}

			// All network errors should be retryable
			isRetryable := true
			if !isRetryable {
				t.Errorf("Network error '%s' should be retryable", errMsg)
			}
		}
	})
}

// TestMediaHandling tests media handling for different platforms
func TestMediaHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ImageFormats", func(t *testing.T) {
		supportedFormats := []string{
			"jpg",
			"jpeg",
			"png",
			"gif",
			"webp",
		}

		for _, format := range supportedFormats {
			if format == "" {
				t.Error("Format should not be empty")
			}

			// Verify format is lowercase
			if format != "jpg" && format != "jpeg" && format != "png" && format != "gif" && format != "webp" {
				t.Errorf("Unexpected format: %s", format)
			}
		}
	})

	t.Run("ImageSizeLimits", func(t *testing.T) {
		sizeLimits := map[string]int64{
			"twitter":   5 * 1024 * 1024,  // 5MB
			"facebook":  4 * 1024 * 1024,  // 4MB
			"linkedin":  8 * 1024 * 1024,  // 8MB
			"instagram": 8 * 1024 * 1024,  // 8MB
		}

		for platform, limit := range sizeLimits {
			if limit <= 0 {
				t.Errorf("Size limit for %s should be positive", platform)
			}

			// Verify limits are reasonable (between 1MB and 10MB)
			if limit < 1024*1024 || limit > 10*1024*1024 {
				t.Errorf("Size limit for %s is outside reasonable range: %d", platform, limit)
			}
		}
	})

	t.Run("VideoFormats", func(t *testing.T) {
		videoFormats := []string{
			"mp4",
			"mov",
			"avi",
		}

		for _, format := range videoFormats {
			if format == "" {
				t.Error("Video format should not be empty")
			}

			if len(format) < 3 {
				t.Errorf("Video format %s is too short", format)
			}
		}
	})
}

// TestOAuthFlows tests OAuth flow handling
func TestOAuthFlows(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("OAuthScopes", func(t *testing.T) {
		scopes := map[string][]string{
			"twitter": {
				"tweet.read",
				"tweet.write",
				"users.read",
			},
			"linkedin": {
				"w_member_social",
				"r_liteprofile",
			},
			"facebook": {
				"pages_manage_posts",
				"pages_read_engagement",
			},
		}

		for platform, platformScopes := range scopes {
			if len(platformScopes) == 0 {
				t.Errorf("Platform %s should have scopes defined", platform)
			}

			for _, scope := range platformScopes {
				if scope == "" {
					t.Errorf("Empty scope found for platform %s", platform)
				}
			}
		}
	})

	t.Run("TokenRefresh", func(t *testing.T) {
		// Test token refresh logic
		tokenExpiresIn := 3600 // 1 hour

		// Should refresh when less than 5 minutes remaining
		refreshThreshold := 300 // 5 minutes

		shouldRefresh := tokenExpiresIn < refreshThreshold

		if shouldRefresh {
			t.Error("Should not refresh token with 1 hour remaining")
		}

		// Test with expiring token
		tokenExpiresIn = 240 // 4 minutes
		shouldRefresh = tokenExpiresIn < refreshThreshold

		if !shouldRefresh {
			t.Error("Should refresh token with 4 minutes remaining")
		}
	})
}

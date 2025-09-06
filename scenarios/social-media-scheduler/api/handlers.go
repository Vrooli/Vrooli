package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// User models
type User struct {
	ID               string    `json:"id"`
	Email            string    `json:"email"`
	FirstName        string    `json:"first_name,omitempty"`
	LastName         string    `json:"last_name,omitempty"`
	SubscriptionTier string    `json:"subscription_tier"`
	Timezone         string    `json:"timezone"`
	Preferences      string    `json:"preferences,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Timezone  string `json:"timezone"`
}

type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// Scheduled post models
type SchedulePostRequest struct {
	Title           string            `json:"title" binding:"required"`
	Content         string            `json:"content" binding:"required"`
	Platforms       []string          `json:"platforms" binding:"required"`
	ScheduledAt     string            `json:"scheduled_at" binding:"required"`
	Timezone        string            `json:"timezone" binding:"required"`
	CampaignID      *string           `json:"campaign_id,omitempty"`
	MediaFiles      []string          `json:"media_files,omitempty"`
	AutoOptimize    bool              `json:"auto_optimize"`
	PlatformVariants map[string]string `json:"platform_variants,omitempty"`
}

type ScheduledPost struct {
	ID               string                 `json:"id"`
	UserID           string                 `json:"user_id"`
	CampaignID       *string                `json:"campaign_id,omitempty"`
	Title            string                 `json:"title"`
	Content          string                 `json:"content"`
	PlatformVariants map[string]string      `json:"platform_variants"`
	MediaURLs        []string               `json:"media_urls"`
	Platforms        []string               `json:"platforms"`
	ScheduledAt      time.Time              `json:"scheduled_at"`
	Timezone         string                 `json:"timezone"`
	Status           string                 `json:"status"`
	PostedAt         *time.Time             `json:"posted_at,omitempty"`
	AnalyticsData    map[string]interface{} `json:"analytics_data,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// Campaign models
type Campaign struct {
	ID               string                 `json:"id"`
	UserID           string                 `json:"user_id"`
	ExternalCampaignID *string              `json:"external_campaign_id,omitempty"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	BrandGuidelines  map[string]interface{} `json:"brand_guidelines,omitempty"`
	Status           string                 `json:"status"`
	StartDate        *string                `json:"start_date,omitempty"`
	EndDate          *string                `json:"end_date,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

type CreateCampaignRequest struct {
	Name             string                 `json:"name" binding:"required"`
	Description      string                 `json:"description"`
	BrandGuidelines  map[string]interface{} `json:"brand_guidelines,omitempty"`
	StartDate        *string                `json:"start_date,omitempty"`
	EndDate          *string                `json:"end_date,omitempty"`
	ExternalCampaignID *string              `json:"external_campaign_id,omitempty"`
}

// Social account models
type SocialAccount struct {
	ID            string                 `json:"id"`
	Platform      string                 `json:"platform"`
	Username      string                 `json:"username"`
	DisplayName   string                 `json:"display_name"`
	AvatarURL     string                 `json:"avatar_url,omitempty"`
	AccountData   map[string]interface{} `json:"account_data,omitempty"`
	IsActive      bool                   `json:"is_active"`
	LastUsedAt    *time.Time             `json:"last_used_at,omitempty"`
	ConnectedAt   time.Time              `json:"connected_at"`
}

// Authentication handlers
func (app *Application) login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Success: false, Error: err.Error()})
		return
	}

	// Get user from database
	var user User
	var passwordHash string
	query := "SELECT id, email, first_name, last_name, subscription_tier, timezone, password_hash, created_at FROM users WHERE email = $1"
	
	err := app.DB.QueryRow(query, req.Email).Scan(
		&user.ID, &user.Email, &user.FirstName, &user.LastName, 
		&user.SubscriptionTier, &user.Timezone, &passwordHash, &user.CreatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, Response{Success: false, Error: "Invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, Response{Success: false, Error: "Database error"})
		return
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, Response{Success: false, Error: "Invalid credentials"})
		return
	}

	// Update last login
	_, err = app.DB.Exec("UPDATE users SET last_login_at = NOW() WHERE id = $1", user.ID)
	if err != nil {
		// Log error but don't fail the login
		fmt.Printf("Failed to update last login: %v\n", err)
	}

	// Generate JWT token
	token, err := app.generateJWTToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Success: false, Error: "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"user":  user,
			"token": token,
		},
	})
}

func (app *Application) register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Success: false, Error: err.Error()})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Success: false, Error: "Failed to hash password"})
		return
	}

	// Set default timezone if not provided
	timezone := req.Timezone
	if timezone == "" {
		timezone = "UTC"
	}

	// Create user
	userID := uuid.New().String()
	query := `
		INSERT INTO users (id, email, password_hash, first_name, last_name, timezone, subscription_tier)
		VALUES ($1, $2, $3, $4, $5, $6, 'free')
		RETURNING created_at`

	var createdAt time.Time
	err = app.DB.QueryRow(query, userID, req.Email, string(hashedPassword), req.FirstName, req.LastName, timezone).Scan(&createdAt)
	if err != nil {
		// Check if it's a unique constraint violation
		if err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"` {
			c.JSON(http.StatusConflict, Response{Success: false, Error: "Email already registered"})
			return
		}
		c.JSON(http.StatusInternalServerError, Response{Success: false, Error: "Failed to create user"})
		return
	}

	// Create user object
	user := User{
		ID:               userID,
		Email:            req.Email,
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		SubscriptionTier: "free",
		Timezone:         timezone,
		CreatedAt:        createdAt,
	}

	// Generate JWT token
	token, err := app.generateJWTToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Success: false, Error: "Failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data: map[string]interface{}{
			"user":  user,
			"token": token,
		},
	})
}

func (app *Application) logout(c *gin.Context) {
	// For JWT, logout is typically handled client-side by removing the token
	// In a production system, you might want to maintain a blacklist of tokens
	c.JSON(http.StatusOK, Response{Success: true, Data: map[string]string{"message": "Logged out successfully"}})
}

func (app *Application) getCurrentUser(c *gin.Context) {
	userID := c.GetString("user_id")
	
	var user User
	query := "SELECT id, email, first_name, last_name, subscription_tier, timezone, created_at FROM users WHERE id = $1"
	
	err := app.DB.QueryRow(query, userID).Scan(
		&user.ID, &user.Email, &user.FirstName, &user.LastName,
		&user.SubscriptionTier, &user.Timezone, &user.CreatedAt,
	)
	
	if err != nil {
		c.JSON(http.StatusNotFound, Response{Success: false, Error: "User not found"})
		return
	}

	c.JSON(http.StatusOK, Response{Success: true, Data: user})
}

func (app *Application) getPlatformConfigs(c *gin.Context) {
	platforms := []map[string]interface{}{
		{
			"platform":     "twitter",
			"display_name": "Twitter",
			"color":        "#1DA1F2",
			"max_chars":    280,
			"media_limit":  4,
			"features":     []string{"text", "images", "videos", "hashtags"},
		},
		{
			"platform":     "instagram",
			"display_name": "Instagram",
			"color":        "#E4405F",
			"max_chars":    2200,
			"media_limit":  10,
			"features":     []string{"images", "videos", "stories", "hashtags"},
			"requirements": []string{"media_required"},
		},
		{
			"platform":     "linkedin",
			"display_name": "LinkedIn",
			"color":        "#0077B5",
			"max_chars":    3000,
			"media_limit":  9,
			"features":     []string{"text", "images", "videos", "articles", "hashtags"},
		},
		{
			"platform":     "facebook",
			"display_name": "Facebook",
			"color":        "#1877F2",
			"max_chars":    63206,
			"media_limit":  10,
			"features":     []string{"text", "images", "videos", "events", "hashtags"},
		},
	}

	c.JSON(http.StatusOK, Response{Success: true, Data: platforms})
}

// Post scheduling handlers
func (app *Application) schedulePost(c *gin.Context) {
	userID := c.GetString("user_id")
	
	var req SchedulePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Success: false, Error: err.Error()})
		return
	}

	// Parse scheduled time
	scheduledAt, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Success: false, Error: "Invalid scheduled_at format"})
		return
	}

	// Validate that scheduled time is in the future
	if scheduledAt.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, Response{Success: false, Error: "Scheduled time must be in the future"})
		return
	}

	// Validate platforms
	supportedPlatforms := app.PlatformMgr.GetSupportedPlatforms()
	for _, platform := range req.Platforms {
		if !contains(supportedPlatforms, platform) {
			c.JSON(http.StatusBadRequest, Response{Success: false, Error: fmt.Sprintf("Unsupported platform: %s", platform)})
			return
		}
	}

	// Check user's usage limits
	// TODO: Implement usage limit checking based on subscription tier

	// Generate platform variants if auto-optimize is enabled
	platformVariants := req.PlatformVariants
	if req.AutoOptimize {
		platformVariants = make(map[string]string)
		for _, platform := range req.Platforms {
			adapter, err := app.PlatformMgr.GetAdapter(platform)
			if err != nil {
				c.JSON(http.StatusBadRequest, Response{Success: false, Error: err.Error()})
				return
			}

			optimized, err := adapter.OptimizeContent(req.Content, req.MediaFiles)
			if err != nil {
				// Fallback to original content if optimization fails
				platformVariants[platform] = req.Content
			} else {
				platformVariants[platform] = optimized.Content
			}
		}
	} else if platformVariants == nil {
		// Use original content for all platforms
		platformVariants = make(map[string]string)
		for _, platform := range req.Platforms {
			platformVariants[platform] = req.Content
		}
	}

	// Create scheduled post
	postID := uuid.New().String()
	
	platformVariantsJSON, _ := json.Marshal(platformVariants)
	mediaURLsJSON, _ := json.Marshal(req.MediaFiles)
	platformsJSON, _ := json.Marshal(req.Platforms)

	query := `
		INSERT INTO scheduled_posts (id, user_id, campaign_id, title, content, platform_variants, media_urls, platforms, scheduled_at, timezone, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'scheduled')
		RETURNING created_at, updated_at`

	var createdAt, updatedAt time.Time
	err = app.DB.QueryRow(query, postID, userID, req.CampaignID, req.Title, req.Content, 
		string(platformVariantsJSON), string(mediaURLsJSON), string(platformsJSON), 
		scheduledAt, req.Timezone).Scan(&createdAt, &updatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Success: false, Error: "Failed to schedule post"})
		return
	}

	// Create response object
	scheduledPost := ScheduledPost{
		ID:               postID,
		UserID:           userID,
		CampaignID:       req.CampaignID,
		Title:            req.Title,
		Content:          req.Content,
		PlatformVariants: platformVariants,
		MediaURLs:        req.MediaFiles,
		Platforms:        req.Platforms,
		ScheduledAt:      scheduledAt,
		Timezone:         req.Timezone,
		Status:           "scheduled",
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}

	// Send real-time notification
	app.WebSocket.NotifyPostScheduled(userID, scheduledPost)

	c.JSON(http.StatusCreated, Response{Success: true, Data: scheduledPost})
}

func (app *Application) getCalendarPosts(c *gin.Context) {
	userID := c.GetString("user_id")
	
	// Parse query parameters
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	platforms := c.QueryArray("platforms")

	// Build query
	query := `
		SELECT id, user_id, campaign_id, title, content, platform_variants, media_urls, platforms, 
			   scheduled_at, timezone, status, posted_at, analytics_data, created_at, updated_at
		FROM scheduled_posts 
		WHERE user_id = $1`
	args := []interface{}{userID}
	argIndex := 2

	if startDate != "" {
		query += fmt.Sprintf(" AND scheduled_at >= $%d", argIndex)
		args = append(args, startDate)
		argIndex++
	}

	if endDate != "" {
		query += fmt.Sprintf(" AND scheduled_at <= $%d", argIndex)
		args = append(args, endDate)
		argIndex++
	}

	if len(platforms) > 0 {
		query += fmt.Sprintf(" AND platforms && $%d", argIndex)
		platformsJSON, _ := json.Marshal(platforms)
		args = append(args, string(platformsJSON))
		argIndex++
	}

	query += " ORDER BY scheduled_at ASC"

	rows, err := app.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Success: false, Error: "Database error"})
		return
	}
	defer rows.Close()

	var posts []ScheduledPost
	for rows.Next() {
		var post ScheduledPost
		var campaignID sql.NullString
		var postedAt sql.NullTime
		var platformVariantsJSON, mediaURLsJSON, platformsJSON, analyticsJSON string

		err := rows.Scan(
			&post.ID, &post.UserID, &campaignID, &post.Title, &post.Content,
			&platformVariantsJSON, &mediaURLsJSON, &platformsJSON,
			&post.ScheduledAt, &post.Timezone, &post.Status, &postedAt,
			&analyticsJSON, &post.CreatedAt, &post.UpdatedAt,
		)

		if err != nil {
			continue // Skip invalid rows
		}

		// Parse JSON fields
		json.Unmarshal([]byte(platformVariantsJSON), &post.PlatformVariants)
		json.Unmarshal([]byte(mediaURLsJSON), &post.MediaURLs)
		json.Unmarshal([]byte(platformsJSON), &post.Platforms)
		json.Unmarshal([]byte(analyticsJSON), &post.AnalyticsData)

		if campaignID.Valid {
			post.CampaignID = &campaignID.String
		}
		if postedAt.Valid {
			post.PostedAt = &postedAt.Time
		}

		posts = append(posts, post)
	}

	c.JSON(http.StatusOK, Response{Success: true, Data: posts})
}

func (app *Application) getPost(c *gin.Context) {
	userID := c.GetString("user_id")
	postID := c.Param("id")

	var post ScheduledPost
	var campaignID sql.NullString
	var postedAt sql.NullTime
	var platformVariantsJSON, mediaURLsJSON, platformsJSON, analyticsJSON string

	query := `
		SELECT id, user_id, campaign_id, title, content, platform_variants, media_urls, platforms,
			   scheduled_at, timezone, status, posted_at, analytics_data, created_at, updated_at
		FROM scheduled_posts 
		WHERE id = $1 AND user_id = $2`

	err := app.DB.QueryRow(query, postID, userID).Scan(
		&post.ID, &post.UserID, &campaignID, &post.Title, &post.Content,
		&platformVariantsJSON, &mediaURLsJSON, &platformsJSON,
		&post.ScheduledAt, &post.Timezone, &post.Status, &postedAt,
		&analyticsJSON, &post.CreatedAt, &post.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, Response{Success: false, Error: "Post not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, Response{Success: false, Error: "Database error"})
		return
	}

	// Parse JSON fields
	json.Unmarshal([]byte(platformVariantsJSON), &post.PlatformVariants)
	json.Unmarshal([]byte(mediaURLsJSON), &post.MediaURLs)
	json.Unmarshal([]byte(platformsJSON), &post.Platforms)
	json.Unmarshal([]byte(analyticsJSON), &post.AnalyticsData)

	if campaignID.Valid {
		post.CampaignID = &campaignID.String
	}
	if postedAt.Valid {
		post.PostedAt = &postedAt.Time
	}

	c.JSON(http.StatusOK, Response{Success: true, Data: post})
}

// Helper functions
func (app *Application) authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, Response{Success: false, Error: "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			c.JSON(http.StatusUnauthorized, Response{Success: false, Error: "Invalid authorization format"})
			c.Abort()
			return
		}

		tokenString := authHeader[7:]

		// Parse and validate JWT token
		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(app.Config.JWTSecret), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, Response{Success: false, Error: "Invalid token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
			c.Set("user_id", claims.UserID)
			c.Set("email", claims.Email)
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, Response{Success: false, Error: "Invalid token claims"})
			c.Abort()
			return
		}
	}
}

func (app *Application) generateJWTToken(user *User) (string, error) {
	claims := JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(app.Config.JWTSecret))
}

// Utility functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Stub handlers for remaining endpoints (to be implemented)
func (app *Application) updatePost(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) deletePost(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) optimizePost(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) duplicatePost(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) previewPost(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) bulkSchedule(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) importPosts(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) bulkReschedule(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) getCampaigns(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) createCampaign(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) getCampaign(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) updateCampaign(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) deleteCampaign(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) getUserSocialAccounts(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) getUserStats(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) updateUserPreferences(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) initiateOAuth(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) handleOAuthCallback(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) disconnectPlatform(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) getAnalyticsOverview(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) getPlatformAnalytics(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) getPostMetrics(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) getOptimalPostingTimes(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) uploadMedia(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) getMediaFiles(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) deleteMedia(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}

func (app *Application) optimizeMediaForPlatforms(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{Success: false, Error: "Not implemented"})
}
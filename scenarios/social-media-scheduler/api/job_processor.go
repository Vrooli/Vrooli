package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

// JobProcessor handles background job processing for scheduled posts
type JobProcessor struct {
	db          *sql.DB
	redis       *redis.Client
	platformMgr *PlatformManager
	running     bool
	stopChan    chan struct{}
}

// PostJob represents a job in the processing queue
type PostJob struct {
	ID             string    `json:"id"`
	ScheduledPostID string    `json:"scheduled_post_id"`
	UserID         string    `json:"user_id"`
	Platform       string    `json:"platform"`
	SocialAccountID string    `json:"social_account_id"`
	Content        string    `json:"content"`
	MediaURLs      []string  `json:"media_urls"`
	ScheduledAt    time.Time `json:"scheduled_at"`
	RetryCount     int       `json:"retry_count"`
	MaxRetries     int       `json:"max_retries"`
	Priority       int       `json:"priority"`
	CreatedAt      time.Time `json:"created_at"`
}

// JobResult represents the result of processing a job
type JobResult struct {
	JobID          string `json:"job_id"`
	Success        bool   `json:"success"`
	PlatformPostID string `json:"platform_post_id,omitempty"`
	PostURL        string `json:"post_url,omitempty"`
	Error          string `json:"error,omitempty"`
	ProcessedAt    time.Time `json:"processed_at"`
	NextRetryAt    *time.Time `json:"next_retry_at,omitempty"`
}

// NewJobProcessor creates a new job processor
func NewJobProcessor(db *sql.DB, redis *redis.Client, platformMgr *PlatformManager) *JobProcessor {
	return &JobProcessor{
		db:          db,
		redis:       redis,
		platformMgr: platformMgr,
		stopChan:    make(chan struct{}),
	}
}

// Start begins the job processing loop
func (jp *JobProcessor) Start(ctx context.Context) {
	jp.running = true
	log.Println("🚀 Job processor started")

	// Start multiple workers for concurrent processing
	for i := 0; i < 5; i++ {
		go jp.worker(ctx, i+1)
	}

	// Start scheduler that moves scheduled posts to the processing queue
	go jp.scheduler(ctx)

	// Start retry handler for failed jobs
	go jp.retryHandler(ctx)

	// Wait for shutdown signal
	select {
	case <-ctx.Done():
		log.Println("📴 Job processor received shutdown signal")
	case <-jp.stopChan:
		log.Println("📴 Job processor stopped")
	}

	jp.running = false
	log.Println("✅ Job processor stopped")
}

// Stop stops the job processor
func (jp *JobProcessor) Stop() {
	if jp.running {
		close(jp.stopChan)
	}
}

// scheduler moves scheduled posts to the processing queue when their time arrives
func (jp *JobProcessor) scheduler(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	log.Println("📅 Scheduler started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := jp.processScheduledPosts(ctx); err != nil {
				log.Printf("⚠️  Scheduler error: %v", err)
			}
		}
	}
}

// processScheduledPosts finds posts ready to be published and queues them
func (jp *JobProcessor) processScheduledPosts(ctx context.Context) error {
	query := `
		SELECT sp.id, sp.user_id, sp.content, sp.platform_variants, sp.media_urls, sp.platforms, sp.scheduled_at, sp.timezone
		FROM scheduled_posts sp
		WHERE sp.status = 'scheduled' 
		AND sp.scheduled_at <= NOW()
		AND sp.scheduled_at >= NOW() - INTERVAL '5 minutes'
		ORDER BY sp.scheduled_at ASC
		LIMIT 100`

	rows, err := jp.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query scheduled posts: %w", err)
	}
	defer rows.Close()

	var processedCount int

	for rows.Next() {
		var (
			postID          string
			userID          string
			content         string
			platformVariants string
			mediaURLs       string
			platforms       string
			scheduledAt     time.Time
			timezone        string
		)

		if err := rows.Scan(&postID, &userID, &content, &platformVariants, &mediaURLs, &platforms, &scheduledAt, &timezone); err != nil {
			log.Printf("⚠️  Error scanning scheduled post: %v", err)
			continue
		}

		// Parse platform variants
		var variants map[string]string
		if err := json.Unmarshal([]byte(platformVariants), &variants); err != nil {
			log.Printf("⚠️  Error parsing platform variants for post %s: %v", postID, err)
			continue
		}

		// Parse media URLs
		var mediaList []string
		if err := json.Unmarshal([]byte(mediaURLs), &mediaList); err != nil {
			log.Printf("⚠️  Error parsing media URLs for post %s: %v", postID, err)
		}

		// Parse platforms
		var platformList []string
		if err := json.Unmarshal([]byte(platforms), &platformList); err != nil {
			log.Printf("⚠️  Error parsing platforms for post %s: %v", postID, err)
			continue
		}

		// Create jobs for each platform
		for _, platform := range platformList {
			if err := jp.createPlatformJob(ctx, postID, userID, platform, variants, mediaList, scheduledAt); err != nil {
				log.Printf("⚠️  Error creating platform job: %v", err)
			} else {
				processedCount++
			}
		}

		// Update post status to 'posting'
		if err := jp.updatePostStatus(ctx, postID, "posting"); err != nil {
			log.Printf("⚠️  Error updating post status: %v", err)
		}
	}

	if processedCount > 0 {
		log.Printf("📤 Queued %d platform jobs for processing", processedCount)
	}

	return nil
}

// createPlatformJob creates a job for posting to a specific platform
func (jp *JobProcessor) createPlatformJob(ctx context.Context, postID, userID, platform string, variants map[string]string, mediaURLs []string, scheduledAt time.Time) error {
	// Get social account for this platform
	socialAccountID, err := jp.getSocialAccountID(ctx, userID, platform)
	if err != nil {
		return fmt.Errorf("failed to get social account for platform %s: %w", platform, err)
	}

	// Get platform-specific content
	content := variants[platform]
	if content == "" {
		// Fallback to optimizing the original content
		content = variants["original"]
		if content == "" {
			return fmt.Errorf("no content available for platform %s", platform)
		}
	}

	job := PostJob{
		ID:             fmt.Sprintf("%s_%s_%d", postID, platform, time.Now().UnixNano()),
		ScheduledPostID: postID,
		UserID:         userID,
		Platform:       platform,
		SocialAccountID: socialAccountID,
		Content:        content,
		MediaURLs:      mediaURLs,
		ScheduledAt:    scheduledAt,
		RetryCount:     0,
		MaxRetries:     3,
		Priority:       1,
		CreatedAt:      time.Now().UTC(),
	}

	// Serialize job
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	// Add to processing queue
	if err := jp.redis.LPush(ctx, "queue:processing", jobData).Err(); err != nil {
		return fmt.Errorf("failed to queue job: %w", err)
	}

	log.Printf("📋 Created job %s for platform %s", job.ID, platform)
	return nil
}

// getSocialAccountID retrieves the social account ID for a user and platform
func (jp *JobProcessor) getSocialAccountID(ctx context.Context, userID, platform string) (string, error) {
	query := `SELECT id FROM social_accounts WHERE user_id = $1 AND platform = $2 AND is_active = true LIMIT 1`
	
	var accountID string
	err := jp.db.QueryRowContext(ctx, query, userID, platform).Scan(&accountID)
	if err != nil {
		return "", fmt.Errorf("no active social account found for user %s on platform %s: %w", userID, platform, err)
	}

	return accountID, nil
}

// worker processes jobs from the processing queue
func (jp *JobProcessor) worker(ctx context.Context, workerID int) {
	log.Printf("👷 Worker %d started", workerID)

	for {
		select {
		case <-ctx.Done():
			log.Printf("👷 Worker %d stopped", workerID)
			return
		default:
			// Process next job
			if err := jp.processNextJob(ctx, workerID); err != nil {
				log.Printf("⚠️  Worker %d error: %v", workerID, err)
				time.Sleep(5 * time.Second) // Wait before retrying
			}
		}
	}
}

// processNextJob processes the next available job
func (jp *JobProcessor) processNextJob(ctx context.Context, workerID int) error {
	// Block and wait for a job (with timeout)
	result, err := jp.redis.BRPop(ctx, 10*time.Second, "queue:processing").Result()
	if err != nil {
		if err == redis.Nil {
			// No jobs available, this is normal
			return nil
		}
		return fmt.Errorf("failed to pop job from queue: %w", err)
	}

	if len(result) != 2 {
		return fmt.Errorf("unexpected queue result format")
	}

	// Parse job data
	var job PostJob
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		return fmt.Errorf("failed to unmarshal job: %w", err)
	}

	log.Printf("👷 Worker %d processing job %s for platform %s", workerID, job.ID, job.Platform)

	// Process the job
	result := jp.executeJob(ctx, &job)

	// Handle result
	if err := jp.handleJobResult(ctx, &job, result); err != nil {
		log.Printf("⚠️  Error handling job result: %v", err)
	}

	if result.Success {
		log.Printf("✅ Worker %d successfully posted to %s (ID: %s)", workerID, job.Platform, result.PlatformPostID)
	} else {
		log.Printf("❌ Worker %d failed to post to %s: %s", workerID, job.Platform, result.Error)
	}

	return nil
}

// executeJob executes a single posting job
func (jp *JobProcessor) executeJob(ctx context.Context, job *PostJob) *JobResult {
	result := &JobResult{
		JobID:       job.ID,
		ProcessedAt: time.Now().UTC(),
	}

	// Get social account credentials
	credentials, err := jp.getSocialCredentials(ctx, job.SocialAccountID)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to get credentials: %v", err)
		return result
	}

	// Check if token needs refresh
	if credentials.ExpiresAt.Before(time.Now().Add(5 * time.Minute)) {
		if err := jp.refreshTokenIfNeeded(ctx, credentials, job.Platform); err != nil {
			result.Error = fmt.Sprintf("Failed to refresh token: %v", err)
			return result
		}
	}

	// Get platform adapter
	adapter, err := jp.platformMgr.GetAdapter(job.Platform)
	if err != nil {
		result.Error = fmt.Sprintf("Unsupported platform: %v", err)
		return result
	}

	// Optimize content
	optimizedContent, err := adapter.OptimizeContent(job.Content, job.MediaURLs)
	if err != nil {
		result.Error = fmt.Sprintf("Content optimization failed: %v", err)
		return result
	}

	if !optimizedContent.IsValid {
		result.Error = fmt.Sprintf("Content validation failed: %v", optimizedContent.Warnings)
		return result
	}

	// Post to platform
	postResult, err := adapter.Post(ctx, optimizedContent, credentials)
	if err != nil {
		result.Error = fmt.Sprintf("Platform posting failed: %v", err)
		return result
	}

	if !postResult.Success {
		result.Error = postResult.Error
		return result
	}

	// Success!
	result.Success = true
	result.PlatformPostID = postResult.PlatformPostID
	result.PostURL = postResult.URL

	return result
}

// getSocialCredentials retrieves OAuth credentials for a social account
func (jp *JobProcessor) getSocialCredentials(ctx context.Context, accountID string) (*SocialCredentials, error) {
	query := `
		SELECT platform_user_id, username, access_token, refresh_token, token_expires_at
		FROM social_accounts
		WHERE id = $1 AND is_active = true`

	var credentials SocialCredentials
	var refreshToken sql.NullString
	var expiresAt sql.NullTime

	err := jp.db.QueryRowContext(ctx, query, accountID).Scan(
		&credentials.UserID,
		&credentials.Username,
		&credentials.AccessToken,
		&refreshToken,
		&expiresAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	if refreshToken.Valid {
		credentials.RefreshToken = refreshToken.String
	}
	if expiresAt.Valid {
		credentials.ExpiresAt = expiresAt.Time
	}

	return &credentials, nil
}

// refreshTokenIfNeeded refreshes OAuth token if it's close to expiring
func (jp *JobProcessor) refreshTokenIfNeeded(ctx context.Context, credentials *SocialCredentials, platform string) error {
	if credentials.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	adapter, err := jp.platformMgr.GetAdapter(platform)
	if err != nil {
		return err
	}

	refreshed, err := adapter.RefreshToken(ctx, credentials.RefreshToken)
	if err != nil {
		return err
	}

	// Update credentials in database
	query := `UPDATE social_accounts SET access_token = $1, refresh_token = $2, token_expires_at = $3, updated_at = NOW() WHERE platform_user_id = $4 AND platform = $5`
	
	_, err = jp.db.ExecContext(ctx, query, refreshed.AccessToken, refreshed.RefreshToken, refreshed.ExpiresAt, credentials.UserID, platform)
	if err != nil {
		return fmt.Errorf("failed to update refreshed token: %w", err)
	}

	// Update local credentials
	credentials.AccessToken = refreshed.AccessToken
	credentials.RefreshToken = refreshed.RefreshToken
	credentials.ExpiresAt = refreshed.ExpiresAt

	log.Printf("🔄 Refreshed OAuth token for %s", platform)
	return nil
}

// handleJobResult handles the result of job processing
func (jp *JobProcessor) handleJobResult(ctx context.Context, job *PostJob, result *JobResult) error {
	if result.Success {
		// Update platform_posts table
		if err := jp.createPlatformPost(ctx, job, result); err != nil {
			return fmt.Errorf("failed to create platform post record: %w", err)
		}

		// Check if all platforms for this scheduled post are complete
		if err := jp.checkScheduledPostCompletion(ctx, job.ScheduledPostID); err != nil {
			log.Printf("⚠️  Error checking post completion: %v", err)
		}

	} else {
		// Handle failure
		if job.RetryCount < job.MaxRetries {
			// Schedule retry
			if err := jp.scheduleRetry(ctx, job); err != nil {
				log.Printf("⚠️  Error scheduling retry: %v", err)
			}
		} else {
			// Mark as permanently failed
			if err := jp.markPlatformPostFailed(ctx, job, result.Error); err != nil {
				log.Printf("⚠️  Error marking post as failed: %v", err)
			}
		}
	}

	return nil
}

// createPlatformPost creates a platform_posts record for successful posting
func (jp *JobProcessor) createPlatformPost(ctx context.Context, job *PostJob, result *JobResult) error {
	query := `
		INSERT INTO platform_posts (scheduled_post_id, social_account_id, platform, platform_post_id, optimized_content, optimized_media_urls, status, posted_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'posted', NOW())`

	mediaJSON, _ := json.Marshal(job.MediaURLs)

	_, err := jp.db.ExecContext(ctx, query,
		job.ScheduledPostID,
		job.SocialAccountID,
		job.Platform,
		result.PlatformPostID,
		job.Content,
		string(mediaJSON),
	)

	return err
}

// checkScheduledPostCompletion checks if all platforms for a scheduled post are complete
func (jp *JobProcessor) checkScheduledPostCompletion(ctx context.Context, scheduledPostID string) error {
	// Get expected platforms for this post
	var platformsJSON string
	err := jp.db.QueryRowContext(ctx, "SELECT platforms FROM scheduled_posts WHERE id = $1", scheduledPostID).Scan(&platformsJSON)
	if err != nil {
		return err
	}

	var expectedPlatforms []string
	if err := json.Unmarshal([]byte(platformsJSON), &expectedPlatforms); err != nil {
		return err
	}

	// Count completed platforms
	var completedCount int
	err = jp.db.QueryRowContext(ctx, 
		"SELECT COUNT(*) FROM platform_posts WHERE scheduled_post_id = $1 AND status IN ('posted', 'failed')", 
		scheduledPostID).Scan(&completedCount)
	if err != nil {
		return err
	}

	// If all platforms are complete, update scheduled post status
	if completedCount >= len(expectedPlatforms) {
		_, err = jp.db.ExecContext(ctx, "UPDATE scheduled_posts SET status = 'posted', posted_at = NOW() WHERE id = $1", scheduledPostID)
		if err != nil {
			return err
		}
		log.Printf("✅ Scheduled post %s completed on all platforms", scheduledPostID)
	}

	return nil
}

// scheduleRetry schedules a failed job for retry
func (jp *JobProcessor) scheduleRetry(ctx context.Context, job *PostJob) error {
	job.RetryCount++
	
	// Exponential backoff: 1m, 5m, 15m
	retryDelays := []time.Duration{1 * time.Minute, 5 * time.Minute, 15 * time.Minute}
	delay := retryDelays[0]
	if job.RetryCount-1 < len(retryDelays) {
		delay = retryDelays[job.RetryCount-1]
	}

	retryAt := time.Now().Add(delay)

	// Serialize updated job
	jobData, err := json.Marshal(job)
	if err != nil {
		return err
	}

	// Schedule retry using Redis sorted set
	score := float64(retryAt.Unix())
	if err := jp.redis.ZAdd(ctx, "queue:retries", &redis.Z{Score: score, Member: jobData}).Err(); err != nil {
		return err
	}

	log.Printf("🔄 Scheduled retry %d/%d for job %s at %v", job.RetryCount, job.MaxRetries, job.ID, retryAt)
	return nil
}

// markPlatformPostFailed marks a platform post as permanently failed
func (jp *JobProcessor) markPlatformPostFailed(ctx context.Context, job *PostJob, errorMsg string) error {
	query := `
		INSERT INTO platform_posts (scheduled_post_id, social_account_id, platform, status, error_message, retry_count)
		VALUES ($1, $2, $3, 'failed', $4, $5)
		ON CONFLICT (scheduled_post_id, social_account_id) 
		DO UPDATE SET status = 'failed', error_message = $4, retry_count = $5`

	_, err := jp.db.ExecContext(ctx, query, job.ScheduledPostID, job.SocialAccountID, job.Platform, errorMsg, job.RetryCount)
	if err != nil {
		return err
	}

	log.Printf("❌ Permanently failed job %s after %d retries: %s", job.ID, job.RetryCount, errorMsg)
	return nil
}

// retryHandler processes jobs scheduled for retry
func (jp *JobProcessor) retryHandler(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Println("🔄 Retry handler started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := jp.processRetries(ctx); err != nil {
				log.Printf("⚠️  Retry handler error: %v", err)
			}
		}
	}
}

// processRetries moves retry jobs back to the processing queue when their time arrives
func (jp *JobProcessor) processRetries(ctx context.Context) error {
	now := time.Now().Unix()

	// Get jobs ready for retry
	results, err := jp.redis.ZRangeByScore(ctx, "queue:retries", &redis.ZRangeBy{
		Min: "0",
		Max: strconv.FormatInt(now, 10),
	}).Result()

	if err != nil {
		return err
	}

	for _, jobData := range results {
		// Move back to processing queue
		if err := jp.redis.LPush(ctx, "queue:processing", jobData).Err(); err != nil {
			log.Printf("⚠️  Error moving retry job to processing queue: %v", err)
			continue
		}

		// Remove from retry queue
		if err := jp.redis.ZRem(ctx, "queue:retries", jobData).Err(); err != nil {
			log.Printf("⚠️  Error removing job from retry queue: %v", err)
		}
	}

	if len(results) > 0 {
		log.Printf("🔄 Moved %d retry jobs back to processing queue", len(results))
	}

	return nil
}

// updatePostStatus updates the status of a scheduled post
func (jp *JobProcessor) updatePostStatus(ctx context.Context, postID, status string) error {
	query := "UPDATE scheduled_posts SET status = $1, updated_at = NOW() WHERE id = $2"
	_, err := jp.db.ExecContext(ctx, query, status, postID)
	return err
}
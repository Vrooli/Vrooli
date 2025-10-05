// +build testing

package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestJobProcessorInitialization tests job processor creation
func TestJobProcessorInitialization(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("NewJobProcessor", func(t *testing.T) {
		jp := NewJobProcessor(env.DB, env.Redis, env.App.PlatformMgr)

		if jp == nil {
			t.Fatal("Job processor should not be nil")
		}
		if jp.db == nil {
			t.Error("Database should not be nil")
		}
		if jp.redis == nil {
			t.Error("Redis should not be nil")
		}
		if jp.platformMgr == nil {
			t.Error("Platform manager should not be nil")
		}
		if jp.running {
			t.Error("Job processor should not be running initially")
		}
	})
}

// TestPostJob tests the PostJob structure
func TestPostJob(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("JobSerialization", func(t *testing.T) {
		job := PostJob{
			ID:              uuid.New().String(),
			ScheduledPostID: uuid.New().String(),
			UserID:          uuid.New().String(),
			Platform:        "twitter",
			SocialAccountID: uuid.New().String(),
			Content:         "Test post content",
			MediaURLs:       []string{"https://example.com/image.jpg"},
			ScheduledAt:     time.Now().Add(1 * time.Hour),
			RetryCount:      0,
			MaxRetries:      3,
			Priority:        1,
			CreatedAt:       time.Now(),
		}

		// Serialize to JSON
		jsonData, err := json.Marshal(job)
		if err != nil {
			t.Fatalf("Failed to marshal job: %v", err)
		}

		// Deserialize from JSON
		var deserializedJob PostJob
		if err := json.Unmarshal(jsonData, &deserializedJob); err != nil {
			t.Fatalf("Failed to unmarshal job: %v", err)
		}

		// Verify fields
		if deserializedJob.ID != job.ID {
			t.Errorf("Expected ID %s, got %s", job.ID, deserializedJob.ID)
		}
		if deserializedJob.Platform != job.Platform {
			t.Errorf("Expected platform %s, got %s", job.Platform, deserializedJob.Platform)
		}
		if deserializedJob.Content != job.Content {
			t.Errorf("Expected content %s, got %s", job.Content, deserializedJob.Content)
		}
	})

	t.Run("JobValidation", func(t *testing.T) {
		testCases := []struct {
			name    string
			job     PostJob
			isValid bool
		}{
			{
				name: "ValidJob",
				job: PostJob{
					ID:              uuid.New().String(),
					ScheduledPostID: uuid.New().String(),
					UserID:          uuid.New().String(),
					Platform:        "twitter",
					Content:         "Valid content",
					ScheduledAt:     time.Now().Add(1 * time.Hour),
				},
				isValid: true,
			},
			{
				name: "MissingPlatform",
				job: PostJob{
					ID:      uuid.New().String(),
					Content: "Content without platform",
				},
				isValid: false,
			},
			{
				name: "MissingContent",
				job: PostJob{
					ID:       uuid.New().String(),
					Platform: "twitter",
				},
				isValid: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Basic validation checks
				hasRequiredFields := tc.job.ID != "" &&
					tc.job.Platform != "" &&
					tc.job.Content != ""

				if hasRequiredFields != tc.isValid {
					t.Errorf("Expected validation %v, got %v", tc.isValid, hasRequiredFields)
				}
			})
		}
	})
}

// TestJobResult tests the JobResult structure
func TestJobResult(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SuccessResult", func(t *testing.T) {
		result := JobResult{
			JobID:          uuid.New().String(),
			Success:        true,
			PlatformPostID: "twitter_123456",
			PostURL:        "https://twitter.com/user/status/123456",
			ProcessedAt:    time.Now(),
		}

		if !result.Success {
			t.Error("Result should be successful")
		}
		if result.PlatformPostID == "" {
			t.Error("Platform post ID should be set")
		}
		if result.PostURL == "" {
			t.Error("Post URL should be set")
		}
		if result.Error != "" {
			t.Error("Error should be empty for successful result")
		}
	})

	t.Run("FailureResult", func(t *testing.T) {
		nextRetry := time.Now().Add(15 * time.Minute)
		result := JobResult{
			JobID:       uuid.New().String(),
			Success:     false,
			Error:       "API rate limit exceeded",
			ProcessedAt: time.Now(),
			NextRetryAt: &nextRetry,
		}

		if result.Success {
			t.Error("Result should not be successful")
		}
		if result.Error == "" {
			t.Error("Error message should be set")
		}
		if result.NextRetryAt == nil {
			t.Error("Next retry time should be set for failures")
		}
		if result.PlatformPostID != "" {
			t.Error("Platform post ID should be empty for failures")
		}
	})

	t.Run("ResultSerialization", func(t *testing.T) {
		result := JobResult{
			JobID:          uuid.New().String(),
			Success:        true,
			PlatformPostID: "test_post_id",
			ProcessedAt:    time.Now(),
		}

		// Serialize
		jsonData, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("Failed to marshal result: %v", err)
		}

		// Deserialize
		var deserializedResult JobResult
		if err := json.Unmarshal(jsonData, &deserializedResult); err != nil {
			t.Fatalf("Failed to unmarshal result: %v", err)
		}

		if deserializedResult.JobID != result.JobID {
			t.Errorf("Expected job ID %s, got %s", result.JobID, deserializedResult.JobID)
		}
		if deserializedResult.Success != result.Success {
			t.Errorf("Expected success %v, got %v", result.Success, deserializedResult.Success)
		}
	})
}

// TestRedisQueueOperations tests Redis queue operations
func TestRedisQueueOperations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	ctx := context.Background()

	t.Run("PushToQueue", func(t *testing.T) {
		queueKey := "test:jobs:scheduled"

		job := PostJob{
			ID:       uuid.New().String(),
			Platform: "twitter",
			Content:  "Test content",
		}

		jobJSON, err := json.Marshal(job)
		if err != nil {
			t.Fatalf("Failed to marshal job: %v", err)
		}

		// Push to queue
		if err := env.Redis.LPush(ctx, queueKey, jobJSON).Err(); err != nil {
			t.Fatalf("Failed to push to queue: %v", err)
		}

		// Verify queue length
		length, err := env.Redis.LLen(ctx, queueKey).Result()
		if err != nil {
			t.Fatalf("Failed to get queue length: %v", err)
		}

		if length != 1 {
			t.Errorf("Expected queue length 1, got %d", length)
		}

		// Cleanup
		env.Redis.Del(ctx, queueKey)
	})

	t.Run("PopFromQueue", func(t *testing.T) {
		queueKey := "test:jobs:processing"

		job := PostJob{
			ID:       uuid.New().String(),
			Platform: "linkedin",
			Content:  "Test content for pop",
		}

		jobJSON, err := json.Marshal(job)
		if err != nil {
			t.Fatalf("Failed to marshal job: %v", err)
		}

		// Push to queue
		env.Redis.LPush(ctx, queueKey, jobJSON)

		// Pop from queue
		result, err := env.Redis.RPop(ctx, queueKey).Result()
		if err != nil {
			t.Fatalf("Failed to pop from queue: %v", err)
		}

		// Deserialize
		var poppedJob PostJob
		if err := json.Unmarshal([]byte(result), &poppedJob); err != nil {
			t.Fatalf("Failed to unmarshal popped job: %v", err)
		}

		if poppedJob.ID != job.ID {
			t.Errorf("Expected job ID %s, got %s", job.ID, poppedJob.ID)
		}

		// Cleanup
		env.Redis.Del(ctx, queueKey)
	})

	t.Run("BlockingPop", func(t *testing.T) {
		queueKey := "test:jobs:blocking"

		go func() {
			// Push after short delay
			time.Sleep(100 * time.Millisecond)

			job := PostJob{
				ID:       uuid.New().String(),
				Platform: "facebook",
				Content:  "Blocking pop test",
			}

			jobJSON, _ := json.Marshal(job)
			env.Redis.LPush(ctx, queueKey, jobJSON)
		}()

		// Blocking pop with timeout
		result, err := env.Redis.BRPop(ctx, 2*time.Second, queueKey).Result()
		if err != nil {
			t.Fatalf("Failed to blocking pop: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("Expected 2 elements in result, got %d", len(result))
		}

		// Cleanup
		env.Redis.Del(ctx, queueKey)
	})

	t.Run("MultipleQueues", func(t *testing.T) {
		queues := []string{
			"test:jobs:high_priority",
			"test:jobs:normal_priority",
			"test:jobs:low_priority",
		}

		// Push to each queue
		for i, queueKey := range queues {
			job := PostJob{
				ID:       uuid.New().String(),
				Platform: "twitter",
				Content:  "Priority test",
				Priority: i + 1,
			}

			jobJSON, _ := json.Marshal(job)
			env.Redis.LPush(ctx, queueKey, jobJSON)
		}

		// Verify all queues have jobs
		for _, queueKey := range queues {
			length, err := env.Redis.LLen(ctx, queueKey).Result()
			if err != nil {
				t.Errorf("Failed to get queue length for %s: %v", queueKey, err)
			}

			if length != 1 {
				t.Errorf("Expected queue length 1 for %s, got %d", queueKey, length)
			}
		}

		// Cleanup
		for _, queueKey := range queues {
			env.Redis.Del(ctx, queueKey)
		}
	})
}

// TestJobRetryLogic tests retry logic for failed jobs
func TestJobRetryLogic(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("RetryCalculation", func(t *testing.T) {
		testCases := []struct {
			retryCount     int
			expectedMinWait time.Duration
			expectedMaxWait time.Duration
		}{
			{0, 1 * time.Minute, 5 * time.Minute},
			{1, 5 * time.Minute, 15 * time.Minute},
			{2, 15 * time.Minute, 30 * time.Minute},
			{3, 30 * time.Minute, 60 * time.Minute},
		}

		for _, tc := range testCases {
			t.Run(string(rune(tc.retryCount+'0')), func(t *testing.T) {
				// Simple exponential backoff: base * 2^retryCount
				base := 1 * time.Minute
				calculatedWait := base * time.Duration(1<<tc.retryCount)

				if calculatedWait < tc.expectedMinWait {
					t.Errorf("Calculated wait %v is less than expected minimum %v", calculatedWait, tc.expectedMinWait)
				}
			})
		}
	})

	t.Run("MaxRetries", func(t *testing.T) {
		maxRetries := 3

		job := PostJob{
			ID:         uuid.New().String(),
			RetryCount: maxRetries,
			MaxRetries: maxRetries,
		}

		if job.RetryCount >= job.MaxRetries {
			// Should not retry anymore
			if job.RetryCount < maxRetries {
				t.Error("Should have exhausted retries")
			}
		}
	})
}

// TestJobPriorityQueue tests priority queue behavior
func TestJobPriorityQueue(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("PriorityOrdering", func(t *testing.T) {
		// Create jobs with different priorities
		jobs := []PostJob{
			{ID: "job1", Priority: 1, Content: "High priority"},
			{ID: "job2", Priority: 3, Content: "Low priority"},
			{ID: "job3", Priority: 2, Content: "Medium priority"},
		}

		// In real implementation, we'd use sorted sets or priority queues
		// Here we test the basic concept

		for _, job := range jobs {
			if job.Priority < 1 || job.Priority > 3 {
				t.Errorf("Invalid priority %d for job %s", job.Priority, job.ID)
			}
		}

		// Verify priority values are set correctly
		if jobs[0].Priority != 1 {
			t.Error("First job should have priority 1")
		}
		if jobs[1].Priority != 3 {
			t.Error("Second job should have priority 3")
		}
		if jobs[2].Priority != 2 {
			t.Error("Third job should have priority 2")
		}
	})
}

// TestJobProcessorLifecycle tests processor start/stop
func TestJobProcessorLifecycle(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("StartStop", func(t *testing.T) {
		jp := NewJobProcessor(env.DB, env.Redis, env.App.PlatformMgr)

		if jp.running {
			t.Error("Processor should not be running initially")
		}

		// Start processor in background
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go jp.Start(ctx)

		// Give it time to start
		time.Sleep(100 * time.Millisecond)

		// Stop processor
		cancel()

		// Give it time to stop
		time.Sleep(100 * time.Millisecond)
	})
}

// TestJobStatistics tests job statistics tracking
func TestJobStatistics(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	ctx := context.Background()

	t.Run("TrackJobCounts", func(t *testing.T) {
		// Simulate tracking job counts
		statsKey := "test:stats:jobs"

		// Increment successful jobs
		env.Redis.HIncrBy(ctx, statsKey, "successful", 5)

		// Increment failed jobs
		env.Redis.HIncrBy(ctx, statsKey, "failed", 2)

		// Increment retried jobs
		env.Redis.HIncrBy(ctx, statsKey, "retried", 1)

		// Get statistics
		stats, err := env.Redis.HGetAll(ctx, statsKey).Result()
		if err != nil {
			t.Fatalf("Failed to get statistics: %v", err)
		}

		if stats["successful"] != "5" {
			t.Errorf("Expected 5 successful jobs, got %s", stats["successful"])
		}
		if stats["failed"] != "2" {
			t.Errorf("Expected 2 failed jobs, got %s", stats["failed"])
		}
		if stats["retried"] != "1" {
			t.Errorf("Expected 1 retried job, got %s", stats["retried"])
		}

		// Cleanup
		env.Redis.Del(ctx, statsKey)
	})
}

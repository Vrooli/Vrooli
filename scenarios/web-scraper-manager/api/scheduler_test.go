// +build testing

package main

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCalculateNextRun(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		schedule *string
		expected time.Duration
	}{
		{
			name:     "NilSchedule",
			schedule: nil,
			expected: 1 * time.Hour,
		},
		{
			name:     "EmptySchedule",
			schedule: strPtr(""),
			expected: 1 * time.Hour,
		},
		{
			name:     "DurationFormat5m",
			schedule: strPtr("5m"),
			expected: 5 * time.Minute,
		},
		{
			name:     "DurationFormat1h",
			schedule: strPtr("1h"),
			expected: 1 * time.Hour,
		},
		{
			name:     "DurationFormat30s",
			schedule: strPtr("30s"),
			expected: 30 * time.Second,
		},
		{
			name:     "CronEvery5Minutes",
			schedule: strPtr("*/5 * * * *"),
			expected: 5 * time.Minute,
		},
		{
			name:     "CronEvery15Minutes",
			schedule: strPtr("*/15 * * * *"),
			expected: 15 * time.Minute,
		},
		{
			name:     "CronEvery2Hours",
			schedule: strPtr("0 */2 * * *"),
			expected: 2 * time.Hour,
		},
		{
			name:     "CronEvery3Hours",
			schedule: strPtr("0 */3 * * *"),
			expected: 3 * time.Hour,
		},
		{
			name:     "InvalidCron",
			schedule: strPtr("invalid cron"),
			expected: 1 * time.Hour,
		},
		{
			name:     "ZeroDuration",
			schedule: strPtr("0s"),
			expected: 1 * time.Hour,
		},
		{
			name:     "NegativeDuration",
			schedule: strPtr("-5m"),
			expected: 1 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextRun := calculateNextRun(tt.schedule, baseTime)
			actualDuration := nextRun.Sub(baseTime)

			if actualDuration != tt.expected {
				t.Errorf("Expected duration %v, got %v", tt.expected, actualDuration)
			}
		})
	}
}

func TestProcessScheduledAgents(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	t.Run("NoAgents", func(t *testing.T) {
		// Clean up any existing agents
		db.Exec("DELETE FROM scraping_agents")

		// Should not error with no agents
		processScheduledAgents()
	})

	t.Run("ProcessSingleAgent", func(t *testing.T) {
		// Clean up first
		db.Exec("DELETE FROM scraping_agents")
		db.Exec("DELETE FROM scraping_results")

		// Create a test agent
		agentID := uuid.New().String()
		schedule := "5m"
		config := map[string]interface{}{"test": "value"}
		configJSON, _ := json.Marshal(config)

		_, err := db.Exec(`
			INSERT INTO scraping_agents (id, name, platform, agent_type, configuration,
			                           schedule_cron, enabled, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		`, agentID, "Test Agent", "huginn", "WebsiteAgent", configJSON, schedule, true)

		if err != nil {
			t.Fatalf("Failed to create test agent: %v", err)
		}

		// Process scheduled agents
		processScheduledAgents()

		// Verify result was created
		var resultCount int
		err = db.QueryRow("SELECT COUNT(*) FROM scraping_results WHERE agent_id = $1", agentID).Scan(&resultCount)
		if err != nil {
			t.Fatalf("Failed to query results: %v", err)
		}

		if resultCount == 0 {
			t.Error("Expected at least one result to be created")
		}

		// Verify agent times were updated
		var lastRun, nextRun sql.NullTime
		err = db.QueryRow("SELECT last_run, next_run FROM scraping_agents WHERE id = $1", agentID).Scan(&lastRun, &nextRun)
		if err != nil {
			t.Fatalf("Failed to query agent: %v", err)
		}

		if !lastRun.Valid {
			t.Error("Expected last_run to be set")
		}

		if !nextRun.Valid {
			t.Error("Expected next_run to be set")
		}
	})

	t.Run("ProcessMultipleAgents", func(t *testing.T) {
		// Clean up first
		db.Exec("DELETE FROM scraping_agents")
		db.Exec("DELETE FROM scraping_results")

		// Create multiple test agents
		for i := 0; i < 3; i++ {
			agentID := uuid.New().String()
			schedule := "10m"
			config := map[string]interface{}{"index": i}
			configJSON, _ := json.Marshal(config)

			_, err := db.Exec(`
				INSERT INTO scraping_agents (id, name, platform, agent_type, configuration,
				                           schedule_cron, enabled, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
			`, agentID, "Test Agent "+string(rune(i)), "huginn", "WebsiteAgent", configJSON, schedule, true)

			if err != nil {
				t.Fatalf("Failed to create test agent: %v", err)
			}
		}

		// Process scheduled agents
		processScheduledAgents()

		// Verify results were created
		var resultCount int
		err := db.QueryRow("SELECT COUNT(*) FROM scraping_results").Scan(&resultCount)
		if err != nil {
			t.Fatalf("Failed to query results: %v", err)
		}

		if resultCount < 3 {
			t.Errorf("Expected at least 3 results, got %d", resultCount)
		}
	})

	t.Run("SkipDisabledAgents", func(t *testing.T) {
		// Clean up first
		db.Exec("DELETE FROM scraping_agents")
		db.Exec("DELETE FROM scraping_results")

		// Create disabled agent
		agentID := uuid.New().String()
		schedule := "5m"
		config := map[string]interface{}{"test": "value"}
		configJSON, _ := json.Marshal(config)

		_, err := db.Exec(`
			INSERT INTO scraping_agents (id, name, platform, agent_type, configuration,
			                           schedule_cron, enabled, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		`, agentID, "Disabled Agent", "huginn", "WebsiteAgent", configJSON, schedule, false)

		if err != nil {
			t.Fatalf("Failed to create test agent: %v", err)
		}

		// Process scheduled agents
		processScheduledAgents()

		// Verify no result was created
		var resultCount int
		err = db.QueryRow("SELECT COUNT(*) FROM scraping_results WHERE agent_id = $1", agentID).Scan(&resultCount)
		if err != nil {
			t.Fatalf("Failed to query results: %v", err)
		}

		if resultCount > 0 {
			t.Error("Expected no results for disabled agent")
		}
	})

	t.Run("SkipFutureAgents", func(t *testing.T) {
		// Clean up first
		db.Exec("DELETE FROM scraping_agents")
		db.Exec("DELETE FROM scraping_results")

		// Create agent with future next_run
		agentID := uuid.New().String()
		schedule := "5m"
		config := map[string]interface{}{"test": "value"}
		configJSON, _ := json.Marshal(config)
		futureTime := time.Now().Add(1 * time.Hour)

		_, err := db.Exec(`
			INSERT INTO scraping_agents (id, name, platform, agent_type, configuration,
			                           schedule_cron, enabled, next_run, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		`, agentID, "Future Agent", "huginn", "WebsiteAgent", configJSON, schedule, true, futureTime)

		if err != nil {
			t.Fatalf("Failed to create test agent: %v", err)
		}

		// Process scheduled agents
		processScheduledAgents()

		// Verify no result was created
		var resultCount int
		err = db.QueryRow("SELECT COUNT(*) FROM scraping_results WHERE agent_id = $1", agentID).Scan(&resultCount)
		if err != nil {
			t.Fatalf("Failed to query results: %v", err)
		}

		if resultCount > 0 {
			t.Error("Expected no results for future-scheduled agent")
		}
	})
}

func TestExecuteScheduledAgent(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	t.Run("ExecuteAgentSuccess", func(t *testing.T) {
		// Clean up first
		db.Exec("DELETE FROM scraping_agents")
		db.Exec("DELETE FROM scraping_results")

		schedule := "15m"
		agent := ScrapingAgent{
			ID:            uuid.New().String(),
			Name:          "Test Execute Agent",
			Platform:      "browserless",
			AgentType:     "screenshot",
			Configuration: map[string]interface{}{"url": "https://example.com"},
			ScheduleCron:  &schedule,
			Enabled:       true,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		// Create agent in database
		configJSON, _ := json.Marshal(agent.Configuration)
		_, err := db.Exec(`
			INSERT INTO scraping_agents (id, name, platform, agent_type, configuration,
			                           schedule_cron, enabled, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, agent.ID, agent.Name, agent.Platform, agent.AgentType, configJSON,
			agent.ScheduleCron, agent.Enabled, agent.CreatedAt, agent.UpdatedAt)

		if err != nil {
			t.Fatalf("Failed to create agent: %v", err)
		}

		// Execute agent
		executeScheduledAgent(agent)

		// Verify result was created
		var resultCount int
		err = db.QueryRow("SELECT COUNT(*) FROM scraping_results WHERE agent_id = $1", agent.ID).Scan(&resultCount)
		if err != nil {
			t.Fatalf("Failed to query results: %v", err)
		}

		if resultCount == 0 {
			t.Error("Expected result to be created")
		}

		// Verify result data
		var status, dataJSON string
		var extractedCount int
		err = db.QueryRow(`
			SELECT status, data, extracted_count
			FROM scraping_results
			WHERE agent_id = $1
		`, agent.ID).Scan(&status, &dataJSON, &extractedCount)

		if err != nil {
			t.Fatalf("Failed to query result: %v", err)
		}

		if status != "success" {
			t.Errorf("Expected status 'success', got '%s'", status)
		}

		if extractedCount <= 0 {
			t.Errorf("Expected positive extracted count, got %d", extractedCount)
		}

		// Verify data contains expected fields
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
			t.Fatalf("Failed to parse result data: %v", err)
		}

		if data["agent_name"] != agent.Name {
			t.Errorf("Expected agent_name '%s', got '%v'", agent.Name, data["agent_name"])
		}

		if data["platform"] != agent.Platform {
			t.Errorf("Expected platform '%s', got '%v'", agent.Platform, data["platform"])
		}
	})

	t.Run("ExecuteAgentWithConfiguration", func(t *testing.T) {
		// Clean up first
		db.Exec("DELETE FROM scraping_agents")
		db.Exec("DELETE FROM scraping_results")

		schedule := "30m"
		agent := ScrapingAgent{
			ID:       uuid.New().String(),
			Name:     "Config Test Agent",
			Platform: "agent-s2",
			AgentType: "navigate",
			Configuration: map[string]interface{}{
				"url":      "https://test.com",
				"selector": "div.content",
				"timeout":  30,
			},
			ScheduleCron: &schedule,
			Enabled:      true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// Create agent in database
		configJSON, _ := json.Marshal(agent.Configuration)
		_, err := db.Exec(`
			INSERT INTO scraping_agents (id, name, platform, agent_type, configuration,
			                           schedule_cron, enabled, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, agent.ID, agent.Name, agent.Platform, agent.AgentType, configJSON,
			agent.ScheduleCron, agent.Enabled, agent.CreatedAt, agent.UpdatedAt)

		if err != nil {
			t.Fatalf("Failed to create agent: %v", err)
		}

		// Execute agent
		executeScheduledAgent(agent)

		// Verify result data includes configuration
		var dataJSON string
		err = db.QueryRow(`
			SELECT data FROM scraping_results WHERE agent_id = $1
		`, agent.ID).Scan(&dataJSON)

		if err != nil {
			t.Fatalf("Failed to query result: %v", err)
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
			t.Fatalf("Failed to parse result data: %v", err)
		}

		config, ok := data["configuration"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected configuration in result data")
		}

		if config["url"] != "https://test.com" {
			t.Errorf("Expected URL in configuration, got %v", config)
		}
	})
}

func TestStartAgentScheduler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	t.Run("SchedulerStartup", func(t *testing.T) {
		// This just verifies the scheduler goroutine starts without panic
		startAgentScheduler()

		// Give it a moment to start
		time.Sleep(100 * time.Millisecond)

		// The scheduler runs in background, so we can't easily test its execution
		// but we can verify it doesn't panic on startup
	})
}

func TestSchedulerConstants(t *testing.T) {
	t.Run("SchedulerConstants", func(t *testing.T) {
		if schedulerInitialDelay != 15*time.Second {
			t.Errorf("Expected initial delay 15s, got %v", schedulerInitialDelay)
		}

		if schedulerInterval != 5*time.Minute {
			t.Errorf("Expected interval 5m, got %v", schedulerInterval)
		}
	})
}

func TestSchedulerWithNullDatabase(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ProcessScheduledAgentsNullDB", func(t *testing.T) {
		originalDB := db
		db = nil
		defer func() { db = originalDB }()

		// Should not panic with null db
		processScheduledAgents()
	})
}

func TestExecuteScheduledAgentEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	t.Run("ExecuteAgentEmptyConfiguration", func(t *testing.T) {
		// Clean up first
		db.Exec("DELETE FROM scraping_agents")
		db.Exec("DELETE FROM scraping_results")

		agent := ScrapingAgent{
			ID:            uuid.New().String(),
			Name:          "Empty Config Agent",
			Platform:      "huginn",
			AgentType:     "WebsiteAgent",
			Configuration: map[string]interface{}{},
			Enabled:       true,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		// Create agent in database
		configJSON, _ := json.Marshal(agent.Configuration)
		_, err := db.Exec(`
			INSERT INTO scraping_agents (id, name, platform, agent_type, configuration,
			                           enabled, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, agent.ID, agent.Name, agent.Platform, agent.AgentType, configJSON,
			agent.Enabled, agent.CreatedAt, agent.UpdatedAt)

		if err != nil {
			t.Fatalf("Failed to create agent: %v", err)
		}

		// Execute agent (should not panic)
		executeScheduledAgent(agent)

		// Verify result was still created
		var resultCount int
		err = db.QueryRow("SELECT COUNT(*) FROM scraping_results WHERE agent_id = $1", agent.ID).Scan(&resultCount)
		if err != nil {
			t.Fatalf("Failed to query results: %v", err)
		}

		if resultCount == 0 {
			t.Error("Expected result to be created even with empty config")
		}
	})

	t.Run("ExecuteAgentNilSchedule", func(t *testing.T) {
		// Clean up first
		db.Exec("DELETE FROM scraping_agents")
		db.Exec("DELETE FROM scraping_results")

		agent := ScrapingAgent{
			ID:            uuid.New().String(),
			Name:          "No Schedule Agent",
			Platform:      "browserless",
			AgentType:     "content",
			Configuration: map[string]interface{}{"url": "https://example.com"},
			ScheduleCron:  nil, // No schedule
			Enabled:       true,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		// Create agent in database
		configJSON, _ := json.Marshal(agent.Configuration)
		_, err := db.Exec(`
			INSERT INTO scraping_agents (id, name, platform, agent_type, configuration,
			                           enabled, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, agent.ID, agent.Name, agent.Platform, agent.AgentType, configJSON,
			agent.Enabled, agent.CreatedAt, agent.UpdatedAt)

		if err != nil {
			t.Fatalf("Failed to create agent: %v", err)
		}

		// Execute agent
		executeScheduledAgent(agent)

		// Verify next_run was set to default (1 hour)
		var nextRun time.Time
		err = db.QueryRow("SELECT next_run FROM scraping_agents WHERE id = $1", agent.ID).Scan(&nextRun)
		if err != nil {
			t.Fatalf("Failed to query agent: %v", err)
		}

		expectedNextRun := time.Now().Add(1 * time.Hour)
		if nextRun.Sub(expectedNextRun).Abs() > 5*time.Second {
			t.Errorf("Expected next_run around %v, got %v", expectedNextRun, nextRun)
		}
	})
}

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}

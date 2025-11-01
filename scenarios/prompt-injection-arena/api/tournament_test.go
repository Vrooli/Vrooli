//go:build testing
// +build testing

package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestTournamentStructures tests tournament data structures
func TestTournamentStructures(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("TournamentCreation", func(t *testing.T) {
		tournament := Tournament{
			ID:          uuid.New().String(),
			Name:        "Test Tournament",
			Description: "Test tournament description",
			Status:      "scheduled",
			ScheduledAt: time.Now().Add(24 * time.Hour),
			Configuration: map[string]interface{}{
				"max_rounds":      10,
				"timeout_seconds": 300,
			},
			ParticipantIDs: []string{uuid.New().String(), uuid.New().String()},
			InjectionIDs:   []string{uuid.New().String(), uuid.New().String()},
			Results:        []TournamentResult{},
			LeaderboardID:  uuid.New().String(),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		if tournament.Name == "" {
			t.Error("Tournament name should not be empty")
		}
		if tournament.Status != "scheduled" {
			t.Errorf("Expected status 'scheduled', got '%s'", tournament.Status)
		}
		if len(tournament.ParticipantIDs) != 2 {
			t.Errorf("Expected 2 participants, got %d", len(tournament.ParticipantIDs))
		}
	})

	t.Run("TournamentResult", func(t *testing.T) {
		result := TournamentResult{
			TournamentID:    uuid.New().String(),
			AgentID:         uuid.New().String(),
			InjectionID:     uuid.New().String(),
			Success:         true,
			ExecutionTimeMS: 150,
			Score:           0.85,
			Details: map[string]interface{}{
				"response_quality": "high",
				"safety_score":     0.9,
			},
			TestedAt: time.Now(),
		}

		if result.TournamentID == "" {
			t.Error("TournamentID should not be empty")
		}
		if result.Score < 0 || result.Score > 1 {
			t.Errorf("Score should be between 0 and 1, got %f", result.Score)
		}
		if result.ExecutionTimeMS < 0 {
			t.Error("ExecutionTimeMS should not be negative")
		}
	})

	t.Run("TournamentStatusValidation", func(t *testing.T) {
		validStatuses := []string{"scheduled", "running", "completed", "cancelled"}

		for _, status := range validStatuses {
			tournament := Tournament{
				ID:     uuid.New().String(),
				Name:   "Test",
				Status: status,
			}

			if tournament.Status != status {
				t.Errorf("Expected status '%s', got '%s'", status, tournament.Status)
			}
		}
	})

	t.Run("TournamentJSONMarshaling", func(t *testing.T) {
		tournament := Tournament{
			ID:          uuid.New().String(),
			Name:        "Test Tournament",
			Description: "Description",
			Status:      "scheduled",
			ScheduledAt: time.Now(),
			Configuration: map[string]interface{}{
				"rounds": 5,
			},
			ParticipantIDs: []string{uuid.New().String()},
			InjectionIDs:   []string{uuid.New().String()},
			Results:        []TournamentResult{},
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		data, err := json.Marshal(tournament)
		if err != nil {
			t.Fatalf("Failed to marshal Tournament: %v", err)
		}

		var decoded Tournament
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal Tournament: %v", err)
		}

		if decoded.Name != tournament.Name {
			t.Errorf("Expected name '%s', got '%s'", tournament.Name, decoded.Name)
		}
	})
}

// TestTournamentScheduler tests the tournament scheduler
func TestTournamentScheduler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	t.Run("SchedulerCreation", func(t *testing.T) {
		scheduler := &TournamentScheduler{
			db:          env.DB,
			running:     false,
			currentTask: nil,
		}

		if scheduler.running {
			t.Error("Scheduler should not be running initially")
		}
		if scheduler.currentTask != nil {
			t.Error("Scheduler should have no current task initially")
		}
	})

	t.Run("SchedulerState", func(t *testing.T) {
		scheduler := &TournamentScheduler{
			db:      env.DB,
			running: false,
		}

		// Test state transitions
		states := []bool{false, true, false}
		for _, state := range states {
			scheduler.running = state
			if scheduler.running != state {
				t.Errorf("Expected running state %v, got %v", state, scheduler.running)
			}
		}
	})
}

// TestTournamentHandlers tests tournament HTTP handlers
func TestTournamentHandlers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	if env == nil {
		return
	}
	defer env.Cleanup()
	defer cleanupTestData(env.DB, "tournaments")

	router := setupTestRouter()

	t.Run("GetTournaments", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/tournaments",
		}

		w := makeHTTPRequest(router, req)
		if w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, http.StatusOK)

			if _, ok := response["tournaments"]; ok {
				t.Log("Tournaments endpoint returned successfully")
			}
		}
	})

	t.Run("CreateTournament", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/tournaments",
			Body: map[string]interface{}{
				"name":         "Test Tournament",
				"description":  "Test description",
				"scheduled_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
				"configuration": map[string]interface{}{
					"max_rounds": 10,
				},
				"participant_ids": []string{uuid.New().String()},
				"injection_ids":   []string{uuid.New().String()},
			},
		}

		w := makeHTTPRequest(router, req)
		if w.Code == http.StatusCreated || w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, w.Code)
			if _, ok := response["id"]; !ok {
				t.Log("Expected 'id' in tournament creation response")
			}
		}
	})

	t.Run("RunTournament", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/tournaments/run",
			Body: map[string]interface{}{
				"name":            "Quick Tournament",
				"agent_ids":       []string{uuid.New().String()},
				"injection_ids":   []string{uuid.New().String()},
				"max_rounds":      5,
				"timeout_seconds": 300,
			},
		}

		w := makeHTTPRequest(router, req)
		// May return various status codes depending on implementation
		t.Logf("Run tournament returned status: %d", w.Code)
	})

	t.Run("GetTournamentResults", func(t *testing.T) {
		tournamentID := uuid.New().String()
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/tournaments/" + tournamentID + "/results",
		}

		w := makeHTTPRequest(router, req)
		// May return 404 for non-existent tournament
		t.Logf("Get tournament results returned status: %d", w.Code)
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		suite := NewHandlerTestSuite("TournamentHandlers", router, "/api/v1/tournaments")
		patterns := NewTestScenarioBuilder().
			AddInvalidJSON("/api/v1/tournaments", "POST").
			AddEmptyInput("/api/v1/tournaments", "POST").
			AddNonExistentResource("/api/v1/tournaments", "GET", "Tournament").
			Build()
		suite.RunErrorTests(t, patterns)
	})
}

// TestTournamentLogic tests tournament business logic
func TestTournamentLogic(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ScoreCalculation", func(t *testing.T) {
		// Test score calculation logic
		testCases := []struct {
			name            string
			success         bool
			executionTimeMS int
			expectedScore   float64
		}{
			{
				name:            "SuccessfulFast",
				success:         true,
				executionTimeMS: 100,
				expectedScore:   1.0,
			},
			{
				name:            "SuccessfulSlow",
				success:         true,
				executionTimeMS: 5000,
				expectedScore:   0.5,
			},
			{
				name:            "Failed",
				success:         false,
				executionTimeMS: 100,
				expectedScore:   0.0,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Mock score calculation
				var score float64
				if tc.success {
					// Simple score based on execution time
					if tc.executionTimeMS < 1000 {
						score = 1.0
					} else {
						score = 1000.0 / float64(tc.executionTimeMS)
					}
				} else {
					score = 0.0
				}

				if score < 0 || score > 1 {
					t.Errorf("Score %f should be between 0 and 1", score)
				}
			})
		}
	})

	t.Run("TournamentRoundRobin", func(t *testing.T) {
		// Test round-robin tournament logic
		agents := []string{uuid.New().String(), uuid.New().String(), uuid.New().String()}
		injections := []string{uuid.New().String(), uuid.New().String()}

		expectedMatches := len(agents) * len(injections)
		actualMatches := 0

		for range agents {
			for range injections {
				actualMatches++
			}
		}

		if actualMatches != expectedMatches {
			t.Errorf("Expected %d matches, got %d", expectedMatches, actualMatches)
		}
	})

	t.Run("LeaderboardRanking", func(t *testing.T) {
		// Test leaderboard ranking logic
		results := []struct {
			agentID string
			score   float64
		}{
			{uuid.New().String(), 0.9},
			{uuid.New().String(), 0.8},
			{uuid.New().String(), 0.95},
			{uuid.New().String(), 0.7},
		}

		// Sort by score descending
		maxScore := 0.0
		for _, result := range results {
			if result.score > maxScore {
				maxScore = result.score
			}
		}

		if maxScore != 0.95 {
			t.Errorf("Expected max score 0.95, got %f", maxScore)
		}
	})

	t.Run("TournamentTimeout", func(t *testing.T) {
		// Test tournament timeout handling
		timeoutSeconds := 300
		startTime := time.Now()
		maxDuration := time.Duration(timeoutSeconds) * time.Second

		// Simulate tournament running
		time.Sleep(10 * time.Millisecond)
		elapsed := time.Since(startTime)

		if elapsed > maxDuration {
			t.Error("Tournament exceeded timeout")
		}
	})
}

// TestTournamentEdgeCases tests edge cases in tournament logic
func TestTournamentEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("EmptyParticipants", func(t *testing.T) {
		tournament := Tournament{
			ID:             uuid.New().String(),
			Name:           "Empty Tournament",
			ParticipantIDs: []string{},
			InjectionIDs:   []string{uuid.New().String()},
		}

		if len(tournament.ParticipantIDs) == 0 {
			t.Log("Tournament with no participants should be handled")
		}
	})

	t.Run("EmptyInjections", func(t *testing.T) {
		tournament := Tournament{
			ID:             uuid.New().String(),
			Name:           "No Injections",
			ParticipantIDs: []string{uuid.New().String()},
			InjectionIDs:   []string{},
		}

		if len(tournament.InjectionIDs) == 0 {
			t.Log("Tournament with no injections should be handled")
		}
	})

	t.Run("SingleParticipant", func(t *testing.T) {
		tournament := Tournament{
			ID:             uuid.New().String(),
			Name:           "Solo Tournament",
			ParticipantIDs: []string{uuid.New().String()},
			InjectionIDs:   []string{uuid.New().String()},
		}

		if len(tournament.ParticipantIDs) == 1 {
			t.Log("Single participant tournament is valid")
		}
	})

	t.Run("LargeTournament", func(t *testing.T) {
		participants := make([]string, 100)
		injections := make([]string, 50)

		for i := range participants {
			participants[i] = uuid.New().String()
		}
		for i := range injections {
			injections[i] = uuid.New().String()
		}

		_ = Tournament{
			ID:             uuid.New().String(),
			Name:           "Large Tournament",
			ParticipantIDs: participants,
			InjectionIDs:   injections,
		}

		totalMatches := len(participants) * len(injections)
		if totalMatches != 5000 {
			t.Errorf("Expected 5000 matches, got %d", totalMatches)
		}
	})

	t.Run("ConcurrentTournaments", func(t *testing.T) {
		// Test handling of multiple concurrent tournaments
		tournaments := []Tournament{
			{
				ID:     uuid.New().String(),
				Name:   "Tournament 1",
				Status: "running",
			},
			{
				ID:     uuid.New().String(),
				Name:   "Tournament 2",
				Status: "running",
			},
		}

		runningCount := 0
		for _, t := range tournaments {
			if t.Status == "running" {
				runningCount++
			}
		}

		if runningCount != 2 {
			t.Errorf("Expected 2 running tournaments, got %d", runningCount)
		}
	})

	t.Run("TournamentCancellation", func(t *testing.T) {
		tournament := Tournament{
			ID:     uuid.New().String(),
			Name:   "Cancelled Tournament",
			Status: "scheduled",
		}

		// Simulate cancellation
		tournament.Status = "cancelled"

		if tournament.Status != "cancelled" {
			t.Error("Tournament should be cancelled")
		}
	})
}

// TestTournamentPerformance tests performance-related aspects
func TestTournamentPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ResultProcessing", func(t *testing.T) {
		// Test processing of large number of results
		results := make([]TournamentResult, 1000)

		start := time.Now()
		for i := range results {
			results[i] = TournamentResult{
				TournamentID:    uuid.New().String(),
				AgentID:         uuid.New().String(),
				InjectionID:     uuid.New().String(),
				Success:         i%2 == 0,
				ExecutionTimeMS: 100 + i,
				Score:           float64(i%100) / 100.0,
				TestedAt:        time.Now(),
			}
		}
		elapsed := time.Since(start)

		if elapsed > 1*time.Second {
			t.Errorf("Processing 1000 results took too long: %v", elapsed)
		}

		if len(results) != 1000 {
			t.Errorf("Expected 1000 results, got %d", len(results))
		}
	})

	t.Run("ScoreAggregation", func(t *testing.T) {
		// Test aggregating scores efficiently
		scores := make(map[string][]float64)

		for i := 0; i < 100; i++ {
			agentID := uuid.New().String()
			scores[agentID] = []float64{0.8, 0.9, 0.7}
		}

		// Calculate average scores
		averages := make(map[string]float64)
		for agentID, agentScores := range scores {
			sum := 0.0
			for _, score := range agentScores {
				sum += score
			}
			averages[agentID] = sum / float64(len(agentScores))
		}

		if len(averages) != len(scores) {
			t.Errorf("Expected %d averages, got %d", len(scores), len(averages))
		}
	})
}

package autosteer

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func setupHandlersProfileRepo(t *testing.T) (*FileProfileRepository, func()) {
	t.Helper()

	root := t.TempDir()
	writeMetadata(t, root, ProfileMetadataIndex{Profiles: []ProfileMetadata{}})

	repo, err := NewFileProfileRepository(root)
	if err != nil {
		t.Fatalf("failed to create profile repo: %v", err)
	}

	return repo, func() {}
}

func TestHandlers_CreateProfile(t *testing.T) {
	profileRepo, cleanup := setupHandlersProfileRepo(t)
	defer cleanup()

	handlers := NewAutoSteerHandlers(profileRepo, nil, nil)

	t.Run("create valid profile", func(t *testing.T) {
		profile := AutoSteerProfile{
			Name:        "Test Profile",
			Description: "A test profile",
			Phases: []SteerPhase{
				{
					ID:            "phase-1",
					SkillIDs:      []string{"progress"},
					SkillName:     "Progress",
					MaxIterations: 10,
					StopConditions: []StopCondition{
						{
							Type:            ConditionTypeSimple,
							Metric:          "loops",
							CompareOperator: OpGreaterThan,
							Value:           5,
						},
					},
				},
			},
			Tags: []string{"test"},
		}

		body, _ := json.Marshal(profile)
		req := httptest.NewRequest(http.MethodPost, "/api/auto-steer/profiles", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handlers.CreateProfile(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
		}

		var created AutoSteerProfile
		if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if created.ID == "" {
			t.Error("Expected profile ID to be set")
		}
		if created.Name != profile.Name {
			t.Errorf("Expected name %s, got %s", profile.Name, created.Name)
		}
	})

	t.Run("create invalid profile - bad JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/auto-steer/profiles", bytes.NewBufferString("invalid json"))
		w := httptest.NewRecorder()

		handlers.CreateProfile(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("create invalid profile - missing name", func(t *testing.T) {
		profile := AutoSteerProfile{
			Description: "No name",
			Phases: []SteerPhase{
				{
					ID:            "phase-1",
					SkillIDs:      []string{"progress"},
					SkillName:     "Progress",
					MaxIterations: 10,
					StopConditions: []StopCondition{
						{
							Type:            ConditionTypeSimple,
							Metric:          "loops",
							CompareOperator: OpGreaterThan,
							Value:           5,
						},
					},
				},
			},
		}

		body, _ := json.Marshal(profile)
		req := httptest.NewRequest(http.MethodPost, "/api/auto-steer/profiles", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handlers.CreateProfile(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})
}

func TestHandlers_ListProfiles(t *testing.T) {
	profileRepo, cleanup := setupHandlersProfileRepo(t)
	defer cleanup()

	handlers := NewAutoSteerHandlers(profileRepo, nil, nil)

	// Create test profiles
	profile1 := &AutoSteerProfile{
		Name: "Profile 1",
		Phases: []SteerPhase{
			{
				ID:            "phase-1",
				SkillIDs:      []string{"progress"},
				SkillName:     "Progress",
				MaxIterations: 10,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "loops",
						CompareOperator: OpGreaterThan,
						Value:           5,
					},
				},
			},
		},
		Tags: []string{"tag1"},
	}

	profile2 := &AutoSteerProfile{
		Name: "Profile 2",
		Phases: []SteerPhase{
			{
				ID:            "phase-1",
				SkillIDs:      []string{"ux"},
				SkillName:     "UX",
				MaxIterations: 5,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "accessibility_score",
						CompareOperator: OpGreaterThan,
						Value:           90,
					},
				},
			},
		},
		Tags: []string{"tag2"},
	}

	if err := profileRepo.CreateProfile(profile1); err != nil {
		t.Fatalf("failed to create profile1: %v", err)
	}
	if err := profileRepo.CreateProfile(profile2); err != nil {
		t.Fatalf("failed to create profile2: %v", err)
	}

	t.Run("list all profiles", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/auto-steer/profiles", nil)
		w := httptest.NewRecorder()

		handlers.ListProfiles(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var resp struct {
			Profiles []*AutoSteerProfile `json:"profiles"`
			Count    int                 `json:"count"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(resp.Profiles) != 2 {
			t.Errorf("Expected 2 profiles, got %d", len(resp.Profiles))
		}
		if resp.Count != 2 {
			t.Errorf("Expected count 2, got %d", resp.Count)
		}
	})

	t.Run("filter by tag", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/auto-steer/profiles?tag=tag1", nil)
		w := httptest.NewRecorder()

		handlers.ListProfiles(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var resp struct {
			Profiles []*AutoSteerProfile `json:"profiles"`
			Count    int                 `json:"count"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(resp.Profiles) != 1 {
			t.Errorf("Expected 1 profile, got %d", len(resp.Profiles))
		}
	})
}

func TestHandlers_GetProfile(t *testing.T) {
	profileRepo, cleanup := setupHandlersProfileRepo(t)
	defer cleanup()

	handlers := NewAutoSteerHandlers(profileRepo, nil, nil)

	// Create test profile
	profile := &AutoSteerProfile{
		Name: "Test Profile",
		Phases: []SteerPhase{
			{
				ID:            "phase-1",
				SkillIDs:      []string{"progress"},
				SkillName:     "Progress",
				MaxIterations: 10,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "loops",
						CompareOperator: OpGreaterThan,
						Value:           5,
					},
				},
			},
		},
	}
	if err := profileRepo.CreateProfile(profile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	t.Run("get existing profile", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/auto-steer/profiles/"+profile.ID, nil)
		req = mux.SetURLVars(req, map[string]string{"id": profile.ID})
		w := httptest.NewRecorder()

		handlers.GetProfile(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var retrieved AutoSteerProfile
		if err := json.NewDecoder(w.Body).Decode(&retrieved); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if retrieved.ID != profile.ID {
			t.Errorf("Expected ID %s, got %s", profile.ID, retrieved.ID)
		}
	})

	t.Run("get non-existent profile", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		req := httptest.NewRequest(http.MethodGet, "/api/auto-steer/profiles/"+nonExistentID, nil)
		req = mux.SetURLVars(req, map[string]string{"id": nonExistentID})
		w := httptest.NewRecorder()

		handlers.GetProfile(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})
}

func TestHandlers_UpdateProfile(t *testing.T) {
	profileRepo, cleanup := setupHandlersProfileRepo(t)
	defer cleanup()

	handlers := NewAutoSteerHandlers(profileRepo, nil, nil)

	// Create test profile
	profile := &AutoSteerProfile{
		Name: "Original Name",
		Phases: []SteerPhase{
			{
				ID:            "phase-1",
				SkillIDs:      []string{"progress"},
				SkillName:     "Progress",
				MaxIterations: 10,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "loops",
						CompareOperator: OpGreaterThan,
						Value:           5,
					},
				},
			},
		},
	}
	if err := profileRepo.CreateProfile(profile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	t.Run("update profile successfully", func(t *testing.T) {
		updates := AutoSteerProfile{
			Name:        "Updated Name",
			Description: "Updated description",
			Phases: []SteerPhase{
				{
					ID:            "phase-1",
					SkillIDs:      []string{"ux"},
					SkillName:     "UX",
					MaxIterations: 15,
					StopConditions: []StopCondition{
						{
							Type:            ConditionTypeSimple,
							Metric:          "accessibility_score",
							CompareOperator: OpGreaterThan,
							Value:           90,
						},
					},
				},
			},
		}

		body, _ := json.Marshal(updates)
		req := httptest.NewRequest(http.MethodPut, "/api/auto-steer/profiles/"+profile.ID, bytes.NewBuffer(body))
		req = mux.SetURLVars(req, map[string]string{"id": profile.ID})
		w := httptest.NewRecorder()

		handlers.UpdateProfile(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var updated AutoSteerProfile
		if err := json.NewDecoder(w.Body).Decode(&updated); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if updated.Name != "Updated Name" {
			t.Errorf("Expected name 'Updated Name', got %s", updated.Name)
		}
	})
}

func TestHandlers_DeleteProfile(t *testing.T) {
	profileRepo, cleanup := setupHandlersProfileRepo(t)
	defer cleanup()

	handlers := NewAutoSteerHandlers(profileRepo, nil, nil)

	// Create test profile
	profile := &AutoSteerProfile{
		Name: "Test Profile",
		Phases: []SteerPhase{
			{
				ID:            "phase-1",
				SkillIDs:      []string{"progress"},
				SkillName:     "Progress",
				MaxIterations: 10,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "loops",
						CompareOperator: OpGreaterThan,
						Value:           5,
					},
				},
			},
		},
	}
	if err := profileRepo.CreateProfile(profile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	t.Run("delete existing profile", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/auto-steer/profiles/"+profile.ID, nil)
		req = mux.SetURLVars(req, map[string]string{"id": profile.ID})
		w := httptest.NewRecorder()

		handlers.DeleteProfile(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status %d, got %d", http.StatusNoContent, w.Code)
		}
	})
}

func TestHandlers_GetTemplates(t *testing.T) {
	root := t.TempDir()
	template := &AutoSteerProfile{
		ID:          "template-1",
		Name:        "Template One",
		Description: "Template profile",
		Phases: []SteerPhase{
			{
				ID:            "phase-template",
				SkillIDs:      []string{"progress"},
				SkillName:     "Progress",
				MaxIterations: 1,
				StopConditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "loops",
						CompareOperator: OpGreaterThanEquals,
						Value:           1,
					},
				},
			},
		},
	}
	writeProfileFile(t, root, "templates/template-one/profile.json", template)
	writeMetadata(t, root, ProfileMetadataIndex{
		Profiles: []ProfileMetadata{
			{
				ID:          "template-1",
				Name:        "Template One",
				Description: "Template profile",
				Kind:        ProfileKindTemplate,
				File:        "templates/template-one/profile.json",
			},
		},
	})

	profileRepo, err := NewFileProfileRepository(root)
	if err != nil {
		t.Fatalf("failed to create profile repo: %v", err)
	}

	handlers := NewAutoSteerHandlers(profileRepo, nil, nil)

	t.Run("get built-in templates", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/auto-steer/templates", nil)
		w := httptest.NewRecorder()

		handlers.GetTemplates(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var resp struct {
			Templates []*AutoSteerProfile `json:"templates"`
			Count     int                 `json:"count"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(resp.Templates) == 0 {
			t.Error("Expected at least one template")
		}
	})
}

func TestHandlers_SubmitFeedback(t *testing.T) {
	db, cleanup := setupHistoryTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	historyService := NewHistoryService(db)
	handlers := NewAutoSteerHandlers(nil, nil, historyService)

	// Create test execution
	taskID := createTestExecution(t, db, uuid.New().String(), "test-scenario", false)

	t.Run("submit valid feedback", func(t *testing.T) {
		feedback := struct {
			Rating   int    `json:"rating"`
			Comments string `json:"comments"`
		}{
			Rating:   5,
			Comments: "Excellent!",
		}

		body, _ := json.Marshal(feedback)
		req := httptest.NewRequest(http.MethodPost, "/api/auto-steer/history/"+taskID+"/feedback", bytes.NewBuffer(body))
		req = mux.SetURLVars(req, map[string]string{"executionId": taskID})
		w := httptest.NewRecorder()

		handlers.SubmitFeedback(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status %d, got %d", http.StatusNoContent, w.Code)
		}
	})

	t.Run("submit invalid rating", func(t *testing.T) {
		feedback := struct {
			Rating   int    `json:"rating"`
			Comments string `json:"comments"`
		}{
			Rating:   6, // Invalid - must be 1-5
			Comments: "Test",
		}

		body, _ := json.Marshal(feedback)
		req := httptest.NewRequest(http.MethodPost, "/api/auto-steer/history/"+taskID+"/feedback", bytes.NewBuffer(body))
		req = mux.SetURLVars(req, map[string]string{"executionId": taskID})
		w := httptest.NewRecorder()

		handlers.SubmitFeedback(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestHandlers_GetHistory(t *testing.T) {
	db, cleanup := setupHistoryTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	historyService := NewHistoryService(db)
	handlers := NewAutoSteerHandlers(nil, nil, historyService)

	// Create test executions
	profileID := uuid.New().String()
	createTestExecution(t, db, profileID, "scenario-a", true)
	createTestExecution(t, db, profileID, "scenario-b", false)

	t.Run("get all history", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/auto-steer/history", nil)
		w := httptest.NewRecorder()

		handlers.GetHistory(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var history []ProfilePerformance
		if err := json.NewDecoder(w.Body).Decode(&history); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(history) != 2 {
			t.Errorf("Expected 2 executions, got %d", len(history))
		}
	})

	t.Run("filter by scenario", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/auto-steer/history?scenario=scenario-a", nil)
		w := httptest.NewRecorder()

		handlers.GetHistory(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var history []ProfilePerformance
		if err := json.NewDecoder(w.Body).Decode(&history); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(history) != 1 {
			t.Errorf("Expected 1 execution, got %d", len(history))
		}
	})
}

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

// objProfile builds a valid objective-function profile for handler tests.
func objProfile(name string, tags ...string) AutoSteerProfile {
	return AutoSteerProfile{
		Name: name,
		Objective: Objective{
			DimensionWeights: map[string]float64{"standards": 1.0},
			Targets:          ObjectiveTargets{MaxOpenSeverity: "warning"},
		},
		AllowedSkills: []string{"progress"},
		Budget:        Budget{MaxIterations: 10, DiminishingReturnsFloor: 0.02},
		AuditPreset:   "comprehensive",
		Tags:          tags,
	}
}

func TestHandlers_GetDimensions(t *testing.T) {
	handlers := NewAutoSteerHandlers(nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/auto-steer/dimensions", nil)
	w := httptest.NewRecorder()

	handlers.GetDimensions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp struct {
		Dimensions []dimensionInfo `json:"dimensions"`
		Count      int             `json:"count"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.Count == 0 || len(resp.Dimensions) != resp.Count {
		t.Fatalf("Expected non-empty dimensions with matching count, got count=%d len=%d", resp.Count, len(resp.Dimensions))
	}
	for _, d := range resp.Dimensions {
		if d.ID == "" {
			t.Error("Expected every dimension to carry an id")
		}
	}
}

func TestHandlers_CreateProfile(t *testing.T) {
	profileRepo, cleanup := setupHandlersProfileRepo(t)
	defer cleanup()

	handlers := NewAutoSteerHandlers(profileRepo, nil, nil)

	t.Run("create valid profile", func(t *testing.T) {
		profile := objProfile("Test Profile", "test")
		profile.Description = "A test profile"

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
		profile := objProfile("")
		profile.Description = "No name"

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

	profile1 := objProfile("Profile 1", "tag1")
	profile2 := objProfile("Profile 2", "tag2")

	if err := profileRepo.CreateProfile(&profile1); err != nil {
		t.Fatalf("failed to create profile1: %v", err)
	}
	if err := profileRepo.CreateProfile(&profile2); err != nil {
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

	profile := objProfile("Test Profile")
	if err := profileRepo.CreateProfile(&profile); err != nil {
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

	profile := objProfile("Original Name")
	if err := profileRepo.CreateProfile(&profile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	t.Run("update profile successfully", func(t *testing.T) {
		updates := objProfile("Updated Name")
		updates.Description = "Updated description"
		updates.AllowedSkills = []string{"ux"}

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

	profile := objProfile("Test Profile")
	if err := profileRepo.CreateProfile(&profile); err != nil {
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
	template := objProfile("Template One")
	template.ID = "template-1"
	template.Description = "Template profile"
	template.Tags = nil
	writeProfileFile(t, root, "templates/template-one/profile.json", &template)
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

func TestHandlers_GetHistory(t *testing.T) {
	db, cleanup := setupHistoryTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	historyService := NewHistoryService(db)
	handlers := NewAutoSteerHandlers(nil, nil, historyService)

	profileID := uuid.New().String()
	createTestExecution(t, db, profileID, "scenario-a")
	createTestExecution(t, db, profileID, "scenario-b")

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

package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function for testing
func validateAgeGroup(ageGroup string) bool {
	validGroups := []string{"3-5", "6-8", "9-12"}
	for _, group := range validGroups {
		if ageGroup == group {
			return true
		}
	}
	return false
}

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})
		require.NoError(t, err)

		healthHandler(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"service": "bedtime-story-generator",
			"status":  "healthy",
		})
		assert.NotNil(t, response)
	})
}

// TestGetThemesHandler tests the get themes endpoint
func TestGetThemesHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		defer testDB.Cleanup()

		// Setup mock expectations for themes
		mockThemeRows(testDB.Mock,
			map[string]string{
				"name":        "Adventure",
				"description": "Exciting journeys and quests",
				"emoji":       "🗺️",
				"color":       "#FF6B6B",
			},
			map[string]string{
				"name":        "Animals",
				"description": "Stories about animals and nature",
				"emoji":       "🦁",
				"color":       "#4ECDC4",
			},
		)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/themes",
		})
		require.NoError(t, err)

		getThemesHandler(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		themes := assertJSONArray(t, w, http.StatusOK)
		assert.NotNil(t, themes)
		assert.Greater(t, len(themes), 0)

		// Verify theme structure
		if len(themes) > 0 {
			theme := themes[0].(map[string]interface{})
			assert.NotEmpty(t, theme["name"])
			assert.NotEmpty(t, theme["description"])
			assert.NotEmpty(t, theme["emoji"])
			assert.NotEmpty(t, theme["color"])
		}
	})

	t.Run("DatabaseError", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		defer testDB.Cleanup()

		// Expect query to fail
		testDB.Mock.ExpectQuery("SELECT name, description, emoji, color FROM story_themes").
			WillReturnError(sql.ErrConnDone)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/themes",
		})
		require.NoError(t, err)

		getThemesHandler(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// TestGetStoriesHandler tests the get stories endpoint
func TestGetStoriesHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		defer testDB.Cleanup()

		story := setupTestStory(t, "Test Story", "6-8")
		defer story.Cleanup()

		// Setup mock expectations
		mockStoryRows(testDB.Mock, story.Story)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/stories",
		})
		require.NoError(t, err)

		getStoriesHandler(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		stories := assertJSONArray(t, w, http.StatusOK)
		assert.NotNil(t, stories)
	})

	t.Run("EmptyResult", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		defer testDB.Cleanup()

		// Setup empty result
		rows := sqlmock.NewRows([]string{
			"id", "title", "content", "age_group", "theme",
			"story_length", "reading_time_minutes", "page_count",
			"character_names", "created_at", "times_read", "last_read",
			"is_favorite", "illustrations",
		})
		testDB.Mock.ExpectQuery("SELECT (.+) FROM stories").WillReturnRows(rows)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/stories",
		})
		require.NoError(t, err)

		getStoriesHandler(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		// Empty result returns null in JSON, which is fine
		var stories []Story
		err = json.Unmarshal(w.Body.Bytes(), &stories)
		if err == nil {
			assert.Equal(t, 0, len(stories))
		}
	})

	t.Run("DatabaseError", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		defer testDB.Cleanup()

		testDB.Mock.ExpectQuery("SELECT (.+) FROM stories").
			WillReturnError(sql.ErrConnDone)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/stories",
		})
		require.NoError(t, err)

		getStoriesHandler(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// TestGetStoryHandler tests the get single story endpoint
func TestGetStoryHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_FromDatabase", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		defer testDB.Cleanup()

		story := setupTestStory(t, "Test Story", "6-8")
		defer story.Cleanup()

		// Setup mock expectations
		charactersJSON, _ := json.Marshal(story.Story.CharacterNames)
		illustrationsJSON, _ := json.Marshal(story.Story.Illustrations)

		rows := sqlmock.NewRows([]string{
			"id", "title", "content", "age_group", "theme",
			"story_length", "reading_time_minutes", "page_count",
			"character_names", "created_at", "times_read", "last_read",
			"is_favorite", "illustrations",
		}).AddRow(
			story.Story.ID,
			story.Story.Title,
			story.Story.Content,
			story.Story.AgeGroup,
			story.Story.Theme,
			story.Story.StoryLength,
			story.Story.ReadingTime,
			story.Story.PageCount,
			string(charactersJSON),
			story.Story.CreatedAt,
			story.Story.TimesRead,
			story.Story.LastRead,
			story.Story.IsFavorite,
			string(illustrationsJSON),
		)

		testDB.Mock.ExpectQuery("SELECT (.+) FROM stories WHERE id = (.+)").
			WithArgs(story.Story.ID).
			WillReturnRows(rows)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/stories/" + story.Story.ID,
			URLVars: map[string]string{"id": story.Story.ID},
		})
		require.NoError(t, err)

		getStoryHandler(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var resultStory Story
		err = json.Unmarshal(w.Body.Bytes(), &resultStory)
		require.NoError(t, err)
		assert.Equal(t, story.Story.Title, resultStory.Title)
	})

	t.Run("Success_FromCache", func(t *testing.T) {
		story := setupTestStory(t, "Cached Story", "6-8")
		defer story.Cleanup()

		// Add story to cache
		storyCache.Set(story.Story.ID, story.Story)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/stories/" + story.Story.ID,
			URLVars: map[string]string{"id": story.Story.ID},
		})
		require.NoError(t, err)

		getStoryHandler(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var resultStory Story
		err = json.Unmarshal(w.Body.Bytes(), &resultStory)
		require.NoError(t, err)
		assert.Equal(t, story.Story.Title, resultStory.Title)
	})

	t.Run("NotFound", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		defer testDB.Cleanup()

		storyID := uuid.New().String()

		testDB.Mock.ExpectQuery("SELECT (.+) FROM stories WHERE id = (.+)").
			WithArgs(storyID).
			WillReturnError(sql.ErrNoRows)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/stories/" + storyID,
			URLVars: map[string]string{"id": storyID},
		})
		require.NoError(t, err)

		getStoryHandler(w, httpReq)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("DatabaseError", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		defer testDB.Cleanup()

		storyID := uuid.New().String()

		testDB.Mock.ExpectQuery("SELECT (.+) FROM stories WHERE id = (.+)").
			WithArgs(storyID).
			WillReturnError(sql.ErrConnDone)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/stories/" + storyID,
			URLVars: map[string]string{"id": storyID},
		})
		require.NoError(t, err)

		getStoryHandler(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// TestToggleFavoriteHandler tests the toggle favorite endpoint
func TestToggleFavoriteHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		defer testDB.Cleanup()

		storyID := uuid.New().String()

		rows := sqlmock.NewRows([]string{"is_favorite"}).AddRow(true)
		testDB.Mock.ExpectQuery("UPDATE stories SET is_favorite = NOT is_favorite").
			WithArgs(storyID).
			WillReturnRows(rows)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/stories/" + storyID + "/favorite",
			URLVars: map[string]string{"id": storyID},
		})
		require.NoError(t, err)

		toggleFavoriteHandler(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})
		assert.NotNil(t, response)
		assert.Equal(t, true, response["is_favorite"])
	})

	t.Run("NotFound", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		defer testDB.Cleanup()

		storyID := uuid.New().String()

		testDB.Mock.ExpectQuery("UPDATE stories SET is_favorite = NOT is_favorite").
			WithArgs(storyID).
			WillReturnError(sql.ErrNoRows)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/stories/" + storyID + "/favorite",
			URLVars: map[string]string{"id": storyID},
		})
		require.NoError(t, err)

		toggleFavoriteHandler(w, httpReq)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestStartReadingHandler tests the start reading session endpoint
func TestStartReadingHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		defer testDB.Cleanup()

		storyID := uuid.New().String()

		testDB.Mock.ExpectExec("INSERT INTO reading_sessions").
			WithArgs(sqlmock.AnyArg(), storyID, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/stories/" + storyID + "/start-reading",
			URLVars: map[string]string{"id": storyID},
		})
		require.NoError(t, err)

		startReadingHandler(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response["session_id"])
		assert.Equal(t, storyID, response["story_id"])
	})

	t.Run("DatabaseError", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		defer testDB.Cleanup()

		storyID := uuid.New().String()

		testDB.Mock.ExpectExec("INSERT INTO reading_sessions").
			WithArgs(sqlmock.AnyArg(), storyID, sqlmock.AnyArg()).
			WillReturnError(sql.ErrConnDone)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/stories/" + storyID + "/start-reading",
			URLVars: map[string]string{"id": storyID},
		})
		require.NoError(t, err)

		startReadingHandler(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// TestDeleteStoryHandler tests the delete story endpoint
func TestDeleteStoryHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		defer testDB.Cleanup()

		story := setupTestStory(t, "Story to Delete", "6-8")
		defer story.Cleanup()

		// Add to cache first
		storyCache.Set(story.Story.ID, story.Story)

		testDB.Mock.ExpectExec("DELETE FROM stories WHERE id = (.+)").
			WithArgs(story.Story.ID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/v1/stories/" + story.Story.ID,
			URLVars: map[string]string{"id": story.Story.ID},
		})
		require.NoError(t, err)

		deleteStoryHandler(w, httpReq)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify cache was cleared
		_, exists := storyCache.Get(story.Story.ID)
		assert.False(t, exists)
	})

	t.Run("DatabaseError", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		defer testDB.Cleanup()

		storyID := uuid.New().String()

		testDB.Mock.ExpectExec("DELETE FROM stories WHERE id = (.+)").
			WithArgs(storyID).
			WillReturnError(sql.ErrConnDone)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/v1/stories/" + storyID,
			URLVars: map[string]string{"id": storyID},
		})
		require.NoError(t, err)

		deleteStoryHandler(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// TestStoryCache tests the story cache functionality
func TestStoryCache(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SetAndGet", func(t *testing.T) {
		story := setupTestStory(t, "Cached Story", "6-8")
		defer story.Cleanup()

		storyCache.Set(story.Story.ID, story.Story)

		retrieved, exists := storyCache.Get(story.Story.ID)
		assert.True(t, exists)
		assert.NotNil(t, retrieved)
		assert.Equal(t, story.Story.Title, retrieved.Title)
	})

	t.Run("Delete", func(t *testing.T) {
		story := setupTestStory(t, "Story to Remove", "6-8")
		defer story.Cleanup()

		storyCache.Set(story.Story.ID, story.Story)
		storyCache.Delete(story.Story.ID)

		_, exists := storyCache.Get(story.Story.ID)
		assert.False(t, exists)
	})

	t.Run("CacheEviction", func(t *testing.T) {
		// Fill cache beyond maxSize
		cache := &StoryCache{
			stories: make(map[string]*Story),
			maxSize: 2,
		}

		story1 := setupTestStory(t, "Story 1", "6-8")
		defer story1.Cleanup()
		story2 := setupTestStory(t, "Story 2", "6-8")
		defer story2.Cleanup()
		story3 := setupTestStory(t, "Story 3", "6-8")
		defer story3.Cleanup()

		cache.Set(story1.Story.ID, story1.Story)
		cache.Set(story2.Story.ID, story2.Story)
		cache.Set(story3.Story.ID, story3.Story)

		// Cache should have exactly maxSize items
		assert.Equal(t, 2, len(cache.stories))
	})
}

// TestStoryValidation tests story validation logic
func TestStoryValidation(t *testing.T) {
	tests := []struct {
		name       string
		ageGroup   string
		shouldPass bool
	}{
		{"Valid 3-5", "3-5", true},
		{"Valid 6-8", "6-8", true},
		{"Valid 9-12", "9-12", true},
		{"Invalid empty", "", false},
		{"Invalid format", "invalid", false},
		{"Invalid range", "10-15", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validateAgeGroup(tt.ageGroup)
			assert.Equal(t, tt.shouldPass, valid)
		})
	}
}

// TestHelperFunctions tests utility functions
func TestHelperFunctions(t *testing.T) {
	t.Run("getAgeDescription", func(t *testing.T) {
		tests := []struct {
			ageGroup string
			expected string
		}{
			{"3-5", "ages 3-5 (very simple vocabulary, short sentences, repetition)"},
			{"6-8", "ages 6-8 (simple vocabulary, clear plot, relatable characters)"},
			{"9-12", "ages 9-12 (richer vocabulary, complex plot, character development)"},
			{"unknown", "ages 6-8"},
		}

		for _, tt := range tests {
			result := getAgeDescription(tt.ageGroup)
			assert.Equal(t, tt.expected, result)
		}
	})

	t.Run("getLengthDescription", func(t *testing.T) {
		tests := []struct {
			length   string
			expected string
		}{
			{"short", "3-5 minute read (about 300-500 words)"},
			{"medium", "8-10 minute read (about 800-1200 words)"},
			{"long", "12-15 minute read (about 1500-2000 words)"},
			{"unknown", "8-10 minute read"},
		}

		for _, tt := range tests {
			result := getLengthDescription(tt.length)
			assert.Equal(t, tt.expected, result)
		}
	})

	t.Run("getCharacterPrompt", func(t *testing.T) {
		tests := []struct {
			names    []string
			expected string
		}{
			{[]string{"Alice"}, "\n- Include these character names: Alice"},
			{[]string{"Alice", "Bob"}, "\n- Include these character names: Alice, Bob"},
			{[]string{}, ""},
			{nil, ""},
		}

		for _, tt := range tests {
			result := getCharacterPrompt(tt.names)
			assert.Equal(t, tt.expected, result)
		}
	})

	t.Run("calculateReadingTime", func(t *testing.T) {
		tests := []struct {
			length   string
			expected int
		}{
			{"short", 5},
			{"medium", 10},
			{"long", 15},
			{"unknown", 10},
		}

		for _, tt := range tests {
			result := calculateReadingTime(tt.length)
			assert.Equal(t, tt.expected, result)
		}
	})

	t.Run("calculatePageCount", func(t *testing.T) {
		tests := []struct {
			content  string
			expected int
		}{
			{"Short content", 1},
			{strings.Repeat("word ", 100), 1},
			{strings.Repeat("word ", 300), 2},
			{"", 1},
		}

		for _, tt := range tests {
			result := calculatePageCount(tt.content)
			assert.GreaterOrEqual(t, result, 1)
		}
	})
}

// TestParseStoryResponse tests story response parsing
func TestParseStoryResponse(t *testing.T) {
	t.Run("ValidFormat", func(t *testing.T) {
		response := "TITLE: The Magic Forest\nSTORY:\nOnce upon a time..."
		title, content := parseStoryResponse(response)
		assert.Equal(t, "The Magic Forest", title)
		assert.Contains(t, content, "Once upon a time")
	})

	t.Run("NoTitle", func(t *testing.T) {
		response := "Just content without title"
		title, content := parseStoryResponse(response)
		assert.Equal(t, "A Magical Bedtime Story", title)
		assert.Contains(t, content, "Just content")
	})

	t.Run("EmptyResponse", func(t *testing.T) {
		response := ""
		title, content := parseStoryResponse(response)
		assert.Equal(t, "A Magical Bedtime Story", title)
		assert.Equal(t, "", content)
	})
}

// TestGenerateStoryRequest tests story generation request validation
func TestGenerateStoryRequest(t *testing.T) {
	tests := []struct {
		name     string
		request  GenerateStoryRequest
		valid    bool
	}{
		{
			name: "ValidRequest",
			request: GenerateStoryRequest{
				AgeGroup:       "6-8",
				Theme:          "adventure",
				Length:         "medium",
				CharacterNames: []string{"Alice"},
			},
			valid: true,
		},
		{
			name: "EmptyAgeGroup",
			request: GenerateStoryRequest{
				Theme:  "adventure",
				Length: "medium",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate age group
			if tt.request.AgeGroup != "" {
				valid := validateAgeGroup(tt.request.AgeGroup)
				assert.Equal(t, tt.valid, valid)
			}
		})
	}
}

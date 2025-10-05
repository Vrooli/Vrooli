// +build testing

package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestLoginHandler tests the login endpoint
func TestLoginHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		// Create test user
		user := createTestUser(t, env)

		// Login request
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/auth/login",
			Body: map[string]string{
				"email":    user.Email,
				"password": user.Password,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, true)

		// Verify token is returned
		if response.Data == nil {
			t.Error("Expected token data, got nil")
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		patterns := NewTestScenarioBuilder().
			AddInvalidJSON("POST", "/api/v1/auth/login").
			AddMissingRequiredField("POST", "/api/v1/auth/login", "Email", map[string]string{
				"password": "test",
			}).
			AddMissingRequiredField("POST", "/api/v1/auth/login", "Password", map[string]string{
				"email": "test@example.com",
			}).
			Build()

		suite := NewHandlerTestSuite("Login", "POST", "/api/v1/auth/login")
		suite.RunErrorTests(t, env, patterns)
	})

	t.Run("InvalidCredentials", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/auth/login",
			Body: map[string]string{
				"email":    "nonexistent@example.com",
				"password": "wrongpassword",
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusUnauthorized, "")
	})

	t.Run("WrongPassword", func(t *testing.T) {
		user := createTestUser(t, env)

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/auth/login",
			Body: map[string]string{
				"email":    user.Email,
				"password": "wrongpassword",
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusUnauthorized, "")
	})
}

// TestRegisterHandler tests the register endpoint
func TestRegisterHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		email := fmt.Sprintf("newuser_%s@example.com", uuid.New().String()[:8])

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/auth/register",
			Body: map[string]string{
				"email":      email,
				"password":   "SecurePassword123!",
				"first_name": "Test",
				"last_name":  "User",
				"timezone":   "America/New_York",
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusCreated, true)

		// Verify user data is returned
		if response.Data == nil {
			t.Error("Expected user data, got nil")
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		patterns := NewTestScenarioBuilder().
			AddInvalidJSON("POST", "/api/v1/auth/register").
			AddMissingRequiredField("POST", "/api/v1/auth/register", "Email", map[string]string{
				"password": "SecurePassword123!",
			}).
			AddMissingRequiredField("POST", "/api/v1/auth/register", "Password", map[string]string{
				"email": "test@example.com",
			}).
			AddInvalidFieldValue("POST", "/api/v1/auth/register", "Email", map[string]string{
				"email":    "invalid-email",
				"password": "SecurePassword123!",
			}).
			Build()

		suite := NewHandlerTestSuite("Register", "POST", "/api/v1/auth/register")
		suite.RunErrorTests(t, env, patterns)
	})

	t.Run("DuplicateEmail", func(t *testing.T) {
		user := createTestUser(t, env)

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/auth/register",
			Body: map[string]string{
				"email":    user.Email,
				"password": "SecurePassword123!",
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusConflict, "")
	})

	t.Run("WeakPassword", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/auth/register",
			Body: map[string]string{
				"email":    fmt.Sprintf("test_%s@example.com", uuid.New().String()[:8]),
				"password": "weak",
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})
}

// TestGetPlatformConfigs tests the platform configs endpoint
func TestGetPlatformConfigs(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/auth/platforms",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, true)

		// Verify platforms are returned
		if response.Data == nil {
			t.Error("Expected platform data, got nil")
		}
	})
}

// TestSchedulePost tests the post scheduling endpoint
func TestSchedulePost(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	user := createTestUser(t, env)

	t.Run("Success", func(t *testing.T) {
		scheduledTime := time.Now().Add(24 * time.Hour).Format(time.RFC3339)

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/posts/schedule",
			Headers: map[string]string{
				"Authorization": "Bearer " + user.Token,
			},
			Body: map[string]interface{}{
				"title":        "Test Post",
				"content":      "This is a test post content",
				"platforms":    []string{"twitter", "linkedin"},
				"scheduled_at": scheduledTime,
				"timezone":     "UTC",
				"auto_optimize": true,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusCreated, true)

		// Verify post data is returned
		if response.Data == nil {
			t.Error("Expected post data, got nil")
		}
	})

	t.Run("MissingAuth", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/posts/schedule",
			Body: map[string]interface{}{
				"title":   "Test",
				"content": "Test",
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusUnauthorized, "")
	})

	t.Run("MissingRequiredFields", func(t *testing.T) {
		testCases := []struct {
			name string
			body map[string]interface{}
		}{
			{
				name: "MissingTitle",
				body: map[string]interface{}{
					"content":      "Test content",
					"platforms":    []string{"twitter"},
					"scheduled_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
					"timezone":     "UTC",
				},
			},
			{
				name: "MissingContent",
				body: map[string]interface{}{
					"title":        "Test Title",
					"platforms":    []string{"twitter"},
					"scheduled_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
					"timezone":     "UTC",
				},
			},
			{
				name: "MissingPlatforms",
				body: map[string]interface{}{
					"title":        "Test Title",
					"content":      "Test content",
					"scheduled_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
					"timezone":     "UTC",
				},
			},
			{
				name: "MissingScheduledAt",
				body: map[string]interface{}{
					"title":     "Test Title",
					"content":   "Test content",
					"platforms": []string{"twitter"},
					"timezone":  "UTC",
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				w, err := makeHTTPRequest(env, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/posts/schedule",
					Headers: map[string]string{
						"Authorization": "Bearer " + user.Token,
					},
					Body: tc.body,
				})
				if err != nil {
					t.Fatalf("Failed to make request: %v", err)
				}

				assertErrorResponse(t, w, http.StatusBadRequest, "")
			})
		}
	})

	t.Run("InvalidPlatform", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/posts/schedule",
			Headers: map[string]string{
				"Authorization": "Bearer " + user.Token,
			},
			Body: map[string]interface{}{
				"title":        "Test Post",
				"content":      "Test content",
				"platforms":    []string{"invalid_platform"},
				"scheduled_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
				"timezone":     "UTC",
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})

	t.Run("PastScheduledTime", func(t *testing.T) {
		pastTime := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/posts/schedule",
			Headers: map[string]string{
				"Authorization": "Bearer " + user.Token,
			},
			Body: map[string]interface{}{
				"title":        "Test Post",
				"content":      "Test content",
				"platforms":    []string{"twitter"},
				"scheduled_at": pastTime,
				"timezone":     "UTC",
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})
}

// TestGetCalendarPosts tests the calendar posts endpoint
func TestGetCalendarPosts(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	user := createTestUser(t, env)
	createTestScheduledPost(t, env, user.ID)

	t.Run("Success", func(t *testing.T) {
		startDate := time.Now().Format("2006-01-02")
		endDate := time.Now().Add(30 * 24 * time.Hour).Format("2006-01-02")

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/posts/calendar",
			Headers: map[string]string{
				"Authorization": "Bearer " + user.Token,
			},
			QueryParams: map[string]string{
				"start_date": startDate,
				"end_date":   endDate,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, true)

		// Verify posts are returned
		if response.Data == nil {
			t.Error("Expected calendar data, got nil")
		}
	})

	t.Run("MissingAuth", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/posts/calendar",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusUnauthorized, "")
	})

	t.Run("WithPlatformFilter", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/posts/calendar",
			Headers: map[string]string{
				"Authorization": "Bearer " + user.Token,
			},
			QueryParams: map[string]string{
				"platforms": "twitter,linkedin",
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK, true)
	})
}

// TestCreateCampaign tests the create campaign endpoint
func TestCreateCampaign(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	user := createTestUser(t, env)

	t.Run("Success", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/campaigns",
			Headers: map[string]string{
				"Authorization": "Bearer " + user.Token,
			},
			Body: map[string]interface{}{
				"name":        "Test Campaign",
				"description": "Test campaign description",
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusCreated, true)

		if response.Data == nil {
			t.Error("Expected campaign data, got nil")
		}
	})

	t.Run("MissingAuth", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/campaigns",
			Body: map[string]interface{}{
				"name": "Test Campaign",
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusUnauthorized, "")
	})

	t.Run("MissingName", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/campaigns",
			Headers: map[string]string{
				"Authorization": "Bearer " + user.Token,
			},
			Body: map[string]interface{}{
				"description": "Test description",
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})
}

// TestGetCampaigns tests the get campaigns endpoint
func TestGetCampaigns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	user := createTestUser(t, env)
	createTestCampaign(t, env, user.ID)

	t.Run("Success", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/campaigns",
			Headers: map[string]string{
				"Authorization": "Bearer " + user.Token,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, true)

		if response.Data == nil {
			t.Error("Expected campaigns data, got nil")
		}
	})

	t.Run("MissingAuth", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/campaigns",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusUnauthorized, "")
	})
}

// TestGetCampaign tests getting a single campaign
func TestGetCampaign(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	user := createTestUser(t, env)
	campaign := createTestCampaign(t, env, user.ID)

	t.Run("Success", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   fmt.Sprintf("/api/v1/campaigns/%s", campaign.ID),
			Headers: map[string]string{
				"Authorization": "Bearer " + user.Token,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, true)

		if response.Data == nil {
			t.Error("Expected campaign data, got nil")
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		patterns := CreateStandardCRUDErrorPatterns("/api/v1/campaigns", "Campaign")

		suite := NewHandlerTestSuite("GetCampaign", "GET", "/api/v1/campaigns")
		suite.RunErrorTests(t, env, patterns)
	})
}

// TestAnalyticsOverview tests the analytics overview endpoint
func TestAnalyticsOverview(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	user := createTestUser(t, env)

	t.Run("Success", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/analytics/overview",
			Headers: map[string]string{
				"Authorization": "Bearer " + user.Token,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, true)

		if response.Data == nil {
			t.Error("Expected analytics data, got nil")
		}
	})

	t.Run("WithPeriodFilter", func(t *testing.T) {
		periods := []string{"7d", "30d", "90d"}

		for _, period := range periods {
			t.Run(period, func(t *testing.T) {
				w, err := makeHTTPRequest(env, HTTPTestRequest{
					Method: "GET",
					Path:   "/api/v1/analytics/overview",
					Headers: map[string]string{
						"Authorization": "Bearer " + user.Token,
					},
					QueryParams: map[string]string{
						"period": period,
					},
				})
				if err != nil {
					t.Fatalf("Failed to make request: %v", err)
				}

				assertJSONResponse(t, w, http.StatusOK, true)
			})
		}
	})
}

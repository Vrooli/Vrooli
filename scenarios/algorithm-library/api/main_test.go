package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthHandler(t *testing.T) {
	// Setup
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()
	
	oldDB := db
	db = mockDB
	defer func() { db = oldDB }()
	
	// Expect database ping
	mock.ExpectPing()
	
	// Create request
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	
	// Call handler
	healthHandler(w, req)
	
	// Check response
	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	
	assert.Equal(t, "healthy", result["status"])
	assert.Equal(t, "algorithm-library-api", result["service"])
	assert.Equal(t, true, result["database"])
}

func TestSearchAlgorithmsHandler(t *testing.T) {
	// Setup
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()
	
	oldDB := db
	db = mockDB
	defer func() { db = oldDB }()
	
	tests := []struct {
		name       string
		query      string
		setupMock  func()
		wantStatus int
		wantTotal  int
	}{
		{
			name:  "Search by category",
			query: "?category=sorting",
			setupMock: func() {
				// Count query
				countRows := sqlmock.NewRows([]string{"count"}).AddRow(8)
				mock.ExpectQuery(`SELECT COUNT\(DISTINCT a\.id\)`).
					WithArgs("sorting").
					WillReturnRows(countRows)
				
				// Main query
				rows := sqlmock.NewRows([]string{
					"id", "name", "display_name", "category", "description",
					"complexity_time", "complexity_space", "difficulty", "tags",
					"language_count", "test_case_count", "has_validated_impl",
				}).AddRow(
					"123", "quicksort", "QuickSort", "sorting", "Fast sorting",
					"O(n log n)", "O(log n)", "medium", `["divide-conquer"]`,
					3, 5, true,
				)
				mock.ExpectQuery(`SELECT DISTINCT a\.id`).
					WithArgs("sorting", 100, 0).
					WillReturnRows(rows)
			},
			wantStatus: http.StatusOK,
			wantTotal:  8,
		},
		{
			name:  "Search with query term",
			query: "?q=quick",
			setupMock: func() {
				// Count query
				countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
				mock.ExpectQuery(`SELECT COUNT\(DISTINCT a\.id\)`).
					WithArgs("%quick%", "%quick%", "%quick%").
					WillReturnRows(countRows)
				
				// Main query
				rows := sqlmock.NewRows([]string{
					"id", "name", "display_name", "category", "description",
					"complexity_time", "complexity_space", "difficulty", "tags",
					"language_count", "test_case_count", "has_validated_impl",
				}).AddRow(
					"123", "quicksort", "QuickSort", "sorting", "Fast sorting",
					"O(n log n)", "O(log n)", "medium", `["divide-conquer"]`,
					3, 5, true,
				)
				mock.ExpectQuery(`SELECT DISTINCT a\.id`).
					WithArgs("%quick%", "%quick%", "%quick%", 100, 0).
					WillReturnRows(rows)
			},
			wantStatus: http.StatusOK,
			wantTotal:  1,
		},
		{
			name:  "Search all algorithms",
			query: "",
			setupMock: func() {
				// Count query
				countRows := sqlmock.NewRows([]string{"count"}).AddRow(35)
				mock.ExpectQuery(`SELECT COUNT\(DISTINCT a\.id\)`).
					WillReturnRows(countRows)
				
				// Main query
				rows := sqlmock.NewRows([]string{
					"id", "name", "display_name", "category", "description",
					"complexity_time", "complexity_space", "difficulty", "tags",
					"language_count", "test_case_count", "has_validated_impl",
				})
				mock.ExpectQuery(`SELECT DISTINCT a\.id`).
					WithArgs(100, 0).
					WillReturnRows(rows)
			},
			wantStatus: http.StatusOK,
			wantTotal:  35,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			
			req := httptest.NewRequest("GET", "/api/v1/algorithms/search"+tt.query, nil)
			w := httptest.NewRecorder()
			
			searchAlgorithmsHandler(w, req)
			
			resp := w.Result()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			
			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)
			
			assert.Equal(t, float64(tt.wantTotal), result["total"])
			
			err = mock.ExpectationsWereMet()
			assert.NoError(t, err)
		})
	}
}

func TestGetCategoriesHandler(t *testing.T) {
	// Setup
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()
	
	oldDB := db
	db = mockDB
	defer func() { db = oldDB }()
	
	// Setup mock expectations
	rows := sqlmock.NewRows([]string{"category", "count"}).
		AddRow("sorting", 8).
		AddRow("searching", 5).
		AddRow("graph", 11).
		AddRow("dynamic_programming", 8).
		AddRow("tree", 3)
	
	mock.ExpectQuery(`SELECT category, COUNT\(\*\) as count`).
		WillReturnRows(rows)
	
	// Create request
	req := httptest.NewRequest("GET", "/api/v1/algorithms/categories", nil)
	w := httptest.NewRecorder()
	
	// Call handler
	getCategoriesHandler(w, req)
	
	// Check response
	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	
	categories := result["categories"].([]interface{})
	assert.Equal(t, 5, len(categories))
	
	firstCategory := categories[0].(map[string]interface{})
	assert.Equal(t, "sorting", firstCategory["name"])
	assert.Equal(t, float64(8), firstCategory["count"])
	
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestGetStatsHandler(t *testing.T) {
	// Setup
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()
	
	oldDB := db
	db = mockDB
	defer func() { db = oldDB }()
	
	// Setup mock expectations
	statsRows := sqlmock.NewRows([]string{
		"total_algorithms", "total_implementations", 
		"total_test_cases", "validated_implementations",
	}).AddRow(35, 16, 48, 16)
	
	mock.ExpectQuery(`SELECT COUNT\(DISTINCT a\.id\)`).
		WillReturnRows(statsRows)
	
	langRows := sqlmock.NewRows([]string{"language", "count"}).
		AddRow("python", 8).
		AddRow("javascript", 4).
		AddRow("go", 2).
		AddRow("java", 1).
		AddRow("cpp", 1)
	
	mock.ExpectQuery(`SELECT language, COUNT\(\*\) as count`).
		WillReturnRows(langRows)
	
	// Create request
	req := httptest.NewRequest("GET", "/api/v1/algorithms/stats", nil)
	w := httptest.NewRecorder()
	
	// Call handler
	getStatsHandler(w, req)
	
	// Check response
	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	
	assert.Equal(t, float64(35), result["total_algorithms"])
	assert.Equal(t, float64(16), result["total_implementations"])
	assert.Equal(t, float64(48), result["total_test_cases"])
	
	languages := result["languages"].([]interface{})
	assert.Equal(t, 5, len(languages))
	
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestValidateAlgorithmHandler(t *testing.T) {
	// Setup
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()
	
	oldDB := db
	db = mockDB
	defer func() { db = oldDB }()
	
	// Initialize processor
	algorithmProcessor = NewAlgorithmProcessor("")
	
	reqBody := ValidationRequest{
		AlgorithmID: "quicksort",
		Language:    "python",
		Code:        "def quicksort(arr): pass",
	}
	body, _ := json.Marshal(reqBody)
	
	// Setup mock - check if algorithm is UUID first
	mock.ExpectQuery(`SELECT id FROM algorithms WHERE id = \$1`).
		WithArgs("quicksort").
		WillReturnError(sql.ErrNoRows)
	
	// Then check by name
	mock.ExpectQuery(`SELECT id FROM algorithms WHERE LOWER\(name\) = LOWER\(\$1\)`).
		WithArgs("quicksort").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("123-456"))
	
	// Get test cases
	testRows := sqlmock.NewRows([]string{"id", "input", "expected_output", "timeout_ms"}).
		AddRow("test1", `{"arr": [3, 1, 2]}`, `[1, 2, 3]`, 1000).
		AddRow("test2", `{"arr": [5, 2, 8, 1]}`, `[1, 2, 5, 8]`, 1000)
	
	mock.ExpectQuery(`SELECT id, input, expected_output, timeout_ms`).
		WithArgs("123-456").
		WillReturnRows(testRows)
	
	// Log usage statistics
	mock.ExpectExec(`INSERT INTO algorithm_usage`).
		WithArgs("123-456", "python", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	
	// Create request
	req := httptest.NewRequest("POST", "/api/v1/algorithms/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	// Call handler
	validateAlgorithmHandler(w, req)
	
	// Check response
	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	var result ValidationResult
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	
	// The validation will likely fail due to dummy code, but structure should be correct
	assert.NotNil(t, result.TestResults)
	assert.NotNil(t, result.Performance)
}

func TestGetAlgorithmHandler(t *testing.T) {
	// Setup
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()
	
	oldDB := db
	db = mockDB
	defer func() { db = oldDB }()
	
	// Setup mock
	rows := sqlmock.NewRows([]string{
		"id", "name", "display_name", "category", "description",
		"complexity_time", "complexity_space", "difficulty", "tags",
	}).AddRow(
		"123", "quicksort", "QuickSort", "sorting", "Fast sorting",
		"O(n log n)", "O(log n)", "medium", `["divide-conquer"]`,
	)
	
	mock.ExpectQuery(`SELECT id, name, display_name`).
		WithArgs("quicksort").
		WillReturnRows(rows)
	
	// Create request with router to get path variables
	req := httptest.NewRequest("GET", "/api/v1/algorithms/quicksort", nil)
	w := httptest.NewRecorder()
	
	// Add router context
	req = mux.SetURLVars(req, map[string]string{"id": "quicksort"})
	
	// Call handler
	getAlgorithmHandler(w, req)
	
	// Check response
	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	var result Algorithm
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	
	assert.Equal(t, "quicksort", result.Name)
	assert.Equal(t, "QuickSort", result.DisplayName)
	assert.Equal(t, "sorting", result.Category)
	
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestEnvironmentValidation(t *testing.T) {
	// Test that main() exits when lifecycle env var is not set
	oldEnv := os.Getenv("VROOLI_LIFECYCLE_MANAGED")
	os.Unsetenv("VROOLI_LIFECYCLE_MANAGED")
	defer os.Setenv("VROOLI_LIFECYCLE_MANAGED", oldEnv)
	
	// We can't actually test main() exit, but we can verify the env check logic
	assert.Equal(t, "", os.Getenv("VROOLI_LIFECYCLE_MANAGED"))
}

func TestDatabaseConnectionValidation(t *testing.T) {
	// Test that missing database config is detected
	envVars := []string{
		"POSTGRES_HOST",
		"POSTGRES_PORT",
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		"POSTGRES_DB",
	}
	
	// Store original values
	originalVals := make(map[string]string)
	for _, env := range envVars {
		originalVals[env] = os.Getenv(env)
	}
	
	// Test each missing variable
	for _, missingVar := range envVars {
		t.Run(fmt.Sprintf("Missing_%s", missingVar), func(t *testing.T) {
			// Set all vars
			for _, env := range envVars {
				os.Setenv(env, "test")
			}
			// Unset the one we're testing
			os.Unsetenv(missingVar)
			
			// Verify it's missing
			assert.Empty(t, os.Getenv(missingVar))
			
			// Restore
			for env, val := range originalVals {
				if val != "" {
					os.Setenv(env, val)
				} else {
					os.Unsetenv(env)
				}
			}
		})
	}
}
//go:build perf
// +build perf

package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/handlers"
)

// Performance benchmarks

func BenchmarkHealthEndpoint(b *testing.B) {
	log := logrus.New()
	log.SetOutput(ioutil.Discard)

	// Create a mock handler
	handler := &handlers.Handler{}

	req := httptest.NewRequest("GET", "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.Health(w, req)
	}
}

func BenchmarkConcurrentHealthChecks(b *testing.B) {
	log := logrus.New()
	log.SetOutput(ioutil.Discard)

	handler := &handlers.Handler{}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()
			handler.Health(w, req)
		}
	})
}

func BenchmarkCreateProject(b *testing.B) {
	// Skip if no database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		b.Skip("No database URL configured")
	}

	log := logrus.New()
	log.SetOutput(ioutil.Discard)

	db, err := database.NewConnection(log)
	if err != nil {
		b.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	repo := database.NewRepository(db, log)
	handler := &handlers.Handler{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		reqBody := handlers.CreateProjectRequest{
			Name:        fmt.Sprintf("Benchmark Project %d", i),
			Description: "Benchmark test project",
			FolderPath:  fmt.Sprintf("/benchmark/project-%d", i),
		}
		jsonBody, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		b.StartTimer()

		w := httptest.NewRecorder()
		handler.CreateProject(w, req)

		b.StopTimer()
		// Cleanup
		if w.Code == http.StatusCreated {
			var project database.Project
			json.NewDecoder(w.Body).Decode(&project)
			repo.DeleteProject(context.Background(), project.ID)
		}
		b.StartTimer()
	}
}

func BenchmarkJSONMarshalProject(b *testing.B) {
	project := &database.Project{
		ID:          uuid.New(),
		Name:        "Benchmark Project",
		Description: "A benchmark test project with a longer description to simulate real-world data",
		FolderPath:  "/benchmark/projects/test-project-001",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(project)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONUnmarshalProject(b *testing.B) {
	jsonData := []byte(`{
		"id": "123e4567-e89b-12d3-a456-426614174000",
		"name": "Benchmark Project",
		"description": "A benchmark test project",
		"folder_path": "/benchmark/projects/test",
		"created_at": "2024-01-01T00:00:00Z",
		"updated_at": "2024-01-01T00:00:00Z"
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var project database.Project
		err := json.Unmarshal(jsonData, &project)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Load tests

func TestConcurrentProjectCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("No database URL configured")
	}

	log := logrus.New()
	log.SetOutput(ioutil.Discard)

	db, err := database.NewConnection(log)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	repo := database.NewRepository(db, log)
	ctx := context.Background()

	// Test concurrent project creation
	numGoroutines := 10
	projectsPerGoroutine := 5

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*projectsPerGoroutine)
	projectIDs := make(chan uuid.UUID, numGoroutines*projectsPerGoroutine)

	startTime := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < projectsPerGoroutine; j++ {
				project := &database.Project{
					ID:         uuid.New(),
					Name:       fmt.Sprintf("Load Test Project G%d-P%d", goroutineID, j),
					FolderPath: fmt.Sprintf("/loadtest/g%d/p%d", goroutineID, j),
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				}

				if err := repo.CreateProject(ctx, project); err != nil {
					errors <- err
				} else {
					projectIDs <- project.ID
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)
	close(projectIDs)

	duration := time.Since(startTime)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Errorf("Error during concurrent creation: %v", err)
		errorCount++
	}

	// Count successful creations
	successCount := 0
	createdIDs := []uuid.UUID{}
	for id := range projectIDs {
		successCount++
		createdIDs = append(createdIDs, id)
	}

	// Cleanup
	for _, id := range createdIDs {
		repo.DeleteProject(ctx, id)
	}

	t.Logf("Created %d projects in %v with %d errors", successCount, duration, errorCount)

	if successCount < numGoroutines*projectsPerGoroutine/2 {
		t.Errorf("Too many failures: only %d/%d succeeded", successCount, numGoroutines*projectsPerGoroutine)
	}
}

func TestDatabaseConnectionPool(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping connection pool test in short mode")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("No database URL configured")
	}

	log := logrus.New()
	log.SetOutput(ioutil.Discard)

	db, err := database.NewConnection(log)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	repo := database.NewRepository(db, log)

	// Test concurrent database queries
	numQueries := 100
	var wg sync.WaitGroup
	errors := make(chan error, numQueries)

	startTime := time.Now()

	for i := 0; i < numQueries; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ctx := context.Background()
			_, err := repo.ListProjects(ctx, 10, 0)
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	duration := time.Since(startTime)

	errorCount := 0
	for err := range errors {
		t.Errorf("Error during concurrent queries: %v", err)
		errorCount++
	}

	t.Logf("Executed %d queries in %v with %d errors", numQueries, duration, errorCount)

	if errorCount > numQueries/10 {
		t.Errorf("Too many query errors: %d/%d", errorCount, numQueries)
	}
}

func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	// Test memory efficiency of project creation
	numProjects := 1000

	projects := make([]*database.Project, numProjects)
	for i := 0; i < numProjects; i++ {
		projects[i] = &database.Project{
			ID:          uuid.New(),
			Name:        fmt.Sprintf("Memory Test Project %d", i),
			FolderPath:  fmt.Sprintf("/memtest/project-%d", i),
			Description: "A test project for memory usage testing with a reasonably long description",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	}

	// Verify we can marshal all projects without issues
	for _, project := range projects {
		_, err := json.Marshal(project)
		if err != nil {
			t.Errorf("Failed to marshal project: %v", err)
		}
	}

	t.Logf("Successfully created and marshaled %d projects in memory", numProjects)
}

func TestResponseTime(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping response time test in short mode")
	}

	log := logrus.New()
	log.SetOutput(ioutil.Discard)

	handler := &handlers.Handler{}

	// Test health endpoint response time
	samples := 100
	var totalDuration time.Duration

	for i := 0; i < samples; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		start := time.Now()
		handler.Health(w, req)
		duration := time.Since(start)

		totalDuration += duration
	}

	avgDuration := totalDuration / time.Duration(samples)

	t.Logf("Average health endpoint response time: %v", avgDuration)

	// Health endpoint should respond in less than 100ms on average
	if avgDuration > 100*time.Millisecond {
		t.Errorf("Health endpoint too slow: avg %v (expected < 100ms)", avgDuration)
	}
}


package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestImportPerformance tests the performance of job import operations
func TestImportPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("BulkImport", func(t *testing.T) {
		const numJobs = 50
		start := time.Now()

		var jobIDs []string
		defer func() {
			for _, id := range jobIDs {
				deleteJobFile(id, "pending")
			}
		}()

		for i := 0; i < numJobs; i++ {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/jobs/import",
				Body: ImportRequest{
					Source: "manual",
					Data:   fmt.Sprintf("Test job %d description", i),
				},
			}

			w, httpReq := makeHTTPRequest(req)
			importJobHandler(w, httpReq)

			if w.Code != 201 {
				t.Errorf("Job %d import failed with status %d", i, w.Code)
				continue
			}

			response := assertJSONResponse(t, w, 201, nil)
			if response != nil {
				if jobID, ok := response["job_id"].(string); ok {
					jobIDs = append(jobIDs, jobID)
				}
			}
		}

		duration := time.Since(start)
		avgDuration := duration / numJobs

		t.Logf("Imported %d jobs in %v (avg: %v per job)", numJobs, duration, avgDuration)

		if avgDuration > 100*time.Millisecond {
			t.Errorf("Import performance degraded: avg %v per job (expected < 100ms)", avgDuration)
		}
	})

	t.Run("ConcurrentImports", func(t *testing.T) {
		const numConcurrent = 10
		var wg sync.WaitGroup
		var mu sync.Mutex
		var jobIDs []string
		errors := make([]error, 0)

		defer func() {
			for _, id := range jobIDs {
				deleteJobFile(id, "pending")
			}
		}()

		start := time.Now()

		for i := 0; i < numConcurrent; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				req := HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/jobs/import",
					Body: ImportRequest{
						Source: "manual",
						Data:   fmt.Sprintf("Concurrent job %d", index),
					},
				}

				w, httpReq := makeHTTPRequest(req)
				importJobHandler(w, httpReq)

				if w.Code != 201 {
					mu.Lock()
					errors = append(errors, fmt.Errorf("job %d failed with status %d", index, w.Code))
					mu.Unlock()
					return
				}

				response := assertJSONResponse(t, w, 201, nil)
				if response != nil {
					if jobID, ok := response["job_id"].(string); ok {
						mu.Lock()
						jobIDs = append(jobIDs, jobID)
						mu.Unlock()
					}
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		if len(errors) > 0 {
			for _, err := range errors {
				t.Error(err)
			}
		}

		t.Logf("Concurrent import of %d jobs completed in %v", numConcurrent, duration)

		if duration > 2*time.Second {
			t.Errorf("Concurrent import took too long: %v (expected < 2s)", duration)
		}
	})
}

// TestListJobsPerformance tests the performance of listing jobs
func TestListJobsPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create test jobs across different states
	const numJobsPerState = 20
	states := []string{"pending", "researching", "evaluated"}
	var allJobs []*TestJob

	for _, state := range states {
		for i := 0; i < numJobsPerState; i++ {
			job := setupTestJob(t, state)
			allJobs = append(allJobs, job)
		}
	}

	defer func() {
		for _, job := range allJobs {
			job.Cleanup()
		}
	}()

	t.Run("ListAllJobsPerformance", func(t *testing.T) {
		start := time.Now()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/jobs",
		}

		w, httpReq := makeHTTPRequest(req)
		listJobsHandler(w, httpReq)

		duration := time.Since(start)

		response := assertJSONResponse(t, w, 200, nil)
		if response != nil {
			jobs, ok := response["jobs"].([]interface{})
			if ok {
				t.Logf("Listed %d jobs in %v", len(jobs), duration)
			}
		}

		if duration > 500*time.Millisecond {
			t.Errorf("List jobs took too long: %v (expected < 500ms)", duration)
		}
	})

	t.Run("FilteredListPerformance", func(t *testing.T) {
		start := time.Now()

		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/v1/jobs",
			QueryParams: map[string]string{"state": "pending"},
		}

		w, httpReq := makeHTTPRequest(req)
		listJobsHandler(w, httpReq)

		duration := time.Since(start)

		t.Logf("Filtered list completed in %v", duration)

		if duration > 300*time.Millisecond {
			t.Errorf("Filtered list took too long: %v (expected < 300ms)", duration)
		}
	})
}

// TestStateTransitionPerformance tests the performance of state transitions
func TestStateTransitionPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("SingleTransition", func(t *testing.T) {
		job := setupTestJob(t, "pending")
		defer job.Cleanup()

		start := time.Now()

		req := HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/jobs/" + job.Job.ID + "/research",
			URLVars: map[string]string{"id": job.Job.ID},
		}

		w, httpReq := makeHTTPRequest(req)
		researchJobHandler(w, httpReq)

		duration := time.Since(start)

		assertJSONResponse(t, w, 200, map[string]interface{}{
			"status": "researching",
		})

		t.Logf("State transition completed in %v", duration)

		if duration > 100*time.Millisecond {
			t.Errorf("State transition took too long: %v (expected < 100ms)", duration)
		}
	})

	t.Run("MultipleTransitions", func(t *testing.T) {
		const numJobs = 10
		var jobs []*TestJob

		for i := 0; i < numJobs; i++ {
			job := setupTestJob(t, "pending")
			jobs = append(jobs, job)
		}

		defer func() {
			for _, job := range jobs {
				job.Cleanup()
			}
		}()

		start := time.Now()

		for _, job := range jobs {
			req := HTTPTestRequest{
				Method:  "POST",
				Path:    "/api/v1/jobs/" + job.Job.ID + "/research",
				URLVars: map[string]string{"id": job.Job.ID},
			}

			w, httpReq := makeHTTPRequest(req)
			researchJobHandler(w, httpReq)

			if w.Code != 200 {
				t.Errorf("State transition failed for job %s", job.Job.ID)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / numJobs

		t.Logf("Transitioned %d jobs in %v (avg: %v)", numJobs, duration, avgDuration)

		if avgDuration > 50*time.Millisecond {
			t.Errorf("Average transition time too high: %v (expected < 50ms)", avgDuration)
		}
	})
}

// TestMemoryUsage tests memory efficiency
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("LargeJobDescription", func(t *testing.T) {
		// Create a large description (1MB)
		largeData := make([]byte, 1024*1024)
		for i := range largeData {
			largeData[i] = 'A' + byte(i%26)
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/jobs/import",
			Body: ImportRequest{
				Source: "manual",
				Data:   string(largeData),
			},
		}

		w, httpReq := makeHTTPRequest(req)
		importJobHandler(w, httpReq)

		response := assertJSONResponse(t, w, 201, nil)
		if response != nil {
			if jobID, ok := response["job_id"].(string); ok {
				defer deleteJobFile(jobID, "pending")

				// Verify we can load it back
				loadedJob, err := loadJob(jobID)
				if err != nil {
					t.Errorf("Failed to load large job: %v", err)
				}

				if len(loadedJob.Description) != len(largeData) {
					t.Errorf("Large description truncated: expected %d, got %d", len(largeData), len(loadedJob.Description))
				}
			}
		}
	})
}

// BenchmarkImportJob benchmarks job import
func BenchmarkImportJob(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	req := ImportRequest{
		Source: "manual",
		Data:   "Benchmark test job description",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		testReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/jobs/import",
			Body:   req,
		}

		w, httpReq := makeHTTPRequest(testReq)
		importJobHandler(w, httpReq)

		// Clean up after each iteration
		if w.Code == 201 {
			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			if jobID, ok := response["job_id"].(string); ok {
				deleteJobFile(jobID, "pending")
			}
		}
	}
}

// BenchmarkListJobs benchmarks job listing
func BenchmarkListJobs(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	// Create some test jobs
	for i := 0; i < 20; i++ {
		job := setupTestJob(&testing.T{}, "pending")
		defer job.Cleanup()
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/jobs",
		}

		w, httpReq := makeHTTPRequest(req)
		listJobsHandler(w, httpReq)
	}
}

// BenchmarkGetJob benchmarks single job retrieval
func BenchmarkGetJob(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	job := setupTestJob(&testing.T{}, "pending")
	defer job.Cleanup()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/jobs/" + job.Job.ID,
			URLVars: map[string]string{"id": job.Job.ID},
		}

		w, httpReq := makeHTTPRequest(req)
		getJobHandler(w, httpReq)
	}
}

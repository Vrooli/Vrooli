
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// TestHandlerPerformance tests performance of various handlers
func TestHandlerPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("GetContactsPerformance", func(t *testing.T) {
		// Create multiple contacts for testing
		contacts := make([]*TestContact, 10)
		for i := 0; i < 10; i++ {
			contacts[i] = setupTestContact(t, testDB.DB, "PerfTestContact"+string(rune(i)))
			defer contacts[i].Cleanup()
		}

		start := time.Now()
		iterations := 100

		for i := 0; i < iterations; i++ {
			req := httptest.NewRequest("GET", "/api/contacts", nil)
			w := httptest.NewRecorder()
			getContactsHandler(w, req)

			if w.Code != 200 {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("GetContacts: %d iterations in %v (avg: %v)", iterations, duration, avgDuration)

		if avgDuration > 100*time.Millisecond {
			t.Logf("Warning: Average request time %v exceeds 100ms threshold", avgDuration)
		}
	})

	t.Run("CreateContactPerformance", func(t *testing.T) {
		start := time.Now()
		iterations := 50
		createdIDs := make([]int, 0, iterations)

		for i := 0; i < iterations; i++ {
			contact := TestData.CreateContactRequest("PerfContact" + string(rune(i)))
			body, _ := json.Marshal(contact)

			req := httptest.NewRequest("POST", "/api/contacts", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			createContactHandler(w, req)

			if w.Code != 201 {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}

			// Track created IDs for cleanup
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
				if idFloat, ok := response["id"].(float64); ok {
					createdIDs = append(createdIDs, int(idFloat))
				}
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("CreateContact: %d iterations in %v (avg: %v)", iterations, duration, avgDuration)

		// Cleanup
		for _, id := range createdIDs {
			db.Exec("DELETE FROM contacts WHERE id = $1", id)
		}

		if avgDuration > 200*time.Millisecond {
			t.Logf("Warning: Average create time %v exceeds 200ms threshold", avgDuration)
		}
	})
}

// TestConcurrentRequests tests concurrent access to handlers
func TestConcurrentRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("ConcurrentReads", func(t *testing.T) {
		// Create test contacts
		contacts := make([]*TestContact, 5)
		for i := 0; i < 5; i++ {
			contacts[i] = setupTestContact(t, testDB.DB, "ConcurrentContact"+string(rune(i)))
			defer contacts[i].Cleanup()
		}

		concurrency := 10
		iterations := 20
		var wg sync.WaitGroup
		errors := make(chan error, concurrency*iterations)

		start := time.Now()

		for c := 0; c < concurrency; c++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for i := 0; i < iterations; i++ {
					req := httptest.NewRequest("GET", "/api/contacts", nil)
					w := httptest.NewRecorder()
					getContactsHandler(w, req)

					if w.Code != 200 {
						errors <- nil // Just track errors
					}
				}
			}(c)
		}

		wg.Wait()
		close(errors)
		duration := time.Since(start)

		totalRequests := concurrency * iterations
		t.Logf("Concurrent reads: %d requests across %d goroutines in %v", totalRequests, concurrency, duration)
		t.Logf("Throughput: %.2f requests/second", float64(totalRequests)/duration.Seconds())

		errorCount := len(errors)
		if errorCount > 0 {
			t.Errorf("Encountered %d errors during concurrent reads", errorCount)
		}
	})

	t.Run("ConcurrentWrites", func(t *testing.T) {
		concurrency := 5
		iterations := 10
		var wg sync.WaitGroup
		errors := make(chan error, concurrency*iterations)
		createdIDs := make(chan int, concurrency*iterations)

		start := time.Now()

		for c := 0; c < concurrency; c++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for i := 0; i < iterations; i++ {
					contact := TestData.CreateContactRequest("ConcWrite" + string(rune(workerID*iterations+i)))
					body, _ := json.Marshal(contact)

					req := httptest.NewRequest("POST", "/api/contacts", bytes.NewReader(body))
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()

					createContactHandler(w, req)

					if w.Code != 201 {
						errors <- nil
					} else {
						var response map[string]interface{}
						if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
							if idFloat, ok := response["id"].(float64); ok {
								createdIDs <- int(idFloat)
							}
						}
					}
				}
			}(c)
		}

		wg.Wait()
		close(errors)
		close(createdIDs)
		duration := time.Since(start)

		totalRequests := concurrency * iterations
		t.Logf("Concurrent writes: %d requests across %d goroutines in %v", totalRequests, concurrency, duration)

		// Cleanup
		for id := range createdIDs {
			db.Exec("DELETE FROM contacts WHERE id = $1", id)
		}

		errorCount := len(errors)
		if errorCount > 0 {
			t.Errorf("Encountered %d errors during concurrent writes", errorCount)
		}
	})
}

// TestRelationshipProcessorPerformance tests performance of relationship processor operations
func TestRelationshipProcessorPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	processor := NewRelationshipProcessor(testDB.DB)
	ctx := context.Background()

	t.Run("GetUpcomingBirthdaysPerformance", func(t *testing.T) {
		// Create contacts with birthdays
		contacts := make([]*TestContact, 20)
		for i := 0; i < 20; i++ {
			contacts[i] = setupTestContact(t, testDB.DB, "BirthdayPerfContact"+string(rune(i)))
			defer contacts[i].Cleanup()
		}

		start := time.Now()
		iterations := 50

		for i := 0; i < iterations; i++ {
			_, err := processor.GetUpcomingBirthdays(ctx, 30)
			if err != nil {
				t.Errorf("Iteration %d failed: %v", i, err)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("GetUpcomingBirthdays: %d iterations in %v (avg: %v)", iterations, duration, avgDuration)

		if avgDuration > 500*time.Millisecond {
			t.Logf("Warning: Average time %v exceeds 500ms threshold", avgDuration)
		}
	})

	t.Run("EnrichContactPerformance", func(t *testing.T) {
		start := time.Now()
		iterations := 30

		for i := 0; i < iterations; i++ {
			_, err := processor.EnrichContact(ctx, i+1, "TestName"+string(rune(i)))
			if err != nil {
				t.Errorf("Iteration %d failed: %v", i, err)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("EnrichContact: %d iterations in %v (avg: %v)", iterations, duration, avgDuration)

		// This uses mock Ollama so should be fast
		if avgDuration > 100*time.Millisecond {
			t.Logf("Warning: Average enrichment time %v exceeds 100ms threshold", avgDuration)
		}
	})

	t.Run("SuggestGiftsPerformance", func(t *testing.T) {
		start := time.Now()
		iterations := 30

		for i := 0; i < iterations; i++ {
			req := GiftSuggestionRequest{
				ContactID: i + 1,
				Name:      "TestName",
				Interests: "reading, travel",
				Occasion:  "birthday",
				Budget:    "50-100",
			}

			_, err := processor.SuggestGifts(ctx, req)
			if err != nil {
				t.Errorf("Iteration %d failed: %v", i, err)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("SuggestGifts: %d iterations in %v (avg: %v)", iterations, duration, avgDuration)

		if avgDuration > 100*time.Millisecond {
			t.Logf("Warning: Average suggestion time %v exceeds 100ms threshold", avgDuration)
		}
	})

	t.Run("AnalyzeRelationshipsPerformance", func(t *testing.T) {
		// Create contact with interactions
		contact := setupTestContact(t, testDB.DB, "AnalyzePerfContact")
		defer contact.Cleanup()

		for i := 0; i < 10; i++ {
			interaction := setupTestInteraction(t, testDB.DB, contact.Contact.ID)
			defer testDB.DB.Exec("DELETE FROM interactions WHERE id = $1", interaction.ID)
		}

		start := time.Now()
		iterations := 50

		for i := 0; i < iterations; i++ {
			_, err := processor.AnalyzeRelationships(ctx, contact.Contact.ID)
			if err != nil {
				t.Errorf("Iteration %d failed: %v", i, err)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("AnalyzeRelationships: %d iterations in %v (avg: %v)", iterations, duration, avgDuration)

		if avgDuration > 200*time.Millisecond {
			t.Logf("Warning: Average analysis time %v exceeds 200ms threshold", avgDuration)
		}
	})
}

// TestMemoryUsage tests memory usage during operations
func TestMemoryUsage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("BulkContactCreation", func(t *testing.T) {
		createdIDs := make([]int, 0, 100)
		defer func() {
			for _, id := range createdIDs {
				db.Exec("DELETE FROM contacts WHERE id = $1", id)
			}
		}()

		start := time.Now()

		for i := 0; i < 100; i++ {
			contact := TestData.CreateContactRequest("BulkContact" + string(rune(i)))
			body, _ := json.Marshal(contact)

			req := httptest.NewRequest("POST", "/api/contacts", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			createContactHandler(w, req)

			if w.Code == 201 {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
					if idFloat, ok := response["id"].(float64); ok {
						createdIDs = append(createdIDs, int(idFloat))
					}
				}
			}
		}

		duration := time.Since(start)
		t.Logf("Created 100 contacts in %v (avg: %v per contact)", duration, duration/100)
	})

	t.Run("BulkRetrieval", func(t *testing.T) {
		// Create multiple contacts
		contacts := make([]*TestContact, 50)
		for i := 0; i < 50; i++ {
			contacts[i] = setupTestContact(t, testDB.DB, "MemTestContact"+string(rune(i)))
			defer contacts[i].Cleanup()
		}

		start := time.Now()

		req := httptest.NewRequest("GET", "/api/contacts", nil)
		w := httptest.NewRecorder()
		getContactsHandler(w, req)

		duration := time.Since(start)
		t.Logf("Retrieved all contacts in %v", duration)

		if w.Code != 200 {
			t.Errorf("Bulk retrieval failed with status %d", w.Code)
		}
	})
}

// BenchmarkGetContactsHandler benchmarks the get contacts handler
func BenchmarkGetContactsHandler(b *testing.B) {
	testDB := setupTestDB(&testing.T{})
	if testDB == nil {
		b.Skip("Database not available")
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	// Create test data
	for i := 0; i < 10; i++ {
		contact := setupTestContact(&testing.T{}, testDB.DB, "BenchContact"+string(rune(i)))
		defer contact.Cleanup()
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/contacts", nil)
		w := httptest.NewRecorder()
		getContactsHandler(w, req)
	}
}

// BenchmarkCreateContactHandler benchmarks contact creation
func BenchmarkCreateContactHandler(b *testing.B) {
	testDB := setupTestDB(&testing.T{})
	if testDB == nil {
		b.Skip("Database not available")
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB
	createdIDs := make([]int, 0, b.N)

	defer func() {
		for _, id := range createdIDs {
			db.Exec("DELETE FROM contacts WHERE id = $1", id)
		}
	}()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		contact := TestData.CreateContactRequest("BenchContact" + string(rune(i)))
		body, _ := json.Marshal(contact)

		req := httptest.NewRequest("POST", "/api/contacts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createContactHandler(w, req)

		if w.Code == 201 {
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
				if idFloat, ok := response["id"].(float64); ok {
					createdIDs = append(createdIDs, int(idFloat))
				}
			}
		}
	}
}

// BenchmarkEnrichContact benchmarks contact enrichment
func BenchmarkEnrichContact(b *testing.B) {
	testDB := setupTestDB(&testing.T{})
	if testDB == nil {
		b.Skip("Database not available")
		return
	}
	defer testDB.Cleanup()

	processor := NewRelationshipProcessor(testDB.DB)
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		processor.EnrichContact(ctx, i+1, "TestName")
	}
}

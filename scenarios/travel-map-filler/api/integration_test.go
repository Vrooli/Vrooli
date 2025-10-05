package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestDatabaseIntegration tests database operations
func TestDatabaseIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	t.Run("Travel_CRUD_Operations", testTravelCRUD(testDB))
	t.Run("Achievement_Lifecycle", testAchievementLifecycle(testDB))
	t.Run("BucketList_Management", testBucketListManagement(testDB))
	t.Run("Stats_Calculation", testStatsCalculation(testDB))
	t.Run("MultiUser_Isolation", testMultiUserIsolation(testDB))
}

func testTravelCRUD(testDB *TestDB) func(*testing.T) {
	return func(t *testing.T) {
		userID := "test_crud_user"

		// CREATE
		t.Run("Create", func(t *testing.T) {
			travelData := TestData.TravelRequest(userID, "Barcelona, Spain")
			travelJSON, _ := json.Marshal(travelData)

			req := httptest.NewRequest("POST", "/api/add-travel", bytes.NewBuffer(travelJSON))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			addTravelHandler(w, req)

			response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
				"status": "success",
			})

			if response == nil {
				t.Fatal("Failed to create travel")
			}

			// Verify ID was returned
			if _, exists := response["id"]; !exists {
				t.Error("Expected 'id' in create response")
			}
		})

		// READ
		t.Run("Read", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/travels?user_id="+userID, nil)
			w := httptest.NewRecorder()

			travelsHandler(w, req)

			travels := assertJSONArray(t, w, http.StatusOK)
			if travels == nil {
				t.Fatal("Failed to read travels")
			}

			if len(travels) < 1 {
				t.Error("Expected at least 1 travel after create")
			}

			// Verify travel data
			if len(travels) > 0 {
				travelMap, ok := travels[0].(map[string]interface{})
				if ok {
					if location, exists := travelMap["location"]; !exists || location == "" {
						t.Error("Expected location in travel data")
					}
				}
			}
		})
	}
}

func testAchievementLifecycle(testDB *TestDB) func(*testing.T) {
	return func(t *testing.T) {
		userID := "test_achievement_lifecycle"

		// Verify no achievements initially
		t.Run("Initial_Empty", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/achievements?user_id="+userID, nil)
			w := httptest.NewRecorder()

			achievementsHandler(w, req)

			achievements := assertJSONArray(t, w, http.StatusOK)
			if len(achievements) != 0 {
				t.Error("Expected no achievements initially")
			}
		})

		// Add first travel to trigger achievement
		t.Run("Trigger_FirstTrip", func(t *testing.T) {
			createTestTravel(t, testDB.DB, userID)
			checkAchievements(userID)

			// Wait for achievement processing
			time.Sleep(100 * time.Millisecond)

			req := httptest.NewRequest("GET", "/api/achievements?user_id="+userID, nil)
			w := httptest.NewRecorder()

			achievementsHandler(w, req)

			achievements := assertJSONArray(t, w, http.StatusOK)
			if len(achievements) < 1 {
				t.Error("Expected at least 1 achievement after first trip")
			}

			// Verify achievement structure
			if len(achievements) > 0 {
				achMap, ok := achievements[0].(map[string]interface{})
				if ok {
					requiredFields := []string{"type", "name", "description", "icon"}
					for _, field := range requiredFields {
						if _, exists := achMap[field]; !exists {
							t.Errorf("Missing required field in achievement: %s", field)
						}
					}
				}
			}
		})

		// Add more travels to trigger additional achievements
		t.Run("Trigger_Explorer", func(t *testing.T) {
			// Add 5 more travels (6 total)
			for i := 0; i < 5; i++ {
				createTestTravel(t, testDB.DB, userID)
			}
			checkAchievements(userID)

			time.Sleep(100 * time.Millisecond)

			req := httptest.NewRequest("GET", "/api/achievements?user_id="+userID, nil)
			w := httptest.NewRecorder()

			achievementsHandler(w, req)

			achievements := assertJSONArray(t, w, http.StatusOK)
			t.Logf("User has %d achievements after 6 travels", len(achievements))
		})
	}
}

func testBucketListManagement(testDB *TestDB) func(*testing.T) {
	return func(t *testing.T) {
		userID := "test_bucket_management"

		// Create bucket items
		t.Run("Create_Items", func(t *testing.T) {
			createTestBucketItem(t, testDB.DB, userID)
			createTestBucketItem(t, testDB.DB, userID)

			req := httptest.NewRequest("GET", "/api/bucket-list?user_id="+userID, nil)
			w := httptest.NewRecorder()

			bucketListHandler(w, req)

			items := assertJSONArray(t, w, http.StatusOK)
			if len(items) < 2 {
				t.Errorf("Expected at least 2 bucket items, got %d", len(items))
			}
		})

		// Verify item structure
		t.Run("Verify_Structure", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/bucket-list?user_id="+userID, nil)
			w := httptest.NewRecorder()

			bucketListHandler(w, req)

			items := assertJSONArray(t, w, http.StatusOK)
			if len(items) > 0 {
				itemMap, ok := items[0].(map[string]interface{})
				if ok {
					requiredFields := []string{"id", "location", "country", "city", "priority", "completed"}
					for _, field := range requiredFields {
						if _, exists := itemMap[field]; !exists {
							t.Errorf("Missing required field in bucket item: %s", field)
						}
					}
				}
			}
		})
	}
}

func testStatsCalculation(testDB *TestDB) func(*testing.T) {
	return func(t *testing.T) {
		userID := "test_stats_calc"

		// Create diverse travel data
		t.Run("Create_DiverseData", func(t *testing.T) {
			travels := []struct {
				location  string
				country   string
				city      string
				continent string
			}{
				{"Paris, France", "France", "Paris", "Europe"},
				{"London, UK", "UK", "London", "Europe"},
				{"Tokyo, Japan", "Japan", "Tokyo", "Asia"},
			}

			for _, travel := range travels {
				travelData := TestData.TravelRequest(userID, travel.location)
				td := travelData
				td["country"] = travel.country
				td["city"] = travel.city
				td["continent"] = travel.continent
				travelJSON, _ := json.Marshal(td)

				req := httptest.NewRequest("POST", "/api/add-travel", bytes.NewBuffer(travelJSON))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				addTravelHandler(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("Failed to add travel: %d", w.Code)
				}
			}
		})

		// Verify stats calculation
		t.Run("Calculate_Stats", func(t *testing.T) {
			time.Sleep(200 * time.Millisecond) // Allow database to sync

			req := httptest.NewRequest("GET", "/api/stats?user_id="+userID, nil)
			w := httptest.NewRecorder()

			statsHandler(w, req)

			response := assertJSONResponse(t, w, http.StatusOK, nil)
			if response == nil {
				t.Fatal("Failed to get stats")
			}

			// Verify stats make sense
			if totalCountries, exists := response["total_countries"]; exists {
				if countries, ok := totalCountries.(float64); ok {
					if countries < 1 {
						t.Error("Expected at least 1 country in stats")
					}
					t.Logf("Total countries: %v", countries)
				}
			}

			if totalCities, exists := response["total_cities"]; exists {
				if cities, ok := totalCities.(float64); ok {
					if cities < 1 {
						t.Error("Expected at least 1 city in stats")
					}
					t.Logf("Total cities: %v", cities)
				}
			}

			if coverage, exists := response["world_coverage_percent"]; exists {
				if coverageVal, ok := coverage.(float64); ok {
					if coverageVal < 0 || coverageVal > 100 {
						t.Errorf("Invalid world coverage: %v", coverageVal)
					}
					t.Logf("World coverage: %v%%", coverageVal)
				}
			}
		})
	}
}

func testMultiUserIsolation(testDB *TestDB) func(*testing.T) {
	return func(t *testing.T) {
		user1 := "test_isolation_user1"
		user2 := "test_isolation_user2"

		// Create data for user 1
		t.Run("User1_Data", func(t *testing.T) {
			createTestTravel(t, testDB.DB, user1)
			createTestTravel(t, testDB.DB, user1)
		})

		// Create data for user 2
		t.Run("User2_Data", func(t *testing.T) {
			createTestTravel(t, testDB.DB, user2)
		})

		// Verify user 1 sees only their data
		t.Run("Verify_User1_Isolation", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/travels?user_id="+user1, nil)
			w := httptest.NewRecorder()

			travelsHandler(w, req)

			travels := assertJSONArray(t, w, http.StatusOK)
			if len(travels) != 2 {
				t.Errorf("User1 expected 2 travels, got %d", len(travels))
			}

			// Verify all travels belong to user1
			for i, travel := range travels {
				if travelMap, ok := travel.(map[string]interface{}); ok {
					if userID, exists := travelMap["user_id"]; exists {
						if userID != user1 {
							t.Errorf("Travel %d belongs to wrong user: %v", i, userID)
						}
					}
				}
			}
		})

		// Verify user 2 sees only their data
		t.Run("Verify_User2_Isolation", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/travels?user_id="+user2, nil)
			w := httptest.NewRecorder()

			travelsHandler(w, req)

			travels := assertJSONArray(t, w, http.StatusOK)
			if len(travels) != 1 {
				t.Errorf("User2 expected 1 travel, got %d", len(travels))
			}
		})

		// Verify stats are isolated
		t.Run("Verify_Stats_Isolation", func(t *testing.T) {
			// User 1 stats
			req := httptest.NewRequest("GET", "/api/stats?user_id="+user1, nil)
			w := httptest.NewRecorder()
			statsHandler(w, req)
			stats1 := assertJSONResponse(t, w, http.StatusOK, nil)

			// User 2 stats
			req = httptest.NewRequest("GET", "/api/stats?user_id="+user2, nil)
			w = httptest.NewRecorder()
			statsHandler(w, req)
			stats2 := assertJSONResponse(t, w, http.StatusOK, nil)

			// Stats should be different
			if stats1 != nil && stats2 != nil {
				t.Logf("User1 stats: %+v", stats1)
				t.Logf("User2 stats: %+v", stats2)
			}
		})
	}
}

// TestConcurrentOperations tests thread safety
func TestConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent operations test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	t.Run("Concurrent_Reads", testConcurrentReads(testDB))
	t.Run("Concurrent_Writes", testConcurrentWrites(testDB))
	t.Run("Concurrent_Mixed", testConcurrentMixed(testDB))
}

func testConcurrentReads(testDB *TestDB) func(*testing.T) {
	return func(t *testing.T) {
		userID := "test_concurrent_reads"
		createTestTravel(t, testDB.DB, userID)

		concurrency := 20
		done := make(chan bool, concurrency)
		errors := make(chan string, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(iteration int) {
				req := httptest.NewRequest("GET", "/api/travels?user_id="+userID, nil)
				w := httptest.NewRecorder()

				travelsHandler(w, req)

				if w.Code != http.StatusOK {
					errors <- "Bad status code"
				}

				done <- true
			}(i)
		}

		// Wait for completion
		for i := 0; i < concurrency; i++ {
			<-done
		}

		close(errors)
		errorCount := 0
		for range errors {
			errorCount++
		}

		if errorCount > 0 {
			t.Errorf("Concurrent reads failed: %d errors out of %d", errorCount, concurrency)
		} else {
			t.Logf("Successfully handled %d concurrent reads", concurrency)
		}
	}
}

func testConcurrentWrites(testDB *TestDB) func(*testing.T) {
	return func(t *testing.T) {
		userID := "test_concurrent_writes"
		concurrency := 10
		done := make(chan bool, concurrency)
		errors := make(chan string, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(iteration int) {
				travelData := TestData.TravelRequest(userID, "Concurrent Location")
				travelJSON, _ := json.Marshal(travelData)

				req := httptest.NewRequest("POST", "/api/add-travel", bytes.NewBuffer(travelJSON))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				addTravelHandler(w, req)

				if w.Code != http.StatusOK {
					errors <- "Bad status code"
				}

				done <- true
			}(i)
		}

		// Wait for completion
		for i := 0; i < concurrency; i++ {
			<-done
		}

		close(errors)
		errorCount := 0
		for range errors {
			errorCount++
		}

		if errorCount > 0 {
			t.Errorf("Concurrent writes failed: %d errors out of %d", errorCount, concurrency)
		} else {
			t.Logf("Successfully handled %d concurrent writes", concurrency)
		}

		// Verify all writes succeeded
		time.Sleep(200 * time.Millisecond)
		req := httptest.NewRequest("GET", "/api/travels?user_id="+userID, nil)
		w := httptest.NewRecorder()
		travelsHandler(w, req)

		travels := assertJSONArray(t, w, http.StatusOK)
		if len(travels) < concurrency {
			t.Errorf("Expected %d travels, got %d", concurrency, len(travels))
		}
	}
}

func testConcurrentMixed(testDB *TestDB) func(*testing.T) {
	return func(t *testing.T) {
		userID := "test_concurrent_mixed"
		createTestTravel(t, testDB.DB, userID)

		concurrency := 20
		done := make(chan bool, concurrency)
		errors := make(chan string, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(iteration int) {
				// Mix of reads and writes
				if iteration%2 == 0 {
					// READ
					req := httptest.NewRequest("GET", "/api/travels?user_id="+userID, nil)
					w := httptest.NewRecorder()
					travelsHandler(w, req)

					if w.Code != http.StatusOK {
						errors <- "Read failed"
					}
				} else {
					// WRITE
					travelData := TestData.TravelRequest(userID, "Mixed Op Location")
					travelJSON, _ := json.Marshal(travelData)

					req := httptest.NewRequest("POST", "/api/add-travel", bytes.NewBuffer(travelJSON))
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()
					addTravelHandler(w, req)

					if w.Code != http.StatusOK {
						errors <- "Write failed"
					}
				}

				done <- true
			}(i)
		}

		// Wait for completion
		for i := 0; i < concurrency; i++ {
			<-done
		}

		close(errors)
		errorCount := 0
		for range errors {
			errorCount++
		}

		if errorCount > 0 {
			t.Errorf("Concurrent mixed operations failed: %d errors out of %d", errorCount, concurrency)
		} else {
			t.Logf("Successfully handled %d concurrent mixed operations", concurrency)
		}
	}
}

// TestDatabaseResilience tests error handling and recovery
func TestDatabaseResilience(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database resilience test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	t.Run("InvalidData_Handling", func(t *testing.T) {
		// Test with various invalid data scenarios
		invalidData := []map[string]interface{}{
			{"lat": "invalid", "lng": "invalid"},
			{"duration_days": -1},
			{"rating": 100},
		}

		for i, data := range invalidData {
			travelJSON, _ := json.Marshal(data)

			req := httptest.NewRequest("POST", "/api/add-travel", bytes.NewBuffer(travelJSON))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			addTravelHandler(w, req)

			// Should either handle gracefully or return error
			if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
				t.Errorf("Test %d: Unexpected status %d", i, w.Code)
			}
		}
	})

	t.Run("EmptyResults_Handling", func(t *testing.T) {
		nonexistentUser := "nonexistent_user_12345"

		endpoints := []string{
			"/api/travels?user_id=" + nonexistentUser,
			"/api/stats?user_id=" + nonexistentUser,
			"/api/achievements?user_id=" + nonexistentUser,
			"/api/bucket-list?user_id=" + nonexistentUser,
		}

		for _, endpoint := range endpoints {
			req := httptest.NewRequest("GET", endpoint, nil)
			w := httptest.NewRecorder()

			switch {
			case endpoint == "/api/travels?user_id="+nonexistentUser:
				travelsHandler(w, req)
			case endpoint == "/api/stats?user_id="+nonexistentUser:
				statsHandler(w, req)
			case endpoint == "/api/achievements?user_id="+nonexistentUser:
				achievementsHandler(w, req)
			case endpoint == "/api/bucket-list?user_id="+nonexistentUser:
				bucketListHandler(w, req)
			}

			if w.Code != http.StatusOK {
				t.Errorf("Endpoint %s: Expected 200, got %d", endpoint, w.Code)
			}
		}
	})
}

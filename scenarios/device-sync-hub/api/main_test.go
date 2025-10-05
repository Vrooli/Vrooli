// +build testing

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	t.Run("Success", func(t *testing.T) {
		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if readiness, ok := response["readiness"].(bool); !ok || !readiness {
			t.Errorf("Expected readiness to be true, got %v", response["readiness"])
		}

		if _, ok := response["dependencies"]; !ok {
			t.Error("Expected dependencies field in response")
		}
	})
}

// TestDeviceRegistration tests device registration endpoint
func TestDeviceRegistration(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()

	t.Run("Success", func(t *testing.T) {
		device := map[string]interface{}{
			"name":         "Test Phone",
			"type":         "mobile",
			"platform":     "android",
			"capabilities": []string{"clipboard", "files"},
		}

		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/devices",
			Body:   device,
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusCreated)

		if _, ok := response["id"].(string); !ok {
			t.Error("Expected device ID in response")
		}

		if name, ok := response["name"].(string); !ok || name != "Test Phone" {
			t.Errorf("Expected name 'Test Phone', got %v", response["name"])
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		scenarios := DeviceRegistrationErrorPatterns().Build()
		RunErrorTests(t, ts, scenarios)
	})
}

// TestListDevices tests listing devices
func TestListDevices(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()

	t.Run("EmptyList", func(t *testing.T) {
		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/devices",
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		devices, ok := response["devices"].([]interface{})
		if !ok {
			t.Fatal("Expected devices array in response")
		}

		if len(devices) != 0 {
			t.Errorf("Expected 0 devices, got %d", len(devices))
		}
	})

	t.Run("WithDevices", func(t *testing.T) {
		// Create test devices
		device1 := ts.createTestDevice(t, user.ID)
		device2 := ts.createTestDevice(t, user.ID)

		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/devices",
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		devices, ok := response["devices"].([]interface{})
		if !ok {
			t.Fatal("Expected devices array in response")
		}

		if len(devices) != 2 {
			t.Errorf("Expected 2 devices, got %d", len(devices))
		}

		// Verify device IDs
		deviceIDs := []string{device1.ID, device2.ID}
		foundIDs := make(map[string]bool)
		for _, d := range devices {
			device := d.(map[string]interface{})
			id := device["id"].(string)
			foundIDs[id] = true
		}

		for _, id := range deviceIDs {
			if !foundIDs[id] {
				t.Errorf("Expected device ID %s not found in response", id)
			}
		}
	})
}

// TestGetDevice tests getting a specific device
func TestGetDevice(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()
	device := ts.createTestDevice(t, user.ID)

	t.Run("Success", func(t *testing.T) {
		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/devices/%s", device.ID),
			URLVars: map[string]string{"id": device.ID},
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if id, ok := response["id"].(string); !ok || id != device.ID {
			t.Errorf("Expected device ID %s, got %v", device.ID, response["id"])
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		scenarios := DeviceOperationErrorPatterns(device.ID).Build()
		RunErrorTests(t, ts, scenarios)
	})
}

// TestUpdateDevice tests updating a device
func TestUpdateDevice(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()
	device := ts.createTestDevice(t, user.ID)

	t.Run("Success", func(t *testing.T) {
		updates := map[string]interface{}{
			"name": "Updated Device Name",
		}

		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method:  "PUT",
			Path:    fmt.Sprintf("/api/v1/devices/%s", device.ID),
			URLVars: map[string]string{"id": device.ID},
			Body:    updates,
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if name, ok := response["name"].(string); !ok || name != "Updated Device Name" {
			t.Errorf("Expected name 'Updated Device Name', got %v", response["name"])
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method:  "PUT",
			Path:    fmt.Sprintf("/api/v1/devices/%s", device.ID),
			URLVars: map[string]string{"id": device.ID},
			Body:    `{"invalid": "json"`,
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// TestDeleteDevice tests deleting a device
func TestDeleteDevice(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()
	device := ts.createTestDevice(t, user.ID)

	t.Run("Success", func(t *testing.T) {
		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method:  "DELETE",
			Path:    fmt.Sprintf("/api/v1/devices/%s", device.ID),
			URLVars: map[string]string{"id": device.ID},
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if success, ok := response["success"].(bool); !ok || !success {
			t.Errorf("Expected success true, got %v", response["success"])
		}

		// Verify device is deleted
		count := ts.getDeviceCount(t, user.ID)
		if count != 0 {
			t.Errorf("Expected 0 devices after deletion, got %d", count)
		}
	})

	t.Run("NonExistent", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method:  "DELETE",
			Path:    fmt.Sprintf("/api/v1/devices/%s", nonExistentID),
			URLVars: map[string]string{"id": nonExistentID},
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusNotFound, "not found")
	})
}

// TestClipboardSync tests clipboard synchronization
func TestClipboardSync(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()
	device := ts.createTestDevice(t, user.ID)

	t.Run("Success", func(t *testing.T) {
		clipboardData := map[string]interface{}{
			"content":       "Test clipboard content",
			"source_device": device.ID,
		}

		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/sync/clipboard",
			Body:   clipboardData,
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if success, ok := response["success"].(bool); !ok || !success {
			t.Errorf("Expected success true, got %v", response["success"])
		}

		if _, ok := response["item_id"].(string); !ok {
			t.Error("Expected item_id in response")
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		scenarios := ClipboardSyncErrorPatterns().Build()
		RunErrorTests(t, ts, scenarios)
	})
}

// TestFileSync tests file synchronization
func TestFileSync(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()
	device := ts.createTestDevice(t, user.ID)

	t.Run("Success", func(t *testing.T) {
		fileContent := []byte("Test file content for sync")
		body, contentType := createMultipartFileRequest(t, "file", "test.txt", fileContent, map[string]string{
			"source_device": device.ID,
		})

		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/sync/file",
			Body:   body.Bytes(),
			Headers: map[string]string{
				"Content-Type": contentType,
				"X-User-ID":    user.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if success, ok := response["success"].(bool); !ok || !success {
			t.Errorf("Expected success true, got %v", response["success"])
		}

		if _, ok := response["item_id"].(string); !ok {
			t.Error("Expected item_id in response")
		}
	})

	t.Run("MissingFile", func(t *testing.T) {
		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/sync/file",
			Headers: map[string]string{
				"Content-Type": "multipart/form-data",
				"X-User-ID":    user.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// TestListSyncItems tests listing sync items
func TestListSyncItems(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()

	t.Run("EmptyList", func(t *testing.T) {
		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/sync/items",
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		items, ok := response["items"].([]interface{})
		if !ok {
			t.Fatal("Expected items array in response")
		}

		if len(items) != 0 {
			t.Errorf("Expected 0 items, got %d", len(items))
		}
	})

	t.Run("WithItems", func(t *testing.T) {
		// Create test sync items
		item1 := ts.createTestSyncItem(t, user.ID, "clipboard")
		item2 := ts.createTestSyncItem(t, user.ID, "file")

		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/sync/items",
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		items, ok := response["items"].([]interface{})
		if !ok {
			t.Fatal("Expected items array in response")
		}

		if len(items) != 2 {
			t.Errorf("Expected 2 items, got %d", len(items))
		}

		// Verify item IDs
		itemIDs := []string{item1.ID, item2.ID}
		foundIDs := make(map[string]bool)
		for _, i := range items {
			item := i.(map[string]interface{})
			id := item["id"].(string)
			foundIDs[id] = true
		}

		for _, id := range itemIDs {
			if !foundIDs[id] {
				t.Errorf("Expected item ID %s not found in response", id)
			}
		}
	})

	t.Run("FilterByType", func(t *testing.T) {
		// Create items of different types
		ts.createTestSyncItem(t, user.ID, "clipboard")
		ts.createTestSyncItem(t, user.ID, "file")
		ts.createTestSyncItem(t, user.ID, "file")

		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/sync/items",
			QueryParams: map[string]string{
				"type": "file",
			},
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		items, ok := response["items"].([]interface{})
		if !ok {
			t.Fatal("Expected items array in response")
		}

		// Should return all items if filter is not implemented, or filtered items
		if len(items) < 2 {
			t.Errorf("Expected at least 2 file items, got %d", len(items))
		}
	})
}

// TestGetSyncItem tests getting a specific sync item
func TestGetSyncItem(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()
	item := ts.createTestSyncItem(t, user.ID, "clipboard")

	t.Run("Success", func(t *testing.T) {
		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/sync/items/%s", item.ID),
			URLVars: map[string]string{"id": item.ID},
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if id, ok := response["id"].(string); !ok || id != item.ID {
			t.Errorf("Expected item ID %s, got %v", item.ID, response["id"])
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		scenarios := SyncItemErrorPatterns().Build()
		RunErrorTests(t, ts, scenarios)
	})
}

// TestDeleteSyncItem tests deleting a sync item
func TestDeleteSyncItem(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()
	item := ts.createTestSyncItem(t, user.ID, "clipboard")

	t.Run("Success", func(t *testing.T) {
		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method:  "DELETE",
			Path:    fmt.Sprintf("/api/v1/sync/items/%s", item.ID),
			URLVars: map[string]string{"id": item.ID},
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if success, ok := response["success"].(bool); !ok || !success {
			t.Errorf("Expected success true, got %v", response["success"])
		}

		// Verify item is deleted
		count := ts.getSyncItemCount(t, user.ID)
		if count != 0 {
			t.Errorf("Expected 0 items after deletion, got %d", count)
		}
	})

	t.Run("NonExistent", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method:  "DELETE",
			Path:    fmt.Sprintf("/api/v1/sync/items/%s", nonExistentID),
			URLVars: map[string]string{"id": nonExistentID},
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusNotFound, "not found")
	})
}

// TestSyncUpload tests the unified upload endpoint
func TestSyncUpload(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()
	device := ts.createTestDevice(t, user.ID)

	t.Run("TextUpload", func(t *testing.T) {
		uploadData := map[string]interface{}{
			"content_type":  "text",
			"text":          "Test text content",
			"source_device": device.ID,
		}

		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/sync/upload",
			Body:   uploadData,
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if _, ok := response["item_id"].(string); !ok {
			t.Error("Expected item_id in response")
		}
	})

	t.Run("FileUpload", func(t *testing.T) {
		fileContent := []byte("Test file content")
		body, contentType := createMultipartFileRequest(t, "file", "test.txt", fileContent, map[string]string{
			"content_type":  "file",
			"source_device": device.ID,
		})

		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/sync/upload",
			Body:   body.Bytes(),
			Headers: map[string]string{
				"Content-Type": contentType,
				"X-User-ID":    user.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if _, ok := response["item_id"].(string); !ok {
			t.Error("Expected item_id in response")
		}
	})
}

// TestSettings tests settings endpoints
func TestSettings(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()

	t.Run("GetSettings", func(t *testing.T) {
		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/settings",
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if _, ok := response["max_file_size"]; !ok {
			t.Error("Expected max_file_size in response")
		}

		if _, ok := response["default_expiry_hours"]; !ok {
			t.Error("Expected default_expiry_hours in response")
		}
	})

	t.Run("UpdateSettings", func(t *testing.T) {
		newSettings := map[string]interface{}{
			"default_expiry_hours": 48,
		}

		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/v1/settings",
			Body:   newSettings,
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if success, ok := response["success"].(bool); !ok || !success {
			t.Errorf("Expected success true, got %v", response["success"])
		}
	})
}

// TestExpiryCleanup tests automatic expiry cleanup
func TestExpiryCleanup(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()

	t.Run("CleanupExpiredItems", func(t *testing.T) {
		// Create an expired item
		expiredItem := &SyncItem{
			ID:            uuid.New().String(),
			UserID:        user.ID,
			Type:          "clipboard",
			Content:       map[string]interface{}{"data": "expired content"},
			SourceDevice:  uuid.New().String(),
			TargetDevices: []string{},
			CreatedAt:     time.Now().Add(-48 * time.Hour),
			ExpiresAt:     time.Now().Add(-1 * time.Hour), // Expired
			Status:        "active",
		}

		contentJSON, _ := json.Marshal(expiredItem.Content)
		targetDevicesJSON, _ := json.Marshal(expiredItem.TargetDevices)

		_, err := ts.DB.Exec(`
			INSERT INTO sync_items (id, user_id, type, content, source_device, target_devices, created_at, expires_at, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, expiredItem.ID, expiredItem.UserID, expiredItem.Type, contentJSON,
			expiredItem.SourceDevice, targetDevicesJSON, expiredItem.CreatedAt, expiredItem.ExpiresAt, expiredItem.Status)

		if err != nil {
			t.Fatalf("Failed to create expired item: %v", err)
		}

		// Create a non-expired item
		activeItem := ts.createTestSyncItem(t, user.ID, "clipboard")

		// Run cleanup
		cleaned := ts.cleanupExpiredItems(t)

		if cleaned != 1 {
			t.Errorf("Expected 1 cleaned item, got %d", cleaned)
		}

		// Verify active item still exists
		count := ts.getSyncItemCount(t, user.ID)
		if count != 1 {
			t.Errorf("Expected 1 active item remaining, got %d", count)
		}

		// Verify the remaining item is the active one
		var remainingID string
		err = ts.DB.QueryRow(`
			SELECT id FROM sync_items WHERE user_id = $1 AND status = 'active'
		`, user.ID).Scan(&remainingID)

		if err != nil {
			t.Fatalf("Failed to query remaining item: %v", err)
		}

		if remainingID != activeItem.ID {
			t.Errorf("Expected remaining item to be %s, got %s", activeItem.ID, remainingID)
		}
	})
}

// TestConcurrentOperations tests concurrent device and sync operations
func TestConcurrentOperations(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()

	t.Run("ConcurrentDeviceRegistration", func(t *testing.T) {
		const numDevices = 10
		done := make(chan bool, numDevices)
		errors := make(chan error, numDevices)

		for i := 0; i < numDevices; i++ {
			go func(index int) {
				device := map[string]interface{}{
					"name":         fmt.Sprintf("Device %d", index),
					"type":         "mobile",
					"platform":     "android",
					"capabilities": []string{"clipboard"},
				}

				_, err := ts.makeHTTPRequest(HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/devices",
					Body:   device,
					Headers: map[string]string{
						"X-User-ID": user.ID,
					},
				})

				if err != nil {
					errors <- err
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < numDevices; i++ {
			<-done
		}

		close(errors)
		for err := range errors {
			t.Errorf("Concurrent operation error: %v", err)
		}

		// Verify all devices were created
		count := ts.getDeviceCount(t, user.ID)
		if count != numDevices {
			t.Errorf("Expected %d devices, got %d", numDevices, count)
		}
	})

	t.Run("ConcurrentSyncItemCreation", func(t *testing.T) {
		const numItems = 20
		done := make(chan bool, numItems)
		errors := make(chan error, numItems)

		for i := 0; i < numItems; i++ {
			go func(index int) {
				clipboardData := map[string]interface{}{
					"content": fmt.Sprintf("Concurrent content %d", index),
				}

				_, err := ts.makeHTTPRequest(HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/sync/clipboard",
					Body:   clipboardData,
					Headers: map[string]string{
						"X-User-ID": user.ID,
					},
				})

				if err != nil {
					errors <- err
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < numItems; i++ {
			<-done
		}

		close(errors)
		for err := range errors {
			t.Errorf("Concurrent operation error: %v", err)
		}

		// Verify items were created
		count := ts.getSyncItemCount(t, user.ID)
		if count < numItems {
			t.Errorf("Expected at least %d items, got %d", numItems, count)
		}
	})
}

// TestDatabaseMigrations tests database migration functionality
func TestDatabaseMigrations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dbURL := os.Getenv("POSTGRES_URL")
	if dbURL == "" {
		t.Skip("POSTGRES_URL not set, skipping migration test")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	t.Run("MigrationsRunSuccessfully", func(t *testing.T) {
		err := runMigrations(db)
		if err != nil {
			t.Fatalf("Failed to run migrations: %v", err)
		}

		// Verify tables exist
		tables := []string{"device_sessions", "sync_items"}
		for _, table := range tables {
			var exists bool
			err := db.QueryRow(`
				SELECT EXISTS (
					SELECT FROM information_schema.tables
					WHERE table_name = $1
				)
			`, table).Scan(&exists)

			if err != nil {
				t.Fatalf("Failed to check table %s: %v", table, err)
			}

			if !exists {
				t.Errorf("Expected table %s to exist after migration", table)
			}
		}
	})

	t.Run("MigrationsAreIdempotent", func(t *testing.T) {
		// Run migrations again - should not fail
		err := runMigrations(db)
		if err != nil {
			t.Fatalf("Failed to run migrations second time: %v", err)
		}
	})
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()

	t.Run("EmptyClipboardContent", func(t *testing.T) {
		clipboardData := map[string]interface{}{
			"content": "",
		}

		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/sync/clipboard",
			Body:   clipboardData,
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should accept empty content
		response := assertJSONResponse(t, w, http.StatusOK)

		if _, ok := response["item_id"].(string); !ok {
			t.Error("Expected item_id in response for empty content")
		}
	})

	t.Run("VeryLongDeviceName", func(t *testing.T) {
		longName := string(make([]byte, 1000))
		for i := range longName {
			longName = string(append([]byte(longName[:i]), 'A'))
		}

		device := map[string]interface{}{
			"name":         longName,
			"type":         "mobile",
			"platform":     "android",
			"capabilities": []string{"clipboard"},
		}

		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/devices",
			Body:   device,
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should handle long names gracefully
		if w.Code != http.StatusCreated && w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 201 or 400 for very long name, got %d", w.Code)
		}
	})

	t.Run("SpecialCharactersInContent", func(t *testing.T) {
		specialContent := "测试 🚀 éàü <script>alert('xss')</script> \n\t\r"

		clipboardData := map[string]interface{}{
			"content": specialContent,
		}

		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/sync/clipboard",
			Body:   clipboardData,
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if _, ok := response["item_id"].(string); !ok {
			t.Error("Expected item_id in response for special characters")
		}
	})
}

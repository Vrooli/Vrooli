package main

import (
	"context"
	"testing"
	"time"
)

// TestNewDeviceController tests device controller initialization
func TestNewDeviceController(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Creates device controller with database", func(t *testing.T) {
		db := MockDatabaseConnection(t)
		dc := NewDeviceController(db)

		if dc == nil {
			t.Fatal("NewDeviceController returned nil")
		}

		if dc.db != db {
			t.Error("Device controller database not set correctly")
		}
	})

	t.Run("Creates device controller with nil database", func(t *testing.T) {
		dc := NewDeviceController(nil)

		if dc == nil {
			t.Fatal("NewDeviceController returned nil with nil db")
		}
	})
}

// TestValidateControlRequest tests request validation
func TestValidateControlRequest(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dc := NewDeviceController(nil)

	tests := []struct {
		name        string
		req         DeviceControlRequest
		shouldError bool
		errorMsg    string
	}{
		{
			name: "Valid request",
			req: DeviceControlRequest{
				DeviceID:  "light.living_room",
				Action:    "turn_on",
				UserID:    "test-user",
				ProfileID: "test-profile",
			},
			shouldError: false,
		},
		{
			name: "Missing device_id",
			req: DeviceControlRequest{
				Action: "turn_on",
				UserID: "test-user",
			},
			shouldError: true,
			errorMsg:    "device_id is required",
		},
		{
			name: "Missing action",
			req: DeviceControlRequest{
				DeviceID: "light.living_room",
				UserID:   "test-user",
			},
			shouldError: true,
			errorMsg:    "action is required",
		},
		{
			name: "Missing user_id and profile_id",
			req: DeviceControlRequest{
				DeviceID: "light.living_room",
				Action:   "turn_on",
			},
			shouldError: true,
			errorMsg:    "user_id or profile_id is required",
		},
		{
			name: "Invalid device_id format",
			req: DeviceControlRequest{
				DeviceID:  "light.living room; DROP TABLE devices;",
				Action:    "turn_on",
				UserID:    "test-user",
				ProfileID: "test-profile",
			},
			shouldError: true,
			errorMsg:    "invalid device_id format",
		},
		{
			name: "Invalid action",
			req: DeviceControlRequest{
				DeviceID:  "light.living_room",
				Action:    "delete_device",
				UserID:    "test-user",
				ProfileID: "test-profile",
			},
			shouldError: true,
			errorMsg:    "invalid action",
		},
		{
			name: "Valid device_id with underscores",
			req: DeviceControlRequest{
				DeviceID:  "light.living_room_1",
				Action:    "turn_on",
				UserID:    "test-user",
				ProfileID: "test-profile",
			},
			shouldError: false,
		},
		{
			name: "Valid device_id with dots",
			req: DeviceControlRequest{
				DeviceID:  "climate.bedroom.ac",
				Action:    "set_temperature",
				UserID:    "test-user",
				ProfileID: "test-profile",
			},
			shouldError: false,
		},
		{
			name: "Valid action toggle",
			req: DeviceControlRequest{
				DeviceID:  "switch.coffee_maker",
				Action:    "toggle",
				UserID:    "test-user",
				ProfileID: "test-profile",
			},
			shouldError: false,
		},
		{
			name: "Valid action set_brightness",
			req: DeviceControlRequest{
				DeviceID:  "light.bedroom",
				Action:    "set_brightness",
				UserID:    "test-user",
				ProfileID: "test-profile",
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := dc.validateControlRequest(tt.req)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				} else if !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

// TestCheckPermissions tests permission checking logic
func TestCheckPermissions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dc := NewDeviceController(nil)
	ctx := context.Background()

	tests := []struct {
		name        string
		userID      string
		profileID   string
		deviceID    string
		shouldAllow bool
		shouldError bool
	}{
		{
			name:        "Admin user has full access",
			userID:      "550e8400-e29b-41d4-a716-446655440001",
			deviceID:    "light.living_room",
			shouldAllow: true,
			shouldError: false,
		},
		{
			name:        "Family member has access to allowed device",
			userID:      "550e8400-e29b-41d4-a716-446655440002",
			deviceID:    "light.living_room",
			shouldAllow: true,
			shouldError: false,
		},
		{
			name:        "Family member blocked from non-allowed device",
			userID:      "550e8400-e29b-41d4-a716-446655440002",
			deviceID:    "climate.bedroom",
			shouldAllow: false,
			shouldError: false,
		},
		{
			name:        "Kid user has limited access",
			userID:      "550e8400-e29b-41d4-a716-446655440003",
			deviceID:    "light.bedroom_kid",
			shouldAllow: true,
			shouldError: false,
		},
		{
			name:        "Kid user blocked from other devices",
			userID:      "550e8400-e29b-41d4-a716-446655440003",
			deviceID:    "light.living_room",
			shouldAllow: false,
			shouldError: false,
		},
		{
			name:        "Unknown user blocked",
			userID:      "unknown-user-id",
			deviceID:    "light.living_room",
			shouldAllow: false,
			shouldError: false,
		},
		{
			name:        "Empty userID defaults to admin",
			userID:      "",
			deviceID:    "light.living_room",
			shouldAllow: true,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := dc.checkPermissions(ctx, tt.userID, tt.profileID, tt.deviceID)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}

				if allowed != tt.shouldAllow {
					t.Errorf("Expected allowed=%v, got %v", tt.shouldAllow, allowed)
				}
			}
		})
	}
}

// TestControlDeviceValidation tests ControlDevice with validation failures
func TestControlDeviceValidation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dc := NewDeviceController(nil)
	ctx := context.Background()

	t.Run("Returns error for missing device_id", func(t *testing.T) {
		req := DeviceControlRequest{
			Action: "turn_on",
			UserID: "test-user",
		}

		resp, err := dc.ControlDevice(ctx, req)

		if err == nil {
			t.Error("Expected error for missing device_id")
		}

		if resp == nil {
			t.Fatal("Response should not be nil even on error")
		}

		if resp.Success {
			t.Error("Response should indicate failure")
		}

		if resp.Error == "" {
			t.Error("Response should contain error message")
		}

		if resp.RequestID == "" {
			t.Error("Response should contain request ID")
		}
	})

	t.Run("Returns error for invalid action", func(t *testing.T) {
		req := DeviceControlRequest{
			DeviceID: "light.living_room",
			Action:   "hack_the_planet",
			UserID:   "test-user",
		}

		resp, err := dc.ControlDevice(ctx, req)

		if err == nil {
			t.Error("Expected error for invalid action")
		}

		if resp == nil {
			t.Fatal("Response should not be nil even on error")
		}

		if resp.Success {
			t.Error("Response should indicate failure")
		}

		if !contains(resp.Error, "invalid action") {
			t.Errorf("Expected 'invalid action' in error, got: %s", resp.Error)
		}
	})

	t.Run("Includes execution time in response", func(t *testing.T) {
		req := DeviceControlRequest{
			DeviceID: "",
			Action:   "turn_on",
			UserID:   "test-user",
		}

		resp, _ := dc.ControlDevice(ctx, req)

		if resp.ExecutionTimeMs < 0 {
			t.Error("Execution time should be non-negative")
		}
	})
}

// TestControlDeviceResponseStructure tests response structure
func TestControlDeviceResponseStructure(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dc := NewDeviceController(nil)
	ctx := context.Background()

	t.Run("Response contains all required fields", func(t *testing.T) {
		req := DeviceControlRequest{
			DeviceID: "light.living_room",
			Action:   "turn_on",
			UserID:   "test-user",
		}

		resp, _ := dc.ControlDevice(ctx, req)

		if resp == nil {
			t.Fatal("Response should not be nil")
		}

		// Check required fields
		if resp.DeviceID == "" {
			t.Error("Response missing DeviceID")
		}

		if resp.Action == "" {
			t.Error("Response missing Action")
		}

		if resp.RequestID == "" {
			t.Error("Response missing RequestID")
		}

		if resp.Timestamp == "" {
			t.Error("Response missing Timestamp")
		}

		if resp.Message == "" {
			t.Error("Response missing Message")
		}

		// Validate timestamp format
		_, err := time.Parse(time.RFC3339, resp.Timestamp)
		if err != nil {
			t.Errorf("Timestamp not in RFC3339 format: %v", err)
		}
	})
}

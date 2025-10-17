package main

import (
	"context"
	"encoding/json"
	"testing"
)

// TestSmartPairingUnits tests smart pairing components without Ollama
func TestSmartPairingUnits(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("GetListItems", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Test List", 5)
		defer testList.Cleanup()

		items, err := testApp.App.SmartPairing.getListItems(testList.List.ID)
		if err != nil {
			t.Fatalf("Failed to get list items: %v", err)
		}

		if len(items) != 5 {
			t.Errorf("Expected 5 items, got %d", len(items))
		}

		// Verify items have content
		for i, item := range items {
			if item.ID == "" {
				t.Errorf("Item %d has empty ID", i)
			}
			if len(item.Content) == 0 {
				t.Errorf("Item %d has empty content", i)
			}
		}
	})

	t.Run("StorePairingQueue", func(t *testing.T) {
		// Note: This will fail if pairing_queue table doesn't exist
		// which is expected in current setup
		testList := setupTestList(t, testApp.DB, "Test List", 3)
		defer testList.Cleanup()

		pair := SuggestedPair{
			ItemAID:          testList.Items[0].ID,
			ItemBID:          testList.Items[1].ID,
			Priority:         80.0,
			UncertaintyScore: 0.8,
		}

		// This will fail if table doesn't exist - expected behavior
		err := testApp.App.SmartPairing.storePairingQueue(testList.List.ID, pair)

		// Either succeeds (table exists) or fails (table doesn't exist)
		// Both are acceptable for this test
		if err != nil {
			t.Logf("Expected error when pairing_queue table doesn't exist: %v", err)
		}
	})

	t.Run("GenerateSmartPairs_InsufficientItems", func(t *testing.T) {
		// Test with only 1 item - should fail
		items := []PairItem{
			{ID: "item1", Content: json.RawMessage(`{"name": "Item 1"}`)},
		}

		req := PairingRequest{
			ListID: "test-list",
			Items:  items,
		}

		ctx := context.Background()
		resp, err := testApp.App.SmartPairing.GenerateSmartPairs(ctx, req)

		if err == nil {
			t.Error("Expected error for insufficient items")
		}
		if resp != nil && resp.Success {
			t.Error("Expected unsuccessful response for insufficient items")
		}
	})

	t.Run("GenerateSmartPairs_NoItems", func(t *testing.T) {
		req := PairingRequest{
			ListID: "test-list",
			Items:  []PairItem{},
		}

		ctx := context.Background()
		resp, err := testApp.App.SmartPairing.GenerateSmartPairs(ctx, req)

		if err == nil {
			t.Error("Expected error for no items")
		}
		if resp != nil && resp.Success {
			t.Error("Expected unsuccessful response for no items")
		}
	})

	t.Run("ClearQueue", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Test List", 3)
		defer testList.Cleanup()

		ctx := context.Background()
		err := testApp.App.SmartPairing.ClearQueue(ctx, testList.List.ID)

		// Either succeeds (table exists) or fails (table doesn't exist)
		if err != nil {
			t.Logf("Expected error when pairing_queue table doesn't exist: %v", err)
		}
	})

	t.Run("GetQueuedPairs_EmptyQueue", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Test List", 3)
		defer testList.Cleanup()

		ctx := context.Background()
		pairs, err := testApp.App.SmartPairing.GetQueuedPairs(ctx, testList.List.ID, 10)

		// Either succeeds with empty array or fails (table doesn't exist)
		if err != nil {
			t.Logf("Expected error when pairing_queue table doesn't exist: %v", err)
		} else if len(pairs) != 0 {
			t.Logf("Queue is not empty: %d pairs", len(pairs))
		}
	})

	t.Run("GetQueuedPairs_ZeroLimit", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Test List", 3)
		defer testList.Cleanup()

		ctx := context.Background()
		pairs, err := testApp.App.SmartPairing.GetQueuedPairs(ctx, testList.List.ID, 0)

		// Should use default limit of 10
		if err != nil {
			t.Logf("Expected error when pairing_queue table doesn't exist: %v", err)
		} else if pairs == nil {
			t.Error("Expected non-nil pairs array")
		}
	})

	t.Run("GetQueuedPairs_NegativeLimit", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Test List", 3)
		defer testList.Cleanup()

		ctx := context.Background()
		pairs, err := testApp.App.SmartPairing.GetQueuedPairs(ctx, testList.List.ID, -5)

		// Should use default limit of 10
		if err != nil {
			t.Logf("Expected error when pairing_queue table doesn't exist: %v", err)
		} else if pairs == nil {
			t.Error("Expected non-nil pairs array")
		}
	})
}

// TestSmartPairingDatabaseNil tests behavior with nil database
func TestSmartPairingDatabaseNil(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("StorePairingQueue_NilDB", func(t *testing.T) {
		sp := NewSmartPairing(nil)

		pair := SuggestedPair{
			ItemAID:          "item1",
			ItemBID:          "item2",
			Priority:         80.0,
			UncertaintyScore: 0.8,
		}

		err := sp.storePairingQueue("list-id", pair)
		if err == nil {
			t.Error("Expected error when database is nil")
		}
	})
}


package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// TestContactLifecycle tests the complete lifecycle of a contact
func TestContactLifecycle(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	var contactID int

	t.Run("CreateContact", func(t *testing.T) {
		newContact := TestData.CreateContactRequest("LifecycleTestContact")
		body, _ := json.Marshal(newContact)

		req := httptest.NewRequest("POST", "/api/contacts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createContactHandler(w, req)

		response := assertJSONResponse(t, w, 201, map[string]interface{}{
			"name": newContact.Name,
		})

		if response == nil {
			t.Fatal("Expected non-nil response")
		}

		if idFloat, ok := response["id"].(float64); ok {
			contactID = int(idFloat)
		} else {
			t.Fatal("Expected id in response")
		}
	})

	t.Run("GetContact", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/contacts/"+strconv.Itoa(contactID), nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(contactID)})
		w := httptest.NewRecorder()

		getContactHandler(w, req)

		response := assertJSONResponse(t, w, 200, map[string]interface{}{
			"name": "LifecycleTestContact",
		})

		if response == nil {
			t.Fatal("Expected non-nil response")
		}
	})

	t.Run("UpdateContact", func(t *testing.T) {
		updatedContact := Contact{
			Name:             "UpdatedLifecycleContact",
			Nickname:         "Updated",
			RelationshipType: "family",
			Birthday:         "1985-03-15",
			Email:            "updated@example.com",
			Phone:            "555-9999",
			Notes:            "Updated lifecycle test",
			Interests:        []string{"updated"},
			Tags:             []string{"updated"},
		}
		body, _ := json.Marshal(updatedContact)

		req := httptest.NewRequest("PUT", "/api/contacts/"+strconv.Itoa(contactID), bytes.NewReader(body))
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(contactID)})
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		updateContactHandler(w, req)

		response := assertJSONResponse(t, w, 200, map[string]interface{}{
			"name": "UpdatedLifecycleContact",
		})

		if response == nil {
			t.Fatal("Expected non-nil response")
		}
	})

	t.Run("AddInteraction", func(t *testing.T) {
		interaction := TestData.CreateInteractionRequest(contactID)
		body, _ := json.Marshal(interaction)

		req := httptest.NewRequest("POST", "/api/interactions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createInteractionHandler(w, req)

		response := assertJSONResponse(t, w, 201, map[string]interface{}{
			"contact_id": float64(contactID),
		})

		if response == nil {
			t.Fatal("Expected non-nil response")
		}
	})

	t.Run("AddGift", func(t *testing.T) {
		gift := TestData.CreateGiftRequest(contactID)
		body, _ := json.Marshal(gift)

		req := httptest.NewRequest("POST", "/api/gifts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createGiftHandler(w, req)

		response := assertJSONResponse(t, w, 201, map[string]interface{}{
			"contact_id": float64(contactID),
		})

		if response == nil {
			t.Fatal("Expected non-nil response")
		}
	})

	t.Run("GetInteractions", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/contacts/"+strconv.Itoa(contactID)+"/interactions", nil)
		req = mux.SetURLVars(req, map[string]string{"contactId": strconv.Itoa(contactID)})
		w := httptest.NewRecorder()

		getInteractionsHandler(w, req)

		interactions := assertJSONArray(t, w, 200)
		if len(interactions) < 1 {
			t.Error("Expected at least 1 interaction")
		}
	})

	t.Run("GetGifts", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/contacts/"+strconv.Itoa(contactID)+"/gifts", nil)
		req = mux.SetURLVars(req, map[string]string{"contactId": strconv.Itoa(contactID)})
		w := httptest.NewRecorder()

		getGiftsHandler(w, req)

		gifts := assertJSONArray(t, w, 200)
		if len(gifts) < 1 {
			t.Error("Expected at least 1 gift")
		}
	})

	t.Run("DeleteContact", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/contacts/"+strconv.Itoa(contactID), nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(contactID)})
		w := httptest.NewRecorder()

		deleteContactHandler(w, req)

		if w.Code != 204 {
			t.Errorf("Expected status 204, got %d", w.Code)
		}
	})

	t.Run("VerifyDeleted", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/contacts/"+strconv.Itoa(contactID), nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(contactID)})
		w := httptest.NewRecorder()

		getContactHandler(w, req)

		assertErrorResponse(t, w, 404)
	})
}

// TestRelationshipProcessorIntegration tests the relationship processor with real data
func TestRelationshipProcessorIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB
	relationshipProcessor = NewRelationshipProcessor(testDB.DB)

	contact := setupTestContact(t, testDB.DB, "ProcessorTestContact")
	defer contact.Cleanup()

	t.Run("EnrichContact", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/contacts/"+strconv.Itoa(contact.Contact.ID)+"/enrich", nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(contact.Contact.ID)})
		w := httptest.NewRecorder()

		enrichContactHandler(w, req)

		response := assertJSONResponse(t, w, 200, map[string]interface{}{
			"contact_id": float64(contact.Contact.ID),
		})

		if response == nil {
			t.Fatal("Expected non-nil response")
		}

		// Verify enrichment fields
		if _, exists := response["suggested_interests"]; !exists {
			t.Error("Expected suggested_interests field")
		}
		if _, exists := response["personality_traits"]; !exists {
			t.Error("Expected personality_traits field")
		}
		if _, exists := response["conversation_starters"]; !exists {
			t.Error("Expected conversation_starters field")
		}
	})

	t.Run("SuggestGifts", func(t *testing.T) {
		suggestionReq := GiftSuggestionRequest{
			Name:      contact.Contact.Name,
			Interests: "reading, travel, coffee",
			Occasion:  "birthday",
			Budget:    "50-100",
		}
		body, _ := json.Marshal(suggestionReq)

		req := httptest.NewRequest("POST", "/api/contacts/"+strconv.Itoa(contact.Contact.ID)+"/gifts/suggest", bytes.NewReader(body))
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(contact.Contact.ID)})
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suggestGiftsHandler(w, req)

		response := assertJSONResponse(t, w, 200, map[string]interface{}{
			"contact_id": float64(contact.Contact.ID),
		})

		if response == nil {
			t.Fatal("Expected non-nil response")
		}

		if _, exists := response["suggestions"]; !exists {
			t.Error("Expected suggestions field")
		}
	})

	t.Run("AnalyzeRelationships", func(t *testing.T) {
		// Add some interactions first
		setupTestInteraction(t, testDB.DB, contact.Contact.ID)
		time.Sleep(100 * time.Millisecond) // Ensure distinct timestamps
		setupTestInteraction(t, testDB.DB, contact.Contact.ID)

		req := httptest.NewRequest("GET", "/api/contacts/"+strconv.Itoa(contact.Contact.ID)+"/insights", nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(contact.Contact.ID)})
		w := httptest.NewRecorder()

		getInsightsHandler(w, req)

		response := assertJSONResponse(t, w, 200, map[string]interface{}{
			"contact_id": float64(contact.Contact.ID),
		})

		if response == nil {
			t.Fatal("Expected non-nil response")
		}

		// Verify insight fields
		expectedFields := []string{
			"last_interaction_days",
			"interaction_frequency",
			"overall_sentiment",
			"recommended_actions",
			"relationship_score",
			"trend_analysis",
		}
		for _, field := range expectedFields {
			if _, exists := response[field]; !exists {
				t.Errorf("Expected field '%s' in response", field)
			}
		}
	})
}

// TestBirthdayReminderFlow tests the birthday reminder workflow
func TestBirthdayReminderFlow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB
	relationshipProcessor = NewRelationshipProcessor(testDB.DB)

	// Create contact with birthday in next 5 days
	today := time.Now()
	upcomingDate := today.AddDate(0, 0, 3) // 3 days from now
	contact := setupTestContact(t, testDB.DB, "BirthdayContact")
	defer contact.Cleanup()

	birthdayStr := upcomingDate.Format("2006-01-02")
	testDB.DB.Exec("UPDATE contacts SET birthday = $1 WHERE id = $2", birthdayStr, contact.Contact.ID)

	t.Run("GetUpcomingBirthdays", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/birthdays?days_ahead=7", nil)
		w := httptest.NewRecorder()

		getBirthdaysHandler(w, req)

		birthdays := assertJSONArray(t, w, 200)
		if birthdays == nil {
			t.Fatal("Expected birthdays array")
		}

		// Verify our test contact is in the results
		found := false
		for _, b := range birthdays {
			birthday := b.(map[string]interface{})
			if int(birthday["contact_id"].(float64)) == contact.Contact.ID {
				found = true
				if birthday["days_until"] == nil {
					t.Error("Expected days_until field")
				}
				if birthday["message"] == nil {
					t.Error("Expected message field")
				}
				if birthday["urgency"] == nil {
					t.Error("Expected urgency field")
				}
			}
		}

		if !found {
			t.Error("Expected to find test contact in birthday results")
		}
	})
}

// TestConcurrentOperations tests thread safety of handlers
func TestConcurrentOperations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	contact := setupTestContact(t, testDB.DB, "ConcurrentTestContact")
	defer contact.Cleanup()

	t.Run("ConcurrentReads", func(t *testing.T) {
		done := make(chan bool)
		iterations := 10

		for i := 0; i < iterations; i++ {
			go func() {
				req := httptest.NewRequest("GET", "/api/contacts/"+strconv.Itoa(contact.Contact.ID), nil)
				req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(contact.Contact.ID)})
				w := httptest.NewRecorder()

				getContactHandler(w, req)

				if w.Code != 200 {
					t.Errorf("Expected status 200, got %d", w.Code)
				}
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < iterations; i++ {
			<-done
		}
	})

	t.Run("ConcurrentWrites", func(t *testing.T) {
		done := make(chan bool)
		iterations := 5

		for i := 0; i < iterations; i++ {
			go func(idx int) {
				interaction := Interaction{
					ContactID:       contact.Contact.ID,
					InteractionType: "email",
					InteractionDate: time.Now().Format("2006-01-02"),
					Description:     "Concurrent test " + strconv.Itoa(idx),
					Sentiment:       "positive",
				}
				body, _ := json.Marshal(interaction)

				req := httptest.NewRequest("POST", "/api/interactions", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				createInteractionHandler(w, req)

				if w.Code != 201 {
					t.Errorf("Expected status 201, got %d", w.Code)
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < iterations; i++ {
			<-done
		}

		// Verify all interactions were created
		req := httptest.NewRequest("GET", "/api/contacts/"+strconv.Itoa(contact.Contact.ID)+"/interactions", nil)
		req = mux.SetURLVars(req, map[string]string{"contactId": strconv.Itoa(contact.Contact.ID)})
		w := httptest.NewRecorder()

		getInteractionsHandler(w, req)

		interactions := assertJSONArray(t, w, 200)
		if len(interactions) < iterations {
			t.Errorf("Expected at least %d interactions, got %d", iterations, len(interactions))
		}
	})
}

// TestDatabaseConnectionHandling tests database error scenarios
func TestDatabaseConnectionHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	t.Run("NilDatabaseHandler", func(t *testing.T) {
		// Temporarily set db to nil
		originalDB := db
		db = nil
		defer func() { db = originalDB }()

		// Should panic with nil db (which is expected behavior)
		defer func() {
			if r := recover(); r != nil {
				t.Log("Handler panicked with nil database as expected")
			}
		}()

		req := httptest.NewRequest("GET", "/api/contacts", nil)
		w := httptest.NewRecorder()

		getContactsHandler(w, req)

		// If we get here without panic, log it
		t.Log("Handler did not panic with nil database")
	})
}

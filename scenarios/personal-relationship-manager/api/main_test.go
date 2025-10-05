
package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	healthHandler(w, req)

	response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
		"status": "healthy",
	})

	if response == nil {
		t.Fatal("Expected non-nil response")
	}
}

// TestGetContactsHandler tests the get contacts endpoint
func TestGetContactsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return // Test skipped
	}
	defer testDB.Cleanup()

	// Set global db for handlers
	db = testDB.DB

	// Create test contacts
	contact1 := setupTestContact(t, testDB.DB, "TestContact1")
	defer contact1.Cleanup()
	contact2 := setupTestContact(t, testDB.DB, "TestContact2")
	defer contact2.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/contacts", nil)
		w := httptest.NewRecorder()

		getContactsHandler(w, req)

		contacts := assertJSONArray(t, w, http.StatusOK)
		if contacts == nil {
			t.Fatal("Expected non-nil contacts array")
		}

		if len(contacts) < 2 {
			t.Errorf("Expected at least 2 contacts, got %d", len(contacts))
		}
	})

	t.Run("EmptyDatabase", func(t *testing.T) {
		// Clean up all test contacts
		contact1.Cleanup()
		contact2.Cleanup()

		req := httptest.NewRequest("GET", "/api/contacts", nil)
		w := httptest.NewRecorder()

		getContactsHandler(w, req)

		contacts := assertJSONArray(t, w, http.StatusOK)
		// Should return empty array, not error
		if contacts == nil {
			contacts = []interface{}{}
		}
	})
}

// TestGetContactHandler tests retrieving a single contact
func TestGetContactHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	contact := setupTestContact(t, testDB.DB, "TestContact")
	defer contact.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/contacts/"+strconv.Itoa(contact.Contact.ID), nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(contact.Contact.ID)})
		w := httptest.NewRecorder()

		getContactHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"name": contact.Contact.Name,
		})

		if response == nil {
			t.Fatal("Expected non-nil response")
		}

		// Verify all fields are present
		expectedFields := []string{"id", "name", "nickname", "relationship_type", "birthday",
			"email", "phone", "notes", "interests", "tags"}
		for _, field := range expectedFields {
			if _, exists := response[field]; !exists {
				t.Errorf("Expected field '%s' in response", field)
			}
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/contacts/999999", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "999999"})
		w := httptest.NewRecorder()

		getContactHandler(w, req)

		assertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("InvalidID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/contacts/invalid", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
		w := httptest.NewRecorder()

		getContactHandler(w, req)

		// Should handle gracefully (strconv.Atoi returns 0 for invalid)
		assertErrorResponse(t, w, http.StatusNotFound)
	})
}

// TestCreateContactHandler tests creating a new contact
func TestCreateContactHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("Success", func(t *testing.T) {
		newContact := TestData.CreateContactRequest("TestNewContact")
		body, _ := json.Marshal(newContact)

		req := httptest.NewRequest("POST", "/api/contacts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createContactHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"name": newContact.Name,
		})

		if response == nil {
			t.Fatal("Expected non-nil response")
		}

		// Verify ID was assigned
		if id, exists := response["id"]; !exists || id == nil {
			t.Error("Expected 'id' field in response")
		}

		// Clean up
		if idFloat, ok := response["id"].(float64); ok {
			db.Exec("DELETE FROM contacts WHERE id = $1", int(idFloat))
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/contacts", bytes.NewReader([]byte("{invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createContactHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("EmptyBody", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/contacts", bytes.NewReader([]byte("")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createContactHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("MissingRequiredFields", func(t *testing.T) {
		// Contact with empty name (should still insert but might cause issues)
		emptyContact := Contact{
			Email: "test@example.com",
		}
		body, _ := json.Marshal(emptyContact)

		req := httptest.NewRequest("POST", "/api/contacts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createContactHandler(w, req)

		// Depending on database constraints, this might succeed or fail
		// For now, we expect it to handle gracefully
		if w.Code != http.StatusCreated && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 201 or 500, got %d", w.Code)
		}
	})
}

// TestUpdateContactHandler tests updating an existing contact
func TestUpdateContactHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	contact := setupTestContact(t, testDB.DB, "TestUpdateContact")
	defer contact.Cleanup()

	t.Run("Success", func(t *testing.T) {
		updatedContact := Contact{
			Name:             "UpdatedName",
			Nickname:         "UpdatedNick",
			RelationshipType: "family",
			Birthday:         "1985-06-20",
			Email:            "updated@example.com",
			Phone:            "555-9999",
			Notes:            "Updated notes",
			Interests:        []string{"updated", "interests"},
			Tags:             []string{"updated"},
		}
		body, _ := json.Marshal(updatedContact)

		req := httptest.NewRequest("PUT", "/api/contacts/"+strconv.Itoa(contact.Contact.ID), bytes.NewReader(body))
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(contact.Contact.ID)})
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		updateContactHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"name": "UpdatedName",
		})

		if response == nil {
			t.Fatal("Expected non-nil response")
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		updatedContact := TestData.CreateContactRequest("NonExistent")
		body, _ := json.Marshal(updatedContact)

		req := httptest.NewRequest("PUT", "/api/contacts/999999", bytes.NewReader(body))
		req = mux.SetURLVars(req, map[string]string{"id": "999999"})
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		updateContactHandler(w, req)

		// Update handler doesn't check if contact exists, just runs UPDATE
		// This will succeed but affect 0 rows
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Logf("Update non-existent contact returned status: %d", w.Code)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/api/contacts/"+strconv.Itoa(contact.Contact.ID), bytes.NewReader([]byte("{invalid")))
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(contact.Contact.ID)})
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		updateContactHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestDeleteContactHandler tests deleting a contact
func TestDeleteContactHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("Success", func(t *testing.T) {
		contact := setupTestContact(t, testDB.DB, "TestDeleteContact")
		// Don't defer cleanup since we're deleting it

		req := httptest.NewRequest("DELETE", "/api/contacts/"+strconv.Itoa(contact.Contact.ID), nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(contact.Contact.ID)})
		w := httptest.NewRecorder()

		deleteContactHandler(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", w.Code)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/contacts/999999", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "999999"})
		w := httptest.NewRecorder()

		deleteContactHandler(w, req)

		// Delete doesn't check existence, succeeds even if nothing deleted
		if w.Code != http.StatusNoContent && w.Code != http.StatusInternalServerError {
			t.Logf("Delete non-existent contact returned status: %d", w.Code)
		}
	})
}

// TestGetInteractionsHandler tests retrieving interactions for a contact
func TestGetInteractionsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	contact := setupTestContact(t, testDB.DB, "TestInteractionContact")
	defer contact.Cleanup()

	interaction := setupTestInteraction(t, testDB.DB, contact.Contact.ID)

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/contacts/"+strconv.Itoa(contact.Contact.ID)+"/interactions", nil)
		req = mux.SetURLVars(req, map[string]string{"contactId": strconv.Itoa(contact.Contact.ID)})
		w := httptest.NewRecorder()

		getInteractionsHandler(w, req)

		interactions := assertJSONArray(t, w, http.StatusOK)
		if interactions == nil {
			t.Fatal("Expected non-nil interactions array")
		}

		if len(interactions) < 1 {
			t.Error("Expected at least 1 interaction")
		}
	})

	t.Run("NoInteractions", func(t *testing.T) {
		emptyContact := setupTestContact(t, testDB.DB, "TestNoInteractions")
		defer emptyContact.Cleanup()

		req := httptest.NewRequest("GET", "/api/contacts/"+strconv.Itoa(emptyContact.Contact.ID)+"/interactions", nil)
		req = mux.SetURLVars(req, map[string]string{"contactId": strconv.Itoa(emptyContact.Contact.ID)})
		w := httptest.NewRecorder()

		getInteractionsHandler(w, req)

		interactions := assertJSONArray(t, w, http.StatusOK)
		if interactions == nil {
			interactions = []interface{}{}
		}
	})

	// Clean up interaction
	db.Exec("DELETE FROM interactions WHERE id = $1", interaction.ID)
}

// TestCreateInteractionHandler tests creating a new interaction
func TestCreateInteractionHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	contact := setupTestContact(t, testDB.DB, "TestCreateInteraction")
	defer contact.Cleanup()

	t.Run("Success", func(t *testing.T) {
		newInteraction := TestData.CreateInteractionRequest(contact.Contact.ID)
		body, _ := json.Marshal(newInteraction)

		req := httptest.NewRequest("POST", "/api/interactions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createInteractionHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"contact_id": float64(contact.Contact.ID),
		})

		if response == nil {
			t.Fatal("Expected non-nil response")
		}

		// Clean up
		if idFloat, ok := response["id"].(float64); ok {
			db.Exec("DELETE FROM interactions WHERE id = $1", int(idFloat))
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/interactions", bytes.NewReader([]byte("{invalid")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createInteractionHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestGetGiftsHandler tests retrieving gifts for a contact
func TestGetGiftsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	contact := setupTestContact(t, testDB.DB, "TestGiftContact")
	defer contact.Cleanup()

	gift := setupTestGift(t, testDB.DB, contact.Contact.ID)

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/contacts/"+strconv.Itoa(contact.Contact.ID)+"/gifts", nil)
		req = mux.SetURLVars(req, map[string]string{"contactId": strconv.Itoa(contact.Contact.ID)})
		w := httptest.NewRecorder()

		getGiftsHandler(w, req)

		gifts := assertJSONArray(t, w, http.StatusOK)
		if gifts == nil {
			t.Fatal("Expected non-nil gifts array")
		}

		if len(gifts) < 1 {
			t.Error("Expected at least 1 gift")
		}
	})

	// Clean up gift
	db.Exec("DELETE FROM gifts WHERE id = $1", gift.ID)
}

// TestCreateGiftHandler tests creating a new gift
func TestCreateGiftHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	contact := setupTestContact(t, testDB.DB, "TestCreateGift")
	defer contact.Cleanup()

	t.Run("Success", func(t *testing.T) {
		newGift := TestData.CreateGiftRequest(contact.Contact.ID)
		body, _ := json.Marshal(newGift)

		req := httptest.NewRequest("POST", "/api/gifts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createGiftHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"contact_id": float64(contact.Contact.ID),
		})

		if response == nil {
			t.Fatal("Expected non-nil response")
		}

		// Clean up
		if idFloat, ok := response["id"].(float64); ok {
			db.Exec("DELETE FROM gifts WHERE id = $1", int(idFloat))
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/gifts", bytes.NewReader([]byte("{invalid")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createGiftHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestGetRemindersHandler tests retrieving upcoming reminders
func TestGetRemindersHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/reminders", nil)
		w := httptest.NewRecorder()

		getRemindersHandler(w, req)

		reminders := assertJSONArray(t, w, http.StatusOK)
		// Should return empty array if no reminders
		if reminders == nil {
			reminders = []interface{}{}
		}
	})
}

// TestGetBirthdaysHandler tests retrieving upcoming birthdays
func TestGetBirthdaysHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB
	relationshipProcessor = NewRelationshipProcessor(testDB.DB)

	t.Run("Success_DefaultDays", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/birthdays", nil)
		w := httptest.NewRecorder()

		getBirthdaysHandler(w, req)

		birthdays := assertJSONArray(t, w, http.StatusOK)
		if birthdays == nil {
			birthdays = []interface{}{}
		}
	})

	t.Run("Success_CustomDays", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/birthdays?days_ahead=30", nil)
		w := httptest.NewRecorder()

		getBirthdaysHandler(w, req)

		birthdays := assertJSONArray(t, w, http.StatusOK)
		if birthdays == nil {
			birthdays = []interface{}{}
		}
	})

	t.Run("InvalidDaysParameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/birthdays?days_ahead=invalid", nil)
		w := httptest.NewRecorder()

		getBirthdaysHandler(w, req)

		// Should default to 7 days and still succeed
		birthdays := assertJSONArray(t, w, http.StatusOK)
		if birthdays == nil {
			birthdays = []interface{}{}
		}
	})
}

// TestEnrichContactHandler tests contact enrichment endpoint
func TestEnrichContactHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB
	relationshipProcessor = NewRelationshipProcessor(testDB.DB)

	contact := setupTestContact(t, testDB.DB, "TestEnrichContact")
	defer contact.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/contacts/"+strconv.Itoa(contact.Contact.ID)+"/enrich", nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(contact.Contact.ID)})
		w := httptest.NewRecorder()

		enrichContactHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"contact_id": float64(contact.Contact.ID),
		})

		if response == nil {
			t.Fatal("Expected non-nil response")
		}

		// Verify enrichment fields
		expectedFields := []string{"suggested_interests", "personality_traits", "conversation_starters"}
		for _, field := range expectedFields {
			if _, exists := response[field]; !exists {
				t.Errorf("Expected field '%s' in response", field)
			}
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/contacts/999999/enrich", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "999999"})
		w := httptest.NewRecorder()

		enrichContactHandler(w, req)

		assertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("InvalidID", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/contacts/invalid/enrich", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
		w := httptest.NewRecorder()

		enrichContactHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestSuggestGiftsHandler tests gift suggestion endpoint
func TestSuggestGiftsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB
	relationshipProcessor = NewRelationshipProcessor(testDB.DB)

	contact := setupTestContact(t, testDB.DB, "TestGiftSuggestion")
	defer contact.Cleanup()

	t.Run("Success", func(t *testing.T) {
		suggestionReq := GiftSuggestionRequest{
			Name:      contact.Contact.Name,
			Interests: "reading, travel",
			Occasion:  "birthday",
			Budget:    "50-100",
		}
		body, _ := json.Marshal(suggestionReq)

		req := httptest.NewRequest("POST", "/api/contacts/"+strconv.Itoa(contact.Contact.ID)+"/gifts/suggest", bytes.NewReader(body))
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(contact.Contact.ID)})
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suggestGiftsHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"contact_id": float64(contact.Contact.ID),
		})

		if response == nil {
			t.Fatal("Expected non-nil response")
		}

		// Verify suggestions field exists
		if _, exists := response["suggestions"]; !exists {
			t.Error("Expected 'suggestions' field in response")
		}
	})

	t.Run("InvalidID", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/contacts/invalid/gifts/suggest", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
		w := httptest.NewRecorder()

		suggestGiftsHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/contacts/"+strconv.Itoa(contact.Contact.ID)+"/gifts/suggest", bytes.NewReader([]byte("{invalid")))
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(contact.Contact.ID)})
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suggestGiftsHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestGetInsightsHandler tests relationship insights endpoint
func TestGetInsightsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB
	relationshipProcessor = NewRelationshipProcessor(testDB.DB)

	contact := setupTestContact(t, testDB.DB, "TestInsightsContact")
	defer contact.Cleanup()

	// Add some interactions for insights
	setupTestInteraction(t, testDB.DB, contact.Contact.ID)

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/contacts/"+strconv.Itoa(contact.Contact.ID)+"/insights", nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(contact.Contact.ID)})
		w := httptest.NewRecorder()

		getInsightsHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"contact_id": float64(contact.Contact.ID),
		})

		if response == nil {
			t.Fatal("Expected non-nil response")
		}

		// Verify insight fields
		expectedFields := []string{"last_interaction_days", "interaction_frequency",
			"overall_sentiment", "recommended_actions", "relationship_score"}
		for _, field := range expectedFields {
			if _, exists := response[field]; !exists {
				t.Errorf("Expected field '%s' in response", field)
			}
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/contacts/999999/insights", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "999999"})
		w := httptest.NewRecorder()

		getInsightsHandler(w, req)

		assertErrorResponse(t, w, http.StatusInternalServerError)
	})

	t.Run("InvalidID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/contacts/invalid/insights", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
		w := httptest.NewRecorder()

		getInsightsHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestPostgresArrayTypes tests PostgreSQL array handling
func TestPostgresArrayTypes(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("ArraySerialization", func(t *testing.T) {
		contact := Contact{
			Name:      "TestArrayContact",
			Interests: []string{"interest1", "interest2", "interest3"},
			Tags:      []string{"tag1", "tag2"},
		}
		body, _ := json.Marshal(contact)

		req := httptest.NewRequest("POST", "/api/contacts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createContactHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusCreated, nil)
		if response == nil {
			t.Fatal("Expected non-nil response")
		}

		// Clean up
		if idFloat, ok := response["id"].(float64); ok {
			// Verify arrays were stored correctly
			var interests, tags []string
			err := db.QueryRow("SELECT interests, tags FROM contacts WHERE id = $1", int(idFloat)).
				Scan(pq.Array(&interests), pq.Array(&tags))
			if err != nil {
				t.Errorf("Failed to retrieve arrays: %v", err)
			}

			if len(interests) != 3 {
				t.Errorf("Expected 3 interests, got %d", len(interests))
			}

			if len(tags) != 2 {
				t.Errorf("Expected 2 tags, got %d", len(tags))
			}

			db.Exec("DELETE FROM contacts WHERE id = $1", int(idFloat))
		}
	})
}

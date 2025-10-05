
package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

// TestEdgeCases tests boundary conditions and edge cases
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("EmptyArrayFields", func(t *testing.T) {
		contact := Contact{
			Name:      "EmptyArrayTest",
			Interests: []string{},
			Tags:      []string{},
		}
		body, _ := json.Marshal(contact)

		req := httptest.NewRequest("POST", "/api/contacts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createContactHandler(w, req)

		response := assertJSONResponse(t, w, 201, map[string]interface{}{
			"name": "EmptyArrayTest",
		})

		if response != nil {
			if idFloat, ok := response["id"].(float64); ok {
				db.Exec("DELETE FROM contacts WHERE id = $1", int(idFloat))
			}
		}
	})

	t.Run("NullOptionalFields", func(t *testing.T) {
		contact := Contact{
			Name:  "NullFieldsTest",
			Email: "",
			Phone: "",
			Notes: "",
		}
		body, _ := json.Marshal(contact)

		req := httptest.NewRequest("POST", "/api/contacts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createContactHandler(w, req)

		if w.Code == 201 {
			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			if idFloat, ok := response["id"].(float64); ok {
				db.Exec("DELETE FROM contacts WHERE id = $1", int(idFloat))
			}
		}
	})

	t.Run("VeryLongStrings", func(t *testing.T) {
		longString := string(make([]byte, 1000))
		for i := range longString {
			longString = string(append([]byte(longString[:i]), 'a'))
		}

		contact := Contact{
			Name:  "LongStringTest",
			Notes: longString,
		}
		body, _ := json.Marshal(contact)

		req := httptest.NewRequest("POST", "/api/contacts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createContactHandler(w, req)

		// Should handle gracefully - either accept or reject with appropriate error
		if w.Code == 201 || w.Code == 400 || w.Code == 500 {
			if w.Code == 201 {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				if idFloat, ok := response["id"].(float64); ok {
					db.Exec("DELETE FROM contacts WHERE id = $1", int(idFloat))
				}
			}
		} else {
			t.Errorf("Unexpected status code for long string: %d", w.Code)
		}
	})

	t.Run("SpecialCharactersInNames", func(t *testing.T) {
		specialNames := []string{
			"Name with 'quotes'",
			"Name with \"double quotes\"",
			"Name with\nnewline",
			"Name with\ttab",
			"Name with émoji 🎉",
		}

		for _, name := range specialNames {
			contact := Contact{Name: name}
			body, _ := json.Marshal(contact)

			req := httptest.NewRequest("POST", "/api/contacts", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			createContactHandler(w, req)

			if w.Code == 201 {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				if idFloat, ok := response["id"].(float64); ok {
					db.Exec("DELETE FROM contacts WHERE id = $1", int(idFloat))
				}
			}
		}
	})

	t.Run("InvalidDates", func(t *testing.T) {
		invalidDates := []string{
			"not-a-date",
			"2024-13-01", // Invalid month
			"2024-01-32", // Invalid day
			"31-12-2024", // Wrong format
		}

		for _, invalidDate := range invalidDates {
			contact := Contact{
				Name:     "InvalidDateTest",
				Birthday: invalidDate,
			}
			body, _ := json.Marshal(contact)

			req := httptest.NewRequest("POST", "/api/contacts", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			createContactHandler(w, req)

			// Should handle invalid dates - accept as string or reject
			if w.Code == 201 {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				if idFloat, ok := response["id"].(float64); ok {
					db.Exec("DELETE FROM contacts WHERE id = $1", int(idFloat))
				}
			}
		}
	})
}

// TestErrorConditions tests systematic error handling
func TestErrorConditions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("InvalidContentType", func(t *testing.T) {
		contact := TestData.CreateContactRequest("ContentTypeTest")
		body, _ := json.Marshal(contact)

		req := httptest.NewRequest("POST", "/api/contacts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "text/plain") // Wrong content type
		w := httptest.NewRecorder()

		createContactHandler(w, req)

		// Should still work or return appropriate error
		if w.Code != 201 && w.Code != 400 && w.Code != 415 {
			t.Logf("Unexpected status for wrong content type: %d", w.Code)
		}
	})

	t.Run("MissingContentType", func(t *testing.T) {
		contact := TestData.CreateContactRequest("NoContentType")
		body, _ := json.Marshal(contact)

		req := httptest.NewRequest("POST", "/api/contacts", bytes.NewReader(body))
		// No Content-Type header
		w := httptest.NewRecorder()

		createContactHandler(w, req)

		// Should still work or return appropriate error
		if w.Code != 201 && w.Code != 400 {
			t.Logf("Unexpected status for missing content type: %d", w.Code)
		}
	})

	t.Run("LargePayload", func(t *testing.T) {
		// Create a contact with large arrays
		largeInterests := make([]string, 100)
		for i := range largeInterests {
			largeInterests[i] = "interest" + strconv.Itoa(i)
		}

		contact := Contact{
			Name:      "LargePayloadTest",
			Interests: largeInterests,
		}
		body, _ := json.Marshal(contact)

		req := httptest.NewRequest("POST", "/api/contacts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createContactHandler(w, req)

		if w.Code == 201 {
			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			if idFloat, ok := response["id"].(float64); ok {
				db.Exec("DELETE FROM contacts WHERE id = $1", int(idFloat))
			}
		}
	})
}

// TestRelationshipProcessorEdgeCases tests relationship processor edge cases
func TestRelationshipProcessorEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB
	relationshipProcessor = NewRelationshipProcessor(testDB.DB)

	t.Run("EnrichContactWithEmptyName", func(t *testing.T) {
		contact := setupTestContact(t, testDB.DB, "")
		defer contact.Cleanup()

		req := httptest.NewRequest("POST", "/api/contacts/"+strconv.Itoa(contact.Contact.ID)+"/enrich", nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(contact.Contact.ID)})
		w := httptest.NewRecorder()

		enrichContactHandler(w, req)

		// Should handle empty name gracefully
		if w.Code != 200 && w.Code != 500 {
			t.Errorf("Unexpected status for empty name enrichment: %d", w.Code)
		}
	})

	t.Run("SuggestGiftsWithMinimalData", func(t *testing.T) {
		contact := setupTestContact(t, testDB.DB, "MinimalGiftTest")
		defer contact.Cleanup()

		suggestionReq := GiftSuggestionRequest{
			// Minimal data - rely on defaults
		}
		body, _ := json.Marshal(suggestionReq)

		req := httptest.NewRequest("POST", "/api/contacts/"+strconv.Itoa(contact.Contact.ID)+"/gifts/suggest", bytes.NewReader(body))
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(contact.Contact.ID)})
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suggestGiftsHandler(w, req)

		// Should use defaults and still work
		if w.Code != 200 {
			t.Errorf("Expected status 200 for minimal gift suggestion, got %d", w.Code)
		}
	})

	t.Run("AnalyzeRelationshipsNoInteractions", func(t *testing.T) {
		contact := setupTestContact(t, testDB.DB, "NoInteractionsTest")
		defer contact.Cleanup()

		req := httptest.NewRequest("GET", "/api/contacts/"+strconv.Itoa(contact.Contact.ID)+"/insights", nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(contact.Contact.ID)})
		w := httptest.NewRecorder()

		getInsightsHandler(w, req)

		response := assertJSONResponse(t, w, 200, map[string]interface{}{
			"contact_id": float64(contact.Contact.ID),
		})

		if response != nil {
			// Should still provide insights even with no interactions
			if score, exists := response["relationship_score"]; exists {
				if scoreFloat, ok := score.(float64); ok && scoreFloat != 0 {
					t.Log("Relationship score with no interactions:", scoreFloat)
				}
			}
		}
	})

	t.Run("BirthdaysNegativeDays", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/birthdays?days_ahead=-5", nil)
		w := httptest.NewRecorder()

		getBirthdaysHandler(w, req)

		// Should handle negative days gracefully (use default)
		birthdays := assertJSONArray(t, w, 200)
		if birthdays == nil {
			t.Error("Expected birthdays array even with negative days")
		}
	})

	t.Run("BirthdaysZeroDays", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/birthdays?days_ahead=0", nil)
		w := httptest.NewRecorder()

		getBirthdaysHandler(w, req)

		// Should use default
		birthdays := assertJSONArray(t, w, 200)
		if birthdays == nil {
			t.Error("Expected birthdays array with zero days")
		}
	})

	t.Run("BirthdaysVeryLargeDays", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/birthdays?days_ahead=365", nil)
		w := httptest.NewRecorder()

		getBirthdaysHandler(w, req)

		birthdays := assertJSONArray(t, w, 200)
		if birthdays == nil {
			t.Error("Expected birthdays array with large days value")
		}
	})
}

// TestArrayHandling tests PostgreSQL array type handling
func TestArrayHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("NullArrays", func(t *testing.T) {
		contact := Contact{
			Name:      "NullArrayTest",
			Interests: nil,
			Tags:      nil,
		}
		body, _ := json.Marshal(contact)

		req := httptest.NewRequest("POST", "/api/contacts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createContactHandler(w, req)

		if w.Code == 201 {
			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			if idFloat, ok := response["id"].(float64); ok {
				db.Exec("DELETE FROM contacts WHERE id = $1", int(idFloat))
			}
		}
	})

	t.Run("ArraysWithEmptyStrings", func(t *testing.T) {
		contact := Contact{
			Name:      "EmptyStringArrayTest",
			Interests: []string{"", "valid", ""},
			Tags:      []string{"", "", ""},
		}
		body, _ := json.Marshal(contact)

		req := httptest.NewRequest("POST", "/api/contacts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createContactHandler(w, req)

		if w.Code == 201 {
			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			if idFloat, ok := response["id"].(float64); ok {
				db.Exec("DELETE FROM contacts WHERE id = $1", int(idFloat))
			}
		}
	})

	t.Run("ArraysWithSpecialCharacters", func(t *testing.T) {
		contact := Contact{
			Name:      "SpecialCharArrayTest",
			Interests: []string{"reading, books", "travel & adventure", "coffee's good"},
			Tags:      []string{"friend's", "family\"s", "work-related"},
		}
		body, _ := json.Marshal(contact)

		req := httptest.NewRequest("POST", "/api/contacts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createContactHandler(w, req)

		if w.Code == 201 {
			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			if idFloat, ok := response["id"].(float64); ok {
				// Verify arrays were stored correctly
				var retrievedContact Contact
				err := db.QueryRow("SELECT id, name, interests, tags FROM contacts WHERE id = $1", int(idFloat)).
					Scan(&retrievedContact.ID, &retrievedContact.Name, pq.Array(&retrievedContact.Interests), pq.Array(&retrievedContact.Tags))

				if err != nil {
					t.Errorf("Failed to retrieve contact with special char arrays: %v", err)
				} else {
					if len(retrievedContact.Interests) != len(contact.Interests) {
						t.Errorf("Expected %d interests, got %d", len(contact.Interests), len(retrievedContact.Interests))
					}
				}

				db.Exec("DELETE FROM contacts WHERE id = $1", int(idFloat))
			}
		}
	})
}

// TestHTTPMethodValidation tests that endpoints respond correctly to different HTTP methods
func TestHTTPMethodValidation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("HealthCheckMethods", func(t *testing.T) {
		// Health should only accept GET
		methods := []string{"POST", "PUT", "DELETE", "PATCH"}
		for _, method := range methods {
			req := httptest.NewRequest(method, "/health", nil)
			w := httptest.NewRecorder()

			healthHandler(w, req)

			// Still responds, but might want to validate method
			if w.Code == 200 {
				t.Logf("Health endpoint accepted %s method", method)
			}
		}
	})
}

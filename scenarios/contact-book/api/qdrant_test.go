package main

import (
	"math"
	"testing"

	"github.com/google/uuid"
)

// TestQdrantClient tests the Qdrant client functionality
func TestNewQdrantClient(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_DefaultURL", func(t *testing.T) {
		client := NewQdrantClient()
		if client == nil {
			t.Fatal("Expected non-nil client")
		}

		if client.baseURL == "" {
			t.Error("Expected base URL to be set")
		}

		if client.collection != "contact_book_persons" {
			t.Errorf("Expected collection 'contact_book_persons', got %s", client.collection)
		}

		if client.httpClient == nil {
			t.Error("Expected HTTP client to be initialized")
		}
	})
}

// TestGenerateEmbedding tests the embedding generation
func TestGenerateEmbedding(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	client := NewQdrantClient()

	t.Run("Success_SimpleText", func(t *testing.T) {
		text := "John Doe software engineer"
		embedding, err := client.GenerateEmbedding(text)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(embedding) != 384 {
			t.Errorf("Expected embedding length 384, got %d", len(embedding))
		}

		// Verify normalization (sum of squares should be approximately 1)
		var sumSquares float32
		for _, val := range embedding {
			sumSquares += val * val
		}
		magnitude := float32(math.Sqrt(float64(sumSquares)))

		// Allow for floating point precision errors
		if magnitude < 0.99 || magnitude > 1.01 {
			t.Errorf("Expected normalized vector (magnitude ~1), got %f", magnitude)
		}
	})

	t.Run("Success_EmptyText", func(t *testing.T) {
		embedding, err := client.GenerateEmbedding("")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(embedding) != 384 {
			t.Errorf("Expected embedding length 384, got %d", len(embedding))
		}
	})

	t.Run("Success_LongText", func(t *testing.T) {
		longText := "This is a very long text with many words and lots of information about a person including their interests hobbies and professional background"
		embedding, err := client.GenerateEmbedding(longText)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(embedding) != 384 {
			t.Errorf("Expected embedding length 384, got %d", len(embedding))
		}
	})

	t.Run("Deterministic_SameInputSameOutput", func(t *testing.T) {
		text := "Test person name"

		embedding1, err1 := client.GenerateEmbedding(text)
		if err1 != nil {
			t.Fatalf("First embedding failed: %v", err1)
		}

		embedding2, err2 := client.GenerateEmbedding(text)
		if err2 != nil {
			t.Fatalf("Second embedding failed: %v", err2)
		}

		// Check that embeddings are identical
		for i := range embedding1 {
			if embedding1[i] != embedding2[i] {
				t.Errorf("Embeddings differ at index %d: %f vs %f", i, embedding1[i], embedding2[i])
				break
			}
		}
	})

	t.Run("Different_DifferentInputDifferentOutput", func(t *testing.T) {
		text1 := "John Doe"
		text2 := "Jane Smith"

		embedding1, err1 := client.GenerateEmbedding(text1)
		if err1 != nil {
			t.Fatalf("First embedding failed: %v", err1)
		}

		embedding2, err2 := client.GenerateEmbedding(text2)
		if err2 != nil {
			t.Fatalf("Second embedding failed: %v", err2)
		}

		// Check that embeddings are different
		identical := true
		for i := range embedding1 {
			if embedding1[i] != embedding2[i] {
				identical = false
				break
			}
		}

		if identical {
			t.Error("Expected different embeddings for different inputs")
		}
	})
}

// TestGetStringValue tests the helper function
func TestGetStringValue(t *testing.T) {
	t.Run("NilPointer", func(t *testing.T) {
		result := getStringValue(nil)
		if result != "" {
			t.Errorf("Expected empty string for nil, got %s", result)
		}
	})

	t.Run("ValidPointer", func(t *testing.T) {
		value := "test value"
		result := getStringValue(&value)
		if result != "test value" {
			t.Errorf("Expected 'test value', got %s", result)
		}
	})

	t.Run("EmptyString", func(t *testing.T) {
		value := ""
		result := getStringValue(&value)
		if result != "" {
			t.Errorf("Expected empty string, got %s", result)
		}
	})
}

// TestQdrantIntegration tests Qdrant integration (will skip if Qdrant is not available)
func TestQdrantIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	client := NewQdrantClient()

	// Try to create collection - if this fails, Qdrant is not available
	err := client.CreateCollection()
	if err != nil {
		t.Skipf("Skipping Qdrant integration test - Qdrant not available: %v", err)
		return
	}

	t.Run("IndexAndSearchPerson", func(t *testing.T) {
		// Create test person with proper UUID
		displayName := "John"
		notes := "Software engineer with expertise in Go"
		person := Person{
			ID:          uuid.New().String(),
			FullName:    "John Doe",
			DisplayName: &displayName,
			Emails:      []string{"john@example.com"},
			Tags:        []string{"engineer", "golang"},
			Notes:       &notes,
		}

		// Index the person
		err := client.IndexPerson(person)
		if err != nil {
			t.Fatalf("Failed to index person: %v", err)
		}

		// Search for the person
		results, err := client.SearchPersons("software engineer golang", 5)
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}

		// Verify results
		found := false
		for _, result := range results {
			if result.ID == person.ID {
				found = true
				if result.Score <= 0 {
					t.Error("Expected positive relevance score")
				}
				break
			}
		}

		if !found {
			t.Error("Expected to find indexed person in search results")
		}

		// Cleanup
		if err := client.DeletePerson(person.ID); err != nil {
			t.Logf("Warning: Failed to cleanup test person: %v", err)
		}
	})

	t.Run("SearchNonExistent", func(t *testing.T) {
		// Search for something that doesn't exist
		results, err := client.SearchPersons("very unique nonexistent query 123456789", 5)
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}

		// Results may be empty or have low scores
		if len(results) > 0 {
			// If there are results, scores should be low
			if results[0].Score > 0.5 {
				t.Logf("Note: Search returned results with score %f for nonexistent query", results[0].Score)
			}
		}
	})

	t.Run("DeleteNonExistentPerson", func(t *testing.T) {
		// Deleting non-existent person with proper UUID format
		err := client.DeletePerson(uuid.New().String())
		// This may or may not error depending on Qdrant configuration
		// Just log the result
		if err != nil {
			t.Logf("Delete non-existent person returned: %v", err)
		}
	})
}

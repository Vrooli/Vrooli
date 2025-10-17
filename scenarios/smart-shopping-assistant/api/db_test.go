// +build testing

package main

import (
	"context"
	"os"
	"testing"
)

// TestDatabaseOperations tests database operations
func TestDatabaseOperations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, err := NewDatabase()
	if err != nil {
		t.Logf("Database initialization returned error: %v (continuing with limited tests)", err)
	}
	if db != nil {
		defer db.Close()
	}

	ctx := context.Background()

	t.Run("Database_SearchProducts", func(t *testing.T) {
		if db == nil {
			t.Skip("Database not available")
		}

		products, err := db.SearchProducts(ctx, "laptop", 1000.00)
		if err != nil {
			t.Errorf("SearchProducts failed: %v", err)
			return
		}

		if len(products) == 0 {
			t.Error("Expected at least one product")
		}

		// Validate product structure
		for _, product := range products {
			if product.ID == "" {
				t.Error("Expected product to have ID")
			}
			if product.Name == "" {
				t.Error("Expected product to have name")
			}
			if product.CurrentPrice <= 0 {
				t.Error("Expected product to have valid current price")
			}
		}
	})

	t.Run("Database_SearchProducts_WithBudget", func(t *testing.T) {
		if db == nil {
			t.Skip("Database not available")
		}

		budget := 500.00
		products, err := db.SearchProducts(ctx, "headphones", budget)
		if err != nil {
			t.Errorf("SearchProducts failed: %v", err)
			return
		}

		// Verify products are within budget
		for _, product := range products {
			if product.CurrentPrice > budget {
				t.Errorf("Product price %.2f exceeds budget %.2f", product.CurrentPrice, budget)
			}
		}
	})

	t.Run("Database_FindAlternatives", func(t *testing.T) {
		if db == nil {
			t.Skip("Database not available")
		}

		productID := "test-product-123"
		price := 199.99

		alternatives := db.FindAlternatives(ctx, productID, price)

		if len(alternatives) == 0 {
			t.Error("Expected at least one alternative")
		}

		// Validate alternatives
		for _, alt := range alternatives {
			if alt.Product.ID == "" {
				t.Error("Expected alternative product to have ID")
			}
			if alt.SavingsAmount <= 0 {
				t.Error("Expected alternative to have positive savings")
			}
			if alt.AlternativeType == "" {
				t.Error("Expected alternative to have type")
			}
			if alt.Reason == "" {
				t.Error("Expected alternative to have reason")
			}
		}
	})

	t.Run("Database_GetPriceHistory", func(t *testing.T) {
		if db == nil {
			t.Skip("Database not available")
		}

		productID := "test-product-456"
		insights := db.GetPriceHistory(ctx, productID)

		if insights.CurrentTrend == "" {
			t.Error("Expected price trend information")
		}
		if insights.HistoricalLow <= 0 {
			t.Error("Expected valid historical low price")
		}
		if insights.HistoricalHigh <= 0 {
			t.Error("Expected valid historical high price")
		}
		if insights.HistoricalLow > insights.HistoricalHigh {
			t.Error("Historical low should be less than historical high")
		}
	})

	t.Run("Database_GenerateAffiliateLinks", func(t *testing.T) {
		if db == nil {
			t.Skip("Database not available")
		}

		products := []Product{
			{
				ID:           "prod-1",
				Name:         "Test Product 1",
				CurrentPrice: 99.99,
			},
			{
				ID:           "prod-2",
				Name:         "Test Product 2",
				CurrentPrice: 149.99,
			},
		}

		links := db.GenerateAffiliateLinks(products)

		if len(links) == 0 {
			t.Error("Expected affiliate links")
		}

		// Validate links
		for _, link := range links {
			if link.ProductID == "" {
				t.Error("Expected link to have product ID")
			}
			if link.Retailer == "" {
				t.Error("Expected link to have retailer")
			}
			if link.URL == "" {
				t.Error("Expected link to have URL")
			}
			if link.Commission < 0 {
				t.Error("Expected non-negative commission")
			}
		}

		// Verify we have links for multiple retailers per product
		productLinks := make(map[string]int)
		for _, link := range links {
			productLinks[link.ProductID]++
		}

		for _, count := range productLinks {
			if count < 2 {
				t.Error("Expected multiple retailer links per product")
			}
		}
	})

	t.Run("Database_CallDeepResearch", func(t *testing.T) {
		if db == nil {
			t.Skip("Database not available")
		}

		// Test with deep research unavailable
		products := db.callDeepResearch(ctx, "smartphone", 500.00)

		// Should return nil when deep research is not available
		if products != nil {
			// If we got products, validate them
			for _, product := range products {
				if product.ID == "" {
					t.Error("Expected product to have ID")
				}
				if product.CurrentPrice <= 0 {
					t.Error("Expected valid price")
				}
			}
		}
	})

	t.Run("Database_CacheTest", func(t *testing.T) {
		if db == nil || db.redis == nil {
			t.Skip("Redis not available")
		}

		query := "test-query-cache"
		budget := 300.00

		// First call - should cache
		products1, err := db.SearchProducts(ctx, query, budget)
		if err != nil {
			t.Errorf("First search failed: %v", err)
		}

		// Second call - should hit cache
		products2, err := db.SearchProducts(ctx, query, budget)
		if err != nil {
			t.Errorf("Second search failed: %v", err)
		}

		if len(products1) != len(products2) {
			t.Error("Cache should return same number of products")
		}
	})
}

// TestDatabaseConnectionHandling tests connection error handling
func TestDatabaseConnectionHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Database_InvalidPostgresConnection", func(t *testing.T) {
		// Test with invalid postgres connection
		oldHost := os.Getenv("POSTGRES_HOST")
		os.Setenv("POSTGRES_HOST", "invalid-host-12345")
		defer os.Setenv("POSTGRES_HOST", oldHost)

		db, err := NewDatabase()
		// Should not return error, but postgres should be nil
		if err != nil {
			t.Logf("NewDatabase returned error: %v", err)
		}
		if db != nil {
			if db.postgres != nil {
				// Connection might succeed if invalid-host resolves
				t.Logf("Postgres connection unexpectedly succeeded")
			}
			db.Close()
		}
	})

	t.Run("Database_InvalidRedisConnection", func(t *testing.T) {
		// Test with invalid redis connection
		oldHost := os.Getenv("REDIS_HOST")
		os.Setenv("REDIS_HOST", "invalid-redis-host")
		defer os.Setenv("REDIS_HOST", oldHost)

		db, err := NewDatabase()
		if err != nil {
			t.Logf("NewDatabase returned error: %v", err)
		}
		if db != nil {
			if db.redis != nil {
				t.Logf("Redis connection unexpectedly succeeded")
			}
			db.Close()
		}
	})

	t.Run("Database_FallbackToMockData", func(t *testing.T) {
		// Create database with no connections
		db := &Database{
			postgres: nil,
			redis:    nil,
		}
		defer db.Close()

		ctx := context.Background()
		products, err := db.SearchProducts(ctx, "test", 100.00)

		// Should return mock data even without database
		if err != nil {
			t.Errorf("Expected SearchProducts to work with mock data, got error: %v", err)
		}

		if len(products) == 0 {
			t.Error("Expected mock products when database is unavailable")
		}
	})
}

// TestDatabaseEdgeCases tests edge cases and boundary conditions
func TestDatabaseEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, err := NewDatabase()
	if err != nil {
		t.Logf("Database initialization returned error: %v", err)
	}
	if db != nil {
		defer db.Close()
	}

	ctx := context.Background()

	t.Run("Database_EmptyQuery", func(t *testing.T) {
		if db == nil {
			t.Skip("Database not available")
		}

		products, err := db.SearchProducts(ctx, "", 1000.00)
		if err != nil {
			t.Logf("SearchProducts with empty query returned error: %v", err)
		}

		// Should handle empty query gracefully
		if products != nil && len(products) > 0 {
			t.Log("Empty query returned products (acceptable behavior)")
		}
	})

	t.Run("Database_ZeroBudget", func(t *testing.T) {
		if db == nil {
			t.Skip("Database not available")
		}

		products, err := db.SearchProducts(ctx, "laptop", 0.00)
		if err != nil {
			t.Logf("SearchProducts with zero budget returned error: %v", err)
		}

		// Behavior with zero budget
		if products != nil {
			t.Logf("Zero budget returned %d products", len(products))
		}
	})

	t.Run("Database_NegativeBudget", func(t *testing.T) {
		if db == nil {
			t.Skip("Database not available")
		}

		products, err := db.SearchProducts(ctx, "laptop", -100.00)
		if err != nil {
			t.Logf("SearchProducts with negative budget returned error: %v", err)
		}

		// Should handle negative budget
		if products != nil {
			t.Logf("Negative budget returned %d products", len(products))
		}
	})

	t.Run("Database_VeryLargeBudget", func(t *testing.T) {
		if db == nil {
			t.Skip("Database not available")
		}

		products, err := db.SearchProducts(ctx, "luxury item", 1000000.00)
		if err != nil {
			t.Errorf("SearchProducts with large budget failed: %v", err)
		}

		if len(products) == 0 {
			t.Error("Expected products even with very large budget")
		}
	})

	t.Run("Database_SpecialCharactersInQuery", func(t *testing.T) {
		if db == nil {
			t.Skip("Database not available")
		}

		queries := []string{
			"laptop's",
			"<script>alert('test')</script>",
			"product with \"quotes\"",
			"item & accessories",
		}

		for _, query := range queries {
			products, err := db.SearchProducts(ctx, query, 500.00)
			if err != nil {
				t.Errorf("SearchProducts failed with query '%s': %v", query, err)
			}
			if products == nil {
				t.Errorf("Expected products for query: %s", query)
			}
		}
	})
}

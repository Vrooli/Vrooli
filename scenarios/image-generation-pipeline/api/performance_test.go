// +build testing

package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestHealthHandlerPerformance tests health endpoint performance
func TestHealthHandlerPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	pattern := PerformanceTestPattern{
		Name:        "HealthHandler",
		Description: "Test health handler response time",
		MaxDuration: 100 * time.Millisecond,
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			start := time.Now()

			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			}

			w, httpReq, err := makeHTTPRequest(req)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			healthHandler(w, httpReq)

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			return time.Since(start)
		},
	}

	RunPerformanceTest(t, pattern)
}

// TestCampaignsHandlerConcurrency tests campaigns endpoint under concurrent load
func TestCampaignsHandlerConcurrency(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	pattern := ConcurrencyTestPattern{
		Name:        "CampaignsHandler_ConcurrentReads",
		Description: "Test campaigns handler under concurrent read load",
		Concurrency: 10,
		Iterations:  50,
		Setup: func(t *testing.T) interface{} {
			// Create test brand
			testBrand := setupTestBrand(t, "Concurrency Test Brand")

			// Create test campaigns
			campaigns := make([]*TestCampaign, 5)
			for i := 0; i < 5; i++ {
				campaigns[i] = setupTestCampaign(t, fmt.Sprintf("Campaign %d", i), testBrand.Brand.ID)
			}

			return map[string]interface{}{
				"brand":     testBrand,
				"campaigns": campaigns,
			}
		},
		Execute: func(t *testing.T, setupData interface{}, iteration int) error {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/campaigns",
			}

			w, httpReq, err := makeHTTPRequest(req)
			if err != nil {
				return err
			}

			getCampaigns(w, httpReq)

			if w.Code != 200 {
				return fmt.Errorf("expected status 200, got %d", w.Code)
			}

			return nil
		},
		Validate: func(t *testing.T, setupData interface{}, results []error) {
			errorCount := 0
			for _, err := range results {
				if err != nil {
					errorCount++
					t.Logf("Error in concurrent execution: %v", err)
				}
			}

			if errorCount > 0 {
				t.Errorf("Expected 0 errors in concurrent execution, got %d", errorCount)
			}
		},
		Cleanup: func(setupData interface{}) {
			if setupData != nil {
				data := setupData.(map[string]interface{})
				if brand, ok := data["brand"].(*TestBrand); ok {
					brand.Cleanup()
				}
				if campaigns, ok := data["campaigns"].([]*TestCampaign); ok {
					for _, campaign := range campaigns {
						campaign.Cleanup()
					}
				}
			}
		},
	}

	RunConcurrencyTest(t, pattern)
}

// TestBrandsHandlerConcurrency tests brands endpoint under concurrent load
func TestBrandsHandlerConcurrency(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	pattern := ConcurrencyTestPattern{
		Name:        "BrandsHandler_ConcurrentReads",
		Description: "Test brands handler under concurrent read load",
		Concurrency: 10,
		Iterations:  50,
		Setup: func(t *testing.T) interface{} {
			// Create test brands
			brands := make([]*TestBrand, 3)
			for i := 0; i < 3; i++ {
				brands[i] = setupTestBrand(t, fmt.Sprintf("Brand %d", i))
			}

			return brands
		},
		Execute: func(t *testing.T, setupData interface{}, iteration int) error {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/brands",
			}

			w, httpReq, err := makeHTTPRequest(req)
			if err != nil {
				return err
			}

			getBrands(w, httpReq)

			if w.Code != 200 {
				return fmt.Errorf("expected status 200, got %d", w.Code)
			}

			return nil
		},
		Validate: func(t *testing.T, setupData interface{}, results []error) {
			errorCount := 0
			for _, err := range results {
				if err != nil {
					errorCount++
				}
			}

			if errorCount > 0 {
				t.Errorf("Expected 0 errors in concurrent execution, got %d", errorCount)
			}
		},
		Cleanup: func(setupData interface{}) {
			if setupData != nil {
				brands := setupData.([]*TestBrand)
				for _, brand := range brands {
					brand.Cleanup()
				}
			}
		},
	}

	RunConcurrencyTest(t, pattern)
}

// TestGenerateImageHandlerPerformance tests image generation endpoint performance
func TestGenerateImageHandlerPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	pattern := PerformanceTestPattern{
		Name:        "GenerateImageHandler",
		Description: "Test image generation handler response time",
		MaxDuration: 500 * time.Millisecond,
		Setup: func(t *testing.T) interface{} {
			testBrand := setupTestBrand(t, "Performance Test Brand")
			testCampaign := setupTestCampaign(t, "Performance Test Campaign", testBrand.Brand.ID)

			// Create mock n8n server
			mockN8N := newMockHTTPServer([]MockResponse{
				{StatusCode: 200, Body: map[string]string{"status": "ok"}},
			})

			return map[string]interface{}{
				"brand":    testBrand,
				"campaign": testCampaign,
				"mock":     mockN8N,
			}
		},
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			data := setupData.(map[string]interface{})
			campaign := data["campaign"].(*TestCampaign)

			start := time.Now()

			genReq := GenerationRequest{
				CampaignID: campaign.Campaign.ID,
				Prompt:     "Performance test prompt",
				Style:      "photographic",
				Dimensions: "1024x1024",
			}

			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/generate",
				Body:   genReq,
			}

			w, httpReq, err := makeHTTPRequest(req)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			generateImageHandler(w, httpReq)

			if w.Code != 202 {
				t.Errorf("Expected status 202, got %d", w.Code)
			}

			return time.Since(start)
		},
		Cleanup: func(setupData interface{}) {
			if setupData != nil {
				data := setupData.(map[string]interface{})
				if brand, ok := data["brand"].(*TestBrand); ok {
					brand.Cleanup()
				}
				if campaign, ok := data["campaign"].(*TestCampaign); ok {
					campaign.Cleanup()
				}
				if mock, ok := data["mock"].(*MockHTTPServer); ok {
					mock.Server.Close()
				}
			}
		},
	}

	RunPerformanceTest(t, pattern)
}

// TestGenerationsHandlerPerformance tests generations listing performance
func TestGenerationsHandlerPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	pattern := PerformanceTestPattern{
		Name:        "GenerationsHandler",
		Description: "Test generations listing handler response time",
		MaxDuration: 200 * time.Millisecond,
		Setup: func(t *testing.T) interface{} {
			testBrand := setupTestBrand(t, "Performance Test Brand")
			testCampaign := setupTestCampaign(t, "Performance Test Campaign", testBrand.Brand.ID)

			// Create multiple test generations
			generations := make([]*TestImageGeneration, 10)
			for i := 0; i < 10; i++ {
				generations[i] = setupTestImageGeneration(t, testCampaign.Campaign.ID)
			}

			return map[string]interface{}{
				"brand":       testBrand,
				"campaign":    testCampaign,
				"generations": generations,
			}
		},
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			start := time.Now()

			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/generations",
			}

			w, httpReq, err := makeHTTPRequest(req)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			generationsHandler(w, httpReq)

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			return time.Since(start)
		},
		Cleanup: func(setupData interface{}) {
			if setupData != nil {
				data := setupData.(map[string]interface{})
				if brand, ok := data["brand"].(*TestBrand); ok {
					brand.Cleanup()
				}
				if campaign, ok := data["campaign"].(*TestCampaign); ok {
					campaign.Cleanup()
				}
				if generations, ok := data["generations"].([]*TestImageGeneration); ok {
					for _, gen := range generations {
						gen.Cleanup()
					}
				}
			}
		},
	}

	RunPerformanceTest(t, pattern)
}

// TestConcurrentImageGeneration tests concurrent image generation requests
func TestConcurrentImageGeneration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	pattern := ConcurrencyTestPattern{
		Name:        "GenerateImageHandler_Concurrent",
		Description: "Test concurrent image generation requests",
		Concurrency: 5,
		Iterations:  20,
		Setup: func(t *testing.T) interface{} {
			testBrand := setupTestBrand(t, "Concurrent Gen Test Brand")
			testCampaign := setupTestCampaign(t, "Concurrent Gen Test Campaign", testBrand.Brand.ID)

			// Create mock n8n server
			mockN8N := newMockHTTPServer([]MockResponse{
				{StatusCode: 200, Body: map[string]string{"status": "ok"}},
			})

			return map[string]interface{}{
				"brand":    testBrand,
				"campaign": testCampaign,
				"mock":     mockN8N,
			}
		},
		Execute: func(t *testing.T, setupData interface{}, iteration int) error {
			data := setupData.(map[string]interface{})
			campaign := data["campaign"].(*TestCampaign)

			genReq := GenerationRequest{
				CampaignID: campaign.Campaign.ID,
				Prompt:     fmt.Sprintf("Concurrent test prompt %d", iteration),
				Style:      "photographic",
				Dimensions: "1024x1024",
			}

			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/generate",
				Body:   genReq,
			}

			w, httpReq, err := makeHTTPRequest(req)
			if err != nil {
				return err
			}

			generateImageHandler(w, httpReq)

			if w.Code != 202 {
				return fmt.Errorf("expected status 202, got %d", w.Code)
			}

			return nil
		},
		Validate: func(t *testing.T, setupData interface{}, results []error) {
			errorCount := 0
			for _, err := range results {
				if err != nil {
					errorCount++
					t.Logf("Error in concurrent execution: %v", err)
				}
			}

			if errorCount > 0 {
				t.Errorf("Expected 0 errors in concurrent execution, got %d", errorCount)
			}
		},
		Cleanup: func(setupData interface{}) {
			if setupData != nil {
				data := setupData.(map[string]interface{})
				if brand, ok := data["brand"].(*TestBrand); ok {
					brand.Cleanup()
				}
				if campaign, ok := data["campaign"].(*TestCampaign); ok {
					campaign.Cleanup()
				}
				if mock, ok := data["mock"].(*MockHTTPServer); ok {
					mock.Server.Close()
				}
			}
		},
	}

	RunConcurrencyTest(t, pattern)
}

// TestDatabaseConnectionPool tests database connection pool under load
func TestDatabaseConnectionPool(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	t.Run("ConcurrentDatabaseAccess", func(t *testing.T) {
		testBrand := setupTestBrand(t, "Pool Test Brand")
		defer testBrand.Cleanup()

		var wg sync.WaitGroup
		iterations := 100
		errors := make([]error, iterations)

		for i := 0; i < iterations; i++ {
			wg.Add(1)
			go func(iteration int) {
				defer wg.Done()

				// Test database query
				var count int
				err := db.QueryRow("SELECT COUNT(*) FROM brands WHERE id = $1", testBrand.Brand.ID).Scan(&count)
				if err != nil {
					errors[iteration] = err
					return
				}

				if count != 1 {
					errors[iteration] = fmt.Errorf("expected count 1, got %d", count)
				}
			}(i)
		}

		wg.Wait()

		errorCount := 0
		for _, err := range errors {
			if err != nil {
				errorCount++
				t.Logf("Error in concurrent database access: %v", err)
			}
		}

		if errorCount > 0 {
			t.Errorf("Expected 0 errors in concurrent database access, got %d", errorCount)
		}
	})
}

// TestMemoryUsage tests memory efficiency of handlers
func TestMemoryUsage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	t.Run("LargeResponseHandling", func(t *testing.T) {
		testBrand := setupTestBrand(t, "Memory Test Brand")
		defer testBrand.Cleanup()

		// Create many campaigns
		campaigns := make([]*TestCampaign, 100)
		for i := 0; i < 100; i++ {
			campaigns[i] = setupTestCampaign(t, fmt.Sprintf("Campaign %d", i), testBrand.Brand.ID)
		}
		defer func() {
			for _, c := range campaigns {
				c.Cleanup()
			}
		}()

		// Test fetching all campaigns
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/campaigns",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getCampaigns(w, httpReq)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		responseSize := len(w.Body.Bytes())
		if responseSize == 0 {
			t.Error("Expected non-empty response")
		}

		t.Logf("Response size for 100 campaigns: %d bytes", responseSize)
	})
}

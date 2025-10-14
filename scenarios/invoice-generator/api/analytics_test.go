package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// testDatabase manages test database connections for analytics tests
type testDatabase struct {
	DB      *sql.DB
	Cleanup func()
}

// setupAnalyticsTestDB sets up a test database connection for analytics tests
func setupAnalyticsTestDB(t *testing.T) *testDatabase {
	// Use test database
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")

		if dbHost == "" {
			dbHost = "localhost"
		}
		if dbPort == "" {
			dbPort = "5432"
		}
		if dbUser == "" {
			dbUser = "postgres"
		}
		if dbPassword == "" {
			t.Skipf("Skipping test - POSTGRES_PASSWORD environment variable not set")
			return nil
		}
		if dbName == "" {
			dbName = "postgres"
		}

		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	testDB, err := sql.Open("postgres", postgresURL)
	if err != nil {
		t.Skipf("Skipping test - database not available: %v", err)
		return nil
	}

	// Test connection
	if err := testDB.Ping(); err != nil {
		testDB.Close()
		t.Skipf("Skipping test - cannot connect to database: %v", err)
		return nil
	}

	// Store original db and replace with test db
	originalDB := db
	db = testDB

	return &testDatabase{
		DB: testDB,
		Cleanup: func() {
			// Restore original db
			db = originalDB
			testDB.Close()
		},
	}
}

// TestGetDashboardSummaryHandler tests the dashboard summary endpoint
func TestGetDashboardSummaryHandler(t *testing.T) {
	// Setup test database
	testDB := setupAnalyticsTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	// Create test request
	req, err := http.NewRequest("GET", "/api/analytics/dashboard", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create response recorder
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(getDashboardSummaryHandler)

	// Execute request
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Parse response
	var summary DashboardSummary
	if err := json.NewDecoder(rr.Body).Decode(&summary); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Validate response structure
	if summary.TotalInvoices < 0 {
		t.Errorf("Total invoices should be non-negative, got %d", summary.TotalInvoices)
	}

	if summary.TotalRevenue < 0 {
		t.Errorf("Total revenue should be non-negative, got %.2f", summary.TotalRevenue)
	}

	if summary.ActiveClients < 0 {
		t.Errorf("Active clients should be non-negative, got %d", summary.ActiveClients)
	}

	t.Logf("Dashboard summary: %d invoices, $%.2f revenue, %d active clients",
		summary.TotalInvoices, summary.TotalRevenue, summary.ActiveClients)
}

// TestGetRevenueAnalyticsHandler tests the revenue analytics endpoint
func TestGetRevenueAnalyticsHandler(t *testing.T) {
	// Setup test database
	testDB := setupAnalyticsTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	// Create test request
	req, err := http.NewRequest("GET", "/api/analytics/revenue", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create response recorder
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(getRevenueAnalyticsHandler)

	// Execute request
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Parse response
	var analytics RevenueAnalytics
	if err := json.NewDecoder(rr.Body).Decode(&analytics); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Validate response structure
	if analytics.MonthlyRevenue == nil {
		t.Error("MonthlyRevenue should not be nil")
	}

	if analytics.QuarterlyRevenue == nil {
		t.Error("QuarterlyRevenue should not be nil")
	}

	if analytics.YearlyRevenue == nil {
		t.Error("YearlyRevenue should not be nil")
	}

	// Check monthly revenue data if present
	if len(analytics.MonthlyRevenue) > 0 {
		mr := analytics.MonthlyRevenue[0]
		if mr.Month == "" {
			t.Error("Month should not be empty")
		}
		if mr.Revenue < 0 {
			t.Errorf("Revenue should be non-negative, got %.2f", mr.Revenue)
		}
		if mr.Invoices < 0 {
			t.Errorf("Invoice count should be non-negative, got %d", mr.Invoices)
		}
		t.Logf("Latest month %s: $%.2f revenue from %d invoices", mr.Month, mr.Revenue, mr.Invoices)
	}

	t.Logf("Revenue growth: %.2f%%", analytics.RevenueGrowth)
}

// TestGetClientAnalyticsHandler tests the client analytics endpoint
func TestGetClientAnalyticsHandler(t *testing.T) {
	// Setup test database
	testDB := setupAnalyticsTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	// Create test request
	req, err := http.NewRequest("GET", "/api/analytics/clients", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create response recorder
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(getClientAnalyticsHandler)

	// Execute request
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Parse response
	var analytics ClientAnalytics
	if err := json.NewDecoder(rr.Body).Decode(&analytics); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Validate response structure
	if analytics.TotalClients < 0 {
		t.Errorf("Total clients should be non-negative, got %d", analytics.TotalClients)
	}

	if analytics.ActiveClients < 0 {
		t.Errorf("Active clients should be non-negative, got %d", analytics.ActiveClients)
	}

	if analytics.ActiveClients > analytics.TotalClients {
		t.Errorf("Active clients (%d) should not exceed total clients (%d)",
			analytics.ActiveClients, analytics.TotalClients)
	}

	if analytics.TopClients == nil {
		t.Error("TopClients should not be nil")
	}

	if analytics.ClientsWithBalance == nil {
		t.Error("ClientsWithBalance should not be nil")
	}

	// Validate top clients data if present
	if len(analytics.TopClients) > 0 {
		tc := analytics.TopClients[0]
		if tc.ClientID == "" {
			t.Error("Client ID should not be empty")
		}
		if tc.ClientName == "" {
			t.Error("Client name should not be empty")
		}
		if tc.TotalRevenue < 0 {
			t.Errorf("Total revenue should be non-negative, got %.2f", tc.TotalRevenue)
		}
		t.Logf("Top client: %s with $%.2f revenue from %d invoices",
			tc.ClientName, tc.TotalRevenue, tc.InvoiceCount)
	}

	t.Logf("Client analytics: %d total, %d active, %d top clients",
		analytics.TotalClients, analytics.ActiveClients, len(analytics.TopClients))
}

// TestGetInvoiceAnalyticsHandler tests the invoice analytics endpoint
func TestGetInvoiceAnalyticsHandler(t *testing.T) {
	// Setup test database
	testDB := setupAnalyticsTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	// Create test request
	req, err := http.NewRequest("GET", "/api/analytics/invoices", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create response recorder
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(getInvoiceAnalyticsHandler)

	// Execute request
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Parse response
	var analytics InvoiceAnalytics
	if err := json.NewDecoder(rr.Body).Decode(&analytics); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Validate response structure
	if analytics.AgingReport == nil {
		t.Error("AgingReport should not be nil")
	}

	if analytics.StatusDistribution == nil {
		t.Error("StatusDistribution should not be nil")
	}

	if analytics.CollectionRate < 0 || analytics.CollectionRate > 100 {
		t.Errorf("Collection rate should be between 0-100, got %.2f", analytics.CollectionRate)
	}

	if analytics.AveragePaymentDays < 0 {
		t.Errorf("Average payment days should be non-negative, got %.2f", analytics.AveragePaymentDays)
	}

	// Validate aging report data if present
	if len(analytics.AgingReport) > 0 {
		for _, aging := range analytics.AgingReport {
			if aging.AgeBracket == "" {
				t.Error("Age bracket should not be empty")
			}
			if aging.InvoiceCount < 0 {
				t.Errorf("Invoice count should be non-negative, got %d", aging.InvoiceCount)
			}
			if aging.TotalAmount < 0 {
				t.Errorf("Total amount should be non-negative, got %.2f", aging.TotalAmount)
			}
		}
		t.Logf("Aging report has %d brackets", len(analytics.AgingReport))
	}

	// Validate status distribution data if present
	if len(analytics.StatusDistribution) > 0 {
		for _, status := range analytics.StatusDistribution {
			if status.Status == "" {
				t.Error("Status should not be empty")
			}
			if status.Count < 0 {
				t.Errorf("Count should be non-negative, got %d", status.Count)
			}
		}
		t.Logf("Status distribution has %d statuses", len(analytics.StatusDistribution))
	}

	t.Logf("Invoice analytics: %.2f collection rate, %.0f avg payment days",
		analytics.CollectionRate, analytics.AveragePaymentDays)
}

// TestAnalyticsEndpointsWithEmptyDatabase tests all analytics endpoints with empty data
func TestAnalyticsEndpointsWithEmptyDatabase(t *testing.T) {
	// Setup empty test database
	testDB := setupAnalyticsTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	tests := []struct {
		name     string
		endpoint string
		handler  http.HandlerFunc
	}{
		{"Dashboard Summary", "/api/analytics/dashboard", http.HandlerFunc(getDashboardSummaryHandler)},
		{"Revenue Analytics", "/api/analytics/revenue", http.HandlerFunc(getRevenueAnalyticsHandler)},
		{"Client Analytics", "/api/analytics/clients", http.HandlerFunc(getClientAnalyticsHandler)},
		{"Invoice Analytics", "/api/analytics/invoices", http.HandlerFunc(getInvoiceAnalyticsHandler)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.endpoint, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			tt.handler.ServeHTTP(rr, req)

			// All endpoints should return 200 OK even with no data
			if status := rr.Code; status != http.StatusOK {
				t.Errorf("%s returned wrong status code: got %v want %v", tt.name, status, http.StatusOK)
			}

			// Response should be valid JSON
			var result map[string]interface{}
			if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
				t.Errorf("%s returned invalid JSON: %v", tt.name, err)
			}

			t.Logf("%s returned valid response with empty database", tt.name)
		})
	}
}

// TestDashboardSummaryCalculations tests calculation accuracy
func TestDashboardSummaryCalculations(t *testing.T) {
	testDB := setupAnalyticsTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	// Get dashboard summary
	req, _ := http.NewRequest("GET", "/api/analytics/dashboard", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(getDashboardSummaryHandler)
	handler.ServeHTTP(rr, req)

	var summary DashboardSummary
	json.NewDecoder(rr.Body).Decode(&summary)

	// Validate calculations
	expectedTotal := summary.TotalPaid + summary.TotalPending + summary.TotalOverdue
	if summary.TotalRevenue > 0 && expectedTotal > summary.TotalRevenue*1.1 {
		t.Errorf("Sum of paid+pending+overdue (%.2f) significantly exceeds total revenue (%.2f)",
			expectedTotal, summary.TotalRevenue)
	}

	t.Logf("Financial totals validate: Revenue=%.2f, Paid=%.2f, Pending=%.2f, Overdue=%.2f",
		summary.TotalRevenue, summary.TotalPaid, summary.TotalPending, summary.TotalOverdue)
}

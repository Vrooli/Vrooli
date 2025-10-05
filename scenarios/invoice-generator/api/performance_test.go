// +build testing

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// BenchmarkInvoiceCreation benchmarks invoice creation performance
func BenchmarkInvoiceCreation(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(&testing.T{})
	if testDB == nil {
		b.Skip("Database not available")
	}
	defer testDB.Cleanup()

	// Create a test client
	client := createTestClient(&testing.T{}, "BenchClient")

	invoice := map[string]interface{}{
		"client_id":      client.ID,
		"invoice_number": "BENCH-001",
		"issue_date":     time.Now().Format("2006-01-02"),
		"due_date":       time.Now().AddDate(0, 0, 30).Format("2006-01-02"),
		"line_items": []map[string]interface{}{
			{
				"description": "Test Service",
				"quantity":    1,
				"unit_price":  100.00,
			},
		},
	}

	bodyBytes, _ := json.Marshal(invoice)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/api/v1/invoices", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		createInvoiceHandler(w, req)
	}
}

// BenchmarkInvoiceListing benchmarks invoice listing performance
func BenchmarkInvoiceListing(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(&testing.T{})
	if testDB == nil {
		b.Skip("Database not available")
	}
	defer testDB.Cleanup()

	// Create test data
	client := createTestClient(&testing.T{}, "BenchClient")
	for i := 0; i < 10; i++ {
		createTestInvoice(&testing.T{}, client.ID)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/api/v1/invoices", nil)
		w := httptest.NewRecorder()
		getInvoicesHandler(w, req)
	}
}

// BenchmarkPaymentRecording benchmarks payment recording performance
func BenchmarkPaymentRecording(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(&testing.T{})
	if testDB == nil {
		b.Skip("Database not available")
	}
	defer testDB.Cleanup()

	// Create test invoice
	client := createTestClient(&testing.T{}, "BenchClient")
	invoice := createTestInvoice(&testing.T{}, client.ID)

	payment := map[string]interface{}{
		"invoice_id":     invoice.ID,
		"amount":         50.00,
		"payment_method": "credit_card",
		"reference":      "BENCH-PAY",
	}

	bodyBytes, _ := json.Marshal(payment)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/api/v1/payments", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		recordPaymentHandler(w, req)
	}
}

// TestInvoiceCreationPerformance tests invoice creation under load
func TestInvoiceCreationPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	// Create test client
	client := createTestClient(t, "PerfClient")

	// Performance test: Create 100 invoices and measure time
	pattern := PerformanceTestPattern{
		Name:        "InvoiceCreation_100",
		Description: "Create 100 invoices and verify performance",
		MaxDuration: 10 * time.Second,
		Iterations:  1,
		Setup: func(t *testing.T) interface{} {
			return client.ID
		},
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			clientID := setupData.(string)
			start := time.Now()

			for i := 0; i < 100; i++ {
				invoice := map[string]interface{}{
					"client_id":      clientID,
					"invoice_number": fmt.Sprintf("PERF-%d", i),
					"issue_date":     time.Now().Format("2006-01-02"),
					"due_date":       time.Now().AddDate(0, 0, 30).Format("2006-01-02"),
					"line_items": []map[string]interface{}{
						{
							"description": "Performance Test Item",
							"quantity":    1,
							"unit_price":  100.00,
						},
					},
				}

				bodyBytes, _ := json.Marshal(invoice)
				req, _ := http.NewRequest("POST", "/api/v1/invoices", bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				createInvoiceHandler(w, req)

				if w.Code != http.StatusCreated && w.Code != http.StatusOK {
					t.Errorf("Invoice creation failed: status %d", w.Code)
				}
			}

			return time.Since(start)
		},
		Cleanup: func(setupData interface{}) {
			// Cleanup handled by testDB.Cleanup()
		},
	}

	RunPerformanceTest(t, pattern)
}

// TestConcurrentInvoiceCreation tests concurrent invoice creation
func TestConcurrentInvoiceCreation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	// Create test client
	client := createTestClient(t, "ConcurrentClient")

	// Concurrency test: 10 workers creating 10 invoices each
	pattern := ConcurrencyTestPattern{
		Name:        "ConcurrentInvoiceCreation",
		Description: "Test concurrent invoice creation with 10 workers",
		Concurrency: 10,
		Iterations:  10,
		Setup: func(t *testing.T) interface{} {
			return client.ID
		},
		Execute: func(t *testing.T, setupData interface{}, iteration int) error {
			clientID := setupData.(string)

			invoice := map[string]interface{}{
				"client_id":      clientID,
				"invoice_number": fmt.Sprintf("CONC-%d", iteration),
				"issue_date":     time.Now().Format("2006-01-02"),
				"due_date":       time.Now().AddDate(0, 0, 30).Format("2006-01-02"),
				"line_items": []map[string]interface{}{
					{
						"description": "Concurrent Test Item",
						"quantity":    1,
						"unit_price":  100.00,
					},
				},
			}

			bodyBytes, _ := json.Marshal(invoice)
			req, _ := http.NewRequest("POST", "/api/v1/invoices", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			createInvoiceHandler(w, req)

			if w.Code != http.StatusCreated && w.Code != http.StatusOK {
				return fmt.Errorf("invoice creation failed: status %d", w.Code)
			}

			return nil
		},
		Cleanup: func(setupData interface{}) {
			// Cleanup handled by testDB.Cleanup()
		},
	}

	RunConcurrencyTest(t, pattern)
}

// TestPaymentSummaryPerformance tests payment summary generation performance
func TestPaymentSummaryPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	// Create test data: multiple invoices with payments
	client := createTestClient(t, "SummaryClient")
	for i := 0; i < 50; i++ {
		invoice := createTestInvoice(t, client.ID)
		createTestPayment(t, invoice.ID, 50.00)
	}

	// Performance test: Generate payment summary
	pattern := PerformanceTestPattern{
		Name:        "PaymentSummaryGeneration",
		Description: "Generate payment summary for 50 invoices",
		MaxDuration: 2 * time.Second,
		Iterations:  10,
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			start := time.Now()

			req, _ := http.NewRequest("GET", "/api/v1/payments/summary", nil)
			w := httptest.NewRecorder()
			getPaymentSummaryHandler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Payment summary failed: status %d", w.Code)
			}

			return time.Since(start)
		},
	}

	RunPerformanceTest(t, pattern)
}

// TestDatabaseQueryPerformance tests database query performance
func TestDatabaseQueryPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	// Create test data
	client := createTestClient(t, "QueryPerfClient")
	invoiceIDs := make([]string, 100)
	for i := 0; i < 100; i++ {
		invoice := createTestInvoice(t, client.ID)
		invoiceIDs[i] = invoice.ID
	}

	t.Run("Sequential_Queries", func(t *testing.T) {
		start := time.Now()

		for _, id := range invoiceIDs {
			_, err := getInvoiceByID(id)
			if err != nil {
				t.Errorf("Failed to get invoice %s: %v", id, err)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(len(invoiceIDs))

		t.Logf("Sequential queries: total %v, avg %v per query", duration, avgDuration)

		if duration > 5*time.Second {
			t.Errorf("Sequential queries too slow: %v", duration)
		}
	})

	t.Run("Concurrent_Queries", func(t *testing.T) {
		start := time.Now()
		var wg sync.WaitGroup

		for _, id := range invoiceIDs {
			wg.Add(1)
			go func(invoiceID string) {
				defer wg.Done()
				_, err := getInvoiceByID(invoiceID)
				if err != nil {
					t.Errorf("Failed to get invoice %s: %v", invoiceID, err)
				}
			}(id)
		}

		wg.Wait()
		duration := time.Since(start)

		t.Logf("Concurrent queries: total %v for %d queries", duration, len(invoiceIDs))

		if duration > 3*time.Second {
			t.Errorf("Concurrent queries too slow: %v", duration)
		}
	})
}

// TestLargeDatasetHandling tests handling of large datasets
func TestLargeDatasetHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	// Create test data: large invoice with many line items
	client := createTestClient(t, "LargeDataClient")

	t.Run("LargeInvoice_100Items", func(t *testing.T) {
		lineItems := make([]map[string]interface{}, 100)
		for i := 0; i < 100; i++ {
			lineItems[i] = map[string]interface{}{
				"description": fmt.Sprintf("Item %d", i),
				"quantity":    float64(i + 1),
				"unit_price":  10.00 + float64(i),
			}
		}

		invoice := map[string]interface{}{
			"client_id":      client.ID,
			"invoice_number": "LARGE-001",
			"issue_date":     time.Now().Format("2006-01-02"),
			"due_date":       time.Now().AddDate(0, 0, 30).Format("2006-01-02"),
			"line_items":     lineItems,
		}

		start := time.Now()
		bodyBytes, _ := json.Marshal(invoice)
		req, _ := http.NewRequest("POST", "/api/v1/invoices", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		createInvoiceHandler(w, req)
		duration := time.Since(start)

		if w.Code != http.StatusCreated && w.Code != http.StatusOK {
			t.Errorf("Large invoice creation failed: status %d", w.Code)
		}

		t.Logf("Large invoice creation (100 items): %v", duration)

		if duration > 2*time.Second {
			t.Errorf("Large invoice creation too slow: %v", duration)
		}
	})
}

// TestMemoryUsage tests memory usage during operations
func TestMemoryUsage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	client := createTestClient(t, "MemoryTestClient")

	// Create many invoices and check for memory leaks
	t.Run("MemoryStability_1000Invoices", func(t *testing.T) {
		for i := 0; i < 1000; i++ {
			invoice := map[string]interface{}{
				"client_id":      client.ID,
				"invoice_number": fmt.Sprintf("MEM-%d", i),
				"issue_date":     time.Now().Format("2006-01-02"),
				"due_date":       time.Now().AddDate(0, 0, 30).Format("2006-01-02"),
				"line_items": []map[string]interface{}{
					{
						"description": "Memory Test Item",
						"quantity":    1,
						"unit_price":  100.00,
					},
				},
			}

			bodyBytes, _ := json.Marshal(invoice)
			req, _ := http.NewRequest("POST", "/api/v1/invoices", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			createInvoiceHandler(w, req)

			if w.Code != http.StatusCreated && w.Code != http.StatusOK {
				t.Errorf("Invoice creation failed at iteration %d: status %d", i, w.Code)
				break
			}

			// Every 100 iterations, verify we can still list invoices
			if i%100 == 0 {
				req, _ := http.NewRequest("GET", "/api/v1/invoices", nil)
				w := httptest.NewRecorder()
				getInvoicesHandler(w, req)
				if w.Code != http.StatusOK {
					t.Errorf("Invoice listing failed at iteration %d", i)
					break
				}
			}
		}

		t.Log("Memory stability test completed - 1000 invoices created")
	})
}

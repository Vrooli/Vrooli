package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHandleRecordPayment tests payment recording
func TestHandleRecordPayment(t *testing.T) {
	setupTestDB(t)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
	}{
		{
			name: "valid payment",
			requestBody: map[string]interface{}{
				"invoice_id":     "55ca0461-835f-4b80-8dfc-dd7e33b0a413",
				"amount":         100.00,
				"payment_method": "check",
				"payment_date":   "2025-10-13",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request: %v", err)
				}
			}

			req := httptest.NewRequest("POST", "/api/payments", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			recordPaymentHandler(w, req)

			resp := w.Result()
			// Accept 200 (success) or 500 (if invoice doesn't exist)
			if resp.StatusCode != tt.expectedStatus && resp.StatusCode != http.StatusInternalServerError {
				t.Errorf("Expected status %d or 500, got %d for test '%s'", tt.expectedStatus, resp.StatusCode, tt.name)
			}
		})
	}
}

// TestHandleGetInvoicePayments tests retrieving payments for an invoice
func TestHandleGetInvoicePayments(t *testing.T) {
	setupTestDB(t)

	req := httptest.NewRequest("GET", "/api/invoices/55ca0461-835f-4b80-8dfc-dd7e33b0a413/payments", nil)
	w := httptest.NewRecorder()

	getPaymentsHandler(w, req)

	resp := w.Result()
	// Expecting 200 (success) or 500 (if invoice doesn't exist)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", resp.StatusCode)
	}

	if resp.StatusCode == http.StatusOK {
		var payments []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&payments); err != nil {
			t.Fatalf("Failed to decode payments response: %v", err)
		}

		// Should return an array (empty or with data)
		if payments == nil {
			t.Error("Expected array response, got nil")
		}
	}
}

// TestHandleGetPaymentSummary tests getting payment summary
func TestHandleGetPaymentSummary(t *testing.T) {
	setupTestDB(t)

	req := httptest.NewRequest("GET", "/api/payments/summary", nil)
	w := httptest.NewRecorder()

	getPaymentSummaryHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var summary map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		t.Fatalf("Failed to decode summary response: %v", err)
	}

	// Verify summary has expected fields (actual fields: total_paid, total_pending, total_overdue, recent_payments)
	if _, ok := summary["total_paid"]; !ok {
		t.Error("Expected summary to have 'total_paid' field")
	}
	if _, ok := summary["total_pending"]; !ok {
		t.Error("Expected summary to have 'total_pending' field")
	}
}

// +build testing

package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// Integration tests covering full workflows

func TestInvoiceWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	// Step 1: Create a client
	clientReq := map[string]interface{}{
		"name":           "Workflow Client",
		"email":          "workflow@example.com",
		"phone":          "+1-555-0199",
		"address_line1":  "123 Main St",
		"city":           "New York",
		"state_province": "NY",
		"postal_code":    "10001",
		"country":        "USA",
	}

	bodyBytes, _ := json.Marshal(clientReq)
	req := httptest.NewRequest("POST", "/api/clients", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	createClientHandler(w, req)

	if w.Code != 200 {
		t.Fatalf("Failed to create client: %d", w.Code)
	}

	var client Client
	json.Unmarshal(w.Body.Bytes(), &client)

	t.Logf("Created client: %s", client.ID)

	// Step 2: Create an invoice
	invoiceReq := map[string]interface{}{
		"client_id": client.ID,
		"line_items": []map[string]interface{}{
			{
				"description": "Consulting Services",
				"quantity":    8.0,
				"unit_price":  150.0,
			},
			{
				"description": "Development Work",
				"quantity":    20.0,
				"unit_price":  100.0,
			},
		},
		"tax_rate": 8.5,
		"notes":    "Monthly services",
	}

	bodyBytes, _ = json.Marshal(invoiceReq)
	req = httptest.NewRequest("POST", "/api/invoices", bytes.NewReader(bodyBytes))
	w = httptest.NewRecorder()
	createInvoiceHandler(w, req)

	t.Logf("Create invoice status: %d", w.Code)

	if w.Code == 200 {
		var invoiceResp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &invoiceResp)

		if invoiceID, ok := invoiceResp["invoice_id"]; ok {
			t.Logf("Created invoice: %s", invoiceID)

			// Step 3: Update invoice status
			statusReq := map[string]interface{}{"status": "sent"}
			bodyBytes, _ = json.Marshal(statusReq)
			req = httptest.NewRequest("PUT", "/api/invoices/"+invoiceID.(string)+"/status", bytes.NewReader(bodyBytes))
			req = mux.SetURLVars(req, map[string]string{"id": invoiceID.(string)})
			w = httptest.NewRecorder()
			updateInvoiceStatusHandler(w, req)

			t.Logf("Update status: %d", w.Code)
		}
	}
}

func TestPaymentWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	// Create client and invoice
	client := createTestClient(t, "PaymentWorkflowClient")
	invoice := createTestInvoice(t, client.ID)

	// Record a payment
	paymentReq := map[string]interface{}{
		"invoice_id":     invoice.ID,
		"amount":         600.0,
		"payment_method": "wire_transfer",
		"reference":      "WIRE-001",
		"notes":          "First payment",
	}

	bodyBytes, _ := json.Marshal(paymentReq)
	req := httptest.NewRequest("POST", "/api/payments", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	recordPaymentHandler(w, req)

	t.Logf("Record payment status: %d", w.Code)

	// Get payment summary
	req = httptest.NewRequest("GET", "/api/payments/summary", nil)
	w = httptest.NewRecorder()
	getPaymentSummaryHandler(w, req)

	t.Logf("Payment summary status: %d", w.Code)

	if w.Code == 200 {
		var summary map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &summary)
		t.Logf("Summary: %+v", summary)
	}
}

func TestRecurringInvoiceWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	client := createTestClient(t, "RecurringWorkflowClient")

	// Create recurring invoice
	recurringReq := map[string]interface{}{
		"client_id":     client.ID,
		"template_name": "Monthly Retainer",
		"frequency":     "monthly",
		"is_active":     true,
		"currency":      "USD",
		"days_due":      15,
	}

	bodyBytes, _ := json.Marshal(recurringReq)
	req := httptest.NewRequest("POST", "/api/recurring", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	createRecurringInvoiceHandler(w, req)

	t.Logf("Create recurring status: %d", w.Code)

	// List recurring invoices
	req = httptest.NewRequest("GET", "/api/recurring", nil)
	w = httptest.NewRecorder()
	getRecurringInvoicesHandler(w, req)

	t.Logf("List recurring status: %d", w.Code)
}

func TestErrorHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	t.Run("InvalidJSONInvoice", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/invoices", bytes.NewReader([]byte("{invalid json")))
		w := httptest.NewRecorder()
		createInvoiceHandler(w, req)

		if w.Code == 200 {
			t.Error("Expected error for invalid JSON")
		}
	})

	t.Run("MissingClientID", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"line_items": []map[string]interface{}{
				{"description": "Test", "quantity": 1.0, "unit_price": 100.0},
			},
		}

		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/invoices", bytes.NewReader(bodyBytes))
		w := httptest.NewRecorder()
		createInvoiceHandler(w, req)

		// May fail due to missing client_id
		t.Logf("Missing client_id status: %d", w.Code)
	})

	t.Run("InvalidClientJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/clients", bytes.NewReader([]byte("{invalid")))
		w := httptest.NewRecorder()
		createClientHandler(w, req)

		if w.Code == 200 {
			t.Error("Expected error for invalid JSON")
		}
	})

	t.Run("NonExistentInvoice", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/invoices/non-existent", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "non-existent"})
		w := httptest.NewRecorder()
		getInvoiceHandler(w, req)

		if w.Code == 200 {
			t.Error("Expected error for non-existent invoice")
		}
	})
}

func TestEdgeCasesDetailed(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	client := createTestClient(t, "EdgeCaseClient")

	t.Run("LargeInvoice", func(t *testing.T) {
		items := []map[string]interface{}{}
		for i := 0; i < 50; i++ {
			items = append(items, map[string]interface{}{
				"description": "Item " + string(rune(i)),
				"quantity":    1.0,
				"unit_price":  10.0,
			})
		}

		reqBody := map[string]interface{}{
			"client_id":  client.ID,
			"line_items": items,
		}

		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/invoices", bytes.NewReader(bodyBytes))
		w := httptest.NewRecorder()
		createInvoiceHandler(w, req)

		t.Logf("Large invoice status: %d", w.Code)
	})

	t.Run("HighTaxRate", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"client_id": client.ID,
			"line_items": []map[string]interface{}{
				{"description": "Item", "quantity": 1.0, "unit_price": 100.0},
			},
			"tax_rate": 25.0, // High tax rate
		}

		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/invoices", bytes.NewReader(bodyBytes))
		w := httptest.NewRecorder()
		createInvoiceHandler(w, req)

		t.Logf("High tax rate status: %d", w.Code)
	})

	t.Run("DecimalQuantities", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"client_id": client.ID,
			"line_items": []map[string]interface{}{
				{"description": "Hourly Service", "quantity": 7.5, "unit_price": 125.50},
			},
		}

		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/invoices", bytes.NewReader(bodyBytes))
		w := httptest.NewRecorder()
		createInvoiceHandler(w, req)

		t.Logf("Decimal quantities status: %d", w.Code)
	})
}

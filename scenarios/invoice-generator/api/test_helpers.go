// +build testing

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes a test-specific logger
func setupTestLogger() func() {
	originalFlags := log.Flags()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	return func() {
		log.SetFlags(originalFlags)
	}
}

// TestDatabase manages test database connections
type TestDatabase struct {
	DB      *sql.DB
	Cleanup func()
}

// setupTestDB creates an isolated test database
func setupTestDB(t *testing.T) *TestDatabase {
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
			dbPassword = "postgres"
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

	// Initialize schema
	if err := initializeDatabase(); err != nil {
		testDB.Close()
		db = originalDB
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	return &TestDatabase{
		DB: testDB,
		Cleanup: func() {
			// Clean up test data
			testDB.Exec("DELETE FROM payments")
			testDB.Exec("DELETE FROM refunds")
			testDB.Exec("DELETE FROM invoice_items")
			testDB.Exec("DELETE FROM invoices")
			testDB.Exec("DELETE FROM recurring_invoice_items")
			testDB.Exec("DELETE FROM recurring_invoices")
			testDB.Exec("DELETE FROM clients WHERE id NOT IN ('11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222')")

			// Restore original db
			db = originalDB
			testDB.Close()
		},
	}
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	URLVars     map[string]string
	QueryParams map[string]string
	Headers     map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %v", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, err
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Add query parameters
	if len(req.QueryParams) > 0 {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Add(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	// Add URL variables (for mux)
	if len(req.URLVars) > 0 {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	w := httptest.NewRecorder()
	return w, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields map[string]interface{}) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	if expectedFields == nil {
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v. Body: %s", err, w.Body.String())
		return
	}

	for key, expectedValue := range expectedFields {
		if actualValue, exists := response[key]; !exists {
			t.Errorf("Expected field '%s' not found in response", key)
		} else if expectedValue != nil && expectedValue != "*" && actualValue != expectedValue {
			t.Errorf("Field '%s': expected %v, got %v", key, expectedValue, actualValue)
		}
	}
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, shouldContainError bool) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	if shouldContainError && w.Body.Len() == 0 {
		t.Error("Expected error message in response body, got empty body")
	}
}

// createTestInvoice creates a test invoice in the database
func createTestInvoice(t *testing.T, clientID string) *Invoice {
	invoice := &Invoice{
		ID:            uuid.New().String(),
		CompanyID:     "00000000-0000-0000-0000-000000000001",
		ClientID:      clientID,
		InvoiceNumber: fmt.Sprintf("TEST-%d", time.Now().Unix()),
		Status:        "draft",
		IssueDate:     time.Now().Format("2006-01-02"),
		DueDate:       time.Now().AddDate(0, 0, 30).Format("2006-01-02"),
		Currency:      "USD",
		Subtotal:      1000.00,
		TaxAmount:     100.00,
		TotalAmount:   1100.00,
		BalanceDue:    1100.00,
	}

	query := `
		INSERT INTO invoices (id, company_id, client_id, invoice_number, status,
			issue_date, due_date, currency, subtotal, tax_amount, total_amount, balance_due)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at, updated_at`

	err := db.QueryRow(query, invoice.ID, invoice.CompanyID, invoice.ClientID,
		invoice.InvoiceNumber, invoice.Status, invoice.IssueDate, invoice.DueDate,
		invoice.Currency, invoice.Subtotal, invoice.TaxAmount, invoice.TotalAmount,
		invoice.BalanceDue).Scan(&invoice.CreatedAt, &invoice.UpdatedAt)

	if err != nil {
		t.Fatalf("Failed to create test invoice: %v", err)
	}

	return invoice
}

// createTestClient creates a test client in the database
func createTestClient(t *testing.T, name string) *Client {
	client := &Client{
		ID:       uuid.New().String(),
		Name:     name,
		Email:    fmt.Sprintf("%s@example.com", name),
		IsActive: true,
	}

	query := `
		INSERT INTO clients (id, name, email, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at`

	err := db.QueryRow(query, client.ID, client.Name, client.Email, client.IsActive).
		Scan(&client.CreatedAt)

	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	return client
}

// createTestPayment creates a test payment in the database
func createTestPayment(t *testing.T, invoiceID string, amount float64) *Payment {
	payment := &Payment{
		ID:            uuid.New().String(),
		InvoiceID:     invoiceID,
		Amount:        amount,
		PaymentDate:   time.Now().Format("2006-01-02"),
		PaymentMethod: "credit_card",
		Reference:     fmt.Sprintf("REF-%d", time.Now().Unix()),
	}

	query := `
		INSERT INTO payments (id, invoice_id, amount, payment_date, payment_method, reference)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at`

	err := db.QueryRow(query, payment.ID, payment.InvoiceID, payment.Amount,
		payment.PaymentDate, payment.PaymentMethod, payment.Reference).
		Scan(&payment.CreatedAt)

	if err != nil {
		t.Fatalf("Failed to create test payment: %v", err)
	}

	return payment
}

// createTestRecurringInvoice creates a test recurring invoice
func createTestRecurringInvoice(t *testing.T, clientID string) string {
	recurringID := uuid.New().String()

	query := `
		INSERT INTO recurring_invoices (id, client_id, template_name, frequency,
			next_date, is_active, currency)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := db.Exec(query, recurringID, clientID, "Test Template", "monthly",
		time.Now().AddDate(0, 1, 0).Format("2006-01-02"), true, "USD")

	if err != nil {
		t.Fatalf("Failed to create test recurring invoice: %v", err)
	}

	return recurringID
}

// waitForAsync waits for async operations to complete (for testing background tasks)
func waitForAsync(duration time.Duration) {
	time.Sleep(duration)
}

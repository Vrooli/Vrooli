package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// TestHandleGetExchangeRates tests retrieving exchange rates
func TestHandleGetExchangeRates(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	// Insert test rates
	insertTestRates(t, "USD", []currencyRate{
		{target: "EUR", rate: 0.85},
		{target: "GBP", rate: 0.75},
		{target: "JPY", rate: 110.0},
	})

	req := httptest.NewRequest("GET", "/api/currencies/rates?base=USD", nil)
	w := httptest.NewRecorder()

	handleGetExchangeRates(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["base_currency"] != "USD" {
		t.Errorf("Expected base_currency USD, got %v", response["base_currency"])
	}

	count := int(response["count"].(float64))
	if count < 3 {
		t.Errorf("Expected at least 3 rates, got %d", count)
	}
}

// TestHandleGetExchangeRatesDefaultBase tests default base currency
func TestHandleGetExchangeRatesDefaultBase(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	insertTestRates(t, "USD", []currencyRate{
		{target: "EUR", rate: 0.85},
	})

	req := httptest.NewRequest("GET", "/api/currencies/rates", nil)
	w := httptest.NewRecorder()

	handleGetExchangeRates(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["base_currency"] != "USD" {
		t.Errorf("Expected default base_currency USD, got %v", response["base_currency"])
	}
}

// TestHandleConvertCurrencySameCurrency tests same currency conversion
func TestHandleConvertCurrencySameCurrency(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	reqBody := CurrencyConversionRequest{
		Amount:       100.0,
		FromCurrency: "USD",
		ToCurrency:   "USD",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/currencies/convert", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleConvertCurrency(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response CurrencyConversionResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.ConvertedAmount != 100.0 {
		t.Errorf("Expected converted amount 100.0, got %f", response.ConvertedAmount)
	}

	if response.ExchangeRate != 1.0 {
		t.Errorf("Expected exchange rate 1.0, got %f", response.ExchangeRate)
	}
}

// TestHandleConvertCurrencyDifferentCurrencies tests conversion between currencies
func TestHandleConvertCurrencyDifferentCurrencies(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	reqBody := CurrencyConversionRequest{
		Amount:       100.0,
		FromCurrency: "USD",
		ToCurrency:   "EUR",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/currencies/convert", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleConvertCurrency(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response CurrencyConversionResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.FromCurrency != "USD" {
		t.Errorf("Expected from_currency USD, got %s", response.FromCurrency)
	}

	if response.ToCurrency != "EUR" {
		t.Errorf("Expected to_currency EUR, got %s", response.ToCurrency)
	}

	// Verify conversion math is correct (amount * rate = converted_amount)
	expectedConverted := response.Amount * response.ExchangeRate
	if response.ConvertedAmount < expectedConverted-0.01 || response.ConvertedAmount > expectedConverted+0.01 {
		t.Errorf("Conversion math incorrect: %.2f * %.6f should be %.2f, got %.2f",
			response.Amount, response.ExchangeRate, expectedConverted, response.ConvertedAmount)
	}

	// Verify exchange rate is reasonable (between 0.5 and 1.5)
	if response.ExchangeRate < 0.5 || response.ExchangeRate > 1.5 {
		t.Errorf("Exchange rate seems unreasonable: %f", response.ExchangeRate)
	}
}

// TestHandleConvertCurrencyInvalidAmount tests validation
func TestHandleConvertCurrencyInvalidAmount(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	tests := []struct {
		name     string
		reqBody  CurrencyConversionRequest
		wantCode int
	}{
		{
			name: "zero amount",
			reqBody: CurrencyConversionRequest{
				Amount:       0,
				FromCurrency: "USD",
				ToCurrency:   "EUR",
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "negative amount",
			reqBody: CurrencyConversionRequest{
				Amount:       -100,
				FromCurrency: "USD",
				ToCurrency:   "EUR",
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "missing from_currency",
			reqBody: CurrencyConversionRequest{
				Amount:     100,
				ToCurrency: "EUR",
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "missing to_currency",
			reqBody: CurrencyConversionRequest{
				Amount:       100,
				FromCurrency: "USD",
			},
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest("POST", "/api/currencies/convert", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleConvertCurrency(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("Expected status %d, got %d", tt.wantCode, w.Code)
			}
		})
	}
}

// TestHandleConvertCurrencyNotFound tests missing exchange rate
func TestHandleConvertCurrencyNotFound(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	reqBody := CurrencyConversionRequest{
		Amount:       100.0,
		FromCurrency: "USD",
		ToCurrency:   "XXX", // Non-existent currency
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/currencies/convert", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleConvertCurrency(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// TestHandleUpdateExchangeRate tests manual rate updates
func TestHandleUpdateExchangeRate(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	reqBody := map[string]interface{}{
		"base_currency":   "USD",
		"target_currency": "EUR",
		"rate":            0.90,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/currencies/rates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleUpdateExchangeRate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["success"] != true {
		t.Error("Expected success to be true")
	}

	if response["base_currency"] != "USD" {
		t.Errorf("Expected base_currency USD, got %v", response["base_currency"])
	}

	if response["target_currency"] != "EUR" {
		t.Errorf("Expected target_currency EUR, got %v", response["target_currency"])
	}

	if response["rate"] != 0.90 {
		t.Errorf("Expected rate 0.90, got %v", response["rate"])
	}

	if response["source"] != "manual" {
		t.Errorf("Expected source 'manual', got %v", response["source"])
	}
}

// TestHandleUpdateExchangeRateValidation tests validation
func TestHandleUpdateExchangeRateValidation(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	tests := []struct {
		name     string
		reqBody  map[string]interface{}
		wantCode int
	}{
		{
			name: "missing base_currency",
			reqBody: map[string]interface{}{
				"target_currency": "EUR",
				"rate":            0.90,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "missing target_currency",
			reqBody: map[string]interface{}{
				"base_currency": "USD",
				"rate":          0.90,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "zero rate",
			reqBody: map[string]interface{}{
				"base_currency":   "USD",
				"target_currency": "EUR",
				"rate":            0,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "negative rate",
			reqBody: map[string]interface{}{
				"base_currency":   "USD",
				"target_currency": "EUR",
				"rate":            -0.90,
			},
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest("POST", "/api/currencies/rates", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleUpdateExchangeRate(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("Expected status %d, got %d", tt.wantCode, w.Code)
			}
		})
	}
}

// TestHandleGetSupportedCurrencies tests listing currencies
func TestHandleGetSupportedCurrencies(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	// Insert test rates for multiple currencies
	insertTestRates(t, "USD", []currencyRate{
		{target: "EUR", rate: 0.85},
		{target: "GBP", rate: 0.75},
		{target: "JPY", rate: 110.0},
	})

	req := httptest.NewRequest("GET", "/api/currencies", nil)
	w := httptest.NewRecorder()

	handleGetSupportedCurrencies(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	count := int(response["count"].(float64))
	if count < 3 {
		t.Errorf("Expected at least 3 currencies, got %d", count)
	}

	currencies := response["currencies"].([]interface{})
	if len(currencies) < 3 {
		t.Errorf("Expected at least 3 currency codes, got %d", len(currencies))
	}
}

// TestHandleConvertCurrencyWithHistoricalDate tests historical conversion
func TestHandleConvertCurrencyWithHistoricalDate(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	// Insert historical rate
	insertTestRatesWithDate(t, "USD", []currencyRate{
		{target: "EUR", rate: 0.80},
	}, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))

	dateStr := "2024-01-01"
	reqBody := CurrencyConversionRequest{
		Amount:       100.0,
		FromCurrency: "USD",
		ToCurrency:   "EUR",
		Date:         &dateStr,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/currencies/convert", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleConvertCurrency(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response CurrencyConversionResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should use historical rate (0.80)
	expected := 80.0
	if response.ConvertedAmount < expected-0.01 || response.ConvertedAmount > expected+0.01 {
		t.Errorf("Expected converted amount ~%.2f, got %.2f", expected, response.ConvertedAmount)
	}
}

// Helper types and functions

type currencyRate struct {
	target string
	rate   float64
}

func insertTestRates(t *testing.T, base string, rates []currencyRate) {
	insertTestRatesWithDate(t, base, rates, time.Now().UTC())
}

func insertTestRatesWithDate(t *testing.T, base string, rates []currencyRate, validFrom time.Time) {
	for _, r := range rates {
		query := `
			INSERT INTO exchange_rates (base_currency, target_currency, rate, source, valid_from)
			VALUES ($1, $2, $3, 'test', $4)
			ON CONFLICT (base_currency, target_currency, valid_from) DO NOTHING
		`
		_, err := db.Exec(query, base, r.target, r.rate, validFrom)
		if err != nil {
			t.Fatalf("Failed to insert test rate %s->%s: %v", base, r.target, err)
		}
	}
}

func setupTestDB(t *testing.T) {
	if db == nil {
		// Build connection string from environment variables
		postgresURL := os.Getenv("POSTGRES_URL")
		if postgresURL == "" {
			dbHost := os.Getenv("POSTGRES_HOST")
			dbPort := os.Getenv("POSTGRES_PORT")
			dbUser := os.Getenv("POSTGRES_USER")
			dbPassword := os.Getenv("POSTGRES_PASSWORD")
			dbName := os.Getenv("POSTGRES_DB")

			if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
				t.Skip("Skipping test: database configuration not available")
			}

			postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
				dbUser, dbPassword, dbHost, dbPort, dbName)
		}

		var err error
		db, err = sql.Open("postgres", postgresURL)
		if err != nil {
			t.Fatalf("Failed to connect to test database: %v", err)
		}
	}

	// Clean existing test rates
	_, err := db.Exec("DELETE FROM exchange_rates WHERE source = 'test'")
	if err != nil {
		t.Fatalf("Failed to clean test data: %v", err)
	}
}

func teardownTestDB(t *testing.T) {
	// Clean up test data
	if db != nil {
		db.Exec("DELETE FROM exchange_rates WHERE source = 'test'")
	}
}

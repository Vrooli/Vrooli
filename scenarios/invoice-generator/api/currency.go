package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// ExchangeRate represents a currency exchange rate
type ExchangeRate struct {
	ID             string    `json:"id"`
	BaseCurrency   string    `json:"base_currency"`
	TargetCurrency string    `json:"target_currency"`
	Rate           float64   `json:"rate"`
	Source         string    `json:"source"`
	ValidFrom      time.Time `json:"valid_from"`
	ValidUntil     *time.Time `json:"valid_until,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CurrencyConversionRequest for converting amounts
type CurrencyConversionRequest struct {
	Amount         float64 `json:"amount"`
	FromCurrency   string  `json:"from_currency"`
	ToCurrency     string  `json:"to_currency"`
	Date           *string `json:"date,omitempty"` // Optional historical date
}

// CurrencyConversionResponse returns converted amount
type CurrencyConversionResponse struct {
	Amount         float64   `json:"amount"`
	FromCurrency   string    `json:"from_currency"`
	ToCurrency     string    `json:"to_currency"`
	ConvertedAmount float64  `json:"converted_amount"`
	ExchangeRate   float64   `json:"exchange_rate"`
	ConversionDate time.Time `json:"conversion_date"`
}

// ExchangeRateAPIResponse from external API (exchangerate-api.com free tier)
type ExchangeRateAPIResponse struct {
	Result            string             `json:"result"`
	BaseCode          string             `json:"base_code"`
	Rates             map[string]float64 `json:"rates"`
	TimeLastUpdateUtc string             `json:"time_last_update_utc"`
}

// GetExchangeRates retrieves current exchange rates
func handleGetExchangeRates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	baseCurrency := r.URL.Query().Get("base")
	if baseCurrency == "" {
		baseCurrency = "USD" // Default to USD
	}

	query := `
		SELECT id, base_currency, target_currency, rate, source, valid_from, valid_until, created_at, updated_at
		FROM exchange_rates
		WHERE base_currency = $1
		  AND valid_from <= CURRENT_TIMESTAMP
		  AND (valid_until IS NULL OR valid_until > CURRENT_TIMESTAMP)
		ORDER BY target_currency
	`

	rows, err := db.Query(query, baseCurrency)
	if err != nil {
		log.Printf("Error querying exchange rates: %v", err)
		http.Error(w, `{"error":"Failed to retrieve exchange rates"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	rates := []ExchangeRate{}
	for rows.Next() {
		var rate ExchangeRate
		err := rows.Scan(
			&rate.ID,
			&rate.BaseCurrency,
			&rate.TargetCurrency,
			&rate.Rate,
			&rate.Source,
			&rate.ValidFrom,
			&rate.ValidUntil,
			&rate.CreatedAt,
			&rate.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning exchange rate: %v", err)
			continue
		}
		rates = append(rates, rate)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"base_currency": baseCurrency,
		"rates":         rates,
		"count":         len(rates),
	})
}

// ConvertCurrency converts an amount between currencies
func handleConvertCurrency(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req CurrencyConversionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Amount <= 0 {
		http.Error(w, `{"error":"Amount must be greater than 0"}`, http.StatusBadRequest)
		return
	}
	if req.FromCurrency == "" || req.ToCurrency == "" {
		http.Error(w, `{"error":"Both from_currency and to_currency are required"}`, http.StatusBadRequest)
		return
	}

	// Same currency = no conversion
	if req.FromCurrency == req.ToCurrency {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CurrencyConversionResponse{
			Amount:          req.Amount,
			FromCurrency:    req.FromCurrency,
			ToCurrency:      req.ToCurrency,
			ConvertedAmount: req.Amount,
			ExchangeRate:    1.0,
			ConversionDate:  time.Now(),
		})
		return
	}

	// Determine conversion date (convert to UTC for database compatibility)
	conversionDate := time.Now().UTC()
	if req.Date != nil {
		parsed, err := time.Parse("2006-01-02", *req.Date)
		if err == nil {
			conversionDate = parsed.UTC()
		}
	}

	// Get exchange rate from database
	// Format time as string for TIMESTAMP WITHOUT TIME ZONE compatibility
	timestampStr := conversionDate.Format("2006-01-02 15:04:05")
	var exchangeRate sql.NullFloat64
	query := `SELECT get_exchange_rate($1::VARCHAR(3), $2::VARCHAR(3), $3::TIMESTAMP)`
	log.Printf("Converting %s to %s on timestamp %s (UTC)", req.FromCurrency, req.ToCurrency, timestampStr)
	err := db.QueryRow(query, req.FromCurrency, req.ToCurrency, timestampStr).Scan(&exchangeRate)
	if err != nil || !exchangeRate.Valid {
		if err != nil {
			log.Printf("Error getting exchange rate: %v", err)
		} else {
			log.Printf("Exchange rate NULL for %s to %s", req.FromCurrency, req.ToCurrency)
		}
		http.Error(w, fmt.Sprintf(`{"error":"Exchange rate not found for %s to %s"}`, req.FromCurrency, req.ToCurrency), http.StatusNotFound)
		return
	}
	log.Printf("Found exchange rate: %.6f", exchangeRate.Float64)

	// Perform conversion
	convertedAmount := req.Amount * exchangeRate.Float64

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CurrencyConversionResponse{
		Amount:          req.Amount,
		FromCurrency:    req.FromCurrency,
		ToCurrency:      req.ToCurrency,
		ConvertedAmount: convertedAmount,
		ExchangeRate:    exchangeRate.Float64,
		ConversionDate:  conversionDate,
	})
}

// RefreshExchangeRates fetches latest rates from external API
func handleRefreshExchangeRates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	baseCurrency := r.URL.Query().Get("base")
	if baseCurrency == "" {
		baseCurrency = "USD"
	}

	// Fetch from exchangerate-api.com (free tier, no API key for latest rates)
	// For production, consider paid tiers or other providers with API keys
	apiURL := fmt.Sprintf("https://open.er-api.com/v6/latest/%s", baseCurrency)

	// Allow custom API URL from environment
	if customURL := os.Getenv("EXCHANGE_RATE_API_URL"); customURL != "" {
		apiURL = fmt.Sprintf("%s/%s", customURL, baseCurrency)
	}

	resp, err := http.Get(apiURL)
	if err != nil {
		log.Printf("Error fetching exchange rates from API: %v", err)
		http.Error(w, `{"error":"Failed to fetch exchange rates from external API"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Exchange rate API returned status %d", resp.StatusCode)
		http.Error(w, `{"error":"Exchange rate API unavailable"}`, http.StatusBadGateway)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading API response: %v", err)
		http.Error(w, `{"error":"Failed to read API response"}`, http.StatusInternalServerError)
		return
	}

	var apiResponse ExchangeRateAPIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		log.Printf("Error parsing API response: %v", err)
		http.Error(w, `{"error":"Failed to parse API response"}`, http.StatusInternalServerError)
		return
	}

	if apiResponse.Result != "success" {
		log.Printf("API returned unsuccessful result: %s", apiResponse.Result)
		http.Error(w, `{"error":"API request was not successful"}`, http.StatusBadGateway)
		return
	}

	// Insert/update rates in database
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		http.Error(w, `{"error":"Database error"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	insertQuery := `
		INSERT INTO exchange_rates (base_currency, target_currency, rate, source, valid_from)
		VALUES ($1, $2, $3, 'api', CURRENT_TIMESTAMP)
		ON CONFLICT (base_currency, target_currency, valid_from)
		DO UPDATE SET rate = EXCLUDED.rate, source = 'api', updated_at = CURRENT_TIMESTAMP
	`

	updatedCount := 0
	for targetCurrency, rate := range apiResponse.Rates {
		if targetCurrency == baseCurrency {
			continue // Skip same currency
		}

		_, err := tx.Exec(insertQuery, baseCurrency, targetCurrency, rate)
		if err != nil {
			log.Printf("Error inserting rate for %s: %v", targetCurrency, err)
			continue
		}
		updatedCount++
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, `{"error":"Failed to save exchange rates"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Refreshed %d exchange rates for base currency %s", updatedCount, baseCurrency)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"base_currency": baseCurrency,
		"updated_count": updatedCount,
		"source":        "api",
		"timestamp":     time.Now(),
	})
}

// UpdateExchangeRate manually updates a specific exchange rate
func handleUpdateExchangeRate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		BaseCurrency   string  `json:"base_currency"`
		TargetCurrency string  `json:"target_currency"`
		Rate           float64 `json:"rate"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.BaseCurrency == "" || req.TargetCurrency == "" || req.Rate <= 0 {
		http.Error(w, `{"error":"base_currency, target_currency, and rate (>0) are required"}`, http.StatusBadRequest)
		return
	}

	query := `
		INSERT INTO exchange_rates (base_currency, target_currency, rate, source, valid_from)
		VALUES ($1, $2, $3, 'manual', CURRENT_TIMESTAMP)
		ON CONFLICT (base_currency, target_currency, valid_from)
		DO UPDATE SET rate = EXCLUDED.rate, source = 'manual', updated_at = CURRENT_TIMESTAMP
		RETURNING id
	`

	var rateID string
	err := db.QueryRow(query, req.BaseCurrency, req.TargetCurrency, req.Rate).Scan(&rateID)
	if err != nil {
		log.Printf("Error updating exchange rate: %v", err)
		http.Error(w, `{"error":"Failed to update exchange rate"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Updated exchange rate: %s -> %s = %.6f", req.BaseCurrency, req.TargetCurrency, req.Rate)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        true,
		"rate_id":        rateID,
		"base_currency":  req.BaseCurrency,
		"target_currency": req.TargetCurrency,
		"rate":           req.Rate,
		"source":         "manual",
	})
}

// GetSupportedCurrencies returns list of currencies with active rates
func handleGetSupportedCurrencies(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	query := `
		SELECT DISTINCT target_currency
		FROM exchange_rates
		WHERE valid_from <= CURRENT_TIMESTAMP
		  AND (valid_until IS NULL OR valid_until > CURRENT_TIMESTAMP)
		ORDER BY target_currency
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error querying currencies: %v", err)
		http.Error(w, `{"error":"Failed to retrieve currencies"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	currencies := []string{}
	for rows.Next() {
		var currency string
		if err := rows.Scan(&currency); err != nil {
			continue
		}
		currencies = append(currencies, currency)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"currencies": currencies,
		"count":      len(currencies),
	})
}

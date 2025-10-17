package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"financial-calculators-hub/lib"
)

// Net Worth Handler
func handleNetWorth(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		handleNetWorthCalculation(w, r)
	case http.MethodGet:
		handleNetWorthHistory(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleNetWorthCalculation(w http.ResponseWriter, r *http.Request) {
	var input lib.NetWorthInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid input: " + err.Error()})
		return
	}

	result := lib.CalculateNetWorth(input)

	// Save to database if available
	if db != nil {
		userID := r.Header.Get("X-User-ID")
		notes := r.Header.Get("X-Notes")
		if id, err := saveCalculation("net-worth", input, result, &userID, &notes); err == nil {
			result.CalculationID = id
		}

		// Also save to net_worth_entries table
		if userID != "" {
			saveNetWorthEntry(userID, input, result.NetWorth)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleNetWorthHistory(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Database not available"})
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = r.Header.Get("X-User-ID")
	}

	if userID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "user_id required"})
		return
	}

	history, err := getNetWorthHistory(userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"history": history,
		"user_id": userID,
	})
}

// Tax Optimizer Handler
func handleTaxOptimizer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input lib.TaxOptimizerInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid input: " + err.Error()})
		return
	}

	if input.Income <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Income must be positive"})
		return
	}

	result := lib.CalculateTaxOptimization(input)

	// Save to database if available
	if db != nil {
		userID := r.Header.Get("X-User-ID")
		notes := r.Header.Get("X-Notes")
		if id, err := saveCalculation("tax-optimizer", input, result, &userID, &notes); err == nil {
			result.CalculationID = id
		}

		// Also save to tax_calculations table
		if userID != "" {
			saveTaxCalculation(userID, input, result)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// History Handler
func handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if db == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Database not available"})
		return
	}

	calcType := r.URL.Query().Get("type")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	userID := r.Header.Get("X-User-ID")

	history, err := getCalculationHistory(calcType, limit, &userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"history": history,
		"count":   len(history),
	})
}

// Batch Calculations Handler
func handleBatchCalculations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input lib.BatchCalculationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid input: " + err.Error()})
		return
	}

	if len(input.Calculations) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "At least one calculation required"})
		return
	}

	results := make([]map[string]interface{}, 0, len(input.Calculations))

	for _, calc := range input.Calculations {
		result := processBatchCalculation(calc)
		results = append(results, result)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results":   results,
		"count":     len(results),
		"timestamp": time.Now().Unix(),
	})
}

func processBatchCalculation(calc map[string]interface{}) map[string]interface{} {
	calcType, ok := calc["type"].(string)
	if !ok {
		return map[string]interface{}{
			"error": "type field required",
		}
	}

	inputData, ok := calc["input"]
	if !ok {
		return map[string]interface{}{
			"error": "input field required",
			"type":  calcType,
		}
	}

	// Convert input to JSON and back to appropriate struct
	inputJSON, err := json.Marshal(inputData)
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
			"type":  calcType,
		}
	}

	switch calcType {
	case "fire":
		var input lib.FIREInput
		if err := json.Unmarshal(inputJSON, &input); err != nil {
			return map[string]interface{}{"error": err.Error(), "type": calcType}
		}
		return map[string]interface{}{
			"type":   calcType,
			"result": lib.CalculateFIRE(input),
		}

	case "compound-interest":
		var input lib.CompoundInterestInput
		if err := json.Unmarshal(inputJSON, &input); err != nil {
			return map[string]interface{}{"error": err.Error(), "type": calcType}
		}
		return map[string]interface{}{
			"type":   calcType,
			"result": lib.CalculateCompoundInterest(input),
		}

	case "mortgage":
		var input lib.MortgageInput
		if err := json.Unmarshal(inputJSON, &input); err != nil {
			return map[string]interface{}{"error": err.Error(), "type": calcType}
		}
		return map[string]interface{}{
			"type":   calcType,
			"result": lib.CalculateMortgage(input),
		}

	case "inflation":
		var input lib.InflationInput
		if err := json.Unmarshal(inputJSON, &input); err != nil {
			return map[string]interface{}{"error": err.Error(), "type": calcType}
		}
		return map[string]interface{}{
			"type":   calcType,
			"result": lib.CalculateInflation(input),
		}

	case "net-worth":
		var input lib.NetWorthInput
		if err := json.Unmarshal(inputJSON, &input); err != nil {
			return map[string]interface{}{"error": err.Error(), "type": calcType}
		}
		return map[string]interface{}{
			"type":   calcType,
			"result": lib.CalculateNetWorth(input),
		}

	case "tax":
		var input lib.TaxOptimizerInput
		if err := json.Unmarshal(inputJSON, &input); err != nil {
			return map[string]interface{}{"error": err.Error(), "type": calcType}
		}
		return map[string]interface{}{
			"type":   calcType,
			"result": lib.CalculateTaxOptimization(input),
		}

	default:
		return map[string]interface{}{
			"error": "Unknown calculation type: " + calcType,
			"type":  calcType,
		}
	}
}

// Database helper functions
func saveNetWorthEntry(userID string, input lib.NetWorthInput, netWorth float64) error {
	if db == nil {
		return nil
	}

	// Calculate totals for assets and liabilities
	assetsTotal := 0.0
	for _, v := range input.Assets {
		assetsTotal += v
	}

	liabilitiesTotal := 0.0
	for _, v := range input.Liabilities {
		liabilitiesTotal += v
	}

	// Create JSONB with individual items and total
	assetsWithTotal := map[string]interface{}{
		"total": assetsTotal,
	}
	for k, v := range input.Assets {
		assetsWithTotal[k] = v
	}

	liabilitiesWithTotal := map[string]interface{}{
		"total": liabilitiesTotal,
	}
	for k, v := range input.Liabilities {
		liabilitiesWithTotal[k] = v
	}

	assetsJSON, _ := json.Marshal(assetsWithTotal)
	liabilitiesJSON, _ := json.Marshal(liabilitiesWithTotal)

	// Note: net_worth is a GENERATED ALWAYS column calculated from assets.total - liabilities.total
	query := `
		INSERT INTO net_worth_entries (user_id, entry_date, assets, liabilities, notes)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, entry_date)
		DO UPDATE SET assets = $3, liabilities = $4, notes = $5, updated_at = CURRENT_TIMESTAMP
	`

	_, err := db.Exec(query, userID, time.Now(), assetsJSON, liabilitiesJSON, input.Notes)
	return err
}

func getNetWorthHistory(userID string) ([]map[string]interface{}, error) {
	if db == nil {
		return nil, nil
	}

	query := `
		SELECT entry_date, assets, liabilities, net_worth, notes, created_at
		FROM net_worth_entries
		WHERE user_id = $1
		ORDER BY entry_date DESC
		LIMIT 100
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var entryDate time.Time
		var assets, liabilities json.RawMessage
		var netWorth float64
		var notes string
		var createdAt time.Time

		err := rows.Scan(&entryDate, &assets, &liabilities, &netWorth, &notes, &createdAt)
		if err != nil {
			continue
		}

		results = append(results, map[string]interface{}{
			"entry_date":  entryDate,
			"assets":      assets,
			"liabilities": liabilities,
			"net_worth":   netWorth,
			"notes":       notes,
			"created_at":  createdAt,
		})
	}

	return results, nil
}

func saveTaxCalculation(userID string, input lib.TaxOptimizerInput, result lib.TaxOptimizerResult) error {
	if db == nil {
		return nil
	}

	deductionsJSON, _ := json.Marshal(input.Deductions)
	creditsJSON, _ := json.Marshal(input.Credits)

	query := `
		INSERT INTO tax_calculations 
		(user_id, tax_year, income, filing_status, deductions, credits, tax_owed, effective_rate, marginal_rate)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := db.Exec(query, userID, input.TaxYear, input.Income, input.FilingStatus,
		deductionsJSON, creditsJSON, result.TaxOwed, result.EffectiveRate, result.MarginalRate)
	return err
}

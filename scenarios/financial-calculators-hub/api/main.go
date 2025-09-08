package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	
	"financial-calculators-hub/lib"
)

type HealthResponse struct {
	Status string `json:"status"`
	Service string `json:"service"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func main() {
	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}
	
	setupRoutes()
	
	log.Printf("Financial Calculators API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func setupRoutes() {
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/api/v1/calculate/fire", handleFIRE)
	http.HandleFunc("/api/v1/calculate/compound-interest", handleCompoundInterest)
	http.HandleFunc("/api/v1/calculate/mortgage", handleMortgage)
	http.HandleFunc("/api/v1/calculate/inflation", handleInflation)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HealthResponse{
		Status: "healthy",
		Service: "financial-calculators-hub",
	})
}

func handleFIRE(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var input lib.FIREInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid input: " + err.Error()})
		return
	}
	
	if input.AnnualIncome <= 0 || input.AnnualExpenses <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Income and expenses must be positive"})
		return
	}
	
	result := lib.CalculateFIRE(input)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleCompoundInterest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var input lib.CompoundInterestInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid input: " + err.Error()})
		return
	}
	
	if input.Principal <= 0 || input.Years <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Principal and years must be positive"})
		return
	}
	
	if input.CompoundFrequency == "" {
		input.CompoundFrequency = "monthly"
	}
	
	result := lib.CalculateCompoundInterest(input)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleMortgage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var input lib.MortgageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid input: " + err.Error()})
		return
	}
	
	if input.LoanAmount <= 0 || input.Years <= 0 || input.AnnualRate < 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid loan parameters"})
		return
	}
	
	result := lib.CalculateMortgage(input)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleInflation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var input lib.InflationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid input: " + err.Error()})
		return
	}
	
	if input.Amount <= 0 || input.Years <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Amount and years must be positive"})
		return
	}
	
	result := lib.CalculateInflation(input)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
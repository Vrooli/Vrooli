package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"financial-calculators-hub/lib"
)

type HealthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
	Readiness bool   `json:"readiness"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start financial-calculators-hub

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	// Initialize database connection (optional - won't fail if not available)
	if err := initDatabase(); err != nil {
		log.Printf("⚠️  Database not available (optional): %v", err)
		log.Println("📝 Running without database - calculations won't be persisted")
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
	http.HandleFunc("/api/v1/calculate/emergency-fund", handleEmergencyFund)
	http.HandleFunc("/api/v1/calculate/budget", handleBudgetAllocation)
	http.HandleFunc("/api/v1/calculate/debt-payoff", handleDebtPayoff)
	http.HandleFunc("/api/v1/calculate/net-worth", handleNetWorth)
	http.HandleFunc("/api/v1/calculate/tax", handleTaxOptimizer)
	http.HandleFunc("/api/v1/history", handleHistory)
	http.HandleFunc("/api/v1/batch", handleBatchCalculations)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HealthResponse{
		Status:    "healthy",
		Service:   "financial-calculators-hub",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Readiness: true,
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

	// Check if export format is requested
	format := r.URL.Query().Get("export")
	if format != "" {
		exportFIRE(w, input, result, format)
		return
	}

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

	// Check if export format is requested
	format := r.URL.Query().Get("export")
	if format != "" {
		exportCompoundInterest(w, input, result, format)
		return
	}

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

func handleEmergencyFund(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input lib.EmergencyFundInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid input: " + err.Error()})
		return
	}

	if input.MonthlyExpenses <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Monthly expenses must be positive"})
		return
	}

	if input.JobStability == "" {
		input.JobStability = "moderate"
	}

	result := lib.CalculateEmergencyFund(input)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleBudgetAllocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input lib.BudgetAllocationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid input: " + err.Error()})
		return
	}

	if input.MonthlyIncome <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Monthly income must be positive"})
		return
	}

	if input.BudgetMethod == "" {
		input.BudgetMethod = "50-30-20"
	}

	result := lib.CalculateBudgetAllocation(input)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleDebtPayoff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input lib.DebtPayoffInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid input: " + err.Error()})
		return
	}

	if len(input.Debts) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "At least one debt is required"})
		return
	}

	if input.Method == "" {
		input.Method = "avalanche"
	} else if input.Method != "avalanche" && input.Method != "snowball" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Method must be 'avalanche' or 'snowball'"})
		return
	}

	result := lib.CalculateDebtPayoff(input)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func exportFIRE(w http.ResponseWriter, input lib.FIREInput, result lib.FIREResult, format string) {
	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=fire-calculation.csv")

		csvWriter := csv.NewWriter(w)
		csvWriter.Write([]string{"Metric", "Value"})
		csvWriter.Write([]string{"Current Age", fmt.Sprintf("%.0f", input.CurrentAge)})
		csvWriter.Write([]string{"Retirement Age", fmt.Sprintf("%.1f", result.RetirementAge)})
		csvWriter.Write([]string{"Years to Retirement", fmt.Sprintf("%.1f", result.YearsToRetirement)})
		csvWriter.Write([]string{"Target Nest Egg", fmt.Sprintf("$%.2f", result.TargetNestEgg)})
		csvWriter.Write([]string{"Monthly Savings Required", fmt.Sprintf("$%.2f", result.MonthlySavingsReq)})
		csvWriter.Write([]string{"Annual Income", fmt.Sprintf("$%.2f", input.AnnualIncome)})
		csvWriter.Write([]string{"Annual Expenses", fmt.Sprintf("$%.2f", input.AnnualExpenses)})
		csvWriter.Write([]string{"Savings Rate", fmt.Sprintf("%.1f%%", input.SavingsRate)})
		csvWriter.Flush()

	case "text":
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Disposition", "attachment; filename=fire-calculation.txt")

		fmt.Fprintf(w, "FIRE CALCULATION REPORT\n")
		fmt.Fprintf(w, "=======================\n\n")
		fmt.Fprintf(w, "INPUTS:\n")
		fmt.Fprintf(w, "Current Age: %.0f\n", input.CurrentAge)
		fmt.Fprintf(w, "Current Savings: $%.2f\n", input.CurrentSavings)
		fmt.Fprintf(w, "Annual Income: $%.2f\n", input.AnnualIncome)
		fmt.Fprintf(w, "Annual Expenses: $%.2f\n", input.AnnualExpenses)
		fmt.Fprintf(w, "Savings Rate: %.1f%%\n", input.SavingsRate)
		fmt.Fprintf(w, "Expected Return: %.1f%%\n", input.ExpectedReturn)
		fmt.Fprintf(w, "\nRESULTS:\n")
		fmt.Fprintf(w, "Retirement Age: %.1f\n", result.RetirementAge)
		fmt.Fprintf(w, "Years to Retirement: %.1f\n", result.YearsToRetirement)
		fmt.Fprintf(w, "Target Nest Egg: $%.2f\n", result.TargetNestEgg)
		fmt.Fprintf(w, "Monthly Savings Required: $%.2f\n\n", result.MonthlySavingsReq)
		fmt.Fprintf(w, "\nPROJECTED NEST EGG BY AGE:\n")
		for age, amount := range result.ProjectedByAge {
			fmt.Fprintf(w, "Age %d: $%.2f\n", age, amount)
		}

	case "pdf":
		exportFIREPDF(w, input, result)

	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid export format. Use csv, text, or pdf"})
	}
}

func exportCompoundInterest(w http.ResponseWriter, input lib.CompoundInterestInput, result lib.CompoundInterestResult, format string) {
	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=compound-interest.csv")

		csvWriter := csv.NewWriter(w)
		csvWriter.Write([]string{"Year", "Balance", "Contribution", "Interest"})
		for _, year := range result.YearByYear {
			csvWriter.Write([]string{
				strconv.Itoa(year.Year),
				fmt.Sprintf("%.2f", year.Balance),
				fmt.Sprintf("%.2f", year.Contribution),
				fmt.Sprintf("%.2f", year.Interest),
			})
		}
		csvWriter.Write([]string{""})
		csvWriter.Write([]string{"Final Amount", fmt.Sprintf("%.2f", result.FinalAmount)})
		csvWriter.Write([]string{"Total Contributions", fmt.Sprintf("%.2f", result.TotalContributions)})
		csvWriter.Write([]string{"Total Interest", fmt.Sprintf("%.2f", result.TotalInterest)})
		csvWriter.Flush()

	case "text":
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Disposition", "attachment; filename=compound-interest.txt")

		fmt.Fprintf(w, "COMPOUND INTEREST REPORT\n")
		fmt.Fprintf(w, "========================\n\n")
		fmt.Fprintf(w, "INPUTS:\n")
		fmt.Fprintf(w, "Principal: $%.2f\n", input.Principal)
		fmt.Fprintf(w, "Annual Rate: %.1f%%\n", input.AnnualRate)
		fmt.Fprintf(w, "Years: %.0f\n", input.Years)
		fmt.Fprintf(w, "Monthly Contribution: $%.2f\n", input.MonthlyContribution)
		fmt.Fprintf(w, "Compound Frequency: %s\n", input.CompoundFrequency)
		fmt.Fprintf(w, "\nFINAL RESULTS:\n")
		fmt.Fprintf(w, "Final Amount: $%.2f\n", result.FinalAmount)
		fmt.Fprintf(w, "Total Contributions: $%.2f\n", result.TotalContributions)
		fmt.Fprintf(w, "Total Interest: $%.2f\n\n", result.TotalInterest)
		fmt.Fprintf(w, "YEAR-BY-YEAR BREAKDOWN:\n")
		for _, year := range result.YearByYear {
			fmt.Fprintf(w, "Year %d: Balance=$%.2f, Contribution=$%.2f, Interest=$%.2f\n",
				year.Year, year.Balance, year.Contribution, year.Interest)
		}

	case "pdf":
		exportCompoundInterestPDF(w, input, result)

	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid export format. Use csv, text, or pdf"})
	}
}
// Test change

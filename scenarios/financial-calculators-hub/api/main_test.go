package main

import (
	"net/http"
	"testing"

	"financial-calculators-hub/lib"
)

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	envCleanup := setupTestEnvironment(t)
	defer envCleanup()

	t.Run("Success", func(t *testing.T) {
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}, handleHealth)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)

		var response HealthResponse
		decodeJSONResponse(t, w, &response)

		if response.Status != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", response.Status)
		}

		if response.Service != "financial-calculators-hub" {
			t.Errorf("Expected service 'financial-calculators-hub', got '%s'", response.Service)
		}
	})
}

// TestHandleFIRE tests the FIRE calculator endpoint
func TestHandleFIRE(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	envCleanup := setupTestEnvironment(t)
	defer envCleanup()

	t.Run("Success", func(t *testing.T) {
		input := lib.FIREInput{
			CurrentAge:           30,
			CurrentSavings:       100000,
			AnnualIncome:         100000,
			AnnualExpenses:       50000,
			SavingsRate:          50,
			ExpectedReturn:       7,
			TargetWithdrawalRate: 4,
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/calculate/fire",
			Body:   input,
		}, handleFIRE)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)

		var result lib.FIREResult
		decodeJSONResponse(t, w, &result)

		// Verify reasonable results
		if result.RetirementAge <= input.CurrentAge {
			t.Error("Retirement age should be greater than current age")
		}

		if result.TargetNestEgg <= 0 {
			t.Error("Target nest egg should be positive")
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		scenarios := NewTestScenarioBuilder().
			AddInvalidJSON("/api/v1/calculate/fire", "POST").
			AddInvalidMethod("/api/v1/calculate/fire", "GET").
			AddNegativeValues("/api/v1/calculate/fire", "POST", lib.FIREInput{
				CurrentAge:     30,
				CurrentSavings: 100000,
				AnnualIncome:   -100000, // Negative
				AnnualExpenses: 50000,
				SavingsRate:    50,
			}).
			AddZeroValues("/api/v1/calculate/fire", "POST", lib.FIREInput{
				CurrentAge:     30,
				CurrentSavings: 100000,
				AnnualIncome:   0, // Zero
				AnnualExpenses: 0,
				SavingsRate:    50,
			}).
			Build()

		RunErrorScenarios(t, handleFIRE, scenarios)
	})
}

// TestHandleCompoundInterest tests the compound interest calculator endpoint
func TestHandleCompoundInterest(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	envCleanup := setupTestEnvironment(t)
	defer envCleanup()

	t.Run("Success_NoContributions", func(t *testing.T) {
		input := lib.CompoundInterestInput{
			Principal:           10000,
			AnnualRate:          7,
			Years:               10,
			MonthlyContribution: 0,
			CompoundFrequency:   "annually",
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/calculate/compound-interest",
			Body:   input,
		}, handleCompoundInterest)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)

		var result lib.CompoundInterestResult
		decodeJSONResponse(t, w, &result)

		if result.FinalAmount <= input.Principal {
			t.Error("Final amount should be greater than principal")
		}

		if result.TotalInterest <= 0 {
			t.Error("Total interest should be positive")
		}
	})

	t.Run("Success_WithContributions", func(t *testing.T) {
		input := lib.CompoundInterestInput{
			Principal:           5000,
			AnnualRate:          6,
			Years:               5,
			MonthlyContribution: 100,
			CompoundFrequency:   "monthly",
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/calculate/compound-interest",
			Body:   input,
		}, handleCompoundInterest)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)

		var result lib.CompoundInterestResult
		decodeJSONResponse(t, w, &result)

		expectedContributions := input.Principal + (input.MonthlyContribution * 12 * input.Years)
		assertFloatEqual(t, result.TotalContributions, expectedContributions, 1, "TotalContributions")
	})

	t.Run("DefaultFrequency", func(t *testing.T) {
		input := lib.CompoundInterestInput{
			Principal:  1000,
			AnnualRate: 5,
			Years:      1,
			// CompoundFrequency omitted - should default to monthly
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/calculate/compound-interest",
			Body:   input,
		}, handleCompoundInterest)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)
	})

	t.Run("ErrorCases", func(t *testing.T) {
		scenarios := NewTestScenarioBuilder().
			AddInvalidJSON("/api/v1/calculate/compound-interest", "POST").
			AddInvalidMethod("/api/v1/calculate/compound-interest", "GET").
			AddZeroValues("/api/v1/calculate/compound-interest", "POST", lib.CompoundInterestInput{
				Principal:  0,
				AnnualRate: 5,
				Years:      0,
			}).
			Build()

		RunErrorScenarios(t, handleCompoundInterest, scenarios)
	})
}

// TestHandleMortgage tests the mortgage calculator endpoint
func TestHandleMortgage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	envCleanup := setupTestEnvironment(t)
	defer envCleanup()

	t.Run("Success_BasicMortgage", func(t *testing.T) {
		input := lib.MortgageInput{
			LoanAmount:          300000,
			AnnualRate:          4.5,
			Years:               30,
			DownPayment:         0,
			ExtraMonthlyPayment: 0,
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/calculate/mortgage",
			Body:   input,
		}, handleMortgage)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)

		var result lib.MortgageResult
		decodeJSONResponse(t, w, &result)

		if result.MonthlyPayment <= 0 {
			t.Error("Monthly payment should be positive")
		}

		if result.TotalPaid <= input.LoanAmount {
			t.Error("Total paid should be greater than loan amount")
		}
	})

	t.Run("Success_WithDownPayment", func(t *testing.T) {
		input := lib.MortgageInput{
			LoanAmount:  200000,
			AnnualRate:  5,
			Years:       15,
			DownPayment: 40000,
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/calculate/mortgage",
			Body:   input,
		}, handleMortgage)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)
	})

	t.Run("ErrorCases", func(t *testing.T) {
		scenarios := NewTestScenarioBuilder().
			AddInvalidJSON("/api/v1/calculate/mortgage", "POST").
			AddInvalidMethod("/api/v1/calculate/mortgage", "PUT").
			AddZeroValues("/api/v1/calculate/mortgage", "POST", lib.MortgageInput{
				LoanAmount: 0,
				AnnualRate: 5,
				Years:      0,
			}).
			AddNegativeValues("/api/v1/calculate/mortgage", "POST", lib.MortgageInput{
				LoanAmount: -100000,
				AnnualRate: 5,
				Years:      30,
			}).
			Build()

		RunErrorScenarios(t, handleMortgage, scenarios)
	})
}

// TestHandleInflation tests the inflation calculator endpoint
func TestHandleInflation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	envCleanup := setupTestEnvironment(t)
	defer envCleanup()

	t.Run("Success", func(t *testing.T) {
		input := lib.InflationInput{
			Amount:        100000,
			Years:         10,
			InflationRate: 3,
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/calculate/inflation",
			Body:   input,
		}, handleInflation)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)

		var result lib.InflationResult
		decodeJSONResponse(t, w, &result)

		if result.FutureValue <= input.Amount {
			t.Error("Future value should be greater than current amount with positive inflation")
		}

		if result.PurchasingPower >= input.Amount {
			t.Error("Purchasing power should be less than current amount with positive inflation")
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		scenarios := NewTestScenarioBuilder().
			AddInvalidJSON("/api/v1/calculate/inflation", "POST").
			AddInvalidMethod("/api/v1/calculate/inflation", "DELETE").
			AddZeroValues("/api/v1/calculate/inflation", "POST", lib.InflationInput{
				Amount: 0,
				Years:  0,
			}).
			Build()

		RunErrorScenarios(t, handleInflation, scenarios)
	})
}

// TestHandleEmergencyFund tests the emergency fund calculator endpoint
func TestHandleEmergencyFund(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	envCleanup := setupTestEnvironment(t)
	defer envCleanup()

	t.Run("Success", func(t *testing.T) {
		input := lib.EmergencyFundInput{
			MonthlyExpenses:        3000,
			JobStability:           "stable",
			NumberOfDependents:     0,
			HasDisabilityInsurance: false,
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/calculate/emergency-fund",
			Body:   input,
		}, handleEmergencyFund)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)

		var result lib.EmergencyFundResult
		decodeJSONResponse(t, w, &result)

		if result.MinimumMonths <= 0 {
			t.Error("Minimum months should be positive")
		}

		if result.RecommendedMonths < result.MinimumMonths {
			t.Error("Recommended months should be >= minimum months")
		}

		expectedMin := float64(result.MinimumMonths) * input.MonthlyExpenses
		assertFloatEqual(t, result.MinimumAmount, expectedMin, 1, "MinimumAmount")
	})

	t.Run("DefaultJobStability", func(t *testing.T) {
		input := lib.EmergencyFundInput{
			MonthlyExpenses: 3000,
			// JobStability omitted - should default to "moderate"
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/calculate/emergency-fund",
			Body:   input,
		}, handleEmergencyFund)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)
	})

	t.Run("ErrorCases", func(t *testing.T) {
		scenarios := NewTestScenarioBuilder().
			AddInvalidJSON("/api/v1/calculate/emergency-fund", "POST").
			AddZeroValues("/api/v1/calculate/emergency-fund", "POST", lib.EmergencyFundInput{
				MonthlyExpenses: 0,
			}).
			Build()

		RunErrorScenarios(t, handleEmergencyFund, scenarios)
	})
}

// TestHandleBudgetAllocation tests the budget allocation calculator endpoint
func TestHandleBudgetAllocation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	envCleanup := setupTestEnvironment(t)
	defer envCleanup()

	t.Run("Success_50_30_20", func(t *testing.T) {
		input := lib.BudgetAllocationInput{
			MonthlyIncome: 5000,
			BudgetMethod:  "50-30-20",
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/calculate/budget",
			Body:   input,
		}, handleBudgetAllocation)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)

		var result lib.BudgetAllocationResult
		decodeJSONResponse(t, w, &result)

		total := result.Needs + result.Wants + result.Savings
		assertFloatEqual(t, total, input.MonthlyIncome, 1, "Total allocation")
	})

	t.Run("DefaultMethod", func(t *testing.T) {
		input := lib.BudgetAllocationInput{
			MonthlyIncome: 4000,
			// BudgetMethod omitted - should default to "50-30-20"
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/calculate/budget",
			Body:   input,
		}, handleBudgetAllocation)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)

		var result lib.BudgetAllocationResult
		decodeJSONResponse(t, w, &result)

		if result.Method != "50-30-20" {
			t.Errorf("Expected default method '50-30-20', got '%s'", result.Method)
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		scenarios := NewTestScenarioBuilder().
			AddInvalidJSON("/api/v1/calculate/budget", "POST").
			AddZeroValues("/api/v1/calculate/budget", "POST", lib.BudgetAllocationInput{
				MonthlyIncome: 0,
			}).
			Build()

		RunErrorScenarios(t, handleBudgetAllocation, scenarios)
	})
}

// TestHandleDebtPayoff tests the debt payoff calculator endpoint
func TestHandleDebtPayoff(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	envCleanup := setupTestEnvironment(t)
	defer envCleanup()

	t.Run("Success_Avalanche", func(t *testing.T) {
		input := lib.DebtPayoffInput{
			Debts: []lib.Debt{
				{Name: "Credit Card", Balance: 5000, InterestRate: 18, MinimumPayment: 150},
				{Name: "Student Loan", Balance: 10000, InterestRate: 6, MinimumPayment: 100},
			},
			ExtraPayment: 200,
			Method:       "avalanche",
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/calculate/debt-payoff",
			Body:   input,
		}, handleDebtPayoff)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)

		var result lib.DebtPayoffResult
		decodeJSONResponse(t, w, &result)

		if result.TotalMonths <= 0 {
			t.Error("Total months should be positive")
		}

		if len(result.PayoffSchedule) == 0 {
			t.Error("Payoff schedule should not be empty")
		}
	})

	t.Run("Success_Snowball", func(t *testing.T) {
		input := lib.DebtPayoffInput{
			Debts: []lib.Debt{
				{Name: "Small Card", Balance: 1000, InterestRate: 15, MinimumPayment: 50},
				{Name: "Big Loan", Balance: 20000, InterestRate: 7, MinimumPayment: 300},
			},
			ExtraPayment: 500,
			Method:       "snowball",
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/calculate/debt-payoff",
			Body:   input,
		}, handleDebtPayoff)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)
	})

	t.Run("DefaultMethod", func(t *testing.T) {
		input := lib.DebtPayoffInput{
			Debts: []lib.Debt{
				{Name: "Debt", Balance: 5000, InterestRate: 10, MinimumPayment: 100},
			},
			ExtraPayment: 100,
			// Method omitted - should default to "avalanche"
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/calculate/debt-payoff",
			Body:   input,
		}, handleDebtPayoff)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)
	})

	t.Run("ErrorCases", func(t *testing.T) {
		scenarios := NewTestScenarioBuilder().
			AddInvalidJSON("/api/v1/calculate/debt-payoff", "POST").
			AddMissingRequiredFields("/api/v1/calculate/debt-payoff", "POST", lib.DebtPayoffInput{
				Debts: []lib.Debt{}, // Empty debts array
			}).
			Build()

		RunErrorScenarios(t, handleDebtPayoff, scenarios)
	})

	t.Run("InvalidMethod", func(t *testing.T) {
		input := lib.DebtPayoffInput{
			Debts: []lib.Debt{
				{Name: "Debt", Balance: 5000, InterestRate: 10, MinimumPayment: 100},
			},
			Method: "invalid-method",
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/calculate/debt-payoff",
			Body:   input,
		}, handleDebtPayoff)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestHandleBatchCalculations tests the batch calculations endpoint
func TestHandleBatchCalculations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	envCleanup := setupTestEnvironment(t)
	defer envCleanup()

	t.Run("Success_MultipleBatchCalculations", func(t *testing.T) {
		input := lib.BatchCalculationInput{
			Calculations: []map[string]interface{}{
				{
					"type": "fire",
					"input": map[string]interface{}{
						"current_age":            30,
						"current_savings":        100000,
						"annual_income":          100000,
						"annual_expenses":        50000,
						"savings_rate":           50,
						"expected_return":        7,
						"target_withdrawal_rate": 4,
					},
				},
				{
					"type": "compound-interest",
					"input": map[string]interface{}{
						"principal":            10000,
						"annual_rate":          7,
						"years":                10,
						"monthly_contribution": 0,
						"compound_frequency":   "annually",
					},
				},
			},
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/batch",
			Body:   input,
		}, handleBatchCalculations)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)

		var response map[string]interface{}
		decodeJSONResponse(t, w, &response)

		results, ok := response["results"].([]interface{})
		if !ok {
			t.Fatal("Expected results array in response")
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		scenarios := NewTestScenarioBuilder().
			AddInvalidJSON("/api/v1/batch", "POST").
			AddInvalidMethod("/api/v1/batch", "GET").
			AddMissingRequiredFields("/api/v1/batch", "POST", lib.BatchCalculationInput{
				Calculations: []map[string]interface{}{}, // Empty array
			}).
			Build()

		RunErrorScenarios(t, handleBatchCalculations, scenarios)
	})

	t.Run("UnknownCalculationType", func(t *testing.T) {
		input := lib.BatchCalculationInput{
			Calculations: []map[string]interface{}{
				{
					"type":  "unknown-type",
					"input": map[string]interface{}{},
				},
			},
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/batch",
			Body:   input,
		}, handleBatchCalculations)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)

		var response map[string]interface{}
		decodeJSONResponse(t, w, &response)

		results := response["results"].([]interface{})
		firstResult := results[0].(map[string]interface{})

		if _, hasError := firstResult["error"]; !hasError {
			t.Error("Expected error field for unknown calculation type")
		}
	})
}

// TestHandleNetWorth tests the net worth calculator endpoint
func TestHandleNetWorth(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	envCleanup := setupTestEnvironment(t)
	defer envCleanup()

	t.Run("Success_PositiveNetWorth", func(t *testing.T) {
		input := lib.NetWorthInput{
			Assets: map[string]float64{
				"cash":        50000,
				"investments": 150000,
				"real_estate": 300000,
			},
			Liabilities: map[string]float64{
				"mortgage":     200000,
				"car_loan":     20000,
				"credit_cards": 5000,
			},
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/calculate/net-worth",
			Body:   input,
		}, handleNetWorthCalculation)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)

		var result lib.NetWorthResult
		decodeJSONResponse(t, w, &result)

		expectedAssets := 500000.0
		expectedLiabilities := 225000.0
		expectedNetWorth := 275000.0

		assertFloatEqual(t, result.TotalAssets, expectedAssets, 0.01, "TotalAssets")
		assertFloatEqual(t, result.TotalLiabilities, expectedLiabilities, 0.01, "TotalLiabilities")
		assertFloatEqual(t, result.NetWorth, expectedNetWorth, 0.01, "NetWorth")
	})

	t.Run("Success_NegativeNetWorth", func(t *testing.T) {
		input := lib.NetWorthInput{
			Assets: map[string]float64{
				"cash": 10000,
				"401k": 25000,
			},
			Liabilities: map[string]float64{
				"student_loans": 80000,
				"credit_cards":  15000,
			},
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/calculate/net-worth",
			Body:   input,
		}, handleNetWorthCalculation)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)

		var result lib.NetWorthResult
		decodeJSONResponse(t, w, &result)

		if result.NetWorth >= 0 {
			t.Error("Expected negative net worth")
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		scenarios := NewTestScenarioBuilder().
			AddInvalidJSON("/api/v1/calculate/net-worth", "POST").
			Build()

		RunErrorScenarios(t, handleNetWorthCalculation, scenarios)
	})
}

// TestHandleTaxOptimizer tests the tax optimizer endpoint
func TestHandleTaxOptimizer(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	envCleanup := setupTestEnvironment(t)
	defer envCleanup()

	t.Run("Success_SingleFiler", func(t *testing.T) {
		input := lib.TaxOptimizerInput{
			Income:       75000,
			TaxYear:      2024,
			FilingStatus: "single",
			Deductions:   map[string]float64{},
			Credits:      map[string]float64{},
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/calculate/tax",
			Body:   input,
		}, handleTaxOptimizer)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)

		var result lib.TaxOptimizerResult
		decodeJSONResponse(t, w, &result)

		if result.TaxableIncome <= 0 {
			t.Error("Taxable income should be positive")
		}

		if result.TaxOwed <= 0 {
			t.Error("Tax owed should be positive")
		}

		if result.EffectiveRate <= 0 || result.EffectiveRate >= 100 {
			t.Errorf("Effective rate should be between 0 and 100, got %v", result.EffectiveRate)
		}
	})

	t.Run("Success_WithDeductionsAndCredits", func(t *testing.T) {
		input := lib.TaxOptimizerInput{
			Income:       200000,
			TaxYear:      2024,
			FilingStatus: "single",
			Deductions: map[string]float64{
				"401k":              23000,
				"mortgage_interest": 12000,
			},
			Credits: map[string]float64{
				"education": 2500,
			},
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/calculate/tax",
			Body:   input,
		}, handleTaxOptimizer)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK)

		var result lib.TaxOptimizerResult
		decodeJSONResponse(t, w, &result)

		if result.TaxableIncome >= input.Income {
			t.Error("Taxable income should be less than total income with deductions")
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		scenarios := NewTestScenarioBuilder().
			AddInvalidJSON("/api/v1/calculate/tax", "POST").
			AddInvalidMethod("/api/v1/calculate/tax", "GET").
			AddZeroValues("/api/v1/calculate/tax", "POST", lib.TaxOptimizerInput{
				Income: 0,
			}).
			AddNegativeValues("/api/v1/calculate/tax", "POST", lib.TaxOptimizerInput{
				Income: -50000,
			}).
			Build()

		RunErrorScenarios(t, handleTaxOptimizer, scenarios)
	})
}

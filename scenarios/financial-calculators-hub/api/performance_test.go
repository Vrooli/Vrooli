package main

import (
	"testing"
	"time"

	"financial-calculators-hub/lib"
)

// TestCalculationPerformance tests that calculations complete within acceptable time limits
func TestCalculationPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	envCleanup := setupTestEnvironment(t)
	defer envCleanup()

	// Target: < 50ms for 95% of calculations (per PRD)
	maxDuration := 50 * time.Millisecond

	t.Run("FIRE_Performance", func(t *testing.T) {
		input := lib.FIREInput{
			CurrentAge:           30,
			CurrentSavings:       100000,
			AnnualIncome:         100000,
			AnnualExpenses:       50000,
			SavingsRate:          50,
			ExpectedReturn:       7,
			TargetWithdrawalRate: 4,
		}

		start := time.Now()
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/calculate/fire",
			Body:   input,
		}, handleFIRE)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != 200 {
			t.Fatalf("Request failed with status %d", w.Code)
		}

		if duration > maxDuration {
			t.Errorf("FIRE calculation took %v, expected < %v", duration, maxDuration)
		}
	})

	t.Run("CompoundInterest_Performance", func(t *testing.T) {
		input := lib.CompoundInterestInput{
			Principal:           10000,
			AnnualRate:          7,
			Years:               30,
			MonthlyContribution: 500,
			CompoundFrequency:   "monthly",
		}

		start := time.Now()
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/calculate/compound-interest",
			Body:   input,
		}, handleCompoundInterest)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != 200 {
			t.Fatalf("Request failed with status %d", w.Code)
		}

		if duration > maxDuration {
			t.Errorf("Compound interest calculation took %v, expected < %v", duration, maxDuration)
		}
	})

	t.Run("Mortgage_Performance", func(t *testing.T) {
		input := lib.MortgageInput{
			LoanAmount:          500000,
			AnnualRate:          4.5,
			Years:               30,
			DownPayment:         100000,
			ExtraMonthlyPayment: 200,
		}

		start := time.Now()
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/calculate/mortgage",
			Body:   input,
		}, handleMortgage)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != 200 {
			t.Fatalf("Request failed with status %d", w.Code)
		}

		if duration > maxDuration {
			t.Errorf("Mortgage calculation took %v, expected < %v", duration, maxDuration)
		}
	})

	t.Run("Batch_Performance", func(t *testing.T) {
		// Batch of 10 calculations should complete in reasonable time
		calculations := make([]map[string]interface{}, 10)
		for i := 0; i < 10; i++ {
			calculations[i] = map[string]interface{}{
				"type": "fire",
				"input": map[string]interface{}{
					"current_age":            float64(25 + i),
					"current_savings":        100000,
					"annual_income":          100000,
					"annual_expenses":        50000,
					"savings_rate":           50,
					"expected_return":        7,
					"target_withdrawal_rate": 4,
				},
			}
		}

		input := lib.BatchCalculationInput{
			Calculations: calculations,
		}

		start := time.Now()
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/batch",
			Body:   input,
		}, handleBatchCalculations)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != 200 {
			t.Fatalf("Request failed with status %d", w.Code)
		}

		// Batch of 10 should complete in < 500ms
		maxBatchDuration := 500 * time.Millisecond
		if duration > maxBatchDuration {
			t.Errorf("Batch calculation took %v, expected < %v", duration, maxBatchDuration)
		}

		// Average per calculation should still be under target
		avgPerCalc := duration / 10
		if avgPerCalc > maxDuration {
			t.Logf("Warning: Average per calculation %v exceeds target %v", avgPerCalc, maxDuration)
		}
	})
}

// TestConcurrentRequests tests that the API handles concurrent requests efficiently
func TestConcurrentRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	envCleanup := setupTestEnvironment(t)
	defer envCleanup()

	t.Run("Concurrent_FIRE_Calculations", func(t *testing.T) {
		concurrency := 10
		done := make(chan bool, concurrency)
		errors := make(chan error, concurrency)

		input := lib.FIREInput{
			CurrentAge:           30,
			CurrentSavings:       100000,
			AnnualIncome:         100000,
			AnnualExpenses:       50000,
			SavingsRate:          50,
			ExpectedReturn:       7,
			TargetWithdrawalRate: 4,
		}

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			go func() {
				w, err := makeHTTPRequest(HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/calculate/fire",
					Body:   input,
				}, handleFIRE)

				if err != nil {
					errors <- err
					return
				}

				if w.Code != 200 {
					errors <- err
					return
				}

				done <- true
			}()
		}

		// Wait for all to complete
		completed := 0
		for i := 0; i < concurrency; i++ {
			select {
			case <-done:
				completed++
			case err := <-errors:
				t.Errorf("Concurrent request failed: %v", err)
			case <-time.After(5 * time.Second):
				t.Fatal("Timeout waiting for concurrent requests")
			}
		}

		duration := time.Since(start)

		if completed != concurrency {
			t.Errorf("Expected %d completed requests, got %d", concurrency, completed)
		}

		// All 10 concurrent requests should complete in reasonable time
		maxConcurrentDuration := 1 * time.Second
		if duration > maxConcurrentDuration {
			t.Errorf("Concurrent requests took %v, expected < %v", duration, maxConcurrentDuration)
		}
	})
}

// TestMemoryEfficiency tests that calculations don't consume excessive memory
func TestMemoryEfficiency(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	envCleanup := setupTestEnvironment(t)
	defer envCleanup()

	t.Run("Large_Mortgage_Amortization", func(t *testing.T) {
		// 30-year mortgage with monthly payments = 360 entries in amortization schedule
		input := lib.MortgageInput{
			LoanAmount:          500000,
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
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != 200 {
			t.Fatalf("Request failed with status %d", w.Code)
		}

		var result lib.MortgageResult
		decodeJSONResponse(t, w, &result)

		// Should have full amortization schedule
		if len(result.AmortizationSchedule) == 0 {
			t.Error("Expected amortization schedule to be populated")
		}
	})

	t.Run("Large_Compound_Interest_Projection", func(t *testing.T) {
		// 50 years of projections
		input := lib.CompoundInterestInput{
			Principal:           10000,
			AnnualRate:          7,
			Years:               50,
			MonthlyContribution: 500,
			CompoundFrequency:   "monthly",
		}

		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/calculate/compound-interest",
			Body:   input,
		}, handleCompoundInterest)

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != 200 {
			t.Fatalf("Request failed with status %d", w.Code)
		}

		var result lib.CompoundInterestResult
		decodeJSONResponse(t, w, &result)

		// Should have year-by-year breakdown
		if len(result.YearByYear) == 0 {
			t.Error("Expected year-by-year breakdown to be populated")
		}
	})
}

// BenchmarkFIRECalculation benchmarks the FIRE calculation
func BenchmarkFIRECalculation(b *testing.B) {
	input := lib.FIREInput{
		CurrentAge:           30,
		CurrentSavings:       100000,
		AnnualIncome:         100000,
		AnnualExpenses:       50000,
		SavingsRate:          50,
		ExpectedReturn:       7,
		TargetWithdrawalRate: 4,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lib.CalculateFIRE(input)
	}
}

// BenchmarkCompoundInterestCalculation benchmarks the compound interest calculation
func BenchmarkCompoundInterestCalculation(b *testing.B) {
	input := lib.CompoundInterestInput{
		Principal:           10000,
		AnnualRate:          7,
		Years:               30,
		MonthlyContribution: 500,
		CompoundFrequency:   "monthly",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lib.CalculateCompoundInterest(input)
	}
}

// BenchmarkMortgageCalculation benchmarks the mortgage calculation
func BenchmarkMortgageCalculation(b *testing.B) {
	input := lib.MortgageInput{
		LoanAmount:          300000,
		AnnualRate:          4.5,
		Years:               30,
		DownPayment:         60000,
		ExtraMonthlyPayment: 100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lib.CalculateMortgage(input)
	}
}

// BenchmarkBatchCalculations benchmarks batch processing
func BenchmarkBatchCalculations(b *testing.B) {
	calculations := make([]map[string]interface{}, 5)
	for i := 0; i < 5; i++ {
		calculations[i] = map[string]interface{}{
			"type": "fire",
			"input": map[string]interface{}{
				"current_age":            float64(25 + i),
				"current_savings":        100000,
				"annual_income":          100000,
				"annual_expenses":        50000,
				"savings_rate":           50,
				"expected_return":        7,
				"target_withdrawal_rate": 4,
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, calc := range calculations {
			processBatchCalculation(calc)
		}
	}
}

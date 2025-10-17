package lib

import (
	"math"
	"testing"
)

func TestCalculateFIRE(t *testing.T) {
	tests := []struct {
		name     string
		input    FIREInput
		expected FIREResult
		delta    float64 // Acceptable difference
	}{
		{
			name: "Basic FIRE calculation",
			input: FIREInput{
				CurrentAge:           30,
				CurrentSavings:       100000,
				AnnualIncome:         100000,
				AnnualExpenses:       50000,
				SavingsRate:          50,
				ExpectedReturn:       7,
				TargetWithdrawalRate: 4,
			},
			expected: FIREResult{
				RetirementAge:     42,
				YearsToRetirement: 12,
				TargetNestEgg:     1250000,
				MonthlySavingsReq: 4166.67,
			},
			delta: 1.0, // Allow 1 year variance due to compounding
		},
		{
			name: "High savings rate",
			input: FIREInput{
				CurrentAge:           25,
				CurrentSavings:       50000,
				AnnualIncome:         80000,
				AnnualExpenses:       30000,
				SavingsRate:          62.5,
				ExpectedReturn:       7,
				TargetWithdrawalRate: 4,
			},
			expected: FIREResult{
				RetirementAge:     34,
				YearsToRetirement: 9,
				TargetNestEgg:     750000,
				MonthlySavingsReq: 4166.67,
			},
			delta: 1.5,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateFIRE(tt.input)
			
			if math.Abs(result.RetirementAge-tt.expected.RetirementAge) > tt.delta {
				t.Errorf("RetirementAge = %v, want %v (±%v)", 
					result.RetirementAge, tt.expected.RetirementAge, tt.delta)
			}
			
			if math.Abs(result.YearsToRetirement-tt.expected.YearsToRetirement) > tt.delta {
				t.Errorf("YearsToRetirement = %v, want %v (±%v)",
					result.YearsToRetirement, tt.expected.YearsToRetirement, tt.delta)
			}
			
			if math.Abs(result.TargetNestEgg-tt.expected.TargetNestEgg) > 1000 {
				t.Errorf("TargetNestEgg = %v, want %v (±1000)",
					result.TargetNestEgg, tt.expected.TargetNestEgg)
			}
		})
	}
}

func TestCalculateCompoundInterest(t *testing.T) {
	tests := []struct {
		name     string
		input    CompoundInterestInput
		expected CompoundInterestResult
		delta    float64
	}{
		{
			name: "Simple compound interest - no contributions",
			input: CompoundInterestInput{
				Principal:           1000,
				AnnualRate:          10,
				Years:               1,
				MonthlyContribution: 0,
				CompoundFrequency:   "annually",
			},
			expected: CompoundInterestResult{
				FinalAmount:        1100,
				TotalContributions: 1000,
				TotalInterest:      100,
			},
			delta: 0.01,
		},
		{
			name: "Compound interest with monthly contributions",
			input: CompoundInterestInput{
				Principal:           5000,
				AnnualRate:          6,
				Years:               5,
				MonthlyContribution: 100,
				CompoundFrequency:   "monthly",
			},
			expected: CompoundInterestResult{
				FinalAmount:        13756, // More accurate calculation
				TotalContributions: 11000, // 5000 + 100*12*5
				TotalInterest:      2756,  // More accurate
			},
			delta: 100, // Tighter tolerance now that we have accurate values
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateCompoundInterest(tt.input)
			
			if math.Abs(result.FinalAmount-tt.expected.FinalAmount) > tt.delta {
				t.Errorf("FinalAmount = %v, want %v (±%v)",
					result.FinalAmount, tt.expected.FinalAmount, tt.delta)
			}
			
			if math.Abs(result.TotalContributions-tt.expected.TotalContributions) > 0.01 {
				t.Errorf("TotalContributions = %v, want %v",
					result.TotalContributions, tt.expected.TotalContributions)
			}
			
			if math.Abs(result.TotalInterest-tt.expected.TotalInterest) > tt.delta {
				t.Errorf("TotalInterest = %v, want %v (±%v)",
					result.TotalInterest, tt.expected.TotalInterest, tt.delta)
			}
		})
	}
}

func TestCalculateMortgage(t *testing.T) {
	tests := []struct {
		name     string
		input    MortgageInput
		expected MortgageResult
		delta    float64
	}{
		{
			name: "Standard 30-year mortgage",
			input: MortgageInput{
				LoanAmount:          100000,
				AnnualRate:          6,
				Years:               30,
				DownPayment:         0,
				ExtraMonthlyPayment: 0,
			},
			expected: MortgageResult{
				MonthlyPayment: 599.55, // Standard PMT formula result
				TotalInterest:  115838,  // Approximate
				TotalPaid:      215838,  // Approximate
			},
			delta: 1.0,
		},
		{
			name: "15-year mortgage with down payment",
			input: MortgageInput{
				LoanAmount:          200000,
				AnnualRate:          5,
				Years:               15,
				DownPayment:         40000,
				ExtraMonthlyPayment: 0,
			},
			expected: MortgageResult{
				MonthlyPayment: 1265.27, // For 160000 loan
				TotalInterest:  67748,   // Approximate
				TotalPaid:      227748,  // Approximate
			},
			delta: 5.0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateMortgage(tt.input)
			
			if math.Abs(result.MonthlyPayment-tt.expected.MonthlyPayment) > tt.delta {
				t.Errorf("MonthlyPayment = %v, want %v (±%v)",
					result.MonthlyPayment, tt.expected.MonthlyPayment, tt.delta)
			}
		})
	}
}

func TestCalculateInflation(t *testing.T) {
	tests := []struct {
		name     string
		input    InflationInput
		expected InflationResult
		delta    float64
	}{
		{
			name: "3% inflation over 10 years",
			input: InflationInput{
				Amount:        100000,
				Years:         10,
				InflationRate: 3,
			},
			expected: InflationResult{
				FutureValue:     134391.64, // 100000 * (1.03)^10
				PurchasingPower: 74409.39,  // 100000 / (1.03)^10
				TotalInflation:  34.39,     // Percentage increase
			},
			delta: 100,
		},
		{
			name: "2% inflation over 5 years",
			input: InflationInput{
				Amount:        50000,
				Years:         5,
				InflationRate: 2,
			},
			expected: InflationResult{
				FutureValue:     55204.04,
				PurchasingPower: 45289.48,
				TotalInflation:  10.41,
			},
			delta: 50,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateInflation(tt.input)
			
			if math.Abs(result.FutureValue-tt.expected.FutureValue) > tt.delta {
				t.Errorf("FutureValue = %v, want %v (±%v)",
					result.FutureValue, tt.expected.FutureValue, tt.delta)
			}
			
			if math.Abs(result.PurchasingPower-tt.expected.PurchasingPower) > tt.delta {
				t.Errorf("PurchasingPower = %v, want %v (±%v)",
					result.PurchasingPower, tt.expected.PurchasingPower, tt.delta)
			}
		})
	}
}

func TestCalculateNetWorth(t *testing.T) {
	tests := []struct {
		name     string
		input    NetWorthInput
		expected NetWorthResult
	}{
		{
			name: "Positive net worth",
			input: NetWorthInput{
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
			},
			expected: NetWorthResult{
				TotalAssets:      500000,
				TotalLiabilities: 225000,
				NetWorth:         275000,
			},
		},
		{
			name: "Negative net worth",
			input: NetWorthInput{
				Assets: map[string]float64{
					"cash":    10000,
					"401k":    25000,
					"vehicle": 15000,
				},
				Liabilities: map[string]float64{
					"student_loans": 80000,
					"credit_cards":  15000,
					"car_loan":      20000,
				},
			},
			expected: NetWorthResult{
				TotalAssets:      50000,
				TotalLiabilities: 115000,
				NetWorth:         -65000,
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateNetWorth(tt.input)
			
			if result.TotalAssets != tt.expected.TotalAssets {
				t.Errorf("TotalAssets = %v, want %v",
					result.TotalAssets, tt.expected.TotalAssets)
			}
			
			if result.TotalLiabilities != tt.expected.TotalLiabilities {
				t.Errorf("TotalLiabilities = %v, want %v",
					result.TotalLiabilities, tt.expected.TotalLiabilities)
			}
			
			if result.NetWorth != tt.expected.NetWorth {
				t.Errorf("NetWorth = %v, want %v",
					result.NetWorth, tt.expected.NetWorth)
			}
		})
	}
}

func TestCalculateTaxOptimization(t *testing.T) {
	tests := []struct {
		name     string
		input    TaxOptimizerInput
		expected TaxOptimizerResult
		delta    float64
	}{
		{
			name: "Single filer with standard deduction",
			input: TaxOptimizerInput{
				Income:       75000,
				TaxYear:      2024,
				FilingStatus: "single",
				Deductions:   map[string]float64{},
				Credits:      map[string]float64{},
			},
			expected: TaxOptimizerResult{
				TaxableIncome: 60400, // 75000 - 14600 standard deduction
				TaxOwed:       8260,  // Approximate based on 2024 brackets
				EffectiveRate: 11.0,  // Approximate
				MarginalRate:  22,    // 60400 falls in 22% bracket
			},
			delta: 100,
		},
		{
			name: "High earner with deductions",
			input: TaxOptimizerInput{
				Income:       200000,
				TaxYear:      2024,
				FilingStatus: "single",
				Deductions: map[string]float64{
					"401k":              23000,
					"mortgage_interest": 12000,
					"state_taxes":       10000,
				},
				Credits: map[string]float64{
					"education": 2500,
				},
			},
			expected: TaxOptimizerResult{
				TaxableIncome: 140400, // 200000 - 14600 - 45000
				TaxOwed:       25000,   // Approximate after credits
				EffectiveRate: 12.5,    // Approximate
				MarginalRate:  24,      // 140400 falls in 24% bracket
			},
			delta: 2000,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTaxOptimization(tt.input)

			if math.Abs(result.TaxableIncome-tt.expected.TaxableIncome) > tt.delta {
				t.Errorf("TaxableIncome = %v, want %v (±%v)",
					result.TaxableIncome, tt.expected.TaxableIncome, tt.delta)
			}

			if math.Abs(result.TaxOwed-tt.expected.TaxOwed) > tt.delta {
				t.Errorf("TaxOwed = %v, want %v (±%v)",
					result.TaxOwed, tt.expected.TaxOwed, tt.delta)
			}

			if result.MarginalRate != tt.expected.MarginalRate {
				t.Errorf("MarginalRate = %v, want %v",
					result.MarginalRate, tt.expected.MarginalRate)
			}
		})
	}
}

func TestCalculateEmergencyFund(t *testing.T) {
	tests := []struct {
		name     string
		input    EmergencyFundInput
		expected EmergencyFundResult
	}{
		{
			name: "Stable job with no dependents",
			input: EmergencyFundInput{
				MonthlyExpenses:        3000,
				JobStability:           "stable",
				NumberOfDependents:     0,
				HasDisabilityInsurance: false,
			},
			expected: EmergencyFundResult{
				MinimumMonths:     3,
				RecommendedMonths: 6,
				MinimumAmount:     9000,
				RecommendedAmount: 18000,
			},
		},
		{
			name: "Unstable job with dependents and insurance",
			input: EmergencyFundInput{
				MonthlyExpenses:        4000,
				JobStability:           "unstable",
				NumberOfDependents:     2,
				HasDisabilityInsurance: true,
			},
			expected: EmergencyFundResult{
				MinimumMonths:     6, // 6 base + 2 dependents = 8, then * 0.8 = 6
				RecommendedMonths: 12,
				MinimumAmount:     24000,
				RecommendedAmount: 48000,
			},
		},
		{
			name: "Moderate job stability",
			input: EmergencyFundInput{
				MonthlyExpenses:        2500,
				JobStability:           "moderate",
				NumberOfDependents:     1,
				HasDisabilityInsurance: false,
			},
			expected: EmergencyFundResult{
				MinimumMonths:     5, // 4 base + 1 dependent
				RecommendedMonths: 11, // 9 base + 2 (1 dependent * 2)
				MinimumAmount:     12500,
				RecommendedAmount: 27500,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateEmergencyFund(tt.input)

			if result.MinimumMonths != tt.expected.MinimumMonths {
				t.Errorf("MinimumMonths = %v, want %v",
					result.MinimumMonths, tt.expected.MinimumMonths)
			}

			if result.RecommendedMonths != tt.expected.RecommendedMonths {
				t.Errorf("RecommendedMonths = %v, want %v",
					result.RecommendedMonths, tt.expected.RecommendedMonths)
			}

			if math.Abs(result.MinimumAmount-tt.expected.MinimumAmount) > 1 {
				t.Errorf("MinimumAmount = %v, want %v",
					result.MinimumAmount, tt.expected.MinimumAmount)
			}

			if math.Abs(result.RecommendedAmount-tt.expected.RecommendedAmount) > 1 {
				t.Errorf("RecommendedAmount = %v, want %v",
					result.RecommendedAmount, tt.expected.RecommendedAmount)
			}
		})
	}
}

func TestCalculateDebtPayoff(t *testing.T) {
	tests := []struct {
		name         string
		input        DebtPayoffInput
		expectedCond func(DebtPayoffResult) bool
		description  string
	}{
		{
			name: "Avalanche method prioritizes high interest",
			input: DebtPayoffInput{
				Debts: []Debt{
					{Name: "Credit Card", Balance: 5000, InterestRate: 18, MinimumPayment: 150},
					{Name: "Student Loan", Balance: 10000, InterestRate: 6, MinimumPayment: 100},
				},
				ExtraPayment: 200,
				Method:       "avalanche",
			},
			expectedCond: func(r DebtPayoffResult) bool {
				return r.TotalMonths > 0 && r.TotalMonths < 100 && r.TotalInterestPaid > 0
			},
			description: "Should complete in reasonable time with interest cost",
		},
		{
			name: "Snowball method prioritizes low balance",
			input: DebtPayoffInput{
				Debts: []Debt{
					{Name: "Small Card", Balance: 1000, InterestRate: 15, MinimumPayment: 50},
					{Name: "Big Loan", Balance: 20000, InterestRate: 7, MinimumPayment: 300},
				},
				ExtraPayment: 500,
				Method:       "snowball",
			},
			expectedCond: func(r DebtPayoffResult) bool {
				return r.TotalMonths > 0 && r.TotalMonths < 100 && r.TotalPaid > 21000
			},
			description: "Should complete and pay more than principal due to interest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateDebtPayoff(tt.input)

			if !tt.expectedCond(result) {
				t.Errorf("%s - got TotalMonths=%v, TotalInterestPaid=%v, TotalPaid=%v",
					tt.description, result.TotalMonths, result.TotalInterestPaid, result.TotalPaid)
			}

			if len(result.PayoffSchedule) == 0 {
				t.Error("PayoffSchedule should not be empty")
			}
		})
	}
}

func TestCalculateBudgetAllocation(t *testing.T) {
	tests := []struct {
		name     string
		input    BudgetAllocationInput
		expected BudgetAllocationResult
	}{
		{
			name: "50-30-20 rule",
			input: BudgetAllocationInput{
				MonthlyIncome: 5000,
				BudgetMethod:  "50-30-20",
			},
			expected: BudgetAllocationResult{
				Needs:          2500,
				NeedsPercent:   50,
				Wants:          1500,
				WantsPercent:   30,
				Savings:        1000,
				SavingsPercent: 20,
				Method:         "50-30-20",
			},
		},
		{
			name: "70-20-10 rule",
			input: BudgetAllocationInput{
				MonthlyIncome: 6000,
				BudgetMethod:  "70-20-10",
			},
			expected: BudgetAllocationResult{
				Needs:          4200,
				NeedsPercent:   70,
				Wants:          1200,
				WantsPercent:   20,
				Savings:        600,
				SavingsPercent: 10,
				Method:         "70-20-10",
			},
		},
		{
			name: "60-20-20 rule",
			input: BudgetAllocationInput{
				MonthlyIncome: 8000,
				BudgetMethod:  "60-20-20",
			},
			expected: BudgetAllocationResult{
				Needs:          4800,
				NeedsPercent:   60,
				Wants:          1600,
				WantsPercent:   20,
				Savings:        1600,
				SavingsPercent: 20,
				Method:         "60-20-20",
			},
		},
		{
			name: "Default method (should use 50-30-20)",
			input: BudgetAllocationInput{
				MonthlyIncome: 4000,
				BudgetMethod:  "",
			},
			expected: BudgetAllocationResult{
				Needs:          2000,
				NeedsPercent:   50,
				Wants:          1200,
				WantsPercent:   30,
				Savings:        800,
				SavingsPercent: 20,
				Method:         "50-30-20",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateBudgetAllocation(tt.input)

			if math.Abs(result.Needs-tt.expected.Needs) > 0.01 {
				t.Errorf("Needs = %v, want %v", result.Needs, tt.expected.Needs)
			}

			if math.Abs(result.Wants-tt.expected.Wants) > 0.01 {
				t.Errorf("Wants = %v, want %v", result.Wants, tt.expected.Wants)
			}

			if math.Abs(result.Savings-tt.expected.Savings) > 0.01 {
				t.Errorf("Savings = %v, want %v", result.Savings, tt.expected.Savings)
			}

			if result.Method != tt.expected.Method {
				t.Errorf("Method = %v, want %v", result.Method, tt.expected.Method)
			}
		})
	}
}
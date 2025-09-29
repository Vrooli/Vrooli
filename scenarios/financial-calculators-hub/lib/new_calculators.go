package lib

// Net Worth types and calculations
type NetWorthInput struct {
	Assets      map[string]float64 `json:"assets"`
	Liabilities map[string]float64 `json:"liabilities"`
	Notes       string            `json:"notes,omitempty"`
}

type NetWorthResult struct {
	TotalAssets      float64            `json:"total_assets"`
	TotalLiabilities float64            `json:"total_liabilities"`
	NetWorth         float64            `json:"net_worth"`
	AssetBreakdown   map[string]float64 `json:"asset_breakdown"`
	LiabilityBreakdown map[string]float64 `json:"liability_breakdown"`
	CalculationID    string             `json:"calculation_id,omitempty"`
}

func CalculateNetWorth(input NetWorthInput) NetWorthResult {
	totalAssets := 0.0
	for _, value := range input.Assets {
		totalAssets += value
	}
	
	totalLiabilities := 0.0
	for _, value := range input.Liabilities {
		totalLiabilities += value
	}
	
	// Add totals to the maps for storage
	input.Assets["total"] = totalAssets
	input.Liabilities["total"] = totalLiabilities
	
	return NetWorthResult{
		TotalAssets:      totalAssets,
		TotalLiabilities: totalLiabilities,
		NetWorth:         totalAssets - totalLiabilities,
		AssetBreakdown:   input.Assets,
		LiabilityBreakdown: input.Liabilities,
	}
}

// Tax Optimizer types and calculations
type TaxOptimizerInput struct {
	Income       float64            `json:"income"`
	TaxYear      int               `json:"tax_year"`
	FilingStatus string            `json:"filing_status"` // single, married_joint, married_separate, head_of_household
	Deductions   map[string]float64 `json:"deductions"`
	Credits      map[string]float64 `json:"credits"`
}

type TaxOptimizerResult struct {
	TaxableIncome    float64            `json:"taxable_income"`
	TaxOwed          float64            `json:"tax_owed"`
	EffectiveRate    float64            `json:"effective_rate"`
	MarginalRate     float64            `json:"marginal_rate"`
	TaxByBracket     []TaxBracketInfo   `json:"tax_by_bracket"`
	Recommendations  []string           `json:"recommendations"`
	PotentialSavings float64            `json:"potential_savings"`
	CalculationID    string             `json:"calculation_id,omitempty"`
}

type TaxBracketInfo struct {
	Bracket      string  `json:"bracket"`
	Rate         float64 `json:"rate"`
	IncomeRange  string  `json:"income_range"`
	TaxInBracket float64 `json:"tax_in_bracket"`
}

func CalculateTaxOptimization(input TaxOptimizerInput) TaxOptimizerResult {
	// 2024 tax brackets (simplified for demonstration)
	// These would normally be loaded from configuration
	standardDeduction := 14600.0 // Single filer for 2024
	if input.FilingStatus == "married_joint" {
		standardDeduction = 29200.0
	}
	
	// Calculate total deductions
	totalDeductions := standardDeduction
	for _, amount := range input.Deductions {
		totalDeductions += amount
	}
	
	// Taxable income
	taxableIncome := input.Income - totalDeductions
	if taxableIncome < 0 {
		taxableIncome = 0
	}
	
	// Calculate tax based on 2024 brackets (single filer example)
	tax := 0.0
	marginalRate := 0.0
	brackets := []TaxBracketInfo{}
	
	if input.FilingStatus == "single" {
		// 2024 single filer brackets
		if taxableIncome > 0 {
			// 10% bracket: $0 - $11,600
			bracketTax := 0.0
			if taxableIncome <= 11600 {
				bracketTax = taxableIncome * 0.10
				marginalRate = 10.0
			} else {
				bracketTax = 11600 * 0.10
			}
			tax += bracketTax
			brackets = append(brackets, TaxBracketInfo{
				Bracket: "10%",
				Rate: 10.0,
				IncomeRange: "$0 - $11,600",
				TaxInBracket: bracketTax,
			})
		}
		
		if taxableIncome > 11600 {
			// 12% bracket: $11,601 - $47,150
			bracketTax := 0.0
			if taxableIncome <= 47150 {
				bracketTax = (taxableIncome - 11600) * 0.12
				marginalRate = 12.0
			} else {
				bracketTax = (47150 - 11600) * 0.12
			}
			tax += bracketTax
			brackets = append(brackets, TaxBracketInfo{
				Bracket: "12%",
				Rate: 12.0,
				IncomeRange: "$11,601 - $47,150",
				TaxInBracket: bracketTax,
			})
		}
		
		if taxableIncome > 47150 {
			// 22% bracket: $47,151 - $100,525
			bracketTax := 0.0
			if taxableIncome <= 100525 {
				bracketTax = (taxableIncome - 47150) * 0.22
				marginalRate = 22.0
			} else {
				bracketTax = (100525 - 47150) * 0.22
			}
			tax += bracketTax
			brackets = append(brackets, TaxBracketInfo{
				Bracket: "22%",
				Rate: 22.0,
				IncomeRange: "$47,151 - $100,525",
				TaxInBracket: bracketTax,
			})
		}
		
		if taxableIncome > 100525 {
			// 24% bracket: $100,526 - $191,950
			bracketTax := 0.0
			if taxableIncome <= 191950 {
				bracketTax = (taxableIncome - 100525) * 0.24
				marginalRate = 24.0
			} else {
				bracketTax = (191950 - 100525) * 0.24
			}
			tax += bracketTax
			brackets = append(brackets, TaxBracketInfo{
				Bracket: "24%",
				Rate: 24.0,
				IncomeRange: "$100,526 - $191,950",
				TaxInBracket: bracketTax,
			})
		}
		
		if taxableIncome > 191950 {
			// 32% bracket: $191,951 - $243,725
			bracketTax := 0.0
			if taxableIncome <= 243725 {
				bracketTax = (taxableIncome - 191950) * 0.32
				marginalRate = 32.0
			} else {
				bracketTax = (243725 - 191950) * 0.32
			}
			tax += bracketTax
			brackets = append(brackets, TaxBracketInfo{
				Bracket: "32%",
				Rate: 32.0,
				IncomeRange: "$191,951 - $243,725",
				TaxInBracket: bracketTax,
			})
		}
		
		if taxableIncome > 243725 {
			// 35% bracket: $243,726 - $609,350
			bracketTax := 0.0
			if taxableIncome <= 609350 {
				bracketTax = (taxableIncome - 243725) * 0.35
				marginalRate = 35.0
			} else {
				bracketTax = (609350 - 243725) * 0.35
			}
			tax += bracketTax
			brackets = append(brackets, TaxBracketInfo{
				Bracket: "35%",
				Rate: 35.0,
				IncomeRange: "$243,726 - $609,350",
				TaxInBracket: bracketTax,
			})
		}
		
		if taxableIncome > 609350 {
			// 37% bracket: $609,351+
			bracketTax := (taxableIncome - 609350) * 0.37
			marginalRate = 37.0
			tax += bracketTax
			brackets = append(brackets, TaxBracketInfo{
				Bracket: "37%",
				Rate: 37.0,
				IncomeRange: "$609,351+",
				TaxInBracket: bracketTax,
			})
		}
	}
	
	// Apply credits
	totalCredits := 0.0
	for _, amount := range input.Credits {
		totalCredits += amount
	}
	tax -= totalCredits
	if tax < 0 {
		tax = 0
	}
	
	// Calculate effective rate
	effectiveRate := 0.0
	if input.Income > 0 {
		effectiveRate = (tax / input.Income) * 100
	}
	
	// Generate recommendations
	recommendations := []string{}
	potentialSavings := 0.0
	
	// Max out retirement contributions
	if _, hasRetirement := input.Deductions["401k"]; !hasRetirement || input.Deductions["401k"] < 23000 {
		maxContribution := 23000.0 // 2024 limit
		currentContribution := input.Deductions["401k"]
		additionalContribution := maxContribution - currentContribution
		savingsFromContribution := additionalContribution * (marginalRate / 100)
		recommendations = append(recommendations, "Consider maxing out 401(k) contributions ($23,000 limit) for tax savings")
		potentialSavings += savingsFromContribution
	}
	
	// Consider IRA contributions
	if _, hasIRA := input.Deductions["ira"]; !hasIRA || input.Deductions["ira"] < 7000 {
		recommendations = append(recommendations, "Consider contributing to a traditional IRA ($7,000 limit)")
		potentialSavings += (7000 - input.Deductions["ira"]) * (marginalRate / 100)
	}
	
	// HSA contributions if eligible
	if _, hasHSA := input.Deductions["hsa"]; !hasHSA {
		recommendations = append(recommendations, "If eligible, consider HSA contributions ($4,150 individual/$8,300 family)")
		potentialSavings += 4150 * (marginalRate / 100)
	}
	
	// Charitable deductions
	if _, hasCharity := input.Deductions["charity"]; !hasCharity {
		recommendations = append(recommendations, "Track and deduct charitable contributions")
	}
	
	// Tax-loss harvesting
	if marginalRate >= 24 {
		recommendations = append(recommendations, "Consider tax-loss harvesting in investment accounts")
	}
	
	return TaxOptimizerResult{
		TaxableIncome:    taxableIncome,
		TaxOwed:          tax,
		EffectiveRate:    effectiveRate,
		MarginalRate:     marginalRate,
		TaxByBracket:     brackets,
		Recommendations:  recommendations,
		PotentialSavings: potentialSavings,
	}
}

// Batch calculation types
type BatchCalculationInput struct {
	Calculations []map[string]interface{} `json:"calculations"`
}
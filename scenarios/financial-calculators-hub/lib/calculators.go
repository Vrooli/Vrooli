package lib

import (
	"math"
	"time"
)

type FIREInput struct {
	CurrentAge           float64 `json:"current_age"`
	CurrentSavings       float64 `json:"current_savings"`
	AnnualIncome         float64 `json:"annual_income"`
	AnnualExpenses       float64 `json:"annual_expenses"`
	SavingsRate          float64 `json:"savings_rate"`
	ExpectedReturn       float64 `json:"expected_return"`
	TargetWithdrawalRate float64 `json:"target_withdrawal_rate"`
}

type FIREResult struct {
	RetirementAge        float64            `json:"retirement_age"`
	YearsToRetirement    float64            `json:"years_to_retirement"`
	TargetNestEgg        float64            `json:"target_nest_egg"`
	ProjectedByAge       map[int]float64    `json:"projected_nest_egg_by_age"`
	MonthlySavingsReq    float64            `json:"monthly_savings_required"`
}

type CompoundInterestInput struct {
	Principal           float64 `json:"principal"`
	AnnualRate          float64 `json:"annual_rate"`
	Years               float64 `json:"years"`
	MonthlyContribution float64 `json:"monthly_contribution,omitempty"`
	CompoundFrequency   string  `json:"compound_frequency"`
}

type CompoundInterestResult struct {
	FinalAmount        float64              `json:"final_amount"`
	TotalContributions float64              `json:"total_contributions"`
	TotalInterest      float64              `json:"total_interest"`
	YearByYear         []YearlyProjection   `json:"year_by_year"`
}

type YearlyProjection struct {
	Year         int     `json:"year"`
	Balance      float64 `json:"balance"`
	Contribution float64 `json:"contribution"`
	Interest     float64 `json:"interest"`
}

type MortgageInput struct {
	LoanAmount         float64 `json:"loan_amount"`
	AnnualRate         float64 `json:"annual_rate"`
	Years              float64 `json:"years"`
	DownPayment        float64 `json:"down_payment,omitempty"`
	ExtraMonthlyPayment float64 `json:"extra_monthly_payment,omitempty"`
}

type MortgageResult struct {
	MonthlyPayment       float64              `json:"monthly_payment"`
	TotalInterest        float64              `json:"total_interest"`
	TotalPaid            float64              `json:"total_paid"`
	PayoffDate           string               `json:"payoff_date"`
	AmortizationSchedule []MonthlyPayment     `json:"amortization_schedule"`
}

type MonthlyPayment struct {
	Month            int     `json:"month"`
	Payment          float64 `json:"payment"`
	Principal        float64 `json:"principal"`
	Interest         float64 `json:"interest"`
	RemainingBalance float64 `json:"remaining_balance"`
}

type InflationInput struct {
	Amount      float64 `json:"amount"`
	Years       float64 `json:"years"`
	InflationRate float64 `json:"inflation_rate"`
}

type InflationResult struct {
	FutureValue      float64 `json:"future_value"`
	PurchasingPower  float64 `json:"purchasing_power"`
	TotalInflation   float64 `json:"total_inflation_percent"`
}

type EmergencyFundInput struct {
	MonthlyExpenses      float64 `json:"monthly_expenses"`
	JobStability         string  `json:"job_stability"` // stable, moderate, unstable
	NumberOfDependents   int     `json:"number_of_dependents"`
	HasDisabilityInsurance bool  `json:"has_disability_insurance"`
}

type EmergencyFundResult struct {
	MinimumMonths       int     `json:"minimum_months"`
	RecommendedMonths   int     `json:"recommended_months"`
	MinimumAmount       float64 `json:"minimum_amount"`
	RecommendedAmount   float64 `json:"recommended_amount"`
	CurrentShortfall    float64 `json:"current_shortfall,omitempty"`
}

type DebtPayoffInput struct {
	Debts              []Debt  `json:"debts"`
	ExtraPayment       float64 `json:"extra_monthly_payment"`
	Method             string  `json:"method"` // "avalanche" or "snowball"
}

type Debt struct {
	Name               string  `json:"name"`
	Balance            float64 `json:"balance"`
	MinimumPayment     float64 `json:"minimum_payment"`
	InterestRate       float64 `json:"interest_rate"`
}

type DebtPayoffResult struct {
	Method             string           `json:"method"`
	TotalMonths        int              `json:"total_months"`
	TotalInterestPaid  float64          `json:"total_interest_paid"`
	TotalPaid          float64          `json:"total_paid"`
	PayoffSchedule     []DebtSchedule   `json:"payoff_schedule"`
	SavingsVsMinimum   float64          `json:"savings_vs_minimum"`
}

type DebtSchedule struct {
	DebtName          string  `json:"debt_name"`
	PayoffMonth       int     `json:"payoff_month"`
	TotalPaid         float64 `json:"total_paid"`
	InterestPaid      float64 `json:"interest_paid"`
}

type BudgetAllocationInput struct {
	MonthlyIncome       float64 `json:"monthly_income"`
	BudgetMethod        string  `json:"budget_method"` // 50-30-20, 70-20-10, zero-based
}

type BudgetAllocationResult struct {
	Needs               float64 `json:"needs_amount"`
	NeedsPercent        float64 `json:"needs_percent"`
	Wants               float64 `json:"wants_amount"`
	WantsPercent        float64 `json:"wants_percent"`
	Savings             float64 `json:"savings_amount"`
	SavingsPercent      float64 `json:"savings_percent"`
	Method              string  `json:"method"`
}

func CalculateFIRE(input FIREInput) FIREResult {
	if input.TargetWithdrawalRate == 0 {
		input.TargetWithdrawalRate = 4.0
	}
	
	targetNestEgg := input.AnnualExpenses * (100 / input.TargetWithdrawalRate)
	monthlySavings := input.AnnualIncome * (input.SavingsRate / 100) / 12
	monthlyReturn := input.ExpectedReturn / 100 / 12
	
	currentBalance := input.CurrentSavings
	months := 0
	projectedByAge := make(map[int]float64)
	
	for currentBalance < targetNestEgg && months < 600 {
		if months%12 == 0 {
			age := int(input.CurrentAge) + months/12
			projectedByAge[age] = currentBalance
		}
		
		currentBalance = currentBalance*(1+monthlyReturn) + monthlySavings
		months++
	}
	
	yearsToRetirement := float64(months) / 12
	retirementAge := input.CurrentAge + yearsToRetirement
	
	finalAge := int(retirementAge)
	if _, exists := projectedByAge[finalAge]; !exists {
		projectedByAge[finalAge] = currentBalance
	}
	
	return FIREResult{
		RetirementAge:     retirementAge,
		YearsToRetirement: yearsToRetirement,
		TargetNestEgg:     targetNestEgg,
		ProjectedByAge:    projectedByAge,
		MonthlySavingsReq: monthlySavings,
	}
}

func CalculateCompoundInterest(input CompoundInterestInput) CompoundInterestResult {
	var compoundsPerYear float64
	switch input.CompoundFrequency {
	case "monthly":
		compoundsPerYear = 12
	case "quarterly":
		compoundsPerYear = 4
	case "annually":
		compoundsPerYear = 1
	default:
		compoundsPerYear = 12
	}
	
	rate := input.AnnualRate / 100
	balance := input.Principal
	totalContributions := input.Principal
	yearByYear := []YearlyProjection{}
	
	for year := 1; year <= int(input.Years); year++ {
		yearStartBalance := balance
		yearContribution := input.MonthlyContribution * 12
		
		for month := 1; month <= 12; month++ {
			balance += input.MonthlyContribution
			if month%int(12/compoundsPerYear) == 0 {
				interest := balance * (rate / compoundsPerYear)
				balance += interest
			}
		}
		
		yearInterest := balance - yearStartBalance - yearContribution
		totalContributions += yearContribution
		
		yearByYear = append(yearByYear, YearlyProjection{
			Year:         year,
			Balance:      balance,
			Contribution: yearContribution,
			Interest:     yearInterest,
		})
	}
	
	totalInterest := balance - totalContributions
	
	return CompoundInterestResult{
		FinalAmount:        balance,
		TotalContributions: totalContributions,
		TotalInterest:      totalInterest,
		YearByYear:         yearByYear,
	}
}

func CalculateMortgage(input MortgageInput) MortgageResult {
	principal := input.LoanAmount - input.DownPayment
	monthlyRate := input.AnnualRate / 100 / 12
	numPayments := input.Years * 12
	
	monthlyPayment := principal * (monthlyRate * math.Pow(1+monthlyRate, numPayments)) / 
		(math.Pow(1+monthlyRate, numPayments) - 1)
	
	monthlyPayment += input.ExtraMonthlyPayment
	
	remainingBalance := principal
	totalInterest := 0.0
	schedule := []MonthlyPayment{}
	month := 0
	
	for remainingBalance > 0.01 && month < int(numPayments)*2 {
		month++
		interestPayment := remainingBalance * monthlyRate
		principalPayment := monthlyPayment - interestPayment
		
		if principalPayment > remainingBalance {
			principalPayment = remainingBalance
		}
		
		remainingBalance -= principalPayment
		totalInterest += interestPayment
		
		schedule = append(schedule, MonthlyPayment{
			Month:            month,
			Payment:          principalPayment + interestPayment,
			Principal:        principalPayment,
			Interest:         interestPayment,
			RemainingBalance: remainingBalance,
		})
		
		if remainingBalance <= 0.01 {
			break
		}
	}
	
	payoffDate := time.Now().AddDate(0, month, 0).Format("2006-01-02")
	totalPaid := principal + totalInterest
	
	return MortgageResult{
		MonthlyPayment:       monthlyPayment,
		TotalInterest:        totalInterest,
		TotalPaid:            totalPaid,
		PayoffDate:           payoffDate,
		AmortizationSchedule: schedule,
	}
}

func CalculateInflation(input InflationInput) InflationResult {
	rate := input.InflationRate / 100
	futureValue := input.Amount * math.Pow(1+rate, input.Years)
	purchasingPower := input.Amount / math.Pow(1+rate, input.Years)
	totalInflation := (futureValue/input.Amount - 1) * 100
	
	return InflationResult{
		FutureValue:     futureValue,
		PurchasingPower: purchasingPower,
		TotalInflation:  totalInflation,
	}
}

func CalculateEmergencyFund(input EmergencyFundInput) EmergencyFundResult {
	// Base months needed based on job stability
	minMonths := 3
	recMonths := 6
	
	switch input.JobStability {
	case "unstable":
		minMonths = 6
		recMonths = 12
	case "moderate":
		minMonths = 4
		recMonths = 9
	case "stable":
		minMonths = 3
		recMonths = 6
	}
	
	// Add months for dependents
	if input.NumberOfDependents > 0 {
		minMonths += input.NumberOfDependents
		recMonths += input.NumberOfDependents * 2
	}
	
	// Reduce if has disability insurance
	if input.HasDisabilityInsurance {
		minMonths = int(float64(minMonths) * 0.8)
		recMonths = int(float64(recMonths) * 0.8)
	}
	
	minAmount := float64(minMonths) * input.MonthlyExpenses
	recAmount := float64(recMonths) * input.MonthlyExpenses
	
	return EmergencyFundResult{
		MinimumMonths:     minMonths,
		RecommendedMonths: recMonths,
		MinimumAmount:     minAmount,
		RecommendedAmount: recAmount,
	}
}

func CalculateDebtPayoff(input DebtPayoffInput) DebtPayoffResult {
	// Sort debts based on method
	debts := make([]Debt, len(input.Debts))
	copy(debts, input.Debts)
	
	if input.Method == "avalanche" {
		// Sort by interest rate (highest first)
		for i := 0; i < len(debts)-1; i++ {
			for j := i + 1; j < len(debts); j++ {
				if debts[j].InterestRate > debts[i].InterestRate {
					debts[i], debts[j] = debts[j], debts[i]
				}
			}
		}
	} else { // snowball - sort by balance (lowest first)
		for i := 0; i < len(debts)-1; i++ {
			for j := i + 1; j < len(debts); j++ {
				if debts[j].Balance < debts[i].Balance {
					debts[i], debts[j] = debts[j], debts[i]
				}
			}
		}
	}
	
	// Calculate payoff schedule
	totalMonths := 0
	totalInterest := 0.0
	totalPaid := 0.0
	schedule := []DebtSchedule{}
	extraPayment := input.ExtraPayment
	
	// Working copies of debt balances
	workingDebts := make([]Debt, len(debts))
	copy(workingDebts, debts)
	
	// Simple simulation - pay minimum on all, extra on targeted debt
	for len(workingDebts) > 0 {
		totalMonths++
		monthlyPayment := 0.0
		
		// Pay debts in order
		for i := 0; i < len(workingDebts); i++ {
			monthlyRate := workingDebts[i].InterestRate / 100 / 12
			interest := workingDebts[i].Balance * monthlyRate
			
			payment := workingDebts[i].MinimumPayment
			if i == 0 && extraPayment > 0 {
				payment += extraPayment
			}
			
			if payment > workingDebts[i].Balance + interest {
				payment = workingDebts[i].Balance + interest
			}
			
			principal := payment - interest
			workingDebts[i].Balance -= principal
			monthlyPayment += payment
			totalInterest += interest
			
			// Check if debt is paid off
			if workingDebts[i].Balance <= 0.01 {
				schedule = append(schedule, DebtSchedule{
					DebtName:     workingDebts[i].Name,
					PayoffMonth:  totalMonths,
					TotalPaid:    payment,
					InterestPaid: interest,
				})
				
				// Roll extra payment to next debt
				extraPayment += workingDebts[i].MinimumPayment
				
				// Remove paid debt
				workingDebts = append(workingDebts[:i], workingDebts[i+1:]...)
				i--
			}
		}
		
		totalPaid += monthlyPayment
		
		// Safety check - prevent infinite loop
		if totalMonths > 360 {
			break
		}
	}
	
	return DebtPayoffResult{
		Method:            input.Method,
		TotalMonths:       totalMonths,
		TotalInterestPaid: totalInterest,
		TotalPaid:         totalPaid,
		PayoffSchedule:    schedule,
		SavingsVsMinimum:  0, // Would need to calculate minimum payment scenario
	}
}

func CalculateBudgetAllocation(input BudgetAllocationInput) BudgetAllocationResult {
	var needsPercent, wantsPercent, savingsPercent float64
	
	switch input.BudgetMethod {
	case "50-30-20":
		needsPercent = 50
		wantsPercent = 30
		savingsPercent = 20
	case "70-20-10":
		needsPercent = 70
		wantsPercent = 20
		savingsPercent = 10
	case "60-20-20":
		needsPercent = 60
		wantsPercent = 20
		savingsPercent = 20
	default:
		// Default to 50-30-20
		needsPercent = 50
		wantsPercent = 30
		savingsPercent = 20
		input.BudgetMethod = "50-30-20"
	}
	
	needsAmount := input.MonthlyIncome * (needsPercent / 100)
	wantsAmount := input.MonthlyIncome * (wantsPercent / 100)
	savingsAmount := input.MonthlyIncome * (savingsPercent / 100)
	
	return BudgetAllocationResult{
		Needs:          needsAmount,
		NeedsPercent:   needsPercent,
		Wants:          wantsAmount,
		WantsPercent:   wantsPercent,
		Savings:        savingsAmount,
		SavingsPercent: savingsPercent,
		Method:         input.BudgetMethod,
	}
}
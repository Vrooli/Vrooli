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
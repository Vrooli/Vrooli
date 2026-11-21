// Package cost contains deterministic, evidence-preserving cost calculations.
package cost

import (
	"context"
	"errors"
	"fmt"
	"time"

	"nutrition-planner/internal/decimalx"
	"nutrition-planner/internal/money"
)

type Requirement struct {
	ItemID string
	Amount decimalx.Decimal
	Unit   string
}

type Stock struct {
	Amount decimalx.Decimal
	Unit   string
}

type Package struct {
	ID       string
	ItemID   string
	Amount   decimalx.Decimal
	Unit     string
	Price    money.Money
	Observed string
}

type Observation struct {
	ID                 string
	WorkspaceID        string
	ItemID             string
	ProductID          string
	PackageLabel       string
	PackageAmount      decimalx.Decimal
	PackageUnit        string
	Price              money.Money
	Retailer           string
	ObservedAt         time.Time
	ValidThrough       time.Time
	Available          bool
	MembershipRequired bool
	CouponRequired     bool
	MinimumBuy         string
	Source             string
}

type Repository interface {
	Create(context.Context, Observation) (Observation, error)
	List(context.Context, string, string) ([]Observation, error)
}

type ErrInvalid struct{ Field, Reason string }

func (e ErrInvalid) Error() string { return e.Field + ": " + e.Reason }

func ValidateObservation(v Observation) error {
	if v.WorkspaceID == "" || v.ItemID == "" {
		return ErrInvalid{"identity", "workspace and item are required"}
	}
	if v.PackageAmount.IsUnknown() || v.PackageAmount.IsZero() || v.PackageUnit == "" {
		return ErrInvalid{"package", "known non-zero amount and unit are required"}
	}
	if !v.Price.Known || v.Price.Currency == "" {
		return ErrInvalid{"price", "known price and currency are required"}
	}
	if v.ObservedAt.IsZero() {
		return ErrInvalid{"observed_at", "is required"}
	}
	if v.Source == "" {
		return ErrInvalid{"source", "is required"}
	}
	return nil
}

type ItemResult struct {
	ItemID             string
	Required           decimalx.Decimal
	Available          decimalx.Decimal
	Missing            decimalx.Decimal
	PackagesToBuy      decimalx.Decimal
	AllocatedMinorCost decimalx.Decimal
	CheckoutMinorCost  decimalx.Decimal
	Currency           string
	PriceKnown         bool
	Complete           bool
	Reason             string
}

type Result struct {
	Items              []ItemResult
	AllocatedMinorCost decimalx.Decimal
	CheckoutMinorCost  decimalx.Decimal
	Currency           string
	Complete           bool
	Unknown            []string
}

// Calculate keeps portion allocation and checkout as separate values. Monetary
// results are expressed in exact currency minor units, so fractional cents are
// retained until a caller chooses a display/settlement policy.
func Calculate(requirement Requirement, stock Stock, offers []Package) (ItemResult, error) {
	if requirement.ItemID == "" || requirement.Amount.IsUnknown() || requirement.Amount.IsZero() {
		return ItemResult{}, errors.New("requirement must have a known non-zero amount and item")
	}
	if len(offers) == 0 {
		return ItemResult{ItemID: requirement.ItemID, Required: requirement.Amount, Available: stock.Amount, Missing: decimalx.Unknown, PackagesToBuy: decimalx.Unknown, AllocatedMinorCost: decimalx.Unknown, CheckoutMinorCost: decimalx.Unknown, Complete: false, Reason: "no compatible price offer"}, nil
	}
	offer := offers[0]
	for _, candidate := range offers[1:] {
		if candidate.Price.Known && (!offer.Price.Known || candidate.Price.Minor < offer.Price.Minor) {
			offer = candidate
		}
	}
	if offer.ItemID != requirement.ItemID || offer.Amount.IsUnknown() || offer.Amount.IsZero() || offer.Unit != requirement.Unit {
		return ItemResult{}, fmt.Errorf("offer is incompatible with requirement")
	}
	if stock.Unit != "" && stock.Unit != requirement.Unit {
		return ItemResult{}, fmt.Errorf("stock unit is incompatible with requirement")
	}
	allocated := decimalx.Unknown
	if offer.Price.Known {
		priceTimesAmount, err := decimalx.Mul(decimalx.KnownInt(offer.Price.Minor), requirement.Amount)
		if err != nil {
			return ItemResult{}, err
		}
		allocated, err = decimalx.Div(priceTimesAmount, offer.Amount)
		if err != nil {
			return ItemResult{}, err
		}
	}
	available := stock.Amount
	if available.IsUnknown() {
		return ItemResult{ItemID: requirement.ItemID, Required: requirement.Amount, Available: available, Missing: decimalx.Unknown, PackagesToBuy: decimalx.Unknown, AllocatedMinorCost: allocated, CheckoutMinorCost: decimalx.Unknown, Currency: offer.Price.Currency, PriceKnown: offer.Price.Known, Complete: false, Reason: "available stock is unknown"}, nil
	}
	missing := decimalx.KnownInt(0)
	if comparison, err := decimalx.Compare(requirement.Amount, available); err != nil {
		return ItemResult{}, err
	} else if comparison > 0 {
		missing, err = decimalx.Sub(requirement.Amount, available)
		if err != nil {
			return ItemResult{}, err
		}
	}
	packages := decimalx.KnownInt(0)
	checkout := decimalx.KnownInt(0)
	if !missing.IsZero() {
		quotient, err := decimalx.Div(missing, offer.Amount)
		if err != nil {
			return ItemResult{}, err
		}
		packages, err = decimalx.Ceil(quotient)
		if err != nil {
			return ItemResult{}, err
		}
		if offer.Price.Known {
			checkout, err = decimalx.Mul(decimalx.KnownInt(offer.Price.Minor), packages)
			if err != nil {
				return ItemResult{}, err
			}
		} else {
			checkout = decimalx.Unknown
		}
	}
	if !offer.Price.Known {
		allocated = decimalx.Unknown
	}
	return ItemResult{ItemID: requirement.ItemID, Required: requirement.Amount, Available: available, Missing: missing, PackagesToBuy: packages, AllocatedMinorCost: allocated, CheckoutMinorCost: checkout, Currency: offer.Price.Currency, PriceKnown: offer.Price.Known, Complete: offer.Price.Known, Reason: "selected lowest known compatible offer"}, nil
}

// CalculateMany sums only known values and preserves an explicit list of
// unresolved items. Unknown price or stock never becomes zero.
func CalculateMany(requirements []Requirement, stock map[string]Stock, offers map[string][]Package) (Result, error) {
	result := Result{AllocatedMinorCost: decimalx.KnownInt(0), CheckoutMinorCost: decimalx.KnownInt(0), Complete: true}
	for _, requirement := range requirements {
		item, err := Calculate(requirement, stock[requirement.ItemID], offers[requirement.ItemID])
		if err != nil {
			return Result{}, err
		}
		result.Items = append(result.Items, item)
		if item.Currency != "" && result.Currency == "" {
			result.Currency = item.Currency
		}
		if !item.AllocatedMinorCost.IsUnknown() {
			result.AllocatedMinorCost, _ = decimalx.Add(result.AllocatedMinorCost, item.AllocatedMinorCost)
		}
		if !item.CheckoutMinorCost.IsUnknown() {
			result.CheckoutMinorCost, _ = decimalx.Add(result.CheckoutMinorCost, item.CheckoutMinorCost)
		}
		if item.AllocatedMinorCost.IsUnknown() || item.CheckoutMinorCost.IsUnknown() || item.Available.IsUnknown() || !item.PriceKnown {
			result.Complete = false
			result.Unknown = append(result.Unknown, item.ItemID)
		}
	}
	return result, nil
}

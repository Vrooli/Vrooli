// Package providers contains optional, server-side adapter contracts. A
// missing provider is a supported capability state; deterministic Daily
// behavior never depends on an implementation in this package.
package providers

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrNotConfigured = errors.New("provider is not configured")

type (
	Status                struct{ Name, State, Reason string }
	NutritionQuery        struct{ FoodID, Query, Preparation, DataType string }
	NutritionSearchQuery  struct{ Query, Preparation, DataType string }
	NutritionSearchResult struct {
		SourceID, SourceVersion, ObservedAt       string
		ConceptID, Name, ProductName, Preparation string
		DataType                                  string
	}
)
type NutritionResult struct {
	SourceID, SourceVersion, ObservedAt string
	Values                              map[string]string
	Evidence                            []string
}
type (
	PriceQuery  struct{ ItemID, Retailer, Currency string }
	PriceResult struct {
		ItemID, Retailer, Currency, Amount, ObservedAt string
		Conditions                                     []string
	}
)
type (
	ReceiptInput  struct{ SourceID, Text string }
	ReceiptResult struct {
		SourceID, TransactionID string
		Lines                   []ReceiptLine
		Warnings                []string
	}
)
type ReceiptLine struct{ Description, ItemID, Amount, Unit, Price string }

func ReceiptDedupKey(result ReceiptResult, line ReceiptLine) string {
	return strings.Join([]string{result.SourceID, result.TransactionID, NormalizeReceiptLine(line)}, "|")
}

func NormalizeReceiptLine(line ReceiptLine) string {
	return strings.Join([]string{strings.ToLower(strings.TrimSpace(line.ItemID)), strings.ToLower(strings.TrimSpace(line.Description)), strings.TrimSpace(line.Amount), strings.ToLower(strings.TrimSpace(line.Unit)), strings.TrimSpace(line.Price)}, "\x1f")
}

func ValidateReceiptIdentity(result ReceiptResult) error {
	if strings.TrimSpace(result.SourceID) == "" || strings.TrimSpace(result.TransactionID) == "" {
		return fmt.Errorf("receipt source and transaction identity are required")
	}
	return nil
}

type Nutrition interface {
	Lookup(context.Context, NutritionQuery) (NutritionResult, error)
}

type NutritionCatalog interface {
	Search(context.Context, NutritionSearchQuery) ([]NutritionSearchResult, error)
	Detail(context.Context, NutritionQuery) (NutritionResult, error)
}
type Price interface {
	Observe(context.Context, PriceQuery) (PriceResult, error)
}
type Receipt interface {
	Parse(context.Context, ReceiptInput) (ReceiptResult, error)
}

type Disabled struct{ Name string }

func (d Disabled) Status() Status {
	return Status{Name: d.Name, State: "not_configured", Reason: "manual entry and deterministic planning remain available"}
}

func (d Disabled) Lookup(context.Context, NutritionQuery) (NutritionResult, error) {
	return NutritionResult{}, ErrNotConfigured
}

func (d Disabled) Search(context.Context, NutritionSearchQuery) ([]NutritionSearchResult, error) {
	return nil, ErrNotConfigured
}

func (d Disabled) Detail(ctx context.Context, query NutritionQuery) (NutritionResult, error) {
	return d.Lookup(ctx, query)
}

func (d Disabled) Observe(context.Context, PriceQuery) (PriceResult, error) {
	return PriceResult{}, ErrNotConfigured
}

func (d Disabled) Parse(context.Context, ReceiptInput) (ReceiptResult, error) {
	return ReceiptResult{}, ErrNotConfigured
}

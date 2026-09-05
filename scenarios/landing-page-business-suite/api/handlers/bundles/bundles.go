// Package bundles owns HTTP transport for editable bundle catalog endpoints.
package bundles

import (
	"context"
	"net/http"
	"strings"

	"landing-page-business-suite-api/internal/commerce"
)

type UpdatePriceRequest struct {
	StripePriceID  *string   `json:"stripe_price_id"`
	PlanName       *string   `json:"plan_name"`
	DisplayWeight  *int      `json:"display_weight"`
	DisplayEnabled *bool     `json:"display_enabled"`
	Subtitle       *string   `json:"subtitle"`
	Badge          *string   `json:"badge"`
	CtaLabel       *string   `json:"cta_label"`
	Highlight      *bool     `json:"highlight"`
	Features       *[]string `json:"features"`
}

type Dependencies struct {
	Catalog                   func(context.Context) (any, error)
	ActiveKey                 func() string
	Update                    func(context.Context, string, string, UpdatePriceRequest) (any, error)
	Path                      func(*http.Request, string) (string, bool)
	DecodeJSON                func(http.ResponseWriter, *http.Request, any) bool
	WriteError                func(http.ResponseWriter, int, string, string)
	WriteSuccess              func(http.ResponseWriter, any)
	ClassifyError             func(error) (int, string, string, bool)
	Query                     func(*http.Request, string) string
	VerifyPrice               func(string) (any, error)
	PreviewImport             func(context.Context) (any, error)
	PreviewUnavailableMessage string
	DeletePrice               func(string) error
	WriteSuccessMessage       func(http.ResponseWriter, string)
	ImportPrices              func(context.Context, StripeImportRequest) (any, error)
	ClassifyImportError       func(error) (int, string)
	CreatePrice               func(context.Context, string, commerce.CreateBundlePriceInput) (any, error)
}

type CreatePriceRequest struct {
	StripePriceID          string   `json:"stripe_price_id"`
	PlanName               string   `json:"plan_name"`
	PlanTier               string   `json:"plan_tier"`
	BillingInterval        string   `json:"billing_interval"`
	AmountCents            *int64   `json:"amount_cents"`
	Currency               *string  `json:"currency"`
	DisplayWeight          *int32   `json:"display_weight"`
	DisplayEnabled         *bool    `json:"display_enabled"`
	MonthlyIncludedCredits *int64   `json:"monthly_included_credits"`
	Subtitle               *string  `json:"subtitle"`
	Badge                  *string  `json:"badge"`
	CtaLabel               *string  `json:"cta_label"`
	Highlight              *bool    `json:"highlight"`
	Features               []string `json:"features"`
}

func (r CreatePriceRequest) input() commerce.CreateBundlePriceInput {
	return commerce.CreateBundlePriceInput{StripePriceID: r.StripePriceID, PlanName: r.PlanName, PlanTier: r.PlanTier, BillingInterval: r.BillingInterval, AmountCents: r.AmountCents, Currency: r.Currency, DisplayWeight: r.DisplayWeight, DisplayEnabled: r.DisplayEnabled, MonthlyIncludedCredits: r.MonthlyIncludedCredits, Subtitle: r.Subtitle, Badge: r.Badge, CtaLabel: r.CtaLabel, Highlight: r.Highlight, Features: r.Features}
}

type StripeImportRequest struct {
	BundleProductID string                         `json:"bundle_product_id"`
	Mode            commerce.StripeImportMode      `json:"mode,omitempty"`
	Selections      []commerce.ImportPlanSelection `json:"selections"`
}

// DeletePrice removes a price from the active bundle.
func DeletePrice(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bundleKey, ok := deps.Path(r, "bundle_key")
		if !ok || bundleKey == "" {
			deps.WriteError(w, http.StatusBadRequest, "Bundle key is required", "validation")
			return
		}
		if bundleKey != deps.ActiveKey() {
			deps.WriteError(w, http.StatusBadRequest, "Bundle key does not match active bundle", "validation")
			return
		}
		priceID, ok := deps.Path(r, "price_id")
		if !ok || priceID == "" {
			deps.WriteError(w, http.StatusBadRequest, "Price id is required", "validation")
			return
		}
		if deps.DeletePrice == nil {
			deps.WriteError(w, http.StatusInternalServerError, "plan store not available", "server_error")
			return
		}
		if err := deps.DeletePrice(priceID); err != nil {
			deps.WriteError(w, http.StatusNotFound, err.Error(), "not_found")
			return
		}
		deps.WriteSuccessMessage(w, "Plan deleted successfully")
	}
}

func ImportStripePrices(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request StripeImportRequest
		if !deps.DecodeJSON(w, r, &request) {
			return
		}
		request.BundleProductID = strings.TrimSpace(request.BundleProductID)
		if request.BundleProductID == "" {
			deps.WriteError(w, http.StatusBadRequest, "bundle_product_id is required", "validation")
			return
		}
		if strings.TrimSpace(string(request.Mode)) == "" {
			request.Mode = commerce.StripeImportModeMerge
		}
		result, err := deps.ImportPrices(r.Context(), request)
		if err != nil {
			status, kind := deps.ClassifyImportError(err)
			deps.WriteError(w, status, err.Error(), kind)
			return
		}
		deps.WriteSuccess(w, result)
	}
}

func CreatePrice(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bundleKey, ok := deps.Path(r, "bundle_key")
		if !ok || bundleKey == "" {
			deps.WriteError(w, http.StatusBadRequest, "Bundle key is required", "validation")
			return
		}
		if bundleKey != deps.ActiveKey() {
			deps.WriteError(w, http.StatusBadRequest, "Bundle key does not match active bundle", "validation")
			return
		}
		var request CreatePriceRequest
		if !deps.DecodeJSON(w, r, &request) {
			return
		}
		if deps.CreatePrice == nil {
			deps.WriteError(w, http.StatusInternalServerError, "plan store not available", "server_error")
			return
		}
		result, err := deps.CreatePrice(r.Context(), bundleKey, request.input())
		if err != nil {
			deps.WriteError(w, http.StatusBadRequest, err.Error(), "validation")
			return
		}
		deps.WriteSuccess(w, result)
	}
}

func Catalog(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		catalog, err := deps.Catalog(r.Context())
		if err != nil {
			deps.WriteError(w, http.StatusInternalServerError, "Failed to load bundle catalog", "server_error")
			return
		}
		deps.WriteSuccess(w, catalog)
	}
}

func UpdatePrice(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bundleKey, ok := deps.Path(r, "bundle_key")
		if !ok || bundleKey == "" {
			deps.WriteError(w, http.StatusBadRequest, "Bundle key is required", "validation")
			return
		}
		if bundleKey != deps.ActiveKey() {
			deps.WriteError(w, http.StatusBadRequest, "Bundle key does not match active bundle", "validation")
			return
		}
		priceID, ok := deps.Path(r, "price_id")
		if !ok || priceID == "" {
			deps.WriteError(w, http.StatusBadRequest, "Price id is required", "validation")
			return
		}
		var request UpdatePriceRequest
		if !deps.DecodeJSON(w, r, &request) {
			return
		}
		price, err := deps.Update(r.Context(), bundleKey, priceID, request)
		if err != nil {
			if status, kind, message, ok := deps.ClassifyError(err); ok {
				deps.WriteError(w, status, message, kind)
				return
			}
			deps.WriteError(w, http.StatusBadRequest, err.Error(), "validation")
			return
		}
		deps.WriteSuccess(w, price)
	}
}

// VerifyStripePrice verifies a Stripe price ID or lookup key before plan edits.
func VerifyStripePrice(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.VerifyPrice == nil {
			deps.WriteError(w, http.StatusServiceUnavailable, "Stripe service unavailable", "server_error")
			return
		}
		key := deps.Query(r, "key")
		if key == "" {
			deps.WriteError(w, http.StatusBadRequest, "price key required", "validation")
			return
		}
		info, err := deps.VerifyPrice(key)
		if err != nil {
			if status, kind, message, ok := deps.ClassifyError(err); ok {
				deps.WriteError(w, status, message, kind)
				return
			}
			deps.WriteError(w, http.StatusBadRequest, err.Error(), "validation")
			return
		}
		deps.WriteSuccess(w, info)
	}
}

// PreviewStripeImport lists Stripe products and prices available for import.
func PreviewStripeImport(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.PreviewImport == nil {
			if deps.PreviewUnavailableMessage != "" {
				deps.WriteError(w, http.StatusInternalServerError, deps.PreviewUnavailableMessage, "server_error")
				return
			}
			deps.WriteError(w, http.StatusServiceUnavailable, "Stripe service unavailable", "server_error")
			return
		}
		preview, err := deps.PreviewImport(r.Context())
		if err != nil {
			if status, kind, message, ok := deps.ClassifyError(err); ok {
				deps.WriteError(w, status, message, kind)
				return
			}
			deps.WriteError(w, http.StatusInternalServerError, "Failed to load Stripe products", "server_error")
			return
		}
		deps.WriteSuccess(w, preview)
	}
}

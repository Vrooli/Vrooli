// Package bundles owns HTTP transport for editable bundle catalog endpoints.
package bundles

import (
	"context"
	"net/http"
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
	Catalog       func(context.Context) (any, error)
	ActiveKey     func() string
	Update        func(context.Context, string, string, UpdatePriceRequest) (any, error)
	Path          func(*http.Request, string) (string, bool)
	DecodeJSON    func(http.ResponseWriter, *http.Request, any) bool
	WriteError    func(http.ResponseWriter, int, string, string)
	WriteSuccess  func(http.ResponseWriter, any)
	ClassifyError func(error) (int, string, string, bool)
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

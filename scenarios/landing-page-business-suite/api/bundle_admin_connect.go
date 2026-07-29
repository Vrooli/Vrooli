package main

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	bundlehttp "landing-page-business-suite-api/handlers/bundles"
)

func bundleConnectDependencies(planService *PlanService, stripe *StripeService) bundlehttp.ConnectDependencies {
	return bundlehttp.ConnectDependencies{
		ListCatalog: func(ctx context.Context) ([]*lpbsv1.BundleCatalogEntry, error) {
			catalog, err := planService.ListBundleCatalog(ctx)
			if err != nil {
				return nil, err
			}
			result := make([]*lpbsv1.BundleCatalogEntry, 0, len(catalog))
			for _, entry := range catalog {
				result = append(result, &lpbsv1.BundleCatalogEntry{Bundle: entry.Bundle, Prices: entry.Prices})
			}
			return result, nil
		},
		UpdatePrice: func(ctx context.Context, bundleKey, priceID string, input bundlehttp.UpdatePriceInput) (*PlanOption, error) {
			var fetcher StripePriceFetcher
			if stripe != nil {
				fetcher = stripe.FetchStripePriceDetails
			}
			return planService.UpdateBundlePriceWithStripe(ctx, bundleKey, priceID, UpdateBundlePriceInput{
				StripePriceID: input.StripePriceID, PlanName: input.PlanName, DisplayWeight: input.DisplayWeight,
				DisplayEnabled: input.DisplayEnabled, Subtitle: input.Subtitle, Badge: input.Badge,
				CtaLabel: input.CtaLabel, Highlight: input.Highlight, Features: input.Features,
			}, fetcher)
		},
		Classify: classifyBundleConnectError,
	}
}

func classifyBundleConnectError(err error) connect.Code {
	var stripeErr *StripeAPIError
	if errors.As(err, &stripeErr) {
		switch stripeErr.Status {
		case http.StatusNotFound:
			return connect.CodeNotFound
		case http.StatusBadGateway, http.StatusServiceUnavailable:
			return connect.CodeUnavailable
		case http.StatusUnauthorized:
			return connect.CodeUnauthenticated
		case http.StatusForbidden:
			return connect.CodePermissionDenied
		default:
			return connect.CodeInvalidArgument
		}
	}
	if status, _, _, ok := classifyStripeError(err); ok {
		switch status {
		case http.StatusNotFound:
			return connect.CodeNotFound
		case http.StatusBadGateway:
			return connect.CodeUnavailable
		case http.StatusServiceUnavailable:
			return connect.CodeUnavailable
		case http.StatusUnauthorized:
			return connect.CodeUnauthenticated
		case http.StatusForbidden:
			return connect.CodePermissionDenied
		default:
			return connect.CodeInvalidArgument
		}
	}
	if strings.Contains(strings.ToLower(err.Error()), "not found") {
		return connect.CodeNotFound
	}
	return connect.CodeInvalidArgument
}

func registerBundleAdminConnectRoutes(router *mux.Router, planService *PlanService, stripe *StripeService, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	bundlehttp.RegisterConnectRoutes(router, bundleConnectDependencies(planService, stripe), requireAdmin)
}

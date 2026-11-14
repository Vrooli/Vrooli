package main

import (
	"context"
	"errors"
	"net/http"
	"strings"

	bundlehttp "landing-page-business-suite-api/handlers/bundles"
	"landing-page-business-suite-api/internal/commerce"
)

func bundleHandlerDependencies(planService *commerce.PlanService, stripe *StripeService) bundlehttp.Dependencies {
	return bundlehttp.Dependencies{
		Catalog: func(ctx context.Context) (any, error) {
			bundles, err := planService.ListBundleCatalog(ctx)
			if err != nil {
				return nil, err
			}
			return bundlehttp.BuildCatalogResponse(bundles)
		},
		ActiveKey: planService.BundleKey,
		Update: func(ctx context.Context, bundleKey, priceID string, request bundlehttp.UpdatePriceRequest) (any, error) {
			var fetcher commerce.StripePriceFetcher
			if stripe != nil {
				fetcher = stripe.FetchStripePriceDetails
			}
			updated, err := planService.UpdateBundlePriceWithStripe(ctx, bundleKey, priceID, commerce.UpdateBundlePriceInput(request), fetcher)
			if err != nil {
				return nil, err
			}
			return bundlehttp.PlanOptionResponseFromProto(updated), nil
		},
		Path:                getPathParam,
		DecodeJSON:          decodeJSONBody,
		WriteError:          writeJSONError,
		WriteSuccess:        writeJSONSuccessData,
		WriteSuccessMessage: writeJSONSuccess,
		ClassifyError:       classifyStripeError,
		Query:               func(r *http.Request, key string) string { return strings.TrimSpace(getQueryParam(r, key)) },
	}
}

func bundleDeleteHandlerDependencies(planService *commerce.PlanService) bundlehttp.Dependencies {
	deps := bundleHandlerDependencies(planService, nil)
	if store := planService.GetPlanStore(); store != nil {
		deps.DeletePrice = store.DeletePlan
	}
	return deps
}

func bundleCreateHandlerDependencies(planService *commerce.PlanService, stripe *StripeService) bundlehttp.Dependencies {
	deps := bundleHandlerDependencies(planService, stripe)
	if store := planService.GetPlanStore(); store != nil && store.GetBundle() != nil {
		deps.CreatePrice = func(ctx context.Context, bundleKey string, input commerce.CreateBundlePriceInput) (any, error) {
			var fetcher commerce.StripePriceFetcher
			if stripe != nil {
				fetcher = stripe.FetchStripePriceDetails
			}
			plan, err := planService.CreateBundlePrice(ctx, bundleKey, input, fetcher)
			if err != nil {
				return nil, err
			}
			return bundlehttp.PlanOptionResponseFromProto(plan), nil
		}
	}
	return deps
}

func bundleStripeHandlerDependencies(stripe *StripeService) bundlehttp.Dependencies {
	deps := bundlehttp.Dependencies{
		Query:         func(r *http.Request, key string) string { return strings.TrimSpace(getQueryParam(r, key)) },
		WriteError:    writeJSONError,
		WriteSuccess:  writeJSONSuccessData,
		ClassifyError: classifyStripeError,
	}
	if stripe != nil {
		deps.VerifyPrice = func(key string) (any, error) { return stripe.VerifyStripePrice(key) }
	}
	return deps
}

func bundleImportHandlerDependencies(stripe *StripeService, planService *commerce.PlanService) bundlehttp.Dependencies {
	deps := bundleStripeHandlerDependencies(stripe)
	if stripe != nil && planService.GetPlanStore() != nil {
		deps.PreviewImport = func(ctx context.Context) (any, error) {
			return stripe.ListStripeProductsWithPrices(ctx, planService.GetPlanStore())
		}
	} else if stripe != nil {
		deps.PreviewUnavailableMessage = "plan store not available"
	}
	return deps
}

func bundleStripeImportDependencies(stripe *StripeService, planService *commerce.PlanService) bundlehttp.Dependencies {
	deps := bundleHandlerDependencies(planService, stripe)
	deps.ImportPrices = func(ctx context.Context, request bundlehttp.StripeImportRequest) (any, error) {
		var fetcher commerce.StripePriceFetcher
		if stripe != nil {
			fetcher = stripe.FetchStripePriceDetails
		}
		return planService.ImportStripePricesForProduct(ctx, request.Selections, request.BundleProductID, request.Mode, fetcher)
	}
	deps.ClassifyImportError = func(err error) (int, string) {
		if errors.Is(err, commerce.ErrStripeImportNoSelections) || errors.Is(err, commerce.ErrStripeImportNoValidSelections) || errors.Is(err, commerce.ErrStripeImportMissingFetcher) || errors.Is(err, commerce.ErrStripeImportBundleMissing) || errors.Is(err, commerce.ErrStripeImportBundleProductMissing) || errors.Is(err, commerce.ErrStripeImportInvalidMode) || errors.Is(err, commerce.ErrStripeImportProductSwitchRequiresReplace) {
			return http.StatusBadRequest, ApiErrorTypeValidation
		}
		return http.StatusInternalServerError, ApiErrorTypeServerError
	}
	return deps
}

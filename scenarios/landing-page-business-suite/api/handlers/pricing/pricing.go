// Package pricing owns public catalog transport.
package pricing

import (
	"net/http"

	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

type Dependencies struct {
	Overview   func() (*sharedv1.PricingOverview, error)
	WriteJSON  func(http.ResponseWriter, any)
	WriteError func(http.ResponseWriter, int, string, string)
	Log        func(string, map[string]any)
}

func Get(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		overview, err := deps.Overview()
		if err != nil {
			deps.Log("plans_load_failed", map[string]any{"error": err.Error()})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to load pricing plans. Please try again.", "server_error")
			return
		}
		deps.WriteJSON(w, &lpbsv1.GetPricingResponse{Pricing: overview})
	}
}

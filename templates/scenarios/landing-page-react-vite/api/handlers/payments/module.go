// Package payments is the billing domain's API contribution: the generated
// LandingPagePaymentsService Connect handler (checkout, subscription
// verify/cancel, pricing, Stripe settings, billing portal) plus the raw
// POST /api/v1/webhooks/stripe receiver, which must read the raw request body
// and the Stripe-Signature header and therefore cannot be a Connect RPC.
// Business logic lives in internal/{plan,paymentsettings,stripe}.
package payments

import (
	"encoding/json"
	"io"
	"landing-page-react-vite-api/internal/module"
	"landing-page-react-vite-api/internal/stripe"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"
)

// webhookPath is the raw Stripe webhook receiver route (REST exception).
const webhookPath = "/api/v1/webhooks/stripe"

// Module returns the payments domain's contribution: the
// LandingPagePaymentsService Connect handler plus the raw Stripe webhook route.
func Module(deps Deps) module.Module {
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	path, handler := landingconnect.NewLandingPagePaymentsServiceHandler(NewConnectHandler(deps))
	return module.Module{
		Name: "payments",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
			r.HandleFunc(webhookPath, webhookHandler(deps.Stripe, deps.Logger)).Methods(http.MethodPost)
		},
		Endpoints: Endpoints,
	}
}

func webhookHandler(svc *stripe.Service, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		signature := r.Header.Get("Stripe-Signature")
		if signature == "" {
			http.Error(w, "Missing Stripe-Signature header", http.StatusBadRequest)
			return
		}

		if err := svc.HandleWebhook(body, signature); err != nil {
			logger.Printf("payments.webhook: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}
}

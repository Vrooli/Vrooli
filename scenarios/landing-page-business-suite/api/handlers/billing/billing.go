// Package billing owns checkout and customer-portal HTTP transport.
package billing

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
)

type Dependencies struct {
	ValidateEmail       func(http.ResponseWriter, string) (string, bool)
	NormalizeRedirect   func(http.ResponseWriter, string, string) (string, bool)
	ValidateOptionalURL func(string) (string, error)
	CreateCheckout      func(string, string, string, string) (*lpbsv1.CheckoutSession, error)
	CreatePortal        func(context.Context, string, string) (any, error)
	UserEmail           func(context.Context) string
	ClassifyError       func(error) (int, string, string, bool)
	WriteJSON           func(http.ResponseWriter, any)
	WriteError          func(http.ResponseWriter, int, string, string)
	Log                 func(string, map[string]any)
}

func Checkout(deps Dependencies, logKey, errorMessage string, requireEmail bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request lpbsv1.CreateCheckoutSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			deps.WriteError(w, http.StatusBadRequest, "Invalid request body", "validation")
			return
		}
		request.PriceId = strings.TrimSpace(request.PriceId)
		if request.PriceId == "" {
			deps.WriteError(w, http.StatusBadRequest, "price_id is required", "validation")
			return
		}
		request.CustomerEmail = strings.TrimSpace(request.CustomerEmail)
		if requireEmail || request.CustomerEmail != "" {
			normalized, ok := deps.ValidateEmail(w, request.CustomerEmail)
			if !ok {
				return
			}
			request.CustomerEmail = normalized
		}
		success, ok := deps.NormalizeRedirect(w, request.SuccessUrl, "success_url")
		if !ok {
			return
		}
		cancel, ok := deps.NormalizeRedirect(w, request.CancelUrl, "cancel_url")
		if !ok {
			return
		}
		session, err := deps.CreateCheckout(request.PriceId, success, cancel, request.CustomerEmail)
		if err != nil {
			deps.Log(logKey, map[string]any{"error": err.Error(), "price_id": request.PriceId})
			if status, kind, message, ok := deps.ClassifyError(err); ok {
				deps.WriteError(w, status, message, kind)
			} else {
				deps.WriteError(w, http.StatusBadRequest, errorMessage, "server_error")
			}
			return
		}
		deps.WriteJSON(w, &lpbsv1.CreateCheckoutSessionResponse{Session: session})
	}
}

func Portal(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := deps.UserEmail(r.Context())
		if user == "" {
			deps.WriteError(w, http.StatusUnauthorized, "Authentication required", "unauthorized")
			return
		}
		returnURL := strings.TrimSpace(r.URL.Query().Get("return_url"))
		if returnURL != "" {
			normalized, err := deps.ValidateOptionalURL(returnURL)
			if err != nil {
				deps.WriteError(w, http.StatusBadRequest, "Invalid return_url format", "validation")
				return
			}
			returnURL = normalized
		}
		response, err := deps.CreatePortal(r.Context(), user, returnURL)
		if err != nil {
			deps.Log("billing_portal_session_failed", map[string]any{"error": err.Error(), "user": user})
			if status, kind, message, ok := deps.ClassifyError(err); ok {
				deps.WriteError(w, status, message, kind)
			} else {
				deps.WriteError(w, http.StatusBadRequest, "Failed to create billing portal session. Please try again.", "server_error")
			}
			return
		}
		deps.WriteJSON(w, response)
	}
}

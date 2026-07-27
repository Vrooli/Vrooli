// Package account owns HTTP transport for authenticated account read models.
package account

import (
	"context"
	"encoding/json"
	"net/http"

	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"google.golang.org/protobuf/encoding/protojson"
	accountdomain "landing-page-business-suite-api/internal/account"
)

type Reader interface {
	GetSubscriptionContext(context.Context, string) (*shared.SubscriptionStatus, error)
	GetCreditsContext(context.Context, string) (*accountdomain.CreditsEnvelope, error)
	GetEntitlementsContext(context.Context, string) (*accountdomain.EntitlementPayload, error)
}

type Dependencies struct {
	UserEmail  func(context.Context) string
	WriteJSON  func(http.ResponseWriter, interface{})
	WriteError func(http.ResponseWriter, int, string, string)
	LogError   func(string, map[string]interface{})
}

func Subscription(dependencies Dependencies, reader Reader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := dependencies.UserEmail(r.Context())
		if user == "" {
			dependencies.WriteError(w, http.StatusUnauthorized, "Authentication required", "unauthorized")
			return
		}
		status, err := reader.GetSubscriptionContext(r.Context(), user)
		if err != nil {
			dependencies.LogError("subscription_fetch_failed", map[string]interface{}{"user": user, "error": err.Error()})
			dependencies.WriteError(w, http.StatusInternalServerError, "Failed to retrieve subscription status. Please try again.", "server_error")
			return
		}
		dependencies.WriteJSON(w, &shared.VerifySubscriptionResponse{Status: status})
	}
}

func Credits(dependencies Dependencies, reader Reader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := dependencies.UserEmail(r.Context())
		if user == "" {
			dependencies.WriteError(w, http.StatusUnauthorized, "Authentication required", "unauthorized")
			return
		}
		credits, err := reader.GetCreditsContext(r.Context(), user)
		if err != nil {
			dependencies.LogError("credits_fetch_failed", map[string]interface{}{"user": user, "error": err.Error()})
			dependencies.WriteError(w, http.StatusInternalServerError, "Failed to retrieve credit balance. Please try again.", "server_error")
			return
		}
		balance := map[string]interface{}{}
		if credits.Balance != nil {
			data, marshalErr := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(credits.Balance)
			if marshalErr == nil {
				if unmarshalErr := json.Unmarshal(data, &balance); unmarshalErr != nil {
					dependencies.LogError("credits_balance_unmarshal_failed", map[string]interface{}{"user": user, "error": unmarshalErr.Error()})
				}
			}
		}
		dependencies.WriteJSON(w, map[string]interface{}{"balance": balance, "display_credits_label": credits.DisplayCreditsLabel, "display_credits_multiplier": credits.DisplayCreditsMultiplier})
	}
}

func Entitlements(dependencies Dependencies, reader Reader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := dependencies.UserEmail(r.Context())
		if user == "" {
			dependencies.WriteError(w, http.StatusUnauthorized, "Authentication required", "unauthorized")
			return
		}
		payload, err := reader.GetEntitlementsContext(r.Context(), user)
		if err != nil {
			dependencies.LogError("entitlements_fetch_failed", map[string]interface{}{"user": user, "error": err.Error()})
			dependencies.WriteError(w, http.StatusInternalServerError, "Failed to retrieve entitlements. Please try again.", "server_error")
			return
		}
		dependencies.WriteJSON(w, payload)
	}
}

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/vrooli/api-core/health"
)

const signInDeliveryWindow = time.Hour

// signInDeliveryCheck makes a failing sign-in mail provider visible in health
// instead of only in logs. It is optional: the API still serves, but
// customers cannot receive codes, so operators see degraded status.
func (s *Server) signInDeliveryCheck() health.Checker {
	return health.Func("sign_in_email", func(ctx context.Context) error {
		if s.userAuthService == nil {
			return errors.New("sign-in service unavailable")
		}
		summary, err := s.userAuthService.DeliveryHealth(ctx, signInDeliveryWindow)
		if err != nil {
			return fmt.Errorf("read sign-in delivery outcomes: %w", err)
		}
		if summary.LastFailure != nil && (summary.LastSuccess == nil || summary.LastFailure.After(*summary.LastSuccess)) {
			return fmt.Errorf("%d of the last %d sign-in emails failed in the past hour; latest: %s", summary.Failed, summary.Failed+summary.Sent, summary.LastError)
		}
		return nil
	})
}

// signInDeliveryReport exposes recent outcomes to administrators.
func (s *Server) signInDeliveryReport(w http.ResponseWriter, r *http.Request) {
	summary, err := s.userAuthService.DeliveryHealth(r.Context(), 24*time.Hour)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Unable to read sign-in delivery outcomes.", ApiErrorTypeServerError)
		return
	}
	writeJSON(w, map[string]any{"window_hours": 24, "delivery": summary})
}

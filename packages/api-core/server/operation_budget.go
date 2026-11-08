package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// WithOperationBudget mounts an explicitly bounded long-running RPC. Its work
// follows the request context, while its response can carry the completion or
// timeout receipt beyond the server's ordinary API write deadline. The handler
// must honor cancellation; this wrapper never detaches work from its request.
func WithOperationBudget(next http.Handler, budget time.Duration) (http.Handler, error) {
	if next == nil || budget <= 0 {
		return nil, fmt.Errorf("operation handler and positive budget are required")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := http.NewResponseController(w).SetWriteDeadline(time.Time{}); err != nil && !errors.Is(err, http.ErrNotSupported) {
			http.Error(w, "operation response deadline unavailable", http.StatusInternalServerError)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), budget)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	}), nil
}

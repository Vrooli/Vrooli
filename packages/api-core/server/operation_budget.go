package server

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const operationResponseWriteGrace = 15 * time.Second

// WithOperationBudget mounts an explicitly bounded long-running RPC. Its work
// follows the request context; the response gets the work budget plus a finite
// write grace after the ordinary API deadline. Slow readers cannot hold the
// connection forever. The handler must honor cancellation; this wrapper never
// detaches work from its request.
func WithOperationBudget(next http.Handler, budget time.Duration) (http.Handler, error) {
	if next == nil || budget <= 0 {
		return nil, fmt.Errorf("operation handler and positive budget are required")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeDeadline := time.Now().Add(budget).Add(operationResponseWriteGrace)
		if err := http.NewResponseController(w).SetWriteDeadline(writeDeadline); err != nil {
			http.Error(w, "operation response deadline unavailable", http.StatusInternalServerError)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), budget)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	}), nil
}

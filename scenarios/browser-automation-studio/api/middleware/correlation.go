package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/services/logutil"
)

// CorrelationIDHeader is the HTTP header used to pass correlation IDs
const CorrelationIDHeader = "X-Correlation-ID"

// CorrelationMiddleware generates and propagates correlation IDs for request tracing
type CorrelationMiddleware struct {
	logger *logrus.Logger
}

// NewCorrelationMiddleware creates a new correlation middleware instance
func NewCorrelationMiddleware(logger *logrus.Logger) *CorrelationMiddleware {
	return &CorrelationMiddleware{
		logger: logger,
	}
}

// InjectCorrelationID returns middleware that injects correlation IDs into requests
func (m *CorrelationMiddleware) InjectCorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for existing correlation ID in request header
		correlationID := r.Header.Get(CorrelationIDHeader)

		// Generate new correlation ID if not present
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		// Add correlation ID to response header
		w.Header().Set(CorrelationIDHeader, correlationID)

		// Add correlation ID to context
		ctx := logutil.ContextWithCorrelationID(r.Context(), correlationID)

		// Create structured logger with correlation ID and add to context
		structuredLogger := logutil.NewStructuredLogger(m.logger).
			WithCorrelationID(correlationID).
			WithContext(logutil.ContextAPI)

		ctx = logutil.ContextWithLogger(ctx, structuredLogger)

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// PropagateCorrelationID adds the correlation ID to an outgoing HTTP request
func PropagateCorrelationID(req *http.Request, correlationID string) {
	if correlationID != "" {
		req.Header.Set(CorrelationIDHeader, correlationID)
	}
}

// PropagateCorrelationIDFromContext adds the correlation ID from context to an outgoing HTTP request
func PropagateCorrelationIDFromContext(req *http.Request) {
	if correlationID := logutil.CorrelationIDFromContext(req.Context()); correlationID != "" {
		req.Header.Set(CorrelationIDHeader, correlationID)
	}
}

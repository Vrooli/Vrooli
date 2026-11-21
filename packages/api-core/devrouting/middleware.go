package devrouting

import (
	"context"
	"fmt"
	"net/http"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
)

// PinnedTestModeMiddleware binds development test requests to this service's
// exact paired SQL/file lease before any handler, asynchronous job or WebSocket
// upgrade can capture their context. Register this same Service as the routing
// RPC owner: its operation lock prevents capturing a half-installed pair.
// Ordinary requests and production-mode behavior retain the marker middleware's
// existing contract. A missing, expired or inconsistent lease refuses a marked
// request rather than reaching primary storage.
func (s *Service) PinnedTestModeMiddleware(next http.Handler) http.Handler {
	if !enabled() {
		return next
	}
	return apihttp.TestModeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !database.IsTestMode(r.Context()) {
			next.ServeHTTP(w, r)
			return
		}
		ctx, err := s.pinRequestContext(r.Context())
		if err != nil {
			http.Error(w, "test routing lease unavailable", http.StatusServiceUnavailable)
			return
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	}))
}

func (s *Service) pinRequestContext(ctx context.Context) (context.Context, error) {
	if s == nil || s.db == nil || s.roots == nil {
		return nil, fmt.Errorf("paired test storage is required")
	}
	s.opMu.Lock()
	defer s.opMu.Unlock()
	leaseID := s.db.ActiveLeaseID()
	ctx, err := s.db.PinTestPool(ctx, leaseID)
	if err != nil {
		return nil, err
	}
	return s.roots.PinTestRoots(ctx, leaseID)
}

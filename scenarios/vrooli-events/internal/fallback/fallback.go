// DOC: docs/guides/integrating-a-scenario.md
// DOC: docs/internal/ERROR-SEMANTICS.md
package fallback

import (
	"errors"
	"net/http"
)

// ErrEventsUnavailable is returned when vrooli-events is not reachable.
var ErrEventsUnavailable = errors.New("vrooli-events is unavailable")

// Mode controls behavior when vrooli-events is unreachable.
type Mode string

const (
	ModeFailOpen   Mode = "fail_open"   // Allow all when events unavailable
	ModeFailClosed Mode = "fail_closed" // Deny all when events unavailable
)

// Check tests if the vrooli-events API is reachable.
// Returns nil if healthy, ErrEventsUnavailable otherwise.
func Check(eventsURL string) error {
	if eventsURL == "" {
		return ErrEventsUnavailable
	}
	resp, err := http.Get(eventsURL + "/health") //nolint:gosec // URL is from trusted config
	if err != nil {
		return ErrEventsUnavailable
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return ErrEventsUnavailable
	}
	return nil
}

// ShouldAllow returns whether a request should proceed given the fallback mode
// and the availability of vrooli-events.
func ShouldAllow(mode Mode, eventsAvailable bool) bool {
	if eventsAvailable {
		return true // Normal operation
	}
	return mode == ModeFailOpen
}

// NoopMiddleware returns a no-op middleware that passes all requests through.
// This is used when vrooli-events integration is not configured (opt-in zero-dep).
func NoopMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return next
	}
}

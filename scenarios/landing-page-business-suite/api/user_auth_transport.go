package main

import (
	"net/http"
	"strings"
	"time"

	userauthhttp "landing-page-business-suite-api/handlers/administration"
)

// userAuthHandlerDependencies is the root composition boundary for browser
// authentication transport. Domain identity remains internal/administration.
func userAuthHandlerDependencies(service userauthhttp.UserAuthService, limiter userauthhttp.EmailRateLimiter) userauthhttp.UserAuthDependencies {
	return userauthhttp.UserAuthDependencies{
		Service:       service,
		RateLimiter:   limiter,
		ClientIP:      getClientIP,
		SessionID:     getSessionID,
		UserID:        getUserID,
		ResolveSecret: resolveSecret,
		SecureCookies: func() bool { return strings.HasPrefix(resolveSecret("AUTH_MAGIC_LINK_BASE_URL"), "https://") },
		Now:           time.Now,
		WriteError:    writeJSONError,
		Log:           logStructured,
		LogError:      logStructuredError,
	}
}

// The declaration keeps net/http imported in this composition file where the
// exposed dependency contract is documented by its handler package.
var _ http.Handler = http.HandlerFunc(nil)

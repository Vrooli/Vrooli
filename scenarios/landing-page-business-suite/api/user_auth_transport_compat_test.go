package main

import (
	"net/http"
	"strings"
	"time"

	userauthhttp "landing-page-business-suite-api/handlers/administration"
	"landing-page-business-suite-api/internal/administration"
)

// These test-only adapters retain the existing characterization suite while
// production routes use handlers/administration directly.
type (
	MagicLinkRequest    = userauthhttp.MagicLinkRequest
	MagicLinkResponse   = userauthhttp.MagicLinkResponse
	TokenRefreshRequest = userauthhttp.TokenRefreshRequest
)

func handleMagicLinkRequest(service *administration.UserAuthService, limiter *RateLimiter) http.HandlerFunc {
	return userauthhttp.RequestMagicLink(userAuthHandlerDependencies(service, limiter))
}

func handleMagicLinkVerify(service *administration.UserAuthService) http.HandlerFunc {
	return userauthhttp.VerifyMagicLink(userAuthHandlerDependencies(service, nil))
}

func handleTokenRefresh(service *administration.UserAuthService) http.HandlerFunc {
	return userauthhttp.RefreshTokens(userAuthHandlerDependencies(service, nil))
}

func handleUserLogout(service *administration.UserAuthService) http.HandlerFunc {
	return userauthhttp.LogoutUser(userAuthHandlerDependencies(service, nil))
}

func handleAuthMe(service *administration.UserAuthService) http.HandlerFunc {
	return userauthhttp.Me(userAuthHandlerDependencies(service, nil))
}

func setAuthCookies(w http.ResponseWriter, pair *administration.TokenPair) {
	userauthhttp.SetAuthCookies(w, pair, isSecureContext(), time.Now())
}
func clearAuthCookies(w http.ResponseWriter) { userauthhttp.ClearAuthCookies(w, isSecureContext()) }
func isSecureContext() bool {
	return strings.HasPrefix(resolveSecret("AUTH_MAGIC_LINK_BASE_URL"), "https://")
}
func formatNullableTime(value *time.Time) any { return userauthhttp.FormatNullableTime(value) }

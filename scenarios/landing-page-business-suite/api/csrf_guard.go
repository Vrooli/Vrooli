package main

import (
	"net/http"
	"net/url"
	"strings"

	"landing-page-business-suite-api/internal/logx"
)

// cookieAuthNames are the cookies that make a browser request carry ambient
// authority. A cross-site page can make the browser send them, so state-
// changing requests that carry them must come from this site.
var cookieAuthNames = []string{"admin_session", "access_token", "refresh_token"}

// sameOriginGuard rejects cross-site, state-changing requests that would be
// authenticated by cookies. SameSite=Lax already blocks most of these; this
// closes the remaining same-site and legacy-browser gaps. Bearer-token and
// webhook callers never carry these cookies and are unaffected.
func sameOriginGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isSafeMethod(r.Method) || !carriesAuthCookie(r) || requestFromThisSite(r) {
			next.ServeHTTP(w, r)
			return
		}
		logx.Info("cross_site_request_blocked", map[string]interface{}{
			"level": "warn", "path": r.URL.Path, "origin": r.Header.Get("Origin"), "sec_fetch_site": r.Header.Get("Sec-Fetch-Site"),
		})
		writeJSONError(w, http.StatusForbidden, "This request must come from this site.", ApiErrorTypeForbidden)
	})
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	return false
}

func carriesAuthCookie(r *http.Request) bool {
	for _, name := range cookieAuthNames {
		if cookie, err := r.Cookie(name); err == nil && cookie.Value != "" {
			return true
		}
	}
	return false
}

func requestFromThisSite(r *http.Request) bool {
	switch strings.ToLower(strings.TrimSpace(r.Header.Get("Sec-Fetch-Site"))) {
	case "same-origin", "none":
		return true
	case "same-site", "cross-site":
		return false
	}
	// Older clients: fall back to Origin, then Referer. Non-browser clients
	// send neither and are not subject to cross-site forgery.
	source := r.Header.Get("Origin")
	if source == "" || source == "null" {
		source = r.Header.Get("Referer")
	}
	if source == "" {
		return r.Header.Get("Origin") != "null"
	}
	parsed, err := url.Parse(source)
	if err != nil || parsed.Host == "" {
		return false
	}
	host := strings.ToLower(parsed.Host)
	for _, allowed := range allowedRequestHosts(r) {
		if host == allowed {
			return true
		}
	}
	return false
}

func allowedRequestHosts(r *http.Request) []string {
	initTrustedProxies()
	hosts := []string{strings.ToLower(r.Host)}
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Host"), ",")[0]); forwarded != "" && isIPFromTrustedProxy(extractIPFromRemoteAddr(r.RemoteAddr)) {
		hosts = append(hosts, strings.ToLower(forwarded))
	}
	if public, err := url.Parse(resolvePublicBaseURL()); err == nil && public.Host != "" {
		hosts = append(hosts, strings.ToLower(public.Host))
	}
	return hosts
}

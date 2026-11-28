// Package httporigin validates browser Origin headers at HTTP/WebSocket
// boundaries, including requests relayed by the same-origin scenario proxy.
package httporigin

import (
	"net/http"
	"net/url"
	"strings"
)

// Matches reports whether origin names the request authority. The trusted
// api-base WebSocket proxy preserves the browser-facing authority in
// X-Forwarded-Host after checking the same-origin policy itself. Direct
// requests use Host and also require the request scheme to match.
func Matches(req *http.Request, origin string) bool {
	if req == nil {
		return false
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Opaque != "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}

	forwardedHost := strings.TrimSpace(req.Header.Get("X-Forwarded-Host"))
	if forwardedHost != "" {
		if strings.Contains(forwardedHost, ",") {
			return false
		}
		originAuthority, ok := normalizeAuthority(parsed.Host, parsed.Scheme)
		if !ok {
			return false
		}
		forwardedAuthority, ok := normalizeAuthority(forwardedHost, parsed.Scheme)
		return ok && originAuthority == forwardedAuthority
	}

	scheme := strings.TrimSpace(req.URL.Scheme)
	if scheme == "" {
		if req.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	if !strings.EqualFold(parsed.Scheme, scheme) {
		return false
	}
	originAuthority, ok := normalizeAuthority(parsed.Host, parsed.Scheme)
	if !ok {
		return false
	}
	requestAuthority, ok := normalizeAuthority(req.Host, parsed.Scheme)
	return ok && originAuthority == requestAuthority
}

func normalizeAuthority(authority, scheme string) (string, bool) {
	parsed, err := url.Parse("//" + authority)
	if err != nil || parsed.Hostname() == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", false
	}
	host := strings.ToLower(parsed.Hostname())
	port := parsed.Port()
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		port = ""
	}
	if port != "" {
		return host + ":" + port, true
	}
	return host, true
}

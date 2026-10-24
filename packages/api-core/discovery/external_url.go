package discovery

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// UIPortKey is the port key for a scenario's browser-facing interface. It is
// the counterpart to defaultPortKey: API_PORT is the address this process
// dials, UI_PORT is the address a person's browser opens. Resolving the first
// where the second was meant produces a link to a Connect endpoint, which
// answers a browser with 404.
const UIPortKey = "UI_PORT"

// externalHostContextKey carries the browser-facing host across the boundary
// between an HTTP handler and the RPC that needs it.
type externalHostContextKey struct{}

// loopbackHosts are the hostnames that mean "the machine this browser is
// already on". A request that arrives on one of them cannot teach us any
// externally meaningful origin, so a port on localhost is the honest answer.
var loopbackHosts = map[string]struct{}{
	"localhost": {},
	"127.0.0.1": {},
	"0.0.0.0":   {},
}

// ExternalHost reports the host a browser used to reach this process. A proxy
// or tunnel in front of the scenario rewrites Host with its own address, so the
// forwarded value wins when one is present.
//
// Go's HTTP server moves the Host header into Request.Host rather than leaving
// it in Header, which is why a Connect handler reading only req.Header() sees
// nothing here. Handlers should call this once and pass the result inward, or
// install ExternalHostMiddleware and read it from the context.
func ExternalHost(r *http.Request) string {
	if r == nil {
		return ""
	}
	if forwarded := firstForwardedHost(r.Header.Get("X-Forwarded-Host")); forwarded != "" {
		return forwarded
	}
	if host := strings.TrimSpace(r.Host); host != "" {
		return host
	}
	return strings.TrimSpace(r.Header.Get("Host"))
}

func firstForwardedHost(header string) string {
	for _, candidate := range strings.Split(header, ",") {
		if trimmed := strings.TrimSpace(candidate); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// ExternalHostMiddleware records the browser-facing host on the request
// context so RPC handlers can resolve cross-scenario links without threading
// an *http.Request through their signatures.
func ExternalHostMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(ContextWithExternalHost(r.Context(), ExternalHost(r))))
	})
}

// ContextWithExternalHost stores a browser-facing host on ctx.
func ContextWithExternalHost(ctx context.Context, host string) context.Context {
	host = strings.TrimSpace(host)
	if ctx == nil || host == "" {
		return ctx
	}
	return context.WithValue(ctx, externalHostContextKey{}, host)
}

// ExternalHostFromContext reads a host recorded by ExternalHostMiddleware. An
// empty result means no request context was available, which BrowserURLForHost
// treats as a local browser.
func ExternalHostFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	host, _ := ctx.Value(externalHostContextKey{}).(string)
	return host
}

// BrowserURLForHost builds the address a browser at requestHost can open to
// reach scenarioSlug's interface.
//
// The caller's own origin decides the shape, because the server cannot know it
// any other way:
//
//   - a loopback or missing host means the browser shares this machine, so a
//     port on localhost is reachable;
//   - a host of three or more labels is a tunnel or proxy domain that gives
//     each scenario its own subdomain, so the first label is replaced;
//   - anything else (a bare hostname, a two-label domain) has no derivable
//     per-scenario address, so localhost remains the only honest answer.
//
// This mirrors resolveExternalUrl in packages/api-base for scenarios whose API
// is written in Go. Both must stay in agreement: a link that resolves one way
// through an Express scenario and another through a Go one is worse than
// either rule alone.
func BrowserURLForHost(requestHost, scenarioSlug string, uiPort int) string {
	localURL := fmt.Sprintf("http://localhost:%d", uiPort)
	scenarioSlug = strings.TrimSpace(scenarioSlug)
	hostname := hostWithoutPort(requestHost)
	if hostname == "" || scenarioSlug == "" {
		return localURL
	}
	if _, loopback := loopbackHosts[hostname]; loopback {
		return localURL
	}
	labels := strings.Split(hostname, ".")
	if len(labels) < 3 {
		return localURL
	}
	labels[0] = scenarioSlug
	return "https://" + strings.Join(labels, ".")
}

// hostWithoutPort strips a port and IPv6 brackets from a Host header value.
func hostWithoutPort(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if stripped, _, err := net.SplitHostPort(host); err == nil {
		host = stripped
	}
	return strings.Trim(host, "[]")
}

// ResolveExternalURL resolves the browser-openable interface URL of another
// scenario, relative to the origin the calling browser used.
//
// requestHost is the value from ExternalHost or ExternalHostFromContext. An
// empty requestHost is not an error: it resolves to localhost, which is correct
// for every same-machine caller and is the only address available when no
// request context exists.
func (r *Resolver) ResolveExternalURL(ctx context.Context, scenarioSlug, requestHost string) (string, error) {
	port, err := r.ResolveScenarioPort(ctx, scenarioSlug, UIPortKey)
	if err != nil {
		return "", err
	}
	return BrowserURLForHost(requestHost, scenarioSlug, port), nil
}

// ResolveExternalURL is a convenience wrapper using the shared resolver.
func ResolveExternalURL(ctx context.Context, scenarioSlug, requestHost string) (string, error) {
	return DefaultResolver().ResolveExternalURL(ctx, scenarioSlug, requestHost)
}

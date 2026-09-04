package analytics

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// Dimensions contains only analytics dimensions. It intentionally has no IP
// field: the connection address is used for classification and never leaves
// Enrich.
type Dimensions struct {
	ReferrerHost, ReferrerKind, CountryCode, DeviceClass string
}

func Enrich(r *http.Request, referrer string) Dimensions {
	host := ""
	if parsed, err := url.Parse(referrer); err == nil {
		host = strings.ToLower(parsed.Hostname())
	}
	return Dimensions{ReferrerHost: host, ReferrerKind: classifyReferrer(host, os.Getenv("LANDING_SITE_HOST")), CountryCode: countryForRequest(r), DeviceClass: classifyDevice(r.Header.Get("User-Agent"))}
}

func classifyReferrer(host, site string) string {
	host = strings.TrimPrefix(strings.ToLower(host), "www.")
	site = strings.TrimPrefix(strings.ToLower(site), "www.")
	if host == "" || (site != "" && host == site) {
		return "direct"
	}
	for _, suffix := range []string{"google.", "bing.", "duckduckgo.", "yahoo."} {
		if strings.Contains(host, suffix) {
			return "search"
		}
	}
	for _, name := range []string{"facebook.", "instagram.", "linkedin.", "twitter.", "x.com", "tiktok.", "youtube."} {
		if strings.Contains(host, name) {
			return "social"
		}
	}
	if strings.Contains(host, "utm") || strings.Contains(host, "ads.") {
		return "paid"
	}
	return "referral"
}

func classifyDevice(ua string) string {
	ua = strings.ToLower(ua)
	if ua == "" {
		return "unknown"
	}
	if strings.Contains(ua, "ipad") || strings.Contains(ua, "tablet") {
		return "tablet"
	}
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "iphone") || strings.Contains(ua, "android") {
		return "mobile"
	}
	return "desktop"
}

func countryForRequest(r *http.Request) string {
	// A deployment may configure a trusted edge's already-resolved country.
	// The address is deliberately only inspected to enforce proxy trust and is
	// never returned, logged, or persisted.
	peer, _, _ := net.SplitHostPort(r.RemoteAddr)
	if peer == "" {
		peer = r.RemoteAddr
	}
	trusted := false
	for _, raw := range strings.Split(os.Getenv("TRUSTED_PROXY_CIDRS"), ",") {
		_, network, err := net.ParseCIDR(strings.TrimSpace(raw))
		if err == nil && network.Contains(net.ParseIP(peer)) {
			trusted = true
			break
		}
	}
	if trusted && r.Header.Get("X-Geo-Country") != "" {
		return strings.ToUpper(strings.TrimSpace(r.Header.Get("X-Geo-Country")))
	}
	if !trusted && r.Header.Get("X-Geo-Country") != "" {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(os.Getenv("ANALYTICS_COUNTRY_CODE")))
}

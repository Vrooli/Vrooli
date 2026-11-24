package delivery

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidateStorefrontLink limits purchase links to the official public listing
// hosts for the store named by the owner. Display data must not be able to
// publish an arbitrary scheme or redirect destination as an app-store link.
func ValidateStorefrontLink(store, rawURL string) error {
	store = strings.TrimSpace(store)
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("storefront url is required")
	}
	u, err := url.Parse(rawURL)
	if err != nil || !u.IsAbs() || !strings.EqualFold(u.Scheme, "https") || u.Hostname() == "" {
		return fmt.Errorf("storefront url must be an absolute HTTPS URL")
	}
	if u.User != nil || strings.Contains(rawURL, "#") || (u.Port() != "" && u.Port() != "443") {
		return fmt.Errorf("storefront url contains unsupported authority or fragment data")
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	switch store {
	case "app_store":
		if host != "apps.apple.com" && host != "itunes.apple.com" {
			return fmt.Errorf("app_store url must use an official Apple App Store host")
		}
		if !isAppleListingPath(u.EscapedPath()) {
			return fmt.Errorf("app_store url must identify an App Store listing")
		}
	case "play_store":
		if host != "play.google.com" {
			return fmt.Errorf("play_store url must use the official Google Play host")
		}
		if u.Path != "/store/apps/details" || strings.TrimSpace(u.Query().Get("id")) == "" {
			return fmt.Errorf("play_store url must identify a Google Play app listing")
		}
	default:
		return fmt.Errorf("unsupported storefront %q", store)
	}
	return nil
}

func isAppleListingPath(path string) bool {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) < 2 {
		return false
	}
	appIndex := -1
	for i, segment := range segments {
		if segment == "app" {
			appIndex = i
			break
		}
	}
	if appIndex < 0 || appIndex > 1 {
		return false
	}
	if appIndex == 1 {
		country := segments[0]
		if len(country) != 2 || !isASCIIAlpha(country[0]) || !isASCIIAlpha(country[1]) {
			return false
		}
	}
	for _, segment := range segments {
		if segment == "" {
			return false
		}
	}
	suffix := segments[appIndex+1:]
	if len(suffix) < 1 || len(suffix) > 2 {
		return false
	}
	id := suffix[len(suffix)-1]
	if len(id) <= 2 || !strings.EqualFold(id[:2], "id") {
		return false
	}
	for _, char := range id[2:] {
		if char < '0' || char > '9' {
			return false
		}
	}
	return len(suffix) == 1 || suffix[0] != ""
}

func isASCIIAlpha(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}

// NormalizeStorefrontLinks trims owner-provided labels and URLs before storing
// them, then rejects unsupported or unsafe store destinations.
func NormalizeStorefrontLinks(storefronts []Storefront) ([]Storefront, error) {
	result := make([]Storefront, len(storefronts))
	copy(result, storefronts)
	for i := range result {
		result[i].Store = strings.TrimSpace(result[i].Store)
		result[i].Label = strings.TrimSpace(result[i].Label)
		result[i].URL = strings.TrimSpace(result[i].URL)
		result[i].Badge = strings.TrimSpace(result[i].Badge)
		if err := ValidateStorefrontLink(result[i].Store, result[i].URL); err != nil {
			return nil, fmt.Errorf("storefront %q: %w", result[i].Store, err)
		}
	}
	return result, nil
}

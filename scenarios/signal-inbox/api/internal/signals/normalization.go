package signals

import (
	"fmt"
	"net/url"
	"strings"
)

// NormalizeSourceIdentity removes non-content URL variations before hashing.
func NormalizeSourceIdentity(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("must be an absolute URL")
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	q := u.Query()
	for key := range q {
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, "utm_") || lower == "fbclid" || lower == "gclid" || lower == "mc_cid" || lower == "mc_eid" {
			q.Del(key)
		}
	}
	// url.Values.Encode is sorted; retain only content-affecting query fields.
	u.RawQuery = q.Encode()
	return u.String(), nil
}

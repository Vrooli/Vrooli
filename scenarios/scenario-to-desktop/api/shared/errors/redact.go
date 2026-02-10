package errors

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	// Common sensitive fields (keep conservative; better to over-redact than leak).
	reBearerToken = regexp.MustCompile(`(?i)\b(bearer)\s+[a-z0-9._\-]+`)
	reAPIKeyKV    = regexp.MustCompile(`(?i)\b(x-update-key|api_key|apikey|token|secret|password)\b\s*[:=]\s*[^ \t\r\n,;]+`)
	reAKIA        = regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)
)

// Redact attempts to remove/neutralize sensitive material from error strings.
// This is used for surfacing underlying causes in user-facing pipeline output.
func Redact(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	// Header-ish patterns / key-value leaks.
	s = reBearerToken.ReplaceAllString(s, "${1} REDACTED")
	s = reAPIKeyKV.ReplaceAllString(s, "$1=REDACTED")
	s = reAKIA.ReplaceAllString(s, "AKIAREDACTED")

	// Presigned URLs: strip/neutralize common signature-bearing query params.
	// Do a best-effort parse; if parse fails, do a simple substring scrub.
	if strings.Contains(s, "http://") || strings.Contains(s, "https://") {
		s = redactURLsInString(s)
	}

	return s
}

func redactURLsInString(s string) string {
	// Fast path: scrub common AWS query param keys without full URL parsing.
	// (We still try parsing below, but this catches most cases.)
	s = regexp.MustCompile(`(?i)(X-Amz-(Algorithm|Credential|Date|Expires|SignedHeaders|Signature|Security-Token|Checksum-Mode))=[^&\s]+`).
		ReplaceAllString(s, "$1=REDACTED")
	s = regexp.MustCompile(`(?i)(AWSAccessKeyId|Signature)=[^&\s]+`).
		ReplaceAllString(s, "$1=REDACTED")

	// Attempt to parse standalone URLs and redact their query fully for known keys.
	// This is best-effort: we only handle obvious URL tokens separated by whitespace.
	parts := strings.Fields(s)
	for i, p := range parts {
		if !(strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://")) {
			continue
		}
		u, err := url.Parse(p)
		if err != nil || u.RawQuery == "" {
			continue
		}
		q := u.Query()
		for key := range q {
			kl := strings.ToLower(key)
			if strings.HasPrefix(kl, "x-amz-") || kl == "signature" || kl == "awsaccesskeyid" || strings.Contains(kl, "token") || strings.Contains(kl, "secret") {
				q.Set(key, "REDACTED")
			}
		}
		u.RawQuery = q.Encode()
		parts[i] = u.String()
	}
	return strings.Join(parts, " ")
}

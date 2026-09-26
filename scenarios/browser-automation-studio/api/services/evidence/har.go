package evidence

import (
	"encoding/json"
	"net/url"
	"strings"

	basevidence "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/evidence"
)

const RedactedValue = "[REDACTED]"

// SanitizeHAR removes secret-bearing headers, query values, and request/response
// bodies from a HAR before it crosses the protected-storage boundary.
func SanitizeHAR(raw []byte, policy *basevidence.EvidencePolicy) ([]byte, error) {
	// INVARIANT: harDerivativeIsRedacted
	// Any HAR derivative that crosses the protected-storage boundary uses policy
	// redaction; callers cannot opt into a raw publication by omitting policy.
	if policy == nil {
		policy = DefaultPolicy()
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, err
	}
	redactHARValue(document, headerSet(policy.RedactedHeaderNames), headerSet(policy.RedactedQueryParameterNames))
	return json.Marshal(document)
}

// SanitizeNetworkURL removes user information and sensitive query values from
// a network URL using the same disclosure policy as HAR evidence.
func SanitizeNetworkURL(raw string, policy *basevidence.EvidencePolicy) string {
	if policy == nil {
		policy = DefaultPolicy()
	}
	if !policy.RedactNetwork {
		return raw
	}
	return redactURL(raw, headerSet(policy.RedactedQueryParameterNames))
}

// SanitizeNetworkHeaders copies headers and redacts policy-listed values.
func SanitizeNetworkHeaders(headers map[string]string, policy *basevidence.EvidencePolicy) map[string]string {
	if headers == nil {
		return nil
	}
	if policy == nil {
		policy = DefaultPolicy()
	}
	result := make(map[string]string, len(headers))
	if !policy.RedactNetwork {
		for key, value := range headers {
			result[key] = value
		}
		return result
	}
	sensitive := headerSet(policy.RedactedHeaderNames)
	for key, value := range headers {
		if _, redact := sensitive[strings.ToLower(strings.TrimSpace(key))]; redact {
			value = RedactedValue
		}
		result[key] = value
	}
	return result
}

// SanitizeNetworkPreview retains structured JSON fields that are safe under
// the evidence policy. Opaque previews are removed because their contents have
// no trustworthy field boundaries for secret detection.
func SanitizeNetworkPreview(raw string, policy *basevidence.EvidencePolicy) string {
	if raw == "" {
		return raw
	}
	if policy == nil {
		policy = DefaultPolicy()
	}
	if !policy.RedactNetwork {
		return raw
	}
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return RedactedValue
	}
	sensitive := headerSet(append(append([]string(nil), policy.RedactedHeaderNames...), policy.RedactedQueryParameterNames...))
	redactHARValue(value, sensitive, sensitive)
	sanitized, err := json.Marshal(value)
	if err != nil {
		return RedactedValue
	}
	return string(sanitized)
}

func headerSet(names []string) map[string]struct{} {
	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		set[strings.ToLower(strings.TrimSpace(name))] = struct{}{}
	}
	return set
}

func redactHARValue(value any, headers, query map[string]struct{}) {
	switch node := value.(type) {
	case map[string]any:
		if name, ok := node["name"].(string); ok {
			if _, sensitive := headers[strings.ToLower(name)]; sensitive {
				node["value"] = RedactedValue
			}
			if _, sensitive := query[strings.ToLower(name)]; sensitive {
				node["value"] = RedactedValue
			}
		}
		if rawURL, ok := node["url"].(string); ok {
			node["url"] = redactURL(rawURL, query)
		}
		for key, child := range node {
			if _, sensitive := headers[strings.ToLower(strings.TrimSpace(key))]; sensitive {
				node[key] = RedactedValue
				continue
			}
			if key == "postData" || key == "content" {
				node[key] = map[string]any{"text": RedactedValue}
				continue
			}
			redactHARValue(child, headers, query)
		}
	case []any:
		for _, child := range node {
			redactHARValue(child, headers, query)
		}
	}
}

func redactURL(raw string, query map[string]struct{}) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if parsed.User != nil {
		parsed.User = url.UserPassword(RedactedValue, RedactedValue)
	}
	values := parsed.Query()
	for key := range values {
		if _, sensitive := query[strings.ToLower(key)]; sensitive {
			values.Set(key, RedactedValue)
		}
	}
	parsed.RawQuery = values.Encode()
	return parsed.String()
}

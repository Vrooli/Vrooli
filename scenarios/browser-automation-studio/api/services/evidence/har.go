package evidence

import (
	"encoding/json"
	"net/url"
	"strings"

	basevidence "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/evidence"
)

const redactedValue = "[REDACTED]"

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
				node["value"] = redactedValue
			}
			if _, sensitive := query[strings.ToLower(name)]; sensitive {
				node["value"] = redactedValue
			}
		}
		if rawURL, ok := node["url"].(string); ok {
			node["url"] = redactURL(rawURL, query)
		}
		for key, child := range node {
			if key == "postData" || key == "content" {
				node[key] = map[string]any{"text": redactedValue}
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
	values := parsed.Query()
	for key := range values {
		if _, sensitive := query[strings.ToLower(key)]; sensitive {
			values.Set(key, redactedValue)
		}
	}
	parsed.RawQuery = values.Encode()
	return parsed.String()
}

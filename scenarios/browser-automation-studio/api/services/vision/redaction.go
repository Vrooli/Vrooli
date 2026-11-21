package vision

import (
	"net/url"
	"regexp"
	"strings"
)

const redactedNavigationValue = "[REDACTED]"

func redactedHumanIntervention(input *HumanInterventionInfo) *HumanInterventionInfo {
	if input == nil {
		return nil
	}
	output := *input
	output.Reason = redactNavigationText(input.Reason)
	output.Instructions = redactNavigationText(input.Instructions)
	output.Trigger = redactNavigationText(input.Trigger)
	output.InterventionType = redactNavigationText(input.InterventionType)
	return &output
}

var (
	sensitiveNavigationSelector   = regexp.MustCompile(`(?i)(password|passwd|passcode|secret|token|otp|one[-_ ]?time|verification[-_ ]?code|cvv|cvc|card[-_ ]?(number|security))`)
	sensitiveNavigationAssignment = regexp.MustCompile(`(?i)\b(password|passwd|passcode|secret|token|otp|one[-_ ]?time[-_ ]?code|verification[-_ ]?code|cvv|cvc|api[-_ ]?key|access[-_ ]?token|refresh[-_ ]?token)\b\s*[:=]\s*[^\s,;]+`)
)

// redactNavigationAction returns a detached action safe for browser-facing
// events and status history. It intentionally preserves action shape,
// selectors, and non-sensitive values so AI playback remains useful.
func redactNavigationAction(action map[string]interface{}) map[string]interface{} {
	if action == nil {
		return nil
	}
	return redactNavigationMap(action, navigationMapSensitive(action))
}

func redactNavigationMap(input map[string]interface{}, sensitive bool) map[string]interface{} {
	output := make(map[string]interface{}, len(input))
	for key, value := range input {
		lower := compactNavigationKey(key)
		switch {
		case isSensitiveNavigationKey(lower):
			output[key] = redactedNavigationValue
		case isNavigationURLKey(lower):
			if raw, ok := value.(string); ok {
				output[key] = redactNavigationURL(raw)
			} else {
				output[key] = redactNavigationValue(value, sensitive)
			}
		case sensitive && isNavigationValueKey(lower):
			output[key] = redactNavigationValue(value, true)
		case isNavigationValueKey(lower):
			// Generic action details (for example evaluate results) may carry a
			// credential assignment even when the selector does not identify a
			// sensitive field. Redact assignment-shaped secrets while preserving
			// ordinary values and structural nested data.
			if raw, ok := value.(string); ok {
				output[key] = redactNavigationText(raw)
			} else {
				output[key] = redactNavigationValue(value, false)
			}
		case isNavigationTextKey(lower):
			if raw, ok := value.(string); ok {
				output[key] = redactNavigationText(raw)
			} else {
				output[key] = redactNavigationValue(value, false)
			}
		default:
			// Preserve structural fields such as type and selector even when
			// the selector identifies a sensitive field. Nested maps compute
			// their own sensitivity from their keys and metadata.
			output[key] = redactNavigationValue(value, false)
		}
	}
	return output
}

func redactNavigationValue(value interface{}, sensitive bool) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		return redactNavigationMap(typed, sensitive || navigationMapSensitive(typed))
	case []interface{}:
		items := make([]interface{}, len(typed))
		for i, item := range typed {
			items[i] = redactNavigationValue(item, sensitive)
		}
		return items
	case string:
		if sensitive {
			return redactedNavigationValue
		}
		return typed
	default:
		return value
	}
}

func navigationMapSensitive(input map[string]interface{}) bool {
	for key, value := range input {
		lower := compactNavigationKey(key)
		if isSensitiveNavigationKey(lower) {
			return true
		}
		if raw, ok := value.(string); ok && isNavigationSelectorKey(lower) && sensitiveNavigationSelector.MatchString(raw) {
			return true
		}
		switch nested := value.(type) {
		case map[string]interface{}:
			if navigationMapSensitive(nested) {
				return true
			}
		case []interface{}:
			for _, item := range nested {
				if child, ok := item.(map[string]interface{}); ok && navigationMapSensitive(child) {
					return true
				}
			}
		}
	}
	return false
}

func compactNavigationKey(key string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(key) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isSensitiveNavigationKey(key string) bool {
	switch key {
	case "password", "passwd", "passcode", "secret", "token", "accesstoken", "refreshtoken", "apikey", "clientsecret", "authorization", "cookie", "setcookie", "otp", "onetimecode", "verificationcode", "cvv", "cvc", "cardnumber", "cardsecuritycode", "securitycode", "pin", "credential":
		return true
	default:
		return false
	}
}

func isNavigationValueKey(key string) bool {
	switch key {
	case "value", "text", "input", "query", "key", "content", "body", "result":
		return true
	default:
		return false
	}
}

func isNavigationSelectorKey(key string) bool {
	switch key {
	case "selector", "ref", "refid", "element", "elementid", "name", "id":
		return true
	default:
		return false
	}
}

func isNavigationURLKey(key string) bool {
	switch key {
	case "url", "currenturl", "finalurl":
		return true
	default:
		return false
	}
}

func isNavigationTextKey(key string) bool {
	switch key {
	case "reasoning", "description", "instructions", "error", "summary":
		return true
	default:
		return false
	}
}

func redactNavigationURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return redactNavigationText(raw)
	}
	if parsed.User != nil {
		parsed.User = url.UserPassword(redactedNavigationValue, redactedNavigationValue)
	}
	query := parsed.Query()
	changed := false
	for key := range query {
		if isSensitiveNavigationKey(compactNavigationKey(key)) {
			query.Set(key, redactedNavigationValue)
			changed = true
		}
	}
	if changed {
		parsed.RawQuery = query.Encode()
	}
	return parsed.String()
}

func redactNavigationText(text string) string {
	return sensitiveNavigationAssignment.ReplaceAllString(text, `${1}=[REDACTED]`)
}

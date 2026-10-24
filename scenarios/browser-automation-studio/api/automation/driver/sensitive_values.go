package driver

import (
	"strings"

	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
)

var sensitiveAutocompleteTokens = map[string]struct{}{
	"current-password": {},
	"new-password":     {},
	"one-time-code":    {},
	"cc-name":          {},
	"cc-number":        {},
	"cc-exp":           {},
	"cc-exp-month":     {},
	"cc-exp-year":      {},
	"cc-csc":           {},
}

// RedactSensitiveValues strips field values from an action identified as a
// password, hidden, one-time-code, or payment input. It preserves the action,
// selector, and field classification metadata for workflow structure.
func RedactSensitiveValues(action *RecordedAction) bool {
	if action == nil || action.ElementMeta == nil {
		return false
	}
	meta := action.ElementMeta
	return RedactSensitiveFields(meta.TagName, &meta.InnerText, meta.Attributes, action.Payload)
}

// RedactSensitiveFields clears secret-bearing content from metadata and payload
// maps while retaining the field classification and other action semantics.
func RedactSensitiveFields(tagName string, innerText *string, attributes map[string]string, payload map[string]interface{}) bool {
	tagName = strings.ToLower(tagName)
	if tagName != "input" && tagName != "textarea" {
		return false
	}
	typeValue := ""
	autocomplete := ""
	for name, value := range attributes {
		switch strings.ToLower(name) {
		case "type":
			typeValue = strings.ToLower(value)
		case "autocomplete":
			autocomplete = strings.ToLower(value)
		}
	}
	if typeValue != "password" && typeValue != "hidden" && !hasSensitiveAutocomplete(autocomplete) {
		return false
	}

	if innerText != nil {
		*innerText = ""
	}
	for name := range attributes {
		lower := strings.ToLower(name)
		if lower == "value" || strings.HasPrefix(lower, "data-") {
			delete(attributes, name)
		}
	}
	for name := range payload {
		if lower := strings.ToLower(name); lower == "text" || lower == "value" {
			payload[name] = ""
		}
	}
	return true
}

// RedactSensitiveTimelineEntry applies the same boundary to typed entries
// received from the driver, including entries buffered before a deployment.
func RedactSensitiveTimelineEntry(entry *bastimeline.TimelineEntry) bool {
	if entry == nil || entry.GetAction() == nil {
		return false
	}
	action := entry.GetAction()
	meta := action.GetMetadata().GetElementSnapshot()
	if meta == nil {
		return false
	}
	innerText := meta.GetInnerText()
	if !RedactSensitiveFields(meta.GetTagName(), &innerText, meta.GetAttributes(), nil) {
		return false
	}
	meta.InnerText = ""
	if input := action.GetInput(); input != nil {
		input.Value = ""
	}
	return true
}

func hasSensitiveAutocomplete(value string) bool {
	for _, token := range strings.Fields(value) {
		if _, ok := sensitiveAutocompleteTokens[token]; ok {
			return true
		}
	}
	return false
}

package validationrunner

import (
	"strings"
)

func statusFromSuccess(success bool) string {
	if success {
		return "passed"
	}
	return "failed"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func normalizeFindingKey(templateID, key string) string {
	key = strings.TrimSpace(strings.ToLower(key))
	key = strings.NewReplacer(" ", "-", "/", ".", ":", ".", "_", "-").Replace(key)
	if strings.HasPrefix(key, templateID+".") {
		return key
	}
	return templateID + "." + key
}

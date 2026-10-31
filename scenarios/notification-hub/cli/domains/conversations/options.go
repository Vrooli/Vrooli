package conversations

import (
	"fmt"
	"strings"
	"time"
)

type askOption struct{ Key, Label string }

// parseOptions turns repeated --option values (key or key=label) into
// structured answers; a missing label shows the key.
func parseOptions(values []string) ([]askOption, error) {
	seen := make(map[string]bool, len(values))
	var options []askOption
	for _, value := range values {
		key, label, _ := strings.Cut(value, "=")
		key = strings.TrimSpace(key)
		label = strings.TrimSpace(label)
		if key == "" {
			return nil, fmt.Errorf("--option %q needs a key (key or key=label)", value)
		}
		if seen[key] {
			return nil, fmt.Errorf("--option key %q is repeated", key)
		}
		seen[key] = true
		if label == "" {
			label = key
		}
		options = append(options, askOption{Key: key, Label: label})
	}
	if len(options) < 2 {
		return nil, fmt.Errorf("an ask needs at least two --option values")
	}
	return options, nil
}

// parseDeadline accepts an RFC 3339 time or a positive duration from now
// (for example 24h) and returns RFC 3339 for the API.
func parseDeadline(value string, now time.Time) (string, error) {
	value = strings.TrimSpace(value)
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC().Format(time.RFC3339), nil
	}
	if duration, err := time.ParseDuration(value); err == nil && duration > 0 {
		return now.Add(duration).UTC().Format(time.RFC3339), nil
	}
	return "", fmt.Errorf("--deadline %q must be an RFC 3339 time or a positive duration such as 24h", value)
}

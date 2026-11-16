package conversations

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type askOption struct{ Key, Label string }

// maxOptions is the most options the Hub accepts for one ask (maxAskOptions in
// api/internal/hub/asks.go). The Hub refuses more; saying so here saves the trip.
const maxOptions = 8

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
	if len(options) < 2 || len(options) > maxOptions {
		return nil, fmt.Errorf("an ask needs 2 to %d --option values, got %d", maxOptions, len(options))
	}
	return options, nil
}

// parseTime reads the value of a time flag (--expires or --timeout): an RFC 3339
// time, or a positive duration from now such as 24h or 7d (whole days, because
// "answer within 7 days" is how decisions are usually written). It returns
// RFC 3339 for the API; flag names the flag in the error.
func parseTime(flag, value string, now time.Time) (string, error) {
	value = strings.TrimSpace(value)
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC().Format(time.RFC3339), nil
	}
	if duration, err := time.ParseDuration(value); err == nil && duration > 0 {
		return now.Add(duration).UTC().Format(time.RFC3339), nil
	}
	if days, ok := strings.CutSuffix(value, "d"); ok {
		if n, err := strconv.Atoi(days); err == nil && n > 0 && n <= maxDays {
			return now.AddDate(0, 0, n).UTC().Format(time.RFC3339), nil
		}
	}
	return "", fmt.Errorf("%s %q must be an RFC 3339 time or a positive duration such as 24h or 7d", flag, value)
}

// maxDays bounds a duration written in days to one year.
const maxDays = 366

// waitGrace lets the server answer its own deadline before the client gives up.
const waitGrace = time.Minute

// waitContext bounds a Wait call by the parsed RFC 3339 deadline plus a short
// grace, so the long-poll is never cut off by a fixed client timeout and never
// outlives the deadline the caller asked for.
func waitContext(deadline string) (context.Context, context.CancelFunc) {
	at, err := time.Parse(time.RFC3339, deadline)
	if err != nil {
		return context.WithCancel(context.Background())
	}
	return context.WithDeadline(context.Background(), at.Add(waitGrace))
}

// parseReassertWithin turns --reassert-within (a duration) into the whole
// seconds the API takes. It needs --subject-key; the hub enforces the bounds.
func parseReassertWithin(value, subjectKey string) (int32, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	if strings.TrimSpace(subjectKey) == "" {
		return 0, fmt.Errorf("--reassert-within needs --subject-key")
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 || duration%time.Second != 0 {
		return 0, fmt.Errorf("--reassert-within %q must be a positive whole-second duration such as 15m", value)
	}
	if duration > 30*24*time.Hour {
		return 0, fmt.Errorf("--reassert-within %q is longer than 30 days", value)
	}
	return int32(duration / time.Second), nil
}

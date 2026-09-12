package apijson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const emptyBodyMessage = "empty response body: test-genie API may have restarted or closed the connection before replying"

// Parse decodes a JSON API response and upgrades the common empty-body failure
// into a clearer transport diagnosis for CLI callers.
func Parse[T any](body []byte, action string) (T, error) {
	var zero T
	if len(bytes.TrimSpace(body)) == 0 {
		return zero, errors.New(emptyBodyMessage)
	}

	var result T
	if err := json.Unmarshal(body, &result); err != nil {
		action = strings.TrimSpace(action)
		if action == "" {
			action = "parse response"
		}
		return zero, fmt.Errorf("%s: %w", action, err)
	}

	return result, nil
}

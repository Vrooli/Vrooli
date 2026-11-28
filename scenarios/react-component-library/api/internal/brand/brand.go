// Package brand exposes the React Component Library's Brand Manager record to
// the workbench shell.
//
// Brand decisions are owned by the operator in Brand Manager. This package
// only resolves the assignment and projects the small identity surface the
// shell needs; it never writes a brand and never hardcodes a colour.
package brand

import (
	"context"
	"strings"
)

// ScenarioName is the assignment key Brand Manager stores for this scenario.
const ScenarioName = "react-component-library"

// Brand is the wire shape the UI consumes. Every optional field is an
// explicit null rather than an empty string, so the client can tell "no
// brand" from "brand without a mark".
type Brand struct {
	Assigned bool    `json:"assigned"`
	Name     string  `json:"name"`
	Mark     *string `json:"mark"`
	Accent   *string `json:"accent"`
	Wordmark *string `json:"wordmark"`
}

// Resolver resolves the scenario's brand. An unavailable Brand Manager or a
// scenario with no assignment both resolve to Brand{} with a nil error: the
// shell's neutral fallback is a normal state, not a failure.
type Resolver interface {
	Resolve(ctx context.Context) (Brand, error)
}

// stringPtr returns a pointer for a non-whitespace value, else nil.
func stringPtr(raw string) *string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	return &value
}

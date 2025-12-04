package lighthouse

import (
	"context"
	"encoding/json"
)

// Client defines the interface for running Lighthouse audits.
// The primary implementation is CLIRunner which uses Google's official Lighthouse CLI.
type Client interface {
	// Health checks if the Lighthouse runner is available.
	Health(ctx context.Context) error

	// Audit runs a Lighthouse audit for the given URL and returns the result.
	Audit(ctx context.Context, req AuditRequest) (*AuditResponse, error)
}

// AuditRequest is the request payload for a Lighthouse audit.
type AuditRequest struct {
	// URL is the full URL to audit.
	URL string `json:"url"`

	// Config is the Lighthouse configuration.
	Config map[string]interface{} `json:"config,omitempty"`

	// GotoOptions contains navigation options for the page load.
	GotoOptions *GotoOptions `json:"gotoOptions,omitempty"`

	// IncludeHTML requests HTML report generation in addition to JSON.
	IncludeHTML bool `json:"-"`
}

// GotoOptions contains options for page navigation before the Lighthouse audit.
type GotoOptions struct {
	// WaitUntil specifies when to consider navigation succeeded.
	// Options: "load", "domcontentloaded", "networkidle0", "networkidle2"
	WaitUntil string `json:"waitUntil,omitempty"`

	// Timeout is the maximum navigation time in milliseconds.
	Timeout int `json:"timeout,omitempty"`
}

// AuditResponse is the response from a Lighthouse audit.
type AuditResponse struct {
	// Categories contains the category scores.
	Categories map[string]CategoryResult `json:"categories"`

	// Audits contains detailed audit results (we don't parse all of these).
	Audits json.RawMessage `json:"audits,omitempty"`

	// Raw is the full JSON response for debugging and report generation.
	Raw json.RawMessage `json:"-"`

	// RawHTML is the HTML report if HTML output was requested.
	RawHTML []byte `json:"-"`
}

// CategoryResult represents a Lighthouse category result.
type CategoryResult struct {
	// ID is the category identifier (e.g., "performance").
	ID string `json:"id"`

	// Title is the human-readable name.
	Title string `json:"title"`

	// Score is the category score (0-1 scale, or null).
	Score *float64 `json:"score"`
}

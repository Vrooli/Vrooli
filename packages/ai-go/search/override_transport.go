package aisearch

import (
	"encoding/json"
	"fmt"
	"strings"
)

// override_transport.go is the ONE shared definition of how a query-time override
// request travels over HTTP between search-hub (which issues overrides during a
// sweep or live A/B) and a provider's search handler (which validates and applies
// them). Both sides import this package, so the header names and the payload
// shape cannot drift apart into two private, silently-incompatible contracts.
//
// The transport is two headers, NOT a body change: the provider's search request
// body is a provider-owned shape (cli-health's SearchRequest, KO's query) that
// search-hub renders from a generic descriptor template, so it has no room for a
// cross-cutting override block. Headers carry the cross-cutting concern cleanly
// and keep the public search body untouched for an ordinary (no-override,
// no-token) request.
//
// This file is deliberately net/http-free: it operates on the header VALUE
// strings, so each side reads/writes its own http.Header (server or client) and
// this package stays a pure read-path/contract library.

const (
	// ControlTokenHeader carries the per-provider control token minted by
	// search-hub at registration. A provider honors overrides only when this
	// header matches the token it cached from its own registration (plus its
	// per-environment experiment flag). Canonical (textproto) MIME header key.
	ControlTokenHeader = "X-Search-Control-Token"

	// OverridesHeader carries the JSON-encoded SearchOverrides (snake_case factor
	// keys; absent factors omitted). Present only when search-hub is actually
	// overriding a factor for this request.
	OverridesHeader = "X-Search-Overrides"
)

// MarshalOverridesHeader encodes overrides as the OverridesHeader value. A zero
// (all-unset) SearchOverrides encodes to "" so the caller can skip the header
// entirely rather than sending an empty object.
func MarshalOverridesHeader(o SearchOverrides) (string, error) {
	if o.IsZero() {
		return "", nil
	}
	b, err := json.Marshal(o)
	if err != nil {
		return "", fmt.Errorf("marshal search overrides: %w", err)
	}
	return string(b), nil
}

// ParseOverridesHeader decodes an OverridesHeader value into a SearchOverrides.
// An empty/whitespace value yields a zero SearchOverrides and no error (the
// "no overrides requested" case). A malformed value is an error the handler
// surfaces as ignore-with-telemetry — never a hard failure of the search.
func ParseOverridesHeader(value string) (SearchOverrides, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return SearchOverrides{}, nil
	}
	var o SearchOverrides
	if err := json.Unmarshal([]byte(value), &o); err != nil {
		return SearchOverrides{}, fmt.Errorf("parse search overrides header: %w", err)
	}
	return o, nil
}

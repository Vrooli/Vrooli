package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

// RenderBody substitutes the {{query}}, {{limit}}, and {{type}} placeholders in
// a provider's body_template. {{query}} and {{type}} sit inside JSON string
// quotes in the template, so they are inserted as JSON-escaped *inner* strings
// (no surrounding quotes); {{limit}} is a bare integer. The result is validated
// as JSON so a malformed template surfaces to the caller as an error rather than
// a confusing downstream parse failure.
//
// Shared by the router fan-out (internal/routing) and the eval runner
// (internal/eval) so both reach a provider's registered endpoint identically —
// no per-domain copy of the placeholder contract.
func RenderBody(tmpl, query string, limit int32, typ string) (string, error) {
	return RenderBodyWithScope(tmpl, query, limit, typ, "")
}

// RenderBodyWithScope additionally substitutes {{scope}}, {{scope_kind}}, and
// {{scope_value}} for providers whose public search surface accepts scoped
// queries. Scope values use the shared corpus syntax:
// ""|"global"|"scenario:<id>"|"path:<prefix>".
func RenderBodyWithScope(tmpl, query string, limit int32, typ, scope string) (string, error) {
	if strings.TrimSpace(tmpl) == "" {
		return "", fmt.Errorf("empty body_template")
	}
	scopeKind, scopeValue := splitScope(scope)
	out := tmpl
	out = strings.ReplaceAll(out, "{{query}}", jsonStringInner(query))
	out = strings.ReplaceAll(out, "{{limit}}", strconv.FormatInt(int64(limit), 10))
	out = strings.ReplaceAll(out, "{{type}}", jsonStringInner(typ))
	out = strings.ReplaceAll(out, "{{scope}}", jsonStringInner(strings.TrimSpace(scope)))
	out = strings.ReplaceAll(out, "{{scope_kind}}", jsonStringInner(scopeKind))
	out = strings.ReplaceAll(out, "{{scope_value}}", jsonStringInner(scopeValue))
	if !json.Valid([]byte(out)) {
		return "", fmt.Errorf("rendered body is not valid JSON")
	}
	return out, nil
}

func splitScope(scope string) (kind, value string) {
	scope = strings.TrimSpace(scope)
	switch {
	case scope == "", scope == "global":
		return "global", ""
	case strings.HasPrefix(scope, "scenario:"):
		return "scenario", strings.TrimSpace(strings.TrimPrefix(scope, "scenario:"))
	case strings.HasPrefix(scope, "path:"):
		return "path", strings.TrimSpace(strings.TrimPrefix(scope, "path:"))
	default:
		return "global", ""
	}
}

// jsonStringInner returns s JSON-escaped with the surrounding quotes stripped,
// so it can be dropped into a "{{placeholder}}" slot that already lives inside
// quotes in the template.
func jsonStringInner(s string) string {
	b, err := json.Marshal(s)
	if err != nil || len(b) < 2 {
		return ""
	}
	return string(b[1 : len(b)-1])
}

// HTTPMethod maps the descriptor's HttpMethod enum to a net/http verb,
// defaulting to POST (the dominant Connect/REST search shape).
func HTTPMethod(m registryv1.HttpMethod) string {
	if m == registryv1.HttpMethod_HTTP_METHOD_GET {
		return http.MethodGet
	}
	return http.MethodPost
}

// ApplyHeaders copies the descriptor's headers onto req, defaulting
// Content-Type to application/json when the descriptor omits it (every search
// endpoint the hub federates speaks JSON).
func ApplyHeaders(req *http.Request, headers map[string]string) {
	hasContentType := false
	for k, v := range headers {
		req.Header.Set(k, v)
		if strings.EqualFold(k, "Content-Type") {
			hasContentType = true
		}
	}
	if !hasContentType {
		req.Header.Set("Content-Type", "application/json")
	}
}

// Package uiselectors implements the portable @vrooli/ui-selectors manifest.
package uiselectors

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var tokenPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

const deferredPrefix = "@ui-selector:"

func EscapeCSSValue(value string) string {
	var out strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			out.WriteRune(r)
		} else {
			fmt.Fprintf(&out, "\\%x ", r)
		}
	}
	return out.String()
}

func escapeCSSString(value string) string {
	var out strings.Builder
	for _, r := range value {
		if r == '"' || r == '\\' || r < 32 || r == 127 {
			fmt.Fprintf(&out, "\\%x ", r)
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}

// Resolve validates named arguments before rendering. Values are typed in the
// manifest API; the textual reference adapter performs numeric parsing.
func Resolve(manifest map[string]interface{}, key string, values map[string]interface{}) (string, error) {
	if literals, ok := manifest["selectors"].(map[string]interface{}); ok {
		if entry, ok := literals[key].(map[string]interface{}); ok {
			if len(values) > 0 {
				return "", fmt.Errorf("selector %s received unknown parameters", key)
			}
			if s, ok := entry["selector"].(string); ok {
				return s, nil
			}
		}
	}
	dynamics, _ := manifest["dynamicSelectors"].(map[string]interface{})
	entry, ok := dynamics[key].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unknown selector %s", key)
	}
	params, _ := entry["params"].([]interface{})
	names := map[string]bool{}
	for _, raw := range params {
		p, ok := raw.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("invalid selector parameter")
		}
		name, _ := p["name"].(string)
		names[name] = true
		value, exists := values[name]
		if !exists {
			return "", fmt.Errorf("missing parameter %s", name)
		}
		switch p["type"] {
		case "number":
			n, ok := value.(float64)
			if !ok || math.IsNaN(n) || math.IsInf(n, 0) {
				return "", fmt.Errorf("parameter %s must be numeric", name)
			}
		case "enum":
			switch value.(type) {
			case string, float64:
			default:
				return "", fmt.Errorf("parameter %s must be an enum scalar", name)
			}
			options, _ := p["values"].([]interface{})
			found := false
			for _, option := range options {
				if (fmt.Sprintf("%T", option) == fmt.Sprintf("%T", value)) && option == value {
					found = true
					break
				}
			}
			if !found {
				return "", fmt.Errorf("parameter %s is outside its enum", name)
			}
		case "string":
			if _, ok := value.(string); !ok {
				return "", fmt.Errorf("parameter %s must be a string", name)
			}
		default:
			return "", fmt.Errorf("unknown parameter type")
		}
	}
	for name := range values {
		if !names[name] {
			return "", fmt.Errorf("unknown parameter %s", name)
		}
	}
	pattern, _ := entry["selectorPattern"].(string)
	idPattern, _ := entry["testIdPattern"].(string)
	if idPattern != "" {
		pattern = idPattern
	}
	if pattern == "" {
		return "", fmt.Errorf("selector %s has no pattern", key)
	}
	var renderErr error
	rendered := tokenPattern.ReplaceAllStringFunc(pattern, func(token string) string {
		name := tokenPattern.FindStringSubmatch(token)[1]
		value, ok := values[name]
		if !ok {
			renderErr = fmt.Errorf("undeclared pattern parameter %s", name)
			return ""
		}
		s := formatValue(value)
		if strings.ContainsRune(s, 0) {
			renderErr = fmt.Errorf("NUL in selector parameter")
			return ""
		}
		if idPattern != "" {
			return s
		}
		return EscapeCSSValue(s)
	})
	if renderErr != nil {
		return "", renderErr
	}
	if idPattern != "" {
		rendered = `[data-testid="` + escapeCSSString(rendered) + `"]`
	}
	return rendered, nil
}

type deferred struct {
	Manifest map[string]interface{} `json:"manifest"`
	Key      string                 `json:"key"`
	Args     map[string]string      `json:"args"`
	Suffix   string                 `json:"suffix"`
	JSQuote  byte                   `json:"jsQuote,omitempty"`
}

func resolveArgs(manifest map[string]interface{}, key string, args map[string]string) (string, error) {
	values := map[string]interface{}{}
	for k, v := range args {
		values[k] = v
	}
	dynamic, _ := manifest["dynamicSelectors"].(map[string]interface{})
	entry, _ := dynamic[key].(map[string]interface{})
	params, _ := entry["params"].([]interface{})
	for _, raw := range params {
		p, _ := raw.(map[string]interface{})
		name, _ := p["name"].(string)
		v, ok := args[name]
		if !ok {
			continue
		}
		numeric := p["type"] == "number"
		if p["type"] == "enum" {
			options, _ := p["values"].([]interface{})
			for _, option := range options {
				if option == v {
					numeric = false
					break
				}
				if _, ok := option.(float64); ok {
					numeric = true
				}
			}
		}
		if numeric {
			n, e := strconv.ParseFloat(v, 64)
			if e != nil {
				return "", e
			}
			values[name] = n
		}
	}
	return Resolve(manifest, key, values)
}

func deferReference(manifest map[string]interface{}, key string, args map[string]string, suffix string) string {
	// Freeze only the referenced definition into the execution plan, never the
	// entire application registry. Raw workflow parameters remain data until runtime.
	entry := map[string]interface{}{"dynamicSelectors": map[string]interface{}{key: manifest["dynamicSelectors"].(map[string]interface{})[key]}}
	data, _ := json.Marshal(deferred{Manifest: entry, Key: key, Args: args, Suffix: suffix})
	return deferredPrefix + base64.RawURLEncoding.EncodeToString(data) + "@"
}

// ResolveDeferred resolves frozen definitions before ordinary string
// interpolation. The supplied interpolator sees parameter values, not CSS.
func ResolveDeferred(text string, interpolate func(string) (string, error)) (string, error) {
	for strings.Contains(text, deferredPrefix) {
		start := strings.Index(text, deferredPrefix)
		tail := text[start+len(deferredPrefix):]
		end := strings.IndexByte(tail, '@')
		if end < 0 {
			return "", fmt.Errorf("unterminated deferred selector")
		}
		raw, e := base64.RawURLEncoding.DecodeString(tail[:end])
		if e != nil {
			return "", e
		}
		var d deferred
		if e = json.Unmarshal(raw, &d); e != nil {
			return "", e
		}
		for k, v := range d.Args {
			value, err := interpolate(v)
			if err != nil {
				return "", fmt.Errorf("selector parameter %s: %w", k, err)
			}
			d.Args[k] = value
		}
		value, e := resolveArgs(d.Manifest, d.Key, d.Args)
		if e != nil {
			return "", e
		}
		value += d.Suffix
		if d.JSQuote != 0 {
			value = EscapeExpressionSelector(value, d.JSQuote)
		}
		text = text[:start] + value + tail[end+1:]
	}
	return text, nil
}

// EscapeExpressionSelector encodes CSS as the contents of a JavaScript string.
// Deferred references carry the quote context until their values are known.
func EscapeExpressionSelector(value string, quote byte) string {
	if strings.HasPrefix(value, deferredPrefix) && strings.HasSuffix(value, "@") {
		raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSuffix(strings.TrimPrefix(value, deferredPrefix), "@"))
		var d deferred
		if err == nil && json.Unmarshal(raw, &d) == nil {
			d.JSQuote = quote
			raw, _ = json.Marshal(d)
			return deferredPrefix + base64.RawURLEncoding.EncodeToString(raw) + "@"
		}
	}
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, string(quote), `\`+string(quote))
	value = strings.NewReplacer("\n", `\n`, "\r", `\r`, "\u2028", `\u2028`, "\u2029", `\u2029`).Replace(value)
	if quote == '`' {
		value = strings.ReplaceAll(value, "${", `\${`)
	}
	return value
}

// Match JavaScript String(number), including its fixed/exponent thresholds.
func formatValue(value interface{}) string {
	n, ok := value.(float64)
	if !ok {
		return fmt.Sprint(value)
	}
	a := math.Abs(n)
	if n == 0 {
		return "0"
	}
	if a >= 1e-6 && a < 1e21 {
		return strconv.FormatFloat(n, 'f', -1, 64)
	}
	parts := strings.Split(strconv.FormatFloat(n, 'e', -1, 64), "e")
	exponent, _ := strconv.Atoi(parts[1])
	return fmt.Sprintf("%se%+d", parts[0], exponent)
}

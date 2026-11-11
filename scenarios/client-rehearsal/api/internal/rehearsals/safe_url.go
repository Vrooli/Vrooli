package rehearsals

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

type safeURLExpressionPart struct {
	literal   string
	nodeName  string
	field     string
	transform string
}

var safeURLReference = regexp.MustCompile(`^\$\('([^']+)'\)\.first\(\)\.json\.([A-Za-z][A-Za-z0-9_]*)(?:\.replace\(/(.*)/,\s*'([^']*)'\))?$`)

// parseSafeSyntheticURLExpression accepts concatenations of fixed strings and
// one statically named prior-node scalar, with only two URL cleanup transforms.
// It is a small data expression grammar; it never evaluates JavaScript.
func parseSafeSyntheticURLExpression(raw string) ([]safeURLExpressionPart, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "={{") || !strings.HasSuffix(raw, "}}") {
		return nil, fmt.Errorf("expression wrapper is not canonical")
	}
	body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(raw, "={{"), "}}"))
	terms, err := splitSafeURLTerms(body)
	if err != nil || len(terms) == 0 || len(terms) > 16 {
		return nil, fmt.Errorf("expression must contain 1 to 16 bounded concatenation terms")
	}
	parts := make([]safeURLExpressionPart, 0, len(terms))
	for _, term := range terms {
		term = strings.TrimSpace(term)
		if len(term) >= 2 && term[0] == '\'' && term[len(term)-1] == '\'' {
			literal := term[1 : len(term)-1]
			if strings.ContainsAny(literal, "\\\x00\r\n") {
				return nil, fmt.Errorf("URL string literals must not contain escapes or controls")
			}
			parts = append(parts, safeURLExpressionPart{literal: literal})
			continue
		}
		match := safeURLReference.FindStringSubmatch(term)
		if match == nil || len(match[1]) == 0 || len(match[1]) > 128 {
			return nil, fmt.Errorf("expression term is not a fixed literal or static node-field read")
		}
		part := safeURLExpressionPart{nodeName: match[1], field: match[2]}
		if match[3] != "" {
			switch match[3] {
			case `\/+$`:
				if match[4] != "" {
					return nil, fmt.Errorf("trailing slash cleanup replacement must be empty")
				}
				part.transform = "trim-trailing-slashes"
			case `^https?:\/\/`:
				if match[4] != "" {
					return nil, fmt.Errorf("scheme removal replacement must be empty")
				}
				part.transform = "strip-http-scheme"
			default:
				return nil, fmt.Errorf("regular expression cleanup is unsupported")
			}
		}
		parts = append(parts, part)
	}
	return parts, nil
}

func splitSafeURLTerms(expression string) ([]string, error) {
	terms := make([]string, 0, 4)
	start, depth := 0, 0
	var quote byte
	escaped := false
	for index := 0; index < len(expression); index++ {
		ch := expression[index]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
			continue
		}
		if ch == '(' {
			depth++
			continue
		}
		if ch == ')' {
			depth--
			if depth < 0 {
				return nil, fmt.Errorf("unbalanced parentheses")
			}
			continue
		}
		if ch == '+' && depth == 0 {
			terms = append(terms, expression[start:index])
			start = index + 1
		}
	}
	if quote != 0 || depth != 0 {
		return nil, fmt.Errorf("unterminated quote or parenthesis")
	}
	terms = append(terms, expression[start:])
	return terms, nil
}

func resolveSafeSyntheticURLExpression(raw string, prior map[string][]map[string]any) (string, error) {
	parts, err := parseSafeSyntheticURLExpression(raw)
	if err != nil {
		return "", err
	}
	var result strings.Builder
	for _, part := range parts {
		if part.nodeName == "" {
			result.WriteString(part.literal)
			continue
		}
		outputs := prior[part.nodeName]
		if len(outputs) == 0 {
			return "", fmt.Errorf("URL expression prior node %q has no output", part.nodeName)
		}
		value, ok := outputs[0][part.field].(string)
		if !ok || value == "" || strings.ContainsAny(value, "\x00\r\n") {
			return "", fmt.Errorf("URL expression prior field %q is not a bounded string", part.field)
		}
		switch part.transform {
		case "trim-trailing-slashes":
			value = strings.TrimRight(value, "/")
		case "strip-http-scheme":
			value = strings.TrimPrefix(value, "https://")
			value = strings.TrimPrefix(value, "http://")
		}
		result.WriteString(value)
		if result.Len() > 2048 {
			return "", fmt.Errorf("resolved URL exceeds the bound")
		}
	}
	resolved := result.String()
	parsed, err := url.Parse(resolved)
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.User != nil || parsed.Port() != "" || parsed.RawQuery != "" || parsed.Fragment != "" || !strings.HasSuffix(strings.ToLower(parsed.Hostname()), ".synthetic.invalid") {
		return "", fmt.Errorf("resolved URL must target an exact synthetic.invalid host without credentials, port, query or fragment")
	}
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	for _, segment := range strings.Split(parsed.EscapedPath(), "/") {
		if segment == "." || segment == ".." || strings.Contains(segment, "%") {
			return "", fmt.Errorf("resolved URL path contains an unsafe segment")
		}
	}
	return parsed.String(), nil
}

func requestWithSafeURLExpression(base Step, expression string, prior map[string][]map[string]any) (Step, error) {
	resolved, err := resolveSafeSyntheticURLExpression(expression, prior)
	if err != nil {
		return Step{}, err
	}
	parsed, err := url.Parse(resolved)
	if err != nil {
		return Step{}, err
	}
	base.Host = parsed.Hostname()
	base.URL = parsed.EscapedPath()
	if base.URL == "" {
		base.URL = "/"
	}
	base.Operation = base.Method + " " + base.URL
	return base, nil
}

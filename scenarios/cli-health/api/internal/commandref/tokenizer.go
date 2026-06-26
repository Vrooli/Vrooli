package commandref

import (
	"fmt"
	"strings"
	"unicode"
)

type shellSyntaxError struct {
	Kind   string
	Symbol string
}

func (e shellSyntaxError) Error() string {
	if e.Symbol == "" {
		return "unsupported shell " + e.Kind
	}
	return fmt.Sprintf("unsupported shell %s %q", e.Kind, e.Symbol)
}

func tokenizeCommand(s string) ([]string, error) {
	var out []string
	var b strings.Builder
	var quote rune
	escaped := false

	flush := func() {
		if b.Len() == 0 {
			return
		}
		out = append(out, b.String())
		b.Reset()
	}

	for _, r := range s {
		if escaped {
			b.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
				continue
			}
			b.WriteRune(r)
			continue
		}
		switch r {
		case '<', '>':
			return nil, shellSyntaxError{Kind: "redirection", Symbol: string(r)}
		case '|', ';', '&', '`':
			return nil, shellSyntaxError{Kind: "syntax", Symbol: string(r)}
		case '\n', '\r':
			return nil, shellSyntaxError{Kind: "syntax", Symbol: "newline"}
		case '$':
			return nil, shellSyntaxError{Kind: "expansion"}
		case '\'', '"':
			quote = r
		default:
			if unicode.IsSpace(r) {
				flush()
				continue
			}
			b.WriteRune(r)
		}
	}
	if escaped {
		b.WriteRune('\\')
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote")
	}
	flush()
	return out, nil
}

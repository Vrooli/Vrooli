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

// unquotedGroup records one leniently parsed unquoted <...> placeholder group
// (text includes the brackets).
type unquotedGroup struct {
	Text string
}

// tokenizeCommandDocs is the DOCS-policy tokenizer: a balanced, non-nested
// <...> group outside quotes is consumed as a single placeholder token (pipes
// inside the group included) instead of hard-failing as shell redirection.
// Every other shell operator outside such a group — bare `>`, `|`, `;`, `&`,
// `$`, backtick — remains a shellSyntaxError exactly as in tokenizeCommand,
// and nested or unterminated `<` groups are still hard errors.
//
// The returned fixed string is the byte-exact preferred form: the same snippet
// with every unquoted group wrapped in double quotes. It is empty when no
// lenient group was consumed.
func tokenizeCommandDocs(s string) (tokens []string, groups []unquotedGroup, fixed string, err error) {
	var out []string
	var b strings.Builder
	var quote rune
	escaped := false
	runes := []rune(s)
	var groupBounds [][2]int // [start, end] rune indices, inclusive of brackets

	flush := func() {
		if b.Len() == 0 {
			return
		}
		out = append(out, b.String())
		b.Reset()
	}

	for i := 0; i < len(runes); i++ {
		r := runes[i]
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
		case '<':
			start := i
			end := -1
			for j := i + 1; j < len(runes); j++ {
				switch runes[j] {
				case '<':
					return nil, nil, "", shellSyntaxError{Kind: "redirection", Symbol: "<"}
				case '\n', '\r':
					return nil, nil, "", shellSyntaxError{Kind: "redirection", Symbol: "<"}
				case '>':
					end = j
				}
				if end != -1 {
					break
				}
			}
			if end == -1 || end == i+1 {
				return nil, nil, "", shellSyntaxError{Kind: "redirection", Symbol: "<"}
			}
			group := string(runes[start : end+1])
			b.WriteString(group)
			groupBounds = append(groupBounds, [2]int{start, end})
			groups = append(groups, unquotedGroup{Text: group})
			i = end
		case '>':
			return nil, nil, "", shellSyntaxError{Kind: "redirection", Symbol: string(r)}
		case '|', ';', '&', '`':
			return nil, nil, "", shellSyntaxError{Kind: "syntax", Symbol: string(r)}
		case '\n', '\r':
			return nil, nil, "", shellSyntaxError{Kind: "syntax", Symbol: "newline"}
		case '$':
			return nil, nil, "", shellSyntaxError{Kind: "expansion"}
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
		return nil, nil, "", fmt.Errorf("unterminated quote")
	}
	flush()

	if len(groupBounds) > 0 {
		var fb strings.Builder
		prev := 0
		for _, bounds := range groupBounds {
			fb.WriteString(string(runes[prev:bounds[0]]))
			fb.WriteByte('"')
			fb.WriteString(string(runes[bounds[0] : bounds[1]+1]))
			fb.WriteByte('"')
			prev = bounds[1] + 1
		}
		fb.WriteString(string(runes[prev:]))
		fixed = fb.String()
	}
	return out, groups, fixed, nil
}

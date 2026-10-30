package runsignal

import (
	"regexp"
	"strconv"
	"strings"

	"agent-manager/internal/domain"
)

// Codex's `exec` tool takes JavaScript, e.g.
//
//	text(await tools.exec_command({cmd:"rg -n foo", workdir:"/x"}))
//
// The JavaScript is not shell. Only the literal cmd/command argument of each
// tools.exec_command call is a command; whatever else the script does is
// code.

// jsCommand is one literal exec_command argument: shell text or argv.
type jsCommand struct {
	Script string
	Args   []string
}

var jsOtherToolCall = regexp.MustCompile(`\btools\.[A-Za-z_$][A-Za-z0-9_$]*\s*\(|\bfor\s*\(|\bwhile\s*\(|\.(?:map|flatMap|forEach|reduce|filter)\s*\(|\bfetch\s*\(|\bimport\s*\(`)

// jsExecInput returns the JavaScript source of a Codex exec call.
func jsExecInput(call *domain.ToolCallEventData) (string, bool) {
	if call == nil {
		return "", false
	}
	src, ok := call.Input["input"].(string)
	if !ok || strings.TrimSpace(src) == "" {
		return "", false
	}
	name := strings.ToLower(call.ToolName)
	if name == "exec" || name == "exec_command" || strings.Contains(src, "tools.") {
		return src, true
	}
	return "", false
}

// jsExecCommands extracts the literal commands of every exec_command call.
// unresolved counts calls whose command is not a literal, and other reports
// whether the remaining script does anything beyond running those calls.
func jsExecCommands(src string) (commands []jsCommand, unresolved int, other bool) {
	masked := maskJS(src)
	rest := []byte(masked)
	for from := 0; ; {
		at := strings.Index(masked[from:], "exec_command")
		if at < 0 {
			break
		}
		start := from + at
		pos := skipJSSpace(src, start+len("exec_command"))
		end := pos
		if pos < len(src) && src[pos] == '(' {
			pos = skipJSSpace(src, pos+1)
			if pos < len(src) && src[pos] == '{' {
				var command jsCommand
				var ok bool
				command, end, ok = parseJSCommandObject(src, pos)
				if ok {
					commands = append(commands, command)
				} else {
					unresolved++
				}
			} else {
				unresolved++
			}
		} else {
			unresolved++
		}
		if end <= start {
			end = start + len("exec_command")
		}
		for i := start; i < end && i < len(rest); i++ {
			rest[i] = ' '
		}
		from = end
	}
	return commands, unresolved, jsOtherToolCall.Match(rest)
}

// skipJSSpace skips whitespace and comments.
func skipJSSpace(s string, i int) int {
	for i < len(s) {
		if strings.IndexByte(" \t\r\n", s[i]) >= 0 {
			i++
		} else if end := jsCommentEnd(s, i); end > i {
			i = end
		} else {
			break
		}
	}
	return i
}

// jsCommentEnd is the index after the // or /* */ comment starting at s[i], or
// i when none starts there.
func jsCommentEnd(s string, i int) int {
	if i+1 >= len(s) || s[i] != '/' {
		return i
	}
	switch s[i+1] {
	case '/':
		if nl := strings.IndexByte(s[i:], '\n'); nl >= 0 {
			return i + nl
		}
		return len(s)
	case '*':
		if end := strings.Index(s[i+2:], "*/"); end >= 0 {
			return i + 2 + end + 2
		}
		return len(s)
	}
	return i
}

// maskJS blanks comments and the contents of string literals, keeping offsets,
// so a search for code finds only code: an apostrophe in a comment starts no
// string and a quoted "exec_command" is no call.
func maskJS(src string) string {
	out := []byte(src)
	blank := func(from, to int) {
		for k := from; k < to && k < len(out); k++ {
			if out[k] != '\n' {
				out[k] = ' '
			}
		}
	}
	for i := 0; i < len(src); {
		switch c := src[i]; {
		case c == '/' && jsCommentEnd(src, i) > i:
			end := jsCommentEnd(src, i)
			blank(i, end)
			i = end
		case c == '"' || c == '\'' || c == '`':
			_, end, _, _ := readJSString(src, i)
			blank(i+1, end-1)
			i = end
		default:
			i++
		}
	}
	return string(out)
}

// parseJSCommandObject parses the object literal at s[i] and returns its cmd
// or command value. end is the index after the object (or where scanning
// stopped).
func parseJSCommandObject(s string, i int) (command jsCommand, end int, ok bool) {
	i++
	for {
		i = skipJSSpace(s, i)
		for i < len(s) && s[i] == ',' {
			i = skipJSSpace(s, i+1)
		}
		if i >= len(s) {
			return command, i, ok
		}
		if s[i] == '}' {
			return command, i + 1, ok
		}
		var key string
		if s[i] == '"' || s[i] == '\'' || s[i] == '`' {
			value, next, _, strOK := readJSString(s, i)
			if !strOK {
				return command, len(s), ok
			}
			key, i = value, next
		} else {
			start := i
			for i < len(s) && (s[i] == '_' || s[i] == '$' || s[i] >= '0' && s[i] <= '9' || s[i] >= 'a' && s[i] <= 'z' || s[i] >= 'A' && s[i] <= 'Z') {
				i++
			}
			if i == start {
				return command, len(s), ok
			}
			key = s[start:i]
		}
		i = skipJSSpace(s, i)
		if i >= len(s) || s[i] != ':' {
			continue // shorthand property
		}
		i = skipJSSpace(s, i+1)
		if (key == "cmd" || key == "command") && !ok {
			if value, next, valueOK := parseJSCommandValue(s, i); valueOK {
				command, ok, i = value, true, next
			}
		}
		i = skipJSValue(s, i)
	}
}

// parseJSCommandValue reads a string literal or an array of string literals.
func parseJSCommandValue(s string, i int) (jsCommand, int, bool) {
	if i >= len(s) {
		return jsCommand{}, i, false
	}
	switch s[i] {
	case '"', '\'', '`':
		value, next, subst, ok := readJSString(s, i)
		after := skipJSSpace(s, next)
		if !ok || subst || after < len(s) && (s[after] == '+' || s[after] == '.') {
			return jsCommand{}, i, false
		}
		return jsCommand{Script: value}, next, strings.TrimSpace(value) != ""
	case '[':
		var args []string
		j := i + 1
		for {
			j = skipJSSpace(s, j)
			if j >= len(s) {
				return jsCommand{}, i, false
			}
			if s[j] == ']' {
				return jsCommand{Args: args}, j + 1, len(args) > 0
			}
			if s[j] != '"' && s[j] != '\'' && s[j] != '`' {
				// A variable, spread or expression: the argv is not literal.
				return jsCommand{}, i, false
			}
			value, next, subst, ok := readJSString(s, j)
			if !ok || subst {
				return jsCommand{}, i, false
			}
			args = append(args, value)
			j = skipJSSpace(s, next)
			if j < len(s) && s[j] == ',' {
				j++
			}
		}
	}
	return jsCommand{}, i, false
}

// readJSString decodes the string literal at s[i]. subst reports a template
// literal with ${...}, whose value is not a literal.
func readJSString(s string, i int) (value string, end int, subst, ok bool) {
	quote := s[i]
	var b strings.Builder
	for j := i + 1; j < len(s); j++ {
		c := s[j]
		switch {
		case c == quote:
			return b.String(), j + 1, subst, true
		case c == '$' && quote == '`' && j+1 < len(s) && s[j+1] == '{':
			subst = true
			b.WriteByte(c)
		case c == '\\' && j+1 < len(s):
			j++
			switch e := s[j]; e {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '\n':
			case 'x', 'u':
				width := 2
				if e == 'u' {
					width = 4
				}
				if j+width < len(s) {
					if r, err := strconv.ParseUint(s[j+1:j+1+width], 16, 32); err == nil {
						b.WriteRune(rune(r))
						j += width
						continue
					}
				}
				b.WriteByte(e)
			default:
				b.WriteByte(e)
			}
		default:
			b.WriteByte(c)
		}
	}
	return "", len(s), subst, false
}

// skipJSValue advances to the comma or closing brace ending a property value.
func skipJSValue(s string, i int) int {
	depth := 0
	for i < len(s) {
		if end := jsCommentEnd(s, i); end > i {
			i = end
			continue
		}
		switch c := s[i]; c {
		case '"', '\'', '`':
			_, next, _, ok := readJSString(s, i)
			if !ok {
				return len(s)
			}
			i = next
			continue
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth == 0 {
				return i
			}
			depth--
		case ',':
			if depth == 0 {
				return i
			}
		}
		i++
	}
	return i
}

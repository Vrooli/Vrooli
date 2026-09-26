// Package utilityclass is the single authority for utility-class detection in
// published React Component Library source.
package utilityclass

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// Hit identifies a utility class that reaches a class-bearing JSX attribute.
type Hit struct {
	Class    string
	Offset   int
	Category string
}

var declarationStartRE = regexp.MustCompile(`\b(?:const|let)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=`)

var exactUtilities = map[string]string{
	"absolute": "layout", "block": "layout", "fixed": "layout", "flex": "layout",
	"grid": "layout", "hidden": "layout", "inline": "layout", "inline-block": "layout",
	"inline-flex": "layout", "relative": "layout", "sr-only": "layout",
	"touch-target": "sizing", "truncate": "typography", "uppercase": "typography",
	"whitespace-nowrap": "typography",
}

var utilityPrefixes = []struct {
	prefix   string
	category string
}{
	{"bg-", "palette"},
	{"border-", "border-effect"},
	{"divide-", "palette"},
	{"from-", "palette"},
	{"outline-", "border-effect"},
	{"ring-", "border-effect"},
	{"text-", "typography"},
	{"to-", "palette"},
	{"via-", "palette"},
	{"absolute", "layout"},
	{"bottom-", "layout"},
	{"end-", "layout"},
	{"flex-", "layout"},
	{"gap-", "spacing"},
	{"grid-", "layout"},
	{"inset-", "layout"},
	{"items-", "layout"},
	{"justify-", "layout"},
	{"left-", "layout"},
	{"right-", "layout"},
	{"start-", "layout"},
	{"top-", "layout"},
	{"z-", "layout"},
	{"m-", "spacing"},
	{"mb-", "spacing"},
	{"ml-", "spacing"},
	{"mr-", "spacing"},
	{"mt-", "spacing"},
	{"mx-", "spacing"},
	{"my-", "spacing"},
	{"p-", "spacing"},
	{"pb-", "spacing"},
	{"pl-", "spacing"},
	{"pr-", "spacing"},
	{"pt-", "spacing"},
	{"px-", "spacing"},
	{"py-", "spacing"},
	{"space-", "spacing"},
	{"h-", "sizing"},
	{"max-", "sizing"},
	{"min-", "sizing"},
	{"size-", "sizing"},
	{"w-", "sizing"},
	{"font-", "typography"},
	{"leading-", "typography"},
	{"tracking-", "typography"},
	{"rounded-", "border-effect"},
	{"shadow-", "border-effect"},
	{"opacity-", "border-effect"},
	{"transition", "border-effect"},
	{"overflow-", "layout"},
	{"pointer-events-", "layout"},
	{"translate-", "layout"},
	{"-translate-", "layout"},
}

// EmitsAny returns stable, de-duplicated utility hits from strings that reach
// className/class attributes. CSS strings, data attributes and comments are
// deliberately outside the scan boundary.
func EmitsAny(source string) []Hit {
	declarations := declarations(source)
	seenVariables := map[string]bool{}
	var literals []literal
	for offset := 0; offset < len(source); {
		name, next := classAttributeAt(source, offset)
		if name == "" {
			offset = next
			continue
		}
		offset = next
		for offset < len(source) && unicode.IsSpace(rune(source[offset])) {
			offset++
		}
		if offset >= len(source) || source[offset] != '=' {
			continue
		}
		offset++
		for offset < len(source) && unicode.IsSpace(rune(source[offset])) {
			offset++
		}
		if offset >= len(source) {
			break
		}
		if isQuote(source[offset]) {
			value, end := quoted(source, offset)
			literals = append(literals, literal{value: value, offset: offset + 1})
			offset = end
			continue
		}
		if source[offset] != '{' {
			continue
		}
		expression, end := balanced(source, offset)
		literals = append(literals, stringLiterals(expression, offset+1)...)
		queue := identifiers(expression)
		for len(queue) > 0 {
			identifier := queue[0]
			queue = queue[1:]
			if seenVariables[identifier] {
				continue
			}
			seenVariables[identifier] = true
			initializer, ok := declarations[identifier]
			if !ok {
				continue
			}
			literals = append(literals, stringLiterals(initializer.value, initializer.offset)...)
			queue = append(queue, identifiers(initializer.value)...)
		}
		offset = end
	}

	seen := map[string]bool{}
	var hits []Hit
	for _, item := range literals {
		cursor := 0
		for _, token := range strings.Fields(item.value) {
			index := strings.Index(item.value[cursor:], token)
			if index < 0 {
				index = 0
			}
			cursor += index
			clean := strings.Trim(token, "(),")
			category := categoryOf(clean)
			if category != "" && !seen[clean] {
				seen[clean] = true
				hits = append(hits, Hit{Class: clean, Offset: item.offset + cursor, Category: category})
			}
			cursor += len(token)
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Offset == hits[j].Offset {
			return hits[i].Class < hits[j].Class
		}
		return hits[i].Offset < hits[j].Offset
	})
	return hits
}

// UndefinedInConsumer returns emitted classes absent from a consumer's
// compiled-class set.
func UndefinedInConsumer(source string, consumerClasses map[string]struct{}) []Hit {
	var missing []Hit
	for _, hit := range EmitsAny(source) {
		if _, ok := consumerClasses[hit.Class]; !ok {
			missing = append(missing, hit)
		}
	}
	return missing
}

type literal struct {
	value  string
	offset int
}

func categoryOf(token string) string {
	token = strings.TrimPrefix(token, "!")
	if token == "" || strings.Contains(token, "${") || strings.ContainsAny(token, "{};'\"`") {
		return ""
	}
	base := token
	if colon := strings.LastIndex(base, ":"); colon >= 0 {
		base = strings.TrimPrefix(base[colon+1:], "!")
		if base == "" {
			return ""
		}
		if categoryOf(base) != "" {
			return "variant"
		}
	}
	if strings.Contains(base, "[") && strings.Contains(base, "]") {
		return "arbitrary"
	}
	if category, ok := exactUtilities[base]; ok {
		return category
	}
	for _, item := range utilityPrefixes {
		if strings.HasPrefix(base, item.prefix) {
			return item.category
		}
	}
	return ""
}

func declarations(source string) map[string]literal {
	result := map[string]literal{}
	for _, match := range declarationStartRE.FindAllStringSubmatchIndex(source, -1) {
		if len(match) < 4 {
			continue
		}
		start := match[1]
		end := declarationValueEnd(source, start)
		result[source[match[2]:match[3]]] = literal{value: source[start:end], offset: start}
	}
	return result
}

func declarationValueEnd(source string, start int) int {
	depth := 0
	quote := byte(0)
	escaped := false
	for index := start; index < len(source); index++ {
		character := source[index]
		if quote != 0 {
			if escaped {
				escaped = false
			} else if character == '\\' {
				escaped = true
			} else if character == quote {
				quote = 0
			}
			continue
		}
		if isQuote(character) {
			quote = character
			continue
		}
		switch character {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case ';', '\n':
			if depth == 0 {
				return index
			}
		}
	}
	return len(source)
}

func classAttributeAt(source string, offset int) (string, int) {
	for offset < len(source) {
		if strings.HasPrefix(source[offset:], "//") {
			if newline := strings.IndexByte(source[offset:], '\n'); newline >= 0 {
				offset += newline + 1
				continue
			}
			return "", len(source)
		}
		if strings.HasPrefix(source[offset:], "/*") {
			if end := strings.Index(source[offset+2:], "*/"); end >= 0 {
				offset += end + 4
				continue
			}
			return "", len(source)
		}
		for _, name := range []string{"className", "class"} {
			if strings.HasPrefix(source[offset:], name) && boundary(source, offset-1) && boundary(source, offset+len(name)) {
				return name, offset + len(name)
			}
		}
		offset++
	}
	return "", offset
}

func boundary(source string, index int) bool {
	if index < 0 || index >= len(source) {
		return true
	}
	r := rune(source[index])
	return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '$' || r == '-')
}

func balanced(source string, start int) (string, int) {
	depth := 0
	quote := byte(0)
	escaped := false
	for index := start; index < len(source); index++ {
		character := source[index]
		if quote != 0 {
			if escaped {
				escaped = false
			} else if character == '\\' {
				escaped = true
			} else if character == quote {
				quote = 0
			}
			continue
		}
		if isQuote(character) {
			quote = character
		} else if character == '{' {
			depth++
		} else if character == '}' {
			depth--
			if depth == 0 {
				return source[start+1 : index], index + 1
			}
		}
	}
	return source[start+1:], len(source)
}

func stringLiterals(source string, base int) []literal {
	var result []literal
	for index := 0; index < len(source); index++ {
		if !isQuote(source[index]) {
			continue
		}
		value, end := quoted(source, index)
		result = append(result, literal{value: value, offset: base + index + 1})
		index = end - 1
	}
	return result
}

func quoted(source string, start int) (string, int) {
	quote := source[start]
	escaped := false
	for index := start + 1; index < len(source); index++ {
		if escaped {
			escaped = false
			continue
		}
		if source[index] == '\\' {
			escaped = true
		} else if source[index] == quote {
			return source[start+1 : index], index + 1
		}
	}
	return source[start+1:], len(source)
}

func identifiers(source string) []string {
	seen := map[string]bool{}
	var result []string
	for index := 0; index < len(source); {
		r := rune(source[index])
		if !(unicode.IsLetter(r) || r == '_' || r == '$') {
			index++
			continue
		}
		start := index
		for index < len(source) {
			r = rune(source[index])
			if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '$') {
				break
			}
			index++
		}
		identifier := source[start:index]
		if !seen[identifier] {
			seen[identifier] = true
			result = append(result, identifier)
		}
	}
	return result
}

func isQuote(character byte) bool {
	return character == '\'' || character == '"' || character == '`'
}

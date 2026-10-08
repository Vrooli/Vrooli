package components

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
)

// DeriveStoryFields extracts the component-owned props that are useful to the
// safe preview controls. It intentionally reads the exported props declaration
// only; inherited DOM attributes and opaque generic types are outside the
// component's authored surface and are omitted.
func DeriveStoryFields(source string) []StoryField {
	body := propsBody(source)
	if body == "" {
		return []StoryField{}
	}
	lines := strings.Split(strings.ReplaceAll(body, ";", ";\n"), "\n")
	fields := make([]StoryField, 0, len(lines))
	var comment strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "/**") || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "//") {
			comment.WriteString(strings.TrimSpace(strings.TrimLeft(trimmed, "/*")))
			comment.WriteByte(' ')
			continue
		}
		match := propLineRE.FindStringSubmatch(trimmed)
		if len(match) == 0 {
			continue
		}
		name := match[1] + match[2]
		typeText := strings.TrimSpace(match[3])
		field, ok := fieldForType(name, typeText, comment.String())
		comment.Reset()
		if ok {
			fields = append(fields, field)
		}
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].Path < fields[j].Path })
	return fields
}

var (
	propsBodyRE    = regexp.MustCompile(`(?s)(?:export\s+)?interface\s+[A-Za-z_$][\w$]*Props(?:\s+extends[^\{]+)?\s*\{(.*?)\}`)
	propsTypeRE    = regexp.MustCompile(`(?s)(?:export\s+)?type\s+[A-Za-z_$][\w$]*Props\s*=\s*\{(.*?)\}`)
	propLineRE     = regexp.MustCompile(`^([A-Za-z_$][\w$]*)(\?)?\s*:\s*(.+?)(?:;|,)?$`)
	unionLiteralRE = regexp.MustCompile(`['"]([^'"]+)['"]`)
	defaultRE      = regexp.MustCompile(`@default\s+([^@]+)`)
)

func propsBody(source string) string {
	if match := propsBodyRE.FindStringSubmatch(source); len(match) == 2 {
		return match[1]
	}
	if match := propsTypeRE.FindStringSubmatch(source); len(match) == 2 {
		return match[1]
	}
	return ""
}

func fieldForType(name, typeText, comment string) (StoryField, bool) {
	path := strings.TrimSuffix(name, "?")
	if strings.EqualFold(path, "className") || strings.Contains(typeText, "HTMLAttributes") || strings.Contains(typeText, "ComponentProps") || strings.Contains(typeText, "=>") {
		return StoryField{}, false
	}
	field := StoryField{Path: path, Label: firstCommentSentence(comment), Required: true}
	if strings.HasSuffix(name, "?") {
		field.Path = path
		field.Required = false
	}
	if strings.Contains(typeText, "undefined") {
		field.Required = false
	}
	typeText = strings.TrimSpace(strings.TrimSuffix(typeText, ";"))
	if strings.Contains(typeText, "ReactNode") || strings.EqualFold(field.Path, "children") {
		field.Kind = StoryFieldText
	} else if strings.Contains(typeText, "|") && len(unionLiteralRE.FindAllStringSubmatch(typeText, -1)) > 0 {
		field.Kind = StoryFieldEnum
		for _, match := range unionLiteralRE.FindAllStringSubmatch(typeText, -1) {
			if raw, err := json.Marshal(match[1]); err == nil {
				field.Options = append(field.Options, raw)
			}
		}
	} else if strings.Contains(typeText, "[]") || strings.HasPrefix(typeText, "Array<") || strings.HasPrefix(typeText, "ReadonlyArray<") {
		field.Kind = StoryFieldArray
	} else if strings.Contains(typeText, "boolean") {
		field.Kind = StoryFieldBoolean
	} else if strings.Contains(typeText, "number") {
		field.Kind = StoryFieldNumber
	} else if strings.Contains(typeText, "string") {
		field.Kind = StoryFieldText
	} else if strings.HasPrefix(typeText, "{") || strings.HasSuffix(typeText, "Props") {
		field.Kind = StoryFieldObject
	} else {
		return StoryField{}, false
	}
	if match := defaultRE.FindStringSubmatch(comment); len(match) == 2 {
		value := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(strings.Split(match[1], "\n")[0]), "*/"))
		if raw, err := json.Marshal(value); err == nil {
			field.Default = raw
		}
	}
	return field, true
}

func firstCommentSentence(comment string) string {
	comment = strings.TrimSpace(comment)
	if index := strings.IndexAny(comment, ".\n"); index >= 0 {
		comment = comment[:index]
	}
	return strings.TrimSpace(comment)
}

func sameStoryFields(left, right []StoryField) bool {
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	return string(leftJSON) == string(rightJSON)
}

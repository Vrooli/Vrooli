package preview

import (
	"encoding/json"
	"fmt"
	"strings"

	"react-component-library/internal/components"
	internalpreview "react-component-library/internal/preview"
)

func jsString(s string) string {
	var sb strings.Builder
	sb.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			sb.WriteString(`\\`)
		case '"':
			sb.WriteString(`\"`)
		case '\n':
			sb.WriteString(`\n`)
		case '\r':
			sb.WriteString(`\r`)
		default:
			if r < 0x20 || r == '<' || r == '>' || r == '&' {
				fmt.Fprintf(&sb, `\u%04x`, r)
			} else {
				sb.WriteRune(r)
			}
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

func jsonObjectLiteral(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "{}"
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return "{}"
	}
	normalized, err := json.Marshal(obj)
	if err != nil {
		return "{}"
	}
	return string(normalized)
}

func compositionJSON(composition *components.StoryComposition) string {
	if composition == nil {
		return "null"
	}
	raw, err := json.Marshal(composition)
	if err != nil {
		return "null"
	}
	return string(raw)
}

func fixtureJSON(fixture *components.StoryFixtureRef, base string) string {
	baseLiteral := jsonObjectLiteral(base)
	if fixture == nil {
		return baseLiteral
	}
	raw, err := json.Marshal(fixture)
	if err != nil {
		return baseLiteral
	}
	payload, payloadErr := internalpreview.ResolveDeterministicFixture(fixture.Asset, fixture.Version, fixture.State)
	if payloadErr != nil {
		failure, _ := json.Marshal(map[string]string{"fixtureError": payloadErr.Error()})
		return "Object.assign(" + baseLiteral + "," + string(raw) + "," + string(failure) + ")"
	}
	payloadJSON, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return "Object.assign(" + baseLiteral + "," + string(raw) + ")"
	}
	return "Object.assign(" + baseLiteral + "," + string(raw) + "," + string(payloadJSON) + ")"
}

func geometryJSON(geometry *components.StoryGeometry) string {
	if geometry == nil {
		return "null"
	}
	raw, err := json.Marshal(geometry)
	if err != nil {
		return "null"
	}
	return string(raw)
}

func jsonArrayLiteral(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "[]"
	}
	var arr []any
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return "[]"
	}
	normalized, err := json.Marshal(arr)
	if err != nil {
		return "[]"
	}
	return string(normalized)
}

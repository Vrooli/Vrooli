package deployability

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"slices"
)

func replacePlatformPolicyEnums(data []byte) []byte {
	replacement, _ := json.Marshal(platformPolicyValues)
	old := []byte(`["no_work_required", "no_equivalent_ever"]`)
	updated := bytes.ReplaceAll(data, old, replacement)
	old = []byte(`["no_equivalent_ever", "no_work_required"]`)
	return bytes.ReplaceAll(updated, old, replacement)
}

func capabilitySchemaEnum(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var schema struct {
		Properties map[string]struct {
			Enum []string `json:"enum"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	got := append([]string(nil), schema.Properties["capability"].Enum...)
	slices.Sort(got)
	return got, nil
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func replaceCapabilityEnum(data, replacement []byte) ([]byte, error) {
	marker := []byte(`"capability":`)
	property := bytes.Index(data, marker)
	if property < 0 {
		return nil, fmt.Errorf("capability property not found")
	}
	enumMarker := []byte(`"enum"`)
	enumOffset := bytes.Index(data[property:], enumMarker)
	if enumOffset < 0 {
		return nil, fmt.Errorf("capability enum not found")
	}
	enumOffset += property
	start := bytes.IndexByte(data[enumOffset+len(enumMarker):], '[')
	if start < 0 {
		return nil, fmt.Errorf("capability enum array not found")
	}
	start += enumOffset + len(enumMarker)
	end, err := jsonArrayEnd(data, start)
	if err != nil {
		return nil, err
	}
	updated := make([]byte, 0, len(data)-end+start+len(replacement))
	updated = append(updated, data[:start]...)
	updated = append(updated, replacement...)
	updated = append(updated, data[end:]...)
	return updated, nil
}

func jsonArrayEnd(data []byte, start int) (int, error) {
	depth := 0
	quoted := false
	escaped := false
	for i := start; i < len(data); i++ {
		if quoted {
			if escaped {
				escaped = false
			} else if data[i] == '\\' {
				escaped = true
			} else if data[i] == '"' {
				quoted = false
			}
			continue
		}
		switch data[i] {
		case '"':
			quoted = true
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i + 1, nil
			}
		}
	}
	return 0, fmt.Errorf("unterminated capability enum array")
}

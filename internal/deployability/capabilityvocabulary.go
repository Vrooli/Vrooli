package deployability

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CapabilityVocabulary reads the repository's single authored capability
// vocabulary and returns a sorted, validated copy suitable for generated
// schema artifacts.
func CapabilityVocabulary(root string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(root, ".vrooli", "capability-vocabulary.json"))
	if err != nil {
		return nil, err
	}
	var document struct {
		Capabilities []string `json:"capabilities"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse capability vocabulary: %w", err)
	}
	if len(document.Capabilities) == 0 {
		return nil, fmt.Errorf("capability vocabulary is empty")
	}
	seen := make(map[string]struct{}, len(document.Capabilities))
	values := make([]string, 0, len(document.Capabilities))
	for _, capability := range document.Capabilities {
		capability = strings.TrimSpace(capability)
		if capability == "" {
			return nil, fmt.Errorf("capability vocabulary contains an empty entry")
		}
		if _, exists := seen[capability]; exists {
			return nil, fmt.Errorf("capability vocabulary duplicates %q", capability)
		}
		seen[capability] = struct{}{}
		values = append(values, capability)
	}
	sort.Strings(values)
	return values, nil
}

// CheckCapabilitySchemaEnums verifies the checked-in schema artifacts against
// the authored vocabulary without relying on a separate hand-maintained list.
func CheckCapabilitySchemaEnums(root string) error {
	want, err := CapabilityVocabulary(root)
	if err != nil {
		return err
	}
	for _, schemaName := range []string{"safeguard.schema.json", "tool.schema.json"} {
		path := filepath.Join(root, ".vrooli", "schemas", schemaName)
		got, err := capabilitySchemaEnum(path)
		if err != nil {
			return err
		}
		if !sameStrings(got, want) {
			return fmt.Errorf("%s capability enum drifted from .vrooli/capability-vocabulary.json: got %v want %v", schemaName, got, want)
		}
	}
	return nil
}

// GenerateCapabilitySchemaEnums updates only the capability enum in the two
// consumer schemas. The rest of each schema remains byte-for-byte untouched,
// keeping generated changes reviewable and avoiding a formatting rewrite.
func GenerateCapabilitySchemaEnums(root string) error {
	values, err := CapabilityVocabulary(root)
	if err != nil {
		return err
	}
	replacement, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("marshal capability enum: %w", err)
	}
	for _, schemaName := range []string{"safeguard.schema.json", "tool.schema.json"} {
		path := filepath.Join(root, ".vrooli", "schemas", schemaName)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		updated, err := replaceCapabilityEnum(data, replacement)
		if err != nil {
			return fmt.Errorf("%s: %w", schemaName, err)
		}
		if bytes.Equal(data, updated) {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, updated, info.Mode().Perm()); err != nil {
			return err
		}
	}
	return nil
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
	sort.Strings(got)
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

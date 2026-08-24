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
	if err := checkPlatformPolicyEnums(filepath.Join(root, ".vrooli", "schemas", "capability-vocabulary.schema.json")); err != nil {
		return err
	}
	if err := checkPlatformStatusEnum(filepath.Join(root, ".vrooli", "schemas", "common.schema.json")); err != nil {
		return err
	}
	if err := CheckCapabilityManifestPlatformStatus(root); err != nil {
		return err
	}
	return nil
}

// CheckCapabilityManifestPlatformStatus is the retirement lint for the legacy
// tool/safeguard platforms array. Capability resolution must consume an
// explicit declaration for every host OS; the array remains metadata for
// older acquisition code but may not be the source of a capability claim.
func CheckCapabilityManifestPlatformStatus(root string) error {
	paths, err := filepath.Glob(filepath.Join(root, "internal", "tools", "*", "tool.json"))
	if err != nil {
		return err
	}
	safeguards, err := filepath.Glob(filepath.Join(root, "internal", "safeguards", "*", "safeguard.json"))
	if err != nil {
		return err
	}
	paths = append(paths, safeguards...)
	if len(paths) == 0 {
		return nil
	}
	want := []string{"linux", "macos", "windows"}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var manifest struct {
			PlatformStatus map[string]json.RawMessage `json:"platform_status"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		for _, hostOS := range want {
			if _, ok := manifest.PlatformStatus[hostOS]; !ok {
				return fmt.Errorf("%s relies on the legacy platforms array for %s capability resolution", path, hostOS)
			}
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
	path := filepath.Join(root, ".vrooli", "schemas", "capability-vocabulary.schema.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	updated := replacePlatformPolicyEnums(data)
	if !bytes.Equal(data, updated) {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, updated, info.Mode().Perm()); err != nil {
			return err
		}
	}
	commonPath := filepath.Join(root, ".vrooli", "schemas", "common.schema.json")
	common, err := os.ReadFile(commonPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	updatedCommon := replacePlatformStatusEnum(common)
	if !bytes.Equal(common, updatedCommon) {
		info, err := os.Stat(commonPath)
		if err != nil {
			return err
		}
		if err := os.WriteFile(commonPath, updatedCommon, info.Mode().Perm()); err != nil {
			return err
		}
	}
	return nil
}

var platformPolicyValues = []string{"no_equivalent_ever", "no_work_required"}

var platformStatusValues = []string{"supported", "build-verified", "experimental", "unqualified", "partial", "unsupported"}

func checkPlatformStatusEnum(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var schema struct {
		Definitions map[string]struct {
			Properties map[string]struct {
				Enum []string `json:"enum"`
			} `json:"properties"`
		} `json:"definitions"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	got := append([]string(nil), schema.Definitions["platformStatus"].Properties["status"].Enum...)
	want := append([]string(nil), platformStatusValues...)
	sort.Strings(got)
	sort.Strings(want)
	if !sameStrings(got, want) {
		return fmt.Errorf("%s platform status enum drifted: got %v want %v", path, got, want)
	}
	return nil
}

func replacePlatformStatusEnum(data []byte) []byte {
	replacement, _ := json.Marshal(platformStatusValues)
	marker := []byte(`"platformStatus"`)
	definition := bytes.Index(data, marker)
	if definition < 0 {
		return data
	}
	property := bytes.Index(data[definition:], []byte(`"status"`))
	if property < 0 {
		return data
	}
	property += definition
	enum := bytes.Index(data[property:], []byte(`"enum"`))
	if enum < 0 {
		return data
	}
	enum += property
	start := bytes.IndexByte(data[enum:], '[')
	if start < 0 {
		return data
	}
	start += enum
	end, err := jsonArrayEnd(data, start)
	if err != nil {
		return data
	}
	updated := make([]byte, 0, len(data)-end+start+len(replacement))
	updated = append(updated, data[:start]...)
	updated = append(updated, replacement...)
	updated = append(updated, data[end:]...)
	return updated
}

func checkPlatformPolicyEnums(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var schema struct {
		Properties struct {
			PlatformPolicies struct {
				AdditionalProperties struct {
					Properties map[string]struct {
						Enum []string `json:"enum"`
					} `json:"properties"`
					PatternProperties map[string]struct {
						Enum []string `json:"enum"`
					} `json:"patternProperties"`
				} `json:"additionalProperties"`
			} `json:"platform_policies"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	want := append([]string(nil), platformPolicyValues...)
	sort.Strings(want)
	for name, property := range schema.Properties.PlatformPolicies.AdditionalProperties.Properties {
		got := append([]string(nil), property.Enum...)
		sort.Strings(got)
		if !sameStrings(got, want) {
			return fmt.Errorf("%s platform policy %s enum drifted: got %v want %v", path, name, got, want)
		}
	}
	for name, property := range schema.Properties.PlatformPolicies.AdditionalProperties.PatternProperties {
		got := append([]string(nil), property.Enum...)
		sort.Strings(got)
		if !sameStrings(got, want) {
			return fmt.Errorf("%s platform policy %s enum drifted: got %v want %v", path, name, got, want)
		}
	}
	return nil
}

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

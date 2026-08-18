package repocontract

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// DefaultCredentialField is the field used by manifest credential
// declarations when field is omitted. It mirrors the credential declaration
// contract without making the repository-contract package depend on the
// control-plane credential implementation.
const DefaultCredentialField = "value"

// CredentialDescriptorDuplicate identifies two declarations in one manifest
// that address the same logical credential. The JSON paths are RFC 6901-style
// paths, which makes the finding actionable without depending on formatting or
// line-number preservation in the manifest.
type CredentialDescriptorDuplicate struct {
	LogicalID     string
	Field         string
	FirstPath     string
	DuplicatePath string
}

// FindCredentialDescriptorDuplicates checks arbitrary manifest JSON for
// repeated logical_id/field pairs. It deliberately walks the document instead
// of assuming a particular manifest owner or nesting shape: resources,
// scenarios, tools, and future manifest kinds can all declare credentials.
// Missing or empty fields use the same "value" default as the runtime
// credential declaration contract.
func FindCredentialDescriptorDuplicates(data []byte) ([]CredentialDescriptorDuplicate, error) {
	var document any
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("decode manifest JSON: %w", err)
	}

	first := map[string]struct {
		logicalID string
		field     string
		path      string
	}{}
	duplicates := make([]CredentialDescriptorDuplicate, 0)
	var walk func(any, string)
	walk = func(value any, path string) {
		switch typed := value.(type) {
		case []any:
			for index, child := range typed {
				walk(child, path+"/"+fmt.Sprintf("%d", index))
			}
		case map[string]any:
			if logicalID, ok := typed["logical_id"].(string); ok && strings.TrimSpace(logicalID) != "" {
				field := DefaultCredentialField
				if rawField, present := typed["field"]; present {
					fieldValue, ok := rawField.(string)
					if !ok {
						// Manifest-shape validation owns the malformed-type error;
						// this check only reports duplicate declarations.
						fieldValue = ""
					}
					if strings.TrimSpace(fieldValue) != "" {
						field = strings.TrimSpace(fieldValue)
					}
				}
				logicalID = strings.TrimSpace(logicalID)
				key := logicalID + "\x00" + field
				if prior, exists := first[key]; exists {
					duplicates = append(duplicates, CredentialDescriptorDuplicate{
						LogicalID: logicalID, Field: field, FirstPath: prior.path, DuplicatePath: path,
					})
				} else {
					first[key] = struct {
						logicalID string
						field     string
						path      string
					}{logicalID: logicalID, field: field, path: path}
				}
			}

			keys := make([]string, 0, len(typed))
			for key := range typed {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				walk(typed[key], path+"/"+escapeJSONPointer(key))
			}
		}
	}
	walk(document, "")
	return duplicates, nil
}

func escapeJSONPointer(value string) string {
	value = strings.ReplaceAll(value, "~", "~0")
	return strings.ReplaceAll(value, "/", "~1")
}

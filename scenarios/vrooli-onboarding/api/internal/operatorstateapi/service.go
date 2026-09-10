// Package operatorstateapi adapts the control-plane operatorstate writer to
// the typed onboarding transport without creating a second store.
package operatorstateapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/vrooli/vrooli/internal/operatorstate"
)

type Service struct {
	store    *operatorstate.Service
	validate operatorstate.DocumentValidator
}

// MarshalDocument includes fields understood by this checkout and opaque
// fields retained by internal/operatorstate so a read can be projected onto
// the complete wire message without changing storage ownership.
func MarshalDocument(document operatorstate.Document) ([]byte, error) {
	data, err := json.Marshal(document)
	if err != nil || len(document.RawFields) == 0 {
		return data, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	for key, value := range document.RawFields {
		if _, exists := fields[key]; !exists {
			fields[key] = value
		}
	}
	return json.Marshal(fields)
}

func New(store *operatorstate.Service, validate operatorstate.DocumentValidator) Service {
	return Service{store: store, validate: validate}
}

func (s Service) Get(ctx context.Context) (operatorstate.Document, error) {
	if s.store == nil {
		return operatorstate.Document{}, fmt.Errorf("operator state store is not configured")
	}
	return s.store.Load(ctx)
}

// Patch applies only the paths named by updateMask. The selected JSON is
// passed to internal/operatorstate, which remains responsible for locking,
// schema validation, merge semantics, and the atomic write.
func (s Service) Patch(ctx context.Context, stateJSON []byte, updateMask []string, expectedRevision string) (operatorstate.Document, error) {
	if s.store == nil {
		return operatorstate.Document{}, fmt.Errorf("operator state store is not configured")
	}
	patch, err := maskedPatch(stateJSON, updateMask)
	if err != nil {
		return operatorstate.Document{}, err
	}
	return s.store.ApplyAtRevisionValidated(ctx, expectedRevision, patch, s.validate)
}

func maskedPatch(stateJSON []byte, paths []string) ([]byte, error) {
	var source map[string]json.RawMessage
	if err := json.Unmarshal(stateJSON, &source); err != nil || source == nil {
		return nil, fmt.Errorf("operator state patch must be a JSON object")
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("update_mask must name at least one operator state field")
	}

	selected := make(map[string]any)
	for _, rawPath := range paths {
		path := strings.TrimSpace(rawPath)
		if path == "" {
			return nil, fmt.Errorf("update_mask contains an empty path")
		}
		parts := strings.Split(path, ".")
		parts[0] = snakeCase(parts[0])
		value, ok := source[parts[0]]
		if !ok {
			return nil, fmt.Errorf("masked operator state field %q is absent", path)
		}
		var decoded any
		if err := json.Unmarshal(value, &decoded); err != nil {
			return nil, fmt.Errorf("decode masked operator state field %q: %w", path, err)
		}
		if len(parts) == 1 {
			selected[parts[0]] = decoded
			continue
		}
		if err := selectNested(selected, parts, decoded, path); err != nil {
			return nil, err
		}
	}
	return json.Marshal(selected)
}

func selectNested(selected map[string]any, parts []string, value any, path string) error {
	root, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("masked operator state field %q traverses a non-object", path)
	}
	for _, part := range parts[1:] {
		part = snakeCase(part)
		next, exists := root[part]
		if !exists {
			return fmt.Errorf("masked operator state field %q is absent", path)
		}
		root, ok = next.(map[string]any)
		if !ok {
			// The final selected value may be a scalar, so attach the original
			// branch at the first map boundary below.
			break
		}
	}

	branch := selected
	for index, part := range parts[:len(parts)-1] {
		part = snakeCase(part)
		child, ok := branch[part].(map[string]any)
		if !ok {
			child = make(map[string]any)
			branch[part] = child
		}
		branch = child
		if index == len(parts)-2 {
			leaf := snakeCase(parts[len(parts)-1])
			selectedValue, err := lookupPath(value, parts[1:], path)
			if err != nil {
				return err
			}
			branch[leaf] = selectedValue
		}
	}
	return nil
}

func lookupPath(value any, parts []string, path string) (any, error) {
	current := value
	for _, rawPart := range parts {
		part := snakeCase(rawPart)
		object, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("masked operator state field %q traverses a non-object", path)
		}
		var exists bool
		current, exists = object[part]
		if !exists {
			return nil, fmt.Errorf("masked operator state field %q is absent", path)
		}
	}
	return current, nil
}

func snakeCase(value string) string {
	var result strings.Builder
	for index, r := range value {
		if unicode.IsUpper(r) && index > 0 {
			result.WriteByte('_')
		}
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
}

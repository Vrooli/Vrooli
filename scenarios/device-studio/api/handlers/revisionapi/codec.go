// Package revisionapi shares only the JSON projection mechanics between typed
// component endpoints. Candidate validation and revision semantics stay owned
// by devices, scenes, and motions.
package revisionapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"connectrpc.com/connect"
	"device-studio/internal/library"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func Decode[T any](message proto.Message) (T, error) {
	var value T
	if message == nil {
		return value, errors.New("candidate message is required")
	}
	// Generated protobuf structs carry proto-name JSON tags. encoding/json
	// preserves exact int64 values as numbers; ProtoJSON would stringify them.
	data, err := json.Marshal(message)
	if err != nil {
		return value, err
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return value, err
	}
	if err := validateFixedArrayShapes(reflect.TypeOf(value), raw); err != nil {
		return value, err
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return value, err
	}
	return value, nil
}

// Repeated protobuf fields do not preserve presence for empty vectors. Refuse
// lossy decoding into fixed arrays when the field is absent or has the wrong
// length; optional pointer-to-array fields may remain absent.
func validateFixedArrayShapes(target reflect.Type, raw any) error {
	for target.Kind() == reflect.Pointer {
		target = target.Elem()
	}
	switch target.Kind() {
	case reflect.Struct:
		object, ok := raw.(map[string]any)
		if !ok {
			return nil
		}
		for i := 0; i < target.NumField(); i++ {
			field := target.Field(i)
			if field.PkgPath != "" {
				continue
			}
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if name == "" {
				name = field.Name
			}
			value, exists := object[name]
			if !exists {
				if field.Type.Kind() == reflect.Array {
					return fmt.Errorf("required array field %s is missing", name)
				}
				continue
			}
			if value == nil {
				continue
			}
			if err := validateFixedArrayShapes(field.Type, value); err != nil {
				return fmt.Errorf("field %s: %w", name, err)
			}
		}
		return nil
	case reflect.Slice, reflect.Array:
		values, ok := raw.([]any)
		if !ok {
			return nil
		}
		if target.Kind() == reflect.Array && len(values) != target.Len() {
			return fmt.Errorf("array requires exactly %d values, received %d", target.Len(), len(values))
		}
		for i := range values {
			if err := validateFixedArrayShapes(target.Elem(), values[i]); err != nil {
				return err
			}
		}
	case reflect.Map:
		values, ok := raw.(map[string]any)
		if !ok {
			return nil
		}
		for key, value := range values {
			if err := validateFixedArrayShapes(target.Elem(), value); err != nil {
				return fmt.Errorf("map value %q: %w", key, err)
			}
		}
	}
	return nil
}

func Encode[T any](value T, message proto.Message) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return (protojson.UnmarshalOptions{}).Unmarshal(data, message)
}

func InvalidCandidate() error {
	return connect.NewError(connect.CodeInvalidArgument, errors.New("candidate is invalid"))
}

func NotFound() error {
	return connect.NewError(connect.CodeNotFound, errors.New("candidate revision not found"))
}

func Conflict() error {
	return connect.NewError(connect.CodeAborted, errors.New("candidate head changed; reload and retry against the current revision"))
}

// ValidateCursorID accepts an empty first-page cursor or a stable owner ID.
// Keeping this check at the RPC boundary prevents malformed IDs from being
// interpreted as valid-but-empty SQL cursor positions.
func ValidateCursorID(value string) error {
	if value != "" && !library.IsValidStableID(value) {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("cursor ID is invalid"))
	}
	return nil
}

func Internal() error {
	return connect.NewError(connect.CodeInternal, errors.New("candidate revision operation failed"))
}

package resources

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// encoding/json matches struct names without regard to case. Native authority
// frames use the exact declared wire names instead, including nested identities.
// Map keys remain application data and retain their own case-sensitive meaning.
func validateOpenRGBNativeFrameFields(data []byte, target any) error {
	return validateOpenRGBNativeFrameType(data, reflect.TypeOf(target), 0)
}
func validateOpenRGBNativeFrameType(data []byte, target reflect.Type, depth int) error {
	if target == nil || depth > 128 {
		return errors.New("invalid native frame schema or nesting")
	}
	for target.Kind() == reflect.Pointer {
		target = target.Elem()
	}
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil
	}
	switch target.Kind() {
	case reflect.Struct:
		if data[0] != '{' {
			return nil
		} // The typed decoder owns value types.
		var object map[string]json.RawMessage
		if err := json.Unmarshal(data, &object); err != nil {
			return err
		}
		fields := map[string]reflect.Type{}
		collectOpenRGBNativeFrameFields(target, fields)
		for key, value := range object {
			field, ok := fields[key]
			if !ok {
				return fmt.Errorf("unknown or noncanonical native frame field %q", key)
			}
			if err := validateOpenRGBNativeFrameType(value, field, depth+1); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		// []byte (including RawMessage) carries opaque bytes, not struct fields.
		if target.Elem().Kind() == reflect.Uint8 || data[0] != '[' {
			return nil
		}
		var array []json.RawMessage
		if err := json.Unmarshal(data, &array); err != nil {
			return err
		}
		for _, value := range array {
			if err := validateOpenRGBNativeFrameType(value, target.Elem(), depth+1); err != nil {
				return err
			}
		}
	case reflect.Map:
		if data[0] != '{' {
			return nil
		}
		var object map[string]json.RawMessage
		if err := json.Unmarshal(data, &object); err != nil {
			return err
		}
		for _, value := range object {
			if err := validateOpenRGBNativeFrameType(value, target.Elem(), depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}
func collectOpenRGBNativeFrameFields(target reflect.Type, fields map[string]reflect.Type) {
	for i := 0; i < target.NumField(); i++ {
		field := target.Field(i)
		if field.PkgPath != "" {
			continue
		}
		tag := strings.Split(field.Tag.Get("json"), ",")[0]
		if tag == "-" {
			continue
		}
		nested := field.Type
		for nested.Kind() == reflect.Pointer {
			nested = nested.Elem()
		}
		if field.Anonymous && tag == "" && nested.Kind() == reflect.Struct {
			collectOpenRGBNativeFrameFields(nested, fields)
			continue
		}
		if tag == "" {
			tag = field.Name
		}
		fields[tag] = field.Type
	}
}

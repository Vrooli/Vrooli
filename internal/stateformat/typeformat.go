package stateformat

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
)

// typeDescription is the canonical, encoding-independent shape of one stored
// Go type. It deliberately records only what changes the bytes a store writes:
// the JSON name of each field, the kind of each value, and the element and key
// kinds of containers. Declaration order, field reordering and a type's Go name
// do not participate, so renaming a struct or moving a field cannot force a
// state transition, while adding, removing, renaming or retyping one can.
type typeDescription struct {
	Kind   string             `json:"kind"`
	Elem   *typeDescription   `json:"elem,omitempty"`
	Key    string             `json:"key,omitempty"`
	Fields []fieldDescription `json:"fields,omitempty"`
}

type fieldDescription struct {
	JSON string           `json:"json"`
	Type *typeDescription `json:"type"`
}

var (
	timeType       = reflect.TypeFor[time.Time]()
	rawMessageType = reflect.TypeFor[json.RawMessage]()
)

const maxTypeDepth = 32

// DescribeTypes returns a canonical description of the given stored types. It
// is derived from the compiled program, not from source text, so a behaviour
// change, a comment, a formatting change or a different Go toolchain cannot
// alter it. Callers digest the result with DigestSchema.
func DescribeTypes(types ...reflect.Type) (any, error) {
	descriptions := make([]typeDescription, 0, len(types))
	for _, stored := range types {
		if stored == nil {
			return nil, fmt.Errorf("stored type is nil")
		}
		description, err := describeType(stored, map[reflect.Type]bool{}, 0)
		if err != nil {
			return nil, err
		}
		descriptions = append(descriptions, *description)
	}
	return descriptions, nil
}

// DigestTypes is DescribeTypes plus the canonical digest of the result.
func DigestTypes(types ...reflect.Type) (string, error) {
	description, err := DescribeTypes(types...)
	if err != nil {
		return "", err
	}
	return DigestSchema(description)
}

func describeType(stored reflect.Type, visiting map[reflect.Type]bool, depth int) (*typeDescription, error) {
	if depth > maxTypeDepth {
		return nil, fmt.Errorf("stored type %s nests deeper than %d", stored, maxTypeDepth)
	}
	for stored.Kind() == reflect.Pointer {
		stored = stored.Elem()
	}
	if stored == timeType {
		return &typeDescription{Kind: "time"}, nil
	}
	if stored == rawMessageType {
		return &typeDescription{Kind: "raw"}, nil
	}
	if visiting[stored] {
		// A recursive shape is described once, at the point it first appears.
		// The marker is stable, so recursion does not make the digest unstable.
		return &typeDescription{Kind: "cycle"}, nil
	}
	switch stored.Kind() {
	case reflect.Struct:
		visiting[stored] = true
		defer delete(visiting, stored)
		fields := make([]fieldDescription, 0, stored.NumField())
		for i := 0; i < stored.NumField(); i++ {
			field := stored.Field(i)
			name, skip := jsonFieldName(field)
			if skip {
				continue
			}
			fieldType, err := describeType(field.Type, visiting, depth+1)
			if err != nil {
				return nil, err
			}
			fields = append(fields, fieldDescription{JSON: name, Type: fieldType})
		}
		// JSON object keys are unordered, so the description is ordered by the
		// wire name. Moving a field must not look like a format change. The
		// rendered type breaks ties, so two embedded shapes stay ordered by
		// content rather than by declaration order.
		sort.SliceStable(fields, func(i, j int) bool {
			if fields[i].JSON != fields[j].JSON {
				return fields[i].JSON < fields[j].JSON
			}
			return renderField(fields[i]) < renderField(fields[j])
		})
		return &typeDescription{Kind: "object", Fields: fields}, nil
	case reflect.Slice, reflect.Array:
		if stored.Elem().Kind() == reflect.Uint8 {
			return &typeDescription{Kind: "bytes"}, nil
		}
		elem, err := describeType(stored.Elem(), visiting, depth+1)
		if err != nil {
			return nil, err
		}
		return &typeDescription{Kind: "array", Elem: elem}, nil
	case reflect.Map:
		elem, err := describeType(stored.Elem(), visiting, depth+1)
		if err != nil {
			return nil, err
		}
		return &typeDescription{Kind: "map", Key: stored.Key().Kind().String(), Elem: elem}, nil
	case reflect.Interface:
		return &typeDescription{Kind: "any"}, nil
	case reflect.Bool:
		return &typeDescription{Kind: "bool"}, nil
	case reflect.String:
		return &typeDescription{Kind: "string"}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return &typeDescription{Kind: "integer"}, nil
	case reflect.Float32, reflect.Float64:
		return &typeDescription{Kind: "number"}, nil
	default:
		return nil, fmt.Errorf("stored type %s has no stored shape", stored)
	}
}

// jsonFieldName returns the wire name of one struct field. Encoding/json skips
// unexported fields and `json:"-"`, falls back to the Go name when a field
// carries no switch, and flattens an untagged anonymous field into its parent,
// so an embedded field cannot contribute its Go type name to the digest.
func jsonFieldName(field reflect.StructField) (string, bool) {
	if field.PkgPath != "" {
		return "", true
	}
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", true
	}
	name, _, _ := strings.Cut(tag, ",")
	if name != "" {
		return name, false
	}
	if field.Anonymous {
		return "<embedded>", false
	}
	return field.Name, false
}

func renderField(field fieldDescription) string {
	data, err := json.Marshal(field.Type)
	if err != nil {
		return field.JSON
	}
	return string(data)
}

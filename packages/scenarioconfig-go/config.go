// Package scenarioconfig loads typed, schema-backed scenario settings.
package scenarioconfig

import (
	"encoding/json"
	"fmt"
	"os"
)

type ValueType string

const (
	StringType  ValueType = "string"
	NumberType  ValueType = "number"
	IntegerType ValueType = "integer"
	BooleanType ValueType = "boolean"
	ObjectType  ValueType = "object"
	ArrayType   ValueType = "array"
	NullType    ValueType = "null"
)

type Setting struct {
	Type    ValueType `json:"type"`
	Default any       `json:"default"`
}

type Schema struct {
	Settings map[string]Setting `json:"settings"`
}

type document struct {
	Settings map[string]json.RawMessage `json:"settings"`
}

// Load resolves a config document against a schema. Values in config.json
// override schema defaults; omitted values use their required default.
func Load(configPath, schemaPath string) (map[string]any, error) {
	var schema Schema
	if err := readJSON(schemaPath, &schema); err != nil {
		return nil, fmt.Errorf("load scenario config schema: %w", err)
	}
	var configured document
	if payload, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(payload, &configured); err != nil {
			return nil, fmt.Errorf("parse scenario config: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read scenario config: %w", err)
	}
	resolved := make(map[string]any, len(schema.Settings))
	for name, setting := range schema.Settings {
		if setting.Type == "" {
			return nil, fmt.Errorf("setting %q has no type", name)
		}
		if setting.Default == nil && setting.Type != NullType {
			return nil, fmt.Errorf("setting %q has no default", name)
		}
		value := setting.Default
		if raw, ok := configured.Settings[name]; ok {
			if err := json.Unmarshal(raw, &value); err != nil {
				return nil, fmt.Errorf("setting %q: %w", name, err)
			}
		}
		if err := validateType(name, setting.Type, value); err != nil {
			return nil, err
		}
		resolved[name] = value
	}
	for name := range configured.Settings {
		if _, ok := schema.Settings[name]; !ok {
			return nil, fmt.Errorf("setting %q is not declared by the schema", name)
		}
	}
	return resolved, nil
}

func readJSON(path string, target any) error {
	payload, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, target)
}

func validateType(name string, typ ValueType, value any) error {
	valid := false
	switch typ {
	case StringType:
		_, valid = value.(string)
	case NumberType:
		_, valid = value.(float64)
		if !valid {
			_, valid = value.(json.Number)
		}
	case IntegerType:
		if number, ok := value.(float64); ok {
			valid = number == float64(int64(number))
		}
	case BooleanType:
		_, valid = value.(bool)
	case ObjectType:
		_, valid = value.(map[string]any)
	case ArrayType:
		_, valid = value.([]any)
	case NullType:
		valid = value == nil
	default:
		return fmt.Errorf("setting %q has unsupported type %q", name, typ)
	}
	if !valid {
		return fmt.Errorf("setting %q has type %q but value has incompatible type", name, typ)
	}
	return nil
}

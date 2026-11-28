// Package clioutput renders decoded API values for command-line output.
package clioutput

import (
	"encoding/json"
	"fmt"
	"sort"
)

// MapRows renders data as deterministically sorted key/value lines.
func MapRows(data map[string]interface{}) []string {
	if len(data) == 0 {
		return []string{"(empty payload)"}
	}
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	rows := make([]string, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, fmt.Sprintf("%s: %s", key, RenderValue(data[key])))
	}
	return rows
}

// RenderValue formats JSON-decoded values for human display.
func RenderValue(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return "null"
	case string:
		return v
	case bool:
		return fmt.Sprintf("%t", v)
	case float64:
		if float64(int64(v)) == v {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprintf("%v", value)
		}
		return string(data)
	}
}

package jsonutil

import "encoding/json"

// RepairTruncatedJSON attempts to salvage valid data from truncated JSON
// that contains an array of objects. It finds the last complete top-level
// object in the first array encountered, then closes the array and outer
// object to produce valid JSON.
// Returns nil if no repair is possible.
func RepairTruncatedJSON(data []byte) []byte {
	// Find the opening bracket of the first array value in the JSON.
	arrayStart := -1
	for i := 0; i < len(data); i++ {
		if data[i] == '[' {
			arrayStart = i
			break
		}
	}
	if arrayStart == -1 {
		return nil
	}

	// Walk forward tracking brace depth to find the last complete
	// top-level object (depth 0→1→0 with '}').
	lastGoodEnd := -1
	depth := 0
	inString := false
	escaped := false

	for i := arrayStart + 1; i < len(data); i++ {
		ch := data[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == '{' || ch == '[' {
			depth++
		} else if ch == '}' || ch == ']' {
			depth--
			if depth == 0 && ch == '}' {
				lastGoodEnd = i
			}
		}
	}

	if lastGoodEnd == -1 {
		return nil
	}

	repaired := make([]byte, 0, lastGoodEnd+1+4)
	repaired = append(repaired, data[:lastGoodEnd+1]...)
	repaired = append(repaired, '\n', ']', '\n', '}')

	// Validate the repaired JSON actually parses
	if !json.Valid(repaired) {
		return nil
	}
	return repaired
}

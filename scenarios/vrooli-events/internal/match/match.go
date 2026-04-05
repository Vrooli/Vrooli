package match

import "strings"

// Glob checks if value matches the given glob pattern using segment-aware matching.
// Segments are separated by ".". "*" matches exactly one segment. "**" matches one or more segments.
// An empty pattern matches everything.
func Glob(pattern, value string) bool {
	if pattern == "" {
		return true
	}
	patSegs := strings.Split(pattern, ".")
	valSegs := strings.Split(value, ".")
	return segments(patSegs, valSegs)
}

func segments(pat, val []string) bool {
	pi, vi := 0, 0
	for pi < len(pat) && vi < len(val) {
		switch pat[pi] {
		case "**":
			// ** must match at least one segment
			// Try matching remaining pattern against every suffix of val
			for vi2 := vi + 1; vi2 <= len(val); vi2++ {
				if segments(pat[pi+1:], val[vi2:]) {
					return true
				}
			}
			// Also check if ** consumes the rest (no more pattern segments)
			return pi+1 == len(pat)
		case "*":
			// * matches exactly one segment (any value)
			pi++
			vi++
		default:
			if pat[pi] != val[vi] {
				return false
			}
			pi++
			vi++
		}
	}
	return pi == len(pat) && vi == len(val)
}

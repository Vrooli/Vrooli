// Package versionrange evaluates the numeric version-range grammar shared by
// runtime consumers of scenario dependency declarations.
package versionrange

import (
	"strconv"
	"strings"
)

// Satisfies reports whether version matches the supported numeric range
// constraints. Comma-separated clauses and whitespace-separated terms are
// conjunctive. Unsupported or malformed explicit terms fail closed. An empty
// range and "*" retain the legacy unrestricted behavior.
func Satisfies(version, constraint string) bool {
	major, minor, patch, ok := parseVersion(version)
	if !ok {
		return false
	}
	constraint = strings.TrimSpace(constraint)
	if constraint == "" || constraint == "*" {
		return true
	}

	clauses := strings.Split(constraint, ",")
	for _, clause := range clauses {
		if strings.TrimSpace(clause) == "" {
			return false
		}
	}

	for _, clause := range clauses {
		for _, term := range strings.Fields(clause) {
			op := "="
			for _, candidate := range []string{">=", "<=", ">", "<", "=", "^", "~"} {
				if strings.HasPrefix(term, candidate) {
					op = candidate
					term = strings.TrimPrefix(term, candidate)
					break
				}
			}
			wantMajor, wantMinor, wantPatch, valid := parseVersion(term)
			if !valid {
				return false
			}
			cmp := compareVersion(major, minor, patch, wantMajor, wantMinor, wantPatch)
			switch op {
			case "=":
				if cmp != 0 {
					return false
				}
			case ">":
				if cmp <= 0 {
					return false
				}
			case ">=":
				if cmp < 0 {
					return false
				}
			case "<":
				if cmp >= 0 {
					return false
				}
			case "<=":
				if cmp > 0 {
					return false
				}
			case "^":
				if cmp < 0 || major != wantMajor {
					return false
				}
			case "~":
				if cmp < 0 || major != wantMajor || minor != wantMinor {
					return false
				}
			}
		}
	}
	return true
}

func parseVersion(value string) (int, int, int, bool) {
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	parts := strings.Split(value, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return 0, 0, 0, false
	}
	values := [3]int{}
	for index, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 {
			return 0, 0, 0, false
		}
		values[index] = value
	}
	return values[0], values[1], values[2], true
}

func compareVersion(aMajor, aMinor, aPatch, bMajor, bMinor, bPatch int) int {
	for _, pair := range [][2]int{{aMajor, bMajor}, {aMinor, bMinor}, {aPatch, bPatch}} {
		if pair[0] < pair[1] {
			return -1
		}
		if pair[0] > pair[1] {
			return 1
		}
	}
	return 0
}

package config

import (
	"fmt"

	rules "scenario-auditor/rules"
)

type Violation = rules.Violation

func toStringOrDefault(val any) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

//go:build !ruletests

package ui

import (
	"testing"

	rules "scenario-auditor/rules"
	"scenario-auditor/rules/testkit"
)

func TestIframeBridgeDocCases(t *testing.T) {
	cases := testkit.LoadDocCases(t, "iframe_bridge_quality.go")
	for _, tc := range cases {
		tc := tc
		t.Run(tc.ID, func(t *testing.T) {
			path := testkit.DefaultPath(tc, "ui/main.tsx")
			violations := CheckIframeBridgeQuality([]byte(tc.Input), path)
			testkit.EvaluateDocCase(t, tc, len(violations), collectMessages(violations), nil)
		})
	}
}

func collectMessages(vs []rules.Violation) []string {
	messages := make([]string, 0, len(vs))
	for _, v := range vs {
		if v.Message != "" {
			messages = append(messages, v.Message)
		}
		if v.Description != "" {
			messages = append(messages, v.Description)
		}
		if v.Title != "" {
			messages = append(messages, v.Title)
		}
	}
	return messages
}

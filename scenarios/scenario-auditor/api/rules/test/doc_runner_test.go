//go:build ruletests
// +build ruletests

package testrules

import (
	"testing"

	"scenario-auditor/rules/testkit"
)

func runDocTestsViolations(t *testing.T, filename, defaultPath string, fn func([]byte, string) []Violation) {
	t.Helper()
	cases := testkit.LoadDocCases(t, filename)
	for _, tc := range cases {
		tc := tc
		t.Run(tc.ID, func(t *testing.T) {
			path := testkit.DefaultPath(tc, defaultPath)
			violations := fn([]byte(tc.Input), path)
			testkit.EvaluateDocCase(t, tc, len(violations), collectMessages(violations), nil)
		})
	}
}

func collectMessages(vs []Violation) []string {
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

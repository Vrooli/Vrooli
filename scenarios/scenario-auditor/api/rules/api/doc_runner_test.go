//go:build ruletests
// +build ruletests

package api

import (
	"testing"

	"scenario-auditor/internal/testutil"
)

func runDocTestsViolations(t *testing.T, filename, defaultPath string, fn func([]byte, string) []Violation) {
	t.Helper()
	cases := testutil.LoadDocCases(t, filename)
	for _, tc := range cases {
		tc := tc
		t.Run(tc.ID, func(t *testing.T) {
			path := testutil.DefaultPath(tc, defaultPath)
			violations := fn([]byte(tc.Input), path)
			testutil.EvaluateDocCase(t, tc, len(violations), collectMessages(violations), nil)
		})
	}
}

func runDocTestsViolationsErr(t *testing.T, filename, defaultPath string, fn func([]byte, string) ([]Violation, error)) {
	t.Helper()
	cases := testutil.LoadDocCases(t, filename)
	for _, tc := range cases {
		tc := tc
		t.Run(tc.ID, func(t *testing.T) {
			path := testutil.DefaultPath(tc, defaultPath)
			violations, err := fn([]byte(tc.Input), path)
			testutil.EvaluateDocCase(t, tc, len(violations), collectMessages(violations), err)
		})
	}
}

func runDocTestsRuleStruct(t *testing.T, filename, defaultPath string, fn func(string, string) ([]Violation, error)) {
	t.Helper()
	cases := testutil.LoadDocCases(t, filename)
	for _, tc := range cases {
		tc := tc
		t.Run(tc.ID, func(t *testing.T) {
			path := testutil.DefaultPath(tc, defaultPath)
			violations, err := fn(tc.Input, path)
			testutil.EvaluateDocCase(t, tc, len(violations), collectMessages(violations), err)
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

//go:build ruletests
// +build ruletests

package config

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

func runDocTestsViolationsErr(t *testing.T, filename, defaultPath string, fn func([]byte, string) ([]Violation, error)) {
	t.Helper()
	cases := testkit.LoadDocCases(t, filename)
	for _, tc := range cases {
		tc := tc
		t.Run(tc.ID, func(t *testing.T) {
			path := testkit.DefaultPath(tc, defaultPath)
			violations, err := fn([]byte(tc.Input), path)
			testkit.EvaluateDocCase(t, tc, len(violations), collectMessages(violations), err)
		})
	}
}

func runDocTestsCustom[T any](t *testing.T, filename, defaultPath string, fn func(string, string) ([]T, error), messageFn func(T) string) {
	t.Helper()
	cases := testkit.LoadDocCases(t, filename)
	for _, tc := range cases {
		tc := tc
		t.Run(tc.ID, func(t *testing.T) {
			path := testkit.DefaultPath(tc, defaultPath)
			results, err := fn(tc.Input, path)
			messages := make([]string, 0, len(results))
			for _, r := range results {
				messages = append(messages, messageFn(r))
			}
			testkit.EvaluateDocCase(t, tc, len(results), messages, err)
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

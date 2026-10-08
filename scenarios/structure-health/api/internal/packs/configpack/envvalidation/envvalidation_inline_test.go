package envvalidation

import "testing"

// TestGoInlineComparisonCountsAsValidation pins the detector gap that reported
// `os.Getenv("FLAG") == "1"` as unvalidated. The scan for a guard starts on the
// line after the read, so an inline comparison — where the raw value never
// escapes the expression at all — was invisible to it.
func TestGoInlineComparisonCountsAsValidation(t *testing.T) {
	content := `package main

import "os"

func update() bool {
	return os.Getenv("UPDATE_CLI_EVIDENCE") == "1"
}
`
	if violations := checkGoEnvValidation(content, "evidence_test.go"); len(violations) != 0 {
		t.Fatalf("violations for an inline comparison = %+v, want none", violations)
	}
}

// TestGoUnguardedReadIsStillAViolation keeps the rule doing its job: a value
// read and used with no constraint anywhere is still reported.
func TestGoUnguardedReadIsStillAViolation(t *testing.T) {
	content := `package main

import (
	"fmt"
	"os"
)

func run() {
	endpoint := os.Getenv("SERVICE_ENDPOINT")
	fmt.Println(endpoint)
}
`
	if violations := checkGoEnvValidation(content, "main.go"); len(violations) == 0 {
		t.Fatal("expected a violation for an unguarded environment read")
	}
}

// TestJSInlineConstraintCountsAsValidation mirrors the Go fix on the JS path:
// a value bounded on the line where it is read needs no later guard.
func TestJSInlineConstraintCountsAsValidation(t *testing.T) {
	cases := map[string]string{
		"comparison": "startScenarioServer({\n  verbose: process.env.NODE_ENV !== 'production',\n})\n",
		"or-default": "startScenarioServer({\n  apiHost: (process.env.API_HOST || '127.0.0.1').trim() || '127.0.0.1',\n})\n",
		"nullish":    "const host = process.env.API_HOST ?? '127.0.0.1'\n",
	}
	for name, content := range cases {
		if v := checkJSEnvValidation(content, "server.js"); len(v) != 0 {
			t.Errorf("%s: violations = %+v, want none", name, v)
		}
	}
}

// TestJSUnconstrainedReadIsStillAViolation keeps the JS rule doing its job.
func TestJSUnconstrainedReadIsStillAViolation(t *testing.T) {
	content := "const endpoint = process.env.SERVICE_ENDPOINT\nconsole.log('starting')\nstart(endpoint)\n"
	if v := checkJSEnvValidation(content, "server.js"); len(v) == 0 {
		t.Fatal("expected a violation for an unconstrained environment read")
	}
}
